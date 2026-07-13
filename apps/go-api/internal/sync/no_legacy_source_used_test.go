// Package sync — no_legacy_source_used_test.go : garde-rail anti-régression du
// câblage store-first de l'access_token achievements (ADR 0023 / gate D2).
//
// Contexte : le post-sync achievements (resolveAccessTokenFromDB, supprimé)
// résolvait l'access_token EXCLUSIVEMENT depuis sync_meta et émettait lui-même
// legacy_source_used=duckdb_oauth à chaque cycle des 4 joueurs prod, sans jamais
// consulter le store watcher_tokens (incident 2026-07-12). Le fix centralise
// l'ordre de résolution dans auth.ResolveMSAccessTokenStoreFirst (store d'abord),
// SEUL émetteur légitime de la télémétrie legacy (uniquement en vraie absence de
// RT store).
//
// Ce test interdit toute ré-introduction d'un émetteur de télémétrie legacy DANS
// le package sync : la résolution doit rester déléguée au helper canonique auth,
// sinon la divergence re-crée le faux positif (CLAUDE.md règle « ≤ 2 copies + un
// garde-rail à la factorisation »).
package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoLegacySourceTelemetryInSyncPackage(t *testing.T) {
	// Motifs interdits dans le package sync (non-test) : la télémétrie de
	// dépréciation legacy appartient au helper canonique auth, pas ici.
	forbidden := []string{
		"RecordLegacySourceUsed", // comptage expvar du gate D2
		"legacy_source_used",     // littéral du WARN slog
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("lecture du répertoire package: %v", err)
	}
	scanned := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Clean(name))
		if err != nil {
			t.Fatalf("lecture %s: %v", name, err)
		}
		scanned++
		content := string(data)
		for _, pat := range forbidden {
			if strings.Contains(content, pat) {
				t.Errorf("%s contient le motif interdit %q — la résolution d'access_token "+
					"doit déléguer à auth.ResolveMSAccessTokenStoreFirst (seul émetteur "+
					"légitime de legacy_source_used, store-first). Cf. incident prod 2026-07-12.",
					name, pat)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("aucun fichier source scanné — le garde-rail ne protège rien")
	}
}
