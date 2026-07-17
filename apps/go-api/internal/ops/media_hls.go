// Package ops — media_hls.go : façade transcoding HLS à l'ingestion.
//
// Orchestre la génération de l'arbre HLS-fMP4 (internal/media.BuildHLS) et la
// bascule du média en DB (file_path/hls_path → master.m3u8, transcode_status).
// Conçu pour être exécuté hors du contexte de la requête HTTP (worker async) :
// RunHLSTranscode ouvre sa propre connexion DB, sérialisée avec IndexMedia via
// indexLock.
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

	mediapkg "levelup/go-api/internal/media"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// Valeurs de media_files.transcode_status. NULL = statut inconnu / hérité (ligne
// jamais évaluée par le pipeline). Colonne dédiée — distincte de `status`
// ('active', rail home).
const (
	TranscodeProcessing = "processing"
	TranscodeReady      = "ready"
	TranscodeFailed     = "failed"
	// TranscodeDirect : média web-natif mono-piste servi en direct (aucun HLS
	// requis). Persisté pour que le balayage EnsurePendingHLS n'ait plus jamais à
	// re-prober ces lignes — sans persistance, chaque balayage post-sync relançait
	// un ffprobe sur toutes les vidéos non candidates, indéfiniment.
	TranscodeDirect = "direct"
)

// transcodeStaleAfter borne la durée au-delà de laquelle une ligne restée
// 'processing' est considérée comme un orphelin de crash (worker tué avant
// finalize/fail) et redevient éligible au transcodage. En deçà, un 'processing'
// frais est un transcodage RÉELLEMENT en cours qu'il ne faut pas relancer en
// parallèle (deux ffmpeg écrivant les mêmes segments dans le même outDir →
// arbre HLS incohérent validé, puis source supprimé = perte définitive). 2 h
// couvrent très largement le pire transcodage. Source UNIQUE du seuil, partagée
// entre la sélection du balayage et le verrou MarkTranscodeProcessing.
const transcodeStaleAfter = 2 * time.Hour

// transcodeNotFreshProcessingSQL : prédicat « pas de transcodage déjà en cours »
// — vrai si la ligne n'est PAS 'processing' frais. Fragment UNIQUE partagé entre
// la sélection du balayage (selectPendingHLSCandidates) et le compare-and-set de
// MarkTranscodeProcessing : toute divergence entre les deux rouvrirait la course
// upload-vs-sweep. Le placeholder `?` attend le seuil de péremption
// (now UTC - transcodeStaleAfter). COALESCE neutralise la logique tri-valuée SQL
// sur transcode_status NULL ; transcode_started_at NULL sur un 'processing' =
// orphelin legacy (ligne d'avant l'horodatage) → éligible.
const transcodeNotFreshProcessingSQL = `(
	COALESCE(transcode_status, '') <> 'processing'
	OR transcode_started_at IS NULL
	OR transcode_started_at <= ?
)`

// logModuleMedia route les logs de transcoding vers logs/media.log. Sans cet
// attribut, le package ops tomberait dans logs/general.log (détection par PC).
const logModuleMedia = "media"

// HLSTranscodeParams décrit un travail de transcoding autonome (exécutable hors
// requête HTTP, dans une goroutine worker).
type HLSTranscodeParams struct {
	SourceAbs string // chemin absolu du fichier source (MKV/AVI/MP4 multipiste)
	OutDir    string // dossier de sortie HLS ({capturesDir}/hls/{stem})
	DBPath    string // shared_social.duckdb — cible de l'UPDATE media_files
	FileRel   string // file_path actuel en DB (clé WHERE) = source relatif stable
	HLSRel    string // futur file_path + hls_path = master.m3u8 relatif stable
	// DeleteSource : supprimer le fichier source après un transcodage HLS prouvé
	// lisible (4e garde anti-perte). false = conserver le source (défaut sûr en
	// local ; le HLS reste servi, le source reste la copie maître). Résolu en amont
	// par config.ResolveMediaDeleteSource (env > store > isProd).
	DeleteSource bool
}

// DetectHLSNeeded probe le fichier et décide s'il doit être transcodé en HLS
// (container non web-natif OU multipiste). Erreur si ffprobe échoue.
func DetectHLSNeeded(ctx context.Context, srcAbs string) (bool, error) {
	streams, err := mediapkg.ProbeStreamsDetailed(ctx, srcAbs)
	if err != nil {
		return false, err
	}
	ext := strings.ToLower(filepath.Ext(srcAbs))
	return mediapkg.NeedsHLS(ext, streams), nil
}

// HLSPathsFor calcule le dossier de sortie HLS et le chemin relatif stable du
// master.m3u8 (format MediaPathStore : {gamertag}/hls/{stem}/master.m3u8).
// Pure : aucune IO. En mode legacy (capturesBase vide), hlsRel retombe sur le
// chemin absolu du master.
//
// Limite connue (collision de stem) : le dossier de sortie ne dépend que du stem
// (nom sans extension). Deux sources au même stem dans un même dossier joueur
// (clip.mkv et clip.mp4) résolvent vers le MÊME hls/{stem} — la seconde
// transcodée écrase l'arbre de la première. Pas de correctif structurel ici :
// désambiguïser le nom (inclure l'extension) casserait les arbres HLS déjà
// produits et référencés en DB. En pratique une capture porte une extension
// stable ; la collision suppose deux conteneurs jumeaux, cas non observé.
func HLSPathsFor(capturesDir, capturesBase, gamertag, sourceAbs string) (outDir, hlsRel string) {
	stem := strings.TrimSuffix(filepath.Base(sourceAbs), filepath.Ext(sourceAbs))
	outDir = filepath.Join(capturesDir, "hls", stem)
	master := filepath.Join(outDir, "master.m3u8")
	if rel := (MediaPathStore{CapturesBase: capturesBase}).ToRel(master, gamertag); rel != "" {
		return outDir, rel
	}
	return outDir, master
}

// withSharedSocialDB ouvre la DB cible (handle pool si présent, sinon sql.Open)
// et exécute fn sous le lock d'indexation du path (sérialise avec IndexMedia,
// évite la race ATTACH/DETACH DuckDB).
func withSharedSocialDB(ctx context.Context, dbPath string, fn func(*sql.DB) error) error {
	unlock := indexLock(dbPath)
	defer unlock()

	if cached, ok := platform_duckdb.LookupCachedDB(dbPath); ok {
		return fn(cached.SQLDb()) // handle possédé par le pool : pas de Close
	}
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return fmt.Errorf("ouverture DB %q: %w", dbPath, err)
	}
	defer db.Close()
	return fn(db)
}

// checkpointBestEffort force un CHECKPOINT pour durcir le WAL après une écriture
// media_files (ADR 0021/0022 — sans CHECKPOINT le WAL peut être perdu au crash).
// Best-effort : un échec est loggé en WARN (règle CLAUDE.md #10, jamais d'erreur
// avalée) mais NON bloquant — l'UPDATE porteur est déjà commité.
func checkpointBestEffort(ctx context.Context, db *sql.DB, where string) {
	if _, err := db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		slog.With("module", logModuleMedia).WarnContext(ctx, where+": CHECKPOINT échoué (WAL non durci)", "err", err)
	}
}

// MarkTranscodeStatus positionne transcode_status pour le média identifié par
// son file_path (transitions 'direct'/'failed' ; le passage à 'processing' passe
// par MarkTranscodeProcessing, qui horodate). CHECKPOINT best-effort (ADR 0021).
func MarkTranscodeStatus(ctx context.Context, dbPath, fileRel, status string) error {
	return withSharedSocialDB(ctx, dbPath, func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx,
			`UPDATE media_files SET transcode_status = ? WHERE file_path = ?`, status, fileRel); err != nil {
			return fmt.Errorf("MarkTranscodeStatus: %w", err)
		}
		checkpointBestEffort(ctx, db, "MarkTranscodeStatus")
		return nil
	})
}

// MarkTranscodeProcessing tente d'ACQUÉRIR le verrou de transcodage du média :
// passage à 'processing' + horodatage (transcode_started_at, UTC), en
// compare-and-set — l'UPDATE ne s'applique QUE si la ligne n'est pas déjà
// 'processing' frais (même prédicat que la sélection du balayage). Appelé AVANT
// RunHLSTranscode par les DEUX chemins (worker upload ET balayage).
//
// Pourquoi conditionnel : le balayage sélectionne ses candidats UNE fois puis
// transcode séquentiellement pendant de longues minutes ; un upload qui marque
// 'processing' et lance son worker entre la sélection et le tour du candidat
// serait écrasé par un UPDATE inconditionnel → deux ffmpeg en parallèle sur le
// même outDir. acquired=false = un autre worker transcode déjà (ou la ligne a
// disparu) → l'appelant SAUTE le transcodage sans erreur.
//
// L'horodatage sert la récupération d'orphelins de crash : un 'processing'
// périmé (cf. transcodeStaleAfter) redevient acquérable. CHECKPOINT best-effort
// pour durcir le WAL (le timestamp doit survivre à un crash, sinon la
// récupération d'orphelin ne se déclenche jamais).
func MarkTranscodeProcessing(ctx context.Context, dbPath, fileRel string) (acquired bool, err error) {
	err = withSharedSocialDB(ctx, dbPath, func(db *sql.DB) error {
		now := time.Now().UTC()
		res, execErr := db.ExecContext(ctx,
			`UPDATE media_files SET transcode_status = ?, transcode_started_at = ?
			 WHERE file_path = ? AND `+transcodeNotFreshProcessingSQL,
			TranscodeProcessing, now, fileRel, now.Add(-transcodeStaleAfter))
		if execErr != nil {
			return fmt.Errorf("MarkTranscodeProcessing: %w", execErr)
		}
		n, raErr := res.RowsAffected()
		if raErr != nil {
			return fmt.Errorf("MarkTranscodeProcessing: RowsAffected: %w", raErr)
		}
		acquired = n > 0
		checkpointBestEffort(ctx, db, "MarkTranscodeProcessing")
		return nil
	})
	return acquired, err
}

// finalizeMediaHLS bascule le média vers sa version HLS : file_path et hls_path
// pointent vers master.m3u8, transcode_status='ready'.
func finalizeMediaHLS(ctx context.Context, dbPath, fileRel, hlsRel string) error {
	return withSharedSocialDB(ctx, dbPath, func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx, `
			UPDATE media_files
			SET file_path = ?, hls_path = ?, transcode_status = ?
			WHERE file_path = ?
		`, hlsRel, hlsRel, TranscodeReady, fileRel); err != nil {
			return fmt.Errorf("finalizeMediaHLS: %w", err)
		}
		checkpointBestEffort(ctx, db, "finalizeMediaHLS")
		return nil
	})
}

// RunHLSTranscode exécute le transcoding complet (autonome, hors requête) :
// BuildHLS → vérification ffprobe → finalize DB → suppression du source.
//
// Dégradations (le source n'est JAMAIS supprimé tant que la lecture HLS n'est
// pas prouvée → le serving retombe sur le remux WebM live du source) :
//   - BuildHLS échoue → transcode_status='failed', source conservé.
//   - HLS produit illisible (ffprobe) → transcode_status='failed', source conservé.
//   - finalize DB échoue → source conservé, status reste 'processing' (repris
//     ultérieurement par le backfill).
func RunHLSTranscode(ctx context.Context, p HLSTranscodeParams) error {
	log := slog.With("module", logModuleMedia)
	res, err := mediapkg.BuildHLS(ctx, p.SourceAbs, p.OutDir, mediapkg.HLSOptions{})
	if err != nil {
		log.ErrorContext(ctx, "RunHLSTranscode: BuildHLS échoué — média laissé en remux legacy",
			"source", p.SourceAbs, "err", err)
		if markErr := MarkTranscodeStatus(ctx, p.DBPath, p.FileRel, TranscodeFailed); markErr != nil {
			log.ErrorContext(ctx, "RunHLSTranscode: mark failed échoué", "err", markErr)
		}
		return err
	}

	// Garde anti-perte de données : BuildHLS ne fait que stat les fichiers
	// produits ; on prouve ici que le master est réellement démultiplexable
	// (ffprobe) ET qu'il déclare toutes les renditions audio attendues AVANT de
	// supprimer le source. Sinon un manifest produit mais cassé (init/segments
	// manquants, rendition perdue) deviendrait illisible et irrécupérable (source
	// détruit, plus de fallback remux, plus de miniature).
	if err := mediapkg.VerifyHLSPlayable(ctx, res.MasterPath, res.AudioTracks); err != nil {
		log.ErrorContext(ctx, "RunHLSTranscode: HLS produit illisible (ffprobe) — source conservé, remux legacy",
			"source", p.SourceAbs, "master", res.MasterPath, "err", err)
		if markErr := MarkTranscodeStatus(ctx, p.DBPath, p.FileRel, TranscodeFailed); markErr != nil {
			log.ErrorContext(ctx, "RunHLSTranscode: mark failed échoué", "err", markErr)
		}
		return err
	}

	if err := finalizeMediaHLS(ctx, p.DBPath, p.FileRel, p.HLSRel); err != nil {
		log.ErrorContext(ctx, "RunHLSTranscode: finalize DB échoué — source conservé",
			"source", p.SourceAbs, "err", err)
		return err
	}

	// Seconde garde anti-perte : ne supprimer le source que si une miniature est
	// déjà liée. Sinon (échec ffmpeg miniature à l'ingestion) le source est le
	// SEUL moyen de régénérer la miniature plus tard (regen-thumbnails) — le
	// détruire laisserait un média lisible mais sans miniature, irréversible.
	// La miniature est générée par IndexMedia AVANT ce transcoding async.
	hasThumb, thumbErr := mediaHasThumbnail(ctx, p.DBPath, p.HLSRel)
	if thumbErr != nil {
		log.WarnContext(ctx, "RunHLSTranscode: vérif miniature échouée — source conservé par prudence",
			"hls", p.HLSRel, "err", thumbErr)
		return nil
	}
	if !hasThumb {
		log.WarnContext(ctx, "RunHLSTranscode: aucune miniature liée — source conservé pour régénération ultérieure",
			"hls", p.HLSRel)
		return nil
	}

	// Quatrième garde : politique de rétention. Le source n'est supprimé que si
	// l'opérateur l'a demandé (media_delete_source_after_transcode). Défaut sûr en
	// local = conservation (le source reste la copie maître ; le HLS est déjà servi).
	if !p.DeleteSource {
		log.InfoContext(ctx, "RunHLSTranscode: source conservé (politique de rétention: media_delete_source_after_transcode=false)",
			"source", p.SourceAbs, "hls", p.HLSRel)
		return nil
	}

	if err := os.Remove(p.SourceAbs); err != nil {
		log.WarnContext(ctx, "RunHLSTranscode: suppression source échouée (non bloquant)",
			"source", p.SourceAbs, "err", err)
	}
	log.InfoContext(ctx, "RunHLSTranscode: terminé",
		"source", p.SourceAbs, "hls", p.HLSRel,
		"audio_tracks", res.AudioTracks, "segments", res.Segments,
		"renditions", res.Renditions)
	return nil
}

// mediaHasThumbnail indique si la ligne média identifiée par son file_path
// courant porte une miniature liée (thumbnail_path non NULL). Lecture sous le
// lock d'indexation (réutilise le handle du pool si présent).
func mediaHasThumbnail(ctx context.Context, dbPath, fileRel string) (bool, error) {
	var n int
	err := withSharedSocialDB(ctx, dbPath, func(db *sql.DB) error {
		return db.QueryRowContext(ctx, `
			SELECT COUNT(*)
			FROM media_files
			WHERE file_path = ? AND thumbnail_path IS NOT NULL
		`, fileRel).Scan(&n)
	})
	return n > 0, err
}
