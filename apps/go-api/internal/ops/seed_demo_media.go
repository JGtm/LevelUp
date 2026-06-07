// Package ops — seed_demo_media.go : copie des vraies vidéos (flux HLS) pour la démo.
//
// Les médias du joueur source sont des flux HLS, comme en prod :
//   - {srcMediaDir}/hls/{name}/  : master.m3u8 + init_*.mp4 + seg_*.m4s + stream_*.m3u8
//   - {srcMediaDir}/thumbs/{name}.webp : vignette
//
// On copie les N captures "Halo Infinite …" les plus récentes dans la démo, on insère
// les media_files (schéma canonique shared_social : kind=video, file_path→master.m3u8,
// thumbnail_path→.webp) et on les attribue grossièrement aux matchs du corpus (même
// carte si possible) → page Média fonctionnelle avec lecture vidéo, comme en prod.
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/migration"

	_ "github.com/duckdb/duckdb-go/v2"
)

// haloInfinitePrefix : préfixe des captures Xbox Halo Infinite à utiliser (le user
// veut explicitement ces médias, pas les autres formats).
const haloInfinitePrefix = "Halo Infinite"

// hlsMedia décrit une capture HLS source à copier dans la démo.
type hlsMedia struct {
	Name         string // "Halo Infinite 2025-12-18 21-48-59"
	HLSDir       string // {srcMediaDir}/hls/{Name}
	ThumbPath    string // {srcMediaDir}/thumbs/{Name}.webp ("" si absente)
	CaptureStart sql.NullTime
}

// extractDemoMedia copie les vidéos HLS "Halo Infinite" du joueur source vers la
// démo et insère les media_files (schéma canonique shared_social) attribués au corpus.
func extractDemoMedia(
	ctx context.Context,
	srcMediaDir, srcSharedDB, outSocialDB, outMediaDir string,
	matchIDs []string,
	playerSlug string,
	maxMedia int,
) (int, error) {
	if err := os.MkdirAll(outMediaDir, 0o755); err != nil {
		return 0, fmt.Errorf("mkdir media: %w", err)
	}
	// Recréer le shared_social fresh à chaque reseed (cf. shared/player DBs). Sous
	// Linux, unlink d'un fichier tenu par le conteneur démo est sûr (inode survit).
	_ = os.Remove(outSocialDB)
	_ = os.Remove(outSocialDB + ".wal")
	if err := applyMigrationsOnPath(outSocialDB, migration.TargetSharedSocial); err != nil {
		return 0, fmt.Errorf("migrations shared_social démo: %w", err)
	}

	// Matchs du corpus courant (par map) pour l'attribution grossière des médias.
	mapIndex, err := buildDemoMapIndex(ctx, srcSharedDB, matchIDs)
	if err != nil {
		return 0, fmt.Errorf("build map index: %w", err)
	}
	corpus := flattenCorpusMatches(mapIndex)

	vids := scanHaloInfiniteHLS(srcMediaDir, maxMedia)
	if len(vids) == 0 {
		slog.InfoContext(ctx, "seed-demo media: aucune vidéo Halo Infinite trouvée", "src", srcMediaDir)
		return 0, nil
	}

	db, err := sql.Open("duckdb", outSocialDB)
	if err != nil {
		return 0, fmt.Errorf("open out shared_social: %w", err)
	}
	defer db.Close()

	imported := 0
	for i, v := range vids {
		if err := copyDir(v.HLSDir, filepath.Join(outMediaDir, v.Name)); err != nil {
			slog.WarnContext(ctx, "seed-demo media: copie HLS échouée", "name", v.Name, "err", err)
			continue
		}
		thumbnail := ""
		if v.ThumbPath != "" {
			tn := v.Name + ".webp"
			if err := copyFile(v.ThumbPath, filepath.Join(outMediaDir, tn)); err == nil {
				thumbnail = tn
			}
		}
		match, _ := pickCorpusMatch(corpus, "", i)
		// capture_start ALIGNÉ sur le match du corpus attribué (pas la date réelle du
		// fichier) : sinon la fenêtre de candidats de réassociation (matchs proches de
		// la capture) est vide en démo (captures déc.-fév. vs corpus récent). Aligné →
		// la réassociation propose les matchs voisins du corpus. Fallback date fichier.
		captureStart := v.CaptureStart
		if !match.TargetStartTime.IsZero() {
			captureStart = sql.NullTime{Time: match.TargetStartTime, Valid: true}
		}
		row := demoMediaRow{
			ID:            v.Name,
			PlayerSlug:    playerSlug,
			FilePath:      v.Name + "/master.m3u8", // HLS : URL /media/files/{name}/master.m3u8
			FileName:      v.Name,
			Kind:          mediaKindVideo,
			FileStem:      v.Name,
			FileExt:       ".m3u8",
			ThumbnailPath: thumbnail,
			MTime:         captureOrNow(captureStart),
			CaptureStart:  captureStart,
			MatchID:       match.TargetMatchID,
		}
		if err := insertDemoMediaRow(ctx, db, row); err != nil {
			slog.WarnContext(ctx, "seed-demo media: insert échouée", "name", v.Name, "err", err)
			continue
		}
		imported++
		slog.InfoContext(ctx, "seed-demo media: copiée",
			"name", v.Name, "match", match.TargetMatchID, "map", match.MapName)
	}
	return imported, nil
}

// scanHaloInfiniteHLS liste les flux HLS "Halo Infinite …" du joueur source (plus
// récents d'abord, nom = horodatage triable), avec leur vignette. Limité à maxMedia.
func scanHaloInfiniteHLS(srcMediaDir string, maxMedia int) []hlsMedia {
	hlsRoot := filepath.Join(srcMediaDir, "hls")
	thumbsRoot := filepath.Join(srcMediaDir, "thumbs")
	entries, err := os.ReadDir(hlsRoot)
	if err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), haloInfinitePrefix) {
			continue
		}
		if _, err := os.Stat(filepath.Join(hlsRoot, e.Name(), "master.m3u8")); err != nil {
			continue // flux incomplet
		}
		names = append(names, e.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names))) // plus récents d'abord
	if maxMedia > 0 && len(names) > maxMedia {
		names = names[:maxMedia]
	}
	out := make([]hlsMedia, 0, len(names))
	for _, n := range names {
		m := hlsMedia{Name: n, HLSDir: filepath.Join(hlsRoot, n), CaptureStart: parseHaloCaptureTime(n)}
		if tp := filepath.Join(thumbsRoot, n+".webp"); fileExists(tp) {
			m.ThumbPath = tp
		}
		out = append(out, m)
	}
	return out
}

// parseHaloCaptureTime extrait l'horodatage du nom "Halo Infinite 2025-12-18 21-48-59".
func parseHaloCaptureTime(name string) sql.NullTime {
	s := strings.TrimSpace(strings.TrimPrefix(name, haloInfinitePrefix))
	t, err := time.Parse("2006-01-02 15-04-05", s)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t.UTC(), Valid: true}
}

func captureOrNow(t sql.NullTime) time.Time {
	if t.Valid {
		return t.Time
	}
	return time.Now().UTC()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// copyDir copie les fichiers de src → dst (dossier HLS plat : master.m3u8 +
// init_*.mp4 + seg_*.m4s + stream_*.m3u8 ; pas de sous-dossiers). Le master.m3u8
// est réécrit pour ne garder qu'UNE piste audio (la 1ère = jeu ; les suivantes,
// ex. voicechat, sont retirées → pas de sélecteur côté lecteur).
func copyDir(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.Name() == "master.m3u8" {
			if err := copyMasterKeepFirstAudio(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}
		if err := copyFile(srcPath, dstPath); err != nil {
			return err
		}
	}
	return nil
}

// copyMasterKeepFirstAudio copie un master.m3u8 en ne conservant que la 1ère
// déclaration de piste audio (#EXT-X-MEDIA:TYPE=AUDIO) — retire le voicechat.
func copyMasterKeepFirstAudio(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	out := make([]string, 0, len(lines))
	audioSeen := false
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "#EXT-X-MEDIA:") && strings.Contains(t, "TYPE=AUDIO") {
			if audioSeen {
				continue // piste audio supplémentaire (voicechat) → retirée
			}
			audioSeen = true
		}
		out = append(out, l)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, []byte(strings.Join(out, "\n")), 0o644)
}

// mediaCandidate identifie un match du corpus (pour l'attribution des médias).
type mediaCandidate struct {
	TargetMatchID   string
	TargetStartTime time.Time
	MapName         string
}

// buildDemoMapIndex retourne {map_name → [matchs du corpus]} pour l'attribution.
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

// flattenCorpusMatches aplatit l'index map→[matchs] en une liste plate.
func flattenCorpusMatches(mapIndex map[string][]mediaCandidate) []mediaCandidate {
	out := make([]mediaCandidate, 0, len(mapIndex))
	for _, ms := range mapIndex {
		out = append(out, ms...)
	}
	return out
}

// pickCorpusMatch choisit un match du corpus pour un média : même map si possible,
// sinon round-robin sur i (répartit les médias). ok=false si corpus vide.
func pickCorpusMatch(corpus []mediaCandidate, mapName string, i int) (mediaCandidate, bool) {
	if len(corpus) == 0 {
		return mediaCandidate{}, false
	}
	if mapName != "" {
		for _, m := range corpus {
			if strings.EqualFold(m.MapName, mapName) {
				return m, true
			}
		}
	}
	return corpus[i%len(corpus)], true
}

// demoMediaRow porte les champs d'une ligne média démo (schéma canonique shared_social).
type demoMediaRow struct {
	ID, PlayerSlug, FilePath, FileName, Kind, FileStem, FileExt, ThumbnailPath string
	MTime                                                                      time.Time
	CaptureStart, CaptureEnd                                                   sql.NullTime
	MatchID                                                                    string
}

// insertDemoMediaRow insère un média au schéma CANONIQUE shared_social (id +
// player_slug + media_file_id) + son association. Idempotent (ON CONFLICT DO NOTHING).
func insertDemoMediaRow(ctx context.Context, db *sql.DB, m demoMediaRow) error {
	var thumb any
	if m.ThumbnailPath != "" {
		thumb = m.ThumbnailPath
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO media_files
			(id, player_slug, file_path, file_name, kind, file_hash, file_size,
			 file_stem, file_ext, thumbnail_path, capture_start_utc, capture_end_utc,
			 mtime, indexed_at, liked, status, created_at, updated_at)
		VALUES (?,?,?,?,?,'',0,?,?,?,?,?,?,CURRENT_TIMESTAMP,FALSE,'active',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO NOTHING`,
		m.ID, m.PlayerSlug, m.FilePath, m.FileName, m.Kind,
		m.FileStem, m.FileExt, thumb, m.CaptureStart, m.CaptureEnd, m.MTime); err != nil {
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

// copyFile copie src→dst en streaming.
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
