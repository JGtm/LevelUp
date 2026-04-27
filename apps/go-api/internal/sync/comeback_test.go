package sync

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis"
)

// ensureDominanceFlagColumn ajoute la colonne dominance_flag si absente.
// EnsurePlayerSchema cree la table de base ; la colonne est ajoutee par la
// migration add_dominance_flag_column qui n'est pas exposee dans les helpers
// de test sync.
func ensureDominanceFlagColumn(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec(
		`ALTER TABLE player_match_enrichment ADD COLUMN IF NOT EXISTS dominance_flag TINYINT DEFAULT 0`,
	); err != nil {
		t.Fatalf("alter player_match_enrichment add dominance_flag: %v", err)
	}
}

// seedCommonComebackFixtures cree match_registry, match_participants, et
// (optionnel) une medaille Steaktacular. Utilise pour tester
// BackfillDominanceFlags sans dependre du pipeline complet du sync.
func seedComebackMatch(
	t *testing.T,
	sharedDB *sql.DB,
	matchID, gameVariant string,
	myXUID string,
	myTeamID, myOutcome int,
	otherTeamID int,
) {
	t.Helper()
	ctx := context.Background()
	if _, err := sharedDB.ExecContext(ctx, `
		INSERT INTO match_registry (match_id, start_time, game_variant_name)
		VALUES (?, '2026-04-26T10:00:00Z', ?)`,
		matchID, gameVariant); err != nil {
		t.Fatalf("insert match_registry: %v", err)
	}
	if _, err := sharedDB.ExecContext(ctx, `
		INSERT INTO match_participants (match_id, xuid, team_id, outcome)
		VALUES (?, ?, ?, ?)`,
		matchID, myXUID, myTeamID, myOutcome); err != nil {
		t.Fatalf("insert participants me: %v", err)
	}
	if _, err := sharedDB.ExecContext(ctx, `
		INSERT INTO match_participants (match_id, xuid, team_id, outcome)
		VALUES (?, ?, ?, ?)`,
		matchID, "enemy_xuid", otherTeamID, 3 /* loss for enemy if I win */); err != nil {
		t.Fatalf("insert participants enemy: %v", err)
	}
}

func seedSteaktacularMedal(t *testing.T, sharedDB *sql.DB, matchID, xuid string) {
	t.Helper()
	if _, err := sharedDB.ExecContext(context.Background(), `
		INSERT INTO medals_earned (medal_name_id, xuid, match_id, count)
		VALUES (?, ?, ?, 1)`,
		analysis.MedalSteaktacularID, xuid, matchID); err != nil {
		t.Fatalf("insert medal: %v", err)
	}
}

func readDominanceFlag(t *testing.T, playerDB *sql.DB, matchID string) int {
	t.Helper()
	var flag sql.NullInt64
	row := playerDB.QueryRowContext(context.Background(),
		`SELECT dominance_flag FROM player_match_enrichment WHERE match_id = ?`, matchID)
	if err := row.Scan(&flag); err != nil {
		t.Fatalf("read dominance_flag for %s: %v", matchID, err)
	}
	if !flag.Valid {
		return 0
	}
	return int(flag.Int64)
}

func TestBackfillDominanceFlags_DominationFromMedalSteaktacular(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const matchID, myXUID = "m_dom", "me"
	seedComebackMatch(t, sharedDB, matchID, "Slayer", myXUID, 0 /* my team */, 2 /* my outcome win */, 1 /* enemy team */)
	seedSteaktacularMedal(t, sharedDB, matchID, myXUID) // Steaktacular gagnee par MOI

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagDomination {
		t.Errorf("flag want %d (Domination), got %d", analysis.DominanceFlagDomination, got)
	}
}

func TestBackfillDominanceFlags_HumiliationFromEnemySteaktacular(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const matchID, myXUID = "m_hum", "me"
	seedComebackMatch(t, sharedDB, matchID, "Slayer", myXUID, 0, 3 /* loss */, 1)
	seedSteaktacularMedal(t, sharedDB, matchID, "enemy_xuid") // Steaktacular gagnee par L'ENNEMI

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagHumiliation {
		t.Errorf("flag want %d (Humiliation), got %d", analysis.DominanceFlagHumiliation, got)
	}
}

func TestBackfillDominanceFlags_NonSlayerNoFlag(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const matchID, myXUID = "m_ctf", "me"
	seedComebackMatch(t, sharedDB, matchID, "CTF" /* pas Slayer */, myXUID, 0, 2, 1)
	// Pas de Steaktacular -> aucune branche ne s'applique pour CTF.

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagNone {
		t.Errorf("non-Slayer sans medaille -> flag 0 attendu, got %d", got)
	}
}

func TestBackfillDominanceFlags_PersistsToPlayerEnrichment(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const matchID, myXUID = "m_persist", "me"
	seedComebackMatch(t, sharedDB, matchID, "Slayer", myXUID, 0, 2, 1)
	seedSteaktacularMedal(t, sharedDB, matchID, myXUID)

	// Pas d'INSERT prealable dans player_match_enrichment : writeDominanceFlag
	// utilise INSERT ... ON CONFLICT DO UPDATE, donc il doit creer la row.
	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}

	var count int
	if err := playerDB.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM player_match_enrichment WHERE match_id = ?`, matchID).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Errorf("attendu 1 row dans player_match_enrichment, got %d", count)
	}
}

func TestBackfillDominanceFlags_UpdatesExistingRow(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const matchID, myXUID = "m_update", "me"
	seedComebackMatch(t, sharedDB, matchID, "Slayer", myXUID, 0, 2, 1)
	seedSteaktacularMedal(t, sharedDB, matchID, myXUID)

	// Pre-inserer une row avec flag 0 pour simuler un sync precedent sans flag.
	if _, err := playerDB.ExecContext(context.Background(),
		`INSERT INTO player_match_enrichment (match_id, dominance_flag)
		 VALUES (?, 0)`, matchID); err != nil {
		t.Fatalf("pre-insert: %v", err)
	}

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID, []string{matchID}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, matchID); got != analysis.DominanceFlagDomination {
		t.Errorf("flag should be updated to %d, got %d", analysis.DominanceFlagDomination, got)
	}
}

func TestBackfillDominanceFlags_BatchMultipleMatches(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const myXUID = "me"
	seedComebackMatch(t, sharedDB, "m_a", "Slayer", myXUID, 0, 2, 1)
	seedSteaktacularMedal(t, sharedDB, "m_a", myXUID)
	seedComebackMatch(t, sharedDB, "m_b", "Slayer", myXUID, 0, 3, 1)
	seedSteaktacularMedal(t, sharedDB, "m_b", "enemy_xuid")
	seedComebackMatch(t, sharedDB, "m_c", "CTF", myXUID, 0, 2, 1)

	if err := BackfillDominanceFlags(context.Background(), sharedDB, playerDB, myXUID,
		[]string{"m_a", "m_b", "m_c"}); err != nil {
		t.Fatalf("BackfillDominanceFlags: %v", err)
	}
	if got := readDominanceFlag(t, playerDB, "m_a"); got != analysis.DominanceFlagDomination {
		t.Errorf("m_a want Domination, got %d", got)
	}
	if got := readDominanceFlag(t, playerDB, "m_b"); got != analysis.DominanceFlagHumiliation {
		t.Errorf("m_b want Humiliation, got %d", got)
	}
	if got := readDominanceFlag(t, playerDB, "m_c"); got != analysis.DominanceFlagNone {
		t.Errorf("m_c want None (CTF), got %d", got)
	}
}

// ─── Tests pour selectMatchesForComebackBadges ──────────────────────────────

func TestSelectMatchesForComebackBadges_ForceAllReturnsAll(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const myXUID = "me"
	for _, id := range []string{"m1", "m2", "m3"} {
		seedComebackMatch(t, sharedDB, id, "Slayer", myXUID, 0, 2, 1)
	}
	// Pre-insertion d'un flag pour m1 (deja traite)
	if _, err := playerDB.ExecContext(context.Background(),
		`INSERT INTO player_match_enrichment (match_id, dominance_flag) VALUES ('m1', 1)`); err != nil {
		t.Fatalf("pre-insert: %v", err)
	}

	got, err := selectMatchesForComebackBadges(context.Background(), playerDB, sharedDB, myXUID, true)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("forceAll=true should return all 3 matches, got %d: %v", len(got), got)
	}
}

func TestSelectMatchesForComebackBadges_DefaultExcludesAlreadyFlagged(t *testing.T) {
	playerDB, sharedDB := newInMemoryDBs(t)
	ensureDominanceFlagColumn(t, playerDB)
	const myXUID = "me"
	for _, id := range []string{"m1", "m2", "m3"} {
		seedComebackMatch(t, sharedDB, id, "Slayer", myXUID, 0, 2, 1)
	}
	// m1 deja flagge (flag > 0)
	if _, err := playerDB.ExecContext(context.Background(),
		`INSERT INTO player_match_enrichment (match_id, dominance_flag) VALUES ('m1', 1)`); err != nil {
		t.Fatalf("pre-insert: %v", err)
	}
	// m2 a une row mais flag=0 (jamais traite) -> doit etre re-traite
	if _, err := playerDB.ExecContext(context.Background(),
		`INSERT INTO player_match_enrichment (match_id, dominance_flag) VALUES ('m2', 0)`); err != nil {
		t.Fatalf("pre-insert m2: %v", err)
	}

	got, err := selectMatchesForComebackBadges(context.Background(), playerDB, sharedDB, myXUID, false)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected m2 + m3, got %d: %v", len(got), got)
	}
	for _, id := range got {
		if id == "m1" {
			t.Errorf("m1 deja flagge ne devrait pas etre selectionne (got %v)", got)
		}
	}
}
