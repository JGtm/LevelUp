//go:build integration

package main

// boot_citations_reconcile_integration_test.go — la réconciliation des citations au démarrage :
// elle garnit metadata.duckdb, ne réécrit rien au démarrage suivant, et son échec n'empêche pas
// migrateMetadataDB de réussir (l'erreur est journalisée, le démarrage se poursuit).

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/ops"
	"levelup/go-api/internal/platform/duckdb"
)

// bootMetadataPath prépare un clone frais (dossier warehouse créé, registre de migrations posé) et
// rend le chemin de metadata.duckdb.
func bootMetadataPath(t *testing.T) string {
	t.Helper()
	pr := title.NewPathResolver(t.TempDir())
	if err := ensureWarehouseDir(pr, title.DefaultSlug); err != nil {
		t.Fatal(err)
	}
	if err := prepareMigrationRegistry(""); err != nil {
		t.Fatal(err)
	}
	return pr.MetadataDBPath(title.DefaultSlug)
}

func countCitationMappings(t *testing.T, metaPath string) int {
	t.Helper()
	db, err := duckdb.OpenReadWrite(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.SQLDb().QueryRow(`SELECT COUNT(*) FROM citation_mappings`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// spyReconcile remplace la réconciliation par un espion qui délègue à la vraie et garde les
// résultats de chaque démarrage.
func spyReconcile(t *testing.T) *[]ops.SeedResult {
	t.Helper()
	var seen []ops.SeedResult
	real := reconcileCitationMappings
	reconcileCitationMappings = func(ctx context.Context, db *sql.DB) (ops.SeedResult, error) {
		res, err := real(ctx, db)
		seen = append(seen, res)
		return res, err
	}
	t.Cleanup(func() { reconcileCitationMappings = real })
	return &seen
}

func TestMigrateMetadataDB_ReconcilieLesCitationsPuisNeReecritRien(t *testing.T) {
	metaPath := bootMetadataPath(t)
	seen := spyReconcile(t)

	for i := 0; i < 2; i++ {
		if err := migrateMetadataDB(metaPath, ""); err != nil {
			t.Fatalf("démarrage %d: %v", i+1, err)
		}
	}
	if len(*seen) != 2 {
		t.Fatalf("réconciliations = %d, want une par démarrage (2)", len(*seen))
	}
	first, second := (*seen)[0], (*seen)[1]
	if first.Inserted == 0 {
		t.Errorf("premier démarrage : Inserted = %d, want > 0", first.Inserted)
	}
	if second.Inserted != 0 || second.Updated != 0 {
		t.Errorf("démarrage suivant : Inserted=%d Updated=%d, want 0 et 0", second.Inserted, second.Updated)
	}
	if n := countCitationMappings(t, metaPath); n != first.Inserted {
		t.Errorf("citation_mappings = %d lignes, want %d", n, first.Inserted)
	}
}

func TestMigrateMetadataDB_EchecDeLaReconciliationDesCitationsNeBloquePas(t *testing.T) {
	logs := captureSlog(t)
	metaPath := bootMetadataPath(t)
	real := reconcileCitationMappings
	reconcileCitationMappings = func(context.Context, *sql.DB) (ops.SeedResult, error) {
		return ops.SeedResult{}, errors.New("textes des citations: langue manquante")
	}
	t.Cleanup(func() { reconcileCitationMappings = real })

	if err := migrateMetadataDB(metaPath, ""); err != nil {
		t.Fatalf("migrateMetadataDB doit réussir malgré l'échec de la réconciliation des citations: %v", err)
	}
	out := logs.String()
	if !strings.Contains(out, "level=ERROR") || !strings.Contains(out, "réconciliation au démarrage échouée") ||
		!strings.Contains(out, "langue manquante") {
		t.Errorf("l'échec doit être journalisé en ERROR avec sa cause, journal :\n%s", out)
	}
}
