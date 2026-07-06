// Package auth — sentinel_test.go : guard-rail anti-régression ADR 0023.
//
// Ce test scanne les sources Go du repo pour détecter l'apparition de nouveaux
// chemins qui contournent MultiUserTokenStore (source unique post-ADR 0023) :
//
//  1. Nouvelles lectures de SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> via os.Getenv
//     hors des sites legacy explicitement deprecated.
//  2. Nouveaux appels à duckdb.WriteOAuthRefreshToken hors des sites de compat
//     transitoire (Phase 5 supprimera ces sites).
//  3. cmd/token-capture et cmd/token-import qui écriraient à nouveau des
//     fichiers .txt (régression UX — l'ancien flux exigeait copy-paste manuel).
//
// Ces guards garantissent qu'aucun futur refactor ne réintroduit silencieusement
// le bug Madina (env.local burnt by Air hot-reload). Chaque exception doit
// être listée dans l'allowlist avec justification.
//
// Sans build tag — exécuté en CI normale (juste grep sur sources).
package auth

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ─── Guard 1 : os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_*") ────────────────────

// envVarReadPattern détecte toute lecture des SPNKR_OAUTH_REFRESH_TOKEN_*.
// Capture os.Getenv("SPNKR_OAUTH_REFRESH_TOKEN_..."), os.Getenv(`...`), ou
// la concaténation "SPNKR_OAUTH_REFRESH_TOKEN_" + ... dans un Getenv.
var envVarReadPattern = regexp.MustCompile(`(?i)\bSPNKR_OAUTH_REFRESH_TOKEN_`)

// allowedEnvReaders : fichiers AUTORISÉS à lire SPNKR_OAUTH_REFRESH_TOKEN_*.
// Baseline post-ADR 0023 : tous les sites existants au moment de la mise en
// place du sentinel. Toute nouvelle entrée ajoutée à ce map doit avoir une
// justification. La règle pour les contributeurs : utiliser
// MultiUserTokenStore (canonique) pour tout nouveau code. L'allowlist ci-dessous
// est gelée — Phase 5 supprimera progressivement les sites legacy.
var allowedEnvReaders = map[string]string{
	// === Core auth (Phases 2/3 du refactor ADR 0023) ===
	"internal/platform/auth/migration.go":                       "Phase 2 : EnvRefreshTokenForGamertag migre les env vars legacy au store au boot. Retiré Phase 5.",
	"internal/platform/auth/migration_test.go":                  "Test du migrateur (t.Setenv).",
	"internal/platform/auth/watcher_refresh.go":                 "Phase 3c fallback DEPRECATED : RefreshTokenFromEnv. Retiré Phase 5.",
	"internal/platform/auth/watcher_refresh_test.go":            "Test watcher (t.Setenv).",
	"internal/platform/auth/pool/discovery.go":                  "Phase 3b fallback DEPRECATED : readOAuthRefreshTokenFromEnv. Retiré Phase 5.",
	"internal/platform/auth/pool/discovery_test.go":             "Test discovery (t.Setenv).",
	"internal/platform/auth/pool/discovery_watcher_test.go":     "Test isolation env var.",
	"internal/api/registry.go":                                  "Phase 3a fallback DEPRECATED : oauthRefreshTokenForPlayer (avant split). Retiré Phase 5.",
	"internal/api/registry_auth.go":                             "Phase 3a fallback DEPRECATED post god-file split : refreshTokensFromDB + oauthRefreshTokenForPlayer. Retiré Phase 5.",
	"internal/platform/auth/oauth_refresh.go":                   "Module OAuth bas-niveau : lit SPNKR_AZURE_* (pas SPNKR_OAUTH_REFRESH_TOKEN_) — string mention dans le module canonique OAuth.",
	"internal/platform/auth/capturecli/capturecli.go":           "ParseRefreshTokenStdin détecte le format env-var-line pour extraire le RT (string match, pas os.Getenv).",
	"internal/platform/auth/capturecli/capturecli_test.go":      "Tests du parser — strings 'SPNKR_OAUTH_REFRESH_TOKEN_X=value' utilisées comme fixtures.",
	"internal/platform/auth/sentinel_test.go":                   "Ce fichier — contient les patterns à détecter.",
	"internal/platform/auth/watcher_refresh_multistore_test.go": "Tests T5 — t.Setenv pour vérifier le fallback env var fonctionne.",
	"internal/platform/auth/pool/discovery_priority_test.go":    "Tests T3b — t.Setenv pour vérifier le fallback env var dans Discovery.",
	"cmd/server/migration_boot_test.go":                         "Tests T6 — t.Setenv pour vérifier la migration env→store.",
	"tests/e2e/air_restart_cycle_test.go":                       "Test T8 pivot — t.Setenv pour le scénario régression Madina.",
	"internal/sync/engine_postsync_csr.go":                      "Fallback legacy sync CSR : lit SPNKR_OAUTH_REFRESH_TOKEN_<GAMERTAG> comme source de dernier recours (après sync_meta + MSALCache). Toléré ADR 0023 §legacy jusqu'à Phase 5.",

	// === Config / Server boot ===
	"internal/config/config.go":              "Mention dans commentaire sur le chargement de .env.local (legacy, retiré Phase 5).",
	"internal/config/config_helpers_test.go": "Test du chargement env (t.Setenv).",
	"cmd/server/main.go":                     "Wiring resolveXUIDForRotation + appel migrateLegacyAuthTokensAtBoot (Phase 2). Mention dans onRotated log.",
	"internal/scheduler/auto_sync.go":        "Header doc qui mentionne le retrait de defaultTokenReader (referenced for historical context).",
	"internal/scheduler/auto_sync_run.go":    "Message d'aide (K2c : extrait de auto_sync.go) citant SPNKR_OAUTH_REFRESH_TOKEN_<GT> dans checkSyncPreconditions — libellé pour l'utilisateur, PAS une lecture d'env.",

	// === Sync engine ===
	"internal/sync/engine_postsync.go": "Mention dans commentaire/log (legacy).",

	// === CLI tools (Phase 4bis migrated via cli_refresh helper, mais lisent encore env comme fallback) ===
	"cmd/token-capture/main.go":               "Référence dans help/log message (pas de Getenv direct — délégué à capturecli).",
	"cmd/refresh-metadata/main.go":            "CLI standalone : env var lue comme LegacyAuthInputs.OAuthRT (Phase 4bis).",
	"cmd/refresh-career-ranks/main.go":        "Référence dans log message (CLI utilise RefreshHaloTokensViaStoreFirst).",
	"cmd/populate-career-rank-images/main.go": "envRefreshTokenForGamertag fournit LegacyAuthInputs au helper canonique.",
	"cmd/diag_emblem_colors/main.go":          "acquireDiagTokens : env var comme LegacyAuthInputs au helper canonique.",

	// === CLI diagnostic / one-shot (legacy, hors scope ADR 0023) ===
	"cmd/diag_backfill_dryrun/main.go":        "Diagnostic one-shot, pas dans le hot path.",
	"cmd/diag_emblem_mapping/main.go":         "Diagnostic one-shot.",
	"cmd/diag_film/main.go":                   "Diagnostic one-shot.",
	"cmd/get-token/main.go":                   "Tool one-shot pour extraire un access_token pour debugging.",
	"cmd/levelup/cmd_sync.go":                 "CLI sync manuel — legacy path, à migrer ultérieurement.",
	"cmd/populate-playlists-catalog/main.go":  "CLI seed one-shot.",
	"cmd/refresh_golden_fixture/main.go":      "Tool one-shot pour refresh fixtures de test.",
	"scripts/warm_bp_assets/main.go":          "Script ops one-shot, pas dans le hot path.",
	"cmd/backfill_all/main.go":                "CLI backfill standalone — legacy auth path à migrer (cf. Phase 4bis non couverte par helper canonique).",
	"cmd/backfill_participation_info/main.go": "CLI one-shot Phase 0 LUSR v2 — backfill participation_info, legacy auth path.",
	"cmd/backfill_quit_timestamps/main.go":    "CLI one-shot Phase 3-quit LUSR v2 — backfill FirstJoinedTime/LastLeaveTime, même pattern que backfill_participation_info.",
	"cmd/bench-rps/main.go":                   "Bench tool one-shot pour RPS rate limit.",
}

// TestSentinel_NoNewEnvVarReaders détecte tout nouveau site qui lit
// SPNKR_OAUTH_REFRESH_TOKEN_*. Toute nouvelle occurrence hors allowlist fail
// le test → le contributeur doit soit utiliser MultiUserTokenStore (canonique)
// soit ajouter une justification dans `allowedEnvReaders`.
func TestSentinel_NoNewEnvVarReaders(t *testing.T) {
	repoRoot := findRepoRootForSentinel(t)
	apiRoot := filepath.Join(repoRoot, "apps", "go-api")

	var violations []string
	err := filepath.Walk(apiRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if !envVarReadPattern.Match(content) {
			return nil
		}
		if _, allowed := allowedEnvReaders[rel]; allowed {
			return nil
		}
		violations = append(violations,
			"NEW env var reader detected: "+rel+
				" — utiliser MultiUserTokenStore au lieu de SPNKR_OAUTH_REFRESH_TOKEN_*, ou ajouter à allowedEnvReaders avec justification ADR 0023")
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("REGRESSION ADR 0023 : %d nouveaux lecteurs d'env var détectés :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// ─── Guard 2 : duckdb.WriteOAuthRefreshToken ──────────────────────────────

// duckdbWritePattern détecte les appels à duckdb.WriteOAuthRefreshToken.
var duckdbWritePattern = regexp.MustCompile(`\bduckdb\.WriteOAuthRefreshToken\b`)

// allowedDuckDBWriters : sites AUTORISÉS à écrire dans sync_meta.oauth_refresh_token.
// Tous transitoires (Phase 5 supprimera ces écritures au profit du store unique).
var allowedDuckDBWriters = map[string]string{
	"internal/platform/duckdb/queries_auth.go": "Définition de la fonction. Sera supprimée Phase 6.",
	"internal/api/registry.go":                 "ADR 0023 compat transitoire (pré-split) : tryRefreshFromLegacy persiste aussi en DuckDB. Retiré Phase 5.",
	"internal/api/registry_auth.go":            "ADR 0023 compat transitoire post-split : tryRefreshFromLegacy persiste aussi en DuckDB. Retiré Phase 5.",
	"internal/api/handlers/admin_auto_sync.go": "ADR 0023 compat transitoire : probe onRotated écrit double (store + DuckDB). Retiré Phase 5.",
	"cmd/server/main.go":                       "ADR 0023 compat transitoire : autoSyncPool onRotated double-write store + DuckDB. Retiré Phase 5.",
	"internal/scheduler/auto_sync_e2e_test.go": "Test E2E historique qui setup sync_meta directement — sera adapté Phase 5.",
	// Tests anti-régression
	"internal/platform/auth/sentinel_test.go": "Ce fichier — contient les patterns à détecter.",
}

func TestSentinel_NoNewDuckDBTokenWriters(t *testing.T) {
	repoRoot := findRepoRootForSentinel(t)
	apiRoot := filepath.Join(repoRoot, "apps", "go-api")

	var violations []string
	err := filepath.Walk(apiRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if !duckdbWritePattern.Match(content) {
			return nil
		}
		if _, allowed := allowedDuckDBWriters[rel]; allowed {
			return nil
		}
		violations = append(violations,
			"NEW duckdb.WriteOAuthRefreshToken call: "+rel+
				" — utiliser MultiUserTokenStore.UpdateOAuthRefreshToken (ADR 0023). Si compat nécessaire, ajouter à allowedDuckDBWriters.")
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("REGRESSION ADR 0023 : %d nouveaux writers DuckDB détectés :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// ─── Guard 3 : token-capture / token-import ne créent pas de fichiers .txt ────

// fileWritePattern détecte os.WriteFile/ioutil.WriteFile/os.Create avec .txt.
var txtFilePattern = regexp.MustCompile(`(?i)(os\.WriteFile|os\.Create|ioutil\.WriteFile)[^)]*\.txt`)

func TestSentinel_TokenCaptureNoTxtFile(t *testing.T) {
	repoRoot := findRepoRootForSentinel(t)
	files := []string{
		"apps/go-api/cmd/token-capture/main.go",
		"apps/go-api/cmd/token-import/main.go",
	}
	for _, rel := range files {
		path := filepath.Join(repoRoot, rel)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("read %s: %v", rel, err)
			continue
		}
		if txtFilePattern.Match(content) {
			t.Errorf("REGRESSION UX %s : écriture de fichier .txt détectée — token-capture/import doivent écrire UNIQUEMENT dans MultiUserTokenStore (ADR 0023). L'ancien comportement (txt file à copier dans .env.local) est interdit.",
				rel)
		}
	}
}

// ─── Guard 4 : pas d'appel direct à os.Getenv sur SPNKR_AZURE_CLIENT_SECRET hors paths attendus ────

// Anti-pattern : si quelqu'un introduit un Getenv("SPNKR_AZURE_CLIENT_SECRET") dans
// un chemin de prod (hors oauth_refresh.go qui en a besoin pour Microsoft auth),
// c'est probablement un bug ou une fuite de secret.
var clientSecretPattern = regexp.MustCompile(`SPNKR_AZURE_CLIENT_SECRET`)

var allowedClientSecretReaders = map[string]string{
	"internal/platform/auth/azure_credentials.go":           "SOURCE UNIQUE des credentials Azure (seam, Phase 1) : seul lecteur prod de SPNKR_AZURE_CLIENT_SECRET. auth_code.go / oauth_refresh.go délèguent désormais à ResolveAzureOAuthClient.",
	"internal/platform/auth/azure_credentials_test.go":      "Golden tests du seam : t.Setenv pour caractériser la résolution client/secret (public vs confidentiel).",
	"internal/platform/auth/oauth_refresh_internal_test.go": "Tests du module canonique OAuth : t.Setenv pour exercer le retry public AADSTS90023 (plan anti-bruit 2026-06-11).",
	"internal/platform/auth/sentinel_test.go":               "Ce fichier.",
}

func TestSentinel_NoNewClientSecretReaders(t *testing.T) {
	repoRoot := findRepoRootForSentinel(t)
	apiRoot := filepath.Join(repoRoot, "apps", "go-api")

	var violations []string
	_ = filepath.Walk(apiRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" || name == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if !clientSecretPattern.Match(content) {
			return nil
		}
		if _, allowed := allowedClientSecretReaders[rel]; allowed {
			return nil
		}
		violations = append(violations, rel)
		return nil
	})
	if len(violations) > 0 {
		t.Errorf("NEW SPNKR_AZURE_CLIENT_SECRET reader detected — vérifier que ce n'est pas une fuite : %v", violations)
	}
}

// ─── Helper ───────────────────────────────────────────────────────────────

// findRepoRootForSentinel remonte les répertoires depuis cwd jusqu'à trouver
// db_profiles.example.json (marqueur racine repo).
func findRepoRootForSentinel(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "db_profiles.example.json")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("repo root introuvable (db_profiles.example.json absent dans les 10 niveaux parents)")
	return ""
}
