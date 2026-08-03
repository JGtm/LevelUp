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
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// Kinds de média indexés dans media_files.kind. Aussi utilisés dans la
// classification des extensions (supportedExtensions) — centralisés ici.
const (
	mediaKindVideo = "video"
	mediaKindImage = "image"
)

// Extensions video reconnues. Centralisées pour réduire la duplication entre
// supportedExtensions (indexation prod) et classifyMediaKind (seed_demo).
const (
	extMP4  = ".mp4"
	extMOV  = ".mov"
	extAVI  = ".avi"
	extMKV  = ".mkv"
	extWEBM = ".webm"
)

// Types de colonne DuckDB récurrents dans les ALTER TABLE ADD COLUMN
// de cette indexation. Centralisés pour réduire le bruit goconst.
const (
	colTypeVarchar     = "VARCHAR"
	colTypeTimestampTZ = "TIMESTAMPTZ"
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
	// TitleSlug : titre pour lequel on indexe — active le routage DEC-8 : les fichiers
	// dont le nom matche un préfixe revendiqué par un AUTRE titre
	// (title.Registry.ForeignMediaFilenamePrefixes, déclaré dans les title.toml) sont
	// SAUTÉS (un clip Halo_5_Guardians-* n'est plus indexé sous halo_infinite).
	// Vide = pas de routage (comportement historique).
	TitleSlug string
	// ForeignFilenamePrefixes : override des préfixes étrangers (tests). nil = résolus
	// depuis TitleSlug via title.DefaultRegistry().
	ForeignFilenamePrefixes []string
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
	extMP4: mediaKindVideo, extMOV: mediaKindVideo, extAVI: mediaKindVideo,
	extMKV: mediaKindVideo, extWEBM: mediaKindVideo,
	".png": mediaKindImage, ".jpg": mediaKindImage, ".jpeg": mediaKindImage,
	".bmp": mediaKindImage, ".gif": mediaKindImage,
}

// ─────────────────────────────────────────────────────────────────────────────
// Indexation
// ─────────────────────────────────────────────────────────────────────────────

// IndexMedia scanne le répertoire captures et indexe les nouveaux fichiers.
// Portage de MediaIndexer.scan_and_index() Python.
// Thread-safe : sérialise les accès par chemin de DB cible (mutex par path).
//
//nolint:funlen // portage fidèle de MediaIndexer.scan_and_index Python — séquentiel
func IndexMedia(ctx context.Context, opts MediaIndexOptions) (MediaIndexResult, error) {
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

	// ROOT CAUSE FIX (2026-05-25) : réutiliser le handle du pool process-wide
	// au lieu d'ouvrir une connexion directe via sql.Open. Sans ça :
	//   1. Le pool ouvre shared_social.duckdb via duckdb.NewConnector(path, initFn)
	//      pour appliquer SET TimeZone (cf. openSQLDBFor avec timezone != "").
	//   2. IndexMedia ouvrait via sql.Open("duckdb", path) — driver standard,
	//      pas de connecteur custom. Les opérations DDL (DROP TABLE / CREATE TABLE
	//      dans dropLegacyMediaFilesIfNeeded + ensureMediaTables) partaient dans
	//      le WAL.
	//   3. Au prochain restart serveur, le pool réouvrait avec son connecteur custom
	//      → DuckDB tentait de rejouer le WAL → INTERNAL Error :
	//      "Calling DatabaseManager::GetDefaultDatabase with no default database set"
	//      → SharedSocial = nil pour tous les joueurs → media rail vide partout.
	//
	// Fix : si la DB est déjà dans le pool (cas normal après openPlayerDB),
	// on emprunte le *sql.DB existant (sans Close — c'est le pool qui possède).
	// Sinon (cas tests ou bootstrap), fallback sur sql.Open + Close apparié.
	var db *sql.DB
	if cached, ok := platform_duckdb.LookupCachedDB(targetPath); ok {
		db = cached.SQLDb()
		slog.Debug("IndexMedia: réutilisation du handle pool", "path", targetPath)
	} else {
		var openErr error
		db, openErr = sql.Open("duckdb", targetPath)
		if openErr != nil {
			return MediaIndexResult{}, fmt.Errorf("ouverture DB: %w", openErr)
		}
		defer db.Close()
		slog.Debug("IndexMedia: handle pool absent, fallback sql.Open", "path", targetPath)
	}

	// Appliquer la timezone après ouverture pour un DATEDIFF correct (DST).
	tz := SanitizeMediaTimezone(opts.Timezone)
	if tz != "" {
		if _, err := db.ExecContext(ctx, "SET TimeZone = '"+tz+"'"); err != nil {
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

	if err := ensureMediaTables(ctx, db); err != nil {
		return MediaIndexResult{}, err
	}

	known, err := loadKnownHashes(ctx, db)
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

	// Réconciliation des orphelins : détecte les entrées DB dont le fichier
	// disque a changé d'extension sans rescan (typique d'une conversion locale
	// .mp4 → .mkv avec préservation du stem). Idempotent — exécuté en best-effort
	// avant le walk pour limiter les inserts redondants en aval.
	if reconciled, err := ReconcileOrphanedMediaFiles(ctx, db, opts.Gamertag, store); err != nil {
		slog.WarnContext(ctx, "IndexMedia: réconciliation orphelins échouée",
			"player", opts.Gamertag, "err", err)
	} else if reconciled > 0 {
		slog.InfoContext(ctx, "IndexMedia: orphelins resynced",
			"player", opts.Gamertag, "count", reconciled)
	}

	foreignPrefixes := opts.ForeignFilenamePrefixes
	if foreignPrefixes == nil && opts.TitleSlug != "" {
		foreignPrefixes = title.DefaultRegistry().ForeignMediaFilenamePrefixes(opts.TitleSlug)
	}
	skippedForeign := 0
	for _, path := range mediaFiles {
		// Routage titre (DEC-8) : un fichier revendiqué par un AUTRE titre (préfixe
		// déclaré dans son title.toml) n'est pas indexé ici — sinon chaque titre
		// indexe tous les clips du dossier captures partagé et les clips étrangers
		// restent « Sans match » à perpétuité.
		if matchesForeignPrefix(filepath.Base(path), foreignPrefixes) {
			skippedForeign++
			continue
		}
		hash, err := HashFile(path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: hash: %v", path, err))
			continue
		}
		if !opts.ForceRescan && known[hash] {
			continue
		}
		var clientTS *int64
		if opts.CaptureTimes != nil {
			clientTS = opts.CaptureTimes[filepath.Base(path)]
		}
		if err := insertMediaFile(ctx, db, path, hash, opts.Gamertag, clientTS, loc, store); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: insert: %v", path, err))
			continue
		}
		result.NewFiles++
	}

	slog.Debug("IndexMedia: scan terminé",
		"scanned", result.Scanned, "new_files", result.NewFiles,
		"skipped_foreign_title", skippedForeign)

	// Association avec les matchs
	assoc, err := AssociateMediaWithMatches(ctx, db, opts.SharedMatchesDBPath, opts.BufferMin, opts.Timezone)
	if err != nil {
		// Échec NON-fatal mais explicitement loggué en ERROR : un échec silencieux
		// laissait des médias sans match sans aucun signal (incident 2026-06-03 —
		// upload concurrent d'un reindex, conflit de lock sur shared_matches).
		result.Errors = append(result.Errors, fmt.Sprintf("association: %v", err))
		slog.ErrorContext(ctx, "IndexMedia: association média↔match échouée (médias non-associés)",
			"player", opts.Gamertag, "shared_matches", opts.SharedMatchesDBPath, "err", err)
	} else {
		result.Associated = assoc
	}

	// Générer les miniatures WebP manquantes via ffmpeg, puis lier en DB.
	// Couvre tous les chemins d'indexation : upload, scan, reindex.
	thumbsDir := filepath.Join(opts.CapturesDir, "thumbs")
	if thumbN, thumbErrs := GenerateThumbnails(ctx, opts.CapturesDir, thumbsDir); thumbN > 0 || len(thumbErrs) > 0 {
		result.Thumbnails += thumbN
		for _, e := range thumbErrs {
			result.Errors = append(result.Errors, "generate_thumbnail: "+e)
		}
	}

	// Lier les miniatures présentes sur disque aux enregistrements dont thumbnail_path est NULL.
	if n, backfillErr := BackfillThumbnailPaths(ctx, db, opts.CapturesDir, thumbsDir, opts.Gamertag, store); backfillErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("backfill_thumbnails: %v", backfillErr))
	} else {
		result.Thumbnails += n
	}

	// CHECKPOINT explicite : vide le WAL DuckDB sur disque immédiatement
	// après l'indexation. Sans ça, le WAL accumulait les writes (INSERT,
	// UPDATE thumbnail, INSERT association) jusqu'au prochain checkpoint
	// automatique (qui dépend de la taille du WAL) ou jusqu'à la fermeture
	// du process. Si Air kill brutalement (Windows SIGKILL) avant ce
	// CHECKPOINT, le WAL restait avec des writes non-checkpointed et au
	// reboot DuckDB tentait de les rejouer → bug upstream #7659 :
	// "INTERNAL Error: Failure while replaying WAL file: Calling
	// DatabaseManager::GetDefaultDatabase".
	//
	// ADR 0021 Phase 3.2 — passé en erreur dure : un CHECKPOINT échoué après
	// une indexation BULK (centaines de INSERT/UPDATE potentiels) laisse une
	// fenêtre d'exposition WAL trop large. Mieux vaut faire échouer l'opération
	// (l'utilisateur retry) que continuer avec un WAL en sursis.
	if _, ckptErr := db.ExecContext(ctx, "CHECKPOINT"); ckptErr != nil {
		slog.ErrorContext(ctx, "IndexMedia: CHECKPOINT échoué — abandon (ADR 0021)",
			"path", targetPath, "err", ckptErr)
		return result, fmt.Errorf("IndexMedia CHECKPOINT post-write: %w", ckptErr)
	}

	slog.Info("IndexMedia: terminé",
		"scanned", result.Scanned,
		"new_files", result.NewFiles,
		"associated", result.Associated,
		"errors", len(result.Errors))

	return result, nil
}

// matchesForeignPrefix indique si le basename matche un préfixe revendiqué par un
// autre titre (comparaison insensible à la casse — noms Windows). DEC-8.
func matchesForeignPrefix(basename string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return false
	}
	lower := strings.ToLower(basename)
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if strings.HasPrefix(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// AssociateMediaWithMatches associe chaque média au match dans la fenêtre
// [start_time, end_time]. Orchestre les 4 helpers de media_associate.go :
// chargement matchs (RO autonome) + chargement médias candidats + algorithme
// pur Go + bulk insert.
//
// IMPORTANT : cette fonction ne fait PLUS de `ATTACH` cross-DB (contrairement
// à l'ancienne version). Le ATTACH écrivait dans le WAL de shared_social une
// entrée non-rejouable au reboot (bug DuckDB #7659), provoquant un crash
// "Calling DatabaseManager::GetDefaultDatabase with no default database set"
// à chaque restart serveur post-indexation. Fix permanent : faire la jointure
// cross-DB côté Go (cf. media_associate.go pour l'algorithme).
//
// sharedMatchesPath : chemin vers shared_matches_v2.duckdb (vide = no-op).
// bufferMin : buffer en minutes autour de la fenêtre du match (défaut 2).
// timezone : DEPRECATED — conservé pour rétrocompat de signature mais plus
// utilisé (les TIMESTAMPTZ sont déjà en UTC, plus besoin de SET TimeZone).
func AssociateMediaWithMatches(ctx context.Context, db *sql.DB, sharedMatchesPath string, bufferMin int, timezone string) (int, error) {
	if sharedMatchesPath == "" {
		return 0, nil
	}
	if bufferMin <= 0 {
		bufferMin = 2
	}

	slog.Debug("AssociateMediaWithMatches: démarrage",
		"shared_matches_path", sharedMatchesPath,
		"buffer_min", bufferMin)

	matches, err := loadMatchTimeWindows(ctx, sharedMatchesPath)
	if err != nil {
		return 0, fmt.Errorf("load match windows: %w", err)
	}

	media, err := loadUnassociatedMedia(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("load unassociated media: %w", err)
	}

	assocs := computeAssociations(media, matches, bufferMin)

	n, err := bulkInsertAssociations(ctx, db, assocs)
	if err != nil {
		return 0, fmt.Errorf("bulk insert associations: %w", err)
	}

	slog.Info("AssociateMediaWithMatches: terminé",
		"associations_created", n,
		"matches_loaded", len(matches),
		"media_candidates", len(media),
		"buffer_min", bufferMin)
	_ = timezone // intentionnellement non-utilisé (cf. docstring)
	return n, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Génération de miniatures (ffprobe/ffmpeg)
// ─────────────────────────────────────────────────────────────────────────────

// GenerateThumbnails génère des miniatures animées WebP pour les nouvelles vidéos.
// Les GIFs legacy déjà présents sur disque sont conservés tels quels (pas de
// regénération) — le backfill DB reconnaît les deux formats.
// Nécessite ffmpeg compilé avec libwebp dans le PATH.
func insertMediaFile(ctx context.Context, db *sql.DB, path, hash, playerSlug string, captureTimeUnix *int64, loc *time.Location, store MediaPathStore) error {
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

	// Priorité 3 : mtime serveur (fallback). Pour les fichiers scannés depuis
	// disque (post-sync hook, ScanAllMedia) sans pattern reconnaissable et sans
	// timestamp client, le mtime filesystem reste l'approximation la plus fiable
	// — OBS / Xbox / ShadowPlay n'écrivent qu'à la fin de l'enregistrement et ne
	// retouchent plus le fichier ensuite. Pour les uploads HTTP, ce code est
	// rarement atteint puisque le navigateur envoie `file.lastModified` via
	// capture_times (cf. priorité 2, web/features/media/queries.ts:302) ; le
	// mtime de la copie serveur serait moins précis mais reste mieux que NULL.
	if captureAt == nil {
		if info, err := os.Stat(path); err == nil {
			t := info.ModTime().UTC()
			captureAt = &t
			slog.Debug("insertMediaFile: datetime depuis mtime serveur (fallback prio 3)",
				"file", baseName, "capture_start_utc", t)
		}
	}

	if captureAt == nil {
		slog.Warn("insertMediaFile: capture_start_utc indéterminé, média non-associable",
			"file", baseName, "player", playerSlug)
	}

	// Durée + capture_end_utc : pour les vidéos on probe ffprobe ; pour les
	// images c'est un instantané. Logique pure isolée dans computeMediaEnd pour
	// les tests unitaires (sans ffprobe).
	var duration float64
	var durationKnown bool
	if kind == mediaKindVideo {
		if d, err := probeVideoDuration(ctx, path); err == nil {
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
	err := db.QueryRowContext(ctx, `
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
		// RÉSURRECTION (item 3.1) : si la ligne visée avait été supprimée, le
		// fichier vient d'être re-déposé — la suppression est annulée en remettant
		// status à NULL. Le CASE est indispensable : écrire NULL inconditionnellement
		// dégraderait une ligne 'active' (valeur exigée par le rail home).
		_, err := db.ExecContext(ctx, `
			UPDATE media_files
			SET file_path = ?, file_name = ?, file_ext = ?, file_hash = ?, kind = ?,
				duration_seconds = COALESCE(?, duration_seconds),
				capture_end_utc = COALESCE(?, capture_end_utc),
				status = CASE WHEN COALESCE(status, '') = ? THEN NULL ELSE status END
			WHERE id = ?
		`, storedPath, baseName, ext, hash, kind, durationSec, captureEnd,
			domain.MediaStatusDeleted, existingID)
		if err != nil {
			slog.ErrorContext(ctx, "insertMediaFile: UPDATE failed for format conversion",
				"err", err, "player", playerSlug, "stem", stem, "id", existingID)
			return err
		}
		slog.Info("insertMediaFile: mise à jour fichier format conversions",
			"player", playerSlug, "stem", stem, "old_path", existingPath, "new_path", storedPath, "id", existingID)
		return nil
	}

	// Nouvelle entrée. Dédup applicative file_path : l'ex-contrainte UNIQUE(file_path)
	// a été retirée pour éradiquer le bug ART DuckDB #23046 (file_path est muté par
	// conversion/HLS/reconcile → muter une colonne indexée ART = FATAL invalidated,
	// blast MAX shared_social). On reproduit la dédup en applicatif (SELECT-then-INSERT) :
	// skip si une ligne porte déjà ce file_path. Race-safe car insertMediaFile tourne sous
	// indexLock(dbPath) (IndexMedia sérialisé par chemin DB). mtime non écrit (cf. supra).
	//
	// RÉSURRECTION (item 3.1) : un doublon SUPPRIMÉ n'est pas un skip — c'est un
	// média que l'utilisateur re-dépose. On rend la ligne existante visible au
	// lieu d'en insérer une seconde portant le même file_path (deux lignes
	// concurrentes rendraient la dédup applicative de la galerie ambiguë).
	var dupID int64
	var dupStatus sql.NullString
	switch err := db.QueryRowContext(ctx,
		`SELECT id, status FROM media_files WHERE file_path = ? LIMIT 1`,
		storedPath).Scan(&dupID, &dupStatus); err {
	case nil:
		if dupStatus.Valid && dupStatus.String == domain.MediaStatusDeleted {
			if _, err := db.ExecContext(ctx,
				`UPDATE media_files SET status = NULL WHERE id = ?`, dupID); err != nil {
				slog.ErrorContext(ctx, "insertMediaFile: résurrection du média supprimé échouée",
					"err", err, "player", playerSlug, "file_path", storedPath, "id", dupID)
				return err
			}
			slog.InfoContext(ctx, "insertMediaFile: média supprimé ressuscité par ré-upload",
				"player", playerSlug, "file_path", storedPath, "id", dupID)
			return nil
		}
		slog.Debug("insertMediaFile: file_path déjà indexé, SKIP",
			"player", playerSlug, "file_path", storedPath)
		return nil
	case sql.ErrNoRows:
		// pas de doublon → INSERT ci-dessous
	default:
		return err
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO media_files (
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
