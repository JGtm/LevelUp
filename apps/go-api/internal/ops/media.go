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
	CapturesDir         string // répertoire captures du joueur ({CapturesBase}/{Gamertag})
	CapturesBase        string // racine multi-player ({CapturesBase}/{slug}/...) — vide en mode legacy
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
//
//nolint:funlen // portage fidèle de MediaIndexer.scan_and_index Python — séquentiel
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

	// Path store : convertit les chemins absolus disk en {owner_slug}/{rel}
	// stables pour stockage DB. Mode legacy si CapturesBase vide.
	store := MediaPathStore{CapturesBase: opts.CapturesBase}

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
		if err := insertMediaFile(db, path, hash, opts.Gamertag, clientTs, loc, store); err != nil {
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

	// Générer les miniatures WebP manquantes via ffmpeg, puis lier en DB.
	// Couvre tous les chemins d'indexation : upload, scan, reindex.
	thumbsDir := filepath.Join(opts.CapturesDir, "thumbs")
	if thumbN, thumbErrs := GenerateThumbnails(opts.CapturesDir, thumbsDir); thumbN > 0 || len(thumbErrs) > 0 {
		result.Thumbnails += thumbN
		for _, e := range thumbErrs {
			result.Errors = append(result.Errors, "generate_thumbnail: "+e)
		}
	}

	// Lier les miniatures présentes sur disque aux enregistrements dont thumbnail_path est NULL.
	if n, backfillErr := BackfillThumbnailPaths(db, opts.CapturesDir, thumbsDir, opts.Gamertag, store); backfillErr != nil {
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

	// La capture doit se situer dans [start_utc - buffer, end_utc + buffer].
	// start_time_utc/end_time_utc sont TIMESTAMPTZ UTC garanti (migration add_start_time_utc_to_match_registry).
	// Fallback sur AT TIME ZONE 'UTC' pour les rares matchs sans start_time_utc.
	//
	// IMPORTANT : un média = UNE seule association. Algorithme de scoring :
	//  1. Préférer un match qui CONTIENT vraiment capture_start_utc (sans buffer)
	//     — un capture pendant le match est plus probable que pendant le buffer
	//     du match précédent/suivant
	//  2. Sinon, distance au CENTRE du match (pas au début) — un replay enregistré
	//     à la fin d'un match a un delta naturel de ~match_duration/2 par rapport
	//     au début, ce qui peut le rendre plus proche du DÉBUT du match suivant
	//     que du match courant si on trie par "delta vs start_time"
	q := fmt.Sprintf(`
		INSERT OR IGNORE INTO media_match_associations (media_file_id, match_id, delta_seconds)
		SELECT media_file_id, match_id, delta_s FROM (
			SELECT
				mf.id AS media_file_id,
				mr.match_id,
				ABS(DATEDIFF('second', mf.capture_start_utc,
					COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC'))) AS delta_s,
				ROW_NUMBER() OVER (
					PARTITION BY mf.id
					ORDER BY
						CASE WHEN mf.capture_start_utc
							BETWEEN COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC')
							    AND COALESCE(mr.end_time_utc,   mr.end_time   AT TIME ZONE 'UTC')
						THEN 0 ELSE 1 END,
						ABS(DATEDIFF('second', mf.capture_start_utc,
							COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC')
							+ (COALESCE(mr.end_time_utc, mr.end_time AT TIME ZONE 'UTC')
							 - COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC')) / 2)) ASC,
						mr.match_id
				) AS rn
			FROM media_files mf
			JOIN shared_matches.match_registry mr
				ON mf.capture_start_utc
					BETWEEN (COALESCE(mr.start_time_utc, mr.start_time AT TIME ZONE 'UTC') - INTERVAL '%d minutes')
					    AND (COALESCE(mr.end_time_utc,   mr.end_time   AT TIME ZONE 'UTC') + INTERVAL '%d minutes')
			WHERE mf.id NOT IN (SELECT media_file_id FROM media_match_associations)
		) ranked
		WHERE rn = 1
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

// GenerateThumbnails génère des miniatures animées WebP pour les nouvelles vidéos.
// Les GIFs legacy déjà présents sur disque sont conservés tels quels (pas de
// regénération) — le backfill DB reconnaît les deux formats.
// Nécessite ffmpeg compilé avec libwebp dans le PATH.
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
		thumbPath := filepath.Join(thumbsDir, base+".webp")
		if _, err := os.Stat(thumbPath); err == nil {
			continue // WebP déjà généré
		}
		if _, err := os.Stat(filepath.Join(thumbsDir, base+".gif")); err == nil {
			continue // GIF legacy présent, on le garde (pas de backfill bulk)
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

// computeMediaEnd dérive (capture_end_utc, duration_seconds) à partir du kind
// et du capture_start_utc connu, sans IO :
//   - kind "image" : capture instantanée → end = start, duration = 0.
//   - kind "video" + durationKnown : end = start + duration, duration_seconds
//     = duration. Si durationKnown=false (ffprobe absent/échec), on laisse
//     end et duration_seconds à NULL — le tri retombe sur capture_start_utc.
//   - kind inconnu : tout à NULL.
//
// Isolé pour les tests unitaires : la logique de mappage end/duration est
// testable sans dépendre de ffprobe ni du filesystem.
func computeMediaEnd(kind string, captureAt *time.Time, duration float64, durationKnown bool) (captureEnd *time.Time, durationSec *float64) {
	switch kind {
	case "image":
		if captureAt != nil {
			end := *captureAt
			captureEnd = &end
		}
		zero := 0.0
		durationSec = &zero
	case "video":
		if durationKnown {
			d := duration
			durationSec = &d
			if captureAt != nil {
				end := captureAt.Add(time.Duration(d * float64(time.Second)))
				captureEnd = &end
			}
		}
	}
	return captureEnd, durationSec
}

// probeVideoDuration retourne la durée d'un fichier vidéo en secondes via
// ffprobe (livré avec ffmpeg, déjà requis pour les miniatures). Retourne 0 et
// une erreur si ffprobe est absent du PATH ou si le fichier est illisible. Le
// caller doit traiter ça comme "durée inconnue" — la durée n'est pas critique
// (juste utilisée pour capture_end_utc = capture_start_utc + duration), donc
// échec silencieux côté insert.
func probeVideoDuration(videoPath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		videoPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == "N/A" {
		return 0, fmt.Errorf("ffprobe: durée vide pour %s", videoPath)
	}
	d, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe: parse durée %q: %w", raw, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("ffprobe: durée non positive (%g) pour %s", d, videoPath)
	}
	return d, nil
}

// generateAnimatedThumbnail génère un WebP animé (3 s à partir de t=5s, 10 fps,
// 480px) via ffmpeg/libwebp en single-pass. Beaucoup plus compact qu'un GIF
// (~25-35 % de taille) et 24-bit (vs palette 8-bit du GIF) pour des couleurs
// fidèles à la source. Compatible avec gif-hover-thumbnail.tsx (Image()/canvas
// indifférents au format animé).
func generateAnimatedThumbnail(videoPath, webpPath string) error {
	cmd := exec.Command("ffmpeg", "-y",
		"-ss", "5",
		"-t", "3",
		"-i", videoPath,
		"-an",
		"-vf", "fps=10,scale=480:-1:flags=lanczos",
		"-c:v", "libwebp",
		"-loop", "0",
		"-q:v", "75",
		"-compression_level", "6",
		"-preset", "picture",
		webpPath,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// BackfillThumbnailPaths met à jour thumbnail_path en DB pour toutes les vidéos
// dont le fichier miniature existe déjà dans thumbsDir mais dont la colonne est NULL.
// Appelé après GenerateThumbnails pour lier les miniatures générées aux enregistrements.
//
// ownerSlug est utilisé pour construire le path relatif stable
// ({owner_slug}/thumbs/{filename}) qui sera stocké en DB. Si store est en mode
// legacy (CapturesBase vide), on stocke le path absolu — comportement pré-refactor.
func BackfillThumbnailPaths(db *sql.DB, videosDir, thumbsDir, ownerSlug string, store MediaPathStore) (int, error) {
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
		switch strings.ToLower(filepath.Ext(e.Name())) {
		case ".webp", ".gif":
			// formats supportés (webp = nouveau, gif = legacy conservé)
		default:
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		thumbAbs := filepath.Join(thumbsDir, e.Name())

		// Path stable à stocker en DB : relatif via store si possible, sinon abs.
		thumbStored := store.ToRel(thumbAbs, ownerSlug)
		if thumbStored == "" {
			thumbStored = thumbAbs
		}

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
		`, thumbStored, videoBase+".%")
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
		{"file_stem", "VARCHAR"},
		{"file_ext", "VARCHAR"},
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

// insertMediaFile insère ou met à jour une ligne media_files.
//
// path est le chemin absolu sur disque (requis pour ffprobe + Stat des
// fichiers existants en cas de stem conflict). Le path ECRIT en DB est
// dérivé via store.ToRel : format stable {owner_slug}/{rel_in_owner_dir}.
// Si le store est en mode legacy (CapturesBase vide ou conversion échoue),
// le path absolu est stocké tel quel — comportement pré-refactor.
func insertMediaFile(db *sql.DB, path, hash, playerSlug string, captureTimeUnix *int64, loc *time.Location, store MediaPathStore) error {
	ext := strings.ToLower(filepath.Ext(path))
	kind := supportedExtensions[ext]
	baseName := filepath.Base(path)
	stem := strings.TrimSuffix(baseName, ext)

	// Path à stocker en DB : relatif si le store peut convertir, sinon
	// fallback sur le path absolu (mode legacy).
	storedPath := store.ToRel(path, playerSlug)
	if storedPath == "" {
		storedPath = path
	}

	var captureAt *time.Time

	// Priorité 1 : datetime extraite du nom de fichier (OBS Studio, Xbox, ShadowPlay).
	if t := parseCaptureTimeFromFilename(baseName, loc); t != nil {
		captureAt = t
		slog.Debug("insertMediaFile: datetime extraite du nom de fichier",
			"file", baseName, "capture_start_utc", *t)
	}

	// Priorité 2 : mtime client fourni par le navigateur (file.lastModified).
	if captureAt == nil && captureTimeUnix != nil && *captureTimeUnix > 0 {
		t := time.Unix(*captureTimeUnix, 0).UTC()
		captureAt = &t
		slog.Debug("insertMediaFile: datetime depuis file.lastModified client",
			"file", baseName, "capture_start_utc", t)
	}

	// Le mtime filesystem du serveur n'est PAS utilisé en fallback : sur un fichier
	// uploadé/copié, il correspond à l'heure d'arrivée et fausserait l'association
	// match (média associé au match en cours pendant l'upload). Mieux vaut laisser
	// capture_start_utc NULL → le média est inséré mais non-associé tant qu'une
	// source fiable (regex nom de fichier ou ré-upload) n'est pas disponible.
	// La colonne `media_files.mtime` n'est pas non plus alimentée ici : sur un
	// fichier uploadé l'os.Stat retournerait l'heure d'écriture serveur, pas
	// l'heure de capture — un faux signal qui empirerait le tri. Le COALESCE
	// du timeOrderExpr s'appuie sur capture_start_utc (déjà fiable) en tête,
	// donc mtime resterait simplement inutilisé.
	if captureAt == nil {
		slog.Warn("insertMediaFile: capture_start_utc indéterminé, média non-associable",
			"file", baseName, "player", playerSlug)
	}

	// Durée + capture_end_utc : pour les vidéos on probe ffprobe ; pour les
	// images c'est un instantané. Logique pure isolée dans computeMediaEnd pour
	// les tests unitaires (sans ffprobe).
	var duration float64
	var durationKnown bool
	if kind == "video" {
		if d, err := probeVideoDuration(path); err == nil {
			duration = d
			durationKnown = true
		} else {
			slog.Debug("insertMediaFile: ffprobe durée indisponible",
				"file", baseName, "err", err)
		}
	}
	captureEnd, durationSec := computeMediaEnd(kind, captureAt, duration, durationKnown)

	// Dédup extension-agnostique : cherche une entrée existante avec le même stem.
	// Si trouvée et ancien fichier encore sur disque → SKIP (les deux coexistent).
	// Si trouvée mais ancien fichier parti → UPDATE (conversion terminée).
	// Sinon → INSERT.
	var existingID string
	var existingPath string
	err := db.QueryRow(`
		SELECT id, file_path FROM media_files
		WHERE player_slug = ? AND file_stem = ?
	`, playerSlug, stem).Scan(&existingID, &existingPath)

	if err == nil && existingID != "" {
		// Entrée trouvée : vérifier si l'ancien fichier existe encore.
		// existingPath peut être relatif (post-migration) ou absolu (legacy) —
		// store.ToAbs gère les deux cas.
		if _, statErr := os.Stat(store.ToAbs(existingPath)); statErr == nil {
			// Ancien fichier existe toujours → SKIP (non-déterministe pendant conversion).
			slog.Debug("insertMediaFile: stem conflict, ancien fichier toujours présent, SKIP nouveau",
				"player", playerSlug, "stem", stem, "old_path", existingPath, "new_path", storedPath)
			return nil
		}

		// Ancien fichier parti → UPDATE l'entrée existante (conversion complétée).
		// On met à jour aussi duration / capture_end_utc puisque le fichier
		// physique a changé (re-encodage) — sauf si la nouvelle valeur est NULL
		// (on garde l'ancienne dans ce cas via COALESCE).
		_, err := db.Exec(`
			UPDATE media_files
			SET file_path = ?, file_name = ?, file_ext = ?, file_hash = ?, kind = ?,
				duration_seconds = COALESCE(?, duration_seconds),
				capture_end_utc = COALESCE(?, capture_end_utc)
			WHERE id = ?
		`, storedPath, baseName, ext, hash, kind, durationSec, captureEnd, existingID)
		if err != nil {
			slog.Error("insertMediaFile: UPDATE failed for format conversion",
				"err", err, "player", playerSlug, "stem", stem, "id", existingID)
			return err
		}
		slog.Info("insertMediaFile: mise à jour fichier format conversions",
			"player", playerSlug, "stem", stem, "old_path", existingPath, "new_path", storedPath, "id", existingID)
		return nil
	}

	// Nouvelle entrée : INSERT avec file_stem + file_ext + timestamps complets
	// (capture_start_utc / capture_end_utc / duration_seconds). mtime n'est
	// volontairement pas écrit (cf. commentaire plus haut).
	// INSERT OR IGNORE évite les doublons par file_path UNIQUE (même contenus uploadé 2×).
	_, err = db.Exec(`
		INSERT OR IGNORE INTO media_files (
			player_slug, file_path, file_name, file_stem, file_ext, file_hash, kind,
			capture_start_utc, capture_end_utc, duration_seconds
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, playerSlug, storedPath, baseName, stem, ext, hash, kind, captureAt, captureEnd, durationSec)
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

// xboxFilenameRe matche le pattern Xbox / NVIDIA ShadowPlay :
// "Halo Infinite 2024.11.15 - 21.30.45.01.mp4"
// Groupe 1=année 2=mois 3=jour 4=heure 5=min 6=sec
var xboxFilenameRe = regexp.MustCompile(
	`(\d{4})\.(\d{2})\.(\d{2}) - (\d{2})\.(\d{2})\.(\d{2})`)

// obsFilenameRe matche le pattern OBS Studio par défaut (%CCYY-%MM-%DD %hh-%mm-%ss) :
// "Replay 2026-04-19 17-10-54.mp4"
// Groupe 1=année 2=mois 3=jour 4=heure 5=min 6=sec
var obsFilenameRe = regexp.MustCompile(
	`(\d{4})-(\d{2})-(\d{2}) (\d{2})-(\d{2})-(\d{2})`)

// captureTimeRegexes liste les patterns de noms de fichiers reconnus,
// du plus spécifique au plus générique. L'ordre importe peu en pratique
// (les patterns ne se chevauchent pas) mais OBS arrive en premier car
// c'est le format le plus fréquent dans nos captures.
var captureTimeRegexes = []*regexp.Regexp{obsFilenameRe, xboxFilenameRe}

// parseCaptureTimeFromFilename tente d'extraire la datetime depuis le nom de fichier.
// Formats supportés : OBS Studio, Xbox / NVIDIA ShadowPlay.
// Retourne nil si aucun pattern connu n'est trouvé ou si loc est nil.
// La datetime est interprétée comme heure locale (loc), puis convertie en UTC.
func parseCaptureTimeFromFilename(name string, loc *time.Location) *time.Time {
	if loc == nil {
		return nil
	}
	for _, re := range captureTimeRegexes {
		m := re.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		year := mustAtoi(m[1])
		month := mustAtoi(m[2])
		day := mustAtoi(m[3])
		hour := mustAtoi(m[4])
		min := mustAtoi(m[5])
		sec := mustAtoi(m[6])
		if year == 0 {
			continue
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
