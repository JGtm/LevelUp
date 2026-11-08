// Package service — media_scan_hook.go : la passe d'indexation des médias d'un joueur, câblée sur les
// réglages vivants.
package service

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/config"
	settings_platform "levelup/go-api/internal/platform/settings"
)

// NewLiveMediaScanHook rend la passe d'indexation des médias de gamertag ([BuildMediaScanHook]) dont
// le dossier des captures, le fuseau et la politique de rétention de l'original sont relus à
// CHAQUE appel dans store : un réglage modifié à chaud prend effet sans redémarrage.
//
// C'est le câblage commun du post-sync, de l'auto-sync, de la synchronisation HTTP, de la tâche de
// démarrage et du rescan qui suit une confirmation : aucun appelant ne recopie les trois lectures.
// store nul : aucun dossier ni fuseau configuré, la rétention suit cfg.MediaDeleteSource(nil).
func NewLiveMediaScanHook(cfg *config.AppConfig, store *settings_platform.Store, gamertag string) func(ctx context.Context) {
	load := func() *settings_platform.AppSettings {
		if store == nil {
			return nil
		}
		s, err := store.Load()
		if err != nil {
			slog.WarnContext(context.Background(), "media: réglages illisibles, valeurs par défaut", "err", err)
			return nil
		}
		return s
	}
	return BuildMediaScanHook(cfg.Roots().Resources, gamertag,
		func() string {
			if s := load(); s != nil {
				return s.MediaCapturesBaseDir
			}
			return ""
		},
		func() string {
			if s := load(); s != nil {
				return s.UserTimezone
			}
			return ""
		},
		func() bool {
			var v *bool
			if s := load(); s != nil {
				v = s.MediaDeleteSourceAfterTranscode
			}
			return cfg.MediaDeleteSource(v)
		},
	)
}
