// Package ops — media_tooling.go : vérification de l'outillage média
// (ffmpeg/ffprobe) au-delà de la simple présence dans le PATH.
//
// L'app exécute ffmpeg/ffprobe pour trois surfaces :
//   - miniatures WebP animées      (internal/ops/media_thumbnails.go : encodeur libwebp) ;
//   - transcodage HLS-fMP4         (internal/media/hls.go : muxer hls + mp4, un encodeur H.264 parmi
//     libx264 / h264_mf / libopenh264, encodeur audio aac) ;
//   - remux live des clips MKV/AVI (internal/media/remux.go).
//
// Un ffmpeg présent mais compilé sans encodeur H.264, sans libwebp/aac ou sans le muxer hls
// échoue silencieusement au premier upload. Ce fichier rend ce risque observable
// à deux endroits, sans dupliquer les execs :
//   - au boot serveur (cmd/server)  : LogMediaToolingStatus — slog non-bloquant ;
//   - dans la CLI `check-env`        : InspectMediaTooling + Report.Summary (stdout).
//
// Le PARSING des sorties ffmpeg (parseFFmpegComponents, missingComponents,
// firstLine, et media.ParseFFmpegComponents) est PUR et testable avec une sortie simulée — aucun test ne dépend
// d'un ffmpeg réellement installé (cf. media_tooling_test.go).
package ops

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/domain"
	mediapkg "levelup/go-api/internal/media"
)

// requiredEncoders liste les encodeurs ffmpeg dont dépend LevelUp :
//   - libwebp : miniatures animées WebP (media_thumbnails.go, -c:v libwebp) ;
//   - aac     : ré-encodage audio des renditions HLS (hls.go, -c:a aac).
//
// L'encodeur vidéo H.264 (ré-encodage quand la copy fMP4 est impossible) n'est pas dans cette
// liste : il se choisit parmi plusieurs (media.SelectedH264Encoder), voir MediaToolingReport.
var requiredEncoders = []string{"libwebp", "aac"}

// requiredMuxers liste les muxers ffmpeg requis :
//   - hls : arbre HLS (master + sous-playlists), hls.go (-f hls) ;
//   - mp4 : conteneur des segments/inits fMP4 — `-hls_segment_type fmp4` s'appuie
//     sur le muxer mp4 fragmenté ; ffmpeg n'expose pas de muxer "fmp4" distinct.
var requiredMuxers = []string{"hls", "mp4"}

// mediaComponentImpact décrit la surface produit cassée par l'absence de chaque
// composant — sert de contexte actionnable dans les logs et la sortie check-env.
var mediaComponentImpact = map[string]string{
	"libwebp": "miniatures animées WebP",
	"aac":     "ré-encodage audio des renditions HLS",
	"hls":     "transcodage HLS (sélection de piste audio + seek)",
	"mp4":     "segments fMP4 des flux HLS",
}

// MediaToolingReport résume l'état de l'outillage média. Construit par
// InspectMediaTooling ; consommé par LogMediaToolingStatus (slog) et Summary (CLI).
type MediaToolingReport struct {
	FFmpegFound   bool
	FFmpegPath    string
	FFmpegVersion string // 1re ligne de `ffmpeg -version` ("" si indisponible)
	FFprobeFound  bool
	FFprobePath   string

	// CapabilitiesProbed vaut true quand l'interrogation encoders/muxers a
	// abouti (ffmpeg présent ET les deux sous-commandes ont répondu). Si false,
	// MissingEncoders/MissingMuxers ne sont PAS significatifs (on ne sait pas).
	CapabilitiesProbed bool
	// CapabilitiesProbeErr porte l'échec d'interrogation (ffmpeg présent mais
	// -encoders/-muxers a erré) — évite un faux « tout manquant ».
	CapabilitiesProbeErr error
	MissingEncoders      []string // sous-ensemble de requiredEncoders absent
	MissingMuxers        []string // sous-ensemble de requiredMuxers absent

	// H264Candidates : encodeurs H.264 listés par ffmpeg, dans l'ordre de préférence.
	H264Candidates []mediapkg.H264Encoder
	// H264Retained : encodeur retenu après essai ("" si aucun ; alors H264Err porte la cause).
	H264Retained mediapkg.H264Encoder
	H264Err      error
}

// InspectMediaTooling résout ffmpeg/ffprobe dans le PATH puis, si ffmpeg est
// présent, interroge sa version et ses capacités (encoders + muxers). Un seul
// exec par sous-commande (`-version`, `-encoders`, `-muxers`). Best-effort :
// n'échoue jamais, ne bloque jamais — tout est reflété dans le rapport.
func InspectMediaTooling(ctx context.Context) MediaToolingReport {
	var r MediaToolingReport
	r.FFmpegPath, r.FFmpegFound = lookupBinary(mediapkg.ResolveFFmpeg)
	r.FFprobePath, r.FFprobeFound = lookupBinary(mediapkg.ResolveFFprobe)
	if !r.FFmpegFound {
		return r // sans ffmpeg, impossible d'interroger version/capacités
	}
	if verOut, err := runFFmpeg(ctx, "-version"); err == nil {
		r.FFmpegVersion = firstLine(verOut)
	}
	encoders, encErr := mediapkg.AvailableEncoders(ctx)
	muxOut, muxErr := runFFmpeg(ctx, "-hide_banner", "-muxers")
	switch {
	case encErr != nil:
		r.CapabilitiesProbeErr = encErr
	case muxErr != nil:
		r.CapabilitiesProbeErr = muxErr
	default:
		r.CapabilitiesProbed = true
		r.MissingEncoders = missingComponents(encoders, requiredEncoders)
		r.MissingMuxers = missingComponents(mediapkg.ParseFFmpegComponents(muxOut), requiredMuxers)
		r.H264Candidates = mediapkg.H264Candidates(encoders)
		r.H264Retained, r.H264Err = mediapkg.SelectedH264Encoder(ctx)
	}
	return r
}

// LogMediaToolingStatus émet l'état de l'outillage média au boot serveur, en
// slog structuré et NON-bloquant : slog.InfoContext si ffmpeg/ffprobe présents
// (avec la version), slog.WarnContext actionnable si l'un manque, puis un WARN
// par composant (encodeur/muxer) requis manquant. Le serveur démarre quoi qu'il
// arrive — cette fonction ne renvoie rien et ne panique jamais.
func LogMediaToolingStatus(ctx context.Context) {
	// Tag explicite → logs/media.log : le package ops est hétérogène (snapshot,
	// général) et n'a pas de module par défaut ; cette sonde appartient au flux média.
	log := slog.With("module", logModuleMedia)
	r := InspectMediaTooling(ctx)
	if r.FFmpegFound && r.FFprobeFound {
		log.InfoContext(ctx, "media tooling: ffmpeg/ffprobe disponibles",
			"ffmpeg_version", r.FFmpegVersion,
			"ffmpeg_path", r.FFmpegPath,
			"ffprobe_path", r.FFprobePath)
	} else {
		log.WarnContext(ctx, "media tooling: ffmpeg/ffprobe absent du PATH — miniatures, transcodage HLS et remux live indisponibles ; installer ffmpeg",
			"ffmpeg_found", r.FFmpegFound,
			"ffprobe_found", r.FFprobeFound)
	}
	if r.CapabilitiesProbeErr != nil {
		log.WarnContext(ctx, "media tooling: interrogation des capacités ffmpeg échouée — encoders/muxers non vérifiés",
			"err", r.CapabilitiesProbeErr)
		return
	}
	for _, name := range r.MissingEncoders {
		log.WarnContext(ctx, "media tooling: encodeur ffmpeg requis manquant",
			"component", name, "impact", mediaComponentImpact[name],
			"hint", "installer un build ffmpeg incluant l'encodeur "+name)
	}
	logH264Status(ctx, log, r)
	for _, name := range r.MissingMuxers {
		log.WarnContext(ctx, "media tooling: muxer ffmpeg requis manquant",
			"component", name, "impact", mediaComponentImpact[name],
			"hint", "installer un build ffmpeg incluant le muxer "+name)
	}
}

// logH264Status journalise l'encodeur H.264 retenu, ou l'absence de tout encodeur utilisable
// (transcodage vidéo HLS alors refusé ; la copie sans réencodage reste possible).
func logH264Status(ctx context.Context, log *slog.Logger, r MediaToolingReport) {
	if r.H264Retained != "" {
		log.InfoContext(ctx, "media tooling: encodeur H.264 retenu",
			"encoder", string(r.H264Retained), "candidates", h264Names(r.H264Candidates))
		return
	}
	if !errors.Is(r.H264Err, mediapkg.ErrNoH264Encoder) {
		log.WarnContext(ctx, "media tooling: détection de l'encodeur H.264 non aboutie, reprise au premier transcodage",
			"candidates", h264Names(r.H264Candidates), "err", r.H264Err)
		return
	}
	log.WarnContext(ctx, "media tooling: aucun encodeur H.264 utilisable — le transcodage vidéo HLS (clips VP8/VP9) sera refusé",
		"candidates", h264Names(r.H264Candidates), "err", r.H264Err,
		"hint", "installer un build ffmpeg incluant libx264, h264_mf ou libopenh264")
}

// h264Names convertit une liste d'encodeurs en noms (champ de journal).
func h264Names(encs []mediapkg.H264Encoder) []string {
	names := make([]string, len(encs))
	for i, e := range encs {
		names[i] = string(e)
	}
	return names
}

// ToHealthStatus projette le rapport sur le DTO exposé par GET /health. On ne
// retient que la présence des binaires et la version ffmpeg (preuve positive) ;
// le détail des encodeurs/muxers manquants reste réservé à la CLI check-env et
// aux WARN du boot (Summary / LogMediaToolingStatus), /health restant concis.
func (r MediaToolingReport) ToHealthStatus() domain.MediaToolingStatus {
	return domain.MediaToolingStatus{
		FFmpeg:        r.FFmpegFound,
		FFprobe:       r.FFprobeFound,
		FFmpegVersion: r.FFmpegVersion,
	}
}

// Summary formate le rapport pour la sortie CLI (check-env). Informatif — ne
// change pas le code de sortie de la commande (les manques sont marqués MANQUANT).
func (r MediaToolingReport) Summary() string {
	var sb strings.Builder
	sb.WriteString("Outillage média (ffmpeg/ffprobe) :\n")
	if r.FFmpegFound {
		if r.FFmpegVersion != "" {
			fmt.Fprintf(&sb, "  [OK] ffmpeg  : %s (%s)\n", r.FFmpegPath, r.FFmpegVersion)
		} else {
			fmt.Fprintf(&sb, "  [OK] ffmpeg  : %s\n", r.FFmpegPath)
		}
	} else {
		sb.WriteString("  [KO] ffmpeg  : introuvable (LEVELUP_FFMPEG_DIR, dossier de l'exécutable, PATH)\n")
	}
	if r.FFprobeFound {
		fmt.Fprintf(&sb, "  [OK] ffprobe : %s\n", r.FFprobePath)
	} else {
		sb.WriteString("  [KO] ffprobe : introuvable (LEVELUP_FFMPEG_DIR, dossier de l'exécutable, PATH)\n")
	}
	switch {
	case r.CapabilitiesProbed:
		writeCapLine(&sb, "encodeurs requis", requiredEncoders, r.MissingEncoders)
		writeCapLine(&sb, "muxers requis   ", requiredMuxers, r.MissingMuxers)
		writeH264Line(&sb, r)
	case r.FFmpegFound && r.CapabilitiesProbeErr != nil:
		fmt.Fprintf(&sb, "  [??] capacités : non vérifiées (%v)\n", r.CapabilitiesProbeErr)
	}
	return sb.String()
}

// writeCapLine écrit une ligne de capacité : [OK] si rien ne manque, sinon [KO]
// avec la liste des composants manquants.
func writeCapLine(sb *strings.Builder, label string, required, missing []string) {
	if len(missing) == 0 {
		fmt.Fprintf(sb, "  [OK] %s : %s\n", label, strings.Join(required, ", "))
		return
	}
	fmt.Fprintf(sb, "  [KO] %s : %s — MANQUANT: %s\n",
		label, strings.Join(required, ", "), strings.Join(missing, ", "))
}

// writeH264Line écrit la ligne de l'encodeur H.264 : [OK] avec l'encodeur retenu, sinon [KO].
func writeH264Line(sb *strings.Builder, r MediaToolingReport) {
	if r.H264Retained != "" {
		fmt.Fprintf(sb, "  [OK] encodeur H.264 : %s (candidats : %s)\n",
			r.H264Retained, strings.Join(h264Names(r.H264Candidates), ", "))
		return
	}
	if !errors.Is(r.H264Err, mediapkg.ErrNoH264Encoder) {
		fmt.Fprintf(sb, "  [??] encodeur H.264 : non vérifié (%v)\n", r.H264Err)
		return
	}
	fmt.Fprintf(sb, "  [KO] encodeur H.264 : aucun utilisable parmi libx264, h264_mf, libopenh264 (%v)\n", r.H264Err)
}

// runFFmpeg exécute `ffmpeg <args>` et retourne sa sortie standard. Les listings
// (-version, -encoders, -muxers) sont écrits sur stdout par ffmpeg ; RunFFmpeg les
// relaie dans un tampon et tient la place des transcodages (PoolTranscode). Le ctx borne la durée
// (le boot passe un timeout).
func runFFmpeg(ctx context.Context, args ...string) (string, error) {
	var out bytes.Buffer
	if err := mediapkg.RunFFmpeg(ctx, mediapkg.FFmpegJob{Args: args, Stdout: &out}); err != nil {
		return "", err
	}
	return out.String(), nil
}

// missingComponents retourne les composants de required absents de present.
func missingComponents(present map[string]bool, required []string) []string {
	var missing []string
	for _, name := range required {
		if !present[name] {
			missing = append(missing, name)
		}
	}
	return missing
}

// firstLine retourne la première ligne non vide (trimmée) de s.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}
