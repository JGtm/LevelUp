package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	_ "modernc.org/sqlite"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/sync"
)

const testE2EXUID = "2533274823110022"

// e2eEnv bundles every dependency the end-to-end test needs.
type e2eEnv struct {
	cfg      *config.AppConfig
	sharedDB *sql.DB
	jobStore *jobs.Store
	handler  *OpenSpartanImportHandler
	tempDir  string
}

// setupE2E builds the full backend stack (shared DuckDB + metadata DuckDB +
// import service + post-import service + jobs store + handler) wired against
// a freshly-created temp directory. The post-import service is included so
// the recompute pipeline runs end-to-end alongside the raw import.
func setupE2E(t *testing.T) *e2eEnv {
	t.Helper()
	tempDir := t.TempDir()

	cfg := &config.AppConfig{
		RepoRoot: tempDir,
		DemoMode: false,
	}

	// Shared DuckDB (RW) + schema.
	sharedPath := config.SharedDBPath(cfg, "")
	if err := os.MkdirAll(filepath.Dir(sharedPath), 0o755); err != nil {
		t.Fatalf("mkdir shared dir: %v", err)
	}
	sharedDB, err := sql.Open("duckdb", sharedPath)
	if err != nil {
		t.Fatalf("open shared duckdb: %v", err)
	}
	t.Cleanup(func() { _ = sharedDB.Close() })
	if err := sync.EnsureSharedSchema(t.Context(), sharedDB); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	// highlight_events lives in a separate migration — mirror it here.
	if _, err := sharedDB.Exec(`
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
		CREATE TABLE IF NOT EXISTS highlight_events (
			id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id VARCHAR NOT NULL, event_type VARCHAR NOT NULL,
			time_ms INTEGER, xuid VARCHAR, type_hint INTEGER, raw_json VARCHAR
		);
	`); err != nil {
		t.Fatalf("create highlight_events: %v", err)
	}
	// Colonnes ajoutées par migration title-owned, absentes de sharedSchemaSQL
	// statique mais écrites par persist.SharedPersister (E1 route l'import via ce
	// persister). Sans ce mirroring : Binder Error "column ..." → import 0 rows.
	for _, col := range []string{"match_intensity DOUBLE", "backfill_completed BIGINT DEFAULT 0"} {
		if _, err := sharedDB.Exec("ALTER TABLE match_registry ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch match_registry %s: %v", col, err)
		}
	}
	for _, col := range []string{
		"backfill_bits INTEGER",
		"assassination_kills SMALLINT DEFAULT 0",
		"ground_pound_kills SMALLINT DEFAULT 0",
		"shoulder_bash_kills SMALLINT DEFAULT 0",
	} {
		if _, err := sharedDB.Exec("ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS " + col); err != nil {
			t.Fatalf("patch match_participants %s: %v", col, err)
		}
	}

	// Metadata DuckDB (citation_mappings — empty is fine, BackfillMatchCitations
	// short-circuits with "no mappings" warning, which is what we want for E2E
	// without injecting fake medal rules).
	metaPath := titlePkg.NewPathResolver(cfg.RepoRoot).MetadataDBPath(titlePkg.DefaultSlug)
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		t.Fatalf("mkdir meta dir: %v", err)
	}
	metaDB, err := sql.Open("duckdb", metaPath)
	if err != nil {
		t.Fatalf("open metadata duckdb: %v", err)
	}
	// Schema mirrors the columns sync.loadFullCitationMappings scans —
	// keeping the table empty is fine (BackfillMatchCitations short-circuits
	// with "no mappings"), but the SELECT must bind successfully.
	if _, err := metaDB.Exec(`
		CREATE TABLE IF NOT EXISTS citation_mappings (
			citation_name_norm  VARCHAR,
			citation_name_display VARCHAR,
			mapping_type        VARCHAR,
			category            VARCHAR,
			medal_id            BIGINT,
			medal_ids           VARCHAR,
			stat_name           VARCHAR,
			award_name          VARCHAR,
			custom_function     VARCHAR,
			composite_children  VARCHAR,
			tier_targets        VARCHAR,
			enabled             BOOLEAN DEFAULT TRUE
		);
		CREATE TABLE IF NOT EXISTS weapon_labels (weapon_id UBIGINT, label VARCHAR);
	`); err != nil {
		t.Fatalf("create metadata tables: %v", err)
	}
	_ = metaDB.Close() // services will re-open RW via OpenReadWriteShared.

	importSvc := service.NewOpenSpartanImportServiceForTest(sharedDB)
	postImportSvc := service.NewOpenSpartanPostImportService(cfg)
	jobStore := jobs.NewStore(filepath.Join(tempDir, "jobs.json"))

	handler := NewOpenSpartanImportHandler(OpenSpartanImportConfig{
		ImportService:     importSvc,
		PostImportService: postImportSvc,
		JobStore:          jobStore,
		TempDir:           filepath.Join(tempDir, "tmp"),
		StashDir:          filepath.Join(tempDir, "players"),
		DemoMode:          false,
	})

	return &e2eEnv{
		cfg:      cfg,
		sharedDB: sharedDB,
		jobStore: jobStore,
		handler:  handler,
		tempDir:  tempDir,
	}
}

// pollJobUntilDone polls the job store every 100ms until the job reaches a
// terminal state (succeeded / failed / cancelled / interrupted) or the
// timeout elapses.
func pollJobUntilDone(t *testing.T, store *jobs.Store, jobID string, timeout time.Duration) *domain.AsyncJobStatus {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job := store.Get(jobID)
		if job != nil && job.IsTerminal() {
			return job
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("job %s did not complete within %v", jobID, timeout)
	return nil
}

// submitImport posts a multipart upload to the handler with the given file
// bytes and a session built from the supplied owner identity. Returns the
// HTTP response recorder and the decoded JSON body.
func submitImport(t *testing.T, h *OpenSpartanImportHandler, fileBytes []byte, ownerXUID, ownerGamertag string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	sess := &domain.SessionData{
		LinkedHaloIdentity: &domain.HaloIdentity{XUID: ownerXUID, Gamertag: ownerGamertag},
	}
	req := buildMultipartRequest(t, "db", fileBytes, sess)
	rr := httptest.NewRecorder()
	h.StartImport(rr, req)

	var resp map[string]any
	if rr.Body.Len() > 0 {
		_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	}
	return rr, resp
}

// readFixtureBytes reads the bytes of a freshly-built fixture file.
func readFixtureBytes(t *testing.T, env *e2eEnv, ownerXUID string) []byte {
	t.Helper()
	fixturePath := buildMinimalOpenSpartanFile(t, env.tempDir, ownerXUID)
	bytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return bytes
}

// ─── E2E TESTS ────────────────────────────────────────────────────────────────

func TestE2E_FullPipeline_HappyPath(t *testing.T) {
	env := setupE2E(t)
	fileBytes := readFixtureBytes(t, env, testE2EXUID)

	rr, resp := submitImport(t, env.handler, fileBytes, testE2EXUID, "TestOwner")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("StartImport: want 202, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	jobID, _ := resp["job_id"].(string)
	if jobID == "" {
		t.Fatalf("no job_id in response: %v", resp)
	}

	job := pollJobUntilDone(t, env.jobStore, jobID, 30*time.Second)

	if job.Status != domain.JobStatusSucceeded {
		errCode := ""
		errMsg := ""
		if job.Error != nil {
			errCode = job.Error.Code
			errMsg = job.Error.Message
		}
		t.Fatalf("job status: want succeeded, got %s (err=%s: %s)", job.Status, errCode, errMsg)
	}

	if job.JobType != string(domain.JobTypeOpenSpartanImport) {
		t.Errorf("JobType: want %s, got %s", domain.JobTypeOpenSpartanImport, job.JobType)
	}
	if job.FinishedAt == nil {
		t.Error("FinishedAt should be set on a terminal job")
	}
	if job.Result == nil {
		t.Fatal("Result should be populated on success")
	}

	// Result values are stored as native Go types in memory (not reloaded
	// from JSON during this test), so numbers stay int / int16 etc.
	if got := numAt(job.Result, "inserted_matches"); got < 1 {
		t.Errorf("inserted_matches: want >=1, got %v", got)
	}
	if got := numAt(job.Result, "inserted_participants"); got < 1 {
		t.Errorf("inserted_participants: want >=1, got %v", got)
	}
	if got := numAt(job.Result, "inserted_aliases"); got < 1 {
		t.Errorf("inserted_aliases: want >=1 (fixture has 1 alias), got %v", got)
	}
	if got := numAt(job.Result, "stashed_friends"); got < 1 {
		t.Errorf("stashed_friends: want >=1 (fixture has 1 friend), got %v", got)
	}
	if v, _ := job.Result["detected_owner_xuid"].(string); v != testE2EXUID {
		t.Errorf("detected_owner_xuid: want %s, got %v", testE2EXUID, v)
	}
	// The handler stages the uploaded `.db` to a tmp path with a uuid name,
	// so filename-based heuristic does NOT match. Only the frequency
	// heuristic identifies the owner — confidence is "medium" by design.
	if v, _ := job.Result["confidence"].(string); v != "medium" {
		t.Errorf("confidence: want medium (tmp file path strips XUID hint), got %v", v)
	}

	// Post-import block must be present even if recompute counts are zero
	// (e.g. fewer than 10 matches for perf scores).
	post, ok := job.Result["post_import"].(map[string]any)
	if !ok {
		t.Fatalf("post_import block missing from Result: %v", job.Result)
	}
	if _, ok := post["sessions_touched"]; !ok {
		t.Error("post_import.sessions_touched key missing")
	}

	// Verify rows landed in DuckDB.
	verifyRowCount(t, env.sharedDB, "match_registry", 1)
	var firstSyncBy string
	if err := env.sharedDB.QueryRow(`SELECT first_sync_by FROM match_registry LIMIT 1`).Scan(&firstSyncBy); err != nil {
		t.Fatalf("scan first_sync_by: %v", err)
	}
	if firstSyncBy != "openspartan_import" {
		t.Errorf("first_sync_by: want openspartan_import, got %q", firstSyncBy)
	}

	// Friends stash on disk.
	stashPath := filepath.Join(env.tempDir, "players", testE2EXUID, "stash", "openspartan_friends.json")
	if _, err := os.Stat(stashPath); err != nil {
		t.Errorf("friends stash should exist at %s: %v", stashPath, err)
	}

	// Temp uploaded file must have been deleted.
	entries, _ := os.ReadDir(filepath.Join(env.tempDir, "tmp"))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".db" {
			t.Errorf("temp upload should have been deleted, found %s", e.Name())
		}
	}
}

func TestE2E_FullPipeline_RejectsXUIDMismatchAsync(t *testing.T) {
	env := setupE2E(t)
	fileBytes := readFixtureBytes(t, env, testE2EXUID)

	// Session XUID does NOT match the database's owner XUID — the import
	// service must refuse, and the failure must surface as a failed job
	// (not as an HTTP error, because the validation happens inside the
	// goroutine after the file is staged).
	rr, resp := submitImport(t, env.handler, fileBytes, "9999999999999999", "Attacker")
	if rr.Code != http.StatusAccepted {
		t.Fatalf("StartImport: want 202 (validation is async), got %d", rr.Code)
	}
	jobID, _ := resp["job_id"].(string)

	job := pollJobUntilDone(t, env.jobStore, jobID, 10*time.Second)

	if job.Status != domain.JobStatusFailed {
		t.Fatalf("status: want failed, got %s", job.Status)
	}
	if job.Error == nil {
		t.Fatal("Error should be populated on failure")
	}
	if job.Error.Code != "xuid_mismatch" {
		t.Errorf("error code: want xuid_mismatch, got %q (msg=%s)", job.Error.Code, job.Error.Message)
	}

	// Critical security check: nothing must have been written.
	verifyRowCount(t, env.sharedDB, "match_registry", 0)
	verifyRowCount(t, env.sharedDB, "match_participants", 0)
	verifyRowCount(t, env.sharedDB, "medals_earned", 0)
}

func TestE2E_FullPipeline_IdempotentOnReimport(t *testing.T) {
	env := setupE2E(t)
	fileBytes := readFixtureBytes(t, env, testE2EXUID)

	// First import.
	_, resp1 := submitImport(t, env.handler, fileBytes, testE2EXUID, "TestOwner")
	job1 := pollJobUntilDone(t, env.jobStore, resp1["job_id"].(string), 30*time.Second)
	if job1.Status != domain.JobStatusSucceeded {
		t.Fatalf("first import failed: %v", job1.Error)
	}
	var registryAfterFirst, participantsAfterFirst, medalsAfterFirst int
	env.sharedDB.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&registryAfterFirst)
	env.sharedDB.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&participantsAfterFirst)
	env.sharedDB.QueryRow(`SELECT COUNT(*) FROM medals_earned`).Scan(&medalsAfterFirst)
	if registryAfterFirst != 1 {
		t.Fatalf("after first import: want 1 registry row, got %d", registryAfterFirst)
	}

	// Second import of the exact same file. The service should detect the
	// existing match via INSERT OR IGNORE-style upserts and NOT duplicate.
	_, resp2 := submitImport(t, env.handler, fileBytes, testE2EXUID, "TestOwner")
	job2 := pollJobUntilDone(t, env.jobStore, resp2["job_id"].(string), 30*time.Second)
	if job2.Status != domain.JobStatusSucceeded {
		t.Fatalf("second import failed: %v", job2.Error)
	}
	var registryAfterSecond, participantsAfterSecond, medalsAfterSecond int
	env.sharedDB.QueryRow(`SELECT COUNT(*) FROM match_registry`).Scan(&registryAfterSecond)
	env.sharedDB.QueryRow(`SELECT COUNT(*) FROM match_participants`).Scan(&participantsAfterSecond)
	env.sharedDB.QueryRow(`SELECT COUNT(*) FROM medals_earned`).Scan(&medalsAfterSecond)

	if registryAfterSecond != registryAfterFirst {
		t.Errorf("match_registry duplicated: %d → %d", registryAfterFirst, registryAfterSecond)
	}
	if participantsAfterSecond != participantsAfterFirst {
		t.Errorf("match_participants duplicated: %d → %d", participantsAfterFirst, participantsAfterSecond)
	}
	if medalsAfterSecond != medalsAfterFirst {
		t.Errorf("medals_earned duplicated: %d → %d", medalsAfterFirst, medalsAfterSecond)
	}
}

func verifyRowCount(t *testing.T, db *sql.DB, table string, want int) {
	t.Helper()
	var got int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Errorf("table %s: want %d rows, got %d", table, want, got)
	}
}

// numAt reads a numeric field from a job.Result map, tolerating the various
// concrete types Go services may store (int, int16, int32, int64, float64).
// Returns 0 when the key is missing or the value is not numeric.
func numAt(m map[string]any, key string) int64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}
