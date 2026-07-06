//go:build integration

// post_sync_progression_test.go — tests d'intégration end-to-end de
// l'orchestrateur progression V2.

package wire

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
)

const (
	testXUID  = "xuid-progression-test"
	testGT    = "ProgressionTester"
	testTitle = "halo_infinite"
)

// progressionTestEnv regroupe les DBs préparées + la PlayerDB associée.
type progressionTestEnv struct {
	pdb     *duckdb.PlayerDB
	cleanup func()
}

// setupProgressionEnv crée 4 DBs DuckDB temporaires (Player, Shared,
// SharedSocial, Metadata) et applique toutes les migrations. Retourne un
// PlayerDB câblé. Idempotent et standalone.
func setupProgressionEnv(t *testing.T) *progressionTestEnv {
	t.Helper()
	dir := t.TempDir()

	openAndMigrate := func(name string, target migration.TargetDB) *duckdb.DB {
		path := filepath.Join(dir, name+".duckdb")
		raw, err := sql.Open("duckdb", path)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		if err := migration.RunForDB(raw, target); err != nil {
			raw.Close()
			t.Fatalf("migrate %s: %v", name, err)
		}
		raw.Close()
		db, err := duckdb.OpenReadWrite(path)
		if err != nil {
			t.Fatalf("reopen %s: %v", name, err)
		}
		return db
	}

	// Topologie post-ADR 0016 : shared ouvert en conn RW dédiée, pas
	// d'ATTACH sur player. Les reads cross-DB passent par SharedReader
	// (LegacySharedReader(pdb.Shared) suffit en test).
	shared := openAndMigrate("shared_matches_v2", migration.TargetShared)
	social := openAndMigrate("shared_social", migration.TargetSharedSocial)
	meta := openAndMigrate("metadata", migration.TargetMetadata)
	player := openAndMigrate("stats", migration.TargetPlayer)

	pdb := &duckdb.PlayerDB{
		Player: player, Shared: shared, SharedSocial: social, Metadata: meta,
		SharedReader: duckdb.LegacySharedReader(shared),
		XUID:         testXUID, Gamertag: testGT, TitleSlug: testTitle,
	}

	cleanup := func() {
		player.Close()
		shared.Close()
		social.Close()
		meta.Close()
		_ = os.RemoveAll(dir)
	}
	t.Cleanup(cleanup)
	return &progressionTestEnv{pdb: pdb, cleanup: cleanup}
}

// seedMatches insère N matchs dans shared_matches_v2 + stats.duckdb pour le
// joueur testXUID, datés en remontant d'un jour à partir de `now`.
func seedMatches(t *testing.T, env *progressionTestEnv, now time.Time, count int) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < count; i++ {
		matchID := "m_" + zeropad(i, 3)
		startTime := now.AddDate(0, 0, -i)
		outcome := 2 // win
		if i%3 == 0 {
			outcome = 3 // loss
		}
		// match_registry sur shared (conn dédiée post-ADR 0016).
		if _, err := env.pdb.Shared.Exec(ctx, `
			INSERT INTO match_registry (match_id, start_time)
			VALUES (?, ?)
		`, matchID, startTime); err != nil {
			t.Fatalf("insert match_registry %s: %v", matchID, err)
		}
		// match_participants sur shared.
		if _, err := env.pdb.Shared.Exec(ctx, `
			INSERT INTO match_participants (
				match_id, xuid, gamertag, team_id, outcome, kills, deaths, assists,
				kda, accuracy, personal_score, time_played_seconds, headshot_kills
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, matchID, testXUID, testGT, 1, outcome, 12+i%4, 8, 3, 1.5, 0.55, 1500, 600, 5+i%3); err != nil {
			t.Fatalf("insert participant %s: %v", matchID, err)
		}
		// player_match_enrichment (stats.duckdb, performance_score) — append-only #23046 :
		// INSERT pur stage='perf' (plus d'ON CONFLICT : match_id n'est plus une PK).
		if _, err := env.pdb.Player.Exec(ctx, `
			INSERT INTO player_match_enrichment (match_id, performance_score, stage)
			VALUES (?, ?, 'perf')
		`, matchID, 70.0+float64(i%10)); err != nil {
			t.Fatalf("insert pme %s: %v", matchID, err)
		}
		// match_skill_rank (stats.duckdb, LUSR rating)
		mu := 1500.0 + float64(i%20)
		if _, err := env.pdb.Player.Exec(ctx, `
			INSERT INTO match_skill_rank (match_id, rating_type, rating_value, start_time)
			VALUES (?, 'LUSR', ?, ?)
		`, matchID, mu, startTime); err != nil {
			t.Fatalf("insert msr %s: %v", matchID, err)
		}
	}
}

// seedMilestoneCatalog charge un catalogue minimal dans metadata.duckdb.
func seedMilestoneCatalog(t *testing.T, env *progressionTestEnv) {
	t.Helper()
	ctx := context.Background()
	if _, err := env.pdb.Metadata.Exec(ctx, `
		INSERT INTO milestone_catalog (id, title_slug, metric, threshold, title_en, title_fr)
		VALUES
			('h.matches.10',  'halo_infinite', 'matches_played',  10, 'Decimus',   'Decimus'),
			('h.matches.100', 'halo_infinite', 'matches_played', 100, 'Centurion', 'Centurion'),
			('h.wins.5',      'halo_infinite', 'wins',             5, 'Winner',    'Vainqueur')
	`); err != nil {
		t.Fatalf("seed milestone_catalog: %v", err)
	}
}

// zeropad retourne n formaté sur w chiffres.
func zeropad(n, w int) string {
	out := make([]byte, w)
	for i := w - 1; i >= 0; i-- {
		out[i] = byte('0' + n%10)
		n /= 10
	}
	return string(out)
}

// ─── Tests ─────────────────────────────────────────────────────────────────

func TestEvaluateProgression_EmptyDB_NoCrash(t *testing.T) {
	env := setupProgressionEnv(t)
	emitter := &recordingEmitter{}
	deps := BuildPlayerProgressionDeps(env.pdb, emitter)

	res, err := EvaluateProgressionAfterSync(
		context.Background(), env.pdb, testTitle, deps,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("EvaluateProgressionAfterSync: %v", err)
	}
	if res.AlertsEmitted > 0 {
		t.Errorf("empty DB should produce 0 alerts, got %d", res.AlertsEmitted)
	}
	if len(emitter.emitted) != 0 {
		t.Errorf("emitter should be empty, got %d", len(emitter.emitted))
	}
}

func TestEvaluateProgression_WithMatches_DetectorsRunAndAlertsFire(t *testing.T) {
	env := setupProgressionEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatches(t, env, now, 15) // 15 matchs sur 15 jours
	seedMilestoneCatalog(t, env)

	emitter := &recordingEmitter{}
	deps := BuildPlayerProgressionDeps(env.pdb, emitter)
	res, err := EvaluateProgressionAfterSync(context.Background(), env.pdb, testTitle, deps, now)
	if err != nil {
		t.Fatalf("EvaluateProgressionAfterSync: %v", err)
	}

	// Au moins le milestone h.matches.10 doit avoir été débloqué (15 >= 10).
	if len(res.MilestoneResults) == 0 {
		t.Errorf("expected non-empty milestone results")
	}
	earnedCount := 0
	for _, r := range res.MilestoneResults {
		if r.Earned {
			earnedCount++
		}
	}
	if earnedCount == 0 {
		t.Errorf("expected at least 1 milestone earned (matches.10), got 0")
	}

	// Records : 15 matchs > MinMatchesForRecord(10), donc on attend des PB sur
	// les 3 fenêtres pour les métriques disponibles (au moins
	// performance_score et kda).
	if len(res.RecordResults) == 0 {
		t.Errorf("expected non-empty record results")
	}
	newPBCount := 0
	for _, r := range res.RecordResults {
		if r.NewPB {
			newPBCount++
		}
	}
	if newPBCount == 0 {
		t.Errorf("expected at least 1 NewPB result (15 >= 10 matches), got 0")
	}

	// Streaks : 15 jours consécutifs → daily_play streak créée + incrémentée
	// à 15 jours. Au moins 1 résultat avec Transition != None.
	if len(res.StreakResults) == 0 {
		t.Errorf("expected non-empty streak results")
	}

	// Au moins 1 alerte générée et émise (milestone unlock h.matches.10 = sûr).
	if res.AlertsGenerated == 0 {
		t.Errorf("expected at least 1 alert generated, got 0")
	}
	if res.AlertsEmitted == 0 {
		t.Errorf("expected at least 1 alert emitted, got 0 (deduped=%d)", res.AlertsDeduped)
	}
	if len(emitter.emitted) == 0 {
		t.Errorf("emitter should have received at least 1 notif")
	}

	// Vérifier qu'on trouve au moins une catégorie progression.
	foundProgression := false
	for _, e := range emitter.emitted {
		switch e.Category {
		case notifications.CategoryMilestoneUnlocked,
			notifications.CategoryPersonalRecord,
			notifications.CategoryStreakMilestone,
			notifications.CategoryRecordNearMiss,
			notifications.CategoryMilestoneNearMiss,
			notifications.CategoryLUSRTierApproach,
			notifications.CategoryComebackWelcome,
			notifications.CategoryThresholdCrossed:
			foundProgression = true
		}
	}
	if !foundProgression {
		t.Errorf("expected at least 1 progression-category notif, got categories %v", categoriesOf(emitter.emitted))
	}
}

// TestEvaluateProgression_DetectorIdempotency : sur 2 passes consécutives, les
// détecteurs eux-mêmes deviennent silencieux (milestones déjà débloqués →
// AlreadyHad, PB déjà persistés → pas de NewPB). Les alerts d'événements
// uniques (record_broken, milestone_unlocked) ne sont donc plus générées.
// Les alerts récurrentes (LUSR tier approach, comeback) peuvent persister
// si leurs conditions sont toujours remplies — la dédup réelle dans la
// fenêtre 24h passe par NotificationsRepo en prod, mais le recordingEmitter
// ne persiste pas dans la DB donc on ne teste pas la dédup ici.
//
// On vérifie ici uniquement la propriété d'idempotence des DÉTECTEURS de
// persistance (records + milestones).
func TestEvaluateProgression_DetectorIdempotency(t *testing.T) {
	env := setupProgressionEnv(t)
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	seedMatches(t, env, now, 15)
	seedMilestoneCatalog(t, env)

	emitter1 := &recordingEmitter{}
	deps := BuildPlayerProgressionDeps(env.pdb, emitter1)
	res1, _ := EvaluateProgressionAfterSync(context.Background(), env.pdb, testTitle, deps, now)

	// 2e passe sur les mêmes données.
	emitter2 := &recordingEmitter{}
	deps2 := BuildPlayerProgressionDeps(env.pdb, emitter2)
	res2, err := EvaluateProgressionAfterSync(context.Background(), env.pdb, testTitle, deps2, now)
	if err != nil {
		t.Fatalf("2nd pass: %v", err)
	}

	// Compter les milestones nouvellement Earned au 1er vs 2e passe.
	countEarned := func(rs []milestones.DetectionResult) int {
		n := 0
		for _, r := range rs {
			if r.Earned {
				n++
			}
		}
		return n
	}
	first := countEarned(res1.MilestoneResults)
	second := countEarned(res2.MilestoneResults)
	if first == 0 {
		t.Fatalf("1st pass should earn at least 1 milestone, got 0")
	}
	if second != 0 {
		t.Errorf("2nd pass should re-earn 0 milestones (idempotence), got %d", second)
	}

	// Compter les NewPB au 1er vs 2e passe.
	countPB := func(rs []records.DetectionResult) int {
		n := 0
		for _, r := range rs {
			if r.NewPB {
				n++
			}
		}
		return n
	}
	firstPB := countPB(res1.RecordResults)
	secondPB := countPB(res2.RecordResults)
	if firstPB == 0 {
		t.Fatalf("1st pass should detect at least 1 NewPB, got 0")
	}
	if secondPB != 0 {
		t.Errorf("2nd pass should detect 0 NewPB (idempotence), got %d", secondPB)
	}
}

func categoriesOf(emits []notifications.EmitInput) []notifications.Category {
	out := make([]notifications.Category, 0, len(emits))
	for _, e := range emits {
		out = append(out, e.Category)
	}
	return out
}
