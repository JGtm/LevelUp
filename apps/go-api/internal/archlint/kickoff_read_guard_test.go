// Package archlint — kickoff_read_guard_test.go : le coup d'envoi d'un match et la durée de jeu ne se
// LISENT que par la chronologie et ses deux fragments SQL (ADR 0050).
//
// RÈGLE, sur les fichiers Go de production sous internal/ et cmd/ (tests et dossiers de migration
// exclus, commentaires Go et SQL retirés) : les noms de colonne `real_start_time`,
// `playable_duration_seconds`, `t0_quality` et `film_match_start_ms` ne figurent que
//
//   - dans les fragments `sqlfrag.SQLT0Ms` (le coup d'envoi) et `sqlfrag.SQLGameplayDurationMs` (la
//     durée de jeu), source unique côté SQL ;
//   - dans la résolution et son écriture : internal/sync/kickoff (lecture de l'état rangé du match),
//     internal/persist/kickoff_persister.go (écriture unique), internal/persist/shared_persister.go
//     (insertion de la synchronisation) ;
//   - dans la liste d'exemptions ci-dessous, DATÉE et justifiée, qui ne fait que diminuer.
//
// Les lecteurs passent par `timeline.ForMatch` (Go) et ces deux fragments (SQL), jamais par la
// colonne : la durée jouable de l'API n'a plus aucun lecteur (la colonne reste en base, DuckDB
// refusant de la retirer d'une table indexée sans la reconstruire) ; `t0_quality` et
// `film_match_start_ms` n'ont ni écrivain ni lecteur, `kickoff_write_guard_test.go` les interdit
// déjà hors des migrations et ce garde-rail les compte aussi, sans exemption.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// kickoffReadColumnsRE : les quatre noms de colonne que seuls la résolution, le persisteur, les fragments
// et les migrations ont le droit de nommer.
var kickoffReadColumnsRE = regexp.MustCompile(`\b(real_start_time|playable_duration_seconds|t0_quality|film_match_start_ms)\b`)

// kickoffReadAllowed : les fichiers de production qui nomment une de ces colonnes. Les cinq premiers sont
// la résolution, son écriture et les fragments ; les suivants sont des exemptions DATÉES.
var kickoffReadAllowed = map[string]string{
	"internal/platform/duckdb/sqlfrag/sql_fragments.go": "fragments uniques SQLT0Ms et SQLGameplayDurationMs",
	"internal/sync/kickoff/store.go":                    "résolution : lecture de l'état rangé du match (coup d'envoi, source)",
	"internal/persist/kickoff_persister.go":             "écriture unique du coup d'envoi",
	"internal/persist/shared_persister.go":              "insertion de la synchronisation : coup d'envoi posé par la résolution",
	"internal/persist/demo_seed_columns.go":             "liste des colonnes de l'insertion de la synchronisation, épinglée par le seeder de démonstration",
	"internal/sync/schema.go": "2026-11-18 : définition de la table `match_registry` d'une base neuve ; la colonne `playable_duration_seconds` y reste " +
		"parce que les bases existantes la gardent (DuckDB refuse de la retirer) et que les deux formes doivent rester égales",
	"internal/sync/testutil/fixture.go": "2026-11-18 : table de fixture des tests de synchronisation, alignée sur le schéma de `internal/sync/schema.go`",
}

// kickoffReadViolations rend les infractions d'un fichier de production (chemin relatif à apps/go-api).
func kickoffReadViolations(rel, src string) []string {
	if _, ok := kickoffReadAllowed[rel]; ok {
		return nil
	}
	var out []string
	for _, line := range strings.Split(stripComments(src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue // commentaire SQL d'un littéral multi-lignes
		}
		if m := kickoffReadColumnsRE.FindString(line); m != "" {
			out = append(out, rel+" : `"+m+"` — passer par timeline.ForMatch, sqlfrag.SQLT0Ms ou sqlfrag.SQLGameplayDurationMs")
		}
	}
	return out
}

func TestKickoffReadsOnlyThroughTheFragments(t *testing.T) {
	root := goAPIRoot(t)
	var violations []string
	seen := map[string]bool{}
	for _, sub := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, sub), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "migrations" || d.Name() == "migration" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			seen[rel] = true
			violations = append(violations, kickoffReadViolations(rel, string(data))...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	for rel := range kickoffReadAllowed {
		if !seen[rel] {
			violations = append(violations, rel+" : entrée d'exemption sans fichier — la retirer")
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("lectures du coup d'envoi ou de la durée de jeu hors des fragments (ADR 0050) :\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestKickoffReadExemptionsDatees : toute exemption autre que la résolution, son écriture et les fragments
// commence par une date.
func TestKickoffReadExemptionsDatees(t *testing.T) {
	date := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\b`)
	permanent := map[string]bool{
		"internal/platform/duckdb/sqlfrag/sql_fragments.go": true,
		"internal/sync/kickoff/store.go":                    true,
		"internal/persist/kickoff_persister.go":             true,
		"internal/persist/shared_persister.go":              true,
		"internal/persist/demo_seed_columns.go":             true,
	}
	for rel, raison := range kickoffReadAllowed {
		if !permanent[rel] && !date.MatchString(raison) {
			t.Errorf("%s : exemption sans date (%q)", rel, raison)
		}
	}
}

// Le garde-rail reconnaît les lectures qu'il interdit et laisse passer celles qu'il permet.
func TestKickoffReadViolations_Motifs(t *testing.T) {
	cas := []struct {
		nom, rel, src string
		viole         bool
	}{
		{"lecture de la durée jouable", "internal/service/x.go", "q := `SELECT r.playable_duration_seconds FROM match_registry r`", true},
		{"lecture du coup d'envoi brut", "internal/platform/duckdb/y.go", "q := `SELECT CASE WHEN r.real_start_time IS NULL THEN 0 END`", true},
		{"qualité morte", "internal/sync/z.go", "q := `SELECT t0_quality FROM match_registry`", true},
		{"colonne héritée", "internal/sync/z.go", "q := `SELECT film_match_start_ms FROM match_registry`", true},
		{"fragment", "internal/platform/duckdb/sqlfrag/sql_fragments.go", "return prefix + \"real_start_time\"", false},
		{"commentaire Go", "internal/sync/z.go", "// real_start_time n'est plus lu ici\nx := 1", false},
		{"commentaire SQL", "internal/sync/z.go", "q := `\n  -- real_start_time : le coup d'envoi\n  SELECT 1`", false},
		{"par le fragment", "internal/platform/duckdb/y.go", "q := `SELECT ` + sqlfrag.SQLT0Ms(\"r\")", false},
	}
	for _, c := range cas {
		if got := len(kickoffReadViolations(c.rel, c.src)) > 0; got != c.viole {
			t.Errorf("%s : violation=%v, attendu %v", c.nom, got, c.viole)
		}
	}
}
