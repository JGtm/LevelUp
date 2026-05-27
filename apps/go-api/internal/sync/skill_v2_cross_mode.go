package sync

// skill_v2_cross_mode.go — Phase 4 (mode correlation) : leak post-EP du
// delta μ vers les autres playlist_groups du même joueur, avec cap w_d ≤ 0.4.
//
// Extrait de skill_v2_shadow.go (2026-05-27). Activation gated par env
// `LEVELUP_LUSR_V2_MODE_COUPLING=1`. Cf. .ai/LUSR_V2_HANDOFF.md "Mode
// correlation cap w_d ≤ 0.4".

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// lusrModeCouplingEnvFlag active la Phase 4 (cross-mode leak). Off = chaque
// playlist_group reste totalement indépendant (comportement Phase 1-3). On =
// applique le leak avec w_d = DefaultModeCouplingWeight après chaque update.
//   - "1" / "true" / "yes" : actif
//   - vide / autre         : inactif (défaut)
const lusrModeCouplingEnvFlag = "LEVELUP_LUSR_V2_MODE_COUPLING"

// IsLUSRV2ModeCouplingEnabled retourne true si la Phase 4 (cross-mode leak)
// doit être appliquée après chaque update. Cf. lusrModeCouplingEnvFlag.
func IsLUSRV2ModeCouplingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(lusrModeCouplingEnvFlag)))
	return v == "1" || v == "true" || v == "yes"
}

// findOwnerPrior cherche l'état AVANT-match du owner dans les deux rosters.
// Retourne nil si owner pas trouvé (cas dégénéré).
func findOwnerPrior(ownerXUID string, priorA, priorB []domain.SkillV2State) *domain.SkillV2State {
	for i := range priorA {
		if priorA[i].XUID == ownerXUID {
			return &priorA[i]
		}
	}
	for i := range priorB {
		if priorB[i].XUID == ownerXUID {
			return &priorB[i]
		}
	}
	return nil
}

// propagateCrossModeLeak applique la Phase 4 : décale les μ des autres modes
// du owner par w_d · (μ_new - μ_old) dans le mode primaire. σ inchangé.
// Aucun écho sur les coéquipiers/adversaires — la mesure du leak nécessite
// une décision UX par joueur (cf. ADR à venir).
func propagateCrossModeLeak(ctx context.Context, repo *duckdb.SkillV2Repo,
	ownerPrior, ownerNew domain.SkillV2State) error {
	allStates, err := repo.LoadAllStates(ctx, ownerPrior.XUID)
	if err != nil {
		return fmt.Errorf("LoadAllStates: %w", err)
	}
	leaked := 0
	for _, s := range allStates {
		if s.PlaylistGroup == ownerPrior.PlaylistGroup {
			continue // déjà écrit par persistTeamSkillV2
		}
		newMu := skillv2.ApplyCrossModeLeak(s.Mu, ownerPrior.Mu, ownerNew.Mu,
			skillv2.DefaultModeCouplingWeight)
		shifted := s
		shifted.Mu = newMu
		// LastMatchID/At ne changent pas — c'est un leak, pas un nouveau match.
		if err := repo.UpsertState(ctx, shifted); err != nil {
			return fmt.Errorf("UpsertState leak %s: %w", s.PlaylistGroup, err)
		}
		leaked++
	}
	if leaked > 0 {
		slog.DebugContext(ctx, "Phase 4 cross-mode leak appliqué",
			"xuid", ownerPrior.XUID,
			"primary_group", ownerPrior.PlaylistGroup,
			"delta_mu", ownerNew.Mu-ownerPrior.Mu,
			"weight", skillv2.DefaultModeCouplingWeight,
			"groups_shifted", leaked,
		)
	}
	return nil
}
