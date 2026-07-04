// Package archlint — no_raw_start_time_literal_test.go : ratchet H1 (règle
// CLAUDE.md n°8, timezone canonique).
//
// Interdit toute NOUVELLE occurrence du littéral SQL brut du timestamp de début
// de match canonique — `COALESCE(<alias>.start_time_utc, <alias>.start_time AT
// TIME ZONE 'UTC')`. La source unique est analysis.SQLStartTimeCanonical(alias)
// (délégué local platform/duckdb.StartTimeCanonicalSQL). Toute divergence de
// cette expression a causé des décalages de fuseau (DETTE first_joined_time).
//
// Le motif est PRÉCIS : il n'attrape ni `real_start_time AT TIME ZONE 'UTC'`
// (colonne distincte, calcul d'epoch/durée), ni la forme diagnostique d'offset
// `epoch_ms(r.start_time AT TIME ZONE 'UTC') - epoch_ms(r.start_time_utc)`
// (cmd/backfill_first_joined_tz, mesure volontairement le décalage), ni la
// définition elle-même (`"COALESCE(" + prefix + "start_time_utc, "...` — coupée
// par la concaténation). L'allowlist est donc VIDE.
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

// rawStartTimeAllowlist : VIDE. Les 97 sites du pattern canonique ont été migrés
// vers le helper (H1, 2026-07-04). migrations/ est sauté (historique gelé : la
// forme y est un CASE `THEN start_time AT TIME ZONE 'UTC'`, non-COALESCE, donc
// non-matchée de toute façon). Toute nouvelle occurrence fait échouer le test.
var rawStartTimeAllowlist = map[string]bool{}

// rawStartTimeRE matche la forme canonique COALESCE avec start_time_utc ET
// start_time. `[a-z_]*\.?` couvre un préfixe d'alias optionnel (r., mr., reg., nu).
var rawStartTimeRE = regexp.MustCompile(`COALESCE\([a-z_]*\.?start_time_utc,\s*[a-z_]*\.?start_time AT TIME ZONE 'UTC'\)`)

func TestNoNewRawStartTimeLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	// thisFile = .../internal/archlint/x_test.go → goAPIRoot = .../ (deux niveaux au-dessus d'internal).
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
				// migrations/ = DDL gelée (ADR 0026) : ne pas soumettre au ratchet.
				if d.Name() == "migrations" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			// La définition canonique du fragment (source unique) est exemptée.
			if rel == "internal/analysis/sql_fragments.go" || rawStartTimeAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if rawStartTimeRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("littéral SQL brut du start_time canonique interdit (H1, règle CLAUDE.md n°8) — "+
			"utiliser analysis.SQLStartTimeCanonical(alias) (ou le délégué local "+
			"duckdb.StartTimeCanonicalSQL) :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
