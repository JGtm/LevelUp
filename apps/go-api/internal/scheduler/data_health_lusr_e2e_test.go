package scheduler_test

// data_health_lusr_e2e_test.go — E2E du volet « trous LUSR » du HealthScheduler :
// arborescence titre réelle (migrations shared + player), match éligible sous le
// watermark sans note LUSR → trou d'intérieur détecté, et déclenchement (ou non)
// de l'auto-heal selon le kill-switch. Réutilise healthE2ESetup (data_health_check_e2e_test.go).

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/skillchain"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/scheduler"
	syncpkg "levelup/go-api/internal/sync"
	"levelup/go-api/internal/sync/skill"
)

// seedLUSRInteriorGaps crée un joueur (dir + xuid.txt + player DB migrée SANS note
// LUSR) et seed nMatches matchs 2v2 éligibles (pair « Slayer » → arena_slayer) tous
// SOUS un watermark arena_slayer → nMatches trous d'intérieur pour ce joueur.
func seedLUSRInteriorGaps(t *testing.T, repoRoot, gamertag, xuid string, nMatches int) {
	t.Helper()
	ctx := context.Background()
	// Classifier LUSR câblé au boot en prod (runMigrations, main.go:404 — AVANT le
	// démarrage du scheduler l.998, gated par ValidateLUSRChainClassifierWired). Le
	// package de test ne boote pas le serveur → on pose le seam ici (idempotent).
	syncpkg.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)
	titleDir := filepath.Join(repoRoot, "data", "titles", "halo_infinite")

	// Player DB migrée (match_skill_rank + vue _latest) + xuid.txt. Aucune ligne
	// LUSR insérée → tous les matchs seront « non notés ».
	playerDir := filepath.Join(titleDir, "players", gamertag)
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatalf("mkdir player dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "xuid.txt"), []byte(xuid), 0o644); err != nil {
		t.Fatalf("write xuid.txt: %v", err)
	}
	pdb, err := duckdb.OpenReadWrite(filepath.Join(playerDir, "stats.duckdb"))
	if err != nil {
		t.Fatalf("open player DB: %v", err)
	}
	if err := migration.RunForDB(pdb.SQLDb(), migration.TargetPlayer); err != nil {
		pdb.Close()
		t.Fatalf("migrate player DB: %v", err)
	}
	pdb.Close()

	// Shared : nMatches matchs 2v2 + un watermark AU-DESSUS de tous.
	sharedPath := filepath.Join(titleDir, "warehouse", "shared_matches_v2.duckdb")
	shared, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		t.Fatalf("open shared rw: %v", err)
	}
	defer shared.Close()
	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < nMatches; i++ {
		matchID := fmt.Sprintf("m_gap_%s_%d", gamertag, i)
		ts := base.Add(time.Duration(i) * time.Minute)
		if _, err := shared.Exec(ctx, `INSERT INTO match_registry
			(match_id, start_time, start_time_utc, pair_name, is_ranked, is_firefight, duration_seconds)
			VALUES (?, ?, ?, 'Slayer', FALSE, FALSE, 600)`, matchID, ts, ts); err != nil {
			t.Fatalf("insert match_registry: %v", err)
		}
		for _, p := range []struct {
			xuid          string
			team, outcome int
		}{{xuid, 0, 2}, {gamertag + "_mate", 0, 2}, {gamertag + "_opp1", 1, 3}, {gamertag + "_opp2", 1, 3}} {
			if _, err := shared.Exec(ctx, `INSERT INTO match_participants
				(match_id, xuid, team_id, outcome, kills, deaths)
				VALUES (?, ?, ?, ?, 10, 8)`, matchID, p.xuid, p.team, p.outcome); err != nil {
				t.Fatalf("insert match_participants: %v", err)
			}
		}
	}
	// Watermark arena_slayer 1h après le dernier match → tous « déjà vus » sans note.
	watermark := base.Add(time.Hour)
	if _, err := shared.Exec(ctx, `INSERT INTO player_skill_state_v2
		(xuid, playlist_group, mu, sigma, experience, last_match_id, last_match_at)
		VALUES (?, 'arena_slayer', 25, 5, 1, 'seeded', ?)`, xuid, watermark); err != nil {
		t.Fatalf("insert watermark: %v", err)
	}
}

// seedUnmeasurableLUSRPlayer crée un joueur (dir + xuid.txt) dont la player DB est un
// fichier CORROMPU (pas une DuckDB valide) : openDBShared échoue au ping → le joueur
// est compté LUSRPlayersUnmeasured (scan partiel) sans être scanné, et la jauge ne
// doit pas être republiée.
func seedUnmeasurableLUSRPlayer(t *testing.T, repoRoot, gamertag, xuid string) {
	t.Helper()
	playerDir := filepath.Join(repoRoot, "data", "titles", "halo_infinite", "players", gamertag)
	if err := os.MkdirAll(playerDir, 0o755); err != nil {
		t.Fatalf("mkdir player dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(playerDir, "xuid.txt"), []byte(xuid), 0o644); err != nil {
		t.Fatalf("write xuid.txt: %v", err)
	}
	// Contenu volontairement non-DuckDB → PingContext échoue à l'ouverture read-only.
	if err := os.WriteFile(filepath.Join(playerDir, "stats.duckdb"),
		[]byte("this is not a valid duckdb database file"), 0o644); err != nil {
		t.Fatalf("write corrupt stats.duckdb: %v", err)
	}
}

func TestHealthScheduler_E2E_LUSRInteriorGap_Detected(t *testing.T) {
	repoRoot := healthE2ESetup(t)
	seedLUSRInteriorGaps(t, repoRoot, "GapPlayer", "2533274811111111", 1)

	res := scheduler.NewDataHealthScheduler(repoRoot).RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	if res.LUSRInteriorGaps < 1 {
		t.Errorf("LUSRInteriorGaps: attendu >= 1 (trou seedé), obtenu %d", res.LUSRInteriorGaps)
	}
	if res.LUSRPlayersScanned < 1 {
		t.Errorf("LUSRPlayersScanned: attendu >= 1, obtenu %d", res.LUSRPlayersScanned)
	}
	// Les trous LUSR ne comptent PAS dans WarningsTotal (signal distinct).
	if res.WarningsTotal != 0 {
		t.Errorf("WarningsTotal: les trous LUSR ne doivent pas gonfler les warnings, obtenu %d", res.WarningsTotal)
	}
	// Cas nominal : joueur mesurable → aucun non mesuré et la jauge est publiée.
	if res.LUSRPlayersUnmeasured != 0 {
		t.Errorf("LUSRPlayersUnmeasured: attendu 0 (joueur mesurable), obtenu %d", res.LUSRPlayersUnmeasured)
	}
	if got := skill.LUSRInteriorGapsGaugeValue(); got != int64(res.LUSRInteriorGaps) {
		t.Errorf("jauge LUSR: got %d, want %d (scan complet → publiée)", got, res.LUSRInteriorGaps)
	}
}

// TestHealthScheduler_E2E_LUSRUnmeasured_GaugeFrozen : un joueur dont la player DB est
// inouvrable rend le scan PARTIEL (LUSRPlayersUnmeasured > 0) → la jauge NE doit PAS
// être republiée (valeur précédente conservée, « unmeasured ≠ sain »).
func TestHealthScheduler_E2E_LUSRUnmeasured_GaugeFrozen(t *testing.T) {
	repoRoot := healthE2ESetup(t)
	seedUnmeasurableLUSRPlayer(t, repoRoot, "Corrupt", "2533274844444444")

	// Valeur sentinelle : simule un dernier scan complet (ou un heal manuel) publié.
	const sentinel = 42
	skill.SetLUSRInteriorGapsGauge(sentinel)
	t.Cleanup(func() { skill.SetLUSRInteriorGapsGauge(0) })

	res := scheduler.NewDataHealthScheduler(repoRoot).RunOnce(context.Background())
	if res == nil {
		t.Fatal("RunOnce a retourné nil")
	}
	if res.LUSRPlayersUnmeasured < 1 {
		t.Errorf("LUSRPlayersUnmeasured: attendu >= 1 (player DB non ouvrable), obtenu %d", res.LUSRPlayersUnmeasured)
	}
	// Scan partiel → la jauge n'est pas écrasée : la valeur précédente est conservée.
	if got := skill.LUSRInteriorGapsGaugeValue(); got != sentinel {
		t.Errorf("jauge écrasée sur scan partiel: got %d, want %d (valeur précédente conservée)", got, sentinel)
	}
}

func TestHealthScheduler_E2E_AutoHeal_FiresWhenEnabled(t *testing.T) {
	t.Setenv("LEVELUP_LUSR_AUTOHEAL_ENABLED", "1")
	repoRoot := healthE2ESetup(t)
	seedLUSRInteriorGaps(t, repoRoot, "HealMe", "2533274822222222", 3) // >= seuil (3)

	var healedTitle, healedGT string
	calls := 0
	sched := scheduler.NewDataHealthScheduler(repoRoot).WithLUSRAutoHeal(
		func(_ context.Context, titleSlug, gamertag string) error {
			healedTitle, healedGT = titleSlug, gamertag
			calls++
			return nil
		})
	res := sched.RunOnce(context.Background())
	if res.LUSRInteriorGaps < 3 {
		t.Fatalf("LUSRInteriorGaps: attendu >= 3, obtenu %d", res.LUSRInteriorGaps)
	}
	if calls != 1 {
		t.Errorf("auto-heal ON : %d appel(s), want 1 (1 joueur/cycle)", calls)
	}
	if healedGT != "HealMe" || healedTitle != "halo_infinite" {
		t.Errorf("auto-heal ciblé = (%q, %q), want (halo_infinite, HealMe)", healedTitle, healedGT)
	}
}

func TestHealthScheduler_E2E_AutoHeal_OffByDefault(t *testing.T) {
	t.Setenv("LEVELUP_LUSR_AUTOHEAL_ENABLED", "0") // défaut = OFF (explicite ici)
	repoRoot := healthE2ESetup(t)
	seedLUSRInteriorGaps(t, repoRoot, "NoHeal", "2533274833333333", 3)

	calls := 0
	sched := scheduler.NewDataHealthScheduler(repoRoot).WithLUSRAutoHeal(
		func(context.Context, string, string) error { calls++; return nil })
	res := sched.RunOnce(context.Background())
	if res.LUSRInteriorGaps < 3 {
		t.Fatalf("LUSRInteriorGaps: attendu >= 3, obtenu %d", res.LUSRInteriorGaps)
	}
	if calls != 0 {
		t.Errorf("auto-heal OFF par défaut : %d appel(s), want 0 (alerte seule)", calls)
	}
}
