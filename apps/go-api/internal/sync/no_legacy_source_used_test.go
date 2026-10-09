// Package sync — no_legacy_source_used_test.go : RATCHET ADR 0023 (gate D2).
//
// Historique : le post-sync achievements (resolveAccessTokenFromDB, supprimé)
// résolvait l'access_token EXCLUSIVEMENT depuis sync_meta et émettait lui-même
// legacy_source_used=duckdb_oauth à chaque cycle des 4 joueurs prod, sans jamais
// consulter le store watcher_tokens (incident 2026-07-12). Le fix a centralisé
// l'ordre de résolution dans auth.ResolveMSAccessTokenStoreFirst.
//
// Phase 5 (2026-08-25) : les sources legacy et la télémétrie associée ont été
// SUPPRIMÉES. Ce garde-rail devient l'invariant du package sync : aucun fichier
// de production ne doit ré-introduire une lecture de credential auth (clé
// sync_meta d'auth, env var SPNKR_OAUTH_REFRESH_TOKEN_*) ni une télémétrie de
// dépréciation locale. La résolution DOIT rester déléguée au helper canonique
// auth.ResolveMSAccessTokenStoreFirst (CLAUDE.md règle « ≤ 2 copies + un
// garde-rail à la factorisation »).
package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoLegacyAuthSourcesInSyncPackage(t *testing.T) {
	// Motifs interdits dans le package sync (non-test) : plus aucune lecture de
	// credential auth legacy, plus de télémétrie de dépréciation locale.
	forbidden := map[string]string{
		"RecordLegacySourceUsed":    "télémétrie de dépréciation supprimée en Phase 5",
		"legacy_source_used":        "littéral du WARN slog supprimé en Phase 5",
		"oauth_refresh_token":       "credential legacy sync_meta — lire MultiUserTokenStore",
		"msal_token_cache":          "credential legacy sync_meta — le cache MSAL n'existe plus",
		"SPNKR_OAUTH_REFRESH_TOKEN": "env var legacy — lire MultiUserTokenStore",
		"MSALCacheJSON":             "champ supprimé du store en Phase 5",
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
		for pat, why := range forbidden {
			if strings.Contains(content, pat) {
				t.Errorf("%s contient le motif interdit %q (%s) — la résolution d'access_token "+
					"doit déléguer à auth.ResolveMSAccessTokenStoreFirst (MultiUserTokenStore, "+
					"source unique ADR 0023 Phase 5). Cf. incident prod 2026-07-12.",
					name, pat, why)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("aucun fichier source scanné — le garde-rail ne protège rien")
	}
}
