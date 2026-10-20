//go:build integration

// Package duckdb — match_view_repo_meta_annulation_test.go : E5.5 du plan backlog 2026-10-16.
// Une requête de vue de match abandonnée (client parti) pendant la lecture du snapshot n'est ni
// une absence ni un schéma en retard : ni bascule en lecture live, ni ERROR, ni compteur
// d'exploitation. L'erreur remonte telle quelle.
package duckdb

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"levelup/go-api/internal/observability"
)

func TestGetMatchMeta_ContexteAnnule_NiBasculeNiCompteurNiError(t *testing.T) {
	pdb := newMetaResolveTestPDB(t)
	pdb.TitleSlug = "titre-annulation-e5"
	buf := captureSlog(t)

	snapSQL, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	t.Cleanup(func() { snapSQL.Close() })
	snap := newTestDB(snapSQL, ":memory:")

	const stale = "match_view_snapshot_stale_schema_live_fallback_total"
	const miss = "match_view_snapshot_miss_live_fallback_total"
	avantStale := observability.LoadCounterT(pdb.TitleSlug, stale)
	avantMiss := observability.LoadCounterT(pdb.TitleSlug, miss)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // le client est parti avant la lecture
	repo := NewMatchViewRepo(pdb, "test-xuid").WithSharedReader(LegacySharedReader(snap))
	if _, err := repo.GetMatchMeta(ctx, "m-annule"); err == nil {
		t.Fatal("une lecture abandonnée doit remonter une erreur")
	}

	if repo.forceLive {
		t.Error("forceLive ne doit pas être armé par une requête abandonnée")
	}
	if got := observability.LoadCounterT(pdb.TitleSlug, stale); got != avantStale {
		t.Errorf("compteur %s : %d -> %d, doit rester inchangé", stale, avantStale, got)
	}
	if got := observability.LoadCounterT(pdb.TitleSlug, miss); got != avantMiss {
		t.Errorf("compteur %s : %d -> %d, doit rester inchangé", miss, avantMiss, got)
	}
	for _, l := range strings.Split(buf.String(), "\n") {
		if strings.Contains(l, `"level":"ERROR"`) || strings.Contains(l, `"level":"WARN"`) {
			t.Errorf("une requête abandonnée ne journalise ni ERROR ni WARN : %s", l)
		}
	}
}
