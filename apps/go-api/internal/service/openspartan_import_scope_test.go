package service

// openspartan_import_scope_test.go — D-B (décision utilisateur du 2026-10-15) : l'import
// OpenSpartan n'écrit que les matchs auxquels le joueur qui importe a participé, et
// n'écrase aucune donnée propre à un autre joueur (registre sécurité C4-3).
//
// Le fichier forgé est un vrai export OpenSpartan (SQLite) où le xuid de l'importeur
// domine — la détection du propriétaire l'accepte — et qui ajoute : un match où ne
// jouent que des tiers (avec son événement marquant et l'alias d'un tiers inconnu),
// un match à l'identifiant piégé, et un alias qui renomme un joueur déjà connu.

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

const (
	foreignXUIDA      = "2533274800000001"
	foreignXUIDB      = "2533274800000002"
	foreignMatchID    = "22222222-aaaa-bbbb-cccc-000000000002"
	ownSecondMatchID  = "33333333-aaaa-bbbb-cccc-000000000003"
	trappedMatchID    = "../../../tmp/x"
	knownTierGamertag = "NomReel"
)

// matchStatsPayload : payload MatchStats minimal (deux humains, une équipe chacun).
func matchStatsPayload(matchID string, xuids ...string) string {
	start := time.Date(2026, 1, 3, 20, 0, 0, 0, time.UTC)
	players := make([]any, 0, len(xuids))
	for i, x := range xuids {
		players = append(players, map[string]any{
			"PlayerId": "xuid(" + x + ")", "PlayerType": 1, "LastTeamId": i % 2, "Outcome": 2, "Rank": i + 1,
			"ParticipationInfo": map[string]any{"TimePlayed": "PT10M"},
			"PlayerTeamStats": []any{map[string]any{"TeamId": i % 2,
				"Stats": map[string]any{"CoreStats": map[string]any{"Kills": 3, "Deaths": 2}}}},
		})
	}
	body, _ := json.Marshal(map[string]any{
		"MatchId": matchID,
		"MatchInfo": map[string]any{
			"StartTime": start.Format(time.RFC3339Nano), "EndTime": start.Add(10 * time.Minute).Format(time.RFC3339Nano),
			"Duration": "PT10M", "PlayableDuration": "PT10M", "LifecycleMode": 3, "GameVariantCategory": 6,
			"LevelId":        "level-1",
			"MapVariant":     map[string]any{"AssetKind": 2, "AssetId": "map-1", "VersionId": "map-v1"},
			"UgcGameVariant": map[string]any{"AssetKind": 6, "AssetId": "variant-1", "VersionId": "variant-v1"},
		},
		"Teams": []any{
			map[string]any{"TeamId": 0, "Outcome": 2, "Rank": 1, "Stats": map[string]any{"CoreStats": map[string]any{"Score": 50}}},
			map[string]any{"TeamId": 1, "Outcome": 3, "Rank": 2, "Stats": map[string]any{"CoreStats": map[string]any{"Score": 40}}},
		},
		"Players": players,
	})
	return string(body)
}

// forgeImportFile part de la fixture nominale et lui ajoute les éléments hostiles.
func forgeImportFile(t *testing.T, dir string) string {
	t.Helper()
	path := buildMinimalFixtureDB(t, dir, testOwnerXUID+".db", testOwnerXUID)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = db.Close() }()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.Exec(q, args...); err != nil {
			t.Fatalf("%s : %v", q, err)
		}
	}
	exec(`INSERT INTO MatchStats(ResponseBody) VALUES (?)`, matchStatsPayload(foreignMatchID, foreignXUIDA, foreignXUIDB))
	exec(`INSERT INTO MatchStats(ResponseBody) VALUES (?)`, matchStatsPayload(trappedMatchID, testOwnerXUID, foreignXUIDB))
	// Un second match de l'importeur : son xuid domine le fichier (détection acceptée).
	exec(`INSERT INTO MatchStats(ResponseBody) VALUES (?)`, matchStatsPayload(ownSecondMatchID, testOwnerXUID, testOtherXUID))
	hl, _ := json.Marshal(map[string]any{"event_type": "kill", "time_ms": 1000, "xuid": 2533274800000001, "type_hint": 50})
	exec(`INSERT INTO HighlightEvents(MatchId, ResponseBody) VALUES (?, ?)`, foreignMatchID, string(hl))
	now := time.Now().UTC().Format(time.RFC3339)
	exec(`INSERT INTO XuidAliases(Xuid, Gamertag, Source, UpdatedAt) VALUES (?, ?, 'api', ?)`, foreignXUIDA, "TiersForge", now)
	return path
}

func countWhere(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatalf("%s : %v", q, err)
	}
	return n
}

func TestOpenSpartanImport_WritesOnlyOwnerMatches(t *testing.T) {
	dir := t.TempDir()
	path := forgeImportFile(t, dir)
	sharedDB := setupSharedDB(t)
	// Un joueur déjà connu sous son vrai nom : le fichier le renomme « OtherPlayer ».
	if _, err := sharedDB.Exec(`INSERT INTO xuid_aliases (xuid, gamertag, last_seen, source) VALUES (?, ?, now(), 'sync')`,
		testOtherXUID, knownTierGamertag); err != nil {
		t.Fatalf("seed alias: %v", err)
	}

	res, err := NewOpenSpartanImportServiceForTest(sharedDB).Import(context.Background(), testOwnerXUID, path,
		ImportOptions{StashDir: filepath.Join(dir, "stash")})
	if err != nil {
		t.Fatalf("Import: %v", err)
	}

	if n := countWhere(t, sharedDB, `SELECT count(*) FROM match_registry WHERE match_id = ?`, foreignMatchID); n != 0 {
		t.Errorf("match sans l'importeur écrit (%d ligne(s) dans match_registry)", n)
	}
	if n := countWhere(t, sharedDB, `SELECT count(*) FROM match_participants WHERE xuid IN (?, ?) AND match_id = ?`,
		foreignXUIDA, foreignXUIDB, foreignMatchID); n != 0 {
		t.Errorf("participants d'un match sans l'importeur écrits : %d", n)
	}
	if n := countWhere(t, sharedDB, `SELECT count(*) FROM highlight_events WHERE match_id = ?`, foreignMatchID); n != 0 {
		t.Errorf("événement marquant d'un match sans l'importeur écrit : %d", n)
	}
	if n := countWhere(t, sharedDB, `SELECT count(*) FROM match_registry WHERE match_id = ?`, trappedMatchID); n != 0 {
		t.Errorf("identifiant de match piégé écrit")
	}
	if n := countWhere(t, sharedDB, `SELECT count(*) FROM xuid_aliases WHERE xuid = ?`, foreignXUIDA); n != 0 {
		t.Errorf("alias d'un tiers absent de tout match de l'importeur écrit")
	}
	var gt string
	if err := sharedDB.QueryRow(`SELECT gamertag FROM xuid_aliases WHERE xuid = ?`, testOtherXUID).Scan(&gt); err != nil {
		t.Fatalf("alias du tiers connu : %v", err)
	}
	if gt != knownTierGamertag {
		t.Errorf("alias d'un tiers existant réécrit en %q (attendu inchangé %q)", gt, knownTierGamertag)
	}
	// Le match de l'importeur, lui, est écrit.
	if res.InsertedMatches != 2 {
		t.Errorf("InsertedMatches = %d, attendu 2 (les matchs de l'importeur à identifiant valide)", res.InsertedMatches)
	}
}
