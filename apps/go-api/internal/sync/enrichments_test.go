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
func insertParticipant(t *testing.T, db *sql.DB, matchID, xuid string, teamID int) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO match_participants (match_id, xuid, team_id)
		VALUES (?, ?, ?)
	`, matchID, xuid, teamID)
	if err != nil {
		t.Fatalf("insert participant %s/%s: %v", matchID, xuid, err)
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

	// m1 : avec bot teammate
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "bid(3.0)", 0)
	insertPMEHadBot(t, player, "m1")

	// m2 : sans bot
	insertParticipant(t, shared, "m2", selfXUID, 0)
	insertParticipant(t, shared, "m2", "9999999999", 0)
	insertPMEHadBot(t, player, "m2")

	// m3 : bot dans team adverse
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
	insertParticipant(t, shared, "m1", selfXUID, 0)
	insertParticipant(t, shared, "m1", "9876543210", 1)
	insertParticipant(t, shared, "m1", "5555555555", 1)
	insertPMEHadBot(t, player, "m1")

	updated, _ := computeAndPersistHadBotTeammate(context.Background(), player, shared, selfXUID)
	if updated != 0 {
		t.Errorf("updated = %d, want 0 (self seul dans son equipe)", updated)
	}
}
