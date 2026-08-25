// Package auth — sentinel_test.go : guard-rail anti-régression ADR 0023.
//
// Ce test scanne les sources Go du repo pour détecter l'apparition de nouveaux
// chemins qui contournent MultiUserTokenStore (source unique post-ADR 0023) :
//
//  1. Toute LECTURE d'env var de refresh token legacy (Getenv / LookupEnv sur
//     le prefixe SPNKR_OAUTH_REFRESH_TOKEN). Depuis la Phase 5 (2026-08-25), UN SEUL fichier est
//     autorisé : la migration one-shot du boot (kill-switch daté).
//  2. Toute LECTURE de sync_meta.oauth_refresh_token via
//     duckdb.ReadOAuthRefreshToken — même exception unique.
//  3. cmd/token-capture et cmd/token-import qui écriraient à nouveau des
//     fichiers .txt (régression UX — l'ancien flux exigeait copy-paste manuel).
//
// Ces guards garantissent qu'aucun futur refactor ne réintroduit silencieusement
// le bug Madina (env.local burnt by Air hot-reload) ni un credential store
// parallèle. Chaque exception doit être listée dans l'allowlist avec
// justification datée.
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

// ─── Guard 1 : lecture d'env var de refresh token legacy ──────────────────

// envVarReadPattern détecte une LECTURE d'env var de refresh token legacy :
// un appel Getenv/LookupEnv dont le premier argument commence par le préfixe
// SPNKR_OAUTH_REFRESH_TOKEN (forme littérale comme forme concaténée
// `"…_" + key`). Les simples mentions en commentaire et les t.Setenv des tests
// ne matchent pas : le guard vise la dépendance runtime, pas le vocabulaire.
var envVarReadPattern = regexp.MustCompile(`(Getenv|LookupEnv)\(\s*"` + legacyEnvPrefix)

// legacyEnvPrefix est assemblé à l'exécution pour que CE fichier ne matche pas
// son propre motif (sinon le sentinel s'auto-allowliste, trou classique).
var legacyEnvPrefix = "SPNKR_OAUTH_REFRESH" + "_TOKEN"

// allowedEnvReaders : fichiers AUTORISÉS à lire SPNKR_OAUTH_REFRESH_TOKEN_*.
// ADR 0023 Phase 5 (2026-08-25) : l'allowlist est passée de ~30 entrées à UNE.
// Tout nouveau lecteur doit utiliser MultiUserTokenStore — aucune exception
// supplémentaire ne sera acceptée (la migration boot elle-même part le 2026-10-01).
var allowedEnvReaders = map[string]string{
	"internal/platform/auth/migration.go": "EXCEPTION UNIQUE ADR 0023 Phase 5 : EnvRefreshTokenForGamertag alimente la migration one-shot du boot (env legacy → store). Kill-switch daté — retrait cible 2026-10-01, critère « 0 token migré au boot sur 30 j de logs prod ».",
}

// TestSentinel_NoNewEnvVarReaders détecte tout nouveau site qui lit
// SPNKR_OAUTH_REFRESH_TOKEN_*. Toute nouvelle occurrence hors allowlist fail
// le test → le contributeur doit utiliser MultiUserTokenStore (source unique).
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
				" — utiliser MultiUserTokenStore au lieu de SPNKR_OAUTH_REFRESH_TOKEN_* (ADR 0023 Phase 5 : l'env var n'est plus une source de credentials)")
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

// ─── Guard 2 : duckdb.ReadOAuthRefreshToken (sync_meta legacy) ─────────────

// duckdbAuthReadPattern détecte les appels au DERNIER lecteur DuckDB du
// credential store legacy (sync_meta.oauth_refresh_token). Les écritures
// (WriteOAuthRefreshToken) et les lectures MSAL n'existent plus depuis la
// Phase 5 : leur simple réapparition compilerait sur une fonction absente.
var duckdbAuthReadPattern = regexp.MustCompile(`\bduckdb(pkg)?\.ReadOAuthRefreshToken\b`)

// allowedDuckDBAuthReaders : sites AUTORISÉS à lire sync_meta.oauth_refresh_token.
// ADR 0023 Phase 5 (2026-08-25) : uniquement la définition et la migration
// one-shot du boot (kill-switch daté, retrait cible 2026-10-01).
var allowedDuckDBAuthReaders = map[string]string{
	"internal/platform/duckdb/queries_auth.go": "Définition de la fonction (dernier lecteur legacy, supprimé avec la migration boot le 2026-10-01).",
	"cmd/server/main.go":                       "EXCEPTION UNIQUE : legacyAuthSourcesReader de migrateLegacyAuthTokensAtBoot (migration one-shot env+sync_meta → store).",
	// Tests
	"internal/platform/duckdb/queries_auth_test.go": "Test du dernier lecteur legacy.",
	"internal/platform/auth/sentinel_test.go":       "Ce fichier — contient les patterns à détecter.",
}

func TestSentinel_NoNewDuckDBAuthReaders(t *testing.T) {
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
		if !duckdbAuthReadPattern.Match(content) {
			return nil
		}
		if _, allowed := allowedDuckDBAuthReaders[rel]; allowed {
			return nil
		}
		violations = append(violations,
			"NEW duckdb.ReadOAuthRefreshToken call: "+rel+
				" — lire MultiUserTokenStore (ADR 0023 Phase 5 : sync_meta n'est plus un credential store).")
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("REGRESSION ADR 0023 : %d nouveaux lecteurs DuckDB legacy détectés :\n  - %s",
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

// ─── Self-check anti-pourrissement (V4d, VF-6) ──────────────────────────────

// TestSentinel_AllowlistEntriesPointToExistingFiles vérifie que chaque entrée
// des allowlists du sentinel pointe un FICHIER EXISTANT. Une entrée dont le
// fichier a disparu (déplacé/supprimé par un refactor) est un trou latent : si
// un fichier est un jour recréé à ce chemin, il contournerait le sentinel sans
// alerte. C'est exactement le défaut révélé par VF-6 (internal/api/registry.go
// supprimé au lot K mais toujours allowlisté). Les clés de ces maps sont TOUTES
// des chemins de fichiers (pas des motifs) → self-check d'existence direct.
func TestSentinel_AllowlistEntriesPointToExistingFiles(t *testing.T) {
	repoRoot := findRepoRootForSentinel(t)
	apiRoot := filepath.Join(repoRoot, "apps", "go-api")

	allowlists := map[string]map[string]string{
		"allowedEnvReaders":          allowedEnvReaders,
		"allowedDuckDBAuthReaders":   allowedDuckDBAuthReaders,
		"allowedClientSecretReaders": allowedClientSecretReaders,
	}
	for name, allowlist := range allowlists {
		for rel := range allowlist {
			path := filepath.Join(apiRoot, filepath.FromSlash(rel))
			if _, err := os.Stat(path); err != nil {
				t.Errorf("%s : entrée %q pointe un fichier inexistant (%v) — refactor de renommage/suppression ?"+
					" Retirer l'entrée morte (trou latent : un fichier recréé à ce chemin contournerait le sentinel).",
					name, rel, err)
			}
		}
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
