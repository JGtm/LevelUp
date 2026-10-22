// Package scheduler_test — data_health_passes_test.go : la jauge « passes par match » du contrôle
// de santé des données (data_health_passes.go) — fichier écrit pour le script hôte de nuit,
// expvar, lecteur injecté (jamais d'écrivain), retenue sur mesure partielle.

package scheduler_test

import (
	"context"
	"database/sql"
	"errors"
	"expvar"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

var horlogeJauge = time.Date(2026, 10, 22, 3, 0, 0, 0, time.UTC)

// seedBombStatsVersions : 6 lignes dans la table réelle d'Assaut, 3 servies (rapport 2).
func seedBombStatsVersions(t *testing.T, sharedPath string) {
	t.Helper()
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared RW: %v", err)
	}
	defer db.Close() //nolint:errcheck // fixture
	for i, l := range []struct{ match, xuid string }{
		{"m1", "x1"}, {"m1", "x1"}, {"m1", "x2"}, {"m2", "x1"}, {"m1", "x1"}, {"m2", "x1"},
	} {
		if _, err := db.SQLDb().Exec(`INSERT INTO match_bomb_stats (match_id, xuid, bomb_arms, written_at)
			VALUES (?, ?, 1, TIMESTAMP '2026-01-01 00:00:00' + to_seconds(?))`, l.match, l.xuid, i); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func compacter(t *testing.T, sharedPath string) {
	t.Helper()
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared RW: %v", err)
	}
	defer db.Close() //nolint:errcheck // fixture
	if _, err := migration.CompactSupersededPasses(context.Background(), db.SQLDb(), false); err != nil {
		t.Fatalf("compaction: %v", err)
	}
}

func lireJauge(t *testing.T, repoRoot string) string {
	t.Helper()
	b, err := os.ReadFile(title.NewPathResolver(repoRoot).PassesGaugePath())
	if err != nil {
		t.Fatalf("fichier de jauge: %v", err)
	}
	return string(b)
}

// lecteurCompte : lecteur de la base partagée qui compte ses appels (preuve que la jauge passe
// par le lecteur injecté, une lecture courte par table) et ouvre en lecture.
func lecteurCompte(repoRoot string, n *atomic.Int32) scheduler.SharedReaderFunc {
	pr := title.NewPathResolver(repoRoot)
	return func(_ context.Context, slug string) (*sql.DB, func(), error) {
		n.Add(1)
		return duckdb.OpenReadForQuery(pr.SharedDBPath(slug))
	}
}

// TestJaugePasses_FichierEtExpvar : au-delà du seuil, le fichier dit 1 table et porte la ligne de
// la table ; après compaction, 0. La mesure passe par le lecteur injecté, une fois par table.
func TestJaugePasses_FichierEtExpvar(t *testing.T) {
	repoRoot := t.TempDir()
	sharedPath := seedTitleSharedDB(t, repoRoot, "halo_infinite")
	seedBombStatsVersions(t, sharedPath)
	var lectures atomic.Int32
	sched := scheduler.NewDataHealthScheduler(repoRoot).
		WithSharedReader(lecteurCompte(repoRoot, &lectures)).
		WithClock(func() time.Time { return horlogeJauge })

	res := sched.RunOnce(context.Background())
	if res.PassesOverThreshold != 1 || res.PassesUnmeasured != 0 {
		t.Fatalf("au-dessus=%d non mesurées=%d, attendu 1 et 0", res.PassesOverThreshold, res.PassesUnmeasured)
	}
	if got, want := int(lectures.Load()), len(migration.CompactableTables()); got != want {
		t.Fatalf("lectures de la base partagée = %d, attendu %d (une par table du registre)", got, want)
	}
	contenu := lireJauge(t, repoRoot)
	for _, attendu := range []string{
		"mesure_utc=2026-10-22T03:00:00Z\n", "seuil=1.50\n", "tables_au_dessus_du_seuil=1\n",
		"table=halo_infinite match_bomb_stats brutes=6 gardees=3 rapport=2.00\n",
	} {
		if !strings.Contains(contenu, attendu) {
			t.Errorf("fichier de jauge sans %q :\n%s", attendu, contenu)
		}
	}
	if v := expvar.Get("levelup.compaction.tables_au_dessus_du_seuil"); v == nil || v.String() != "1" {
		t.Errorf("expvar tables_au_dessus_du_seuil = %v, attendu 1", v)
	}
	m, ok := expvar.Get("levelup.compaction.passes_par_match").(*expvar.Map)
	if !ok || m.Get("halo_infinite/match_bomb_stats") == nil || m.Get("halo_infinite/match_bomb_stats").String() != "2" {
		t.Errorf("expvar passes_par_match[halo_infinite/match_bomb_stats] absent ou ≠ 2 : %v", m)
	}

	compacter(t, sharedPath)
	res = sched.RunOnce(context.Background())
	if res.PassesOverThreshold != 0 {
		t.Fatalf("après compaction : au-dessus=%d, attendu 0", res.PassesOverThreshold)
	}
	if contenu := lireJauge(t, repoRoot); !strings.Contains(contenu, "tables_au_dessus_du_seuil=0\n") ||
		!strings.Contains(contenu, "match_bomb_stats brutes=3 gardees=3 rapport=1.00\n") {
		t.Errorf("fichier après compaction :\n%s", contenu)
	}
}

// TestJaugePasses_MesurePartielleNePublieRien : un lecteur en échec → sondes en échec, tables non
// mesurées, et AUCUN fichier écrit (« non mesuré ≠ sain » : le script ne doit pas lire 0).
func TestJaugePasses_MesurePartielleNePublieRien(t *testing.T) {
	repoRoot := t.TempDir()
	seedTitleSharedDB(t, repoRoot, "halo_infinite")
	sched := scheduler.NewDataHealthScheduler(repoRoot).
		WithSharedReader(func(context.Context, string) (*sql.DB, func(), error) {
			return nil, nil, errors.New("swap en cours")
		})
	res := sched.RunOnce(context.Background())
	if res.PassesUnmeasured != len(migration.CompactableTables()) || res.ProbeErrors < res.PassesUnmeasured {
		t.Fatalf("non mesurées=%d sondes en échec=%d, attendu %d tables non mesurées comptées en sondes",
			res.PassesUnmeasured, res.ProbeErrors, len(migration.CompactableTables()))
	}
	if _, err := os.Stat(title.NewPathResolver(repoRoot).PassesGaugePath()); !os.IsNotExist(err) {
		t.Fatalf("fichier de jauge écrit sur une mesure partielle (err=%v)", err)
	}
}

// TestJaugePasses_TitreIllisibleNePublieRien : deux titres, la base partagée du second est
// illisible (son audit échoue à l'ouverture). Le fichier ne doit PAS être publié frais avec le seul
// titre lisible : le titre illisible (peut-être le plus gonflé) n'y apparaîtrait plus et ne serait
// jamais compacté, en silence.
func TestJaugePasses_TitreIllisibleNePublieRien(t *testing.T) {
	useTwoTitleRegistry(t)
	repoRoot := t.TempDir()
	sharedPath := seedTitleSharedDB(t, repoRoot, "halo_infinite")
	seedBombStatsVersions(t, sharedPath)
	h5 := title.NewPathResolver(repoRoot).SharedDBPath("halo_5")
	if err := os.MkdirAll(filepath.Dir(h5), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(h5, []byte("ceci n'est pas une base DuckDB"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := scheduler.NewDataHealthScheduler(repoRoot).
		WithClock(func() time.Time { return horlogeJauge }).
		RunOnce(context.Background())
	if res.PassesUnmeasured == 0 {
		t.Fatalf("titre illisible non compté comme non mesuré : %+v", res)
	}
	if _, err := os.Stat(title.NewPathResolver(repoRoot).PassesGaugePath()); !os.IsNotExist(err) {
		t.Fatalf("fichier de jauge publié sans le titre illisible (err=%v)", err)
	}
}
