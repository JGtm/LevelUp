package sync

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

const friendsTestGamertag = "FriendOne"

// seedFriendsRecompute prépare une base partagée de n matchs (le joueur de test en équipe 0
// avec opponent_2, opponent_4, opponent_6), l'alias de l'ami opponent_2, et une base joueur
// dont chaque match porte is_with_friends = FALSE.
func seedFriendsRecompute(t *testing.T, sharedPath, playerPath string, n int) {
	t.Helper()
	seedRecomputerShared(t, sharedPath, n)

	shared, err := OpenSharedDB(sharedPath)
	if err != nil {
		t.Fatalf("OpenSharedDB: %v", err)
	}
	if _, err := shared.SQLDb().Exec(`INSERT INTO xuid_aliases (xuid, gamertag) VALUES (?, ?)`,
		"opponent_2", friendsTestGamertag); err != nil {
		t.Fatalf("insert xuid_aliases: %v", err)
	}
	if err := shared.Close(); err != nil {
		t.Fatalf("close shared: %v", err)
	}

	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()
	for i := range n {
		mid := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		if _, err := player.SQLDb().Exec(
			`INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES (?, FALSE)`, mid); err != nil {
			t.Fatalf("insert player_match_enrichment: %v", err)
		}
	}
}

func countWithFriends(t *testing.T, playerPath string) int {
	t.Helper()
	player, err := OpenPlayerDB(playerPath)
	if err != nil {
		t.Fatalf("OpenPlayerDB: %v", err)
	}
	defer player.Close()
	var n int
	if err := player.SQLDb().QueryRow(
		`SELECT COUNT(*) FROM player_match_enrichment_latest WHERE is_with_friends = TRUE`).Scan(&n); err != nil {
		t.Fatalf("lecture is_with_friends: %v", err)
	}
	return n
}

// Le recalcul « avec amis » lit la base partagée et n'écrit qu'en base joueur : il
// n'acquiert jamais le writer partagé (une bascule en écriture ferait répondre 503 aux
// lecteurs de tout le site) et la base partagée reste en lecture seule.
func TestRecomputeIsWithFriends_NeverAcquiresTheSharedWriter(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedFriendsRecompute(t, sharedPath, playerPath, 6)

	base, err := sharedprovider.New(sharedPath)
	if err != nil {
		t.Fatalf("sharedprovider.New: %v", err)
	}
	t.Cleanup(func() { _ = base.Close() })
	spy := &spySharedProvider{Provider: base}

	res, err := RecomputeIsWithFriends(context.Background(), spy, playerPath, sharedPath,
		recomputerTestXUID, []string{friendsTestGamertag})
	if err != nil {
		t.Fatalf("RecomputeIsWithFriends: %v", err)
	}

	if got := spy.writers.Load(); got != 0 {
		t.Fatalf("writer partagé acquis %d fois, attendu 0", got)
	}
	if got := spy.readers.Load(); got < 1 {
		t.Fatalf("lecteur du fournisseur jamais utilisé (%d)", got)
	}
	if res.MatchesPromoted != 6 {
		t.Fatalf("matchs promus = %d, attendu 6", res.MatchesPromoted)
	}
	if got := countWithFriends(t, playerPath); got != 6 {
		t.Fatalf("is_with_friends = TRUE sur %d matchs en base joueur, attendu 6", got)
	}
	if base.State() != sharedprovider.StateRO {
		t.Fatalf("la base partagée n'est pas restée en lecture seule: %v", base.State())
	}
}

// Sans fournisseur (mode historique), la base partagée est lue par OpenReadForQuery : le
// recalcul réussit sans prendre le bail d'écriture de la base partagée.
func TestRecomputeIsWithFriends_NoProviderLeavesTheSharedLeaseFree(t *testing.T) {
	dir := t.TempDir()
	sharedPath := filepath.Join(dir, "shared_matches_v2.duckdb")
	playerPath := filepath.Join(dir, "player", "stats.duckdb")
	seedFriendsRecompute(t, sharedPath, playerPath, 6)

	if _, err := RecomputeIsWithFriends(context.Background(), nil, playerPath, sharedPath,
		recomputerTestXUID, []string{friendsTestGamertag}); err != nil {
		t.Fatalf("RecomputeIsWithFriends: %v", err)
	}
	if got := countWithFriends(t, playerPath); got != 6 {
		t.Fatalf("is_with_friends = TRUE sur %d matchs en base joueur, attendu 6", got)
	}
	lease, err := dblease.AcquireWriterCtx(context.Background(), nil, sharedPath, dblease.KindSharedMatches)
	if err != nil {
		t.Fatalf("le bail d'écriture partagé n'est pas libre après le recalcul: %v", err)
	}
	lease.Release()
}
