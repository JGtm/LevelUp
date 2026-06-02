package sync

import (
	"context"
	"testing"
)

// recentMatchJSON construit un JSON de match minimal (1 joueur cible) au shape
// attendu par ExtractParticipants (CoreStats dans PlayerTeamStats[0].Stats).
func recentMatchJSON(matchID, playerID string, kills, deaths, assists, score int) map[string]any {
	return map[string]any{
		"MatchId": matchID,
		"Players": []any{
			map[string]any{
				"PlayerId":   playerID,
				"LastTeamId": float64(0),
				"Outcome":    float64(2),
				"Rank":       float64(1),
				"PlayerTeamStats": []any{
					map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
						"Kills":         float64(kills),
						"Deaths":        float64(deaths),
						"Assists":       float64(assists),
						"PersonalScore": float64(score),
						"DamageDealt":   float64(3500),
						"DamageTaken":   float64(2800),
					}}},
				},
			},
		},
	}
}

func TestBuildRecentMatchesFromStats(t *testing.T) {
	xuid := "1234567890"
	pid := "xuid(" + xuid + ")"
	history := []MatchHistoryEntry{
		{MatchID: "m-recent", StartTime: "2025-03-16T10:00:00Z"},
		{MatchID: "m-older", StartTime: "2025-03-15T10:00:00Z"},
		{MatchID: "m-nostats", StartTime: "2025-03-14T10:00:00Z"}, // absent du map → ignoré
	}
	stats := map[string]map[string]any{
		"m-recent": recentMatchJSON("m-recent", pid, 15, 8, 5, 2500),
		"m-older":  recentMatchJSON("m-older", pid, 10, 10, 2, 1500),
	}

	rows := buildRecentMatchesFromStats(history, stats, xuid)
	if len(rows) != 2 {
		t.Fatalf("attendu 2 lignes (m-nostats sans stats ignoré), got %d", len(rows))
	}
	// Ordre chronologique ASCENDANT : m-older avant m-recent.
	if rows[0].MatchID != "m-older" || rows[1].MatchID != "m-recent" {
		t.Errorf("ordre = [%s, %s], want [m-older, m-recent]", rows[0].MatchID, rows[1].MatchID)
	}

	r := rows[1] // m-recent
	if r.Kills != 15 || r.Deaths != 8 || r.Assists != 5 {
		t.Errorf("kills/deaths/assists = %d/%d/%d, want 15/8/5", r.Kills, r.Deaths, r.Assists)
	}
	if want := (15.0 + 5.0) / 8.0; r.KDA != want { // ExtractParticipants: (k+a)/d
		t.Errorf("KDA = %v, want %v", r.KDA, want)
	}
	if r.Score != 2500 {
		t.Errorf("score = %d, want 2500", r.Score)
	}
	if r.DamageDealt != 3500 || r.DamageTaken != 2800 {
		t.Errorf("damage = %d/%d, want 3500/2800", r.DamageDealt, r.DamageTaken)
	}
	if r.Outcome != 2 {
		t.Errorf("outcome = %d, want 2", r.Outcome)
	}
	if r.Rank == nil || *r.Rank != 1 {
		t.Errorf("rank = %v, want 1", r.Rank)
	}
}

func TestBuildRecentMatchesFromStats_TargetAbsent(t *testing.T) {
	history := []MatchHistoryEntry{{MatchID: "m1", StartTime: "2025-03-16T10:00:00Z"}}
	stats := map[string]map[string]any{"m1": recentMatchJSON("m1", "xuid(999)", 5, 5, 1, 100)}
	if rows := buildRecentMatchesFromStats(history, stats, "1234567890"); len(rows) != 0 {
		t.Errorf("cible absente du match → 0 ligne, got %d", len(rows))
	}
}

func TestRecentMatchesFetcher_NoAuth(t *testing.T) {
	rows, err := NewRecentMatchesFetcher(10).FetchRecentMatches(context.Background(), "1234567890", 20)
	if err != nil || rows != nil {
		t.Errorf("sans tokens en contexte → (nil, nil), got rows=%v err=%v", rows, err)
	}
}
