// Package sync — append_only_state_guard_test.go : garde-fou anti-régression pour
// les tables d'ÉTAT migrées en APPEND-ONLY (doctrine zéro DELETE / zéro UPDATE
// indexé / zéro ON CONFLICT DO UPDATE).
//
// no_art_patterns_test.go ne couvre QUE ON CONFLICT / INSERT OR REPLACE et reste
// aveugle au DELETE nu (faux positifs file-level). Ce test comble le trou pour les
// tables converties : tout `DELETE FROM <table>` ou `INSERT … ON CONFLICT … DO
// UPDATE` / `INSERT OR REPLACE` sur l'une d'elles, dans le hot path serveur
// (hors _test / migration / ops / cmd / scripts), fait échouer la CI. L'état se lit
// via `<table>_latest` ; toute mutation est un INSERT pur.
//
// Ajouter une table ici à CHAQUE conversion append-only (campagne en cours).

package sync

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// appendOnlyStateTables : tables d'état converties en append-only (event-log).
// Aucun DELETE / ON CONFLICT DO UPDATE / INSERT OR REPLACE toléré dessus.
var appendOnlyStateTables = []string{
	"match_favorites",
	"match_favorites_history",
	"media_likes",
	"media_likes_history",
	"squad_member",
	"squad_member_history",
	"notification_preferences",
	"notification_preferences_history",
	"media_match_associations",
	"media_match_associations_history",
	"player_notifications",
	"player_notifications_history",
	"squad_challenge_participant",
	"squad_challenge_participant_history",
}

func TestNoMutationOnAppendOnlyStateTables(t *testing.T) {
	repoRoot := findRepoRoot(t)

	reOnConflictDoUpdate := regexp.MustCompile(`(?is)\bON\s+CONFLICT\b[^;]*\bDO\s+UPDATE\b`)
	reInsertOrReplace := regexp.MustCompile(`(?i)\bINSERT\s+OR\s+REPLACE\b`)

	var violations []string
	for _, table := range appendOnlyStateTables {
		reDelete := regexp.MustCompile(`(?i)\bDELETE\s+FROM\s+` + regexp.QuoteMeta(table) + `\b`)
		// INSERT ... ON CONFLICT/REPLACE visant cette table : on borne au statement
		// commençant par INSERT INTO <table>.
		reInsertTable := regexp.MustCompile(`(?is)\bINSERT\s+(?:OR\s+REPLACE\s+)?INTO\s+` + regexp.QuoteMeta(table) + `\b[^;]*`)

		err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				name := info.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" ||
					name == "data" || name == "logs" || name == "dist" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if strings.Contains(path, "/migration/") || strings.Contains(path, "\\migration\\") ||
				strings.Contains(path, "/ops/") || strings.Contains(path, "\\ops\\") ||
				strings.Contains(path, "/cmd/") || strings.Contains(path, "\\cmd\\") ||
				strings.Contains(path, "/scripts/") || strings.Contains(path, "\\scripts\\") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			text := stripGoComments(string(content))
			rel, _ := filepath.Rel(repoRoot, path)
			rel = filepath.ToSlash(rel)
			if reDelete.MatchString(text) {
				violations = append(violations, "DELETE FROM "+table+" dans "+rel)
			}
			for _, stmt := range reInsertTable.FindAllString(text, -1) {
				if reOnConflictDoUpdate.MatchString(stmt) || reInsertOrReplace.MatchString(stmt) {
					violations = append(violations, "INSERT ON CONFLICT/REPLACE sur "+table+" dans "+rel)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk: %v", err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("RÉGRESSION append-only : %d mutation(s) interdite(s) sur table d'état "+
			"(append-only = INSERT pur + vue _latest ; zéro DELETE/ON CONFLICT) :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}
