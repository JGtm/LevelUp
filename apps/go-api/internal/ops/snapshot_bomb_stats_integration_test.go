//go:build integration

package ops

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
)

// produireSnapshotAvecBombe monte un shared portant match_bomb_stats (deux générations pour m1,
// une pour m2 non prêt), produit un cut et rend le résolveur de chemins et la version.
func produireSnapshotAvecBombe(t *testing.T) (*title.PathResolver, int64) {
	t.Helper()
	ctx := context.Background()
	paths := title.NewPathResolver(t.TempDir(), nil)

	shared := seedFidelityShared(t)
	snapExecT(t, shared, migration.MatchBombStatsTableSQL("match_bomb_stats"))
	snapExecT(t, shared, migration.MatchBombStatsLatestViewSQL("match_bomb_stats"))
	when := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	insert := `INSERT INTO match_bomb_stats (match_id, xuid, bomb_detonations, bomb_arms, bomb_grabs,
		time_as_bomb_carrier_seconds, bomb_carriers_killed, written_at) VALUES (?, 'xKiller', ?, 2, 3, 4.5, 1, ?)`
	snapExecT(t, shared, insert, "m1", 1, when)
	snapExecT(t, shared, insert, "m1", 7, when.Add(time.Minute)) // la génération que la vue doit servir
	snapExecT(t, shared, insert, "m2", 9, when)
	player := seedPlayerDB(t, []string{"m1"}, []string{"m2"})

	res, err := ProduceSnapshot(ctx, SnapshotOptions{
		TitleSlug:    title.DefaultSlug,
		Paths:        paths,
		Shared:       fakeSharedOpener{db: shared},
		PlayerOpener: fakePlayerOpener{byGT: map[string]*sql.DB{"Solo": player}},
		Players:      []string{"Solo"},
		Now:          time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC),
	})
	if err != nil || !res.Produced {
		t.Fatalf("produce: res=%+v err=%v", res, err)
	}
	version, err := CurrentSnapshotVersion(paths, title.DefaultSlug)
	if err != nil || version == 0 {
		t.Fatalf("CurrentSnapshotVersion: %d, %v", version, err)
	}
	return paths, version
}

// TestSnapshot_ExporteLesStatsDAssaut : le cut emporte match_bomb_stats (matchs prêts
// seulement) et le lecteur en reconstruit la vue match_bomb_stats_latest (dernière génération),
// de sorte qu'un match servi du snapshot garde ses statistiques d'Assaut.
func TestSnapshot_ExporteLesStatsDAssaut(t *testing.T) {
	ctx := context.Background()
	paths, _ := produireSnapshotAvecBombe(t)
	q, err := OpenSnapshotShared(ctx, paths, title.DefaultSlug)
	if err != nil {
		t.Fatalf("OpenSnapshotShared: %v", err)
	}
	defer q.Close()

	var detonations, lignes int
	if err := q.DB.QueryRowContext(ctx,
		`SELECT bomb_detonations FROM match_bomb_stats_latest WHERE match_id = 'm1' AND xuid = 'xKiller'`).Scan(&detonations); err != nil {
		t.Fatalf("lecture de match_bomb_stats_latest : %v", err)
	}
	if detonations != 7 {
		t.Errorf("bomb_detonations = %d, attendu 7 (dernière génération)", detonations)
	}
	if err := q.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM match_bomb_stats_latest WHERE match_id = 'm2'`).Scan(&lignes); err != nil {
		t.Fatalf("lecture m2 : %v", err)
	}
	if lignes != 0 {
		t.Errorf("m2 (non prêt) = %d ligne(s), attendu 0", lignes)
	}
}

// TestSnapshot_CutAnterieurAuxStatsDAssaut_resteLisible : un cut coupé avant l'export de
// match_bomb_stats (le Parquet est absent) s'ouvre toujours — table et vue manquent au schéma
// reconstruit, comme avant l'ajout — au lieu d'être refusé (ErrSnapshotIncomplete).
func TestSnapshot_CutAnterieurAuxStatsDAssaut_resteLisible(t *testing.T) {
	ctx := context.Background()
	paths, version := produireSnapshotAvecBombe(t)
	fichier := filepath.Join(paths.SnapshotVersionDir(title.DefaultSlug, version), "shared", "match_bomb_stats.parquet")
	if err := os.Remove(fichier); err != nil {
		t.Fatalf("retrait du Parquet (simule un cut ancien) : %v", err)
	}

	q, err := OpenSnapshotShared(ctx, paths, title.DefaultSlug)
	if err != nil {
		t.Fatalf("un cut sans match_bomb_stats doit rester lisible, obtenu : %v", err)
	}
	defer q.Close()
	for _, absente := range []string{"match_bomb_stats", "match_bomb_stats_latest"} {
		var n int
		if err := q.DB.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?`, absente).Scan(&n); err != nil {
			t.Fatalf("information_schema %s : %v", absente, err)
		}
		if n != 0 {
			t.Errorf("relation %q présente alors que le cut ne l'emporte pas", absente)
		}
	}
}
