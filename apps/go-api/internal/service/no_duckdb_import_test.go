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
// Allowlist DÉCROISSANTE : sites de write-IO media (OpenReadWrite — ouverture de
// connexion d'écriture, hors chemin de lecture canonique). Le critère Phase 2 cible
// les services de LECTURE produit ; l'extraction d'un MediaRepository (port) est un
// refactor distinct. À vider quand ce port sera extrait.
var duckdbImportAllowlist = map[string]bool{
	"media_service.go":       true, // duckdbpkg.OpenReadWrite — write-IO media (atomique)
	"media_index_service.go": true, // idem — indexation media (write-IO)
}

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
