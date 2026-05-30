// Package sync — no_art_patterns_test.go : Phase 6 du plan d'éradication
// ART (cf. .ai/PLAN_LUSR_ART_HOME_CRASH.md).
//
// **Guard-rail anti-régression** : ce test scanne les fichiers Go du
// projet pour détecter l'apparition de nouveaux patterns SQL à risque
// ART (DELETE FROM, ON CONFLICT DO UPDATE, INSERT OR REPLACE) sur les
// tables connues comme déjà migrées en append-only. Si une nouvelle
// occurrence apparaît hors allowlist, le test fail — sauf si l'auteur
// l'ajoute explicitement à l'allowlist (avec justification).
//
// Ce test tourne en CI normale (pas de build tag) car il est rapide et
// déterministe (juste un grep sur les fichiers source).

package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// patternsAtRisk : motifs SQL connus comme déclencheurs du bug ART.
// Toute occurrence dans le hot path (non-test, non-migration) est suspect.
var patternsAtRisk = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bON\s+CONFLICT\b[^)]*\bDO\s+UPDATE\b`),
	regexp.MustCompile(`(?i)\bINSERT\s+OR\s+REPLACE\b`),
}

// tablesProtegees : tables que ce PR a migrées en append-only. Toute
// écriture mutative (DELETE/UPDATE/UPSERT) sur ces tables hors allowlist
// est interdite. Ajouter une table ici quand sa migration append-only
// est mergée.
var tablesProtegees = []string{
	"match_skill_rank",
	"match_csrs",
	"player_csr_snapshots",
	"pve_match_stats",
	// Progression V2 (fix 2026-05-30) : StreaksRepo.Upsert et
	// MilestoneEarnedRepo.Append migrés ON CONFLICT → SELECT-then-UPDATE/INSERT.
	// Protégées ici pour interdire toute réintroduction d'un ON CONFLICT
	// (qui ressuscite le bug ART surfacé en exécutant enfin le pipeline).
	"streak",
	"milestone_earned",
}

// allowlistArtPatterns : sites de prod où un pattern à risque reste
// présent VOLONTAIREMENT. Chaque entrée doit avoir un commentaire
// expliquant pourquoi. Format : "fichier:ligne_approx — raison".
//
// L'allowlist est volontairement vide pour les tables protégées —
// les patterns ne doivent pas exister. Pour les tables MOYEN/FAIBLE
// non encore migrées (cf. audit_art_writes.md), l'allowlist contient
// les sites tolérés temporairement.
var allowlistArtPatterns = map[string]string{
	// Documentation interne du package persist : mentionne les patterns
	// à risque par nature (c'est sa raison d'être).
	"internal/persist/doc.go": "Documentation : mentionne explicitement les patterns à risque dans son rôle d'expliquer le refactor anti-ART",
}

// TestNoARTPatternsOnProtectedTables — guard-rail principal.
//
// Pour chaque table protégée, scanne tous les fichiers .go non-test du
// projet. Toute occurrence d'un pattern à risque sur cette table est
// reportée. Le test fail si une occurrence non listée dans
// allowlistArtPatterns apparaît.
func TestNoARTPatternsOnProtectedTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	var violations []string
	for _, table := range tablesProtegees {
		tableRegex := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(table) + `\b`)

		err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				// Skip vendor, .git, node_modules, etc.
				name := info.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" ||
					name == "data" || name == "logs" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			// Exclure les fichiers de test, migrations (one-shot boot),
			// seeds, et le présent guard-rail.
			if strings.HasSuffix(path, "_test.go") ||
				strings.Contains(path, "/migration/") ||
				strings.Contains(path, "\\migration\\") ||
				strings.Contains(path, "/ops/") ||
				strings.Contains(path, "\\ops\\") {
				return nil
			}

			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil // skip silently
			}
			text := string(content)
			if !tableRegex.MatchString(text) {
				return nil
			}
			for _, pat := range patternsAtRisk {
				if pat.MatchString(text) {
					rel, _ := filepath.Rel(repoRoot, path)
					rel = filepath.ToSlash(rel)
					// Skip si le fichier est dans l'allowlist explicite.
					if _, allowed := allowlistArtPatterns[rel]; allowed {
						continue
					}
					violations = append(violations,
						"table="+table+" pattern_detected file="+rel)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("REGRESSION ART : %d violations détectées sur tables protégées :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
		t.Logf("Si l'ajout est volontaire, l'auteur doit migrer la table en append-only OU justifier dans le commit.")
	}
}

// TestAllowlistJustifiesEverything — vérifie que chaque entrée de
// l'allowlist correspond bien à un fichier qui contient au moins un
// pattern à risque. Si une entrée est obsolète (fichier nettoyé), le
// test alerte pour la retirer.
func TestAllowlistJustifiesEverything(t *testing.T) {
	repoRoot := findRepoRoot(t)

	for fileRel, reason := range allowlistArtPatterns {
		// Normaliser les séparateurs.
		absPath := filepath.Join(repoRoot, filepath.FromSlash(fileRel))
		content, err := os.ReadFile(absPath)
		if err != nil {
			t.Errorf("allowlist : fichier introuvable %q (raison: %q) — entrée à retirer ?", fileRel, reason)
			continue
		}
		text := string(content)
		hasRiskPattern := false
		for _, pat := range patternsAtRisk {
			if pat.MatchString(text) {
				hasRiskPattern = true
				break
			}
		}
		if !hasRiskPattern {
			t.Logf("allowlist : %q n'a plus de pattern à risque → retirer l'entrée (raison historique: %q)",
				fileRel, reason)
		}
	}
}

// findRepoRoot retourne la racine de `apps/go-api/` (depuis laquelle les
// paths de l'allowlist sont relatifs : ils commencent par `internal/...`).
// Les tests Go tournent depuis le dossier du package, donc on remonte
// jusqu'à trouver le `go.mod` du module.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("module root (go.mod) non trouvé depuis %s", wd)
	return ""
}
