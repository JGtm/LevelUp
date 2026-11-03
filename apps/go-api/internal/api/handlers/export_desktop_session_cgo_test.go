//go:build cgo

package handlers_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/transfer"
)

const otherXuid = "2533274800000002"

// sessionExportServer monte la route d'export derrière une session injectée (xuid, gamertag) : le cookie n'entre
// pas dans l'épreuve, seul ce que le handler lit de la session compte.
func sessionExportServer(t *testing.T, h *handlers.ExportDesktopHandler, xuid, gamertag string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := &domain.SessionData{LinkedHaloIdentity: &domain.HaloIdentity{XUID: xuid, Gamertag: gamertag}}
		h.Export(w, r.WithContext(middleware.InjectSession(r.Context(), sess)))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newExportFixture : une instance hébergée à deux profils suivis (SpartanB et Her), chacun avec sa base joueur.
func newExportFixture(t *testing.T) (*config.AppConfig, *title.PathResolver) {
	t.Helper()
	root := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: root, AppVersion: "9.9.9-test", DBProfilesPath: filepath.Join(root, "db_profiles.json")}
	profiles := `{"version":"2.1","profiles":{"SpartanB":{"db_path":"x","xuid":"` + exportXuid + `"},"Her":{"db_path":"y","xuid":"` + otherXuid + `"}}}`
	if err := os.WriteFile(cfg.DBProfilesPath, []byte(profiles), 0o600); err != nil {
		t.Fatal(err)
	}
	pr := cfg.PathResolver()
	seedExportDB(t, pr.SharedDBPath(title.DefaultSlug),
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, map_name VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, kills INTEGER)`,
		`INSERT INTO match_registry VALUES ('m1','Aquarius')`,
		`INSERT INTO match_participants VALUES ('m1','`+exportXuid+`',5)`)
	enrich := `CREATE TABLE player_match_enrichment (match_id VARCHAR PRIMARY KEY, performance_score DOUBLE)`
	seedExportDB(t, pr.PlayerDBPath(title.DefaultSlug, "SpartanB"), enrich, `INSERT INTO player_match_enrichment VALUES ('m1', 71.5)`)
	seedExportDB(t, pr.PlayerDBPath(title.DefaultSlug, "Her"), enrich, `INSERT INTO player_match_enrichment VALUES ('mHer', 12.0)`)
	return cfg, pr
}

func sessionGet(t *testing.T, srv *httptest.Server, withHeader bool) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	if withHeader {
		req.Header.Set("X-Requested-With", "levelup")
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func exportEntries(t *testing.T, resp *http.Response) map[string][]byte {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("archive illisible (%d) : %v", resp.StatusCode, err)
	}
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, _ := f.Open()
		out[f.Name], _ = io.ReadAll(rc)
		_ = rc.Close()
	}
	return out
}

// TestExportDesktop_PlayerScopeFollowsTheXuidNotTheSessionGamertag : une session dont le xuid est celui de
// SpartanB mais dont le gamertag est celui d'une autre joueuse n'exporte JAMAIS la base joueur de celle-ci (S5.B).
func TestExportDesktop_PlayerScopeFollowsTheXuidNotTheSessionGamertag(t *testing.T) {
	cfg, _ := newExportFixture(t)
	h := handlers.NewExportDesktopHandler(cfg, transfer.NewTokenStore())
	srv := sessionExportServer(t, h, exportXuid, "Her") // xuid de SpartanB, gamertag de Her
	resp := sessionGet(t, srv, true)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("session valide : %d", resp.StatusCode)
	}
	entries := exportEntries(t, resp)
	var manifest transfer.Manifest
	if err := json.Unmarshal(entries["manifest.json"], &manifest); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tb := range manifest.Tables {
		if tb.Name == "player_match_enrichment" {
			found = true
			if tb.Rows != 1 {
				t.Fatalf("enrichissements exportés : %d ligne(s), attendu 1 (ceux du profil qui porte le xuid)", tb.Rows)
			}
		}
	}
	if !found {
		t.Fatal("la base joueur du profil qui porte le xuid devait être exportée")
	}
	if bytes.Contains(entries["player/player_match_enrichment.parquet"], []byte("mHer")) {
		t.Fatal("la base joueur de l'autre profil a été exportée sur la foi d'un gamertag de session")
	}
}

// TestExportDesktop_NoTrackedProfileSkipsPlayerScope : sans profil suivi pour le xuid, aucune base joueur n'est
// choisie (jamais celle d'un gamertag de session).
func TestExportDesktop_NoTrackedProfileSkipsPlayerScope(t *testing.T) {
	cfg, _ := newExportFixture(t)
	h := handlers.NewExportDesktopHandler(cfg, transfer.NewTokenStore())
	srv := sessionExportServer(t, h, "2533274800000077", "SpartanB") // xuid inconnu, gamertag d'un profil suivi
	for name := range exportEntries(t, sessionGet(t, srv, true)) {
		if strings.HasPrefix(name, "player/") {
			t.Fatalf("entrée %s : aucun profil suivi ne porte ce xuid, la base joueur ne doit pas sortir", name)
		}
	}
}

// TestExportDesktop_SessionNeedsNonSimpleRequest : un simple GET de session (lien tiers, image) ne déclenche pas
// d'export ; le quota n'est pas touché par ce refus (S5.A).
func TestExportDesktop_SessionNeedsNonSimpleRequest(t *testing.T) {
	cfg, _ := newExportFixture(t)
	h := handlers.NewExportDesktopHandler(cfg, transfer.NewTokenStore())
	srv := sessionExportServer(t, h, exportXuid, "SpartanB")
	refused := sessionGet(t, srv, false)
	_ = refused.Body.Close()
	if refused.StatusCode != http.StatusForbidden {
		t.Fatalf("GET de session sans en-tête : %d, attendu 403", refused.StatusCode)
	}
	ok := sessionGet(t, srv, true)
	_ = ok.Body.Close()
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("après le refus, l'export par session doit passer encore : %d", ok.StatusCode)
	}
}

// TestExportDesktop_FailedExportKeepsTheQuota : un export qui échoue (base partagée illisible) ne consomme pas le
// quota d'un par heure : le joueur réessaie tout de suite (S5.A).
func TestExportDesktop_FailedExportKeepsTheQuota(t *testing.T) {
	cfg, pr := newExportFixture(t)
	good, err := os.ReadFile(pr.SharedDBPath(title.DefaultSlug))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pr.SharedDBPath(title.DefaultSlug), []byte("pas une base duckdb"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := handlers.NewExportDesktopHandler(cfg, transfer.NewTokenStore())
	srv := sessionExportServer(t, h, exportXuid, "SpartanB")
	do := func() int {
		resp := sessionGet(t, srv, true)
		_ = resp.Body.Close()
		return resp.StatusCode
	}
	if got := do(); got != http.StatusInternalServerError {
		t.Fatalf("base illisible : %d, attendu 500", got)
	}
	if err := os.WriteFile(pr.SharedDBPath(title.DefaultSlug), good, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := do(); got != http.StatusOK {
		t.Fatalf("réessai après l'échec : %d, attendu 200 (le quota n'a pas été consommé)", got)
	}
	if got := do(); got != http.StatusTooManyRequests {
		t.Fatalf("second export réussi dans l'heure : %d, attendu 429", got)
	}
}

// TestExportDesktop_RequestDuringAnExportIsRefused : un GET qui arrive pendant l'export du même joueur n'en lance
// pas un second (S5.A) : il reçoit 429.
func TestExportDesktop_RequestDuringAnExportIsRefused(t *testing.T) {
	cfg, _ := newExportFixture(t)
	store := transfer.NewTokenStore()
	h := handlers.NewExportDesktopHandler(cfg, store)
	if !store.AllowSession(exportXuid, time.Now()) { // un export du joueur est en cours
		t.Fatal("ouverture de l'export")
	}
	srv := sessionExportServer(t, h, exportXuid, "SpartanB")
	resp := sessionGet(t, srv, true)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("export du joueur en cours : %d, attendu 429", resp.StatusCode)
	}
}
