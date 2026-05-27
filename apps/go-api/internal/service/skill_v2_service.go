// Package service — skill_v2_service.go : orchestre lecture/calcul/écriture du LUSR v2.
//
// Délimite la frontière entre la math pure (internal/analysis/skill_v2) et la
// persistance (internal/platform/duckdb/SkillV2Repo). Le caller (typiquement
// le pipeline sync gated par LEVELUP_LUSR_V2_ENABLED) appelle UpdateAfterMatch
// pour chaque match observé, en ordre chronologique.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// SkillV2Service applique le modèle TrueSkill classique (Phase 1) sur les matchs
// observés. Aucune connaissance du schéma DB en dehors de SkillV2Repo, aucune
// connaissance des conventions sync en dehors du paramètre playlistGroup (qui
// arrive précalculé par le caller via internal/sync.GetLUSRChain).
type SkillV2Service struct {
	repo   *duckdb.SkillV2Repo
	priors skillv2.Priors
}

// NewSkillV2Service crée un service prêt à recevoir des matchs.
// Le caller fournit les Priors — soit DefaultPriors() pour démarrer avec les
// constantes du paper, soit des Priors chargés via repo.LoadHyperparams après
// un batch de ré-estimation (Phase 5, non implémentée).
func NewSkillV2Service(repo *duckdb.SkillV2Repo, priors skillv2.Priors) *SkillV2Service {
	return &SkillV2Service{repo: repo, priors: priors}
}

// MatchInput décrit l'observation d'un match prête à être appliquée. Tous les
// XUID des deux équipes doivent être présents : les états des joueurs absents
// de la DB seront initialisés depuis Priors (premier match).
type MatchInput struct {
	MatchID       string
	PlaylistGroup string
	StartTime     time.Time
	TeamAXUIDs    []string
	TeamBXUIDs    []string
	// Outcome décrit l'issue VUE de teamA : Win, Loss ou Draw.
	OutcomeA skillv2.TeamResult
}

// ErrEmptyTeam est renvoyé si une des équipes est vide.
var ErrEmptyTeam = errors.New("skill_v2: match avec une équipe vide")

// UpdateAfterMatch applique un match à l'état latent de tous ses participants.
// Idempotence : la table sous-jacente est append-only — appeler deux fois pour
// le même match produira deux snapshots. C'est au caller (pipeline sync) de
// dédupliquer en filtrant les matchs déjà vus (cf. last_match_id dans state).
func (s *SkillV2Service) UpdateAfterMatch(ctx context.Context, m MatchInput) error {
	if len(m.TeamAXUIDs) == 0 || len(m.TeamBXUIDs) == 0 {
		return fmt.Errorf("%w: match=%s (nA=%d, nB=%d)", ErrEmptyTeam, m.MatchID, len(m.TeamAXUIDs), len(m.TeamBXUIDs))
	}

	// 1. Récupérer les états AVANT-match pour tous les joueurs.
	teamAStates, err := s.loadStates(ctx, m.TeamAXUIDs, m.PlaylistGroup)
	if err != nil {
		return fmt.Errorf("loadStates teamA: %w", err)
	}
	teamBStates, err := s.loadStates(ctx, m.TeamBXUIDs, m.PlaylistGroup)
	if err != nil {
		return fmt.Errorf("loadStates teamB: %w", err)
	}

	// 2. Convertir en Gaussians pour la math pure.
	teamAGauss := statesToGaussians(teamAStates)
	teamBGauss := statesToGaussians(teamBStates)

	// 3. Appliquer la math TrueSkill.
	newA, newB, err := skillv2.UpdateTwoTeam(skillv2.TwoTeamMatch{
		TeamA:   teamAGauss,
		TeamB:   teamBGauss,
		ResultA: m.OutcomeA,
	}, s.priors)
	if err != nil {
		return fmt.Errorf("UpdateTwoTeam: %w", err)
	}

	// 4. Persister les nouveaux états (append-only).
	if err := s.persistTeam(ctx, teamAStates, newA, m); err != nil {
		return fmt.Errorf("persistTeam A: %w", err)
	}
	if err := s.persistTeam(ctx, teamBStates, newB, m); err != nil {
		return fmt.Errorf("persistTeam B: %w", err)
	}
	return nil
}

// PredictWin retourne la probabilité estimée que teamA batte teamB selon le
// modèle, AVANT tout match. Utile pour matchmaking ou métriques de calibration.
// Si un joueur n'a pas encore d'état, il prend Priors.NewPlayerState().
func (s *SkillV2Service) PredictWin(ctx context.Context, playlistGroup string, teamAXUIDs, teamBXUIDs []string) (float64, error) {
	a, err := s.loadStates(ctx, teamAXUIDs, playlistGroup)
	if err != nil {
		return 0, err
	}
	b, err := s.loadStates(ctx, teamBXUIDs, playlistGroup)
	if err != nil {
		return 0, err
	}
	return skillv2.PredictWinProbability(statesToGaussians(a), statesToGaussians(b), s.priors), nil
}

// loadStates récupère l'état latest de chaque XUID ou le crée depuis Priors.
// Préserve l'ordre des XUID en entrée — la math et la persistance s'appuient dessus.
func (s *SkillV2Service) loadStates(ctx context.Context, xuids []string, playlistGroup string) ([]domain.SkillV2State, error) {
	out := make([]domain.SkillV2State, len(xuids))
	for i, x := range xuids {
		st, err := s.repo.LoadState(ctx, x, playlistGroup)
		if err != nil {
			return nil, fmt.Errorf("LoadState(%s, %s): %w", x, playlistGroup, err)
		}
		if st == nil {
			// Premier match pour ce joueur sur ce groupe → priors.
			seed := s.priors.NewPlayerState()
			out[i] = domain.SkillV2State{
				XUID:          x,
				PlaylistGroup: playlistGroup,
				Mu:            seed.Mu,
				Sigma:         seed.Sigma,
			}
			continue
		}
		out[i] = *st
	}
	return out, nil
}

func statesToGaussians(states []domain.SkillV2State) []skillv2.Gaussian {
	out := make([]skillv2.Gaussian, len(states))
	for i, s := range states {
		out[i] = skillv2.Gaussian{Mu: s.Mu, Sigma: s.Sigma}
	}
	return out
}

// persistTeam écrit les nouveaux états de chaque joueur d'une équipe.
// Incrémente experience de 1 et met à jour last_match_id/last_match_at.
func (s *SkillV2Service) persistTeam(ctx context.Context, prior []domain.SkillV2State, posterior []skillv2.Gaussian, m MatchInput) error {
	if len(prior) != len(posterior) {
		return fmt.Errorf("persistTeam: tailles incompatibles (prior=%d, posterior=%d)", len(prior), len(posterior))
	}
	matchID := m.MatchID
	startTime := m.StartTime
	for i, p := range prior {
		next := domain.SkillV2State{
			XUID:          p.XUID,
			PlaylistGroup: p.PlaylistGroup,
			Mu:            posterior[i].Mu,
			Sigma:         posterior[i].Sigma,
			Experience:    p.Experience + 1,
			LastMatchID:   &matchID,
			LastMatchAt:   &startTime,
		}
		if err := s.repo.UpsertState(ctx, next); err != nil {
			return err
		}
	}
	return nil
}
