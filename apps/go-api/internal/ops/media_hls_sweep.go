// Package ops — media_hls_sweep.go : balayage « assure le HLS » partagé.
//
// Routine unique qui transcode en HLS toute vidéo média encore sans version HLS
// (hls_path IS NULL). Elle est appelée par DEUX chemins, ce qui supprime
// l'asymétrie upload/scan à l'origine du bug HEVC :
//   - le scan/sync média (déclenchement automatique in-process, cf.
//     media_index_service.go) — les captures scannées passent désormais en HLS
//     comme les uploads ;
//   - le CLI backfill-media-hls (rattrapage manuel, serveur arrêté).
//
// Le HLS gère le HEVC en copie (zéro réencodage) ; le remux WebM live, lui, le
// rejette. Faire converger les deux chemins vers le HLS rend donc les captures
// HEVC lisibles sans intervention manuelle.
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
)

// hlsSweepMu garantit qu'un seul balayage EnsurePendingHLS tourne à la fois dans
// le process. Le transcodage HLS est CPU-lourd ; empiler un balayage par
// scan/sync ne ferait que saturer la machine. On DÉPOSE (TryLock) plutôt que
// d'empiler : un balayage en cours couvre déjà tous les médias sans HLS, et le
// prochain scan relancera un balayage pour les fichiers indexés entre-temps.
var hlsSweepMu sync.Mutex

// EnsureHLSParams configure un balayage EnsurePendingHLS.
type EnsureHLSParams struct {
	DBPath       string // shared_social.duckdb (lecture candidats + bascule file_path/hls_path)
	CapturesBase string // MediaCapturesBaseDir — racine multi-player {base}/{owner}/...
	OnlySlug     string // ne traiter qu'un joueur ("" = tous)
	Limit        int    // nombre max de clips transcodés (0 = illimité)
	DryRun       bool   // lister sans transcoder
	// DeleteSource : propagé tel quel à HLSTranscodeParams.DeleteSource (politique de
	// rétention du source après transcodage). Résolu en amont par l'appelant.
	DeleteSource bool
}

// EnsureHLSStats résume un balayage.
type EnsureHLSStats struct {
	Transcoded    int
	SkippedDirect int  // web-natif / mono-piste → pas de HLS requis
	Missing       int  // source absente sur disque
	Failed        int  // probe ou transcodage en échec
	Busy          bool // un autre balayage tournait déjà → no-op (single-flight)
}

type hlsCandidate struct {
	id         int64
	playerSlug string
	filePath   string
}

// EnsurePendingHLS transcode en HLS toutes les vidéos média encore sans version
// HLS. Réutilise DetectHLSNeeded + RunHLSTranscode (copy HEVC/H264/AV1 conservés,
// réencode ciblé sinon, suppression du source au succès, bascule DB). Single-flight :
// retourne immédiatement (Busy=true) si un balayage tourne déjà. Séquentiel pour
// borner la charge CPU.
func EnsurePendingHLS(ctx context.Context, p EnsureHLSParams) (EnsureHLSStats, error) {
	log := slog.With("module", logModuleMedia)
	if !hlsSweepMu.TryLock() {
		log.DebugContext(ctx, "hls sweep: déjà en cours, ignoré (single-flight)")
		return EnsureHLSStats{Busy: true}, nil
	}
	defer hlsSweepMu.Unlock()

	candidates, err := selectPendingHLSCandidates(ctx, p.DBPath, p.OnlySlug)
	if err != nil {
		return EnsureHLSStats{}, fmt.Errorf("EnsurePendingHLS: sélection candidats: %w", err)
	}
	if len(candidates) == 0 {
		return EnsureHLSStats{}, nil
	}
	log.InfoContext(ctx, "hls sweep: démarrage",
		"candidats", len(candidates), "captures_base", p.CapturesBase, "dry_run", p.DryRun)

	var st EnsureHLSStats
	store := MediaPathStore{CapturesBase: p.CapturesBase}
	for _, c := range candidates {
		if ctx.Err() != nil {
			return st, ctx.Err()
		}
		if p.Limit > 0 && st.Transcoded >= p.Limit {
			break
		}
		processHLSCandidate(ctx, c, store, p, &st)
	}

	log.InfoContext(ctx, "hls sweep: terminé",
		"transcoded", st.Transcoded, "skipped_direct", st.SkippedDirect,
		"missing", st.Missing, "failed", st.Failed)
	return st, nil
}

// selectPendingHLSCandidates liste les vidéos sans HLS via le handle DuckDB
// mutualisé (withSharedSocialDB) — JAMAIS un sql.Open READ_ONLY concurrent, qui
// déclencherait "different configuration" sur une DB déjà tenue RW in-process
// (cf. handle cache LookupCachedDB). En CLI (serveur arrêté, pas de pool),
// withSharedSocialDB retombe sur un sql.Open classique.
func selectPendingHLSCandidates(ctx context.Context, dbPath, onlySlug string) ([]hlsCandidate, error) {
	var out []hlsCandidate
	err := withSharedSocialDB(ctx, dbPath, func(db *sql.DB) error {
		q := `SELECT id, COALESCE(player_slug, ''), COALESCE(file_path, '')
		      FROM media_files
		      WHERE kind = 'video' AND hls_path IS NULL`
		args := []any{}
		if onlySlug != "" {
			q += ` AND player_slug = ?`
			args = append(args, onlySlug)
		}
		rows, err := db.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close() //nolint:errcheck
		for rows.Next() {
			var c hlsCandidate
			if err := rows.Scan(&c.id, &c.playerSlug, &c.filePath); err != nil {
				return err
			}
			out = append(out, c)
		}
		return rows.Err()
	})
	return out, err
}

// processHLSCandidate transcode un candidat si un HLS est requis, met à jour st.
// owner est dérivé de player_slug, à défaut du premier segment du file_path.
func processHLSCandidate(ctx context.Context, c hlsCandidate, store MediaPathStore, p EnsureHLSParams, st *EnsureHLSStats) {
	log := slog.With("module", logModuleMedia)
	owner := c.playerSlug
	if owner == "" {
		owner = strings.SplitN(filepath.ToSlash(c.filePath), "/", 2)[0]
	}
	abs := store.ToAbs(c.filePath)
	if _, err := os.Stat(abs); err != nil {
		log.DebugContext(ctx, "hls sweep: source absente sur disque", "file", c.filePath, "abs", abs)
		st.Missing++
		return
	}
	needed, err := DetectHLSNeeded(ctx, abs)
	if err != nil {
		log.WarnContext(ctx, "hls sweep: probe échouée", "file", c.filePath, "err", err)
		st.Failed++
		return
	}
	if !needed {
		st.SkippedDirect++
		return
	}
	outDir, hlsRel := HLSPathsFor(filepath.Join(p.CapturesBase, owner), p.CapturesBase, owner, abs)
	if p.DryRun {
		log.InfoContext(ctx, "hls sweep: serait transcodé", "file", c.filePath, "hls", hlsRel)
		st.Transcoded++
		return
	}
	if err := RunHLSTranscode(ctx, HLSTranscodeParams{
		SourceAbs: abs, OutDir: outDir, DBPath: p.DBPath, FileRel: c.filePath, HLSRel: hlsRel,
		DeleteSource: p.DeleteSource,
	}); err != nil {
		log.WarnContext(ctx, "hls sweep: transcodage échoué", "file", c.filePath, "err", err)
		st.Failed++
		return
	}
	st.Transcoded++
}
