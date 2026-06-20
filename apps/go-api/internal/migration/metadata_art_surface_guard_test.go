// Package migration — metadata_art_surface_guard_test.go : garde-fou de CLASSE
// (cross-DB) contre la réintroduction du bug ART DuckDB.
//
// Le bug ("Failed to delete all rows from index" → DB FATAL invalidated → toute
// l'app/cloche/home tombe) est réapparu ≥4 fois, TOUJOURS via le même pattern :
// un UPDATE/DELETE per-row qui touche une LIGNE couverte par un index ART, en
// particulier un UPDATE qui mute une COLONNE INDEXÉE.
//   - playlists_catalog (idx active/experience)         — 2026-06-01
//   - catalog_fetch_queue (PK + idx attempts)           — 2026-06-19
//   - game_variants_catalog (idx mode_canonical)        — 2026-06-19
//   - player_notifications (idx_pn_xuid_unread/read_at) — droppé 2026-05-15 PUIS
//     RECRÉÉ par le rebuild purge_data_health → re-cassé jusqu'au 2026-06-19.
//   - challenge (idx status/arc_id/campaign_id)         — 2026-06-19.
//
// no_art_patterns_test.go (package sync) ne détecte QUE ON CONFLICT / INSERT OR
// REPLACE — aveugle au vrai déclencheur (UPDATE/DELETE nu sur colonne/table
// indexée). Ce test comble le trou par CONSTRUCTION du schéma : il interdit, dans
// toutes les migrations, de (re)créer un index ART sur une surface mutée.
//
// Deux règles :
//  1. noSecondaryIndexTables : tables de RÉFÉRENCE minuscules, mutées par
//     UPDATE et/ou vidées par DELETE → AUCUN index secondaire toléré (PK-only ;
//     la PK n'est jamais mutée). Un scan complet y est instantané.
//  2. forbiddenIndexedColumns : tables plus grosses où certains index sont OK,
//     mais où une colonne MUTÉE par un UPDATE ne doit JAMAIS être indexée.

package migration

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Règle 1 — PK-only total (aucun index secondaire).
var noSecondaryIndexTables = []string{
	"playlists_catalog",
	"maps_catalog",
	"game_variants_catalog",
	"map_mode_pair_definitions",
	"catalog_fetch_queue",
	"battlepass_track_definitions",
	"battlepass_item_definitions",
}

// Règle 2 — colonnes mutées par un UPDATE qui ne doivent jamais être indexées.
var forbiddenIndexedColumns = map[string][]string{
	"player_notifications": {"read_at"},                              // MarkNotifications* SET read_at
	"challenge":            {"status", "arc_id", "campaign_id"},      // UpdateStatus/DetachFromArc/campaign
	"playlists_catalog":    {"experience", "is_active", "is_ranked"}, // seedPlaylistsCatalog
}

var (
	reBlockCommentMig = regexp.MustCompile(`(?s)/\*.*?\*/`)
	reLineCommentMig  = regexp.MustCompile(`//[^\n]*`)
	reSQLLineComment  = regexp.MustCompile(`--[^\n]*`)
	// CREATE INDEX [IF NOT EXISTS] <name> ON <table>(<cols>)
	reCreateIndexGuard = regexp.MustCompile(`(?is)\bCREATE\s+INDEX\s+(?:IF\s+NOT\s+EXISTS\s+)?\S+\s+ON\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([^)]*)\)`)
)

func stripAllComments(src string) string {
	src = reBlockCommentMig.ReplaceAllString(src, "")
	src = reLineCommentMig.ReplaceAllString(src, "")
	src = reSQLLineComment.ReplaceAllString(src, "")
	return src
}

// indexedColumnNames extrait les noms de colonnes d'un groupe `(col1, col2 DESC)`.
func indexedColumnNames(group string) []string {
	var cols []string
	for _, part := range strings.Split(group, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 {
			continue
		}
		cols = append(cols, strings.ToLower(fields[0]))
	}
	return cols
}

// TestNoARTSurfaceIndexInMigrations échoue si une migration (re)crée un index ART
// sur une surface mutée — règle 1 (table PK-only) ou règle 2 (colonne mutée).
func TestNoARTSurfaceIndexInMigrations(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	noIndex := make(map[string]struct{}, len(noSecondaryIndexTables))
	for _, tbl := range noSecondaryIndexTables {
		noIndex[tbl] = struct{}{}
	}

	var violations []string
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return werr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		text := stripAllComments(string(content))
		for _, m := range reCreateIndexGuard.FindAllStringSubmatch(text, -1) {
			table := strings.ToLower(m[1])
			rel, _ := filepath.Rel(dir, path)
			rel = filepath.ToSlash(rel)
			if _, bad := noIndex[table]; bad {
				violations = append(violations,
					"index interdit sur table PK-only '"+table+"' ("+rel+")")
				continue
			}
			if forbiddenCols, ok := forbiddenIndexedColumns[table]; ok {
				cols := indexedColumnNames(m[2])
				for _, c := range cols {
					for _, fc := range forbiddenCols {
						if c == fc {
							violations = append(violations,
								"index sur colonne MUTÉE '"+table+"."+c+"' ("+rel+") — surface ART")
						}
					}
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk: %v", walkErr)
	}

	if len(violations) > 0 {
		t.Errorf("RÉGRESSION ART : %d index sur surface mutée (UPDATE/DELETE per-row sur "+
			"colonne/table indexée corrompt DuckDB → FATAL invalidated). PK-only obligatoire :\n  - %s",
			len(violations), strings.Join(violations, "\n  - "))
	}
}
