package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// ReconcileOrphanedMediaFiles parcourt media_files pour un player donné et
// détecte les entrées dont file_path stocké ne correspond plus à un fichier sur
// disque (typiquement après une conversion locale type .mp4 → .mkv qui change
// le hash mais aussi les cas de rename pur où le hash est identique et donc le
// scan walkMediaDir skip l'entrée).
//
// Pour chaque orphelin :
//   - cherche un fichier de même file_stem dans le dossier parent
//   - si exactement un match avec une extension reconnue → UPDATE file_path/name/ext
//   - si zéro match → laisse l'entrée intacte (fichier supprimé par l'utilisateur)
//   - si plusieurs matches → log warning, ne touche pas (ambiguïté)
//
// Idempotent : un second passage sans changement disque ne touche rien.
//
// Retourne le nombre d'entrées effectivement mises à jour.
func ReconcileOrphanedMediaFiles(ctx context.Context, db *sql.DB, playerSlug string, store MediaPathStore) (int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, file_path, file_stem
		FROM media_files
		WHERE player_slug = ?
		  AND file_stem IS NOT NULL
		  AND file_stem != ''
	`, playerSlug)
	if err != nil {
		return 0, fmt.Errorf("reconcile: SELECT: %w", err)
	}
	type entry struct {
		ID   int64
		Path string
		Stem string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.Path, &e.Stem); err != nil {
			rows.Close()
			return 0, fmt.Errorf("reconcile: scan: %w", err)
		}
		entries = append(entries, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	updated := 0
	for _, e := range entries {
		absPath := store.ToAbs(e.Path)
		if _, statErr := os.Stat(absPath); statErr == nil {
			continue
		}
		dir := filepath.Dir(absPath)
		matches, scanErr := findFilesByStem(dir, e.Stem)
		if scanErr != nil {
			slog.WarnContext(ctx, "reconcile: scan dossier orphelin échoué",
				"dir", dir, "stem", e.Stem, "err", scanErr)
			continue
		}
		if len(matches) == 0 {
			continue
		}
		if len(matches) > 1 {
			slog.WarnContext(ctx, "reconcile: plusieurs candidats pour le même stem, skip",
				"stem", e.Stem, "matches", matches)
			continue
		}
		newAbs := matches[0]
		newExt := strings.ToLower(filepath.Ext(newAbs))
		newBase := filepath.Base(newAbs)
		newStored := store.ToRel(newAbs, playerSlug)
		if newStored == "" {
			newStored = newAbs
		}
		if _, err := db.ExecContext(ctx, `
			UPDATE media_files
			SET file_path = ?, file_name = ?, file_ext = ?
			WHERE id = ?
		`, newStored, newBase, newExt, e.ID); err != nil {
			slog.ErrorContext(ctx, "reconcile: UPDATE échoué",
				"id", e.ID, "old_path", e.Path, "new_path", newStored, "err", err)
			continue
		}
		slog.InfoContext(ctx, "reconcile: file_path resynced",
			"id", e.ID, "old_path", e.Path, "new_path", newStored, "new_ext", newExt)
		updated++
	}
	return updated, nil
}

// findFilesByStem scanne dir et retourne les chemins absolus des fichiers dont
// le nom (sans extension) correspond exactement à stem ET dont l'extension est
// dans supportedExtensions. Ne descend pas récursivement.
func findFilesByStem(dir, stem string) ([]string, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if _, ok := supportedExtensions[ext]; !ok {
			continue
		}
		nameStem := strings.TrimSuffix(name, filepath.Ext(name))
		if nameStem == stem {
			matches = append(matches, filepath.Join(dir, name))
		}
	}
	return matches, nil
}
