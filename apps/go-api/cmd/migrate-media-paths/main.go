// migrate-media-paths : convertit les paths absolus (legacy) en paths relatifs
// {owner_slug}/{rel} dans media_files (file_path, thumbnail_path) et propage
// au PK de media_likes (media_path). Idempotent : skip les paths déjà relatifs.
//
// Usage :
//
//	migrate-media-paths --db shared_social.duckdb [--captures-base C:\Captures] [--dry-run]
//
// Heuristique fallback pour les paths cassés (thumbnail pointant vers un fichier
// inexistant après conversion) : NULL out la colonne — le prochain BackfillThumbnailPaths
// (ou cmd/regen-thumbnails) repointera vers le bon emplacement.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ops"
)

type appSettings struct {
	MediaCapturesBaseDir string `json:"media_captures_base_dir"`
}

func loadCapturesBase(settingsPath string) string {
	if settingsPath == "" {
		return ""
	}
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return ""
	}
	var s appSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return ""
	}
	return s.MediaCapturesBaseDir
}

type mediaRow struct {
	id              int64
	playerSlug      string
	filePath        string
	thumbnailPath   sql.NullString
	newFilePath     string
	newThumbnail    sql.NullString
	thumbnailExists bool
}

func main() {
	dbPath := flag.String("db", "", "path vers shared_social.duckdb (requis)")
	capturesBase := flag.String("captures-base", "", "MediaCapturesBaseDir (sinon lu depuis --settings)")
	settingsPath := flag.String("settings", "app_settings.json", "path vers app_settings.json (fallback)")
	dryRun := flag.Bool("dry-run", false, "afficher les UPDATE sans les exécuter")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "--db requis")
		os.Exit(2)
	}

	base := *capturesBase
	if base == "" {
		base = loadCapturesBase(*settingsPath)
	}
	if base == "" {
		fmt.Fprintln(os.Stderr, "captures base introuvable (ni --captures-base ni app_settings.json:media_captures_base_dir)")
		os.Exit(2)
	}
	fmt.Printf("DB:            %s\n", *dbPath)
	fmt.Printf("CapturesBase:  %s\n", base)
	fmt.Printf("DryRun:        %v\n\n", *dryRun)

	store := ops.MediaPathStore{CapturesBase: base}

	// READ_ONLY en dry-run pour pouvoir tourner pendant que le serveur a la DB ouverte.
	openPath := *dbPath
	if *dryRun {
		openPath = *dbPath + "?access_mode=READ_ONLY"
	}
	db, err := sql.Open("duckdb", openPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer db.Close() //nolint:errcheck

	rows, err := db.Query("SELECT id, COALESCE(player_slug, ''), file_path, thumbnail_path FROM media_files")
	if err != nil {
		fmt.Fprintln(os.Stderr, "select media_files:", err)
		os.Exit(1)
	}
	var items []mediaRow
	for rows.Next() {
		var r mediaRow
		if err := rows.Scan(&r.id, &r.playerSlug, &r.filePath, &r.thumbnailPath); err != nil {
			rows.Close()
			fmt.Fprintln(os.Stderr, "scan:", err)
			os.Exit(1)
		}
		items = append(items, r)
	}
	rows.Close()

	stats := struct {
		total, fpAbs, fpRel, fpConverted, fpUnchanged              int
		thumbNull, thumbAbs, thumbRel, thumbConverted, thumbNulled int
		likesUpdated                                               int
	}{total: len(items)}

	for i := range items {
		r := &items[i]
		// file_path
		if filepath.IsAbs(r.filePath) {
			stats.fpAbs++
			rel := convertPath(store, r.filePath, r.playerSlug)
			if rel != "" && rel != r.filePath {
				r.newFilePath = rel
				stats.fpConverted++
			} else {
				stats.fpUnchanged++
			}
		} else {
			stats.fpRel++
		}

		// thumbnail_path
		if !r.thumbnailPath.Valid {
			stats.thumbNull++
			continue
		}
		if !filepath.IsAbs(r.thumbnailPath.String) {
			stats.thumbRel++
			continue
		}
		stats.thumbAbs++
		relThumb := convertPath(store, r.thumbnailPath.String, r.playerSlug)
		if relThumb != "" && relThumb != r.thumbnailPath.String {
			// Vérifier si le fichier existe au nouveau path résolu
			if _, err := os.Stat(store.ToAbs(relThumb)); err == nil {
				r.newThumbnail = sql.NullString{String: relThumb, Valid: true}
				r.thumbnailExists = true
				stats.thumbConverted++
			} else {
				// Path cassé : NULL out, sera relinké par BackfillThumbnailPaths
				r.newThumbnail = sql.NullString{Valid: false}
				stats.thumbNulled++
			}
		}
	}

	// Apply updates
	if !*dryRun {
		tx, err := db.Begin()
		if err != nil {
			fmt.Fprintln(os.Stderr, "begin tx:", err)
			os.Exit(1)
		}
		for _, r := range items {
			if r.newFilePath != "" {
				if _, err := tx.Exec("UPDATE media_files SET file_path = ? WHERE id = ?", r.newFilePath, r.id); err != nil {
					tx.Rollback() //nolint:errcheck
					fmt.Fprintln(os.Stderr, "update media_files:", err)
					os.Exit(1)
				}
				// Propager au PK media_likes
				res, err := tx.Exec("UPDATE media_likes SET media_path = ? WHERE media_path = ?", r.newFilePath, r.filePath)
				if err != nil {
					tx.Rollback() //nolint:errcheck
					fmt.Fprintln(os.Stderr, "update media_likes:", err)
					os.Exit(1)
				}
				n, _ := res.RowsAffected()
				stats.likesUpdated += int(n)
			}
			if r.newThumbnail.Valid || (r.thumbnailPath.Valid && !r.newThumbnail.Valid && filepath.IsAbs(r.thumbnailPath.String) && !r.thumbnailExists && r.newFilePath != "") {
				// Note: la condition complexe gère le NULL-out quand la conversion a abouti
				// à un fichier introuvable. r.newThumbnail.Valid=false dans ce cas mais
				// on veut quand même écrire NULL en DB.
			}
			// Update thumbnail_path : 3 cas
			//   1. r.newThumbnail.Valid → write rel path
			//   2. thumbnail était abs + path cassé après conversion → write NULL
			//   3. rien à faire
			switch {
			case r.newThumbnail.Valid:
				if _, err := tx.Exec("UPDATE media_files SET thumbnail_path = ? WHERE id = ?", r.newThumbnail.String, r.id); err != nil {
					tx.Rollback() //nolint:errcheck
					fmt.Fprintln(os.Stderr, "update thumbnail_path:", err)
					os.Exit(1)
				}
			case r.thumbnailPath.Valid && filepath.IsAbs(r.thumbnailPath.String) && !r.thumbnailExists && convertPath(store, r.thumbnailPath.String, r.playerSlug) != "":
				// abs converti mais fichier inexistant → NULL out
				if _, err := tx.Exec("UPDATE media_files SET thumbnail_path = NULL WHERE id = ?", r.id); err != nil {
					tx.Rollback() //nolint:errcheck
					fmt.Fprintln(os.Stderr, "null thumbnail_path:", err)
					os.Exit(1)
				}
			}
		}
		if err := tx.Commit(); err != nil {
			fmt.Fprintln(os.Stderr, "commit:", err)
			os.Exit(1)
		}
	}

	fmt.Println("Résultats:")
	fmt.Printf("  media_files total       : %d\n", stats.total)
	fmt.Printf("  file_path absolu        : %d (converti=%d, inchangé=%d)\n", stats.fpAbs, stats.fpConverted, stats.fpUnchanged)
	fmt.Printf("  file_path déjà relatif  : %d\n", stats.fpRel)
	fmt.Printf("  thumbnail NULL          : %d\n", stats.thumbNull)
	fmt.Printf("  thumbnail absolu        : %d (converti=%d, nullé=%d)\n", stats.thumbAbs, stats.thumbConverted, stats.thumbNulled)
	fmt.Printf("  thumbnail déjà relatif  : %d\n", stats.thumbRel)
	if !*dryRun {
		fmt.Printf("  media_likes propagés    : %d\n", stats.likesUpdated)
	} else {
		fmt.Println("\n(dry-run : aucune écriture)")
	}
}

// convertPath transforme un path absolu en path relatif {owner_slug}/{rel}.
// Stratégies en cascade :
//  1. MediaPathStore.ToRel (cas standard : path sous capturesBase).
//  2. Heuristique : chercher /{slug}/ dans le path et tronquer.
//
// Retourne "" si aucune stratégie n'aboutit.
func convertPath(store ops.MediaPathStore, absPath, ownerSlug string) string {
	if ownerSlug == "" {
		return ""
	}
	if rel := store.ToRel(absPath, ownerSlug); rel != "" {
		return rel
	}
	// Heuristique : pour les paths legacy hors capturesBase (ex: data/players/...).
	// Chercher la première occurrence de `{slug}/` ou `{slug}\` dans le path
	// et prendre tout à partir de là.
	normalized := filepath.ToSlash(absPath)
	marker := "/" + ownerSlug + "/"
	idx := strings.Index(normalized, marker)
	if idx < 0 {
		// pas de slash devant slug ? essayer en tête
		if strings.HasPrefix(normalized, ownerSlug+"/") {
			return normalized
		}
		return ""
	}
	return normalized[idx+1:] // skip leading "/"
}
