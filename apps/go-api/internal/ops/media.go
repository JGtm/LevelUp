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
	"time"

	"levelup/go-api/internal/domain"
	mediapkg "levelup/go-api/internal/media"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/verrous"
)

// Kinds de média indexés dans media_files.kind (mediaKindOf).
const (
	mediaKindVideo = "video"
	mediaKindImage = "image"
)

// extMP4 : extension par défaut d'un clip de démo sans extension.
const extMP4 = ".mp4"

// Types de colonne DuckDB récurrents dans les ALTER TABLE ADD COLUMN
// de cette indexation. Centralisés pour réduire le bruit goconst.
const (
	colTypeVarchar     = "VARCHAR"
	colTypeTimestampTZ = "TIMESTAMPTZ"
)

// indexMu sérialise les IndexMedia par chemin de DB cible.
// DuckDB ne supporte pas ATTACH/DETACH concurrent sur la même instance.
var indexMu verrous.Registre

func indexLock(path string) func() {
	mu := indexMu.De(path)
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
	// MissingPolicy : ce que l'indexation fait d'une fiche dont plus aucun fichier
	// n'existe (MissingIgnore par défaut : envoi, CLI, qui ne voient qu'une partie du
	// dossier).
	MissingPolicy MissingPolicy
	// SkipFreshFiles : laisser au passage suivant les fichiers modifiés depuis moins
	// d'une minute (un clip en cours d'écriture changerait d'empreinte).
	SkipFreshFiles bool
}

// MediaIndexResult résume le résultat de l'indexation.
type MediaIndexResult struct {
	Scanned    int
	NewFiles   int
	Associated int
	Thumbnails int
	Errors     []string
}

// mediaKindOf rend la nature d'un fichier d'après son extension, sans tenir compte de la
// casse : mediaKindVideo, mediaKindImage, ou "" pour un format non indexé. Les formats
// sont ceux de internal/media, source unique de l'envoi, du service et de l'indexation
// (garde-rail archlint/media_ext_list_single_source_test.go).
func mediaKindOf(ext string) string {
	switch {
	case mediapkg.IsVideoExt(ext):
		return mediaKindVideo
	case mediapkg.IsImageExt(ext):
		return mediaKindImage
	}
	return ""
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

	// ROOT CAUSE FIX (2026-05-25, re-routé 2026-08-26) : passer par le cache
	// process-wide duckdb au lieu d'ouvrir une connexion directe via sql.Open.
	// Sans ça :
	//   1. Le pool ouvre shared_social.duckdb via OpenReadWriteShared → clé de
	//      cache "rw:"+path, DSN et connecteur custom (SET TimeZone + bornes
	//      mémoire/threads, cf. openSQLDBFor).
	//   2. IndexMedia ouvrait via sql.Open("duckdb", path) — driver standard,
	//      pas de connecteur custom, HORS cache. Les opérations DDL (DROP TABLE /
	//      CREATE TABLE dans dropLegacyMediaFilesIfNeeded + ensureMediaTables)
	//      partaient dans le WAL.
	//   3. Au prochain restart serveur, le pool réouvrait avec son connecteur custom
	//      → DuckDB tentait de rejouer le WAL → INTERNAL Error :
	//      "Calling DatabaseManager::GetDefaultDatabase with no default database set"
	//      → SharedSocial = nil pour tous les joueurs → media rail vide partout.
	//
	// L'emprunt `LookupCachedDB` + fallback `sql.Open` qui corrigeait (3) laissait
	// deux trous : le fallback restait un bare connect (hors cache, hors provider
	// — anti-pattern n°5 CLAUDE.md), et l'emprunt était NON POSSÉDANT (aucun
	// refCount : le propriétaire pouvait fermer le handle en cours d'indexation).
	//
	// `OpenReadWriteShared` couvre les DEUX branches d'un seul appel : cache hit →
	// réutilisation du handle du pool avec refCount++ (le Close différé ci-dessous
	// ne fait que décrémenter, il ne ferme pas le handle du pool) ; cache miss →
	// ouverture RW via le MÊME connecteur custom, indexée sous la MÊME clé "rw:".
	// C'est bien un handle RW qu'il faut ici : IndexMedia fait du DDL, des INSERT,
	// des UPDATE et un CHECKPOINT — jamais OpenReadForQuery, qui rendrait un RO.
	//
	// FENÊTRE ASSUMÉE (revue 2026-08-26) : sur cache MISS, le handle ouvert ici est
	// inscrit au cache le temps de l'indexation — il devient donc visible des
	// emprunteurs NUS (sans indexLock ni refcount) : le ticker checkpoint de
	// cmd/server/main.go et l'attach du backup. Si l'un d'eux capture le pointeur
	// juste avant notre Close, il verra « database is closed » — un WARN best-effort
	// bénin (le checkpoint suivant ou celui du shutdown rattrape), impossible avant
	// puisque sql.Open était invisible du cache. Le vrai défaut est chez ces
	// emprunteurs nus (classe interdite par le garde-rail de routage) — consigné
	// pour un lot dédié, pas traité ici.
	tz := SanitizeMediaTimezone(opts.Timezone)
	handle, err := platform_duckdb.OpenReadWriteShared(targetPath, tz)
	if err != nil {
		return MediaIndexResult{}, fmt.Errorf("ouverture DB: %w", err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// Appliquer la timezone après ouverture pour un DATEDIFF correct (DST) : sur
	// un cache hit, le handle du pool porte SA timezone (celle du profil joueur),
	// pas celle passée ci-dessus — le SET reste donc nécessaire.
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

	result := MediaIndexResult{}
	// Fenêtres des médias indexés avant la règle par source : corrigées AVANT l'association,
	// qui lit capture_start_utc (media_capture_recalc.go). Idempotent ; un échec n'empêche pas
	// l'indexation (retenté au passage suivant) mais se journalise.
	if err := recalcBeforeIndex(ctx, db, loc, opts.Gamertag); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("capture_times: %v", err))
	}
	mediaFiles, err := walkMediaDir(opts.CapturesDir)
	if err != nil {
		return MediaIndexResult{}, fmt.Errorf("scan répertoire: %w", err)
	}
	result.Scanned = len(mediaFiles)
	slog.Debug("IndexMedia: scan répertoire", "scanned", result.Scanned)

	// Path store : convertit les chemins absolus disk en {owner_slug}/{rel}
	// stables pour stockage DB. Mode legacy si CapturesBase vide.
	store := MediaPathStore{CapturesBase: opts.CapturesBase}

	// Les fiches suivent le disque d'abord (renommage, déplacement, conversion,
	// disparition) : un fichier déplacé n'est pas un nouveau média.
	scan, err := reconcileIndexedFiles(ctx, db, opts, store, mediaFiles)
	if err != nil {
		return result, fmt.Errorf("réconciliation avec le disque: %w", err)
	}
	known, err := loadKnownHashes(ctx, db)
	if err != nil {
		return result, err
	}

	for _, path := range scan.files {
		hash, ok := scan.rec.Hashes[pathKey(path)]
		if !ok {
			continue // fichier trop récent, ou illisible (déjà journalisé par la passe)
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
		"skipped_foreign_title", scan.skippedForeign)

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

	// Miniatures manquantes : générées ou liées pour chaque vidéo visible, quel que soit
	// son dossier. Couvre tous les chemins d'indexation : upload, scan, reindex.
	thumbN, thumbErrs := EnsureVideoThumbnails(ctx, db, opts, store)
	result.Thumbnails += thumbN
	for _, e := range thumbErrs {
		result.Errors = append(result.Errors, "generate_thumbnail: "+e)
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

// insertMediaFile indexe un fichier de capture dans media_files : heure de capture et sa source
// (resolveCaptureInstant), fenêtre début/fin déduite de l'ancre de la source et de la durée
// (captureWindow), puis INSERT, ou UPDATE de la ligne de même stem quand l'ancien fichier a
// disparu (conversion de format).
func insertMediaFile(ctx context.Context, db *sql.DB, path, hash, playerSlug string, captureTimeUnix *int64, loc *time.Location, store MediaPathStore) error {
	ext := strings.ToLower(filepath.Ext(path))
	kind := mediaKindOf(ext)
	baseName := filepath.Base(path)
	stem := strings.TrimSuffix(baseName, ext)

	// Path à stocker en DB : relatif si le store peut convertir, sinon
	// fallback sur le path absolu (mode legacy).
	storedPath := store.ToRel(path, playerSlug)
	if storedPath == "" {
		storedPath = path
	}

	// Heure de capture : nom de fichier, sinon file.lastModified du navigateur (envoi HTTP,
	// web/features/media/queries.ts), sinon mtime du disque — et ce qu'elle dit (début ou fin).
	inst := resolveCaptureInstant(path, baseName, captureTimeUnix, loc)
	var sourceKey *string
	if inst == nil {
		slog.Warn("insertMediaFile: capture_start_utc indéterminé, média non-associable",
			"file", baseName, "player", playerSlug)
	} else {
		sourceKey = &inst.Source.Key
		slog.Debug("insertMediaFile: heure de capture lue",
			"file", baseName, "at", inst.At, "source", inst.Source.Key, "anchor", inst.Source.Anchor)
	}

	// Durée : ffprobe pour les vidéos (une image est un instant).
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
	win := captureWindow(kind, inst, duration, durationKnown)

	// Dédup extension-agnostique : cherche la fiche de même nom et de même dossier
	// (findSameNameRow). Si trouvée et ancien fichier encore sur disque → SKIP (les
	// deux coexistent). Si trouvée mais ancien fichier parti → UPDATE (conversion
	// terminée). Sinon → INSERT. L'original (chemin, taille, date) est celui du
	// fichier indexé.
	srcSize, srcMtime := sourceStat(ctx, path)
	same, found, err := findSameNameRow(ctx, db, sameNameQuery{
		player: playerSlug, stem: stem, abs: path, stored: storedPath, store: store,
	})
	if err != nil {
		return err
	}

	if found {
		existingID, existingPath := same.id, same.filePath
		// Entrée trouvée : vérifier si l'ancien fichier existe encore.
		// existingPath peut être relatif (post-migration) ou absolu (legacy) —
		// store.ToAbs gère les deux cas.
		if _, statErr := os.Stat(store.ToAbs(existingPath)); statErr == nil {
			// Ancien fichier existe toujours → SKIP (non-déterministe pendant conversion).
			slog.Debug("insertMediaFile: stem conflict, ancien fichier toujours présent, SKIP nouveau",
				"player", playerSlug, "stem", stem, "old_path", existingPath, "new_path", storedPath)
			return nil
		}

		// Ancien fichier parti → UPDATE l'entrée existante (conversion complétée), ses
		// likes suivent le changement de file_path (updateConvertedRow).
		upStart, upEnd, upSource := conversionTimes(inst, kind, win)
		err := updateConvertedRow(ctx, db, conversionUpdate{
			id: existingID, oldPath: existingPath, storedPath: storedPath, baseName: baseName,
			ext: ext, hash: hash, kind: kind, win: win, start: upStart, end: upEnd, source: upSource,
			srcSize: srcSize, srcMtime: srcMtime,
		})
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
	// a été retirée pour éradiquer le bug ART DuckDB #23645 (file_path est muté par
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
		`SELECT id, status FROM media_files WHERE file_path = ? OR source_path = ? ORDER BY id LIMIT 1`,
		storedPath, storedPath).Scan(&dupID, &dupStatus); err {
	case nil:
		if dupStatus.Valid && dupStatus.String == domain.MediaStatusDeleted {
			if err := reviveDeletedMedia(ctx, db, store, revival{
				id: dupID, storedPath: storedPath, srcSize: srcSize, srcMtime: srcMtime,
			}); err != nil {
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
			capture_start_utc, capture_end_utc, duration_seconds, capture_time_source,
			source_path, source_size, source_mtime_ns
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, playerSlug, storedPath, baseName, stem, ext, hash, kind, win.Start, win.End, win.DurationSec, sourceKey,
		storedPath, srcSize, srcMtime)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers timezone + filename parser
// ─────────────────────────────────────────────────────────────────────────────

// SanitizeMediaTimezone valide un nom de timezone IANA pour éviter l'injection SQL.
// Retourne "" si la valeur contient des caractères non autorisés.
// Exportée pour être utilisée par service.NewMediaService.
