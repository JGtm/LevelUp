// reindex-media-thumbs : aligne media_files.thumbnail_path sur l'état réel du
// disque sans regénérer les miniatures.
//
// Pourquoi : après les migrations successives (paths absolus → relatifs,
// .gif → .webp, sous-dossier "captures/thumbs" → "thumbs"), des entrées DB
// pointent vers des miniatures qui n'existent plus ou plus à cet emplacement.
// Cela cause des 404 sur GET /api/v1/players/{slug}/media/files/... à chaque
// affichage de la page médias.
//
// Vs cmd/regen-thumbnails : ce cmd NE supprime PAS les miniatures et NE les
// regénère PAS — il se contente de réparer les pointeurs DB par recherche par
// stem dans {capturesBase}/{owner}/thumbs/. Sûr à lancer pendant que le
// serveur tourne (la table media_files est dans shared_social.duckdb, ouverte
// en RW par le serveur ; on utilise donc --dry-run par défaut ou la conn
// du serveur via un appel administratif).
//
// Usage :
//
//	reindex-media-thumbs --db shared_social.duckdb [--captures-base C:\Captures] [--slug JGtm] [--dry-run]
//
// Algorithme par ligne media_files (kind='video') :
//  1. Si thumbnail_path est NULL ou pointe vers un fichier inexistant
//     (après MediaPathStore.ToAbs), chercher {capturesBase}/{owner}/thumbs/
//     un fichier dont le stem matche file_stem (ou file_name moins extension).
//     Prefer .webp > .gif. Strip le suffixe hash éventuel pour matcher
//     "Replay 2026-03-27 23-17-43_bd811d72870d" ↔ "Replay 2026-03-27 23-17-43".
//  2. Trouvé → UPDATE thumbnail_path avec le path relatif canonique.
//  3. Pas trouvé → UPDATE thumbnail_path = NULL (le front affichera un placeholder).
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/ops"
)

type appSettings struct {
	MediaCapturesBaseDir string `json:"media_captures_base_dir"`
}

// thumbHashSuffixRe matche les suffixes hash 8-16 hex ajoutés par l'indexer
// Python legacy (ex : `_bd811d72870d`). Strip pour aligner avec les fichiers
// disque qui ne portent plus ce suffixe.
var thumbHashSuffixRe = regexp.MustCompile(`_[0-9a-f]{8,16}$`)

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
	id            int64
	playerSlug    string
	fileName      string
	thumbnailPath sql.NullString
}

// findThumb cherche un fichier miniature pour fileName dans thumbsDir.
// Stratégie : strip extension + suffixe hash → matcher les fichiers .webp ou
// .gif dont le stem (après le même strip) matche. Retourne le filename trouvé
// ou "" si aucun match.
func findThumb(thumbsDir, fileName string) string {
	stem := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	stem = thumbHashSuffixRe.ReplaceAllString(stem, "")
	entries, err := os.ReadDir(thumbsDir)
	if err != nil {
		return ""
	}
	var gifMatch string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".webp" && ext != ".gif" {
			continue
		}
		entryStem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		entryStem = thumbHashSuffixRe.ReplaceAllString(entryStem, "")
		if entryStem != stem {
			continue
		}
		if ext == ".webp" {
			return e.Name() // priorité .webp
		}
		if gifMatch == "" {
			gifMatch = e.Name()
		}
	}
	return gifMatch
}

func main() {
	dbPath := flag.String("db", "", "path vers shared_social.duckdb (requis)")
	capturesBase := flag.String("captures-base", "", "MediaCapturesBaseDir (sinon lu depuis --settings)")
	settingsPath := flag.String("settings", "app_settings.json", "path vers app_settings.json (fallback)")
	onlySlug := flag.String("slug", "", "ne traiter qu'un seul joueur (optionnel)")
	dryRun := flag.Bool("dry-run", false, "afficher les actions sans écrire")
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
	if *onlySlug != "" {
		fmt.Printf("Slug filter:   %s\n", *onlySlug)
	}
	fmt.Printf("DryRun:        %v\n\n", *dryRun)

	store := ops.MediaPathStore{CapturesBase: base}

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

	q := `SELECT id, COALESCE(player_slug, ''), COALESCE(file_name, ''), thumbnail_path
	      FROM media_files
	      WHERE kind = 'video'`
	args := []any{}
	if *onlySlug != "" {
		q += ` AND player_slug = ?`
		args = append(args, *onlySlug)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "select media_files:", err)
		os.Exit(1)
	}
	var items []mediaRow
	for rows.Next() {
		var r mediaRow
		if err := rows.Scan(&r.id, &r.playerSlug, &r.fileName, &r.thumbnailPath); err != nil {
			rows.Close()
			fmt.Fprintln(os.Stderr, "scan:", err)
			os.Exit(1)
		}
		items = append(items, r)
	}
	rows.Close()

	stats := struct {
		total, alreadyOK, relinked, nulled, ownerless, skipped int
	}{total: len(items)}

	updates := make(map[int64]sql.NullString)

	for _, r := range items {
		// Si déjà OK (path résolu existe), rien à faire.
		if r.thumbnailPath.Valid {
			if _, err := os.Stat(store.ToAbs(r.thumbnailPath.String)); err == nil {
				stats.alreadyOK++
				continue
			}
		}

		if r.playerSlug == "" {
			stats.ownerless++
			continue
		}
		if r.fileName == "" {
			stats.skipped++
			continue
		}

		thumbsDir := filepath.Join(base, r.playerSlug, "thumbs")
		found := findThumb(thumbsDir, r.fileName)
		if found == "" {
			// pas de miniature → NULL out (si actuellement non-NULL)
			if r.thumbnailPath.Valid {
				updates[r.id] = sql.NullString{Valid: false}
				stats.nulled++
			} else {
				stats.skipped++
			}
			continue
		}

		// Trouvé : stocker en path relatif canonique
		abs := filepath.Join(thumbsDir, found)
		rel := store.ToRel(abs, r.playerSlug)
		if rel == "" {
			// fallback abs si store inopérant
			rel = abs
		}
		// Skip si déjà identique à ce qu'on aurait écrit (cas rare)
		if r.thumbnailPath.Valid && r.thumbnailPath.String == rel {
			stats.alreadyOK++
			continue
		}
		updates[r.id] = sql.NullString{String: rel, Valid: true}
		stats.relinked++
	}

	if !*dryRun && len(updates) > 0 {
		tx, err := db.Begin()
		if err != nil {
			fmt.Fprintln(os.Stderr, "begin tx:", err)
			os.Exit(1)
		}
		for id, val := range updates {
			var execErr error
			if val.Valid {
				_, execErr = tx.Exec(
					`UPDATE media_files SET thumbnail_path = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
					val.String, id)
			} else {
				_, execErr = tx.Exec(
					`UPDATE media_files SET thumbnail_path = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
					id)
			}
			if execErr != nil {
				tx.Rollback() //nolint:errcheck
				fmt.Fprintln(os.Stderr, "update:", execErr)
				os.Exit(1)
			}
		}
		if err := tx.Commit(); err != nil {
			fmt.Fprintln(os.Stderr, "commit:", err)
			os.Exit(1)
		}
	}

	fmt.Println("Résultats:")
	fmt.Printf("  media_files video       : %d\n", stats.total)
	fmt.Printf("  thumbnail déjà OK       : %d\n", stats.alreadyOK)
	fmt.Printf("  relinkés                : %d\n", stats.relinked)
	fmt.Printf("  nullés (fichier absent) : %d\n", stats.nulled)
	fmt.Printf("  player_slug manquant    : %d\n", stats.ownerless)
	fmt.Printf("  skipped (rien à faire)  : %d\n", stats.skipped)
	if *dryRun {
		fmt.Println("\n(dry-run : aucune écriture)")
	}
}
