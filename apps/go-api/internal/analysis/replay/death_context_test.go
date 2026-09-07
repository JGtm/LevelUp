package replay

// death_context_test.go — LES QUATRE ÉTATS D'UN COÉQUIPIER, chacun avec son mode de panne.
//
// Chaque état existe parce que le confondre avec un autre produit un faux verdict :
//
//	visible confondu avec hors de vue    une mort à 3 m d'un coéquipier sort « isolée » ;
//	hors de vue confondu avec en attente un coéquipier en véhicule est compté mort, et la mort
//	                                     sort « équipe à terre » (défaut P0 de la ronde 1) ;
//	parti confondu avec hors de vue      un joueur déconnecté à la première minute reste un
//	                                     coéquipier disponible jusqu'à la fin.

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

// dcEntree monte l'entrée de référence : moi (slot 1) et trois coéquipiers (slots 2, 3, 4),
// tous dans l'équipe 0 ; un adversaire (slot 9) en équipe 1. Décalage d'horloge nul.
func dcEntree(pos []filmdec.BipedPosition, journal []MortDuJournal,
	departs map[uint64]int64,
) EntreeContexteMorts {
	return EntreeContexteMorts{
		Positions:  pos,
		SlotXUID:   map[uint32]uint64{1: 111, 2: 222, 3: 333, 4: 444, 9: 999},
		DecalageMS: 0,
		Journal:    journal,
		Equipes:    map[uint64]int{111: 0, 222: 0, 333: 0, 444: 0, 999: 1},
		DepartMS:   departs,
	}
}

// TestContextesDesMorts_LesQuatreEtats — LE CAS DE RÉFÉRENCE, les quatre à la fois.
//
// Je meurs à 10 000 ms en (0,0). À cet instant :
//
//	222   répliqué à 9 800 ms en (3,0)   -> VISIBLE, à 3 m
//	333   mort à 6 000 ms, plus rien     -> EN ATTENTE
//	444   plus répliqué depuis 4 000 ms  -> HORS DE VUE (véhicule), et VIVANT
//	999   adversaire visible à 1 m       -> n'entre nulle part : il n'accompagne personne
//
// Un cinquième joueur PARTI est ajouté par le test suivant, pour que celui-ci reste lisible.
func TestContextesDesMorts_LesQuatreEtats(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posMonde(1, 9_900, 0, 0),   // moi, juste avant ma mort
		posMonde(2, 9_800, 3, 0),   // coéquipier visible à 3 m
		posMonde(3, 5_000, 20, 20), // mort à 6 000, sa dernière position est AVANT
		posMonde(4, 4_000, 40, 40), // plus répliqué depuis 6 s : véhicule
		posMonde(9, 9_900, 1, 0),   // adversaire tout proche
	}
	journal := []MortDuJournal{
		{VictimeXUID: 333, TempsMS: 6_000},
		{VictimeXUID: 111, TempsMS: 10_000},
	}
	out := ContextesDesMorts(dcEntree(pos, journal, nil))

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

// TestContextesDesMorts_LeDepartPrimeSurLaPosition — LA BASE FAIT FOI SUR LES DÉPARTS.
//
// Le coéquipier 222 est répliqué à 3 m juste avant ma mort, mais la base dit qu'il a quitté à
// 9 000 ms. Il ne peut plus accompagner : sans cette priorité, un joueur déconnecté resterait un
// coéquipier disponible jusqu'à la fin du match, et toutes les morts suivantes passeraient pour
// accompagnées.
func TestContextesDesMorts_LeDepartPrimeSurLaPosition(t *testing.T) {
	pos := []filmdec.BipedPosition{posMonde(1, 9_900, 0, 0), posMonde(2, 9_800, 3, 0)}
	journal := []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}
	out := ContextesDesMorts(dcEntree(pos, journal, map[uint64]int64{222: 9_000}))

	c := contexteDe(t, out, 111, 10_000)
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
//
// Une position vieille d'exactement `FenetreVisibiliteMs` compte encore ; une milliseconde de
// plus, non. Sans borne stricte, la dernière position connue d'un joueur vaudrait pour tout le
// reste du match — un stationnement posthume.
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
			pos := []filmdec.BipedPosition{
				posMonde(1, 10_000, 0, 0),
				posMonde(2, 10_000-cas.age, 3, 0),
			}
			out := ContextesDesMorts(dcEntree(pos,
				[]MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}, nil))
			c := contexteDe(t, out, 111, 10_000)
			if got := c.Visibles == 1; got != cas.visible {
				t.Fatalf("position vieille de %d ms : visible = %v, attendu %v",
					cas.age, got, cas.visible)
			}
		})
	}
}

// TestContextesDesMorts_ReapparuNEstPasEnAttente — une mort SUIVIE d'une réapparition ne met
// personne en attente.
//
// L'OBSERVATION GAGNE SUR LA DÉDUCTION : le coéquipier 222 est mort à 3 000 ms, mais le film le
// remontre à 9 800 ms. Il est donc revenu, et il est VISIBLE — pas « en attente depuis 3 000 ».
func TestContextesDesMorts_ReapparuNEstPasEnAttente(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posMonde(1, 9_900, 0, 0),
		posMonde(2, 2_000, 50, 50), // avant sa mort
		posMonde(2, 9_800, 3, 0),   // apres sa reapparition
	}
	journal := []MortDuJournal{
		{VictimeXUID: 222, TempsMS: 3_000},
		{VictimeXUID: 111, TempsMS: 10_000},
	}
	c := contexteDe(t, ContextesDesMorts(dcEntree(pos, journal, nil)), 111, 10_000)
	if c.Visibles != 1 || c.EnAttente != 0 {
		t.Fatalf("visible %d / attente %d, attendu 1/0 : une position posterieure a la mort "+
			"prouve la reapparition", c.Visibles, c.EnAttente)
	}
}

// TestContextesDesMorts_MortSansLieu_NeSortPas — une mort dont le film ne montre pas la victime
// ne produit AUCUNE ligne.
//
// Écrire la ligne avec `nearest_teammate_m` NULL la ferait lire « aucun coéquipier à portée »,
// donc ISOLÉE : une absence de mesure deviendrait un verdict.
func TestContextesDesMorts_MortSansLieu_NeSortPas(t *testing.T) {
	pos := []filmdec.BipedPosition{posMonde(2, 9_800, 3, 0)} // la victime n'est jamais repliquee
	out := ContextesDesMorts(dcEntree(pos,
		[]MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}, nil))
	if len(out) != 0 {
		t.Fatalf("contextes = %+v, attendu aucun : la mort n'a pas de lieu", out)
	}
}

// TestContextesDesMorts_LaSommeDesEtatsFaitLeTotal — INVARIANT.
//
// Un état non nommé se cacherait dans l'écart, et le persister refuserait la passe entière.
func TestContextesDesMorts_LaSommeDesEtatsFaitLeTotal(t *testing.T) {
	pos := []filmdec.BipedPosition{
		posMonde(1, 9_900, 0, 0), posMonde(2, 9_800, 3, 0),
		posMonde(3, 5_000, 20, 20), posMonde(4, 4_000, 40, 40),
	}
	journal := []MortDuJournal{
		{VictimeXUID: 333, TempsMS: 6_000}, {VictimeXUID: 111, TempsMS: 10_000},
	}
	for _, c := range ContextesDesMorts(dcEntree(pos, journal, map[uint64]int64{444: 9_000})) {
		if somme := c.Visibles + c.EnAttente + c.HorsDeVue + c.Partis; somme != c.Total {
			t.Fatalf("%+v : somme des etats = %d, total = %d", c, somme, c.Total)
		}
	}
}

// TestContextesDesMorts_HorlogeDuFilmConvertie — le décalage d'horloge est APPLIQUÉ.
//
// Le film démarre 4 s avant le match : une position à 13 800 ms de film vaut 9 800 ms de match,
// donc elle est visible pour une mort à 10 000. Sans la conversion, elle serait vue 4 s dans le
// futur et n'entrerait dans aucune fenêtre — tous les coéquipiers sortiraient « hors de vue ».
func TestContextesDesMorts_HorlogeDuFilmConvertie(t *testing.T) {
	e := dcEntree([]filmdec.BipedPosition{
		posMonde(1, 13_900, 0, 0), posMonde(2, 13_800, 3, 0),
	}, []MortDuJournal{{VictimeXUID: 111, TempsMS: 10_000}}, nil)
	e.DecalageMS = 4_000

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
