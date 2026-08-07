//go:build integration

// Package sync — friends_recompute_integration_test.go : tests integration
// pour updateIsWithFriendsBatch avec DuckDB :memory:.
//
// Couvre la régression "Solo qui revient" : les lignes player_match_enrichment
// inserées par le sync avant l'ajout de DEFAULT FALSE sont à NULL. La query
// `WHERE is_with_friends = FALSE` historique skippait NULL en logique 3-valeurs
// SQL, donc le badge "Solo" persistait. Le fix passe à `COALESCE(...) = FALSE`
// qui couvre les deux cas.
package sync

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/migration"
)

func openPlayerForFriendsRecompute(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE player_match_enrichment (
			match_id        VARCHAR PRIMARY KEY,
			is_with_friends BOOLEAN,
			updated_at      TIMESTAMP DEFAULT CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		)
	`); err != nil {
		t.Fatalf("create player_match_enrichment: %v", err)
	}
	if err := migration.EnsurePlayerMatchEnrichmentAppendOnly(db); err != nil {
		t.Fatalf("EnsurePlayerMatchEnrichmentAppendOnly: %v", err)
	}
	return db
}

// TestUpdateIsWithFriendsBatch_PromotesNullAndFalse vérifie que le UPDATE
// promeut TANT les lignes is_with_friends=FALSE QUE les lignes
// is_with_friends=NULL. Régression : avant le fix, le filtre `= FALSE`
// skippait NULL → badge Solo permanent sur les matchs récents.
func TestUpdateIsWithFriendsBatch_PromotesNullAndFalse(t *testing.T) {
	db := openPlayerForFriendsRecompute(t)

	if _, err := db.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES
		    ('m_null',  NULL),
		    ('m_false', FALSE),
		    ('m_true',  TRUE),
		    ('m_other', NULL)
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	affected, err := updateIsWithFriendsBatch(context.Background(), db, []string{"m_null", "m_false", "m_true"})
	if err != nil {
		t.Fatalf("updateIsWithFriendsBatch: %v", err)
	}
	// Doit toucher m_null et m_false (2). m_true est déjà TRUE → exclu par la garde
	// `COALESCE(is_with_friends, FALSE) = FALSE`. m_other n'est pas dans la batch.
	if affected != 2 {
		t.Errorf("expected 2 rows promoted (NULL + FALSE), got %d", affected)
	}

	rows, err := db.Query(`
		SELECT match_id, is_with_friends
		FROM player_match_enrichment_latest
		ORDER BY match_id
	`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer rows.Close()

	got := map[string]sql.NullBool{}
	for rows.Next() {
		var mid string
		var v sql.NullBool
		if err := rows.Scan(&mid, &v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[mid] = v
	}

	cases := map[string]sql.NullBool{
		"m_null":  {Bool: true, Valid: true},  // NULL → TRUE (le fix)
		"m_false": {Bool: true, Valid: true},  // FALSE → TRUE (comportement existant)
		"m_true":  {Bool: true, Valid: true},  // TRUE → TRUE (garde idempotente)
		"m_other": {Bool: false, Valid: true}, // pas dans batch : reste legacy NULL, lu FALSE via _latest (COALESCE)
	}
	for mid, want := range cases {
		g := got[mid]
		if g.Valid != want.Valid || g.Bool != want.Bool {
			t.Errorf("match=%s : got is_with_friends=%v(valid=%v), want %v(valid=%v)",
				mid, g.Bool, g.Valid, want.Bool, want.Valid)
		}
	}
}

// TestUpdateIsWithFriendsBatch_Idempotent vérifie que rejouer le UPDATE sur un
// set déjà promu ne touche aucune ligne (garde idempotente via COALESCE).
func TestUpdateIsWithFriendsBatch_Idempotent(t *testing.T) {
	db := openPlayerForFriendsRecompute(t)

	if _, err := db.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES ('m1', NULL)
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	first, err := updateIsWithFriendsBatch(context.Background(), db, []string{"m1"})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if first != 1 {
		t.Errorf("expected 1 promoted on first call, got %d", first)
	}

	second, err := updateIsWithFriendsBatch(context.Background(), db, []string{"m1"})
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if second != 0 {
		t.Errorf("expected 0 promoted on second call (idempotent), got %d", second)
	}
}

// openSharedForFriendsRecompute crée une shared DB :memory: avec les tables
// minimales nécessaires à RecomputeIsWithFriendsCore.
func openSharedForFriendsRecompute(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open shared: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR)
	`); err != nil {
		t.Fatalf("create xuid_aliases: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, team_id INTEGER)
	`); err != nil {
		t.Fatalf("create match_participants: %v", err)
	}
	return db
}

// TestPlayerMatchEnrichment_DefaultFalse vérifie que le DDL de production
// (EnsurePlayerSchema) crée is_with_friends avec DEFAULT FALSE : un INSERT
// sans cette colonne doit retourner FALSE, jamais NULL.
// Guard de premier niveau : élimine la source du bug à la racine DDL.
func TestPlayerMatchEnrichment_DefaultFalse(t *testing.T) {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if err := EnsurePlayerSchema(t.Context(), db); err != nil {
		t.Fatalf("EnsurePlayerSchema: %v", err)
	}

	if _, err := db.Exec(
		`INSERT INTO player_match_enrichment (match_id) VALUES ('m_default')`,
	); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	var v sql.NullBool
	if err := db.QueryRow(
		`SELECT is_with_friends FROM player_match_enrichment WHERE match_id = 'm_default'`,
	).Scan(&v); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if !v.Valid {
		t.Error("is_with_friends = NULL : DEFAULT FALSE manquant dans le schéma de production")
	}
	if v.Bool {
		t.Error("is_with_friends = TRUE après INSERT vide, attendu FALSE")
	}
}

// TestRecomputeIsWithFriendsCore_PromotesLegacyNull vérifie le flux complet :
// résolution gamertag→xuid, détection match commun dans shared, promotion NULL→TRUE.
// Couvre la régression "Solo qui revient" de bout en bout.
func TestRecomputeIsWithFriendsCore_PromotesLegacyNull(t *testing.T) {
	playerDB := openPlayerForFriendsRecompute(t)
	sharedDB := openSharedForFriendsRecompute(t)

	const (
		playerXUID = "xuid_player_001"
		friendXUID = "xuid_friend_001"
		friendGT   = "FriendGamertag"
		matchID    = "m_squad_001"
	)

	if _, err := sharedDB.Exec(
		`INSERT INTO xuid_aliases VALUES (?, ?)`, friendXUID, friendGT,
	); err != nil {
		t.Fatalf("seed xuid_aliases: %v", err)
	}
	// Joueur et ami sur la même équipe (team_id=1).
	if _, err := sharedDB.Exec(`
		INSERT INTO match_participants VALUES (?, ?, ?), (?, ?, ?)`,
		matchID, playerXUID, 1,
		matchID, friendXUID, 1,
	); err != nil {
		t.Fatalf("seed match_participants: %v", err)
	}
	// Ligne héritée avec NULL (inserts avant DEFAULT FALSE).
	if _, err := playerDB.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES (?, NULL)`, matchID,
	); err != nil {
		t.Fatalf("seed player_match_enrichment: %v", err)
	}

	res, err := RecomputeIsWithFriendsCore(
		context.Background(), playerDB, sharedDB, playerXUID, []string{friendGT}, false,
	)
	if err != nil {
		t.Fatalf("RecomputeIsWithFriendsCore: %v", err)
	}
	if res.FriendXUIDsCount != 1 {
		t.Errorf("expected 1 friend xuid resolved, got %d", res.FriendXUIDsCount)
	}
	if res.MatchesPromoted != 1 {
		t.Errorf("expected 1 match promoted, got %d", res.MatchesPromoted)
	}

	var v sql.NullBool
	if err := playerDB.QueryRow(
		`SELECT is_with_friends FROM player_match_enrichment_latest WHERE match_id = ?`, matchID,
	).Scan(&v); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if !v.Valid || !v.Bool {
		t.Errorf("expected is_with_friends = TRUE, got Valid=%v Bool=%v", v.Valid, v.Bool)
	}
}

// TestRecomputeIsWithFriendsCore_DemotesStaleMatches vérifie la sémantique
// CONVERGENTE : un match actuellement TRUE qui n'a plus d'ami courant est démoté
// (TRUE→FALSE). Couvre le retrait d'ami et la suppression du dernier ami.
func TestRecomputeIsWithFriendsCore_DemotesStaleMatches(t *testing.T) {
	playerDB := openPlayerForFriendsRecompute(t)
	sharedDB := openSharedForFriendsRecompute(t)

	const (
		playerXUID = "xuid_player_001"
		friendXUID = "xuid_friend_001"
		friendGT   = "FriendGamertag"
		matchWith  = "m_with_friend" // ami présent → reste TRUE
		matchStale = "m_stale"       // aucun ami → doit être démoté
	)

	if _, err := sharedDB.Exec(`INSERT INTO xuid_aliases VALUES (?, ?)`, friendXUID, friendGT); err != nil {
		t.Fatalf("seed xuid_aliases: %v", err)
	}
	// matchWith : joueur + ami même équipe. matchStale : joueur seul.
	if _, err := sharedDB.Exec(`
		INSERT INTO match_participants VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)`,
		matchWith, playerXUID, 1,
		matchWith, friendXUID, 1,
		matchStale, playerXUID, 1,
	); err != nil {
		t.Fatalf("seed match_participants: %v", err)
	}
	// Les DEUX matchs sont actuellement flaggés TRUE (état hérité).
	if _, err := playerDB.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES (?, TRUE), (?, TRUE)`,
		matchWith, matchStale,
	); err != nil {
		t.Fatalf("seed player_match_enrichment: %v", err)
	}

	// Recompute avec l'ami courant → matchStale doit être démoté, matchWith reste TRUE.
	res, err := RecomputeIsWithFriendsCore(
		context.Background(), playerDB, sharedDB, playerXUID, []string{friendGT}, false,
	)
	if err != nil {
		t.Fatalf("RecomputeIsWithFriendsCore: %v", err)
	}
	if res.MatchesDemoted != 1 {
		t.Errorf("expected 1 match demoted, got %d", res.MatchesDemoted)
	}
	assertIsWithFriends(t, playerDB, matchWith, true)
	assertIsWithFriends(t, playerDB, matchStale, false)

	// Recompute avec liste vide (dernier ami retiré) → matchWith démoté aussi.
	res2, err := RecomputeIsWithFriendsCore(
		context.Background(), playerDB, sharedDB, playerXUID, nil, false,
	)
	if err != nil {
		t.Fatalf("RecomputeIsWithFriendsCore (empty): %v", err)
	}
	if res2.MatchesDemoted != 1 {
		t.Errorf("expected 1 match demoted on empty friends, got %d", res2.MatchesDemoted)
	}
	assertIsWithFriends(t, playerDB, matchWith, false)
}

func assertIsWithFriends(t *testing.T, db *sql.DB, matchID string, want bool) {
	t.Helper()
	var v sql.NullBool
	if err := db.QueryRow(
		`SELECT is_with_friends FROM player_match_enrichment_latest WHERE match_id = ?`, matchID,
	).Scan(&v); err != nil {
		t.Fatalf("SELECT %s: %v", matchID, err)
	}
	if !v.Valid || v.Bool != want {
		t.Errorf("match %s: is_with_friends Valid=%v Bool=%v, want %v", matchID, v.Valid, v.Bool, want)
	}
}

// TestRecomputeIsWithFriendsCore_PlayerInFriendList vérifie que si le joueur
// principal est lui-même dans friendGamertags (setup multi-joueurs où tous les
// membres du groupe sont trackés et listés comme amis), seuls les matchs où un
// AMI DIFFÉRENT était présent sont promus. Sans le guard `p2.xuid != p1.xuid`,
// chaque match du joueur satisferait la self-join → tous les matchs seraient
// faussement promus.
func TestRecomputeIsWithFriendsCore_PlayerInFriendList(t *testing.T) {
	playerDB := openPlayerForFriendsRecompute(t)
	sharedDB := openSharedForFriendsRecompute(t)

	const (
		playerXUID   = "xuid_player_002"
		playerGT     = "PlayerGamertag"
		friendXUID   = "xuid_friend_002"
		friendGT     = "FriendGamertag2"
		soloMatchID  = "m_solo_002"
		squadMatchID = "m_squad_002"
	)

	// Seed shared : joueur dans xuid_aliases (il est aussi dans friend_gamertags)
	if _, err := sharedDB.Exec(`INSERT INTO xuid_aliases VALUES (?, ?), (?, ?)`,
		playerXUID, playerGT, friendXUID, friendGT,
	); err != nil {
		t.Fatalf("seed xuid_aliases: %v", err)
	}
	// Match solo : joueur seul dans l'équipe.
	if _, err := sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?)`,
		soloMatchID, playerXUID, 1,
	); err != nil {
		t.Fatalf("seed solo match: %v", err)
	}
	// Match escouade : joueur + ami dans la même équipe.
	if _, err := sharedDB.Exec(`INSERT INTO match_participants VALUES (?, ?, ?), (?, ?, ?)`,
		squadMatchID, playerXUID, 1,
		squadMatchID, friendXUID, 1,
	); err != nil {
		t.Fatalf("seed squad match: %v", err)
	}
	// Seed player DB : les deux matchs à NULL.
	if _, err := playerDB.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES (?, NULL), (?, NULL)`,
		soloMatchID, squadMatchID,
	); err != nil {
		t.Fatalf("seed player_match_enrichment: %v", err)
	}

	// friend_gamertags inclut le joueur lui-même (setup multi-joueurs complet).
	res, err := RecomputeIsWithFriendsCore(
		context.Background(), playerDB, sharedDB, playerXUID,
		[]string{playerGT, friendGT}, false,
	)
	if err != nil {
		t.Fatalf("RecomputeIsWithFriendsCore: %v", err)
	}
	// Seul le match escouade doit être promu (1, pas 2).
	if res.MatchesPromoted != 1 {
		t.Errorf("expected 1 promoted (squad only), got %d — self-join guard missing?", res.MatchesPromoted)
	}

	// Match solo doit rester NULL (pas promu).
	// Lecture brute (PAS _latest) : la vue applique COALESCE(is_with_friends, FALSE)
	// donc une row legacy NULL y apparaît FALSE — ce test vérifie spécifiquement que
	// le match solo n'a JAMAIS été écrit (reste NULL). Le solo n'est ni promu ni démoté
	// → 1 seule row (legacy) en table brute, lecture sûre.
	var soloV sql.NullBool
	if err := playerDB.QueryRow(
		`SELECT is_with_friends FROM player_match_enrichment WHERE match_id = ?`, soloMatchID,
	).Scan(&soloV); err != nil {
		t.Fatalf("SELECT solo: %v", err)
	}
	if soloV.Valid {
		t.Errorf("solo match should stay NULL (not promoted), got Bool=%v", soloV.Bool)
	}

	// Match escouade doit être TRUE.
	var squadV sql.NullBool
	if err := playerDB.QueryRow(
		`SELECT is_with_friends FROM player_match_enrichment_latest WHERE match_id = ?`, squadMatchID,
	).Scan(&squadV); err != nil {
		t.Fatalf("SELECT squad: %v", err)
	}
	if !squadV.Valid || !squadV.Bool {
		t.Errorf("squad match should be TRUE, got Valid=%v Bool=%v", squadV.Valid, squadV.Bool)
	}
}

// TestRecomputeIsWithFriendsCore_FriendNotResolved vérifie que si l'ami n'est
// pas dans xuid_aliases (jamais croisé en match), le recompute ne touche rien
// et ne retourne pas d'erreur. Ligne reste NULL — comportement safe.
func TestRecomputeIsWithFriendsCore_FriendNotResolved(t *testing.T) {
	playerDB := openPlayerForFriendsRecompute(t)
	sharedDB := openSharedForFriendsRecompute(t)

	if _, err := playerDB.Exec(`
		INSERT INTO player_match_enrichment (match_id, is_with_friends) VALUES ('m1', NULL)`,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, err := RecomputeIsWithFriendsCore(
		context.Background(), playerDB, sharedDB, "xuid_player", []string{"UnknownFriend"}, false,
	)
	if err != nil {
		t.Fatalf("RecomputeIsWithFriendsCore error: %v", err)
	}
	if res.MatchesPromoted != 0 {
		t.Errorf("expected 0 promoted when friend unresolved, got %d", res.MatchesPromoted)
	}

	// Lecture brute (PAS _latest) : la vue COALESCE(is_with_friends, FALSE) masquerait
	// le NULL en FALSE. Ce test vérifie qu'aucune écriture n'a eu lieu (row legacy NULL
	// intacte) : ami non résolu → aucun stage écrit → 1 seule row brute, lecture sûre.
	var v sql.NullBool
	if err := playerDB.QueryRow(
		`SELECT is_with_friends FROM player_match_enrichment WHERE match_id = 'm1'`,
	).Scan(&v); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if v.Valid {
		t.Errorf("expected is_with_friends to stay NULL when friend unresolved, got Bool=%v", v.Bool)
	}
}
