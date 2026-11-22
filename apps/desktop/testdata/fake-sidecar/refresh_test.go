package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRefreshOnly_RendLeCodeDuFichierEtNote(t *testing.T) {
	data := t.TempDir()
	env := map[string]string{"LEVELUP_DATA_DIR": data, "LEVELUP_RESOURCES_DIR": "R", "LEVELUP_LOGS_DIR": "L"}
	getenv := func(k string) string { return env[k] }
	args := []string{"-mode", "refresh-only"}

	if code := run(args, getenv, "C", nil, nil, &bytes.Buffer{}); code != 0 {
		t.Fatalf("sans fichier de code : %d, attendu 0", code)
	}
	if err := os.WriteFile(filepath.Join(data, refreshExitFile), []byte("7\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := run(args, getenv, "C", nil, nil, &bytes.Buffer{}); code != 7 {
		t.Fatalf("code = %d, attendu 7", code)
	}
	ran, err := os.ReadFile(filepath.Join(data, refreshRanFile))
	if err != nil || !strings.Contains(string(ran), "cwd=C resources=R") || !strings.Contains(string(ran), "secret_set=false") {
		t.Fatalf("temoin = %q, %v", ran, err)
	}
}

func TestRefreshOnly_AutreModeNEstPasLui(t *testing.T) {
	if _, ok := refreshOnly([]string{"-mode", "desktop", "-arret-sur-stdin"}, func(string) string { return "" }, "", &bytes.Buffer{}); ok {
		t.Fatal("le mode desktop n'est pas refresh-only")
	}
}
