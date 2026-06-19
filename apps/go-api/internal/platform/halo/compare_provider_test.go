package halo

import (
	"strings"
	"testing"

	"levelup/go-api/internal/games"
)

func TestBuildSeasonServiceRecordURL(t *testing.T) {
	tru, fls := true, false
	const expectedPrefix = defaultStatsHost + "/hi/players/JGtm/Matchmade/servicerecord?"

	// Sans filtre ranked : seasonId encodé (slashes → %2F), pas de isRanked.
	// game_prefix "hi" (DefaultGamePrefix) → URL byte-identique à l'avant-refactor.
	u := buildSeasonServiceRecordURL(games.DefaultGamePrefix, "JGtm", "Seasons/Season7.json", nil)
	if !strings.HasPrefix(u, expectedPrefix) {
		t.Fatalf("préfixe inattendu: %s", u)
	}
	if !strings.Contains(u, "seasonId=Seasons%2FSeason7.json") {
		t.Errorf("seasonId mal encodé: %s", u)
	}
	if strings.Contains(u, "isRanked") {
		t.Errorf("isRanked ne doit pas apparaître quand nil: %s", u)
	}

	// Filtre classé.
	if u := buildSeasonServiceRecordURL(games.DefaultGamePrefix, "JGtm", "Seasons/Season7.json", &tru); !strings.Contains(u, "isRanked=true") {
		t.Errorf("isRanked=true attendu: %s", u)
	}
	// Filtre non-classé.
	if u := buildSeasonServiceRecordURL(games.DefaultGamePrefix, "JGtm", "Seasons/Season7.json", &fls); !strings.Contains(u, "isRanked=false") {
		t.Errorf("isRanked=false attendu: %s", u)
	}
}

func TestParseISO8601DurationSeconds(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"", 0},
		{"garbage", 0},
		{"P1DT7H50M24.6360455S", 1*86400 + 7*3600 + 50*60 + 24}, // 114624
		{"PT30M", 1800},
		{"PT1H", 3600},
		{"PT45.9S", 45},
		{"P2D", 172800},
		{"PT0S", 0},
	}
	for _, c := range cases {
		if got := parseISO8601DurationSeconds(c.in); got != c.want {
			t.Errorf("parseISO8601DurationSeconds(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
