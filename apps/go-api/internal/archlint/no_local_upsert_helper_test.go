// Package archlint — no_local_upsert_helper_test.go : ratchet dédup #6 (K1d).
//
// Interdit toute nouvelle copie locale du helper générique d'upsert ART-safe
// « SELECT-d'existence puis UPDATE|INSERT paramétré » — signature canonique
// `existsQuery string, existsArgs []any, updateQuery ..., insertQuery ...`.
// La source unique est duckdb.UpsertRowNoConflict (+ la méthode *DB.UpsertNoConflict
// qui délègue). Ce pattern avait été copié-collé dans ops/catalog_refresh,
// service/catalog_fetcher_service et api/registry_catalog_expand (3e copie) —
// leçon CLAUDE.md règle 6 (centraliser + garde-rail).
//
// Exception allowlistée : service.CatalogFetcherService.upsertRowNoConflict garde
// sa copie locale car ADR 0025 D-MV2 INTERDIT à la couche service d'importer le
// package de données internal/platform/duckdb (verrou TestServicesDoNotImportDuckDB).
// Tant que ce service tient un *sql.DB brut plutôt qu'un port, il ne peut pas
// appeler la canonique — sa copie est donc architecturalement justifiée, pas de la
// dette de dédup. À supprimer le jour où un CatalogRepository (port) est extrait.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// genericUpsertSigRE matche la ligne de paramètres distinctive du helper générique
// (existsQuery + existsArgs). Les upserts bespoke (SQL inline spécifique, sans ce
// triplet de paramètres) ne matchent pas et restent hors périmètre.
var genericUpsertSigRE = regexp.MustCompile(`existsQuery\s+string,\s+existsArgs\s+\[\]any`)

// upsertHelperAllowed : chemins (relatifs à apps/go-api) autorisés à porter la
// signature du helper générique. duckdb/db.go = source canonique ;
// catalog_fetcher_service.go = copie forcée par le verrou D-MV2 (cf. entête).
var upsertHelperAllowed = map[string]bool{
	"internal/platform/duckdb/db.go":              true,
	"internal/service/catalog_fetcher_service.go": true,
}

func TestNoLocalGenericUpsertHelper(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	var violations []string
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if upsertHelperAllowed[rel] {
				return nil
			}
			for i, line := range strings.Split(string(data), "\n") {
				if genericUpsertSigRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("copie locale du helper d'upsert générique interdite (dédup #6 K1d) — "+
			"utiliser duckdb.UpsertRowNoConflict :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
