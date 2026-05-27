//go:build integration

// Package sync — enrichments_test.go : tests pour computeAndPersistHadBotTeammate.
//
// Reference D.4 du plan de tests. Valide :
//   - bot teammate detection (xuid LIKE 'bid(%')
//   - filtre team_id (bot dans autre team != teammate)
//   - filtre xuid != self
//   - idempotence (re-run sans changer DB)
//   - aucun match → return 0
//   - schema absent → return error
package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

// setupEnrichmentsTestDBs cree shared + player en memoire avec schemas appliques.
func setupEnrichmentsTestDBs(t *testing.T) (sharedDB, playerDB *sql.DB) {
	t.Helper()

	shared, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open shared: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	if err := migration.RunForDB(shared, migration.TargetShared); err != nil {
		t.Fatalf("migrate shared: %v", err)
	}

	player, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("open player: %v", err)
	}
	t.Cleanup(func() { _ = player.Close() })
	if err := EnsurePlayerSchema(context.Background(), player); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}
	if err := migration.RunForDB(player, migration.TargetPlayer); err != nil {
		t.Fatalf("migrate player: %v", err)
	}

	return shared, player
}

// insertParticipant insere une ligne match_participants avec match_id, xuid, team_id donnes.
// time_played_seconds par defaut a 600s (10 min) pour passer le seuil hybride
// (cf. botPresenceMinSeconds/Ratio) dans les tests qui ne s'en preoccupent pas.
// Utiliser insertParticipantWithTime pour controler explicitement le temps de jeu.
func insertParticipant(t *testing.T, db *sql.DB, matchID, xuid string, teamID int) {
	t.Helper()
	insertParticipantWithTime(t, db, matchID, xuid, teamID, 600)
}

// insertParticipantWithTime variante qui force time_played_seconds. Utile pour
// les tests de seuil hybride (bot bref vs significatif).
func insertParticipantWithTime(t *testing.T, db *sql.DB, matchID, xuid string, teamID, timePlayedSec int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, team_id, time_played_seconds)
		VALUES (?, ?, ?, ?)
	`, matchID, xuid, teamID, timePlayedSec)
	if err != nil {
		t.Fatalf("insert participant %s/%s: %v", matchID, xuid, err)
	}
}

// insertBotWithParticipation insere un bot avec les 4 BOOLEAN ParticipationInfo
// (mini-Phase 0.5 schema). Utile pour les tests du raffinement late-join.
func insertBotWithParticipation(t *testing.T, db *sql.DB, matchID, botXUID string,
	teamID, timePlayedSec int,
	presentAtBeginning, joinedInProgress, leftInProgress, presentAtCompletion bool,
) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_participants (
			match_id, xuid, team_id, time_played_seconds,
			present_at_beginning, joined_in_progress, left_in_progress, present_at_completion
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, matchID, botXUID, teamID, timePlayedSec,
		presentAtBeginning, joinedInProgress, leftInProgress, presentAtCompletion)
	if err != nil {
		t.Fatalf("insert bot participation %s/%s: %v", matchID, botXUID, err)
	}
}

// insertMatchRegistry insere une ligne shared.match_registry avec duration_seconds.
// Le JOIN match_registry est obligatoire dans la query "significant bot" depuis
// le seuil hybride 2026-05-27 — sans cette row le match n'est pas considere
// significatif (match_duration = 0 dans la query, donc filtre echoue).
func insertMatchRegistry(t *testing.T, db *sql.DB, matchID string, durationSec int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_registry (match_id, duration_seconds)
		VALUES (?, ?)
	`, matchID, durationSec)
	if err != nil {
		t.Fatalf("insert match_registry %s: %v", matchID, err)
	}
}

// insertPMEHadBot insere une ligne PME minimale (had_bot_teammate FALSE).
func insertPMEHadBot(t *testing.T, db *sql.DB, matchID string) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO player_match_enrichment (match_id, had_bot_teammate)
		VALUES (?, FALSE)
	`, matchID)
	if err != nil {
		t.Fatalf("insert PME %s: %v", matchID, err)
	}
}

// getHadBot retourne le flag had_bot_teammate pour un match_id.
func getHadBot(t *testing.T, db *sql.DB, matchID string) bool {
	t.Helper()
	var v sql.NullBool
	if err := db.QueryRow("SELECT had_bot_teammate FROM player_match_enrichment WHERE match_id = ?", matchID).Scan(&v); err != nil {
		t.Fatalf("query had_bot %s: %v", matchID, err)
	}
	return v.Valid && v.Bool
}

func TestComputeAndPersistHadBotTeammate_BotInSameTeam_TRUE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Match avec self team 0 + bot team 0 + un autre humain team 1.
	// Bot 600s sur match 600s (100%) -> bien au-dessus du seuil hybride 30s/15%.
	insertMatchRegistry(t, shared, "m1", 600)
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "bid(3.0)", 0)
	insertParticipant(t, shared, "m1", "9876543210", 1)
	insertPMEHadBot(t, player, "m1")

	updated, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1", updated)
	}
	if !getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre TRUE")
	}
}

func TestComputeAndPersistHadBotTeammate_BotInOppositeTeam_FALSE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Match avec self team 0 + bot dans team 1 (adverse) → had_bot = FALSE.
	insertMatchRegistry(t, shared, "m1", 600)
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "bid(7.0)", 1)
	insertPMEHadBot(t, player, "m1")

	updated, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0 (bot dans team adverse)", updated)
	}
	if getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit rester FALSE")
	}
}

func TestComputeAndPersistHadBotTeammate_NoBots_FALSE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	insertMatchRegistry(t, shared, "m1", 600)
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "9876543210", 0)
	insertParticipant(t, shared, "m1", "5555555555", 1)
	insertPMEHadBot(t, player, "m1")

	updated, _ := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if updated != 0 {
		t.Errorf("updated = %d, want 0 (aucun bot)", updated)
	}
	if getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit rester FALSE")
	}
}

func TestComputeAndPersistHadBotTeammate_MultipleMatches_OnlyAffectedUpdated(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// m1 : avec bot teammate (bot 600s sur match 600s = 100% → significatif)
	insertMatchRegistry(t, shared, "m1", 600)
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "bid(3.0)", 0)
	insertPMEHadBot(t, player, "m1")

	// m2 : sans bot
	insertMatchRegistry(t, shared, "m2", 600)
	insertParticipant(t, shared, "m2", selfXUID, 0)
	insertParticipant(t, shared, "m2", "9999999999", 0)
	insertPMEHadBot(t, player, "m2")

	// m3 : bot dans team adverse
	insertMatchRegistry(t, shared, "m3", 600)
	insertParticipant(t, shared, "m3", selfXUID, 0)
	insertParticipant(t, shared, "m3", "bid(5.0)", 1)
	insertPMEHadBot(t, player, "m3")

	updated, _ := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if updated != 1 {
		t.Errorf("updated = %d, want 1 (seul m1)", updated)
	}

	if !getHadBot(t, player, "m1") {
		t.Error("m1 had_bot doit etre TRUE")
	}
	if getHadBot(t, player, "m2") {
		t.Error("m2 had_bot doit etre FALSE")
	}
	if getHadBot(t, player, "m3") {
		t.Error("m3 had_bot doit etre FALSE (bot adverse)")
	}
}

func TestComputeAndPersistHadBotTeammate_Idempotent_SecondRunZero(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	insertMatchRegistry(t, shared, "m1", 600)
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "bid(3.0)", 0)
	insertPMEHadBot(t, player, "m1")

	// 1er run
	updated1, _ := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if updated1 != 1 {
		t.Errorf("1er run : updated = %d, want 1", updated1)
	}

	// 2eme run : 0 (deja TRUE)
	updated2, _ := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if updated2 != 0 {
		t.Errorf("2eme run : updated = %d, want 0 (idempotent)", updated2)
	}
}

func TestComputeAndPersistHadBotTeammate_NoMatchesForXUID(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)

	updated, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, "non_existing_xuid")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if updated != 0 {
		t.Errorf("updated = %d, want 0 (xuid sans matchs)", updated)
	}
}

func TestComputeAndPersistHadBotTeammate_OnlySelfInTeam_NoBotTeammate(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Match avec uniquement self dans team 0, autres team 1.
	insertMatchRegistry(t, shared, "m1", 600)
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "9876543210", 1)
	insertParticipant(t, shared, "m1", "5555555555", 1)
	insertPMEHadBot(t, player, "m1")

	updated, _ := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if updated != 0 {
		t.Errorf("updated = %d, want 0 (self seul dans son equipe)", updated)
	}
}

// ---------------------------------------------------------------------------
// Tests du seuil hybride (botPresenceMinSeconds=30 ET botPresenceMinRatio=15%)
// Ajoutés 2026-05-27 — cf. thought_log.
// ---------------------------------------------------------------------------

func TestComputeAndPersistHadBotTeammate_BotBrief_FALSE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Bot 10s sur match 480s (2%) → FALSE (sous les deux seuils).
	insertMatchRegistry(t, shared, "m1", 480)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 480)
	insertParticipantWithTime(t, shared, "m1", "bid(3.0)", 0, 10)
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre FALSE (bot 10s = anecdotique)")
	}
}

func TestComputeAndPersistHadBotTeammate_BotBelowAbsolute_FALSE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Bot 25s sur match 100s : ratio = 25% (>= 15%) MAIS time < 30s → FALSE
	// (plancher absolu sur matchs courts).
	insertMatchRegistry(t, shared, "m1", 100)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 100)
	insertParticipantWithTime(t, shared, "m1", "bid(3.0)", 0, 25)
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre FALSE (bot 25s sous le plancher 30s)")
	}
}

func TestComputeAndPersistHadBotTeammate_BotBelowRatio_FALSE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Bot 50s sur match 500s : time >= 30s MAIS ratio = 10% (< 15%) → FALSE
	// (ratio sous le seuil sur matchs longs).
	insertMatchRegistry(t, shared, "m1", 500)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 500)
	insertParticipantWithTime(t, shared, "m1", "bid(3.0)", 0, 50)
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre FALSE (bot 50s/500s = 10% sous le ratio 15%)")
	}
}

func TestComputeAndPersistHadBotTeammate_BotMeetsBothThresholds_TRUE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Bot 100s sur match 480s : time >= 30s ET ratio = 20.8% (>= 15%) → TRUE.
	insertMatchRegistry(t, shared, "m1", 480)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 480)
	insertParticipantWithTime(t, shared, "m1", "bid(3.0)", 0, 100)
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre TRUE (bot 100s/480s = 20.8% passe les deux seuils)")
	}
}

func TestComputeAndPersistHadBotTeammate_MultipleBotsSummed_TRUE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Deux bots dans ma team : 30s + 30s = 60s cumules sur match 200s (30%).
	// Chacun pris individuellement sous le ratio 15% — c'est la SOMME qui compte.
	insertMatchRegistry(t, shared, "m1", 200)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 200)
	insertParticipantWithTime(t, shared, "m1", "bid(3.0)", 0, 30)
	insertParticipantWithTime(t, shared, "m1", "bid(5.0)", 0, 30)
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre TRUE (2 bots cumulant 60s/200s = 30%)")
	}
}

// ---------------------------------------------------------------------------
// Tests du raffinement late-join (mini-Phase 0.5 ParticipationInfo).
// Ajoutés 2026-05-27 — cf. thought_log.
// Bot `joined_in_progress=TRUE` avec time_played < botLateJoinIgnoreRatio (30%)
// du match est ignoré dans la SUM. Reflète sémantiquement un remplaçant tardif.
// ---------------------------------------------------------------------------

func TestComputeAndPersistHadBotTeammate_LateJoinBrief_FALSE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Bot late-join, 80s sur 480s (16.6% < 30% ratio late-join) → ignoré
	// dans la SUM même si ratio normal 16.6% > 15% aurait passé sans
	// raffinement. Sémantique : bot remplaçant tardif, pas de pollution.
	insertMatchRegistry(t, shared, "m1", 480)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 480)
	insertBotWithParticipation(t, shared, "m1", "bid(3.0)", 0, 80,
		false, true, false, true) // late-join, est resté jusqu'à la fin
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre FALSE (late-join 16.6% sous le seuil late-join 30%)")
	}
}

func TestComputeAndPersistHadBotTeammate_LateJoinLong_TRUE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Bot late-join, 200s sur 480s (41.6% >= 30%) → compté dans la SUM.
	// Sémantique : humain quitté à mi-match, bot prend la suite → vrai
	// déséquilibre sur une portion significative du match.
	insertMatchRegistry(t, shared, "m1", 480)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 480)
	insertBotWithParticipation(t, shared, "m1", "bid(3.0)", 0, 200,
		false, true, false, true)
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre TRUE (late-join 41.6% >= seuil late-join 30%)")
	}
}

func TestComputeAndPersistHadBotTeammate_FromStartBrief_PassesNormalThresholds(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Bot present_at_beginning=TRUE (pas late-join), time_played=100s sur 480s
	// (20.8%). Le raffinement late-join NE s'applique PAS → SUM=100s, ratio
	// 20.8% >= 15% normal ET 100s >= 30s normal → TRUE.
	insertMatchRegistry(t, shared, "m1", 480)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 480)
	insertBotWithParticipation(t, shared, "m1", "bid(3.0)", 0, 100,
		true, false, false, false) // from start, parti en cours
	insertPMEHadBot(t, player, "m1")

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit etre TRUE (from-start, passe seuils hybrides normaux)")
	}
}

func TestComputeAndPersistHadBotTeammate_MadinaRealCase_FALSE(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274858283686"

	// Reproduit exactement le cas Madina/615b3ebc : bot late-join, 6s sur
	// 390s (1.5%). Ce match a motivé l'amélioration ; il doit être FALSE
	// après recompute.
	insertMatchRegistry(t, shared, "615b3ebc", 390)
	insertParticipantWithTime(t, shared, "615b3ebc", selfXUID, 0, 390)
	insertBotWithParticipation(t, shared, "615b3ebc", "bid(18.0)", 0, 6,
		false, true, false, true)
	// Seed PME avec flag TRUE (ancien algo binaire).
	if _, err := player.Exec(`INSERT INTO player_match_enrichment (match_id, had_bot_teammate) VALUES (?, TRUE)`, "615b3ebc"); err != nil {
		t.Fatalf("seed PME: %v", err)
	}

	if _, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID); err != nil {
		t.Fatalf("err: %v", err)
	}
	if getHadBot(t, player, "615b3ebc") {
		t.Error("Cas Madina : had_bot_teammate doit etre FALSE (bot late-join 6s/390s = 1.5%)")
	}
}

func TestComputeAndPersistHadBotTeammate_Recompute_FlipsTrueToFalse(t *testing.T) {
	shared, player := setupEnrichmentsTestDBs(t)
	const selfXUID = "2533274823110022"

	// Simule l'etat post ancien algo : bot 10s, flag deja TRUE en DB.
	insertMatchRegistry(t, shared, "m1", 480)
	insertParticipantWithTime(t, shared, "m1", selfXUID, 0, 480)
	insertParticipantWithTime(t, shared, "m1", "bid(3.0)", 0, 10)
	// PME deja flag TRUE (issu de l'ancien calcul binaire).
	if _, err := player.Exec(`INSERT INTO player_match_enrichment (match_id, had_bot_teammate) VALUES (?, TRUE)`, "m1"); err != nil {
		t.Fatalf("seed pre-existing TRUE: %v", err)
	}

	updated, err := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if updated != 1 {
		t.Errorf("updated = %d, want 1 (flip TRUE -> FALSE)", updated)
	}
	if getHadBot(t, player, "m1") {
		t.Error("had_bot_teammate doit avoir flip a FALSE apres recalcul")
	}
}
