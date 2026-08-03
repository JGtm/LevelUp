// media_repo_delete_guard_test.go — garde-rail du prédicat de visibilité média
// (v7.3 lot 2, item 3.1, règle CLAUDE.md n°6 : centralisation + garde-rail).
//
// POURQUOI CE TEST EXISTE. La suppression d'un média est un SOFT-DELETE
// (media_files.status = 'deleted') : la ligne reste en base et n'est masquée que
// par un prédicat WHERE. Une factorisation sans garde-rail re-diverge — ici la
// divergence est visible par l'utilisateur : une seule lecture non filtrée fait
// réapparaître un média supprimé (rail d'accueil, onglet médias d'un match,
// filtre Auteurs, candidats d'association...), avec un fichier qui n'existe plus
// sur le disque, donc une vignette cassée impossible à faire disparaître.
package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// reFromMediaFiles capture les lectures de la table. `media_files__rebuild`
// (table temporaire de migration) est exclue par la frontière de mot.
var reFromMediaFiles = regexp.MustCompile(`(?i)\bFROM\s+media_files\b(?:__)?`)

// coveredByVisiblePredicate : formes acceptées comme filtrage des supprimés.
//   - MediaVisiblePredicate : l'helper canonique ;
//   - `status = 'active'` : filtre PRÉEXISTANT (rail d'accueil, citations) qui
//     exclut 'deleted' par construction — plus restrictif encore.
var coveredByVisiblePredicate = []string{
	"MediaVisiblePredicate",
	"status = 'active'",
}

// allowlistMediaFilesRawRead : fichiers lisant media_files SANS prédicat de
// visibilité, volontairement. Toute entrée porte sa justification.
var allowlistMediaFilesRawRead = map[string]string{
	"internal/ops/media.go": "insertMediaFile — dédup d'INDEXATION : doit au contraire VOIR " +
		"les lignes supprimées pour les ressusciter quand le fichier est re-déposé " +
		"(sinon un média supprimé puis ré-uploadé resterait invisible pour toujours).",
	"internal/persist/shared_social_persister_batch.go": "persistMediaFiles — dédup " +
		"applicative file_path à l'INSERT (remplace l'ex-contrainte UNIQUE retirée pour " +
		"cause de bug ART) : une ligne supprimée doit rester détectée comme doublon, " +
		"sinon deux lignes du même file_path coexisteraient.",
	"internal/ops/media_reconcile.go": "ReconcileOrphanedMediaFiles — réconciliation " +
		"chemins DB↔disque, opère sur l'index complet (maintenance, ne sert rien à l'écran).",
	"internal/ops/media_hls.go": "finalizeMediaHLS + comptage de transcodage — plomberie " +
		"d'indexation par id, jamais une liste servie à l'utilisateur.",
	"internal/ops/seed_demo_media.go":    "seed du mode démo (base neuve, aucun média supprimé).",
	"internal/ops/seed_demo_media_h5.go": "seed du mode démo (base neuve, aucun média supprimé).",
}

// TestAllMediaFilesReadsFilterDeleted — garde-rail principal.
func TestAllMediaFilesReadsFilterDeleted(t *testing.T) {
	repoRoot := findModuleRootForMediaGuard(t)

	var violations []string
	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			switch info.Name() {
			case "vendor", ".git", "node_modules", "data", "logs", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// Migrations et outils one-shot : hors serveur, rebuild de schéma complet.
		norm := filepath.ToSlash(path)
		if strings.Contains(norm, "/migration/") || strings.Contains(norm, "/migrations/") ||
			strings.Contains(norm, "/cmd/") || strings.Contains(norm, "/scripts/") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		text := string(content)
		if !reFromMediaFiles.MatchString(text) {
			return nil
		}
		for _, form := range coveredByVisiblePredicate {
			if strings.Contains(text, form) {
				return nil
			}
		}
		rel, _ := filepath.Rel(repoRoot, path)
		relSlash := filepath.ToSlash(rel)
		if _, allowed := allowlistMediaFilesRawRead[relSlash]; allowed {
			return nil
		}
		violations = append(violations, relSlash)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(violations) > 0 {
		t.Errorf("lecture(s) de media_files sans prédicat de visibilité — un média SUPPRIMÉ "+
			"y réapparaîtrait (fichier absent du disque → vignette cassée irréparable). "+
			"Ajouter `AND `+MediaVisiblePredicate(alias) à la requête, ou justifier dans "+
			"allowlistMediaFilesRawRead :\n  - %s", strings.Join(violations, "\n  - "))
	}
}

// TestMediaFilesReadGuard_Sanity vérifie que le garde-rail MORD : un garde-rail
// qui ne détecte jamais rien ne protège rien.
func TestMediaFilesReadGuard_Sanity(t *testing.T) {
	unfiltered := "q := `SELECT file_path FROM media_files WHERE kind = 'video'`"
	if !reFromMediaFiles.MatchString(unfiltered) {
		t.Error("le motif devrait détecter une lecture nue de media_files")
	}
	filtered := "q := `SELECT file_path FROM media_files WHERE ` + MediaVisiblePredicate(\"\")"
	covered := false
	for _, form := range coveredByVisiblePredicate {
		if strings.Contains(filtered, form) {
			covered = true
		}
	}
	if !covered {
		t.Error("une lecture filtrée par MediaVisiblePredicate devrait être reconnue comme couverte")
	}
}

// TestAllowlistMediaFilesReadIsAlive refuse les entrées d'allowlist devenues
// obsolètes (fichier disparu ou qui ne lit plus media_files) : une allowlist qui
// pourrit finit par couvrir du code qu'elle n'a jamais examiné.
func TestAllowlistMediaFilesReadIsAlive(t *testing.T) {
	repoRoot := findModuleRootForMediaGuard(t)
	for fileRel, reason := range allowlistMediaFilesRawRead {
		content, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(fileRel)))
		if err != nil {
			t.Errorf("allowlist : fichier introuvable %q (raison: %q) — entrée à retirer ?", fileRel, reason)
			continue
		}
		if !reFromMediaFiles.MatchString(string(content)) {
			t.Errorf("allowlist obsolète : %q ne lit plus media_files → retirer l'entrée "+
				"(raison historique: %q)", fileRel, reason)
		}
	}
}

// TestMediaVisiblePredicate_Forms verrouille la forme du prédicat : le COALESCE
// est INDISPENSABLE (status vaut NULL sur la majorité des lignes ; un
// `status <> 'deleted'` nu les éliminerait toutes, vidant la galerie).
func TestMediaVisiblePredicate_Forms(t *testing.T) {
	withAlias := MediaVisiblePredicate("mf")
	if !strings.Contains(withAlias, "COALESCE(mf.status") {
		t.Errorf("prédicat avec alias = %q, want un COALESCE sur mf.status", withAlias)
	}
	noAlias := MediaVisiblePredicate("")
	if !strings.Contains(noAlias, "COALESCE(status") {
		t.Errorf("prédicat sans alias = %q, want un COALESCE sur status", noAlias)
	}
	for _, p := range []string{withAlias, noAlias} {
		if !strings.Contains(p, "'deleted'") {
			t.Errorf("prédicat %q ne référence pas la valeur 'deleted'", p)
		}
	}
}

// findModuleRootForMediaGuard remonte jusqu'au go.mod du module.
func findModuleRootForMediaGuard(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
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
	t.Fatalf("module root (go.mod) introuvable")
	return ""
}
