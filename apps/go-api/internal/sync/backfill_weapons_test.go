//go:build integration

package sync

import (
	"context"
	"database/sql"
	"testing"

	"levelup/go-api/internal/analysis"

	_ "github.com/duckdb/duckdb-go/v2"
)

// ─── Compositions mono-handle (tests) ────────────────────────────────────────
//
// En PROD le post-sync exécute COLLECT (hors writer, lectures RO) puis FLUSH
// (burst writer court) — cf. postSyncFilmSteps.runWeaponKills. Les contrats
// historiques (bit-honnêteté, done/noFilm, idempotence, annulation) restent
// vérifiés ici en composant les deux phases sur un MÊME handle : ces helpers
// remplacent les ex-BackfillWeaponKillsForMatch / processWeaponKillsInline,
// supprimés avec leur dernier caller de prod (règle « 0 code mort »).

// weaponKillsOneShot : COLLECT+FLUSH d'un match unique.
// Retourne (filmFound, error) — mêmes valeurs que l'ex-BackfillWeaponKillsForMatch.
func weaponKillsOneShot(ctx context.Context, client HaloClient, db *sql.DB, matchID, xuid string) (bool, error) {
	collected, err := collectWeaponKillsForMatch(ctx, client, db, matchID, xuid)
	if err != nil {
		return collected.filmFound, err
	}
	if err := flushWeaponKillsForMatch(ctx, db, collected); err != nil {
		return collected.filmFound, err
	}
	return collected.filmFound, nil
}

// weaponKillsInline : COLLECT+FLUSH d'un lot sur un même handle.
// Contrat (identique à l'ex-processWeaponKillsInline) :
//   - matchIDs vide → (0, 0, nil)
//   - tous films présents → done = len(matchIDs), noFilm = 0 ; tous absents → l'inverse
//   - 1 call par match_id, idempotent sur runs consécutifs
//   - ctx.Cancel → ctx.Err() (le lot collecté non flushé est repris au cycle suivant)
func weaponKillsInline(ctx context.Context, db *sql.DB, client HaloClient, xuid string, matchIDs []string) (done, noFilm int, err error) {
	if len(matchIDs) == 0 {
		return 0, 0, nil
	}
	done, noFilm = flushWeaponKillsChunk(ctx, db, collectWeaponKillsChunk(ctx, db, client, xuid, matchIDs))
	if ctx.Err() != nil {
		return done, noFilm, ctx.Err()
	}
	return done, noFilm, nil
}

func openWeaponDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	ddl := `
		CREATE TABLE highlight_events (
			match_id VARCHAR, xuid VARCHAR,
			event_type VARCHAR, time_ms INTEGER
		);
		CREATE TABLE killer_victim_pairs (
			match_id VARCHAR, killer_xuid VARCHAR,
			weapon_type VARCHAR, time_ms INTEGER
		);
		CREATE TABLE match_participants (
			match_id VARCHAR, xuid VARCHAR,
			team_id INTEGER, rank INTEGER
		);
		CREATE TABLE match_registry (
			match_id VARCHAR PRIMARY KEY,
			backfill_completed INTEGER DEFAULT 0
		);
		CREATE SEQUENCE IF NOT EXISTS weapon_kills_generation_seq START 1;
		CREATE TABLE weapon_kills (
			match_id VARCHAR, xuid VARCHAR,
			time_ms INTEGER, weapon_id UBIGINT,
			reconciled_as UBIGINT, delta_ms INTEGER,
			confidence VARCHAR, attribution_path VARCHAR,
			swap_detected BOOLEAN, delayed_damage BOOLEAN,
			player_index INTEGER, generation_id BIGINT DEFAULT 0
		);
		CREATE VIEW v_weapon_kills AS
		SELECT * EXCLUDE (rk) FROM (
			SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id,
			       DENSE_RANK() OVER (PARTITION BY match_id, xuid ORDER BY generation_id DESC) AS rk
			FROM weapon_kills)
		WHERE rk = 1;
	`
	if err := execScript(t.Context(), db, ddl); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestGetKillsForPlayer_Empty(t *testing.T) {
	db := openWeaponDB(t)
	kills, err := getKillsForPlayer(t.Context(), db, "m1", "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 0 {
		t.Fatalf("expected 0, got %d", len(kills))
	}
}

func TestGetKillsForPlayer_WithData(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid1', 'kill', 10000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m1', 'xuid2', 'kill', 7000)`)

	kills, err := getKillsForPlayer(t.Context(), db, "m1", "xuid1")
	if err != nil {
		t.Fatal(err)
	}
	if len(kills) != 2 {
		t.Fatalf("expected 2, got %d", len(kills))
	}
	if kills[0].TimeMS != 5000 {
		t.Fatalf("expected time_ms=5000, got %d", kills[0].TimeMS)
	}
}

func TestGetXuidToPI_Empty(t *testing.T) {
	db := openWeaponDB(t)
	result, err := getXuidToPI(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestGetXuidToPI_WithData(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid1', 0, 1)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid2', 0, 2)`)
	db.Exec(`INSERT INTO match_participants VALUES ('m1', 'xuid3', 1, 1)`)

	result, err := getXuidToPI(t.Context(), db, "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	if result["xuid1"] != 0 {
		t.Fatalf("expected xuid1=0, got %d", result["xuid1"])
	}
}

func TestInsertWeaponKills_Empty(t *testing.T) {
	db := openWeaponDB(t)
	err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestInsertWeaponKills_WithData(t *testing.T) {
	db := openWeaponDB(t)
	rows := []WeaponKillRow{
		{TimeMS: 5000, WeaponID: ptrU64(123), Confidence: "high", AttributionPath: "fire_event"},
		{TimeMS: 10000, WeaponID: ptrU64(456), Confidence: "medium", AttributionPath: "timeline"},
	}
	err := InsertWeaponKills(t.Context(), db, "m1", "xuid1", rows)
	if err != nil {
		t.Fatal(err)
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills").Scan(&count)
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestMarkWeaponKillsDone_WithFilm(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	err := MarkWeaponKillsDone(t.Context(), db, "m1", false)
	if err != nil {
		t.Fatal(err)
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKills) == 0 {
		t.Fatal("expected weapon kills bit set")
	}
}

func TestMarkWeaponKillsDone_NoFilm(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec("INSERT INTO match_registry (match_id) VALUES ('m1')")

	err := MarkWeaponKillsDone(t.Context(), db, "m1", true)
	if err != nil {
		t.Fatal(err)
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m1'").Scan(&bits)
	if bits&int(MBitWeaponKillsNoFilm) == 0 {
		t.Fatal("expected weapon kills no-film bit set")
	}
}

func ptrU64(v uint64) *uint64 { return &v }

// ─── weaponTestClient ────────────────────────────────────────────────────────

// weaponTestClient est un mock minimal de HaloClient pour les tests du pipeline weapons (COLLECT+FLUSH).
type weaponTestClient struct {
	filmChunks  map[int]FilmChunkData
	filmPresent bool
	filmErr     error
}

func (w *weaponTestClient) GetMatchHistory(_ context.Context, _, _ string, _, _ int) ([]MatchHistoryEntry, error) {
	return nil, nil
}
func (w *weaponTestClient) GetMatchStats(_ context.Context, _ string) (map[string]any, error) {
	return nil, nil
}
func (w *weaponTestClient) GetMatchSkill(_ context.Context, _ string, _ []string) (map[string]*MatchSkillData, error) {
	return map[string]*MatchSkillData{}, nil
}
func (w *weaponTestClient) GetMatchFilm(_ context.Context, _ string) (map[int]FilmChunkData, bool, error) {
	return w.filmChunks, w.filmPresent, w.filmErr
}
func (w *weaponTestClient) GetHighlightEventsChunk(_ context.Context, _ string) ([]byte, int, bool, error) {
	return nil, 0, false, nil
}
func (w *weaponTestClient) GetCareerRank(_ context.Context, _ string) (*CareerRankData, error) {
	return nil, nil
}
func (w *weaponTestClient) GetPlayerCSRs(_ context.Context, _, _ string) ([]PlayerPlaylistCSR, error) {
	return nil, nil
}

func (w *weaponTestClient) GetPlaylistCsr(_ context.Context, _, _, _ string) (*PlayerPlaylistCSR, error) {
	return nil, nil
}

// ─── Pipeline weapons par match (COLLECT+FLUSH) ──────────────────────────────

// TestBackfillWeaponKillsForMatch_NoFilm vérifie que le pipeline s'arrête
// proprement quand le film est absent (404/410) et que MBitWeaponKillsNoFilm est posé.
func TestBackfillWeaponKillsForMatch_NoFilm(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m_nofilm')`)

	client := &weaponTestClient{filmPresent: false}
	found, err := weaponKillsOneShot(context.Background(), client, db, "m_nofilm", "xuid1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Fatal("expected found=false when film absent")
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m_nofilm'").Scan(&bits)
	if bits&int(MBitWeaponKillsNoFilm) == 0 {
		t.Fatal("expected MBitWeaponKillsNoFilm bit set")
	}
}

// TestBackfillWeaponKillsForMatch_WithHighlightEvents vérifie le pipeline complet
// quand le film est présent et que des highlight_events existent pour le joueur.
// Les chunks binaires vides génèrent 0 fire events → le pipeline court-circuite sur kills==0.
func TestBackfillWeaponKillsForMatch_WithHighlightEvents(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m_hev')`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m_hev', 'xuid1', 'kill', 5000)`)
	db.Exec(`INSERT INTO highlight_events VALUES ('m_hev', 'xuid1', 'kill', 10000)`)

	client := &weaponTestClient{
		filmPresent: true,
		filmChunks: map[int]FilmChunkData{
			0: {Data: []byte{}, StartMS: 0, DurationMS: 1000},
		},
	}
	found, err := weaponKillsOneShot(context.Background(), client, db, "m_hev", "xuid1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true when film present")
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m_hev'").Scan(&bits)
	if bits&int(MBitWeaponKills) == 0 {
		t.Fatal("expected MBitWeaponKills bit set")
	}
}

// TestBackfillWeaponKillsForMatch_EmptyHighlightEvents vérifie que le pipeline
// se termine sans erreur quand highlight_events ne contient pas de kills pour le joueur.
//
// Garde-fou anti-perte de données (cf. thought_log 2026-05-09) : on NE MARQUE
// PAS bit21 quand 0 ligne weapon_kills n'a été insérée. Marquer ce bit alors
// que la table est vide faisait croire au sweep --weapons que ~1010 matchs
// étaient "déjà traités" et empêchait toute relance. Le contrat : bit21 ⇔ rows.
func TestBackfillWeaponKillsForMatch_EmptyHighlightEvents(t *testing.T) {
	db := openWeaponDB(t)
	db.Exec(`INSERT INTO match_registry (match_id) VALUES ('m_empty')`)

	client := &weaponTestClient{
		filmPresent: true,
		filmChunks: map[int]FilmChunkData{
			0: {Data: []byte{}, StartMS: 0, DurationMS: 1000},
		},
	}
	found, err := weaponKillsOneShot(context.Background(), client, db, "m_empty", "xuid1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true when film present")
	}

	var bits int
	db.QueryRow("SELECT backfill_completed FROM match_registry WHERE match_id='m_empty'").Scan(&bits)
	if bits&int(MBitWeaponKills) != 0 {
		t.Fatalf("expected MBitWeaponKills NOT set (no rows inserted = no done), got bits=%d", bits)
	}

	var killCount int
	db.QueryRow("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m_empty'").Scan(&killCount)
	if killCount != 0 {
		t.Fatalf("expected 0 weapon_kills rows, got %d", killCount)
	}
}

// ─────────────────────────────────────────────────────────────────────────────

func TestAttributionsToRows(t *testing.T) {
	attrs := []analysis.KillAttribution{
		{XUID: "xuid1", TimeMS: 5000, WeaponID: ptrU64(123), Confidence: "high"},
		{XUID: "xuid2", TimeMS: 7000, WeaponID: ptrU64(456), Confidence: "low"},
		{XUID: "xuid1", TimeMS: 10000, WeaponID: ptrU64(789), Confidence: "medium"},
	}
	rows := attributionsToRows(attrs, "xuid1")
	if len(rows) != 2 {
		t.Fatalf("expected 2, got %d", len(rows))
	}
}
