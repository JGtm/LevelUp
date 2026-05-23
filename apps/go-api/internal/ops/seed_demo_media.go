// Package ops — seed_demo_media.go : extraction des médias pour la démo.
//
// Portage de scripts/prepare_demo_data.py:_extract_media + _reimport_existing_media
// (supprimés au commit c03707aa lors du nettoyage Python legacy).
//
// Stratégie :
//  1. Si media_registry.json existe déjà dans outMediaDir (regen VPS après upload
//     initial) → réimporte les entrées dans la nouvelle DB sans recopier les fichiers.
//  2. Sinon : sélectionne max N médias liés aux matchs démo (direct), fallback
//     fuzzy par carte si insuffisant. Copie fichiers + insère rows media_files +
//     media_match_associations + écrit media_registry.json pour les regens futurs.
package ops

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// mediaRegistryEntry est sérialisé dans media_registry.json. Permet aux regens
// VPS de re-lier les fichiers déjà présents sur disque sans recopie.
type mediaRegistryEntry struct {
	Filename       string  `json:"filename"`
	FileHash       string  `json:"file_hash"`
	MatchID        string  `json:"match_id"`
	MatchStartTime *string `json:"match_start_time,omitempty"`
	MapName        *string `json:"map_name,omitempty"`
}

// DDL minimal pour media_files + media_match_associations. Synchronisé avec
// src/data/media_indexer.py de l'ancien projet (cf. seed_demo_media_ddl.sql
// inline ici pour autonomie).
const mediaFilesDDL = `
CREATE TABLE IF NOT EXISTS media_files (
    file_path VARCHAR PRIMARY KEY,
    file_hash VARCHAR NOT NULL,
    file_name VARCHAR NOT NULL,
    file_size BIGINT NOT NULL,
    file_ext VARCHAR NOT NULL,
    kind VARCHAR NOT NULL,
    mtime DOUBLE NOT NULL,
    mtime_paris_epoch DOUBLE,
    thumbnail_path VARCHAR,
    thumbnail_generated_at TIMESTAMP,
    first_seen_at TIMESTAMP,
    last_scan_at TIMESTAMP,
    scan_version INTEGER,
    capture_start_utc TIMESTAMP,
    capture_end_utc TIMESTAMP,
    duration_seconds DOUBLE,
    title VARCHAR,
    status VARCHAR NOT NULL DEFAULT 'active'
)`

const mediaAssocDDL = `
CREATE TABLE IF NOT EXISTS media_match_associations (
    media_path VARCHAR NOT NULL,
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    match_start_time TIMESTAMP NOT NULL,
    association_confidence DOUBLE DEFAULT 1.0,
    associated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    map_id VARCHAR,
    map_name VARCHAR,
    PRIMARY KEY (media_path, match_id, xuid)
)`

// extractDemoMedia est le point d'entrée du sous-pipeline média.
// Retourne le nombre de fichiers réellement importés/réimportés dans outPlayerDB.
//
//nolint:gocyclo // pipeline complet média : DDL + branch reimport/extract + copy + insert.
func extractDemoMedia(
	ctx context.Context,
	srcPlayerDB, srcSharedDB, outPlayerDB, outMediaDir string,
	matchIDs []string,
	demoXUID string,
	maxMedia int,
) (int, error) {
	if err := os.MkdirAll(outMediaDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir media: %w", err)
	}
	if err := ensureMediaTablesInPlayerDB(ctx, outPlayerDB); err != nil {
		return 0, fmt.Errorf("ensure media tables: %w", err)
	}

	// Branch 1 : registry présent → réimport sans copie de fichiers.
	registryPath := filepath.Join(outMediaDir, "media_registry.json")
	if entries, err := loadMediaRegistry(registryPath); err == nil && len(entries) > 0 {
		slog.InfoContext(ctx, "seed-demo media: registry trouvé, réimport sans copie",
			"entries", len(entries))
		return reimportExistingMedia(ctx, outPlayerDB, outMediaDir, demoXUID, entries)
	}

	// Branch 2 : extraction depuis source + copie fichiers.
	srcSrc, err := sql.Open("duckdb", srcPlayerDB+"?access_mode=READ_ONLY")
	if err != nil {
		return 0, fmt.Errorf("open src player: %w", err)
	}
	defer srcSrc.Close()

	// Skip silencieux si tables média absentes.
	if !mediaTablesExist(ctx, srcSrc) {
		slog.InfoContext(ctx, "seed-demo media: tables source absentes, skip")
		return 0, nil
	}

	mapIndex, err := buildDemoMapIndex(ctx, srcSharedDB, matchIDs)
	if err != nil {
		return 0, fmt.Errorf("build map index: %w", err)
	}

	candidates, err := collectMediaCandidates(ctx, srcSrc, matchIDs, mapIndex, maxMedia)
	if err != nil {
		return 0, fmt.Errorf("collect candidates: %w", err)
	}
	if len(candidates) == 0 {
		slog.InfoContext(ctx, "seed-demo media: aucun candidat sur disque, skip")
		return 0, nil
	}

	dst, err := sql.Open("duckdb", outPlayerDB)
	if err != nil {
		return 0, fmt.Errorf("open out player: %w", err)
	}
	defer dst.Close()

	imported, registry, err := copyAndInsertMedia(ctx, srcSrc, dst, candidates, outMediaDir, demoXUID)
	if err != nil {
		return imported, fmt.Errorf("copy+insert: %w", err)
	}

	if len(registry) > 0 {
		if err := saveMediaRegistry(registryPath, registry); err != nil {
			slog.WarnContext(ctx, "seed-demo media: registry save failed", "err", err)
		}
	}
	return imported, nil
}

// ensureMediaTablesInPlayerDB ouvre la player DB et crée les 2 tables si absentes.
func ensureMediaTablesInPlayerDB(ctx context.Context, playerDBPath string) error {
	db, err := sql.Open("duckdb", playerDBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, mediaFilesDDL); err != nil {
		return fmt.Errorf("media_files DDL: %w", err)
	}
	if _, err := db.ExecContext(ctx, mediaAssocDDL); err != nil {
		return fmt.Errorf("media_match_associations DDL: %w", err)
	}
	return nil
}

// mediaTablesExist vérifie la présence des 2 tables dans la DB source.
func mediaTablesExist(ctx context.Context, db *sql.DB) bool {
	var n int
	_ = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'main'
		  AND table_name IN ('media_files', 'media_match_associations')`).Scan(&n)
	return n == 2
}

// loadMediaRegistry lit media_registry.json. Retourne err si fichier absent.
func loadMediaRegistry(path string) ([]mediaRegistryEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []mediaRegistryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// saveMediaRegistry sérialise les entries dans media_registry.json.
func saveMediaRegistry(path string, entries []mediaRegistryEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// reimportExistingMedia ré-insère les rows media_files + media_match_associations
// pour les fichiers déjà présents dans outMediaDir (use case : regen VPS après
// upload initial).
func reimportExistingMedia(
	ctx context.Context,
	playerDBPath, outMediaDir, demoXUID string,
	entries []mediaRegistryEntry,
) (int, error) {
	db, err := sql.Open("duckdb", playerDBPath)
	if err != nil {
		return 0, fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	demoMediaRoot := buildDemoMediaRoot(outMediaDir)

	imported := 0
	for _, e := range entries {
		filePath := filepath.Join(outMediaDir, e.Filename)
		stat, err := os.Stat(filePath)
		if err != nil {
			slog.WarnContext(ctx, "seed-demo media: fichier absent disque", "file", e.Filename)
			continue
		}
		newPath := demoMediaRoot + "/" + e.Filename
		ext := filepath.Ext(e.Filename)
		kind := classifyMediaKind(ext)
		mtimeUnix := float64(stat.ModTime().Unix())

		// INSERT idempotent (ON CONFLICT DO NOTHING).
		if _, err := db.ExecContext(ctx, `
			INSERT INTO media_files (file_path, file_hash, file_name, file_size, file_ext, kind, mtime, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'active')
			ON CONFLICT (file_path) DO NOTHING`,
			newPath, e.FileHash, e.Filename, stat.Size(), ext, kind, mtimeUnix); err != nil {
			slog.WarnContext(ctx, "seed-demo media: insert file failed", "file", e.Filename, "err", err)
			continue
		}

		startTime := parseRegistryTime(e.MatchStartTime)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO media_match_associations
				(media_path, match_id, xuid, match_start_time, map_name)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (media_path, match_id, xuid) DO NOTHING`,
			newPath, e.MatchID, demoXUID, startTime, derefStr(e.MapName)); err != nil {
			slog.WarnContext(ctx, "seed-demo media: insert assoc failed", "file", e.Filename, "err", err)
			continue
		}
		imported++
	}
	return imported, nil
}

// buildDemoMediaRoot retourne le chemin absolu media root tel qu'il sera lu
// par le serveur Go (LEVELUP_ROOT/data/players/DEMO/media). Fallback sur le
// chemin local outMediaDir si LEVELUP_ROOT non défini (tests).
func buildDemoMediaRoot(outMediaDir string) string {
	if root := os.Getenv("LEVELUP_ROOT"); root != "" {
		return filepath.ToSlash(filepath.Join(root, "data", "players", DefaultDemoGamertag, "media"))
	}
	return filepath.ToSlash(outMediaDir)
}

// mediaCandidate identifie un fichier média à copier + son match associé.
type mediaCandidate struct {
	SrcFilePath     string
	TargetMatchID   string
	TargetStartTime time.Time
	MapName         string
	IsFuzzy         bool // true si réassigné via map_name (pas direct)
}

// buildDemoMapIndex retourne {map_name → [(match_id, start_time), ...]} pour
// les matchs démo (utilisé pour le fallback fuzzy par carte).
func buildDemoMapIndex(ctx context.Context, sharedDBPath string, matchIDs []string) (map[string][]mediaCandidate, error) {
	db, err := sql.Open("duckdb", sharedDBPath+"?access_mode=READ_ONLY")
	if err != nil {
		return nil, fmt.Errorf("open shared: %w", err)
	}
	defer db.Close()

	idsLit := formatIDsLiteral(matchIDs)
	stmt := fmt.Sprintf(`
		SELECT match_id, map_name, start_time
		FROM match_registry
		WHERE match_id IN (%s) AND map_name IS NOT NULL`, idsLit)
	rows, err := db.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	index := make(map[string][]mediaCandidate)
	for rows.Next() {
		var mid, mname string
		var startTime time.Time
		if err := rows.Scan(&mid, &mname, &startTime); err != nil {
			return nil, err
		}
		index[mname] = append(index[mname], mediaCandidate{
			TargetMatchID:   mid,
			TargetStartTime: startTime,
			MapName:         mname,
		})
	}
	return index, rows.Err()
}

// collectMediaCandidates sélectionne jusqu'à maxMedia fichiers : priorité directe
// (médias associés à un match démo), fallback fuzzy par map_name.
func collectMediaCandidates(
	ctx context.Context,
	src *sql.DB,
	matchIDs []string,
	mapIndex map[string][]mediaCandidate,
	maxMedia int,
) ([]mediaCandidate, error) {
	idsLit := formatIDsLiteral(matchIDs)
	seen := make(map[string]bool)
	var result []mediaCandidate

	// Direct : médias liés à un match démo.
	directStmt := fmt.Sprintf(`
		SELECT DISTINCT mf.file_path, mma.match_id, mma.match_start_time, COALESCE(mma.map_name, '')
		FROM media_files mf
		JOIN media_match_associations mma ON mma.media_path = mf.file_path
		WHERE mf.status = 'active' AND mma.match_id IN (%s)
		ORDER BY mf.mtime DESC LIMIT %d`, idsLit, maxMedia)
	rows, err := src.QueryContext(ctx, directStmt)
	if err != nil {
		return nil, fmt.Errorf("direct query: %w", err)
	}
	for rows.Next() {
		var fp, mid, mname string
		var startTime time.Time
		if err := rows.Scan(&fp, &mid, &startTime, &mname); err != nil {
			rows.Close()
			return nil, err
		}
		if _, err := os.Stat(fp); err != nil {
			continue // fichier absent du disque
		}
		result = append(result, mediaCandidate{
			SrcFilePath: fp, TargetMatchID: mid, TargetStartTime: startTime, MapName: mname,
		})
		seen[fp] = true
	}
	rows.Close()
	if len(result) >= maxMedia {
		return result[:maxMedia], nil
	}

	// Fallback fuzzy : médias d'autres matchs réassignés par map_name.
	fuzzyStmt := fmt.Sprintf(`
		SELECT DISTINCT mf.file_path, COALESCE(mma.map_name, '')
		FROM media_files mf
		JOIN media_match_associations mma ON mma.media_path = mf.file_path
		WHERE mf.status = 'active' AND mma.match_id NOT IN (%s)
		  AND mma.map_name IS NOT NULL
		ORDER BY mf.mtime DESC LIMIT 200`, idsLit)
	fuzzyRows, err := src.QueryContext(ctx, fuzzyStmt)
	if err != nil {
		return result, fmt.Errorf("fuzzy query: %w", err)
	}
	defer fuzzyRows.Close()
	for fuzzyRows.Next() {
		if len(result) >= maxMedia {
			break
		}
		var fp, mname string
		if err := fuzzyRows.Scan(&fp, &mname); err != nil {
			return result, err
		}
		if seen[fp] {
			continue
		}
		if _, err := os.Stat(fp); err != nil {
			continue
		}
		demoMatches, ok := mapIndex[mname]
		if !ok || len(demoMatches) == 0 {
			continue
		}
		target := demoMatches[0]
		result = append(result, mediaCandidate{
			SrcFilePath: fp, TargetMatchID: target.TargetMatchID,
			TargetStartTime: target.TargetStartTime, MapName: mname, IsFuzzy: true,
		})
		seen[fp] = true
	}
	return result, fuzzyRows.Err()
}

// copyAndInsertMedia copie les fichiers vers outMediaDir et insère les rows
// media_files + media_match_associations dans dst. Retourne (imported, registry).
//
// aujourd'hui best-effort par-fichier (warnings loggés, pas d'erreur globale).
//
//nolint:unparam // err maintenu pour signature cohérente avec extract* siblings ;
func copyAndInsertMedia(
	ctx context.Context,
	src, dst *sql.DB,
	candidates []mediaCandidate,
	outMediaDir, demoXUID string,
) (int, []mediaRegistryEntry, error) {
	demoMediaRoot := buildDemoMediaRoot(outMediaDir)
	registry := make([]mediaRegistryEntry, 0, len(candidates))
	imported := 0

	for _, c := range candidates {
		filename := filepath.Base(c.SrcFilePath)
		newPath := demoMediaRoot + "/" + filename
		dstPath := filepath.Join(outMediaDir, filename)

		// Récupérer les colonnes complètes depuis source.
		row := src.QueryRowContext(ctx, `SELECT file_hash, file_size, file_ext, kind, mtime FROM media_files WHERE file_path = ?`, c.SrcFilePath)
		var fileHash, fileExt, kind string
		var fileSize int64
		var mtime float64
		if err := row.Scan(&fileHash, &fileSize, &fileExt, &kind, &mtime); err != nil {
			slog.WarnContext(ctx, "seed-demo media: source row introuvable", "file", filename, "err", err)
			continue
		}

		// Copie fichier physique.
		if err := copyFile(c.SrcFilePath, dstPath); err != nil {
			slog.WarnContext(ctx, "seed-demo media: copy failed", "file", filename, "err", err)
			continue
		}

		// INSERT media_files.
		if _, err := dst.ExecContext(ctx, `
			INSERT INTO media_files (file_path, file_hash, file_name, file_size, file_ext, kind, mtime, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'active')
			ON CONFLICT (file_path) DO NOTHING`,
			newPath, fileHash, filename, fileSize, fileExt, kind, mtime); err != nil {
			slog.WarnContext(ctx, "seed-demo media: insert file failed", "file", filename, "err", err)
			continue
		}

		// INSERT media_match_associations.
		var mapNameVal any
		if c.MapName != "" {
			mapNameVal = c.MapName
		}
		if _, err := dst.ExecContext(ctx, `
			INSERT INTO media_match_associations
				(media_path, match_id, xuid, match_start_time, map_name)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (media_path, match_id, xuid) DO NOTHING`,
			newPath, c.TargetMatchID, demoXUID, c.TargetStartTime, mapNameVal); err != nil {
			slog.WarnContext(ctx, "seed-demo media: insert assoc failed", "file", filename, "err", err)
			continue
		}

		entry := mediaRegistryEntry{
			Filename: filename, FileHash: fileHash, MatchID: c.TargetMatchID,
		}
		if !c.TargetStartTime.IsZero() {
			s := c.TargetStartTime.Format(time.RFC3339)
			entry.MatchStartTime = &s
		}
		if c.MapName != "" {
			mn := c.MapName
			entry.MapName = &mn
		}
		registry = append(registry, entry)
		imported++

		fuzzyLabel := "direct"
		if c.IsFuzzy {
			fuzzyLabel = "fuzzy"
		}
		slog.InfoContext(ctx, "seed-demo media: imported",
			"file", filename, "type", fuzzyLabel, "map", c.MapName)
	}
	return imported, registry, nil
}

// copyFile copie src→dst en streaming (équivalent shutil.copy2 sans préservation mtime).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// classifyMediaKind retourne mediaKindVideo pour .mp4/.mov/etc., sinon mediaKindImage.
// Réutilise les constantes définies dans media.go.
func classifyMediaKind(ext string) string {
	switch ext {
	case extMP4, extMOV, extAVI, extMKV, extWEBM:
		return mediaKindVideo
	default:
		return mediaKindImage
	}
}

// parseRegistryTime parse une chaîne RFC3339 → time.Time (zero si nil/invalide).
func parseRegistryTime(s *string) time.Time {
	if s == nil || *s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func derefStr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
