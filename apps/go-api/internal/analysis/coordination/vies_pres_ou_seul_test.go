package coordination

import (
	"testing"

	"levelup/go-api/internal/domain"
)

// (`metres` est celui d'isolation_test.go.)

func camp(v int) *int { return &v }

// fragAdverse — un frag du joueur (camp 0) sur un adversaire (camp 1).
func fragAdverse(match string, t int64) domain.FragLu {
	return domain.FragLu{MatchID: match, TimeMS: t, CampTueur: camp(0), CampVictime: camp(1)}
}

// temoinSixVies — le témoin chiffré à la main (portée 18 m sur m1 et m2, aucune sur m3) :
//
//	m1  vie A [0, 60 s) mort à 50 s, coéquipier à 10 m  -> PRÈS, 2 frags (10 s, 55 s posthume)
//	    vie B [60 s, 120 s) mort à 110 s, à 18 m pile     -> PRÈS (borne incluse), 1 frag
//	    vie C [120 s, …) fin du film (survivant)          -> pas une vie terminée par une mort
//	m2  vie D [0, 40 s) mort à 30 s, à 25 m               -> SEUL, 3 frags (dont un au-delà de la
//	                                                         mort, avant la vie suivante)
//	    vie E [40 s, …) mort à 90 s, distance NULL        -> écartée (aucun coéquipier situé)
//	m3  vie F [0, …) mort à 20 s, à 5 m                   -> écartée (match sans portée)
//
// Plus : une trahison et un frag au camp inconnu (écartés), un frag antérieur à la première vie.
func temoinSixVies() (domain.ViesLues, map[string]float64) {
	l := domain.ViesLues{
		Vies: []domain.VieLue{
			{MatchID: "m1", StartMS: 0, EndCause: CauseVieMort},
			{MatchID: "m1", StartMS: 60_000, EndCause: CauseVieMort},
			{MatchID: "m1", StartMS: 120_000, EndCause: "film_end"},
			{MatchID: "m2", StartMS: 0, EndCause: CauseVieMort},
			{MatchID: "m2", StartMS: 40_000, EndCause: CauseVieMort},
			{MatchID: "m3", StartMS: 0, EndCause: CauseVieMort},
		},
		Morts: []domain.MortSituee{
			{MatchID: "m1", TimeMS: 50_000, PlusProcheM: metres(10)},
			{MatchID: "m1", TimeMS: 110_000, PlusProcheM: metres(18)},
			{MatchID: "m2", TimeMS: 30_000, PlusProcheM: metres(25)},
			{MatchID: "m2", TimeMS: 90_000},
			{MatchID: "m3", TimeMS: 20_000, PlusProcheM: metres(5)},
		},
		Frags: []domain.FragLu{
			fragAdverse("m1", 10_000), fragAdverse("m1", 55_000), fragAdverse("m1", 70_000),
			fragAdverse("m1", 130_000), // vie C : pas une vie terminée par une mort
			fragAdverse("m2", 5_000), fragAdverse("m2", 20_000), fragAdverse("m2", 35_000),
			fragAdverse("m2", 50_000), // vie E, écartée
			fragAdverse("m3", 10_000), // vie F, écartée
			{MatchID: "m1", TimeMS: 80_000, CampTueur: camp(0), CampVictime: camp(0)}, // trahison
			{MatchID: "m1", TimeMS: 85_000, CampTueur: camp(0)},                       // victime sans camp
		},
	}
	return l, map[string]float64{"m1": 18, "m2": 18}
}

func TestViesPresOuSeul_TemoinSixVies(t *testing.T) {
	l, rayons := temoinSixVies()
	got, ecartes := ViesPresOuSeul(l, rayons)
	want := domain.TimeseriesLivesNearTeammate{
		Near:              domain.LivesSideCount{Lives: 2, Kills: 3},
		Alone:             domain.LivesSideCount{Lives: 1, Kills: 3},
		ExcludedUnlocated: 1, ExcludedNoRadar: 1, MatchesRead: 3, MatchesWithoutRadar: 1,
	}
	if got != want {
		t.Errorf("vies = %+v\nattendu %+v", got, want)
	}
	if ecartes != 2 {
		t.Errorf("frags écartés = %d, attendu 2 (une trahison, une victime sans camp)", ecartes)
	}
}

// La fenêtre d'une vie court jusqu'au DÉBUT de la suivante, pas jusqu'à sa fin : un frag posthume
// (grenade, échange) est celui de la vie qui vient de finir.
func TestViesPresOuSeul_FragPosthume(t *testing.T) {
	l := domain.ViesLues{
		Vies:  []domain.VieLue{{MatchID: "m", StartMS: 0, EndCause: CauseVieMort}, {MatchID: "m", StartMS: 10_000, EndCause: "film_end"}},
		Morts: []domain.MortSituee{{MatchID: "m", TimeMS: 5_000, PlusProcheM: metres(3)}},
		Frags: []domain.FragLu{fragAdverse("m", 6_000)},
	}
	got, _ := ViesPresOuSeul(l, map[string]float64{"m": 18})
	if got.Near != (domain.LivesSideCount{Lives: 1, Kills: 1}) {
		t.Errorf("près = %+v, attendu 1 vie et son frag posthume", got.Near)
	}
}

// Une vie terminée par une mort sans ligne de contexte dans sa fenêtre est écartée et comptée.
func TestViesPresOuSeul_SansContexteEcartee(t *testing.T) {
	l := domain.ViesLues{Vies: []domain.VieLue{{MatchID: "m", StartMS: 0, EndCause: CauseVieMort}}}
	got, _ := ViesPresOuSeul(l, map[string]float64{"m": 18})
	if got.ExcludedUnlocated != 1 || got.Near.Lives+got.Alone.Lives != 0 {
		t.Errorf("vies = %+v, attendu 1 écartée faute de coéquipier situé", got)
	}
}

// Aucune lecture : tout à zéro, rien d'inventé.
func TestViesPresOuSeul_Vide(t *testing.T) {
	got, ecartes := ViesPresOuSeul(domain.ViesLues{}, nil)
	if got != (domain.TimeseriesLivesNearTeammate{}) || ecartes != 0 {
		t.Errorf("(%+v, %d), attendu des zéros", got, ecartes)
	}
}
