// Package wire — sync_history_reader.go : la lecture de l'état de l'historique servie par le
// statut de synchronisation d'un joueur.
package wire

import (
	"fmt"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	go_sync "levelup/go-api/internal/sync"
)

// NewSyncHistoryReader rend le lecteur de la clé de fin d'historique (sync_meta de la player DB du
// joueur pour le titre demandé). Le chemin de la base se résout par PathResolver depuis le profil
// suivi dans ce titre ; un joueur absent du titre est une erreur, que le service sert comme
// « non complet » après l'avoir journalisée.
func NewSyncHistoryReader(cfg *config.AppConfig) *duckdb.SyncHistoryRepo {
	return duckdb.NewSyncHistoryRepo(func(playerSlug, titleSlug string) (string, error) {
		players, err := cfg.LoadPlayers(titleSlug)
		if err != nil {
			return "", err
		}
		for i := range players {
			if players[i].PlayerSlug == playerSlug {
				return cfg.PathResolver().PlayerDBPath(titleSlug, players[i].Gamertag), nil
			}
		}
		return "", fmt.Errorf("%w: slug=%q titre=%q", config.ErrPlayerNotFound, playerSlug, titleSlug)
	}, go_sync.SyncMetaHistoryCompleteAt)
}
