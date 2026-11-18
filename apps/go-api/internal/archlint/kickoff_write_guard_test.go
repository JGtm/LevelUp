// Package archlint — kickoff_write_guard_test.go : le coup d'envoi d'un match et le temps de jeu de
// ses joueurs n'ont QU'UN écrivain (ADR 0050).
//
// RÈGLES, sur les fichiers Go de production sous internal/ et cmd/ (tests et dossiers de migration
// exclus, commentaires retirés) :
//
//  1. un `UPDATE … SET` de `real_start_time`, `real_start_source` ou `time_played_seconds` ne vit que
//     dans le persisteur unique, internal/persist/kickoff_persister.go ;
//  2. une insertion dans `match_registry` ou `match_participants` qui pose une de ces colonnes ne vit
//     que dans l'insertion de la synchronisation (internal/persist/shared_persister.go, valeurs posées
//     par la résolution unique avant l'insertion), ou sur la liste d'exemptions datée ;
//  3. `t0_quality` et `film_match_start_ms` ne sont plus ni écrites ni lues : leurs noms ne figurent
//     nulle part hors des migrations qui les ont créées (les colonnes restent en base, DuckDB refusant
//     de les retirer d'une table indexée sans la reconstruire).
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// kickoffPersisterFile : le seul fichier qui met à jour ces colonnes.
const kickoffPersisterFile = "internal/persist/kickoff_persister.go"

// kickoffInsertAllowed : les insertions qui posent ces colonnes. La première est l'insertion de la
// synchronisation ; les autres sont des exemptions DATÉES, qui ne font que diminuer.
var kickoffInsertAllowed = map[string]string{
	"internal/persist/shared_persister.go": "insertion de la synchronisation : coup d'envoi, source et temps de jeu posés par kickoff.ApplyToRows avant l'insertion",
	"internal/ops/seed_demo_synthetic_shared.go": "2026-11-18 : corpus de démonstration fabriqué (temps de jeu inventé, aucune heure d'arrivée) ; " +
		"aucune résolution possible, la base de démonstration n'est jamais une base réelle",
	"cmd/h5-roster-refetch/insert.go": "2026-11-18 : participants Halo 5 reconstruits depuis l'API (TotalTimePlayed, heures d'arrivée NULL) : " +
		"présence inconnue, la valeur de l'API est celle que la résolution garderait",
}

var (
	kickoffColumnsRE = regexp.MustCompile(`\b(real_start_time|real_start_source|time_played_seconds)\b`)
	// updateSetRE capture l'affectation d'un UPDATE jusqu'à son WHERE (ou la fin du littéral).
	updateSetRE = regexp.MustCompile("(?is)\\bUPDATE\\s+\\w+\\s+SET\\b([^`\"]*?)(?:\\bWHERE\\b|`|\"|$)")
	// insertColsRE capture la liste de colonnes d'un INSERT dans une des deux tables.
	insertColsRE  = regexp.MustCompile(`(?is)\bINSERT\s+(?:OR\s+\w+\s+)?INTO\s+(?:match_registry|match_participants)\s*\(([^)]*)\)`)
	deadColumnsRE = regexp.MustCompile(`\b(t0_quality|film_match_start_ms)\b`)
)

// kickoffViolations rend les infractions d'un fichier de production (chemin relatif à apps/go-api).
func kickoffViolations(rel, src string) []string {
	code := stripComments(src)
	var out []string
	for _, m := range updateSetRE.FindAllStringSubmatch(code, -1) {
		if kickoffColumnsRE.MatchString(m[1]) && rel != kickoffPersisterFile {
			out = append(out, rel+" : UPDATE du coup d'envoi ou du temps de jeu hors du persisteur unique")
		}
	}
	for _, m := range insertColsRE.FindAllStringSubmatch(code, -1) {
		if _, ok := kickoffInsertAllowed[rel]; kickoffColumnsRE.MatchString(m[1]) && !ok {
			out = append(out, rel+" : insertion du coup d'envoi ou du temps de jeu hors de la synchronisation")
		}
	}
	if deadColumnsRE.MatchString(code) {
		out = append(out, rel+" : t0_quality / film_match_start_ms ne s'écrivent ni ne se lisent plus")
	}
	return out
}

func TestKickoffWritesOnlyThroughThePersister(t *testing.T) {
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
			violations = append(violations, kickoffViolations(rel, string(data))...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}
	for rel := range kickoffInsertAllowed {
		if !seen[rel] {
			violations = append(violations, rel+" : entrée d'exemption sans fichier — la retirer")
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("écritures du coup d'envoi (ADR 0050) :\n  %s", strings.Join(violations, "\n  "))
	}
}

// TestKickoffExemptionsDatees : toute exemption autre que l'insertion de la synchronisation commence
// par une date.
func TestKickoffExemptionsDatees(t *testing.T) {
	date := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}\b`)
	for rel, raison := range kickoffInsertAllowed {
		if rel != "internal/persist/shared_persister.go" && !date.MatchString(raison) {
			t.Errorf("%s : exemption sans date (%q)", rel, raison)
		}
	}
}

// Le garde-rail reconnaît les formes qu'il interdit et laisse passer celles qu'il permet.
func TestKickoffViolations_Motifs(t *testing.T) {
	cas := []struct {
		nom, rel, src string
		viole         bool
	}{
		{"update hors persisteur", "internal/sync/x.go",
			"q := `UPDATE match_participants SET time_played_seconds = ? WHERE match_id = ?`", true},
		{"update du persisteur", kickoffPersisterFile,
			"q := `UPDATE match_registry SET real_start_time = ?, real_start_source = ? WHERE match_id = ?`", false},
		{"update d'une autre colonne", "internal/sync/x.go",
			"q := `UPDATE match_registry SET match_intensity = ? WHERE real_start_time IS NULL`", false},
		{"insertion hors synchronisation", "internal/ops/y.go",
			"q := `INSERT INTO match_registry (match_id, real_start_time) VALUES (?, ?)`", true},
		{"insertion de la synchronisation", "internal/persist/shared_persister.go",
			"q := `INSERT INTO match_participants (match_id, time_played_seconds) VALUES (?, ?)`", false},
		{"colonne morte", "internal/sync/z.go", "q := `SELECT t0_quality FROM match_registry`", true},
		{"colonne morte en commentaire", "internal/sync/z.go", "// t0_quality n'existe plus\nx := 1", false},
	}
	for _, c := range cas {
		if got := len(kickoffViolations(c.rel, c.src)) > 0; got != c.viole {
			t.Errorf("%s : violation=%v, attendu %v", c.nom, got, c.viole)
		}
	}
}
