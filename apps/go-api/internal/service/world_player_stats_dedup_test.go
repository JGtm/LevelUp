package service

import (
	"context"
	"sync"
	"testing"

	syncpkg "levelup/go-api/internal/sync"
)

// countingSource compte les GetMatchStats par matchID (validation du dédup).
type countingSource struct {
	inner      *fakeMatchSource
	mu         sync.Mutex
	statsCalls map[string]int
}

func (c *countingSource) GetMatchHistory(ctx context.Context, gt, mt string, start, count int) ([]syncpkg.MatchHistoryEntry, error) {
	return c.inner.GetMatchHistory(ctx, gt, mt, start, count)
}

func (c *countingSource) GetMatchStats(ctx context.Context, matchID string) (map[string]any, error) {
	c.mu.Lock()
	c.statsCalls[matchID]++
	c.mu.Unlock()
	return c.inner.GetMatchStats(ctx, matchID)
}

// buildMultiMatch fabrique un match avec PLUSIEURS joueurs : map xuid -> [kills, deaths, assists].
func buildMultiMatch(seasonPath, playlist string, players map[string][3]int) map[string]any {
	ps := make([]any, 0, len(players))
	for xuid, kda := range players {
		ps = append(ps, map[string]any{
			"PlayerId": "xuid(" + xuid + ")",
			"Outcome":  float64(2),
			"PlayerTeamStats": []any{map[string]any{"Stats": map[string]any{"CoreStats": map[string]any{
				"Kills": float64(kda[0]), "Deaths": float64(kda[1]), "Assists": float64(kda[2]),
			}}}},
			"ParticipationInfo": map[string]any{"TimePlayed": "PT10M"},
		})
	}
	return map[string]any{
		"MatchInfo": map[string]any{"SeasonId": seasonPath, "Playlist": map[string]any{"AssetId": playlist}},
		"Players":   ps,
	}
}

// TestSharedMatch_FetchedOnce_ProcessesAllPlayers : un match partagé par 2 joueurs
// mondiaux n'est fetché qu'UNE fois (dédup) et les 2 reçoivent leurs propres stats.
func TestSharedMatch_FetchedOnce_ProcessesAllPlayers(t *testing.T) {
	const xa, xb = "111", "222"
	m := buildMultiMatch("Csr/Seasons/CsrSeason13-2.json", tArena, map[string][3]int{
		xa: {15, 8, 4}, xb: {10, 12, 2},
	})
	inner := &fakeMatchSource{
		history: map[string][]string{"xuid(" + xa + ")": {"M"}, "xuid(" + xb + ")": {"M"}},
		stats:   map[string]map[string]any{"M": m},
	}
	src := &countingSource{inner: inner, statsCalls: map[string]int{}}
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Alpha": xa, "Beta": xb}},
		WorldStatsAggregatorConfig{TargetSeasons: map[string]bool{"csrseason13-2": true}, Concurrency: 2})

	all, errs := agg.Run(context.Background(), []string{"Alpha", "Beta"})
	if len(errs) != 0 {
		t.Fatalf("errs inattendues: %v", errs)
	}
	if src.statsCalls["M"] != 1 {
		t.Errorf("match M fetché %d fois, want 1 (dédup : 1 fetch → 2 joueurs)", src.statsCalls["M"])
	}
	byGT := map[string]int64{}
	for _, b := range all {
		byGT[b.Gamertag] = b.Kills
	}
	if byGT["Alpha"] != 15 {
		t.Errorf("Alpha kills = %d, want 15", byGT["Alpha"])
	}
	if byGT["Beta"] != 10 {
		t.Errorf("Beta kills = %d, want 10", byGT["Beta"])
	}
}

// TestRankedPlaylistsOnly : un match en playlist NON classée est ignoré (le social
// ne pollue pas l'agrégat même s'il est dans la saison cible).
func TestRankedPlaylistsOnly(t *testing.T) {
	const xuid = "42"
	const social = "social-playlist-xyz"
	src := &fakeMatchSource{
		history: map[string][]string{"xuid(" + xuid + ")": {"ranked", "social"}},
		stats: map[string]map[string]any{
			"ranked": buildMatch(xuid, "Csr/Seasons/CsrSeason13-2.json", tArena, 2, 15, 8, 4),
			"social": buildMatch(xuid, "Csr/Seasons/CsrSeason13-2.json", social, 2, 99, 1, 1),
		},
	}
	agg := NewWorldStatsAggregator(src, &fakeResolver{m: map[string]string{"Neo": xuid}},
		WorldStatsAggregatorConfig{
			TargetSeasons:   map[string]bool{"csrseason13-2": true},
			RankedPlaylists: map[string]bool{tArena: true}, // seul Arena est classé
		})

	out, err := agg.AggregatePlayer(context.Background(), "Neo")
	if err != nil {
		t.Fatalf("AggregatePlayer: %v", err)
	}
	if len(out) != 1 || out[0].PlaylistID != tArena {
		t.Fatalf("attendu 1 bucket Arena (social ignoré), got %+v", out)
	}
	if out[0].Kills != 15 {
		t.Errorf("Arena kills = %d, want 15 (le match social à 99 doit être ignoré)", out[0].Kills)
	}
}
