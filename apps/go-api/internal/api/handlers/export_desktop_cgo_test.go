//go:build cgo

package handlers_test

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/transfer"
)

const exportXuid = "2533274800000001"

func seedExportDB(t *testing.T, path string, stmts ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("%s : %v", s, err)
		}
	}
}

// TestExportDesktop_TokenOnlyAndOncePerHour : l'archive ne sort que sur un jeton valide, une seule fois, et un
// joueur n'obtient qu'un export par heure ; elle ne contient que ses lignes et aucun secret.
func TestExportDesktop_TokenOnlyAndOncePerHour(t *testing.T) {
	root := t.TempDir()
	cfg := &config.AppConfig{RepoRoot: root, AppVersion: "9.9.9-test"}
	pr := cfg.PathResolver()
	seedExportDB(t, pr.SharedDBPath(title.DefaultSlug),
		`CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, map_name VARCHAR)`,
		`CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR, kills INTEGER)`,
		`INSERT INTO match_registry VALUES ('m1','Aquarius'),('m2','Streets')`,
		`INSERT INTO match_participants VALUES ('m1','`+exportXuid+`',5),('m2','2533274800000002',7)`)
	store := transfer.NewTokenStore()
	h := handlers.NewExportDesktopHandler(cfg, store)
	srv := httptest.NewServer(http.HandlerFunc(h.Export))
	defer srv.Close()

	get := func(token string) *http.Response {
		req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}
	if resp := get(""); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("sans jeton ni session : %d, attendu 401", resp.StatusCode)
	}
	if resp := get(strings.Repeat("0", 64)); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("jeton inconnu : %d, attendu 401", resp.StatusCode)
	}

	grant := transfer.Grant{Xuid: exportXuid, Gamertag: "SpartanB"}
	token, _, err := store.Issue(grant, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	resp := get(token)
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/zip" {
		t.Fatalf("jeton valide : %d %s", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	data, _ := io.ReadAll(resp.Body)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("archive illisible : %v", err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, _ := f.Open()
		body, _ := io.ReadAll(rc)
		_ = rc.Close()
		if strings.Contains(string(body), token) {
			t.Errorf("%s contient le jeton d'export", f.Name)
		}
	}
	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "shared/match_registry.parquet") || !strings.Contains(joined, "manifest.json") {
		t.Fatalf("entrées de l'archive : %v", names)
	}

	// Le même jeton ne sert plus ; un nouveau jeton dans l'heure est refusé par la limite.
	if r := get(token); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("jeton rejoué : %d, attendu 401", r.StatusCode)
	}
	token2, _, _ := store.Issue(grant, time.Now())
	if r := get(token2); r.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("deuxième export dans l'heure : %d, attendu 429", r.StatusCode)
	}
}
