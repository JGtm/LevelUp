// Package mediaaudio lit/écrit le sidecar JSON de réglage audio média par joueur
// (data/titles/{slug}/players/{gamertag}/media_audio_config.json).
//
// Modèle platform/settings.Store, SANS base+overlay : un fichier par joueur, réglage
// rare. Écriture atomique (fichier temporaire + rename) pour ne jamais laisser un
// sidecar tronqué. Sidecar absent → réglage par défaut (mode auto).
package mediaaudio

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// Store gère la lecture/écriture d'un sidecar media_audio_config.json.
type Store struct {
	mu   sync.RWMutex
	path string
}

// NewStore crée un Store pour le fichier sidecar donné.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Load lit le sidecar. Fichier absent → réglage par défaut (mode auto) sans erreur.
// JSON illisible → erreur (le réglage par défaut est aussi retourné pour permettre
// une dégradation gracieuse côté appelant).
func (s *Store) Load() (domain.PlayerMediaAudioConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.DefaultPlayerMediaAudioConfig(), nil
		}
		return domain.DefaultPlayerMediaAudioConfig(), fmt.Errorf("mediaaudio.Load %q: %w", s.path, err)
	}
	var cfg domain.PlayerMediaAudioConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return domain.DefaultPlayerMediaAudioConfig(), fmt.Errorf("mediaaudio.Load parse %q: %w", s.path, err)
	}
	if cfg.Mode == "" {
		cfg.Mode = domain.MediaAudioModeAuto
	}
	return cfg, nil
}

// Save écrit le sidecar de façon atomique (temp + rename). Crée le dossier parent
// si besoin.
func (s *Store) Save(cfg domain.PlayerMediaAudioConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("mediaaudio.Save mkdir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("mediaaudio.Save marshal: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("mediaaudio.Save write temp: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("mediaaudio.Save rename: %w", err)
	}
	return nil
}

// ManualRolesForPlayer charge le sidecar d'un joueur et retourne les rôles de piste
// (chaînes "game"/"voice"/"other") quand le mode est MANUEL ; nil sinon (mode auto).
//
// Un sidecar illisible est LOGGÉ et traité comme auto : le transcodage HLS n'est
// jamais bloqué par un réglage corrompu. Helper commun aux call-sites du pipeline
// (upload, balayage) qui doivent charger le réglage AVANT BuildHLS.
func ManualRolesForPlayer(ctx context.Context, paths *titlePkg.PathResolver, titleSlug, gamertag string) []string {
	cfg, err := NewStore(paths.PlayerMediaAudioConfigPath(titleSlug, gamertag)).Load()
	if err != nil {
		slog.WarnContext(ctx, "mediaaudio: sidecar illisible, mode auto",
			"titleSlug", titleSlug, "gamertag", gamertag, "err", err)
		return nil
	}
	if cfg.Mode != domain.MediaAudioModeManual {
		return nil
	}
	roles := make([]string, len(cfg.TrackRoles))
	for i, r := range cfg.TrackRoles {
		roles[i] = string(r)
	}
	return roles
}
