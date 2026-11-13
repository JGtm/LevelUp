// Package wire — live_runner_builder.go : la sélection du runner de suivi en direct d'un titre pour
// la sync initiale lancée depuis l'API.
package wire

import (
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/games/halo_5/livesync"
	"levelup/go-api/internal/service"
)

// NewLiveRunnerBuilder rend le LiveRunnerBuilder du service des jobs de sync : le runner du titre quand
// `livesync` en câble un, nil quand le titre passe par le moteur de sync. Le résultat est une
// interface nil dans ce second cas (un *Runner nil rangé dans l'interface serait non nil).
func NewLiveRunnerBuilder(cfg *config.AppConfig) service.LiveRunnerBuilder {
	return func(titleSlug, gamertag, xuid string) service.DeltaRunner {
		if r := livesync.RunnerForTitle(titleSlug, cfg, gamertag, xuid); r != nil {
			return r
		}
		return nil
	}
}
