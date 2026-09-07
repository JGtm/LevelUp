package replay

// death_context_test.go — LES ÉTATS D'UN COÉQUIPIER, chacun avec son mode de panne.
//
// Chaque état existe parce que le confondre avec un autre produit un faux verdict :
//
//	visible confondu avec hors de vue    une mort à 3 m d'un coéquipier sort « isolée » ;
//	hors de vue confondu avec en attente un coéquipier en véhicule est compté mort, et la mort
//	                                     sort « équipe à terre » (défaut P0 de la ronde 1) ;
//	parti confondu avec hors de vue      un joueur déconnecté à la première minute reste un
//	                                     coéquipier disponible jusqu'à la fin ;
//	absent confondu avec hors de vue     un `joined_in_progress` « accompagne » des morts
//	                                     survenues avant son arrivée ;
//	mort confondu avec visible           la réplication s'arrête ~34 ms APRÈS la mort, donc un
//	                                     joueur mort depuis 500 ms a encore une position fraîche.
//
// LE PONT EST LE VRAI (`ResolveSlotXUID`) : les fixtures nomment leurs vies par des morts du
// fil, comme un film. Poser un `SlotXUID` à la main court-circuitait le nommage — donc aussi le
// défaut P0-2, où le second occupant d'un slot recyclé hérite des positions du premier.

import (
	"math"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// posMonde pose une position monde d'un slot, à un instant de l'horloge du FILM.
func posMonde(slot uint32, tMS int64, x, y float32) filmdec.BipedPosition {
	return filmdec.BipedPosition{
		Slot: slot, TimestampUS: uint64(tMS) * 1000, X: x, Y: y, HasWorld: true,
	}
}

// pisteMonde échantillonne un slot toutes les 100 ms, à position fixe.
func pisteMonde(slot uint32, deMS, aMS int64, x, y float32) []filmdec.BipedPosition {
	out := []filmdec.BipedPosition{}
	for t := deMS; t <= aMS; t += 100 {
		out = append(out, posMonde(slot, t, x, y))
	}
	return out
}

// mortFilm : une mort du fil du FILM — c'est elle qui NOMME la vie qu'elle clôt.
func mortFilm(xuid uint64, tMS int64) Death { return Death{XUID: xuid, TimeMS: tMS} }

// dcEntree monte une entrée : le pont est construit par `ResolveSlotXUID`, le vrai.
func dcEntree(pos []filmdec.BipedPosition, mortsFilm []Death, journal []MortDuJournal,
	departs, arrivees map[uint64]int64,
) EntreeContexteMorts {
	_, rep := ResolveSlotXUID(pos, mortsFilm, indexDe(111, 222, 333, 444, 999))
	return EntreeContexteMorts{
		Positions: pos,
		Report:    rep,
		Journal:   journal,
		Equipes:   map[uint64]int{111: 0, 222: 0, 333: 0, 444: 0, 999: 1},
		DepartMS:  departs,
		ArriveeMS: arrivees,
	}
}

// corpusDeReference : moi (111) meurs à 10 000 en (0,0), et à cet instant —
//
//	222   répliqué en (3,0), vivant             -> VISIBLE, à 3 m
//	333   mort à 6 000, plus rien depuis        -> EN ATTENTE
//	444   sa vie de 0 à 4 000 n'est pas nommée  -> HORS DE VUE (véhicule), et VIVANT
//	999   adversaire en (1,0), visible          -> n'entre nulle part
//
// Les vies se nomment par les morts du fil : chaque joueur en a une qui clôt sa vie.
func corpusDeReference() ([]filmdec.BipedPosition, []Death, []MortDuJournal) {
	var pos []filmdec.BipedPosition
	pos = append(pos, pisteMonde(1, 0, 10_000, 0, 0)...)  // moi
	pos = append(pos, pisteMonde(2, 0, 20_000, 3, 0)...)  // visible a 3 m
	pos = append(pos, pisteMonde(3, 0, 6_000, 20, 20)...) // meurt a 6 000
	pos = append(pos, pisteMonde(4, 0, 4_000, 40, 40)...) // vie NON nommee : vehicule
	pos = append(pos, pisteMonde(4, 20_000, 25_000, 5, 5)...)
	pos = append(pos, pisteMonde(9, 0, 20_000, 1, 0)...) // adversaire tout proche

	mortsFilm := []Death{
		mortFilm(333, 6_000), mortFilm(111, 10_000),
		mortFilm(222, 20_000), mortFilm(999, 20_000), mortFilm(444, 25_000),
	}
	journal := []MortDuJournal{
		{VictimeXUID: 333, TempsMS: 6_000}, {VictimeXUID: 111, TempsMS: 10_000},
	}
	return pos, mortsFilm, journal
}

// TestContextesDesMorts_LesQuatreEtats — LE CAS DE RÉFÉRENCE.
func TestContextesDesMorts_LesQuatreEtats(t *testing.T) {
	pos, mortsFilm, journal := corpusDeReference()
	out := ContextesDesMorts(dcEntree(pos, mortsFilm, journal, nil, nil))

	c := contexteDe(t, out, 111, 10_000)
	if c.Visibles != 1 || c.EnAttente != 1 || c.HorsDeVue != 1 || c.Partis != 0 {
		t.Fatalf("etats = visible %d / attente %d / hors de vue %d / parti %d, "+
			"attendu 1/1/1/0 — %+v", c.Visibles, c.EnAttente, c.HorsDeVue, c.Partis, c)
	}
	if c.Total != 3 {
		t.Fatalf("total = %d, attendu 3 coequipiers : l'ADVERSAIRE visible a 1 m ne doit "+
			"entrer nulle part", c.Total)
	}
	if c.PlusProcheM == nil || math.Abs(*c.PlusProcheM-3) > 0.001 {
		t.Fatalf("plus proche = %v, attendu 3 m (le coequipier visible, jamais l'adversaire)",
			c.PlusProcheM)
	}
}

// TestContextesDesMorts_SlotRecycle_LePremierOccupantNHeritePas — LE DÉFAUT P0-2.
//
// Le slot 2 porte DEUX vies nommées : 222 de 0 à 10 000, puis 333 de 18 000 à 28 000. Une mort
// à 20 000 doit voir 333 — pas 222, qui n'est plus là depuis 10 s.
//
// Le pont aplati (`SlotXUID`) donne tout l'intervalle du slot à son PREMIER porteur nommé :
// 222 aurait hérité des positions de 333, `teammates_visible` et `nearest_teammate_m` auraient
// été faux, et rien ne l'aurait signalé. C'est le bug que `nameTracksByLives` a corrigé pour
// les traces le 2026-09-02 ; l'attribution se fait ici par la VIE QUI COUVRE L'INSTANT.
func TestContextesDesMorts_SlotRecycle_LePremierOccupantNHeritePas(t *testing.T) {
	var pos []filmdec.BipedPosition
	pos = append(pos, pisteMonde(1, 0, 25_000, 0, 0)...)      // moi, tout du long
	pos = append(pos, pisteMonde(2, 0, 10_000, 50, 50)...)    // 222, loin, puis mort
	pos = append(pos, pisteMonde(2, 18_000, 28_000, 3, 0)...) // 333 REPREND LE SLOT, a 3 m

	mortsFilm := []Death{mortFilm(222, 10_000), mortFilm(333, 28_000), mortFilm(111, 25_000)}
	// LE JOURNAL PORTE TOUTES LES MORTS, y compris celle de 222 : c'est lui qui dit qui
	// attend sa reapparition.
	journal := []MortDuJournal{
		{VictimeXUID: 222, TempsMS: 10_000}, {VictimeXUID: 111, TempsMS: 20_000},
	}

	e := dcEntree(pos, mortsFilm, journal, nil, nil)
	e.Equipes = map[uint64]int{111: 0, 222: 0, 333: 0}
	c := contexteDe(t, ContextesDesMorts(e), 111, 20_000)

	if c.Visibles != 1 {
		t.Fatalf("visibles = %d, attendu 1 (333, l'occupant COURANT du slot) — %+v", c.Visibles, c)
	}
	if c.PlusProcheM == nil || math.Abs(*c.PlusProcheM-3) > 0.001 {
		t.Fatalf("plus proche = %v, attendu 3 m", c.PlusProcheM)
	}
	// 222 est mort a 10 000 et n'a rien montre depuis : il attend.
	if c.EnAttente != 1 {
		t.Fatalf("en attente = %d, attendu 1 (222, mort a 10 000) — s'il est « visible », le "+
			"slot recycle a credite ses positions au PREMIER occupant", c.EnAttente)
	}
}

// TestContextesDesMorts_MortRecente_NEstPasVisible — LA RÉPLICATION SURVIT À LA MORT.
//
// Elle s'arrête ~34 ms APRÈS (médiane mesurée, `deathMatchWindowMS`), donc un coéquipier tué
// 500 ms avant moi a encore une position DANS la fenêtre de visibilité. Sans le test de
// vitalité, il sortait « visible » AVEC UNE DISTANCE, et ma mort se lisait « accompagnée ».
func TestContextesDesMorts_MortRecente_NEstPasVisible(t *testing.T) {
	var pos []filmdec.BipedPosition
	pos = append(pos, pisteMonde(1, 0, 10_000, 0, 0)...)
	pos = append(pos, pisteMonde(2, 0, 9_500, 3, 0)...) // 222 meurt a 9 500, a 3 m de moi

	mortsFilm := []Death{mortFilm(222, 9_500), mortFilm(111, 10_000)}
	journal := []MortDuJournal{
		{VictimeXUID: 222, TempsMS: 9_500}, {VictimeXUID: 111, TempsMS: 10_000},
	}
	e := dcEntree(pos, mortsFilm, journal, nil, nil)
	e.Equipes = map[uint64]int{111: 0, 222: 0}
	c := contexteDe(t, ContextesDesMorts(e), 111, 10_000)

	if c.Visibles != 0 {
		t.Fatalf("visibles = %d, attendu 0 : 222 est MORT 500 ms avant moi — sa position est "+
			"fraiche, lui non", c.Visibles)
	}
	if c.EnAttente != 1 {
		t.Fatalf("en attente = %d, attendu 1", c.EnAttente)
	}
	if c.PlusProcheM != nil {
		t.Fatalf("plus proche = %v, attendu nil : un mort n'est pas a une distance", *c.PlusProcheM)
	}
}

// TestContextesDesMorts_PasEncoreArrive_NeCompteNullePart — `joined_in_progress`.
//
// Un joueur qui rejoint APRÈS l'instant n'est pas « hors de vue » : il n'est pas dans la
// partie. Le compter ainsi le rendrait « en mesure d'accompagner » toutes les morts qui
// précèdent son arrivée — et ferait grossir le dénominateur d'un coéquipier fantôme.
func TestContextesDesMorts_PasEncoreArrive_NeCompteNullePart(t *testing.T) {
	var pos []filmdec.BipedPosition
	pos = append(pos, pisteMonde(1, 0, 10_000, 0, 0)...)
	pos = append(pos, pisteMonde(2, 0, 20_000, 3, 0)...)

	mortsFilm := []Death{mortFilm(111, 10_000), mortFilm(222, 20_000)}
	journal := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}

	e := dcEntree(pos, mortsFilm, journal, nil, map[uint64]int64{333: 15_000})
	e.Equipes = map[uint64]int{111: 0, 222: 0, 333: 0}
	c := contexteDe(t, ContextesDesMorts(e), 111, 10_000)

	if c.Total != 1 {
		t.Fatalf("total = %d, attendu 1 : 333 arrive a 15 000, il n'etait pas la a 10 000 — "+
			"il ne compte dans AUCUN etat, ni au total", c.Total)
	}
	if c.Visibles != 1 {
		t.Fatalf("visibles = %d, attendu 1 (222)", c.Visibles)
	}
}

// TestContextesDesMorts_LeDepartPrimeSurLaPosition — LA BASE FAIT FOI SUR LES DÉPARTS.
func TestContextesDesMorts_LeDepartPrimeSurLaPosition(t *testing.T) {
	var pos []filmdec.BipedPosition
	pos = append(pos, pisteMonde(1, 0, 10_000, 0, 0)...)
	pos = append(pos, pisteMonde(2, 0, 20_000, 3, 0)...)

	mortsFilm := []Death{mortFilm(111, 10_000), mortFilm(222, 20_000)}
	journal := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}

	e := dcEntree(pos, mortsFilm, journal, map[uint64]int64{222: 9_000}, nil)
	e.Equipes = map[uint64]int{111: 0, 222: 0}
	c := contexteDe(t, ContextesDesMorts(e), 111, 10_000)

	if c.Partis != 1 || c.Visibles != 0 {
		t.Fatalf("parti %d / visible %d, attendu 1/0 : le depart de la BASE prime sur ce que "+
			"le film montre encore", c.Partis, c.Visibles)
	}
	if c.PlusProcheM != nil {
		t.Fatalf("plus proche = %v, attendu nil : un joueur parti n'est pas a une distance",
			*c.PlusProcheM)
	}
}

// TestContextesDesMorts_LaFenetreDeVisibilite — LA BORNE, à la milliseconde.
func TestContextesDesMorts_LaFenetreDeVisibilite(t *testing.T) {
	for _, cas := range []struct {
		nom     string
		age     int64
		visible bool
	}{
		{"pile sur la borne", FenetreVisibiliteMs, true},
		{"une ms de trop", FenetreVisibiliteMs + 1, false},
	} {
		t.Run(cas.nom, func(t *testing.T) {
			var pos []filmdec.BipedPosition
			pos = append(pos, pisteMonde(1, 0, 10_000, 0, 0)...)
			// 222 vit d'un bout a l'autre (donc VIVANT a 10 000, UNE seule vie : le trou
			// reste sous `lifeGapUS`), mais sa REPLICATION s'interrompt juste avant ma mort.
			// La fenetre porte sur la POSITION, pas sur la vie — les deux conditions sont
			// distinctes, et c'est ce qui rend ce test different du precedent.
			pos = append(pos, pisteMonde(2, 0, 10_000-cas.age, 3, 0)...)
			pos = append(pos, pisteMonde(2, 10_100, 20_000, 3, 0)...)

			mortsFilm := []Death{mortFilm(111, 10_000), mortFilm(222, 20_000)}
			journal := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}
			e := dcEntree(pos, mortsFilm, journal, nil, nil)
			e.Equipes = map[uint64]int{111: 0, 222: 0}

			c := contexteDe(t, ContextesDesMorts(e), 111, 10_000)
			if got := c.Visibles == 1; got != cas.visible {
				t.Fatalf("position vieille de %d ms : visible = %v, attendu %v (%+v)",
					cas.age, got, cas.visible, c)
			}
		})
	}
}

// TestContextesDesMorts_MortSansLieu_NeSortPas — une mort dont le film ne montre pas la victime
// ne produit AUCUNE ligne.
//
// Écrire la ligne avec `nearest_teammate_m` NULL la ferait lire « aucun coéquipier à portée »,
// donc ISOLÉE : une absence de mesure deviendrait un verdict.
func TestContextesDesMorts_MortSansLieu_NeSortPas(t *testing.T) {
	pos, mortsFilm, _ := corpusDeReference()
	// Une mort de 111 a 15 000 : il n'est plus repliqué depuis 10 000.
	journal := []MortDuJournal{{VictimeXUID: 111, TempsMS: 15_000}}
	for _, c := range ContextesDesMorts(dcEntree(pos, mortsFilm, journal, nil, nil)) {
		if c.VictimeXUID == 111 && c.TempsMS == 15_000 {
			t.Fatalf("contexte rendu pour une mort sans lieu : %+v", c)
		}
	}
}

// TestContextesDesMorts_IndexIncoherent_RienNeSort — LE NOMMAGE LUI-MÊME EST FAUX.
//
// Une identité lue de deux façons d'un chunk à l'autre n'est pas une ambiguïté à arbitrer,
// c'est le symptôme d'une lecture cassée. Le rejeu refuse de publier ; écrire quand même des
// faits en base serait pire — ils n'ont pas d'écran pour montrer leur réserve.
func TestContextesDesMorts_IndexIncoherent_RienNeSort(t *testing.T) {
	pos, mortsFilm, journal := corpusDeReference()
	e := dcEntree(pos, mortsFilm, journal, nil, nil)
	e.Report.IndexDisagreements = 1

	if out := ContextesDesMorts(e); len(out) != 0 {
		t.Fatalf("contextes = %+v, attendu aucun : une identite lue de deux facons rend le "+
			"nommage faux", out)
	}
}

// TestContextesDesMorts_SlotRecycleNeRefusePas — LA DIFFÉRENCE ASSUMÉE AVEC LE REJEU.
//
// `SlotCollisions` dit qu'un SLOT a porté deux joueurs. Il invalide le pont APLATI que le
// document de rejeu publie ; il n'invalide pas les VIES, qui portent chacune leur occupant.
// Refuser dessus écarterait exactement les films que la correction P0-2 existe pour traiter.
func TestContextesDesMorts_SlotRecycleNeRefusePas(t *testing.T) {
	pos, mortsFilm, journal := corpusDeReference()
	e := dcEntree(pos, mortsFilm, journal, nil, nil)
	e.Report.SlotCollisions = 3

	if out := ContextesDesMorts(e); len(out) == 0 {
		t.Fatal("aucun contexte : un slot RECYCLE ne doit pas faire refuser la projection — " +
			"c'est le cas que l'attribution par vie sait traiter")
	}
}

// TestContextesDesMorts_LaSommeDesEtatsFaitLeTotal — INVARIANT.
func TestContextesDesMorts_LaSommeDesEtatsFaitLeTotal(t *testing.T) {
	pos, mortsFilm, journal := corpusDeReference()
	for _, c := range ContextesDesMorts(dcEntree(pos, mortsFilm, journal,
		map[uint64]int64{444: 9_000}, nil)) {
		if somme := c.Visibles + c.EnAttente + c.HorsDeVue + c.Partis; somme != c.Total {
			t.Fatalf("%+v : somme des etats = %d, total = %d", c, somme, c.Total)
		}
	}
}

// TestContextesDesMorts_HorlogeDuFilmConvertie — le décalage d'horloge est APPLIQUÉ.
//
// Le film démarre 4 s avant le match. Sans la conversion, les positions seraient vues 4 s dans
// le futur et n'entreraient dans aucune fenêtre : tous les coéquipiers sortiraient « hors de
// vue ». L'erreur serait un décalage CONSTANT, invisible à l'œil.
func TestContextesDesMorts_HorlogeDuFilmConvertie(t *testing.T) {
	const dec = 4_000
	var pos []filmdec.BipedPosition
	pos = append(pos, pisteMonde(1, dec, dec+10_000, 0, 0)...)
	pos = append(pos, pisteMonde(2, dec, dec+20_000, 3, 0)...)

	// Le fil des morts est sur l'horloge du MATCH : 10 000 et 20 000.
	mortsFilm := []Death{mortFilm(111, 10_000), mortFilm(222, 20_000)}
	journal := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}
	e := dcEntree(pos, mortsFilm, journal, nil, nil)
	e.Equipes = map[uint64]int{111: 0, 222: 0}

	if e.Report.DeathOffsetMS != dec {
		t.Fatalf("DeathOffsetMS = %d, attendu %d", e.Report.DeathOffsetMS, dec)
	}
	c := contexteDe(t, ContextesDesMorts(e), 111, 10_000)
	if c.Visibles != 1 {
		t.Fatalf("visible = %d, attendu 1 : l'horloge du FILM n'a pas ete convertie en horloge "+
			"du MATCH", c.Visibles)
	}
}

// contexteDe retrouve le contexte d'une mort, ou fait échouer le test.
func contexteDe(t *testing.T, out []ContexteMort, xuid uint64, tMS int64) ContexteMort {
	t.Helper()
	for _, c := range out {
		if c.VictimeXUID == xuid && c.TempsMS == tMS {
			return c
		}
	}
	t.Fatalf("aucun contexte pour la mort de %d a %d ms — rendus : %+v", xuid, tMS, out)
	return ContexteMort{}
}
