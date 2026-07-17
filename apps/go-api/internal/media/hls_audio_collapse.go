package media

// hls_audio_collapse.go — correction EN PLACE des arbres HLS multipistes legacy
// dont les renditions `game`/`voices`/`full` portent le même son (clips sans voix
// isolée, ex. sessions solo OBS où la piste 1 « capture de sortie » et les pistes
// composantes contiennent toutes l'audio du jeu). Sur ces clips, `full` = amix doublé
// = écho, et basculer Jeu↔Voix est inaudible (les deux = le jeu). Cf. fix audio HLS
// Firefox juin 2026.
//
// On ne peut pas re-mixer (sources supprimées après transcodage). Mais la rendition
// `game` (piste source 0 = mix de sortie) est une copie PROPRE et complète : il
// suffit de réécrire le master pour n'exposer qu'elle (pas de toggle, pas d'écho).
// Gardé par le même critère de redondance que l'ingestion (corrélation d'enveloppe
// game vs voices) : un vrai clip jeu+voix (renditions distinctes) n'est PAS touché.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CollapseResult résume le traitement d'un arbre HLS.
type CollapseResult struct {
	Dir       string
	Collapsed bool    // master réécrit vers une rendition audio unique
	Corr      float64 // corrélation d'enveloppe game vs voices (si mesurée)
	Skipped   string  // raison du skip (vide si traité)
}

// defaultAttrRe matche l'attribut DEFAULT=YES|NO d'une ligne EXT-X-MEDIA.
var defaultAttrRe = regexp.MustCompile(`DEFAULT=(YES|NO)`)

// collapseMasterToSingleAudio réécrit un master.m3u8 pour ne garder que la rendition
// audio keepSlug (mise en DEFAULT=YES) et supprime les autres lignes
// EXT-X-MEDIA:TYPE=AUDIO. Le groupe `aud` ne contient plus qu'une rendition → le
// lecteur n'affiche pas de sélecteur. Pur. Retourne (master, changed).
func collapseMasterToSingleAudio(master, keepSlug string) (string, bool) {
	lines := strings.Split(master, "\n")
	out := make([]string, 0, len(lines))
	changed := false
	for _, line := range lines {
		if strings.HasPrefix(line, "#EXT-X-MEDIA:") && strings.Contains(line, "TYPE=AUDIO") {
			slug := ""
			if m := uriAttrRe.FindStringSubmatch(line); m != nil {
				slug = renditionSlugFromURI(m[1])
			}
			if slug != keepSlug {
				changed = true
				continue // retire les autres renditions audio
			}
			line = setDefaultYes(line)
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n"), changed
}

// setDefaultYes force DEFAULT=YES sur une ligne EXT-X-MEDIA (remplace DEFAULT=NO ou
// insère l'attribut après TYPE=AUDIO si absent).
func setDefaultYes(line string) string {
	if defaultAttrRe.MatchString(line) {
		return defaultAttrRe.ReplaceAllString(line, "DEFAULT=YES")
	}
	return strings.Replace(line, "TYPE=AUDIO", "TYPE=AUDIO,DEFAULT=YES", 1)
}

// renditionEnvelope décode une rendition (sous-playlist HLS) en enveloppe RMS.
func renditionEnvelope(ctx context.Context, hlsDir, slug string) ([]float64, error) {
	playlist := filepath.ToSlash(filepath.Join(hlsDir, "stream_"+slug+".m3u8"))
	return audioEnvelope(ctx, playlist, "", "0:a:0")
}

// CollapseRedundantHLSAudio réécrit en place le master d'un arbre HLS multipiste si
// ses renditions `game`/`voices` sont redondantes (même contenu → écho sur `full`),
// en ne gardant que `game` (copie propre). Sans effet sur un vrai clip jeu+voix
// (renditions distinctes) ou un clip mono-piste. dryRun=true rapporte sans écrire.
func CollapseRedundantHLSAudio(ctx context.Context, hlsDir string, dryRun bool) (CollapseResult, error) {
	res := CollapseResult{Dir: hlsDir}
	masterPath := filepath.Join(hlsDir, "master.m3u8")
	raw, err := os.ReadFile(masterPath)
	if err != nil {
		return res, fmt.Errorf("lecture master: %w", err)
	}
	master := string(raw)
	slugs := map[string]bool{}
	for _, r := range parseMasterAudioRenditions(master) {
		slugs[r.Slug] = true
	}
	if !slugs["game"] || !slugs["voices"] {
		res.Skipped = "pas un clip multipiste game/voices"
		return res, nil
	}

	envGame, err := renditionEnvelope(ctx, hlsDir, "game")
	if err != nil {
		return res, fmt.Errorf("enveloppe game: %w", err)
	}
	envVoices, err := renditionEnvelope(ctx, hlsDir, "voices")
	if err != nil {
		return res, fmt.Errorf("enveloppe voices: %w", err)
	}
	res.Corr = pearson(envGame, envVoices)
	if stdDev(envGame) < minEnvelopeStdDevDB || stdDev(envVoices) < minEnvelopeStdDevDB {
		res.Skipped = "enveloppe trop stationnaire (corrélation non fiable)"
		return res, nil
	}
	if res.Corr < fullMixEnvelopeCorrThreshold {
		res.Skipped = "renditions distinctes (toggle conservé)"
		return res, nil
	}

	newMaster, changed := collapseMasterToSingleAudio(master, "game")
	if !changed {
		res.Skipped = "rien à collapser"
		return res, nil
	}
	res.Collapsed = true
	if dryRun {
		return res, nil
	}
	return res, writeMasterAtomic(masterPath, newMaster)
}

// ForceCollapseHLSAudioTree réécrit le master d'UN arbre HLS multipiste pour n'exposer
// que la rendition `game` (copie propre = piste 0), SANS le critère de corrélation de
// CollapseRedundantHLSAudio. Destiné aux clips legacy dont la redondance est STRUCTURELLE
// (game = piste 0 = mix complet AVEC voix, voices = amix redondant) mais dont la
// corrélation d'enveloppe tombe sous le seuil à cause de la voix, ET dont les sources
// sont supprimées (ni re-transcodage ni collapse auto possibles). N'exige pas ffmpeg
// (pas de mesure d'enveloppe). dryRun=true rapporte sans écrire. Réutilise la mécanique
// de CollapseRedundantHLSAudio (collapseMasterToSingleAudio + writeMasterAtomic).
func ForceCollapseHLSAudioTree(hlsDir string, dryRun bool) (CollapseResult, error) {
	res := CollapseResult{Dir: hlsDir}
	masterPath := filepath.Join(hlsDir, "master.m3u8")
	raw, err := os.ReadFile(masterPath)
	if err != nil {
		return res, fmt.Errorf("lecture master: %w", err)
	}
	master := string(raw)
	hasGame := false
	for _, r := range parseMasterAudioRenditions(master) {
		if r.Slug == audioRenditionGameSlug {
			hasGame = true
		}
	}
	if !hasGame {
		res.Skipped = "pas de rendition game à conserver"
		return res, nil
	}
	newMaster, changed := collapseMasterToSingleAudio(master, audioRenditionGameSlug)
	if !changed {
		res.Skipped = "déjà mono-audio (game seul)"
		return res, nil
	}
	res.Collapsed = true
	if dryRun {
		return res, nil
	}
	return res, writeMasterAtomic(masterPath, newMaster)
}

// writeMasterAtomic écrit le master via un fichier temporaire + rename (évite un
// master tronqué si l'écriture est interrompue).
func writeMasterAtomic(masterPath, content string) error {
	tmp := masterPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("écriture master tmp: %w", err)
	}
	if err := os.Rename(tmp, masterPath); err != nil {
		return fmt.Errorf("rename master: %w", err)
	}
	return nil
}
