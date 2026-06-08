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

	mediapkg "levelup/go-api/internal/media"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// Valeurs de media_files.transcode_status. NULL = média servi en direct (non
// transcodé). Colonne dédiée — distincte de `status` ('active', rail home).
const (
	TranscodeProcessing = "processing"
	TranscodeReady      = "ready"
	TranscodeFailed     = "failed"
)

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

// MarkTranscodeStatus positionne transcode_status pour le média identifié par
// son file_path. CHECKPOINT best-effort pour durcir le WAL (ADR 0021).
func MarkTranscodeStatus(ctx context.Context, dbPath, fileRel, status string) error {
	return withSharedSocialDB(ctx, dbPath, func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx,
			`UPDATE media_files SET transcode_status = ? WHERE file_path = ?`, status, fileRel); err != nil {
			return fmt.Errorf("MarkTranscodeStatus: %w", err)
		}
		_, _ = db.ExecContext(ctx, "CHECKPOINT")
		return nil
	})
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
		_, _ = db.ExecContext(ctx, "CHECKPOINT")
		return nil
	})
}

// RunHLSTranscode exécute le transcoding complet (autonome, hors requête) :
// BuildHLS → finalize DB → suppression du source.
//
// Dégradations :
//   - BuildHLS échoue → transcode_status='failed', source conservé (le serving
//     retombe sur le remux WebM live du source).
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

	if err := finalizeMediaHLS(ctx, p.DBPath, p.FileRel, p.HLSRel); err != nil {
		log.ErrorContext(ctx, "RunHLSTranscode: finalize DB échoué — source conservé",
			"source", p.SourceAbs, "err", err)
		return err
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
