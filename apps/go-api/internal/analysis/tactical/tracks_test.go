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
