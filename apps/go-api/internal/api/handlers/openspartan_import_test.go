package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	_ "modernc.org/sqlite"

	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/sync"
)

const testHandlerXUID = "2533274823110022"

// newJobStoreForTest constructs an isolated jobs.Store backed by a temp file.
func newJobStoreForTest(t *testing.T) *jobs.Store {
	t.Helper()
	return jobs.NewStore(filepath.Join(t.TempDir(), "jobs.json"))
}

// newSharedDuckDBForTest opens a temporary DuckDB shared database with the
// schema needed by sync.Insert*. The high-level service uses the connection.
func newSharedDuckDBForTest(t *testing.T) *sql.DB {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "shared.duckdb")
	db, err := sql.Open("duckdb", tmp)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := sync.EnsureSharedSchema(db); err != nil {
		t.Fatalf("EnsureSharedSchema: %v", err)
	}
	if _, err := db.Exec(`
		CREATE SEQUENCE IF NOT EXISTS highlight_events_id_seq;
		CREATE TABLE IF NOT EXISTS highlight_events (
			id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
			match_id VARCHAR NOT NULL, event_type VARCHAR NOT NULL,
			time_ms INTEGER, xuid VARCHAR, type_hint INTEGER, raw_json VARCHAR
		);
	`); err != nil {
		t.Fatalf("create highlight_events: %v", err)
	}
	return db
}

// buildMinimalOpenSpartanFile writes a tiny but valid OpenSpartan SQLite file
// containing one matchmaking match with the owner XUID.
func buildMinimalOpenSpartanFile(t *testing.T, dir, ownerXUID string) string {
	t.Helper()
	path := filepath.Join(dir, ownerXUID+".db")
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	for _, ddl := range []string{
		`CREATE TABLE MatchStats (ResponseBody TEXT, MatchId TEXT GENERATED ALWAYS AS (json_extract(ResponseBody, '$.MatchId')) VIRTUAL)`,
		`CREATE TABLE PlayerMatchStats (ResponseBody TEXT, MatchId TEXT)`,
		`CREATE TABLE HighlightEvents (MatchId TEXT NOT NULL, ResponseBody TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			t.Fatalf("ddl: %v", err)
		}
	}
	body := `{"MatchId":"11111111-aaaa-bbbb-cccc-000000000001","MatchInfo":{"StartTime":"2026-01-02T20:18:01Z","EndTime":"2026-01-02T20:30:00Z","Duration":"PT12M","PlayableDuration":"PT12M","LifecycleMode":3,"GameVariantCategory":6,"LevelId":"L","MapVariant":{"AssetKind":2,"AssetId":"map-1","VersionId":"v"},"UgcGameVariant":{"AssetKind":6,"AssetId":"var-1","VersionId":"v"},"TeamsEnabled":true,"TeamScoringEnabled":true},"Teams":[],"Players":[{"PlayerId":"xuid(` + ownerXUID + `)","PlayerType":1,"LastTeamId":0,"Outcome":2,"Rank":1,"ParticipationInfo":{"TimePlayed":"PT12M"},"PlayerTeamStats":[{"TeamId":0,"Stats":{"CoreStats":{"Kills":10,"Deaths":5,"Assists":2,"KDA":7}}}]}]}`
	if _, err := db.Exec(`INSERT INTO MatchStats(ResponseBody) VALUES (?)`, body); err != nil {
		t.Fatalf("insert MatchStats: %v", err)
	}
	abs, _ := filepath.Abs(path)
	return abs
}

// buildMultipartRequest creates an HTTP request whose body is a multipart
// form carrying a single field "db" with the given file contents. When
// fieldName is empty, the field is omitted entirely (for negative tests).
func buildMultipartRequest(t *testing.T, fieldName string, fileBytes []byte, ctxSess *domain.SessionData) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if fieldName != "" {
		fw, err := writer.CreateFormFile(fieldName, "upload.db")
		if err != nil {
			t.Fatalf("CreateFormFile: %v", err)
		}
		if _, err := fw.Write(fileBytes); err != nil {
			t.Fatalf("write file part: %v", err)
		}
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/import/openspartan", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if ctxSess != nil {
		req = req.WithContext(middleware.InjectSession(req.Context(), ctxSess))
	}
	return req
}

func sessionWithXUID(xuid string) *domain.SessionData {
	return &domain.SessionData{
		LinkedHaloIdentity: &domain.HaloIdentity{XUID: xuid},
	}
}

func newHandlerForTest(t *testing.T, demoMode bool) *OpenSpartanImportHandler {
	t.Helper()
	sharedDB := newSharedDuckDBForTest(t)
	svc := service.NewOpenSpartanImportService(sharedDB)
	return NewOpenSpartanImportHandler(OpenSpartanImportConfig{
		ImportService: svc,
		JobStore:      newJobStoreForTest(t),
		TempDir:       t.TempDir(),
		StashDir:      filepath.Join(t.TempDir(), "players"),
		DemoMode:      demoMode,
	})
}

func TestStartImport_DemoModeReturns503(t *testing.T) {
	h := newHandlerForTest(t, true)
	req := buildMultipartRequest(t, "db", []byte("ignored"), sessionWithXUID(testHandlerXUID))
	rr := httptest.NewRecorder()

	h.StartImport(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status: want 503, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "demo_mode") {
		t.Errorf("body should mention demo_mode, got %s", rr.Body.String())
	}
}

func TestStartImport_NoSessionReturns401(t *testing.T) {
	h := newHandlerForTest(t, false)
	req := buildMultipartRequest(t, "db", []byte("data"), nil)
	rr := httptest.NewRecorder()

	h.StartImport(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: want 401, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "halo_auth_required") {
		t.Errorf("body should mention halo_auth_required, got %s", rr.Body.String())
	}
}

func TestStartImport_SessionWithoutLinkedIdentityReturns401(t *testing.T) {
	h := newHandlerForTest(t, false)
	sess := &domain.SessionData{} // no LinkedHaloIdentity
	req := buildMultipartRequest(t, "db", []byte("data"), sess)
	rr := httptest.NewRecorder()

	h.StartImport(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: want 401, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestStartImport_MissingDBFieldReturns400(t *testing.T) {
	h := newHandlerForTest(t, false)
	req := buildMultipartRequest(t, "", nil, sessionWithXUID(testHandlerXUID))
	rr := httptest.NewRecorder()

	h.StartImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "upload_failed") {
		t.Errorf("body should mention upload_failed, got %s", rr.Body.String())
	}
}

func TestStartImport_EmptyFileReturns400(t *testing.T) {
	h := newHandlerForTest(t, false)
	req := buildMultipartRequest(t, "db", []byte{}, sessionWithXUID(testHandlerXUID))
	rr := httptest.NewRecorder()

	h.StartImport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d (body=%s)", rr.Code, rr.Body.String())
	}
}

func TestStartImport_HappyPathReturns202WithJobID(t *testing.T) {
	dir := t.TempDir()
	fixturePath := buildMinimalOpenSpartanFile(t, dir, testHandlerXUID)
	fileBytes, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	h := newHandlerForTest(t, false)
	req := buildMultipartRequest(t, "db", fileBytes, sessionWithXUID(testHandlerXUID))
	rr := httptest.NewRecorder()

	h.StartImport(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("status: want 202, got %d (body=%s)", rr.Code, rr.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if id, ok := resp["job_id"].(string); !ok || id == "" {
		t.Errorf("response should carry a non-empty job_id, got %v", resp)
	}
	if status, _ := resp["status"].(string); status != string(domain.JobStatusQueued) && status != string(domain.JobStatusRunning) {
		t.Errorf("status should be queued|running, got %v", resp["status"])
	}
}
