package service

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// TestBuildIntensityRows_OrdersOldestFirst : la heatmap d'intensité expose les
// matchs en ordre chronologique ASC (plus ancien en premier), cohérent avec
// match_rows et les autres graphes Progression — le front NE réinverse PLUS
// (même contrat que la page Escouade). Garde-fou anti-régression : un retour au
// tri DESC (bug de double inversion) ferait échouer ce test.
//
// Les MatchID sont choisis pour que leur ordre ALPHABÉTIQUE ("mid" < "new" <
// "old", l'ordre dans lequel ComputeMatchIntensityProfiles les rend) DIFFÈRE de
// l'ordre chronologique attendu ("old" < "mid" < "new") — sans le tri par
// start_time de buildIntensityRows, la sortie ne serait pas chronologique.
func TestBuildIntensityRows_OrdersOldestFirst(t *testing.T) {
	const xuid = "player1"
	tOld := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	tMid := time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)
	tNew := time.Date(2026, 1, 3, 10, 0, 0, 0, time.UTC)

	// Deux frags du joueur par match → un profil d'intensité par match.
	events := []canonical.HighlightEvent{
		{MatchID: "old", EventType: string(canonical.EventKill), TimeMS: 100, XUID: xuid},
		{MatchID: "old", EventType: string(canonical.EventKill), TimeMS: 500, XUID: xuid},
		{MatchID: "mid", EventType: string(canonical.EventKill), TimeMS: 100, XUID: xuid},
		{MatchID: "mid", EventType: string(canonical.EventKill), TimeMS: 500, XUID: xuid},
		{MatchID: "new", EventType: string(canonical.EventKill), TimeMS: 100, XUID: xuid},
		{MatchID: "new", EventType: string(canonical.EventKill), TimeMS: 500, XUID: xuid},
	}
	// `matches` arrive déjà ASC (comme GetPage le garantit). Le label suit le
	// contrat du builder ("Carte — date") : le numéro #N est posé côté web.
	matches := []legacymatch.StatsMatchRow{
		{MatchID: "old", StartTime: tOld, MapName: "Aquarius"},
		{MatchID: "mid", StartTime: tMid, MapName: "Bazaar"},
		{MatchID: "new", StartTime: tNew, MapName: "Catalyst"},
	}
	durations := map[string]int64{"old": 1000, "mid": 1000, "new": 1000}

	rows := buildIntensityRows(events, matches, xuid, durations)
	if len(rows) != 3 {
		t.Fatalf("attendu 3 rows d'intensité, obtenu %d", len(rows))
	}

	wantOrder := []string{"old", "mid", "new"} // ASC oldest-first
	for i, want := range wantOrder {
		if rows[i].MatchID != want {
			t.Errorf("rows[%d].MatchID = %q, want %q (ordre chronologique ASC attendu)", i, rows[i].MatchID, want)
		}
	}
	// Label = "Carte — JJ/MM" sans numéro (le builder web pose le #N ; le
	// doubler côté Go rendait "#1 #1 Carte" à l'écran).
	if rows[0].Label != "Aquarius — 01/01" {
		t.Errorf("rows[0].Label = %q, want %q (match le plus ancien)", rows[0].Label, "Aquarius — 01/01")
	}
	if rows[2].Label != "Catalyst — 03/01" {
		t.Errorf("rows[2].Label = %q, want %q (match le plus récent)", rows[2].Label, "Catalyst — 03/01")
	}
}
