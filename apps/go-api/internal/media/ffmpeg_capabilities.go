package media

// ffmpeg_capabilities.go : liste des encodeurs que le ffmpeg résolu sait produire, lue une fois
// par chemin de binaire et gardée en mémoire. Source unique du parsing de `ffmpeg -encoders` /
// `ffmpeg -muxers` (la sonde d'outillage de internal/ops et le choix de l'encodeur H.264 la
// consomment).

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
)

// capabilitiesCache garde la dernière liste d'encodeurs lue avec succès, avec le chemin du
// binaire qui l'a produite : un autre chemin (variable d'environnement changée, binaire remplacé
// par un autre dossier) relance la lecture. Le verrou ne couvre jamais le lancement de ffmpeg :
// un appelant qui tient déjà la place des transcodages ne doit pas attendre un autre qui la
// demande.
type capabilitiesCache struct {
	mu       sync.Mutex
	path     string
	encoders map[string]bool
	h264     H264Encoder
	h264Path string
}

var capabilities capabilitiesCache

// AvailableEncoders rend l'ensemble des encodeurs listés par `ffmpeg -hide_banner -encoders`.
// Le résultat d'une lecture réussie est gardé pour le chemin résolu de ffmpeg ; un échec n'est
// pas gardé (la lecture suivante réessaie). L'ensemble rendu est partagé : ne pas le modifier.
func AvailableEncoders(ctx context.Context) (map[string]bool, error) {
	path, err := ResolveFFmpeg()
	if err != nil {
		return nil, err
	}
	capabilities.mu.Lock()
	if capabilities.path == path && capabilities.encoders != nil {
		set := capabilities.encoders
		capabilities.mu.Unlock()
		return set, nil
	}
	capabilities.mu.Unlock()

	var out bytes.Buffer
	if err := RunFFmpeg(ctx, FFmpegJob{Args: []string{"-hide_banner", "-encoders"}, Stdout: &out}); err != nil {
		return nil, fmt.Errorf("lecture des encodeurs ffmpeg: %w", err)
	}
	set := ParseFFmpegComponents(out.String())
	capabilities.mu.Lock()
	if capabilities.path != path {
		capabilities.h264, capabilities.h264Path = "", ""
	}
	capabilities.path, capabilities.encoders = path, set
	capabilities.mu.Unlock()
	return set, nil
}

// resetCapabilitiesCache vide les caches (tests).
func resetCapabilitiesCache() {
	capabilities.mu.Lock()
	capabilities.path, capabilities.encoders = "", nil
	capabilities.h264, capabilities.h264Path = "", ""
	capabilities.mu.Unlock()
}

// ParseFFmpegComponents extrait l'ensemble des noms de composants d'une sortie
// `ffmpeg -encoders` ou `ffmpeg -muxers`. Chaque ligne de composant a la forme
// "<flags> <name>  <description>" (flags = lettres majuscules + points, ex
// "V....D" pour un encodeur, "E"/"DE" pour un muxer) ; le nom est le 2e champ.
// Les en-têtes et la légende (« V..... = Video ») sont écartés car leur 2e champ
// vaut "=" ou leur 1er champ n'est pas un jeu de flags. Pur : aucune entrée-sortie.
func ParseFFmpegComponents(output string) map[string]bool {
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

// isFlagsToken dit si tok est un jeu de flags ffmpeg (1 à 6 caractères, uniquement lettres
// majuscules et points). Les noms de composants sont en minuscules : aucune ambiguïté.
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
