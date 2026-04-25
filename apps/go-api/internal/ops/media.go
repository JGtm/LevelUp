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
//	    BufferMin:       2,
//	    Timezone:        "Europe/Paris",
//	})
package ops

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// indexMu sérialise les IndexMedia par chemin de DB cible.
// DuckDB ne supporte pas ATTACH/DETACH concurrent sur la même instance.
var (
	indexMuMap sync.Map // map[string]*sync.Mutex
)

func indexLock(path string) func() {
	val, _ := indexMuMap.LoadOrStore(path, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// MediaIndexOptions configure l'indexation des médias.
type MediaIndexOptions struct {
	PlayerDBPath        string
	SharedSocialDBPath  string // shared_social.duckdb (cible d'écriture médias)
	SharedMatchesDBPath string // shared_matches_v2.duckdb (lecture match_registry)
	CapturesDir         string // répertoire captures du joueur
	ForceRescan         bool   // réindexer tous les fichiers même connus
	BufferMin           int    // buffer en minutes autour de [start_time, end_time] (défaut 2)
	Gamertag            string
	Timezone            string            // IANA (ex: "Europe/Paris") — SET TimeZone à l'ouverture
	CaptureTimes        map[string]*int64 // basename → unix ts client (optionnel, depuis upload)
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
// Thread-safe : sérialise les accès par chemin de DB cible (mutex par path).
func IndexMedia(opts MediaIndexOptions) (MediaIndexResult, error) {
	if opts.BufferMin <= 0 {
		opts.BufferMin = 2
	}

	// Utiliser shared_social.duckdb si disponible, sinon fallback sur stats.duckdb (transition).
	targetPath := opts.PlayerDBPath
	if opts.SharedSocialDBPath != "" {
		targetPath = opts.SharedSocialDBPath
	}

	// Sérialiser les IndexMedia sur le même fichier DB pour éviter la race
	// ATTACH/DETACH dans AssociateMediaWithMatches (duckdb-go partage l'instance).
	unlock := indexLock(targetPath)
	defer unlock()

	db, err := sql.Open("duckdb", targetPath)
	if err != nil {
		return MediaIndexResult{}, fmt.Errorf("ouverture DB: %w", err)
	}
	defer db.Close()

	// Appliquer la timezone après ouverture pour un DATEDIFF correct (DST).
	tz := SanitizeMediaTimezone(opts.Timezone)
	if tz != "" {
		if _, err := db.Exec("SET TimeZone = '" + tz + "'"); err != nil {
			slog.Warn("IndexMedia: SET TimeZone échoué, DST possiblement incorrect",
				"timezone", tz, "err", err)
		} else {
			slog.Debug("IndexMedia: SET TimeZone appliqué", "timezone", tz)
		}
	}

	// Calculer loc une seule fois pour le filename parser.
	var loc *time.Location
	if opts.Timezone != "" {
		if l, err := time.LoadLocation(opts.Timezone); err == nil {
			loc = l
		} else {
			slog.Warn("IndexMedia: timezone invalide pour filename parser",
				"timezone", opts.Timezone, "err", err)
		}
	}

	slog.Debug("IndexMedia: démarrage",
		"captures_dir", opts.CapturesDir,
		"buffer_min", opts.BufferMin,
		"timezone", opts.Timezone,
		"force_rescan", opts.ForceRescan)

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
	slog.Debug("IndexMedia: scan répertoire", "scanned", result.Scanned)

	for _, path := range mediaFiles {
		hash, err := fileHash(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: hash: %v", path, err))
			continue
		}
		if !opts.ForceRescan && known[hash] {
			continue
		}
		var clientTs *int64
		if opts.CaptureTimes != nil {
			clientTs = opts.CaptureTimes[filepath.Base(path)]
		}
		if err := insertMediaFile(db, path, hash, opts.Gamertag, clientTs, loc); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: insert: %v", path, err))
			continue
		}
		result.NewFiles++
	}

	slog.Debug("IndexMedia: scan terminé",
		"scanned", result.Scanned, "new_files", result.NewFiles)

	// Association avec les matchs
	assoc, err := AssociateMediaWithMatches(db, opts.SharedMatchesDBPath, opts.BufferMin, opts.Timezone)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("association: %v", err))
	} else {
		result.Associated = assoc
	}

	// Lier les miniatures existantes sur disque aux enregistrements dont thumbnail_path est NULL.
	thumbsDir := filepath.Join(opts.CapturesDir, "thumbs")
	if n, backfillErr := BackfillThumbnailPaths(db, opts.CapturesDir, thumbsDir); backfillErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("backfill_thumbnails: %v", backfillErr))
	} else {
		result.Thumbnails += n
	}

	slog.Info("IndexMedia: terminé",
		"scanned", result.Scanned,
		"new_files", result.NewFiles,
		"associated", result.Associated,
		"errors", len(result.Errors))

	return result, nil
}

// AssociateMediaWithMatches associe chaque média au match dans la fenêtre [start_time, end_time].
// Portage de MediaIndexer.associate_with_matches() Python.
// sharedMatchesPath : chemin vers shared_matches_v2.duckdb (peut être vide — association ignorée).
// bufferMin : buffer en minutes autour de la fenêtre du match (défaut 2).
// timezone : IANA pour SET TimeZone (nécessaire car start_time est un TIMESTAMP naïf Paris).
func AssociateMediaWithMatches(db *sql.DB, sharedMatchesPath string, bufferMin int, timezone string) (int, error) {
	if sharedMatchesPath == "" {
		return 0, nil
	}
	if bufferMin <= 0 {
		bufferMin = 2
	}

	// SET TimeZone pour interpréter correctement les TIMESTAMP naïfs de match_registry.
	if tz := SanitizeMediaTimezone(timezone); tz != "" {
		if _, err := db.Exec("SET TimeZone = '" + tz + "'"); err != nil {
			slog.Warn("AssociateMediaWithMatches: SET TimeZone échoué",
				"timezone", tz, "err", err)
		}
	}
	slog.Debug("AssociateMediaWithMatches: démarrage",
		"shared_matches_path", sharedMatchesPath,
		"buffer_min", bufferMin,
		"timezone", timezone)

	// ATTACH la DB des matchs en lecture seule pour accéder à match_registry.
	if _, err := db.Exec(fmt.Sprintf(`ATTACH '%s' AS shared_matches (READ_ONLY)`, sharedMatchesPath)); err != nil {
		return 0, fmt.Errorf("attach shared_matches: %w", err)
	}
	defer db.Exec(`DETACH shared_matches`) //nolint:errcheck

	// La capture doit se situer dans [start_time - buffer, end_time + buffer].
	// start_time et end_time sont des TIMESTAMP naïfs en heure Paris — SET TimeZone les corrige.
	q := fmt.Sprintf(`
		INSERT OR IGNORE INTO media_match_associations (media_file_id, match_id, delta_seconds)
		SELECT
			mf.id,
			mr.match_id,
			ABS(DATEDIFF('second', mf.capture_start_utc, mr.start_time)) AS delta_s
		FROM media_files mf
		JOIN shared_matches.match_registry mr
			ON mf.capture_start_utc
				BETWEEN (mr.start_time - INTERVAL '%d minutes')
				    AND (mr.end_time   + INTERVAL '%d minutes')
		WHERE mf.id NOT IN (SELECT media_file_id FROM media_match_associations)
	`, bufferMin, bufferMin)
	res, err := db.Exec(q)
	if err != nil {
		slog.Error("AssociateMediaWithMatches: erreur SQL", "err", err)
		return 0, err
	}
	n, _ := res.RowsAffected()
	slog.Info("AssociateMediaWithMatches: terminé",
		"associations_created", n, "buffer_min", bufferMin, "timezone", timezone)
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
		thumbPath := filepath.Join(thumbsDir, base+".gif")
		if _, err := os.Stat(thumbPath); err == nil {
			continue // déjà générée
		}
		srcPath := filepath.Join(videosDir, e.Name())
		if err := generateAnimatedThumbnail(srcPath, thumbPath); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Name(), err))
			continue
		}
		generated++
	}
	return generated, errs
}

// generateAnimatedThumbnail génère un GIF animé (3 s à partir de t=5s, 10 fps, 480px) via ffmpeg two-pass palette.
func generateAnimatedThumbnail(videoPath, gifPath string) error {
	// Two-pass ffmpeg : 1) générer la palette optimale, 2) l'appliquer au GIF.
	// Filtre : fps=10, scale=480:-1 (largeur 480, hauteur proportionnelle), dithering sierra2_4a.
	palette := gifPath + ".palette.png"
	defer os.Remove(palette) //nolint:errcheck

	// Pass 1 — palette
	pass1 := exec.Command("ffmpeg", "-y",
		"-ss", "5",
		"-t", "3",
		"-i", videoPath,
		"-vf", "fps=10,scale=480:-1:flags=lanczos,palettegen=stats_mode=diff",
		palette,
	)
	pass1.Stdout = nil
	pass1.Stderr = nil
	if err := pass1.Run(); err != nil {
		return fmt.Errorf("palettegen: %w", err)
	}

	// Pass 2 — GIF
	pass2 := exec.Command("ffmpeg", "-y",
		"-ss", "5",
		"-t", "3",
		"-i", videoPath,
		"-i", palette,
		"-lavfi", "fps=10,scale=480:-1:flags=lanczos[x];[x][1:v]paletteuse=dither=sierra2_4a",
		gifPath,
	)
	pass2.Stdout = nil
	pass2.Stderr = nil
	return pass2.Run()
}

// BackfillThumbnailPaths met à jour thumbnail_path en DB pour toutes les vidéos
// dont le fichier miniature existe déjà dans thumbsDir mais dont la colonne est NULL.
// Appelé après GenerateThumbnails pour lier les miniatures générées aux enregistrements.
func BackfillThumbnailPaths(db *sql.DB, videosDir, thumbsDir string) (int, error) {
	entries, err := os.ReadDir(thumbsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("BackfillThumbnailPaths ReadDir: %w", err)
	}

	updated := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(e.Name())) != ".gif" {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		thumbAbs := filepath.Join(thumbsDir, e.Name())

		// Stripper le suffixe hash (éventuellement ajouté par le Python indexer)
		// Ex: "Halo Infinite 2025-12-18 17-40-46_9430c6551833" → "Halo Infinite 2025-12-18 17-40-46"
		videoBase := thumbHashSuffixRe.ReplaceAllString(base, "")

		// Mettre à jour la vidéo dont file_name commence par videoBase (n'importe quelle extension vidéo).
		// On utilise LIKE 'base.%' pour éviter les faux positifs de préfixe.
		res, err := db.Exec(`
			UPDATE media_files
			SET thumbnail_path = ?
			WHERE thumbnail_path IS NULL
			  AND kind = 'video'
			  AND file_name LIKE ?
		`, thumbAbs, videoBase+".%")
		if err != nil {
			slog.Warn("BackfillThumbnailPaths: update échoué",
				"base", base, "err", err)
			continue
		}
		n, _ := res.RowsAffected()
		updated += int(n)
	}
	slog.Info("BackfillThumbnailPaths: terminé", "updated", updated)
	return updated, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func ensureMediaTables(db *sql.DB) error {
	if _, err := db.Exec(`CREATE SEQUENCE IF NOT EXISTS media_files_id_seq START 1`); err != nil {
		return err
	}

	// Si la table existe avec l'ancien schéma (id VARCHAR, issu de create_base_player_schema)
	// et qu'elle est vide, on la supprime pour la recréer correctement.
	if err := dropLegacyMediaFilesIfNeeded(db); err != nil {
		return err
	}

	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS media_files (
			id INTEGER PRIMARY KEY DEFAULT nextval('media_files_id_seq'),
			player_slug VARCHAR,
			file_path VARCHAR UNIQUE,
			file_name VARCHAR,
			file_hash VARCHAR,
			kind VARCHAR,
			thumbnail_path VARCHAR,
			capture_start_utc TIMESTAMPTZ,
			capture_end_utc TIMESTAMPTZ,
			duration_seconds DOUBLE,
			status VARCHAR,
			mtime TIMESTAMPTZ,
			liked BOOLEAN DEFAULT FALSE,
			liked_at TIMESTAMPTZ,
			discord_notified BOOLEAN DEFAULT FALSE,
			indexed_at TIMESTAMPTZ DEFAULT NOW()
		)
	`)
	if err != nil {
		return err
	}
	// Migration idempotente : ajoute les colonnes absentes dans les DBs créées
	// par d'anciennes migrations qui n'avaient pas capture_start_utc.
	// ADD COLUMN IF NOT EXISTS évite d'avorter la connexion.
	for _, col := range []struct{ name, typ string }{
		{"capture_start_utc", "TIMESTAMPTZ"},
		{"capture_end_utc", "TIMESTAMPTZ"},
		{"file_hash", "VARCHAR"},
		{"kind", "VARCHAR"},
		{"thumbnail_path", "VARCHAR"},
		{"player_slug", "VARCHAR"},
		{"duration_seconds", "DOUBLE"},
		{"status", "VARCHAR"},
		{"mtime", "TIMESTAMPTZ"},
		{"indexed_at", "TIMESTAMPTZ DEFAULT NOW()"},
		{"liked", "BOOLEAN DEFAULT FALSE"},
		{"liked_at", "TIMESTAMPTZ"},
		{"discord_notified", "BOOLEAN DEFAULT FALSE"},
	} {
		if _, err := db.Exec("ALTER TABLE media_files ADD COLUMN IF NOT EXISTS " + col.name + " " + col.typ); err != nil {
			return fmt.Errorf("ensureMediaTables: ajout colonne %s: %w", col.name, err)
		}
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

// dropLegacyMediaFilesIfNeeded supprime la table media_files si elle a l'ancien schéma
// (id VARCHAR, issue de create_base_player_schema) et qu'elle est vide.
// Ceci permet à ensureMediaTables de recréer la table avec le bon schéma.
func dropLegacyMediaFilesIfNeeded(db *sql.DB) error {
	// Vérifier si la colonne id est de type VARCHAR (ancien schéma).
	var dataType string
	err := db.QueryRow(
		"SELECT data_type FROM information_schema.columns WHERE table_schema = 'main' AND table_name = 'media_files' AND column_name = 'id'",
	).Scan(&dataType)
	if err != nil {
		// Table inexistante ou autre erreur → rien à faire.
		return nil //nolint:nilerr
	}
	if dataType != "VARCHAR" {
		return nil // Schéma déjà correct ou inconnu.
	}
	// Vérifier que la table est vide avant de la supprimer.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM media_files").Scan(&count); err != nil || count > 0 {
		return nil // Table non-vide ou erreur : on ne touche pas aux données.
	}
	if _, err := db.Exec("DROP TABLE media_files"); err != nil {
		return fmt.Errorf("dropLegacyMediaFilesIfNeeded: DROP TABLE: %w", err)
	}
	if _, err := db.Exec("DROP TABLE IF EXISTS media_match_associations"); err != nil {
		return fmt.Errorf("dropLegacyMediaFilesIfNeeded: DROP TABLE media_match_associations: %w", err)
	}
	return nil
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
		if err != nil {
			return err
		}
		// Ignorer le répertoire thumbs/ (miniatures générées)
		if info.IsDir() {
			if info.Name() == "thumbs" {
				return filepath.SkipDir
			}
			return nil
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

func insertMediaFile(db *sql.DB, path, hash, playerSlug string, captureTimeUnix *int64, loc *time.Location) error {
	ext := strings.ToLower(filepath.Ext(path))
	kind := supportedExtensions[ext]

	var captureAt *time.Time

	// Priorité 1 : datetime extraite du nom de fichier Xbox.
	if t := parseCaptureTimeFromFilename(filepath.Base(path), loc); t != nil {
		captureAt = t
		slog.Debug("insertMediaFile: datetime extraite du nom de fichier",
			"file", filepath.Base(path), "capture_start_utc", *t)
	}

	// Priorité 2 : mtime client fourni par le navigateur (file.lastModified).
	if captureAt == nil && captureTimeUnix != nil && *captureTimeUnix > 0 {
		t := time.Unix(*captureTimeUnix, 0).UTC()
		captureAt = &t
		slog.Debug("insertMediaFile: datetime depuis file.lastModified client",
			"file", filepath.Base(path), "capture_start_utc", t)
	}

	// Priorité 3 : mtime filesystem (fallback — incorrect sur serveur, correct en local).
	if captureAt == nil {
		fi, _ := os.Stat(path)
		if fi != nil {
			t := fi.ModTime().UTC()
			captureAt = &t
			slog.Debug("insertMediaFile: datetime depuis mtime filesystem (fallback)",
				"file", filepath.Base(path), "capture_start_utc", t)
		}
	}

	_, err := db.Exec(`
		INSERT OR IGNORE INTO media_files (player_slug, file_path, file_name, file_hash, kind, capture_start_utc)
		VALUES (?, ?, ?, ?, ?, ?)
	`, playerSlug, path, filepath.Base(path), hash, kind, captureAt)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers timezone + filename parser
// ─────────────────────────────────────────────────────────────────────────────

// SanitizeMediaTimezone valide un nom de timezone IANA pour éviter l'injection SQL.
// Retourne "" si la valeur contient des caractères non autorisés.
// Exportée pour être utilisée par service.NewMediaService.
func SanitizeMediaTimezone(tz string) string {
	for _, c := range tz {
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9':
		case c == '/' || c == '_' || c == '-' || c == '+':
		default:
			return ""
		}
	}
	return tz
}

// thumbHashSuffixRe matche le suffixe hash ajouté aux miniatures GIF.
// Ex: "Halo Infinite 2025-12-18 17-40-46_9430c6551833" → base "Halo Infinite 2025-12-18 17-40-46"
var thumbHashSuffixRe = regexp.MustCompile(`_[0-9a-fA-F]{6,}$`)

// xboxFilenameRe matche le pattern Xbox :
// "Halo Infinite 2024.11.15 - 21.30.45.01.mp4"
// Groupe 1=année 2=mois 3=jour 4=heure 5=min 6=sec
var xboxFilenameRe = regexp.MustCompile(
	`(\d{4})\.(\d{2})\.(\d{2}) - (\d{2})\.(\d{2})\.(\d{2})`)

// parseCaptureTimeFromFilename tente d'extraire la datetime depuis le nom de fichier Xbox.
// Retourne nil si aucun pattern connu n'est trouvé.
// La datetime est interprétée comme heure locale (loc), puis convertie en UTC.
func parseCaptureTimeFromFilename(name string, loc *time.Location) *time.Time {
	if loc == nil {
		return nil
	}
	if m := xboxFilenameRe.FindStringSubmatch(name); m != nil {
		year := mustAtoi(m[1])
		month := mustAtoi(m[2])
		day := mustAtoi(m[3])
		hour := mustAtoi(m[4])
		min := mustAtoi(m[5])
		sec := mustAtoi(m[6])
		if year == 0 {
			return nil
		}
		t := time.Date(year, time.Month(month), day, hour, min, sec, 0, loc).UTC()
		return &t
	}
	return nil
}

// mustAtoi convertit une string en int, retourne 0 en cas d'erreur.
func mustAtoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
