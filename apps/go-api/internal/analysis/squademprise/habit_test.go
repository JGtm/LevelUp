package squademprise

import (
	"fmt"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

// soireeFilmee ajoute une soirée à l'entrée : un match par famille, bonus pris usCamo / themCamo.
func soireeFilmee(in *Input, label string, debut time.Time, famille string, usCamo, themCamo int) Match {
	id := fmt.Sprintf("%s-%s", label, famille)
	m := Match{MatchID: id, StartTime: debut, SessionLabel: label, Family: famille}
	in.Timeline = append(in.Timeline, m)
	in.Film.Films[id] = sessionusage.FilmRow{MatchID: id}
	in.Film.Participants = append(in.Film.Participants, participants(id)...)
	in.Film.Players = append(in.Film.Players,
		sessionusage.PlayerRow{MatchID: id, XUID: "P", TakenByFamily: map[string]int{camo: usCamo}},
		sessionusage.PlayerRow{MatchID: id, XUID: "E1", TakenByFamily: map[string]int{camo: themCamo}})
	return m
}

func TestHabit_SoireesComparablesEtCompteDeSession(t *testing.T) {
	in := entreeUnMatch()
	in.Current[0].StartTime = t0
	in.Timeline = append(in.Timeline, in.Current...)
	in.SessionMatchCounts = map[string]int{"s": 7, "a": 4}
	// Soirée « a » : un match Assassin (comparable) et un Bases (écarté).
	soireeFilmee(&in, "a", t0.Add(-72*time.Hour), "Assassin", 3, 1)
	soireeFilmee(&in, "a", t0.Add(-71*time.Hour), "Bases", 9, 0)
	// Soirée « b » : Bases seulement — aucune famille de ce soir : hors comparaison, lue sur ses
	// matchs filmés.
	soireeFilmee(&in, "b", t0.Add(-48*time.Hour), "Bases", 5, 5)
	// Soirée « c » : Assassin mais sans film.
	in.Timeline = append(in.Timeline, Match{MatchID: "c1", StartTime: t0.Add(-24 * time.Hour), SessionLabel: "c", Family: "Assassin"})
	// Soirée « d » : APRÈS le périmètre — jamais « précédente ».
	soireeFilmee(&in, "d", t0.Add(24*time.Hour), "Assassin", 1, 1)

	h := Build(in).Habit
	if h == nil {
		t.Fatal("habitude absente")
	}
	if len(h.Families) != 1 || h.Families[0] != "Assassin" {
		t.Errorf("familles = %v", h.Families)
	}
	if h.Current.SessionLabel != "s" || h.Current.MatchCount != 7 || h.Current.MeasuredMatches != 1 || !h.Current.Comparable {
		t.Errorf("ce soir = %+v", h.Current)
	}
	if len(h.Previous) != 2 || h.Previous[0].SessionLabel != "a" || h.Previous[0].MatchCount != 4 ||
		h.Previous[1].SessionLabel != "b" {
		t.Fatalf("soirées précédentes = %+v, attendu « a » puis « b » (« c » sans film)", h.Previous)
	}
	a, b := h.Previous[0], h.Previous[1]
	if !a.Comparable || len(a.Families) != 1 || a.Families[0] != "Assassin" {
		t.Errorf("« a » = %+v, attendu comparable, lue sur Assassin", a)
	}
	if len(a.Shares) != 1 || a.Shares[0].Taken != (domain.SquadEmpriseCount{Us: 3, Them: 1}) || math.Abs(a.Shares[0].Share-0.75) > 1e-9 {
		t.Errorf("part de « a » = %+v, attendu 3 / 1 (le match Bases écarté)", a.Shares)
	}
	if b.Comparable || len(b.Families) != 1 || b.Families[0] != "Bases" || b.MeasuredMatches != 1 {
		t.Errorf("« b » = %+v, attendu hors comparaison, lue sur Bases", b)
	}
	if len(b.Shares) != 1 || b.Shares[0].Taken != (domain.SquadEmpriseCount{Us: 5, Them: 5}) {
		t.Errorf("part de « b » = %+v, attendu 5 / 5", b.Shares)
	}
	ids := HabitCandidates(chronologique(in.Current), in.Timeline)
	if len(ids) != 4 {
		t.Errorf("candidats = %v, attendu les deux matchs de « a », celui de « b » et c1", ids)
	}
}

func TestHabit_DixSoireesAuPlus(t *testing.T) {
	in := entreeUnMatch()
	in.Timeline = append(in.Timeline, in.Current...)
	for i := 0; i < 12; i++ {
		soireeFilmee(&in, fmt.Sprintf("p%02d", i), t0.Add(time.Duration(i-20)*24*time.Hour), "Assassin", 1, 1)
	}
	h := Build(in).Habit
	if len(h.Previous) != domain.SquadEmpriseHabitMaxPrevious || h.Previous[0].SessionLabel != "p02" ||
		h.Previous[9].SessionLabel != "p11" {
		t.Errorf("soirées = %d, de %s à %s ; attendu les dix plus récentes", len(h.Previous),
			h.Previous[0].SessionLabel, h.Previous[len(h.Previous)-1].SessionLabel)
	}
}

func TestHabit_SansHistoriquePasDHabitude(t *testing.T) {
	if h := Build(entreeUnMatch()).Habit; h != nil {
		t.Errorf("habitude publiée sans historique (sans coéquipier sélectionné) : %+v", h)
	}
}

// TestHabit_FiltrePartiel_LaSoireeAfficheeNEstJamaisPrecedente — constat R10 de la revue L6.1 : un
// filtre partiel écarte le PREMIER match de la soirée affichée. La soirée commence alors avant le
// premier match du périmètre : seule la règle « une soirée qui contient un match du périmètre
// n'est pas précédente » l'empêche de se comparer à elle-même.
func TestHabit_FiltrePartiel_LaSoireeAfficheeNEstJamaisPrecedente(t *testing.T) {
	in := entreeUnMatch()                                             // m1, session « s », à t0
	soireeFilmee(&in, "s", t0.Add(-30*time.Minute), "Assassin", 1, 1) // premier match de « s », hors filtre
	in.Timeline = append(in.Timeline, in.Current...)
	soireeFilmee(&in, "a", t0.Add(-48*time.Hour), "Assassin", 3, 1)
	h := Build(in).Habit
	if h == nil {
		t.Fatal("habitude absente")
	}
	for _, e := range h.Previous {
		if e.SessionLabel == "s" {
			t.Fatalf("la soirée affichée « s » figure parmi les soirées précédentes : %+v", h.Previous)
		}
	}
	if len(h.Previous) != 1 || h.Previous[0].SessionLabel != "a" {
		t.Errorf("soirées précédentes = %+v, attendu la seule « a »", h.Previous)
	}
}

// TestHabit_SoireeComparableSansFilmLueHorsComparaison — une soirée dont les matchs de la famille
// de ce soir ne sont pas filmés, mais dont un autre match l'est : lue sur ses matchs filmés, hors
// comparaison, ses familles sont celles des seuls matchs lus.
func TestHabit_SoireeComparableSansFilmLueHorsComparaison(t *testing.T) {
	in := entreeUnMatch()
	in.Current[0].StartTime = t0
	in.Timeline = append(in.Timeline, in.Current...)
	in.Timeline = append(in.Timeline, Match{MatchID: "e1", StartTime: t0.Add(-49 * time.Hour), SessionLabel: "e", Family: "Assassin"})
	soireeFilmee(&in, "e", t0.Add(-48*time.Hour), "Bases", 2, 2)
	h := Build(in).Habit
	if h == nil || len(h.Previous) != 1 {
		t.Fatalf("habitude = %+v, attendu la seule soirée « e »", h)
	}
	e := h.Previous[0]
	if e.Comparable || len(e.Families) != 1 || e.Families[0] != "Bases" || e.MeasuredMatches != 1 {
		t.Errorf("« e » = %+v, attendu hors comparaison, lue sur Bases seulement", e)
	}
}
