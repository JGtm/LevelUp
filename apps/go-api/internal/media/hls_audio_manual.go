package media

// hls_audio_manual.go — mapping audio HLS piloté par des rôles DÉCLARÉS (mode manuel).
//
// Quand le joueur a déclaré le rôle de chaque piste source (voix / jeu / autres), on
// court-circuite l'analyse acoustique (NNLS de hls_audio_analyze.go) et on construit
// directement les renditions. Le triplet de sortie reste INCHANGÉ (game / voices /
// full) : `game` = mix des pistes « game », `voices` = mix des pistes « voice » ∪
// « other », `full` = mix de toutes les pistes. Pas de 3e toggle lecteur (différé).
//
// Tout est pur (aucune IO) : testable sans ffmpeg, comme planHLS.

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Rôles de piste tels que reçus dans HLSOptions.ManualAudioRoles. Miroir des valeurs
// domain.AudioTrackRole ("game"/"voice"/"other") — le package media reste découplé du
// domaine, l'appelant convertit AudioTrackRole → string.
const (
	roleGame  = "game"
	roleVoice = "voice"
	roleOther = "other"
)

// buildHLSPlan décide le plan audio d'un transcodage HLS : rôles MANUELS (bypass de
// l'analyse NNLS) quand ils sont fournis ET cohérents avec le nombre de pistes audio
// source, sinon analyse automatique (comportement historique). Un nombre de rôles
// incohérent est ignoré avec un WARN (jamais bloquant). IO ffmpeg seulement en auto.
func buildHLSPlan(ctx context.Context, srcPath string, streams, audioStreams []AVStreamDetail, opts HLSOptions) (hlsPlan, error) {
	if roles := opts.ManualAudioRoles; len(roles) > 0 {
		if len(roles) == len(audioStreams) {
			slog.InfoContext(ctx, "BuildHLS: mapping audio MANUEL (analyse NNLS ignorée)",
				"src", srcPath, "audio_count", len(audioStreams), "roles", roles)
			return planHLSManual(streams, roles)
		}
		slog.WarnContext(ctx, "BuildHLS: rôles audio manuels ignorés (nb pistes incohérent) — analyse auto",
			"src", srcPath, "roles", len(roles), "audio_tracks", len(audioStreams))
	}
	// Piste 0 = mix complet OBS ? Classement acoustique de la composante jeu. Décision
	// IO (décode l'audio), faite ici pour garder planHLS pur.
	var layout audioLayout
	if len(audioStreams) >= 2 {
		l, r2, derr := analyzeAudioLayout(ctx, srcPath, audioStreams)
		if derr != nil {
			slog.WarnContext(ctx, "BuildHLS: analyse pistes audio échouée, mapping par défaut",
				"src", srcPath, "err", derr)
		} else {
			layout = l
			slog.InfoContext(ctx, "BuildHLS: analyse pistes audio",
				"src", srcPath, "audio_count", len(audioStreams), "track0_full_mix", l.Track0FullMix,
				"game_component", l.GameComponent, "fullmix_r2", r2)
		}
	}
	return planHLS(streams, layout)
}

// planHLSManual construit le plan de transcodage à partir de rôles déclarés. Pur :
// aucune IO. Hypothèse (garantie par buildHLSPlan) : len(roles) == nb pistes audio.
func planHLSManual(streams []AVStreamDetail, roles []string) (hlsPlan, error) {
	plan, srcAudios, err := collectStreamsForPlan(streams)
	if err != nil {
		return hlsPlan{}, err
	}
	plan.Audios, plan.FilterComplex = planAudioRenditionsManual(srcAudios, roles)
	plan.VarStreamMap = buildVarStreamMap(plan)
	return plan, nil
}

// planAudioRenditionsManual dérive les renditions game/voices/full des rôles déclarés :
//   - `game`   = mix des pistes de rôle "game" ;
//   - `voices` = mix des pistes de rôle "voice" ou "other" ;
//   - `full`   = mix de TOUTES les pistes (DEFAULT).
//
// Mono-piste → rendition directe unique (pas de toggle). Une seule catégorie présente
// (que du jeu, ou que de la voix — rien à séparer) → mix complet en rendition unique.
// Codec uniformisé AAC dès qu'une rendition provient d'un amix (bascule de piste
// fiable Firefox/Safari). Pur.
func planAudioRenditionsManual(src []AVStreamDetail, roles []string) ([]audioRendition, string) {
	if len(src) < 2 {
		return []audioRendition{singleAudioRendition(src[0])}, ""
	}
	var gameIdx, voiceIdx []int
	for i := range src {
		role := ""
		if i < len(roles) {
			role = strings.ToLower(strings.TrimSpace(roles[i]))
		}
		switch role {
		case roleGame:
			gameIdx = append(gameIdx, i)
		case roleVoice, roleOther:
			voiceIdx = append(voiceIdx, i)
		default:
			// Rôle inconnu (défensif — la validation API l'interdit) → bucket voix : aucune
			// piste n'est perdue, `full` mixe de toute façon toutes les pistes.
			voiceIdx = append(voiceIdx, i)
		}
	}
	if len(gameIdx) == 0 || len(voiceIdx) == 0 {
		return manualSingleFullMix(src)
	}
	game, gameFC := manualComponent(src, gameIdx, audioRenditionGameSlug)
	voices, voicesFC := manualComponent(src, voiceIdx, "voices")
	full := audioRendition{Slug: audioRenditionFullSlug, Display: audioRenditionFullSlug, MapSpec: mapSpecFull, Action: actionReencode, Default: true}
	fullFC := amixFilter(rangeIdx(0, len(src)), audioRenditionFullSlug)
	fc := joinFilters(gameFC, voicesFC, fullFC)
	return []audioRendition{game, voices, full}, fc
}

// manualComponent construit une rendition à partir d'un ensemble d'indices de piste :
// piste unique → map direct (copy si AAC, sinon réencode) ; plusieurs → amix vers le
// label `slug`. Retourne la rendition et son fragment -filter_complex ("" si direct).
func manualComponent(src []AVStreamDetail, idx []int, slug string) (audioRendition, string) {
	r := audioRendition{Slug: slug, Display: slug}
	if len(idx) == 1 {
		r.MapSpec = fmt.Sprintf("0:a:%d", idx[0])
		r.Action = aacUniformAction(src[idx[0]].CodecName)
		r.Language = sanitizeLanguage(src[idx[0]].Language)
		return r, ""
	}
	r.MapSpec = "[" + slug + "]"
	r.Action = actionReencode
	return r, amixFilter(idx, slug)
}

// manualSingleFullMix : une seule catégorie de rôle déclarée (pas de séparation
// jeu/voix possible) → on sert le mix complet en rendition unique.
func manualSingleFullMix(src []AVStreamDetail) ([]audioRendition, string) {
	r := audioRendition{
		Slug: "a0", Display: audioDisplay(src[0], 0),
		MapSpec: mapSpecFull, Action: actionReencode, Default: true,
	}
	return []audioRendition{r}, amixFilter(rangeIdx(0, len(src)), audioRenditionFullSlug)
}

// joinFilters assemble les fragments -filter_complex non vides, séparés par ';'.
func joinFilters(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ";")
}
