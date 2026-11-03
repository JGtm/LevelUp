//go:build cgo

package diagnostic

import (
	"archive/zip"
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
)

// Faux secrets construits à l'exécution : aucun littéral de jeton dans le dépôt.
var (
	fakeRefresh  = "M.C5" + "_" + strings.Repeat("Ab1Cd2Ef3", 6)
	fakeBearer   = strings.Repeat("Zy9Xw8", 6)
	fakeWebhook  = "https://discord.example/api/webhooks/" + strings.Repeat("7", 12) + "/" + strings.Repeat("k", 20)
	fakeLauncher = "launcher-" + strings.Repeat("s3cr", 8)
	fakeCode     = strings.Repeat("c0de", 6)
	fakeJWT      = "eyJ" + strings.Repeat("a", 12) + "." + strings.Repeat("b", 10) + "." + strings.Repeat("c", 10)
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// readZip rend le contenu de chaque entrée de l'archive.
func readZip(t *testing.T, path string) map[string]string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("archive illisible : %v", err)
	}
	defer func() { _ = zr.Close() }()
	out := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = string(b)
	}
	return out
}

// TestWrite_NeLaisseSortirAucunSecret est le test sentinelle du rapport : un faux jeton, un faux secret de
// lanceur, un faux webhook et un faux code d'autorisation sont semés dans les journaux, les réglages et les
// documents joints ; AUCUNE entrée de l'archive ne doit en contenir une trace.
func TestWrite_NeLaisseSortirAucunSecret(t *testing.T) {
	root := t.TempDir()
	pr := title.NewPathResolverRoots(title.Roots{Resources: root, Data: filepath.Join(root, "data"),
		Logs: filepath.Join(root, "logs"), DataSet: true})
	logs := pr.LogsRoot()
	writeFile(t, filepath.Join(logs, "server.log"), strings.Join([]string{
		`2026-11-03 INFO sync: ok status_code=200 duration_ms=12`,
		`2026-11-03 WARN auth: refresh failed refresh_token=` + fakeRefresh + ` xuid=2533274800000001`,
		`2026-11-03 INFO http: GET /desktop/enter?secret=` + fakeLauncher + ` 200`,
		`2026-11-03 INFO http: Authorization: Bearer ` + fakeBearer,
		`2026-11-03 INFO auth: callback ?code=` + fakeCode + `&state=zzz`,
		`2026-11-03 ERROR auth: {"access_token":"` + fakeJWT + `","note":"x"}`,
		`2026-11-03 INFO env: launcher=` + fakeLauncher,
	}, "\n")+"\n")
	writeFile(t, filepath.Join(logs, "desktop.log"), "shell: updater idle\n")
	writeFile(t, filepath.Join(logs, "notes.txt"), "ne doit pas être joint")
	settings := filepath.Join(root, "app_settings.json")
	writeFile(t, settings, `{"lang":"fr","discord_webhook_url":"`+fakeWebhook+`","extra":{"nested_token":"`+fakeRefresh+`","keep":"visible"},"backup_keep":7}`)

	// Une base joueur, pour exercer la porte de lecture.
	dbPath := pr.PlayerDBPath("halo_infinite", "SpartanB")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE player_match_enrichment (match_id VARCHAR, performance_score DOUBLE)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	path, err := Write(context.Background(), Input{
		Paths: pr, LogsRoot: logs, AppVersion: "9.9.9-test", Deployment: "desktop",
		Capabilities: map[string]bool{"ParquetBackup": true}, SettingsFile: settings,
		Extras: map[string]any{
			"scheduled_tasks": map[string]any{"state": "present", "refresh_token": fakeRefresh},
			"updater":         map[string]any{"version": "9.9.9-test", "endpoint_code": "x"},
		},
		Secrets: []string{fakeLauncher},
		Now:     func() time.Time { return time.Date(2026, 11, 3, 9, 30, 0, 0, time.Local) },
	})
	if err != nil {
		t.Fatalf("Write : %v", err)
	}
	if want := filepath.Join(pr.DataRoot(), "diagnostics", "levelup-diag-20261103-0930.zip"); path != want {
		t.Fatalf("chemin = %s, attendu %s", path, want)
	}
	entries := readZip(t, path)

	for name, content := range entries {
		for label, secret := range map[string]string{"jeton de rafraîchissement": fakeRefresh, "bearer": fakeBearer,
			"webhook": fakeWebhook, "secret du lanceur": fakeLauncher, "code d'autorisation": fakeCode, "JWT": fakeJWT} {
			if strings.Contains(content, secret) {
				t.Errorf("%s : contient un %s", name, label)
			}
		}
	}

	// Ce qui sert au diagnostic reste lisible.
	if !strings.Contains(entries["logs/server.log"], "status_code=200") || !strings.Contains(entries["logs/server.log"], "duration_ms=12") {
		t.Errorf("le journal expurgé a perdu des informations non sensibles :\n%s", entries["logs/server.log"])
	}
	for _, name := range []string{"app_info.json", "settings.json", "scheduled_tasks.json", "updater.json", "logs/desktop.log",
		"databases/halo_infinite_player_SpartanB.txt"} {
		if _, ok := entries[name]; !ok {
			t.Errorf("pièce absente de l'archive : %s", name)
		}
	}
	if _, ok := entries["logs/notes.txt"]; ok {
		t.Error("seuls les fichiers .log sont joints")
	}
	if !strings.Contains(entries["app_info.json"], "9.9.9-test") || !strings.Contains(entries["settings.json"], `"keep": "visible"`) {
		t.Errorf("version ou réglage non sensible absents :\n%s\n%s", entries["app_info.json"], entries["settings.json"])
	}
	if !strings.Contains(entries["databases/halo_infinite_player_SpartanB.txt"], "player_match_enrichment") {
		t.Errorf("diagnostic de base sans table :\n%s", entries["databases/halo_infinite_player_SpartanB.txt"])
	}
}

func TestWrite_JournalTronqueAuxDerniersOctets(t *testing.T) {
	root := t.TempDir()
	pr := title.NewPathResolverRoots(title.Roots{Resources: root, Data: filepath.Join(root, "data"),
		Logs: filepath.Join(root, "logs"), DataSet: true})
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, "ligne numero "+strings.Repeat("x", 40))
	}
	lines = append(lines, "DERNIERE")
	writeFile(t, filepath.Join(pr.LogsRoot(), "server.log"), strings.Join(lines, "\n")+"\n")
	path, err := Write(context.Background(), Input{Paths: pr, LogsRoot: pr.LogsRoot(), MaxLogBytes: 300})
	if err != nil {
		t.Fatal(err)
	}
	got := readZip(t, path)["logs/server.log"]
	if len(got) > 300 || !strings.HasSuffix(strings.TrimSpace(got), "DERNIERE") || !strings.HasPrefix(got, "ligne numero") {
		t.Fatalf("la fin du journal, à ligne entière, bornée à 300 octets, était attendue : %d octets\n%q", len(got), got)
	}
}
