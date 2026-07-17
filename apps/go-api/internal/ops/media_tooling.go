// Package ops — media_tooling.go : vérification de l'outillage média
// (ffmpeg/ffprobe) au-delà de la simple présence dans le PATH.
//
// L'app exécute ffmpeg/ffprobe pour trois surfaces :
//   - miniatures WebP animées      (internal/ops/media_thumbnails.go : encodeur libwebp) ;
//   - transcodage HLS-fMP4         (internal/media/hls.go : muxer hls + mp4, encodeurs libx264/aac) ;
//   - remux live des clips MKV/AVI (internal/media/remux.go).
//
// Un ffmpeg présent mais compilé sans libx264/libwebp/aac ou sans le muxer hls
// échoue silencieusement au premier upload. Ce fichier rend ce risque observable
// à deux endroits, sans dupliquer les execs :
//   - au boot serveur (cmd/server)  : LogMediaToolingStatus — slog non-bloquant ;
//   - dans la CLI `check-env`        : InspectMediaTooling + Report.Summary (stdout).
//
// Le PARSING des sorties ffmpeg (parseFFmpegComponents, missingComponents,
// firstLine) est PUR et testable avec une sortie simulée — aucun test ne dépend
// d'un ffmpeg réellement installé (cf. media_tooling_test.go).
package ops

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"

	"levelup/go-api/internal/domain"
)

// requiredEncoders liste les encodeurs ffmpeg dont dépend LevelUp :
//   - libwebp : miniatures animées WebP (media_thumbnails.go, -c:v libwebp) ;
//   - libx264 : ré-encodage vidéo H.264 quand la copy fMP4 est impossible (hls.go, -c:v libx264) ;
//   - aac     : ré-encodage audio des renditions HLS (hls.go, -c:a aac).
var requiredEncoders = []string{"libwebp", "libx264", "aac"}

// requiredMuxers liste les muxers ffmpeg requis :
//   - hls : arbre HLS (master + sous-playlists), hls.go (-f hls) ;
//   - mp4 : conteneur des segments/inits fMP4 — `-hls_segment_type fmp4` s'appuie
//     sur le muxer mp4 fragmenté ; ffmpeg n'expose pas de muxer "fmp4" distinct.
var requiredMuxers = []string{"hls", "mp4"}

// mediaComponentImpact décrit la surface produit cassée par l'absence de chaque
// composant — sert de contexte actionnable dans les logs et la sortie check-env.
var mediaComponentImpact = map[string]string{
	"libwebp": "miniatures animées WebP",
	"libx264": "ré-encodage vidéo H.264 (HLS quand la copy est impossible)",
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
}

// InspectMediaTooling résout ffmpeg/ffprobe dans le PATH puis, si ffmpeg est
// présent, interroge sa version et ses capacités (encoders + muxers). Un seul
// exec par sous-commande (`-version`, `-encoders`, `-muxers`). Best-effort :
// n'échoue jamais, ne bloque jamais — tout est reflété dans le rapport.
func InspectMediaTooling(ctx context.Context) MediaToolingReport {
	var r MediaToolingReport
	r.FFmpegPath, r.FFmpegFound = lookupBinary("ffmpeg")
	r.FFprobePath, r.FFprobeFound = lookupBinary("ffprobe")
	if !r.FFmpegFound {
		return r // sans ffmpeg, impossible d'interroger version/capacités
	}
	if verOut, err := runFFmpeg(ctx, "-version"); err == nil {
		r.FFmpegVersion = firstLine(verOut)
	}
	encOut, encErr := runFFmpeg(ctx, "-hide_banner", "-encoders")
	muxOut, muxErr := runFFmpeg(ctx, "-hide_banner", "-muxers")
	switch {
	case encErr != nil:
		r.CapabilitiesProbeErr = encErr
	case muxErr != nil:
		r.CapabilitiesProbeErr = muxErr
	default:
		r.CapabilitiesProbed = true
		r.MissingEncoders = missingComponents(parseFFmpegComponents(encOut), requiredEncoders)
		r.MissingMuxers = missingComponents(parseFFmpegComponents(muxOut), requiredMuxers)
	}
	return r
}

// LogMediaToolingStatus émet l'état de l'outillage média au boot serveur, en
// slog structuré et NON-bloquant : slog.InfoContext si ffmpeg/ffprobe présents
// (avec la version), slog.WarnContext actionnable si l'un manque, puis un WARN
// par composant (encodeur/muxer) requis manquant. Le serveur démarre quoi qu'il
// arrive — cette fonction ne renvoie rien et ne panique jamais.
func LogMediaToolingStatus(ctx context.Context) {
	r := InspectMediaTooling(ctx)
	if r.FFmpegFound && r.FFprobeFound {
		slog.InfoContext(ctx, "media tooling: ffmpeg/ffprobe disponibles",
			"ffmpeg_version", r.FFmpegVersion,
			"ffmpeg_path", r.FFmpegPath,
			"ffprobe_path", r.FFprobePath)
	} else {
		slog.WarnContext(ctx, "media tooling: ffmpeg/ffprobe absent du PATH — miniatures, transcodage HLS et remux live indisponibles ; installer ffmpeg",
			"ffmpeg_found", r.FFmpegFound,
			"ffprobe_found", r.FFprobeFound)
	}
	if r.CapabilitiesProbeErr != nil {
		slog.WarnContext(ctx, "media tooling: interrogation des capacités ffmpeg échouée — encoders/muxers non vérifiés",
			"err", r.CapabilitiesProbeErr)
		return
	}
	for _, name := range r.MissingEncoders {
		slog.WarnContext(ctx, "media tooling: encodeur ffmpeg requis manquant",
			"component", name, "impact", mediaComponentImpact[name],
			"hint", "installer un build ffmpeg incluant l'encodeur "+name)
	}
	for _, name := range r.MissingMuxers {
		slog.WarnContext(ctx, "media tooling: muxer ffmpeg requis manquant",
			"component", name, "impact", mediaComponentImpact[name],
			"hint", "installer un build ffmpeg incluant le muxer "+name)
	}
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
		sb.WriteString("  [KO] ffmpeg  : introuvable dans le PATH\n")
	}
	if r.FFprobeFound {
		fmt.Fprintf(&sb, "  [OK] ffprobe : %s\n", r.FFprobePath)
	} else {
		sb.WriteString("  [KO] ffprobe : introuvable dans le PATH\n")
	}
	switch {
	case r.CapabilitiesProbed:
		writeCapLine(&sb, "encodeurs requis", requiredEncoders, r.MissingEncoders)
		writeCapLine(&sb, "muxers requis   ", requiredMuxers, r.MissingMuxers)
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

// runFFmpeg exécute `ffmpeg <args>` et retourne sa sortie standard. Les listings
// (-version, -encoders, -muxers) sont écrits sur stdout par ffmpeg ; .Output()
// les capture. Le ctx borne la durée (le boot passe un timeout).
func runFFmpeg(ctx context.Context, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, "ffmpeg", args...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// parseFFmpegComponents extrait l'ensemble des noms de composants d'une sortie
// `ffmpeg -encoders` ou `ffmpeg -muxers`. Chaque ligne de composant a la forme
// "<flags> <name>  <description>" (flags = lettres majuscules + points, ex
// "V....D" pour un encodeur, "E"/"DE" pour un muxer) ; le nom est le 2e champ.
// Les en-têtes et la légende (« V..... = Video ») sont écartés car leur 2e champ
// vaut "=" ou leur 1er champ n'est pas un jeu de flags. Pur — aucune IO.
func parseFFmpegComponents(output string) map[string]bool {
	set := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		flags, name := fields[0], fields[1]
		if name == "=" || !isFlagsToken(flags) {
			continue
		}
		set[name] = true
	}
	return set
}

// isFlagsToken retourne true si tok est un jeu de flags ffmpeg (1 à 6 caractères,
// uniquement lettres majuscules et points). Les noms de composants sont en
// minuscules → aucune ambiguïté avec un nom.
func isFlagsToken(tok string) bool {
	if tok == "" || len(tok) > 6 {
		return false
	}
	for _, r := range tok {
		if r != '.' && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
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
