package tactical

import (
	"math"
	"testing"

	"levelup/go-api/internal/domain"
)

// intervalleTest : 100 ms par frame, la cadence des artefacts du depot
// (replay.DefaultFrameIntervalMS). Ecrite ici en clair pour que les comptes attendus se
// verifient a la main : 250 ms = 2,5 frames, 2 s = 20 frames.
const intervalleTest = 100

// pisteImmobile pose une vie qui ne bouge pas, de la frame 0 a la frame `fin`.
func pisteImmobile(xuid string, fin int, x, y float64) Piste {
	pts := make([]PointPiste, 0, fin+1)
	for t := 0; t <= fin; t++ {
		pts = append(pts, PointPiste{T: t, X: x, Y: y})
	}
	return Piste{XUID: xuid, Points: pts, StartFrame: 0, EndFrame: fin}
}

func entree(pistes ...Piste) EntreeOccupation {
	return EntreeOccupation{MatchID: "m1", IntervalleFrameMs: intervalleTest, Pistes: pistes}
}

// TestOccupationImmobileDeuxSecondes — LE COMPTE EXACT DE LA FENETRE DEMI-OUVERTE :
// 2 s de presence a 250 ms de pas font HUIT echantillons, pas neuf. Compter la borne
// haute inventerait un quart de seconde par vie.
func TestOccupationImmobileDeuxSecondes(t *testing.T) {
	g := GrilleParDefaut()
	// frames 0..20 a 100 ms = [0 ms, 2000 ms].
	out := Occupation(g, entree(pisteImmobile("111", 20, 3.2, 7.4)), PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d, attendu 1", len(out))
	}
	j := out[0]
	if j.XUID != "111" {
		t.Fatalf("xuid = %q", j.XUID)
	}
	if len(j.Echantillons) != 8 {
		t.Fatalf("echantillons = %d, attendu 8 (2000 ms / 250 ms, borne haute exclue)", len(j.Echantillons))
	}
	// Tous dans LA MEME cellule : le joueur n'a pas bouge.
	if len(j.PremieresEntrees) != 1 {
		t.Fatalf("cellules atteintes = %d, attendu 1 : %+v", len(j.PremieresEntrees), j.PremieresEntrees)
	}
	attendue, _ := g.Cellule(3.2, 7.4)
	e := j.PremieresEntrees[0]
	if e.Col != attendue.Col || e.Lig != attendue.Lig {
		t.Fatalf("cellule = (%d,%d), attendu (%d,%d)", e.Col, e.Lig, attendue.Col, attendue.Lig)
	}
	if e.Frame != 0 {
		t.Fatalf("premiere entree a la frame %d, attendu 0", e.Frame)
	}
	if len(j.Spawns) != 1 || j.Spawns[0].Frame != 0 || j.Spawns[0].X != 3.2 {
		t.Fatalf("spawns = %+v, attendu un seul a la frame 0 en x=3,2", j.Spawns)
	}
}

// TestOccupationSautDeCelluleAUneSeconde — 4 + 4 : la cellule A tient [0, 1000 ms), la
// cellule B tient [1000, 2000 ms). C'est le test de la POSITION TENUE : entre deux points
// l'echantillon prend la derniere position connue, jamais une interpolation.
func TestOccupationSautDeCelluleAUneSeconde(t *testing.T) {
	g := GrilleParDefaut()
	// Une vie qui reste en (0,25 ; 0,25) jusqu'a la frame 9, saute en (10,25 ; 0,25) a la
	// frame 10 (= 1000 ms), et y reste jusqu'a la frame 20 (= 2000 ms).
	pts := make([]PointPiste, 0, 21)
	for t := 0; t <= 20; t++ {
		x := 0.25
		if t >= 10 {
			x = 10.25
		}
		pts = append(pts, PointPiste{T: t, X: x, Y: 0.25})
	}
	out := Occupation(g, entree(Piste{XUID: "111", Points: pts, StartFrame: 0, EndFrame: 20}), PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d", len(out))
	}
	j := out[0]
	if len(j.Echantillons) != 8 {
		t.Fatalf("echantillons = %d, attendu 8", len(j.Echantillons))
	}
	// Les quatre premiers en x=0,25 ; les quatre suivants en x=10,25.
	for i, s := range j.Echantillons {
		attenduX := 0.25
		if i >= 4 {
			attenduX = 10.25
		}
		if s.X != attenduX {
			t.Fatalf("echantillon %d en x=%v, attendu %v (saut a t=1 s)", i, s.X, attenduX)
		}
		if s.MatchID != "m1" {
			t.Fatalf("echantillon %d sans match", i)
		}
	}
	if len(j.PremieresEntrees) != 2 {
		t.Fatalf("cellules = %+v, attendu 2", j.PremieresEntrees)
	}
	// La cellule d'arrivee (x=10,25 -> col 20) est entree a la frame 10, pas avant.
	arrivee, _ := g.Cellule(10.25, 0.25)
	trouvee := false
	for _, e := range j.PremieresEntrees {
		if e.Col == arrivee.Col && e.Lig == arrivee.Lig {
			trouvee = true
			if e.Frame != 10 {
				t.Fatalf("premiere entree dans la cellule d'arrivee a la frame %d, attendu 10", e.Frame)
			}
		}
	}
	if !trouvee {
		t.Fatalf("cellule d'arrivee absente : %+v", j.PremieresEntrees)
	}
}

// TestOccupationIgnoreLesPistesSansXUID — une vie que le film n'a pas nommee n'appartient
// a personne : la rattacher par son slot prendrait un ORDRE pour une identite.
func TestOccupationIgnoreLesPistesSansXUID(t *testing.T) {
	out := Occupation(GrilleParDefaut(), entree(
		pisteImmobile("", 20, 1, 1),
		pisteImmobile("222", 20, 2, 2),
	), PasOccupationMs)
	if len(out) != 1 || out[0].XUID != "222" {
		t.Fatalf("joueurs = %+v, attendu le seul 222", out)
	}
}

// TestOccupationEcarteLesPointsHorsFenetre — StartFrame/EndFrame BORNENT la vie : un
// point qui en sort (slot reattribue, queue de film) n'est pas de cette vie-la.
func TestOccupationEcarteLesPointsHorsFenetre(t *testing.T) {
	pts := []PointPiste{
		{T: 0, X: 100, Y: 100}, // AVANT la fenetre
		{T: 10, X: 1.25, Y: 1.25},
		{T: 20, X: 1.25, Y: 1.25},
		{T: 30, X: 200, Y: 200}, // APRES la fenetre
	}
	out := Occupation(GrilleParDefaut(),
		entree(Piste{XUID: "111", Points: pts, StartFrame: 10, EndFrame: 20}), PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d", len(out))
	}
	j := out[0]
	// La fenetre retenue est [1000 ms, 2000 ms) : 4 echantillons.
	if len(j.Echantillons) != 4 {
		t.Fatalf("echantillons = %d, attendu 4", len(j.Echantillons))
	}
	for i, s := range j.Echantillons {
		if s.X != 1.25 || s.Y != 1.25 {
			t.Fatalf("echantillon %d = (%v,%v) : un point hors fenetre a ete lu", i, s.X, s.Y)
		}
	}
	if len(j.Spawns) != 1 || j.Spawns[0].Frame != 10 {
		t.Fatalf("spawn = %+v, attendu la frame 10 (le premier point DANS la fenetre)", j.Spawns)
	}
	if len(j.PremieresEntrees) != 1 {
		t.Fatalf("cellules = %+v, attendu 1", j.PremieresEntrees)
	}
}

// TestOccupationFenetreNonDeclaree — StartFrame et EndFrame sont `omitempty` dans
// l'artefact : un artefact ancien ne les porte pas, et les points doivent alors se
// borner eux-memes plutot que de rendre une occupation vide.
func TestOccupationFenetreNonDeclaree(t *testing.T) {
	pts := []PointPiste{{T: 0, X: 1.25, Y: 1.25}, {T: 20, X: 1.25, Y: 1.25}}
	out := Occupation(GrilleParDefaut(),
		entree(Piste{XUID: "111", Points: pts}), PasOccupationMs)
	if len(out) != 1 || len(out[0].Echantillons) != 8 {
		t.Fatalf("occupation = %+v, attendu 8 echantillons sans fenetre declaree", out)
	}
}

// TestOccupationPlusieursVies — un joueur a autant de pistes que de morts : ses
// echantillons s'additionnent, et la premiere entree d'une cellule est la PLUS PRECOCE
// de toutes ses vies, meme si les pistes arrivent dans le desordre.
func TestOccupationPlusieursVies(t *testing.T) {
	tard := Piste{XUID: "111", StartFrame: 40, EndFrame: 60, Points: []PointPiste{
		{T: 40, X: 1.25, Y: 1.25}, {T: 60, X: 1.25, Y: 1.25},
	}}
	tot := Piste{XUID: "111", StartFrame: 0, EndFrame: 20, Points: []PointPiste{
		{T: 0, X: 1.25, Y: 1.25}, {T: 20, X: 1.25, Y: 1.25},
	}}
	out := Occupation(GrilleParDefaut(), entree(tard, tot), PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d, attendu 1 (deux vies du meme xuid)", len(out))
	}
	j := out[0]
	if len(j.Echantillons) != 16 {
		t.Fatalf("echantillons = %d, attendu 16 (8 + 8)", len(j.Echantillons))
	}
	if len(j.Spawns) != 2 || j.Spawns[0].Frame != 0 || j.Spawns[1].Frame != 40 {
		t.Fatalf("spawns = %+v, attendu tries par frame 0 puis 40", j.Spawns)
	}
	if len(j.PremieresEntrees) != 1 || j.PremieresEntrees[0].Frame != 0 {
		t.Fatalf("premieres entrees = %+v, attendu la frame 0 (la vie la plus precoce)", j.PremieresEntrees)
	}
}

// TestOccupationPositionNonFinie — un decodage qui derape ne doit pas fabriquer une
// cellule : l'echantillon existe (le temps a bien passe), la cellule non.
func TestOccupationPositionNonFinie(t *testing.T) {
	nan := math.NaN()
	pts := []PointPiste{{T: 0, X: nan, Y: nan}, {T: 20, X: nan, Y: nan}}
	out := Occupation(GrilleParDefaut(),
		entree(Piste{XUID: "111", Points: pts, StartFrame: 0, EndFrame: 20}), PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d", len(out))
	}
	if len(out[0].Echantillons) != 8 {
		t.Fatalf("echantillons = %d, attendu 8", len(out[0].Echantillons))
	}
	if len(out[0].PremieresEntrees) != 0 {
		t.Fatalf("cellules = %+v, attendu aucune", out[0].PremieresEntrees)
	}
}

// TestOccupationSansEchelleDeTemps — l'echelle de l'axe de temps vient de l'artefact et
// l'appelant la resout ; ce paquet ne DEVINE aucune cadence. Sans elle, rien n'est
// mesurable — et surtout, aucune division par zero.
func TestOccupationSansEchelleDeTemps(t *testing.T) {
	e := EntreeOccupation{MatchID: "m1", IntervalleFrameMs: 0, Pistes: []Piste{pisteImmobile("111", 20, 1, 1)}}
	if out := Occupation(GrilleParDefaut(), e, PasOccupationMs); out != nil {
		t.Fatalf("occupation = %+v, attendu rien sans echelle de temps", out)
	}
	e.IntervalleFrameMs = intervalleTest
	if out := Occupation(GrilleParDefaut(), e, 0); out != nil {
		t.Fatalf("occupation = %+v, attendu rien avec un pas nul", out)
	}
}

// TestOccupationPointsDesordonnes — l'artefact les rend croissants, mais le curseur de
// l'echantillonnage NE RECULE PAS : une piste lue de travers rendrait une occupation
// silencieusement fausse. Le tri defensif la rend identique a la piste triee.
func TestOccupationPointsDesordonnes(t *testing.T) {
	ordonnee := Piste{XUID: "111", StartFrame: 0, EndFrame: 20, Points: []PointPiste{
		{T: 0, X: 0.25, Y: 0.25}, {T: 10, X: 10.25, Y: 0.25}, {T: 20, X: 10.25, Y: 0.25},
	}}
	desordonnee := Piste{XUID: "111", StartFrame: 0, EndFrame: 20, Points: []PointPiste{
		{T: 20, X: 10.25, Y: 0.25}, {T: 0, X: 0.25, Y: 0.25}, {T: 10, X: 10.25, Y: 0.25},
	}}
	a := Occupation(GrilleParDefaut(), entree(ordonnee), PasOccupationMs)
	b := Occupation(GrilleParDefaut(), entree(desordonnee), PasOccupationMs)
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("a = %+v, b = %+v", a, b)
	}
	if len(a[0].Echantillons) != len(b[0].Echantillons) {
		t.Fatalf("echantillons %d vs %d", len(a[0].Echantillons), len(b[0].Echantillons))
	}
	for i := range a[0].Echantillons {
		if a[0].Echantillons[i] != b[0].Echantillons[i] {
			t.Fatalf("echantillon %d differe : %+v vs %+v", i, a[0].Echantillons[i], b[0].Echantillons[i])
		}
	}
}

// TestSecondesParEchantillon — un echantillon de 250 ms vaut un quart de seconde, et un
// pas absurde ne vaut rien plutot qu'une division par zero.
func TestSecondesParEchantillon(t *testing.T) {
	if s := SecondesParEchantillon(PasOccupationMs); s != 0.25 {
		t.Fatalf("secondes par echantillon = %v, attendu 0,25", s)
	}
	if s := SecondesParEchantillon(0); s != 0 {
		t.Fatalf("secondes par echantillon = %v pour un pas nul, attendu 0", s)
	}
}

// TestEnSecondesNeTouchePasLeBrut — la VALEUR passe en secondes, le BRUT reste le compte
// d'echantillons : c'est la mesure, la seconde n'en est que l'unite de lecture (doctrine
// « jamais un taux seul »). L'entree n'est pas modifiee.
func TestEnSecondesNeTouchePasLeBrut(t *testing.T) {
	in := []domain.CelluleTactique{{Col: 1, Lig: 2, Valeur: 8, Brut: 24, Matchs: 3}}
	out := EnSecondes(in, PasOccupationMs)
	if out[0].Valeur != 2 {
		t.Fatalf("valeur = %v, attendu 2 s (8 echantillons x 0,25 s)", out[0].Valeur)
	}
	if out[0].Brut != 24 {
		t.Fatalf("brut = %v, attendu 24 echantillons inchanges", out[0].Brut)
	}
	if in[0].Valeur != 8 {
		t.Fatalf("l'entree a ete modifiee : %v", in[0].Valeur)
	}
}

// ─── LE TEMPS EN VEHICULE ──────────────────────────────────────────────────────
//
// Ce que ces cas gardent : sans eux, un occupant embarque disparait de la mesure pendant
// tout son trajet (son bipede cesse de repliquer, et un trou > 5 s coupe deja sa vie en
// amont), alors meme que le match compte comme mesure.

// vehiculeQuiTraverse : une trajectoire de vehicule qui va de la cellule (0,0) a la
// cellule (40,0), a mi-parcours de la fenetre.
func vehiculeQuiTraverse(t0, t1 int) []PointPiste {
	return []PointPiste{
		{T: t0, X: 0.25, Y: 0.25},
		{T: (t0 + t1) / 2, X: 20.25, Y: 0.25},
	}
}

// TestOccupationEmbarquement_SuitLeVehicule — CAS (a) : un episode de 4 s sur un vehicule
// qui traverse deux cellules rend 16 echantillons SUR LES CELLULES DU VEHICULE, et zero
// sur la cellule d'embarquement.
func TestOccupationEmbarquement_SuitLeVehicule(t *testing.T) {
	g := GrilleParDefaut()
	// Le bipede n'a qu'un point, a la cellule (100,100) : c'est la ou il monte. Sa vie ne
	// produit aucun echantillon (fenetre de duree nulle), mais elle le NOMME.
	piste := Piste{XUID: "111", StartFrame: 0, EndFrame: 0, Points: []PointPiste{{T: 0, X: 50.25, Y: 50.25}}}
	e := EntreeOccupation{
		MatchID: "m1", IntervalleFrameMs: intervalleTest, Pistes: []Piste{piste},
		// 4 s a 100 ms/frame = frames 0 a 40.
		Embarquements: []Embarquement{{XUID: "111", T0: 0, T1: 40, Points: vehiculeQuiTraverse(0, 40)}},
	}
	out := Occupation(g, e, PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d, attendu 1", len(out))
	}
	j := out[0]
	if len(j.Echantillons) != 16 {
		t.Fatalf("echantillons = %d, attendu 16 (4 s / 250 ms) — le temps en vehicule est PERDU",
			len(j.Echantillons))
	}
	// Deux cellules, celles du VEHICULE : (0,0) puis (40,0).
	depart, _ := g.Cellule(0.25, 0.25)
	arrivee, _ := g.Cellule(20.25, 0.25)
	embarquement, _ := g.Cellule(50.25, 50.25)
	vues := map[Cellule]int{}
	for _, c := range j.PremieresEntrees {
		vues[Cellule{Col: c.Col, Lig: c.Lig}] = c.Frame
	}
	if len(vues) != 2 {
		t.Fatalf("cellules atteintes = %+v, attendu les 2 du vehicule", j.PremieresEntrees)
	}
	if _, ok := vues[depart]; !ok {
		t.Fatalf("la cellule de depart du vehicule est absente : %+v", j.PremieresEntrees)
	}
	if frame, ok := vues[arrivee]; !ok {
		t.Fatalf("la cellule d'arrivee du vehicule est absente : %+v", j.PremieresEntrees)
	} else if frame != 20 {
		t.Fatalf("premiere entree dans la cellule d'arrivee a la frame %d, attendu 20", frame)
	}
	if _, ok := vues[embarquement]; ok {
		t.Fatalf("la cellule D'EMBARQUEMENT du bipede a ete peinte : %+v", j.PremieresEntrees)
	}
}

// TestOccupationEmbarquement_SansPointDeVehicule — CAS (b) : un episode dont le vehicule
// n'a aucun point n'attribue RIEN. On ATTRIBUE, on n'invente pas — lui preter la position
// d'embarquement fabriquerait un stationnement la ou il y a eu un trajet.
func TestOccupationEmbarquement_SansPointDeVehicule(t *testing.T) {
	piste := Piste{XUID: "111", StartFrame: 0, EndFrame: 0, Points: []PointPiste{{T: 0, X: 50.25, Y: 50.25}}}
	e := EntreeOccupation{
		MatchID: "m1", IntervalleFrameMs: intervalleTest, Pistes: []Piste{piste},
		Embarquements: []Embarquement{{XUID: "111", T0: 0, T1: 40}},
	}
	out := Occupation(GrilleParDefaut(), e, PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d, attendu 1", len(out))
	}
	if len(out[0].Echantillons) != 0 {
		t.Fatalf("echantillons = %d, attendu 0 : rien n'est mesure, donc rien n'est attribue",
			len(out[0].Echantillons))
	}
	// Le spawn de la vie, lui, est inchange.
	if len(out[0].Spawns) != 1 || out[0].Spawns[0].Frame != 0 {
		t.Fatalf("spawns = %+v, attendu celui de la vie, intact", out[0].Spawns)
	}
}

// TestOccupationEmbarquement_NeCreeAucunSpawn — CAS (c) : monter dans un vehicule n'est
// pas reapparaitre. Un episode seul (aucune vie nommee) donne un joueur mesure, sans
// spawn — sans quoi les grappes de reapparition de la phase 7 seraient polluees par des
// points de montee en vehicule.
func TestOccupationEmbarquement_NeCreeAucunSpawn(t *testing.T) {
	e := EntreeOccupation{
		MatchID: "m1", IntervalleFrameMs: intervalleTest,
		Embarquements: []Embarquement{{XUID: "111", T0: 0, T1: 40, Points: vehiculeQuiTraverse(0, 40)}},
	}
	out := Occupation(GrilleParDefaut(), e, PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d, attendu 1 (l'occupant est nomme par l'episode)", len(out))
	}
	if len(out[0].Spawns) != 0 {
		t.Fatalf("spawns = %+v, attendu aucun : un embarquement n'est pas une reapparition",
			out[0].Spawns)
	}
	if len(out[0].Echantillons) != 16 {
		t.Fatalf("echantillons = %d, attendu 16", len(out[0].Echantillons))
	}
}

// TestOccupationEmbarquement_LeVehiculeGagneSurLeBipede — CAS (d) : un point de bipede
// DANS la fenetre de l'episode est ignore au profit du vehicule. Pendant l'episode le
// bipede ne replique plus : sa derniere position connue ne dit plus ou est le joueur, et
// la tenir peindrait un stationnement au point de montee.
func TestOccupationEmbarquement_LeVehiculeGagneSurLeBipede(t *testing.T) {
	g := GrilleParDefaut()
	// Une vie qui couvre TOUTE la fenetre de l'episode, immobile en (50,25 ; 50,25).
	piste := pisteImmobile("111", 40, 50.25, 50.25)
	e := EntreeOccupation{
		MatchID: "m1", IntervalleFrameMs: intervalleTest, Pistes: []Piste{piste},
		Embarquements: []Embarquement{{XUID: "111", T0: 0, T1: 40, Points: vehiculeQuiTraverse(0, 40)}},
	}
	out := Occupation(g, e, PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d", len(out))
	}
	j := out[0]
	// La vie couvre [0, 4000 ms) et l'episode [0, 40] frames : TOUS ses instants sont
	// couverts, donc seuls les 16 echantillons du vehicule subsistent.
	if len(j.Echantillons) != 16 {
		t.Fatalf("echantillons = %d, attendu 16 : le bipede a ete compte EN PLUS du vehicule",
			len(j.Echantillons))
	}
	embarquement, _ := g.Cellule(50.25, 50.25)
	for _, c := range j.PremieresEntrees {
		if c.Col == embarquement.Col && c.Lig == embarquement.Lig {
			t.Fatalf("la cellule du bipede a ete peinte pendant l'episode : %+v", j.PremieresEntrees)
		}
	}
}

// TestOccupationEmbarquement_ChevauchementDuMemeJoueur — le document PUBLIE l'ambiguite :
// conducteur et passager ne se departagent pas par la geometrie, et une vie de vehicule
// peut porter deux episodes chevauchants. Pour un MEME xuid, les sommer compterait deux
// fois le temps d'une seule personne.
func TestOccupationEmbarquement_ChevauchementDuMemeJoueur(t *testing.T) {
	pts := vehiculeQuiTraverse(0, 40)
	e := EntreeOccupation{
		MatchID: "m1", IntervalleFrameMs: intervalleTest,
		Embarquements: []Embarquement{
			{XUID: "111", T0: 0, T1: 40, Points: pts},
			{XUID: "111", T0: 20, T1: 60, Points: pts}, // chevauche le premier
		},
	}
	out := Occupation(GrilleParDefaut(), e, PasOccupationMs)
	if len(out) != 1 || len(out[0].Echantillons) != 16 {
		t.Fatalf("echantillons = %+v, attendu 16 : le second episode chevauchant doit etre ecarte", out)
	}
}

// TestOccupationEmbarquement_SansOccupantNomme — un episode dont le pont du fil des morts
// n'a pas nomme le slot n'attribue ce temps a personne.
func TestOccupationEmbarquement_SansOccupantNomme(t *testing.T) {
	e := EntreeOccupation{
		MatchID: "m1", IntervalleFrameMs: intervalleTest,
		Embarquements: []Embarquement{{T0: 0, T1: 40, Points: vehiculeQuiTraverse(0, 40)}},
	}
	if out := Occupation(GrilleParDefaut(), e, PasOccupationMs); len(out) != 0 {
		t.Fatalf("occupation = %+v, attendu personne", out)
	}
}

// ─── DEUX MUTATIONS QUI PASSAIENT (C10) ────────────────────────────────────────

// TestOccupationFrameDEntreeEstCelleDeLEchantillon — la frame publiee est celle de
// L'ECHANTILLON, pas celle du point tenu. Elles different des que le point precede la
// frontiere d'echantillon : ici le point est a T=3 (300 ms) et le premier echantillon qui
// le voit est a 500 ms, soit la frame 5. La mutation `frame := p.T` publierait 3 — et le
// clic sur la cellule ouvrirait le rejeu 200 ms avant que le joueur n'y soit.
func TestOccupationFrameDEntreeEstCelleDeLEchantillon(t *testing.T) {
	pts := []PointPiste{
		{T: 0, X: 0.25, Y: 0.25},
		{T: 3, X: 10.25, Y: 0.25}, // 300 ms : entre l'echantillon de 250 ms et celui de 500 ms
		{T: 20, X: 10.25, Y: 0.25},
	}
	out := Occupation(GrilleParDefaut(),
		entree(Piste{XUID: "111", Points: pts, StartFrame: 0, EndFrame: 20}), PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d", len(out))
	}
	arrivee, _ := GrilleParDefaut().Cellule(10.25, 0.25)
	for _, c := range out[0].PremieresEntrees {
		if c.Col == arrivee.Col && c.Lig == arrivee.Lig {
			if c.Frame != 5 {
				t.Fatalf("premiere entree a la frame %d, attendu 5 (500 ms) : la frame publiee "+
					"est celle de l'echantillon, pas celle du point tenu (T=3)", c.Frame)
			}
			return
		}
	}
	t.Fatalf("cellule d'arrivee absente : %+v", out[0].PremieresEntrees)
}

// TestOccupationVieAUnSeulPoint — une vie d'un seul point couvre une duree NULLE : zero
// echantillon, mais un spawn. La mutation `if len(points) < 2 { continue }` supprimerait
// le spawn — et les grappes de reapparition de la phase 7 perdraient les vies les plus
// courtes, celles ou le joueur meurt aussitot ne.
func TestOccupationVieAUnSeulPoint(t *testing.T) {
	out := Occupation(GrilleParDefaut(),
		entree(Piste{XUID: "111", Points: []PointPiste{{T: 7, X: 1.25, Y: 1.25}}}), PasOccupationMs)
	if len(out) != 1 {
		t.Fatalf("joueurs = %d, attendu 1", len(out))
	}
	if len(out[0].Echantillons) != 0 {
		t.Fatalf("echantillons = %d, attendu 0 (duree nulle)", len(out[0].Echantillons))
	}
	if len(out[0].Spawns) != 1 || out[0].Spawns[0].Frame != 7 {
		t.Fatalf("spawns = %+v, attendu un seul a la frame 7", out[0].Spawns)
	}
}
