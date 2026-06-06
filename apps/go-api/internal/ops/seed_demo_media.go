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
	"strings"
	"time"

	"levelup/go-api/internal/migration"

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

// extractDemoMedia est le point d'entrée du sous-pipeline média.
// Écrit dans outSocialDB (shared_social.duckdb démo) au schéma CANONIQUE (via les
// migrations TargetSharedSocial), pas un DDL bespoke : c'est le seul schéma que le
// pipeline de lecture (Q37, mode shared_social : id/media_file_id/player_slug) sait
// lire. playerSlug = identité du DemoPlayer (filtre "mine" côté media repo).
// Retourne le nombre de fichiers réellement importés/réimportés.
//
//nolint:gocyclo // pipeline complet média : migrations + branch reimport/extract + copy + insert.
func extractDemoMedia(
	ctx context.Context,
	srcPlayerDB, srcSharedDB, outSocialDB, outMediaDir string,
	matchIDs []string,
	playerSlug string,
	maxMedia int,
) (int, error) {
	if err := os.MkdirAll(outMediaDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir media: %w", err)
	}
	// Recréer le shared_social fresh à chaque reseed (comme shared/player DBs) :
	// sinon les rows d'un reseed précédent persistent (ex. player_slug périmé) et
	// l'INSERT ON CONFLICT DO NOTHING les conserve → seed silencieusement obsolète.
	// Sous Linux, unlink d'un fichier tenu ouvert par le conteneur démo est sûr
	// (l'ancien inode survit jusqu'au force-recreate ; le seed écrit un inode neuf).
	_ = os.Remove(outSocialDB)
	_ = os.Remove(outSocialDB + ".wal")
	if err := applyMigrationsOnPath(outSocialDB, migration.TargetSharedSocial); err != nil {
		return 0, fmt.Errorf("migrations shared_social démo: %w", err)
	}

	// Branch 1 : registry présent → réimport sans copie de fichiers.
	registryPath := filepath.Join(outMediaDir, "media_registry.json")
	if entries, err := loadMediaRegistry(registryPath); err == nil && len(entries) > 0 {
		slog.InfoContext(ctx, "seed-demo media: registry trouvé, réimport sans copie",
			"entries", len(entries))
		return reimportExistingMedia(ctx, outSocialDB, outMediaDir, playerSlug, entries)
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

	dst, err := sql.Open("duckdb", outSocialDB)
	if err != nil {
		return 0, fmt.Errorf("open out shared_social: %w", err)
	}
	defer dst.Close()

	imported, registry, err := copyAndInsertMedia(ctx, srcSrc, dst, candidates, outMediaDir, playerSlug)
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
	socialDBPath, outMediaDir, playerSlug string,
	entries []mediaRegistryEntry,
) (int, error) {
	db, err := sql.Open("duckdb", socialDBPath)
	if err != nil {
		return 0, fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	imported := 0
	for _, e := range entries {
		filePath := filepath.Join(outMediaDir, e.Filename)
		stat, err := os.Stat(filePath)
		if err != nil {
			slog.WarnContext(ctx, "seed-demo media: fichier absent disque", "file", e.Filename)
			continue
		}
		ext := filepath.Ext(e.Filename)
		row := demoMediaRow{
			ID:         mediaID(e.FileHash, e.Filename),
			PlayerSlug: playerSlug,
			// file_path RELATIF (= filename) : mediaStoredPathToURL le sert tel quel
			// (/media/files/{filename}) et ServeMediaFile le résout contre
			// MediaCapturesBaseDir (réglé sur le dossier média démo, cf. demoAppSettings).
			FilePath:     e.Filename,
			FileName:     e.Filename,
			FileHash:     e.FileHash,
			FileSize:     stat.Size(),
			Kind:         classifyMediaKind(ext),
			FileStem:     strings.TrimSuffix(e.Filename, ext),
			FileExt:      ext,
			MTime:        stat.ModTime().UTC(),
			CaptureStart: nullTimeFromRegistry(e.MatchStartTime),
			MatchID:      e.MatchID,
		}
		if err := insertDemoMediaRow(ctx, db, row); err != nil {
			slog.WarnContext(ctx, "seed-demo media: insert failed", "file", e.Filename, "err", err)
			continue
		}
		imported++
	}
	return imported, nil
}

// demoMediaRow porte les champs d'une ligne média démo (schéma canonique shared_social).
type demoMediaRow struct {
	ID, PlayerSlug, FilePath, FileName, FileHash, Kind, FileStem, FileExt string
	FileSize                                                              int64
	MTime                                                                 time.Time
	CaptureStart, CaptureEnd                                              sql.NullTime
	MatchID                                                               string
}

// insertDemoMediaRow insère un média au schéma CANONIQUE shared_social (id +
// player_slug + media_file_id) + son association. Idempotent (ON CONFLICT DO NOTHING).
func insertDemoMediaRow(ctx context.Context, db *sql.DB, m demoMediaRow) error {
	if _, err := db.ExecContext(ctx, `
		INSERT INTO media_files
			(id, player_slug, file_path, file_name, kind, file_hash, file_size,
			 file_stem, file_ext, thumbnail_path, capture_start_utc, capture_end_utc,
			 mtime, indexed_at, liked, status, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,NULL,?,?,?,CURRENT_TIMESTAMP,FALSE,'active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING`,
		m.ID, m.PlayerSlug, m.FilePath, m.FileName, m.Kind, m.FileHash, m.FileSize,
		m.FileStem, m.FileExt, m.CaptureStart, m.CaptureEnd, m.MTime); err != nil {
		return fmt.Errorf("insert media_files: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO media_match_associations (media_file_id, match_id, delta_seconds, created_at, is_manual)
		VALUES (?, ?, 0, CURRENT_TIMESTAMP, FALSE)
		ON CONFLICT (media_file_id, match_id) DO NOTHING`,
		m.ID, m.MatchID); err != nil {
		return fmt.Errorf("insert media_match_associations: %w", err)
	}
	return nil
}

// mediaID dérive l'id (PK media_files) : file_hash si présent, sinon le nom de
// fichier (les médias démo ont des noms distincts).
func mediaID(fileHash, fileName string) string {
	if h := strings.TrimSpace(fileHash); h != "" {
		return h
	}
	return fileName
}

// nullTimeFromRegistry convertit le match_start_time du registry (RFC3339) en
// sql.NullTime — sert de capture_start_utc proxy au reimport (pas de capture réelle).
func nullTimeFromRegistry(s *string) sql.NullTime {
	t := parseRegistryTime(s)
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
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

	// Direct : médias liés à un match démo. Filtre format Xbox (fichiers
	// "Halo Infinite…") + priorité aux plus anciennes captures (ère Xbox).
	directStmt := fmt.Sprintf(`
		SELECT DISTINCT mf.file_path, mma.match_id, mma.match_start_time, COALESCE(mma.map_name, '')
		FROM media_files mf
		JOIN media_match_associations mma ON mma.media_path = mf.file_path
		WHERE mf.status = 'active' AND mma.match_id IN (%s)
		  AND mf.file_name LIKE 'Halo Infinite%%'
		ORDER BY mf.capture_start_utc ASC NULLS LAST, mf.mtime ASC LIMIT %d`, idsLit, maxMedia)
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

	// Fallback fuzzy : médias d'autres matchs réassignés par map_name. C'est ici
	// que les vieilles captures Xbox ("Halo Infinite…", antérieures au corpus) sont
	// rattachées aux matchs démo qui partagent la même carte. Plus anciennes d'abord.
	fuzzyStmt := fmt.Sprintf(`
		SELECT DISTINCT mf.file_path, COALESCE(mma.map_name, '')
		FROM media_files mf
		JOIN media_match_associations mma ON mma.media_path = mf.file_path
		WHERE mf.status = 'active' AND mma.match_id NOT IN (%s)
		  AND mma.map_name IS NOT NULL
		  AND mf.file_name LIKE 'Halo Infinite%%'
		ORDER BY mf.capture_start_utc ASC NULLS LAST, mf.mtime ASC LIMIT 200`, idsLit)
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
	outMediaDir, playerSlug string,
) (int, []mediaRegistryEntry, error) {
	registry := make([]mediaRegistryEntry, 0, len(candidates))
	imported := 0

	for _, c := range candidates {
		filename := filepath.Base(c.SrcFilePath)
		dstPath := filepath.Join(outMediaDir, filename)

		// Récupérer les colonnes complètes depuis source (schéma legacy player DB).
		row := src.QueryRowContext(ctx, `SELECT file_hash, file_size, file_ext, kind, mtime, capture_start_utc, capture_end_utc FROM media_files WHERE file_path = ?`, c.SrcFilePath)
		var fileHash, fileExt, kind string
		var fileSize int64
		var mtime float64
		var captureStart, captureEnd sql.NullTime
		if err := row.Scan(&fileHash, &fileSize, &fileExt, &kind, &mtime, &captureStart, &captureEnd); err != nil {
			slog.WarnContext(ctx, "seed-demo media: source row introuvable", "file", filename, "err", err)
			continue
		}

		// Copie fichier physique.
		if err := copyFile(c.SrcFilePath, dstPath); err != nil {
			slog.WarnContext(ctx, "seed-demo media: copy failed", "file", filename, "err", err)
			continue
		}

		if !captureStart.Valid && !c.TargetStartTime.IsZero() {
			captureStart = sql.NullTime{Time: c.TargetStartTime, Valid: true}
		}
		mediaRow := demoMediaRow{
			ID:           mediaID(fileHash, filename),
			PlayerSlug:   playerSlug,
			FilePath:     filename, // relatif (cf. reimport) → servi via MediaCapturesBaseDir démo
			FileName:     filename,
			FileHash:     fileHash,
			FileSize:     fileSize,
			Kind:         kind,
			FileStem:     strings.TrimSuffix(filename, fileExt),
			FileExt:      fileExt,
			MTime:        time.Unix(int64(mtime), 0).UTC(),
			CaptureStart: captureStart,
			CaptureEnd:   captureEnd,
			MatchID:      c.TargetMatchID,
		}
		if err := insertDemoMediaRow(ctx, dst, mediaRow); err != nil {
			slog.WarnContext(ctx, "seed-demo media: insert failed", "file", filename, "err", err)
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
