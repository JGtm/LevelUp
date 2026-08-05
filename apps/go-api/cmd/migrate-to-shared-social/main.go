// cmd/migrate-to-shared-social — migration one-shot des données médias et likes
// vers shared_social.duckdb.
//
// À exécuter UNE SEULE FOIS après la mise à jour vers la nouvelle architecture.
// Idempotent : les données déjà présentes sont ignorées (INSERT OR IGNORE).
//
// Usage :
//
//	cd apps/go-api && go run ./cmd/migrate-to-shared-social [--repo-root /path/to/repo] [--dry-run]
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	_ "github.com/duckdb/duckdb-go/v2"

	titlePkg "levelup/go-api/internal/domain/title"
)

func main() {
	repoRoot := flag.String("repo-root", autoDetectRepoRoot(), "Racine du repo LevelUp")
	dryRun := flag.Bool("dry-run", false, "Afficher les actions sans les exécuter")
	flag.Parse()

	setupLogging()

	slog.Info("migration vers shared_social.duckdb",
		"repo_root", *repoRoot,
		"dry_run", *dryRun,
	)

	if err := run(*repoRoot, *dryRun); err != nil {
		slog.Error("migration échouée", "err", err)
		os.Exit(1)
	}
	slog.Info("migration terminée avec succès")
}

func run(repoRoot string, dryRun bool) error {
	pr := titlePkg.NewPathResolver(repoRoot)
	socialPath := pr.SharedSocialDBPath(titlePkg.DefaultSlug)
	sharedPath := pr.SharedDBPath(titlePkg.DefaultSlug)
	profilesPath := filepath.Join(repoRoot, "db_profiles.json")

	// Ouvrir la DB cible en lecture-écriture.
	socialDB, err := sql.Open("duckdb", socialPath)
	if err != nil {
		return fmt.Errorf("ouverture shared_social: %w", err)
	}
	defer socialDB.Close()

	if err := socialDB.Ping(); err != nil {
		return fmt.Errorf("ping shared_social: %w", err)
	}

	// Charger les profils joueurs.
	profiles, err := loadProfiles(profilesPath)
	if err != nil {
		return fmt.Errorf("lecture db_profiles.json: %w", err)
	}

	totalFiles := 0
	totalAssoc := 0

	// Pour chaque joueur : migrer media_files + media_match_associations.
	for gamertag, profile := range profiles {
		dbPath := resolveDBPath(repoRoot, profile)
		if dbPath == "" {
			slog.Warn("chemin stats.duckdb introuvable", "gamertag", gamertag)
			continue
		}
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			slog.Info("stats.duckdb absent, skip", "gamertag", gamertag, "path", dbPath)
			continue
		}

		slog.Info("migration joueur", "gamertag", gamertag, "db", dbPath)
		nFiles, nAssoc, err := migratePlayerMedia(socialDB, dbPath, gamertag, dryRun)
		if err != nil {
			slog.Warn("erreur migration joueur", "gamertag", gamertag, "err", err)
			continue
		}
		totalFiles += nFiles
		totalAssoc += nAssoc
		slog.Info("joueur migré", "gamertag", gamertag, "media_files", nFiles, "associations", nAssoc)
	}

	// Migrer media_likes depuis shared_matches_v2.
	if _, err := os.Stat(sharedPath); err == nil {
		nLikes, err := migrateMediaLikes(socialDB, sharedPath, dryRun)
		if err != nil {
			slog.Warn("erreur migration media_likes", "err", err)
		} else {
			slog.Info("media_likes migrés", "count", nLikes)
		}
	} else {
		slog.Info("shared_matches_v2.duckdb absent, pas de migration media_likes")
	}

	slog.Info("bilan global", "total_media_files", totalFiles, "total_associations", totalAssoc)
	return nil
}

// migratePlayerMedia copie media_files et media_match_associations d'un joueur vers shared_social.
//
//nolint:funlen,gocyclo // pipeline de migration séquentiel : open, scan, transform, insert, close
func migratePlayerMedia(dst *sql.DB, srcPath, gamertag string, dryRun bool) (int, int, error) {
	src, err := sql.Open("duckdb", srcPath+"?access_mode=read_only")
	if err != nil {
		return 0, 0, fmt.Errorf("ouverture stats.duckdb: %w", err)
	}
	defer src.Close()

	// Vérifier que media_files existe dans cette DB.
	var tableCount int
	err = src.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'media_files'",
	).Scan(&tableCount)
	if err != nil || tableCount == 0 {
		return 0, 0, nil // Table absente, rien à migrer.
	}

	rows, err := src.Query(`
		SELECT
			COALESCE(CAST(id AS VARCHAR), file_path),
			file_path,
			COALESCE(
				COALESCE(
					(SELECT column_name FROM information_schema.columns
					WHERE table_name='media_files' AND column_name='file_name' LIMIT 1),
					NULL
				), ''
			),
			COALESCE(kind, 'video'),
			file_hash,
			indexed_at
		FROM media_files
		WHERE file_path IS NOT NULL
	`)
	if err != nil {
		return 0, 0, fmt.Errorf("lecture media_files: %w", err)
	}
	defer rows.Close()

	nFiles := 0
	for rows.Next() {
		var id, filePath, kind string
		var fileHash sql.NullString
		var fileName string
		var indexedAt sql.NullTime

		if err := rows.Scan(&id, &filePath, &fileName, &kind, &fileHash, &indexedAt); err != nil {
			slog.Warn("scan media_files", "err", err)
			continue
		}
		if fileName == "" {
			fileName = filepath.Base(filePath)
		}
		if !dryRun {
			_, err = dst.Exec(`
				INSERT OR IGNORE INTO media_files
					(id, player_slug, file_path, file_name, kind, file_hash, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, id, gamertag, filePath, fileName, kind, fileHash, indexedAt,
			)
			if err != nil {
				slog.Warn("insert media_files", "file_path", filePath, "err", err)
				continue
			}
		}
		nFiles++
	}
	if err := rows.Err(); err != nil {
		return nFiles, 0, err
	}

	// media_match_associations
	var assocCount int
	err = src.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'media_match_associations'",
	).Scan(&assocCount)
	if err != nil || assocCount == 0 {
		return nFiles, 0, nil
	}

	assocRows, err := src.Query(`
		SELECT CAST(media_file_id AS VARCHAR), match_id, delta_seconds FROM media_match_associations
	`)
	if err != nil {
		return nFiles, 0, fmt.Errorf("lecture media_match_associations: %w", err)
	}
	defer assocRows.Close()

	nAssoc := 0
	for assocRows.Next() {
		var mediaFileID, matchID string
		var delta sql.NullInt64
		if err := assocRows.Scan(&mediaFileID, &matchID, &delta); err != nil {
			continue
		}
		if !dryRun {
			_, _ = dst.Exec(`
				INSERT OR IGNORE INTO media_match_associations (media_file_id, match_id, delta_seconds)
				VALUES (?, ?, ?)
			`, mediaFileID, matchID, delta)
		}
		nAssoc++
	}
	return nFiles, nAssoc, assocRows.Err()
}

// migrateMediaLikes copie media_likes depuis shared_matches_v2 vers shared_social.
func migrateMediaLikes(dst *sql.DB, sharedPath string, dryRun bool) (int, error) {
	src, err := sql.Open("duckdb", sharedPath+"?access_mode=read_only")
	if err != nil {
		return 0, fmt.Errorf("ouverture shared_matches_v2: %w", err)
	}
	defer src.Close()

	var tableCount int
	if err := src.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'media_likes'",
	).Scan(&tableCount); err != nil || tableCount == 0 {
		return 0, nil
	}

	rows, err := src.Query(`SELECT media_path, liker_slug, liker_gamertag, liked_at FROM media_likes`)
	if err != nil {
		return 0, fmt.Errorf("lecture media_likes: %w", err)
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var mediaPath, likerSlug string
		var likerGamertag sql.NullString
		var likedAt sql.NullTime
		if err := rows.Scan(&mediaPath, &likerSlug, &likerGamertag, &likedAt); err != nil {
			continue
		}
		if !dryRun {
			_, _ = dst.Exec(`
				INSERT OR IGNORE INTO media_likes (media_path, liker_slug, liker_gamertag, liked_at)
				VALUES (?, ?, ?, ?)
			`, mediaPath, likerSlug, likerGamertag, likedAt)
		}
		n++
	}
	return n, rows.Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

type profileEntry struct {
	DBPath   string `json:"db_path"`
	Gamertag string `json:"gamertag"`
	XUID     string `json:"xuid"`
}

type profilesFile struct {
	Profiles map[string]profileEntry `json:"profiles"`
}

func loadProfiles(path string) (map[string]profileEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pf profilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, err
	}
	return pf.Profiles, nil
}

func resolveDBPath(repoRoot string, p profileEntry) string {
	if p.DBPath != "" {
		if filepath.IsAbs(p.DBPath) {
			return p.DBPath
		}
		return filepath.Join(repoRoot, p.DBPath)
	}
	if p.Gamertag != "" {
		return titlePkg.NewPathResolver(repoRoot).PlayerDBPath(titlePkg.DefaultSlug, p.Gamertag)
	}
	return ""
}

func autoDetectRepoRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	// Remonter depuis apps/go-api/cmd/migrate-to-shared-social
	dir := filepath.Dir(exe)
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "db_profiles.json")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return "."
}

func setupLogging() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}
