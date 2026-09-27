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
	// Soirée « b » : Bases seulement — aucune famille de ce soir, pas de point.
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
	if h.Current.SessionLabel != "s" || h.Current.MatchCount != 7 || h.Current.MeasuredMatches != 1 {
		t.Errorf("ce soir = %+v", h.Current)
	}
	if len(h.Previous) != 1 || h.Previous[0].SessionLabel != "a" || h.Previous[0].MatchCount != 4 {
		t.Fatalf("soirées précédentes = %+v, attendu la seule « a »", h.Previous)
	}
	share := h.Previous[0].Shares
	if len(share) != 1 || share[0].Taken != (domain.SquadEmpriseCount{Us: 3, Them: 1}) || math.Abs(share[0].Share-0.75) > 1e-9 {
		t.Errorf("part de « a » = %+v, attendu 3 / 1 (le match Bases écarté)", share)
	}
	ids := HabitCandidates(chronologique(in.Current), in.Timeline)
	if len(ids) != 2 {
		t.Errorf("candidats = %v, attendu le match Assassin de « a » et c1", ids)
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
