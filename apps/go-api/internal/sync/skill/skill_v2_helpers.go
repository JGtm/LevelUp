package skill

// skill_v2_helpers.go — petits helpers utilisés par le shadow runner :
// conversions de types, load-or-seed, persist par équipe. Extraits de
// skill_v2_shadow.go (2026-05-27) pour respecter le seuil 500L.

import (
	"context"
	"fmt"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

func extractXUIDs(roster []rosterMember) []string {
	out := make([]string, len(roster))
	for i, m := range roster {
		out[i] = m.xuid
	}
	return out
}

func loadStatesOrSeed(ctx context.Context, repo port.SkillV2Repository, xuids []string, playlistGroup string, priors skillv2.Priors) ([]domain.SkillV2State, error) {
	out := make([]domain.SkillV2State, len(xuids))
	for i, x := range xuids {
		st, err := repo.LoadState(ctx, x, playlistGroup)
		if err != nil {
			return nil, err
		}
		if st == nil {
			seed := priors.NewPlayerState()
			out[i] = domain.SkillV2State{
				XUID: x, PlaylistGroup: playlistGroup,
				Mu: seed.Mu, Sigma: seed.Sigma,
			}
			continue
		}
		out[i] = *st
	}
	return out, nil
}

func shadowStatesToGaussians(states []domain.SkillV2State) []skillv2.Gaussian {
	out := make([]skillv2.Gaussian, len(states))
	for i, s := range states {
		out[i] = skillv2.Gaussian{Mu: s.Mu, Sigma: s.Sigma}
	}
	return out
}

func persistTeamSkillV2(ctx context.Context, repo port.SkillV2Repository, prior []domain.SkillV2State, posterior []skillv2.Gaussian, matchID string, startTime time.Time) error {
	if len(prior) != len(posterior) {
		return fmt.Errorf("persistTeamSkillV2: tailles incompatibles (prior=%d, posterior=%d)", len(prior), len(posterior))
	}
	mid := matchID
	st := startTime
	for i, p := range prior {
		next := domain.SkillV2State{
			XUID: p.XUID, PlaylistGroup: p.PlaylistGroup,
			Mu: posterior[i].Mu, Sigma: posterior[i].Sigma,
			Experience:  p.Experience + 1,
			LastMatchID: &mid,
			LastMatchAt: &st,
		}
		if err := repo.UpsertState(ctx, next); err != nil {
			return err
		}
	}
	return nil
}
