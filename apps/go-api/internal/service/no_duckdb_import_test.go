package service_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestServicesDoNotImportDuckDB verrouille le critère de complétion Phase 2
// (ADR 0025 D-MV2) : aucun service ne dépend du package de données
// internal/platform/duckdb — la couche service consomme des types canoniques /
// domaine via des ports. Toute régression (un service réintroduisant l'import du
// package duckdb) fait échouer ce test.
//
// Portée : l'import EXACT du package de données `internal/platform/duckdb`. Les
// sous-packages (ex. `internal/platform/duckdb/sharedprovider`, importé par
// openspartan_import_service) ne sont PAS le package de données et ne comptent pas.
//
// Allowlist VIDE (K1m, 2026-07-06). Plus aucun service n'importe le package data
// `internal/platform/duckdb` :
//   - media_index_service : le seul importeur (resetPlayerMediaIndex) était un NO-OP
//     depuis drop_media_from_player_db (media en shared_social append-only) — supprimé.
//   - media_service : n'importait déjà PLUS le package data (entrée d'allowlist périmée).
//
// Toute réintroduction d'un import `internal/platform/duckdb` dans un service = régression
// D-MV2 (ADR 0025) → ce test échoue. Ne PAS ré-agrandir sans justification datée.
var duckdbImportAllowlist = map[string]bool{}

func TestServicesDoNotImportDuckDB(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	serviceDir := filepath.Dir(thisFile)

	const dataPkgImport = `"levelup/go-api/internal/platform/duckdb"`

	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		t.Fatalf("read service dir: %v", err)
	}
	var violations []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if duckdbImportAllowlist[name] {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(serviceDir, name))
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		if strings.Contains(string(data), dataPkgImport) {
			violations = append(violations, name)
		}
	}
	if len(violations) > 0 {
		t.Errorf("services important le package de données internal/platform/duckdb (interdit — "+
			"Phase 2 D-MV2 : dépendre d'un port + types canoniques) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
