// Package scheduler_test — data_health_check_e2e_test.go : tests E2E du
// HealthScheduler.
//
// Depuis 2026-05-20 le scheduler n'émet plus de notification utilisateur
// (cf. data_health_check.go pour la décision). Les tests se concentrent
// désormais sur le calcul des compteurs et l'absence d'effets de bord.

package scheduler_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
)

// healthE2ESetup prépare l'arborescence repoRoot/data/titles/halo_infinite/
// avec shared_matches_v2.duckdb (TargetShared). Retourne repoRoot.
func healthE2ESetup(t *testing.T) string {
	t.Helper()
	repoRoot := t.TempDir()
	titleDir := filepath.Join(repoRoot, "data", "titles", "halo_infinite")
	warehouseDir := filepath.Join(titleDir, "warehouse")
	if err := os.MkdirAll(warehouseDir, 0o755); err != nil {
		t.Fatalf("mkdir warehouse: %v", err)
	}

	sharedPath := filepath.Join(warehouseDir, "shared_matches_v2.duckdb")
	shared, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared_matches_v2: %v", err)
	}
	if err := migration.RunForDB(shared.SQLDb(), migration.TargetShared); err != nil {
		shared.Close()
		t.Fatalf("RunForDB(Shared): %v", err)
	}
	shared.Close()

	return repoRoot
}

// seedRawUUIDMapName injecte une anomalie UUID brut dans match_registry.
func seedRawUUIDMapName(t *testing.T, sharedPath string) {
	t.Helper()
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared rw: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(context.Background(), `
		INSERT INTO match_registry (match_id, map_name, pair_name)
		VALUES ('match-uuid-anomaly', '12345678-1234-1234-1234-1234567890ab', 'normal_pair')
	`)
	if err != nil {
		t.Fatalf("seed raw UUID map_name: %v", err)
	}
}

// seedOrphanParticipant injecte un participant dont le xuid n'a AUCUN alias
// résolu → compté comme orphelin par data_health (et masqué "Joueur ####" à
// l'affichage, cf. fix XUID 2026-05-30). xuid réaliste (16 chiffres, non-bot).
func seedOrphanParticipant(t *testing.T, sharedPath string) {
	t.Helper()
	db, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared rw: %v", err)
	}
	defer db.Close()
	_, err = db.Exec(context.Background(), `
		INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome)
		VALUES ('match-orphan', '2533274800099999', NULL, 0, 2)
	`)
	if err != nil {
		t.Fatalf("seed orphan participant: %v", err)
	}
}

// ─── 1. RunOnce sur DB vierge : compteurs à 0 ──────────────────────────────

func TestHealthScheduler_E2E_NoWarnings(t *testing.T) {
	repoRoot := healthE2ESetup(t)

	sched := scheduler.NewDataHealthScheduler(repoRoot)
	res := sched.RunOnce(context.Background())

	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	if res.WarningsTotal != 0 {
		t.Errorf("WarningsTotal: attendu 0 sur DB vierge, obtenu %d", res.WarningsTotal)
	}
}

// ─── 2. Anomalie détectée mais aucun effet de bord (plus d'émission notif) ─

func TestHealthScheduler_E2E_WithAnomaly_DetectsAndLogs(t *testing.T) {
	repoRoot := healthE2ESetup(t)
	sharedPath := filepath.Join(repoRoot, "data", "titles", "halo_infinite",
		"warehouse", "shared_matches_v2.duckdb")
	seedRawUUIDMapName(t, sharedPath)

	sched := scheduler.NewDataHealthScheduler(repoRoot)

	// Cycle 1 : doit détecter l'anomalie et l'inclure dans le total.
	res := sched.RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	if res.UUIDsRawCount < 1 {
		t.Errorf("UUIDsRawCount: attendu >= 1, obtenu %d", res.UUIDsRawCount)
	}
	if res.WarningsTotal < 1 {
		t.Errorf("WarningsTotal: attendu >= 1, obtenu %d", res.WarningsTotal)
	}

	// Cycle 2 : l'anomalie est toujours là → même résultat (idempotent).
	res2 := sched.RunOnce(context.Background())
	if res2.WarningsTotal != res.WarningsTotal {
		t.Errorf("cycle 2 WarningsTotal: attendu %d, obtenu %d",
			res.WarningsTotal, res2.WarningsTotal)
	}
}

// ─── 3. xuids orphelins : comptés + loggés, mais hors WarningsTotal ─────────

func TestHealthScheduler_E2E_OrphanXUIDs_Counted(t *testing.T) {
	repoRoot := healthE2ESetup(t)
	sharedPath := filepath.Join(repoRoot, "data", "titles", "halo_infinite",
		"warehouse", "shared_matches_v2.duckdb")
	seedOrphanParticipant(t, sharedPath)

	sched := scheduler.NewDataHealthScheduler(repoRoot)
	res := sched.RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	// Le xuid sans alias doit être compté (signal derrière le masquage "Joueur ####").
	if res.OrphanXUIDs < 1 {
		t.Errorf("OrphanXUIDs: attendu >= 1 (xuid sans alias), obtenu %d", res.OrphanXUIDs)
	}
	// orphan_xuids reste informatif : ne doit PAS gonfler WarningsTotal.
	if res.WarningsTotal != 0 {
		t.Errorf("WarningsTotal: orphan ne doit pas compter comme warning, obtenu %d",
			res.WarningsTotal)
	}
}
