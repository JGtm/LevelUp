package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// SoloLivesRepository — les vies, morts situées et frags d'UN joueur sur une fenêtre de matchs
// (carte « Mes vies : près d'un coéquipier ou seul » des Séries temporelles) : UN chargement par
// requête, borné par les matchs de la fenêtre sur le `match_id` de chaque vue `_latest` et par le
// joueur après la fenêtre (ADR 0036 I2) — `match_lives_latest`, `match_death_context_latest`,
// `match_kill_events_latest`, plus les camps (`match_participants`) et la variante de chaque match
// (`match_registry.game_variant_name`, la clé de la portée du radar).
//
// Implémenté par internal/platform/duckdb.SoloLivesRepo, câblé sous `film.kill_positions` (la porte
// des vies et du contexte des morts au sync). Table absente : games.ErrCapabilityNotSupported.
type SoloLivesRepository interface {
	LoadLivesNearTeammate(ctx context.Context, matchIDs []string, xuid string) (domain.ViesLues, error)
}
