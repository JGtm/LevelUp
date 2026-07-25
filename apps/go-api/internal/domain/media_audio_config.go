// Package domain — media_audio_config.go : réglage par joueur du rôle des pistes
// audio de ses médias (voix / jeu / autres), en mode automatique (l'analyse NNLS
// existante décide) ou manuel (le joueur déclare l'ordre de ses pistes source).
//
// Rationale : l'ordre des pistes exportées par OBS d'un joueur est stable dans le
// temps ; un réglage par joueur s'applique donc à tous ses futurs transcodages HLS.
// Le stockage est un sidecar JSON par joueur (réglage rare, hors périmètre anti-ART),
// jamais DuckDB.
package domain

import (
	"fmt"
	"time"
)

// MaxAudioTrackRoles borne la taille de TrackRoles en mode manuel. Un enregistrement
// multipiste réaliste (jeu + micro + Discord + ...) dépasse rarement 4 pistes ; 16 est
// une garde large contre un payload aberrant.
const MaxAudioTrackRoles = 16

// AudioTrackRole est le rôle acoustique déclaré d'une piste audio source.
type AudioTrackRole string

const (
	// AudioTrackRoleGame : son du jeu (rendition `game`).
	AudioTrackRoleGame AudioTrackRole = "game"
	// AudioTrackRoleVoice : voix du joueur / micro (rendition `voices`).
	AudioTrackRoleVoice AudioTrackRole = "voice"
	// AudioTrackRoleOther : autre source (Discord, musique, ...). Mixée avec la
	// voix dans la rendition `voices` (pas de 3e toggle lecteur — décision différée).
	AudioTrackRoleOther AudioTrackRole = "other"
)

// Valid indique si le rôle est l'une des valeurs connues.
func (r AudioTrackRole) Valid() bool {
	switch r {
	case AudioTrackRoleGame, AudioTrackRoleVoice, AudioTrackRoleOther:
		return true
	default:
		return false
	}
}

// MediaAudioMode est le mode de résolution du rôle des pistes audio.
type MediaAudioMode string

const (
	// MediaAudioModeAuto : l'analyse acoustique (NNLS) décide à l'ingestion (défaut).
	MediaAudioModeAuto MediaAudioMode = "auto"
	// MediaAudioModeManual : les rôles déclarés (TrackRoles) font foi, l'analyse
	// est court-circuitée.
	MediaAudioModeManual MediaAudioMode = "manual"
)

// PlayerMediaAudioConfig est le réglage audio média d'un joueur (sidecar JSON).
//
// TrackRoles est indexé par piste audio SOURCE dans l'ordre ffprobe (TrackRoles[0]
// = 0:a:0, ...). Vide en mode auto.
type PlayerMediaAudioConfig struct {
	Mode       MediaAudioMode   `json:"mode" enum:"auto,manual" doc:"Mode de résolution du rôle des pistes audio."`
	TrackRoles []AudioTrackRole `json:"track_roles,omitempty" enum:"game,voice,other" doc:"Rôle de chaque piste audio source. Requis et non vide en mode manuel."`
	UpdatedAt  time.Time        `json:"updated_at" doc:"Horodatage serveur de la dernière écriture (ignoré en entrée)."`
}

// DefaultPlayerMediaAudioConfig retourne le réglage par défaut (mode auto) appliqué
// quand aucun sidecar n'existe pour le joueur.
func DefaultPlayerMediaAudioConfig() PlayerMediaAudioConfig {
	return PlayerMediaAudioConfig{Mode: MediaAudioModeAuto}
}

// Validate vérifie la cohérence d'un réglage reçu via l'API PUT :
//   - mode dans l'énumération connue ;
//   - en mode manuel : au moins une piste, au plus MaxAudioTrackRoles, tous les
//     rôles valides.
//
// En mode auto, TrackRoles est ignoré (peut être vide ou renseigné — non contraignant).
func (c PlayerMediaAudioConfig) Validate() error {
	switch c.Mode {
	case MediaAudioModeAuto:
		return nil
	case MediaAudioModeManual:
		return c.validateManualRoles()
	default:
		return fmt.Errorf("mode invalide: %q (attendu %q ou %q)", c.Mode, MediaAudioModeAuto, MediaAudioModeManual)
	}
}

// validateManualRoles applique les contraintes propres au mode manuel.
func (c PlayerMediaAudioConfig) validateManualRoles() error {
	if len(c.TrackRoles) == 0 {
		return fmt.Errorf("track_roles requis en mode manuel")
	}
	if len(c.TrackRoles) > MaxAudioTrackRoles {
		return fmt.Errorf("track_roles trop long: %d (max %d)", len(c.TrackRoles), MaxAudioTrackRoles)
	}
	for i, role := range c.TrackRoles {
		if !role.Valid() {
			return fmt.Errorf("rôle invalide à la piste %d: %q", i, role)
		}
	}
	return nil
}
