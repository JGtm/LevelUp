package port

import (
	"context"

	"levelup/go-api/internal/domain"
)

// SkillV2Repository abstrait l'accès aux états LUSR v2 (player_skill_state_v2)
// et aux hyperparamètres (lusr_hyperparams_v2) tel que consommé par la pipeline
// sync shadow. Implémenté par duckdb.SkillV2Repo.
//
// Surface minimale : seules les 4 méthodes utilisées par le shadow runner sont
// exposées (LoadStateHistory / UpsertHyperparam du repo concret servent aux CLI
// de batch, pas au runtime sync). Découple la logique de calcul du moteur de
// stockage et la rend testable avec un mock.
type SkillV2Repository interface {
	LoadState(ctx context.Context, xuid, playlistGroup string) (*domain.SkillV2State, error)
	LoadAllStates(ctx context.Context, xuid string) ([]domain.SkillV2State, error)
	UpsertState(ctx context.Context, s domain.SkillV2State) error
	LoadHyperparams(ctx context.Context, playlistGroup string) (map[string]float64, error)
}

// SquadOffsetRepository abstrait la lecture des offsets de synergie d'escouade
// (LUSR v2 Sprint 1.C, player_squad_offset) tel que consommé par le shadow
// runner. Implémenté par duckdb.SquadOffsetRepo.
//
// Le runtime sync ne fait que LIRE les offsets (l'écriture vient du CLI
// lusr_v2_squad_estimate), d'où une interface à une seule méthode.
type SquadOffsetRepository interface {
	LoadSquadOffsets(ctx context.Context, xuid, playlistGroup string) (map[string]float64, error)
}
