//go:build integration

package ops

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

// produceStdSnapshot seede le dataset standard (2 joueurs Alpha/Bravo, m1-m4 ; Alpha
// ready m1,m2 ; Bravo ready m2,m4 ; m3 not-ready) et produit la v1. Retourne le
// PathResolver + le slug pour la relecture.
func produceStdSnapshot(t *testing.T) (*title.PathResolver, string) {
	t.Helper()
	paths := title.NewPathResolver(t.TempDir(), nil)
	slug := title.DefaultSlug

	shared := snapOpenMem(t)
	seedSharedSchema(t, shared)
	when := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for _, m := range []string{"m1", "m2", "m3", "m4"} {
		seedSharedMatch(t, shared, m, when)
	}
	alpha := seedPlayerDB(t, []string{"m1", "m2"}, []string{"m3"})
	bravo := seedPlayerDB(t, []string{"m2", "m4"}, nil)

	res, err := ProduceSnapshot(context.Background(), SnapshotOptions{
		TitleSlug:    slug,
		Paths:        paths,
		Shared:       fakeSharedOpener{db: shared},
		PlayerOpener: fakePlayerOpener{byGT: map[string]*sql.DB{"Alpha": alpha, "Bravo": bravo}},
		Players:      []string{"Alpha", "Bravo"},
		Now:          time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC),
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce std snapshot: res=%+v err=%v", res, err)
	}
	return paths, slug
}

// TestOpenSnapshotForPlayer_RoundTrip_integration : les vues read_parquet exposent les
// MÊMES noms que les DB live, le filtrage ready est préservé, et shared+dérivés se
// joignent en une seule requête sur le handle :memory: du snapshot.
func TestOpenSnapshotForPlayer_RoundTrip_integration(t *testing.T) {
	ctx := context.Background()
	paths, slug := produceStdSnapshot(t)

	q, err := OpenSnapshotForPlayer(ctx, paths, slug, "Alpha")
	if err != nil {
		t.Fatalf("OpenSnapshotForPlayer: %v", err)
	}
	defer q.Close()
	if q.Version != 1 {
		t.Fatalf("Version = %d, attendu 1", q.Version)
	}

	count := func(query string) int {
		t.Helper()
		var n int
		if err := q.DB.QueryRowContext(ctx, query).Scan(&n); err != nil {
			t.Fatalf("query %q: %v", query, err)
		}
		return n
	}

	// Faits shared globaux : 3 matchs ready (m1,m2,m4), m3 exclu.
	if n := count(`SELECT COUNT(*) FROM match_registry`); n != 3 {
		t.Errorf("match_registry = %d, attendu 3", n)
	}
	if n := count(`SELECT COUNT(*) FROM match_registry WHERE match_id = 'm3'`); n != 0 {
		t.Errorf("m3 (not-ready) présent dans le snapshot shared")
	}
	// Dérivés _latest du joueur Alpha : m1,m2 (ses ready).
	if n := count(`SELECT COUNT(*) FROM player_match_enrichment_latest`); n != 2 {
		t.Errorf("player_match_enrichment_latest (Alpha) = %d, attendu 2", n)
	}
	if n := count(`SELECT COUNT(*) FROM match_skill_rank_latest`); n != 2 {
		t.Errorf("match_skill_rank_latest (Alpha) = %d, attendu 2", n)
	}
	// Jointure shared + dérivés EN UNE REQUÊTE (impossible en live = 2 DB séparées).
	joined := count(`
		SELECT COUNT(*) FROM match_registry r
		JOIN player_match_enrichment_latest p ON p.match_id = r.match_id`)
	if joined != 2 {
		t.Errorf("jointure shared×dérivés (Alpha) = %d, attendu 2 (m1,m2)", joined)
	}
}

// TestOpenSnapshotForPlayer_NoSnapshot_integration : aucune version active → ErrNoSnapshot
// (le caller dégrade vers la lecture live).
func TestOpenSnapshotForPlayer_NoSnapshot_integration(t *testing.T) {
	paths := title.NewPathResolver(t.TempDir(), nil)
	_, err := OpenSnapshotForPlayer(context.Background(), paths, title.DefaultSlug, "Alpha")
	if !errors.Is(err, ErrNoSnapshot) {
		t.Fatalf("err = %v, attendu ErrNoSnapshot", err)
	}
}
