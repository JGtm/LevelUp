// Package auth — sentinel_test.go : guard-rail anti-régression ADR 0023.
//
// Ce test scanne les sources Go de PRODUCTION (hors _test.go) pour détecter
// l'apparition de chemins qui contournent MultiUserTokenStore, source unique
// post-ADR 0023 :
//
//  1. Toute occurrence du littéral de l'env var de refresh token legacy —
//     lecture, concaténation, format, ou simple mention. Détection LARGE
//     assumée : c'est ce qui faisait la force du guard d'origine ; un motif
//     restreint à `Getenv("PREFIX` laissait passer `os.Getenv(prefix + key)`
//     et `fmt.Sprintf` (trou relevé par la revue adversariale r1).
//  2. Tout appel à auth.EnvRefreshTokenForGamertag — la fonction reste exportée
//     pour la migration boot ; hors d'elle, l'appeler revient à ressusciter la
//     source env SANS jamais écrire le littéral (donc invisible au guard 1).
//  3. Tout appel à duckdb.ReadOAuthRefreshToken — dernier lecteur du credential
//     store DuckDB. Motif INDÉPENDANT de l'alias d'import : le package est
//     importé sous 5 noms différents dans le repo (duckdb, duckdbpkg, ddb,
//     duckdbPlatform, platform_duckdb) ; un motif `\bduckdb(pkg)?\.` en ratait 3.
//  4. cmd/token-capture et cmd/token-import qui écriraient à nouveau des
//     fichiers .txt (régression UX — l'ancien flux exigeait copy-paste manuel).
//
// Ces guards garantissent qu'aucun futur refactor ne réintroduit silencieusement
// le bug Madina (env.local brûlé par le hot-reload Air) ni un credential store
// parallèle. Chaque exception est listée avec justification DATÉE.
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

// legacyEnvPrefix est assemblé à l'exécution pour que CE fichier ne matche pas
// son propre motif (sinon le sentinel s'auto-allowliste, trou classique). Les
// autres fichiers _test.go sont hors périmètre : le guard vise le code livré.
var legacyEnvPrefix = "SPNKR_OAUTH_REFRESH" + "_TOKEN"

// scanProductionGoFiles applique `match` à chaque .go de production sous
// apps/go-api (hors _test.go, vendor, tmp) et retourne les chemins relatifs
// qui matchent sans être allowlistés. Factorisé : les 3 guards de littéral
// partagent exactement cette mécanique (CLAUDE.md règle « ≤ 2 copies »).
func scanProductionGoFiles(t *testing.T, allowlist map[string]string, match func(content []byte) bool) []string {
	t.Helper()
	repoRoot := findRepoRootForSentinel(t)
	apiRoot := filepath.Join(repoRoot, "apps", "go-api")

	var hits []string
	scanned := 0
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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		scanned++
		if !match(content) {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)
		if _, allowed := allowlist[rel]; allowed {
			return nil
		}
		hits = append(hits, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	// Anti-pourrissement : un walk qui ne voit plus aucun fichier rendrait tous
	// les guards verts pour de mauvaises raisons.
	if scanned == 0 {
		t.Fatal("aucun fichier de production scanné — les guards ne protègent rien")
	}
	return hits
}

// ─── Guard 1 : littéral de l'env var de refresh token legacy ──────────────

// allowedEnvReaders : fichiers de PRODUCTION autorisés à mentionner
// SPNKR_OAUTH_REFRESH_TOKEN_*. ADR 0023 Phase 5 (2026-08-25) : l'allowlist est
// passée de ~30 entrées à 3, toutes justifiées et datées. Tout nouveau code doit
// lire MultiUserTokenStore — aucune exception supplémentaire ne sera acceptée.
//
// Mordant (mutation mentale) : réintroduire `os.Getenv("SPNKR_OAUTH_..." + key)`
// dans n'importe quel fichier hors de cette liste — y compris via une
// concaténation, un fmt.Sprintf ou une constante intermédiaire — fait échouer
// ce test, puisqu'on cherche le LITTÉRAL et non une forme d'appel.
var allowedEnvReaders = map[string]string{
	"internal/platform/auth/migration.go":             "EXCEPTION UNIQUE ADR 0023 Phase 5 : EnvRefreshTokenForGamertag alimente la migration one-shot du boot (env legacy → store). Kill-switch daté — retrait cible 2026-10-01, critère « 0 token migré au boot sur 30 j de logs prod ».",
	"cmd/server/main.go":                              "Wiring + godoc de migrateLegacyAuthTokensAtBoot (même kill-switch daté 2026-10-01). Ne LIT pas l'env var lui-même : il délègue à auth.EnvRefreshTokenForGamertag.",
	"internal/platform/auth/capturecli/capturecli.go": "ParseRefreshTokenStdin accepte une ligne au format `SPNKR_OAUTH_REFRESH_TOKEN_X=valeur` collée par l'utilisateur (ergonomie de cmd/token-import). String match sur stdin, JAMAIS une lecture d'environnement.",
}

// TestSentinel_NoNewEnvVarReaders détecte tout fichier de production qui
// mentionne le littéral de l'env var legacy hors allowlist datée.
func TestSentinel_NoNewEnvVarReaders(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(legacyEnvPrefix))
	violations := scanProductionGoFiles(t, allowedEnvReaders, pattern.Match)

	if len(violations) > 0 {
		t.Errorf("REGRESSION ADR 0023 Phase 5 : %d fichier(s) de production mentionnent l'env var legacy "+
			"de refresh token hors allowlist — utiliser MultiUserTokenStore (source unique) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// ─── Guard 2 : auth.EnvRefreshTokenForGamertag ────────────────────────────

// allowedEnvHelperCallers : appelants autorisés du helper exporté qui lit
// l'env var. Il survit UNIQUEMENT pour la migration boot (kill-switch daté
// 2026-10-01) ; tout autre appelant recréerait la source legacy sans jamais
// écrire le littéral, donc sans déclencher le guard 1.
//
// Mordant : ajouter `auth.EnvRefreshTokenForGamertag(gt)` dans un CLI ou un
// service fait échouer ce test même si le fichier ne contient aucun littéral.
var allowedEnvHelperCallers = map[string]string{
	"internal/platform/auth/migration.go": "Définition + usage par la migration one-shot du boot (retrait 2026-10-01).",
	"cmd/server/main.go":                  "legacyAuthSourcesReader de migrateLegacyAuthTokensAtBoot — seul appelant légitime.",
}

func TestSentinel_NoNewEnvHelperCallers(t *testing.T) {
	pattern := regexp.MustCompile(`\bEnvRefreshTokenForGamertag\(`)
	violations := scanProductionGoFiles(t, allowedEnvHelperCallers, pattern.Match)

	if len(violations) > 0 {
		t.Errorf("REGRESSION ADR 0023 Phase 5 : %d appelant(s) de EnvRefreshTokenForGamertag hors migration boot "+
			"— ce helper ressuscite la source env var :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// ─── Guard 3 : duckdb.ReadOAuthRefreshToken (sync_meta legacy) ─────────────

// duckdbAuthReadPattern détecte les appels au DERNIER lecteur DuckDB du
// credential store legacy (sync_meta.oauth_refresh_token). Les écritures
// (WriteOAuthRefreshToken) et les lectures MSAL n'existent plus depuis la
// Phase 5 : leur simple réapparition ne compilerait pas.
//
// INDÉPENDANT DE L'ALIAS D'IMPORT : internal/platform/duckdb est importé sous 5
// noms dans le repo (duckdb, duckdbpkg, ddb, duckdbPlatform, platform_duckdb).
// Un motif figé sur `duckdb(pkg)?.` en manquait 3 (revue adversariale r1) →
// `\w+\.` capture n'importe quel alias.
//
// Mordant : `ddb.ReadOAuthRefreshToken(ctx, db)` dans un nouveau service fait
// échouer ce test, là où le motif précédent le laissait passer.
var duckdbAuthReadPattern = regexp.MustCompile(`\b\w+\.ReadOAuthRefreshToken\b`)

// allowedDuckDBAuthReaders : sites de PRODUCTION autorisés à lire
// sync_meta.oauth_refresh_token. ADR 0023 Phase 5 (2026-08-25) : uniquement la
// définition et la migration one-shot du boot (kill-switch daté, retrait cible
// 2026-10-01).
var allowedDuckDBAuthReaders = map[string]string{
	"internal/platform/duckdb/queries_auth.go": "Définition de la fonction (dernier lecteur legacy, supprimé avec la migration boot le 2026-10-01).",
	"cmd/server/main.go":                       "EXCEPTION UNIQUE : legacyAuthSourcesReader de migrateLegacyAuthTokensAtBoot (migration one-shot env+sync_meta → store).",
}

func TestSentinel_NoNewDuckDBAuthReaders(t *testing.T) {
	violations := scanProductionGoFiles(t, allowedDuckDBAuthReaders, duckdbAuthReadPattern.Match)

	if len(violations) > 0 {
		t.Errorf("REGRESSION ADR 0023 Phase 5 : %d lecteur(s) DuckDB du credential store legacy hors allowlist "+
			"— lire MultiUserTokenStore (sync_meta n'est plus un credential store) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// ─── Guard 4 : token-capture / token-import ne créent pas de fichiers .txt ────

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

// ─── Guard 5 : pas de nouveau lecteur de SPNKR_AZURE_CLIENT_SECRET ────────

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
		"allowedEnvHelperCallers":    allowedEnvHelperCallers,
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
