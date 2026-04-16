// Package ops — media.go : indexation et association des médias (vidéos/captures).
//
// Portage de src/data/media_indexer.py + scripts/index_media.py (Python).
//
// Usage :
//
//	result, err := IndexMedia(MediaIndexOptions{
//	    PlayerDBPath:    "data/players/SpartanB/stats.duckdb",
//	    CapturesDir:     "data/players/SpartanB/captures",
//	    ForceRescan:     false,
//	    ToleranceMin:    5,
//	})
package ops

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// MediaIndexOptions configure l'indexation des médias.
type MediaIndexOptions struct {
	PlayerDBPath string
	CapturesDir  string // répertoire captures du joueur
	ForceRescan  bool   // réindexer tous les fichiers même connus
	ToleranceMin int    // tolérance d'association match en minutes (défaut 5)
	Gamertag     string
}

// MediaIndexResult résume le résultat de l'indexation.
type MediaIndexResult struct {
	Scanned    int
	NewFiles   int
	Associated int
	Thumbnails int
	Errors     []string
}

// supportedExtensions sont les formats vidéo/image reconnus.
var supportedExtensions = map[string]string{
	".mp4": "video", ".mov": "video", ".avi": "video",
	".mkv": "video", ".webm": "video",
	".png": "image", ".jpg": "image", ".jpeg": "image",
	".bmp": "image", ".gif": "image",
}

// ─────────────────────────────────────────────────────────────────────────────
// Indexation
// ─────────────────────────────────────────────────────────────────────────────

// IndexMedia scanne le répertoire captures et indexe les nouveaux fichiers.
// Portage de MediaIndexer.scan_and_index() Python.
func IndexMedia(opts MediaIndexOptions) (MediaIndexResult, error) {
	if opts.ToleranceMin == 0 {
		opts.ToleranceMin = 5
	}

	db, err := sql.Open("duckdb", opts.PlayerDBPath)
	if err != nil {
		return MediaIndexResult{}, fmt.Errorf("ouverture player DB: %w", err)
	}
	defer db.Close()

	if err := ensureMediaTables(db); err != nil {
		return MediaIndexResult{}, err
	}

	known, err := loadKnownHashes(db)
	if err != nil {
		return MediaIndexResult{}, err
	}

	result := MediaIndexResult{}
	mediaFiles, err := walkMediaDir(opts.CapturesDir)
	if err != nil {
		return MediaIndexResult{}, fmt.Errorf("scan répertoire: %w", err)
	}
	result.Scanned = len(mediaFiles)

	for _, path := range mediaFiles {
		hash, err := fileHash(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: hash: %v", path, err))
			continue
		}
		if !opts.ForceRescan && known[hash] {
			continue
		}
		if err := insertMediaFile(db, path, hash); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: insert: %v", path, err))
			continue
		}
		result.NewFiles++
	}

	// Association avec les matchs
	assoc, err := AssociateMediaWithMatches(db, opts.ToleranceMin)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("association: %v", err))
	} else {
		result.Associated = assoc
	}

	return result, nil
}

// AssociateMediaWithMatches associe chaque média au match le plus proche en temps.
// Portage de MediaIndexer.associate_with_matches() Python.
func AssociateMediaWithMatches(db *sql.DB, toleranceMin int) (int, error) {
	// Requête DuckDB : rapprocher par timestamp
	q := fmt.Sprintf(`
		INSERT OR IGNORE INTO media_match_associations (media_file_id, match_id, delta_seconds)
		SELECT
			mf.id,
			mr.match_id,
			ABS(DATEDIFF('second', mf.capture_start_utc, mr.start_time)) AS delta_s
		FROM media_files mf
		JOIN match_registry mr
		    ON ABS(DATEDIFF('minute', mf.capture_start_utc, mr.start_time)) <= %d
		WHERE mf.id NOT IN (SELECT media_file_id FROM media_match_associations)
	`, toleranceMin)
	res, err := db.Exec(q)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Génération de miniatures (ffprobe/ffmpeg)
// ─────────────────────────────────────────────────────────────────────────────

// GenerateThumbnails génère des miniatures pour les nouvelles vidéos.
// Portage de MediaIndexer.generate_thumbnails_for_new() Python.
// Nécessite ffmpeg dans le PATH.
func GenerateThumbnails(videosDir, thumbsDir string) (int, []string) {
	os.MkdirAll(thumbsDir, 0o755) //nolint:errcheck
	generated := 0
	var errs []string

	entries, err := os.ReadDir(videosDir)
	if err != nil {
		return 0, []string{fmt.Sprintf("ReadDir: %v", err)}
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if supportedExtensions[ext] != "video" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		thumbPath := filepath.Join(thumbsDir, base+".jpg")
		if _, err := os.Stat(thumbPath); err == nil {
			continue // déjà générée
		}
		srcPath := filepath.Join(videosDir, e.Name())
		if err := generateThumbnail(srcPath, thumbPath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		generated++
	}
	return generated, errs
}

// generateThumbnail génère une miniature à t=5s via ffmpeg.
func generateThumbnail(videoPath, thumbPath string) error {
	cmd := exec.Command("ffmpeg", "-y",
		"-ss", "5",
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", "2",
		thumbPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func ensureMediaTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS media_files (
			id INTEGER PRIMARY KEY,
			file_path VARCHAR UNIQUE,
			file_hash VARCHAR,
			kind VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			discord_notified BOOLEAN DEFAULT FALSE,
			indexed_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS media_match_associations (
			media_file_id INTEGER,
			match_id VARCHAR,
			delta_seconds INTEGER,
			PRIMARY KEY (media_file_id, match_id)
		)
	`)
	return err
}

func loadKnownHashes(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT file_hash FROM media_files WHERE file_hash IS NOT NULL")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	known := make(map[string]bool)
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			continue
		}
		known[h] = true
	}
	return known, rows.Err()
}

func walkMediaDir(dir string) ([]string, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := supportedExtensions[ext]; ok {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16], nil
}

func insertMediaFile(db *sql.DB, path, hash string) error {
	ext := strings.ToLower(filepath.Ext(path))
	kind := supportedExtensions[ext]
	fi, _ := os.Stat(path)
	var captureAt *time.Time
	if fi != nil {
		t := fi.ModTime().UTC()
		captureAt = &t
	}
	_, err := db.Exec(`
		INSERT OR IGNORE INTO media_files (file_path, file_hash, kind, capture_start_utc)
		VALUES (?, ?, ?, ?)
	`, path, hash, kind, captureAt)
	return err
}
