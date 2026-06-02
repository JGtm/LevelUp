// Package ops — media_thumbnails.go : génération/liaison des miniatures via
// ffmpeg/ffprobe + calcul capture_end. Extrait de media.go (refactor god-file).
package ops

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func GenerateThumbnails(ctx context.Context, videosDir, thumbsDir string) (int, []string) {
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
		if supportedExtensions[ext] != mediaKindVideo {
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
		if err := generateAnimatedThumbnail(ctx, srcPath, thumbPath); err != nil {
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
	case mediaKindImage:
		if captureAt != nil {
			end := *captureAt
			captureEnd = &end
		}
		zero := 0.0
		durationSec = &zero
	case mediaKindVideo:
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
func probeVideoDuration(ctx context.Context, videoPath string) (float64, error) {
	cmd := exec.CommandContext(ctx, "ffprobe",
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

// thumbnailWindow calcule le point de départ et la durée d'extraction pour le
// WebP animé en fonction de la durée totale du clip. Règle : skip=20%,
// extract=20%, avec bornes min/max pour éviter les cas extrêmes. Si
// skip+extract dépasse la durée disponible, on repart de 0.
func thumbnailWindow(totalDur float64) (skip, extract float64) {
	const (
		skipPct    = 0.20
		extractPct = 0.20
		minSkip    = 3.0
		maxSkip    = 15.0
		minExtract = 3.0
		maxExtract = 8.0
	)
	clamp := func(v, lo, hi float64) float64 {
		if v < lo {
			return lo
		}
		if v > hi {
			return hi
		}
		return v
	}
	skip = clamp(totalDur*skipPct, minSkip, maxSkip)
	extract = clamp(totalDur*extractPct, minExtract, maxExtract)
	if skip+extract > totalDur-1 {
		skip = 0
		extract = clamp(totalDur*0.4, minExtract, maxExtract)
	}
	return skip, extract
}

// generateAnimatedThumbnail génère un WebP animé via ffmpeg/libwebp.
// La fenêtre d'extraction (skip + durée) est calculée proportionnellement à la
// durée du clip via thumbnailWindow — fallback sur 5s/3s si ffprobe échoue.
// Résolution 480px largeur, fps=12, qualité 75, compression max.
func generateAnimatedThumbnail(ctx context.Context, videoPath, webpPath string) error {
	skip, extract := 5.0, 3.0
	if dur, err := probeVideoDuration(ctx, videoPath); err == nil {
		skip, extract = thumbnailWindow(dur)
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-ss", strconv.FormatFloat(skip, 'f', 2, 64),
		"-t", strconv.FormatFloat(extract, 'f', 2, 64),
		"-i", videoPath,
		"-an",
		"-vf", "fps=12,scale=480:-1:flags=lanczos",
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
func BackfillThumbnailPaths(ctx context.Context, db *sql.DB, videosDir, thumbsDir, ownerSlug string, store MediaPathStore) (int, error) {
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
		res, err := db.ExecContext(ctx, `
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
