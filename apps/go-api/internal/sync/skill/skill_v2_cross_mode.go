package skill

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
	"levelup/go-api/internal/port"
)

// lusrModeCouplingEnvFlag contrôle la Phase 4 (cross-mode leak). **ON par défaut**
// (décision produit 2026-05-28) : le leak applique le poids de la matrice de
// corrélation (Sprint 2.B), avec fallback w_d = DefaultModeCouplingWeight tant que
// la matrice n'est pas calculée. Mettre explicitement "0"/"false"/"no" pour
// désactiver.
const lusrModeCouplingEnvFlag = "LEVELUP_LUSR_V2_MODE_COUPLING"

// IsLUSRV2ModeCouplingEnabled retourne true sauf si le flag est explicitement
// désactivé ("0"/"false"/"no"). ON par défaut. Cf. lusrModeCouplingEnvFlag.
func IsLUSRV2ModeCouplingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(lusrModeCouplingEnvFlag)))
	return v != "0" && v != "false" && v != "no"
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
func propagateCrossModeLeak(ctx context.Context, repo port.SkillV2Repository,
	ownerPrior, ownerNew domain.SkillV2State) error {
	allStates, err := repo.LoadAllStates(ctx, ownerPrior.XUID)
	if err != nil {
		return fmt.Errorf("LoadAllStates: %w", err)
	}
	// Sprint 2.B : poids de couplage par paire de modes (matrice empirique du
	// batch, rows mode_coupling_<source>_<target> dans lusr_hyperparams_v2).
	// Best-effort : si le chargement échoue ou si la paire n'a pas d'entrée, on
	// retombe sur le scalaire DefaultModeCouplingWeight (comportement Phase 4).
	coupling, err := repo.LoadHyperparams(ctx, ownerPrior.PlaylistGroup)
	if err != nil {
		slog.WarnContext(ctx, "Phase 4: LoadHyperparams couplage échoué — scalaire par défaut",
			"group", ownerPrior.PlaylistGroup, "err", err)
		coupling = nil
	}
	leaked := 0
	for _, s := range allStates {
		if s.PlaylistGroup == ownerPrior.PlaylistGroup {
			continue // déjà écrit par persistTeamSkillV2
		}
		w := skillv2.CouplingWeightFor(coupling, ownerPrior.PlaylistGroup, s.PlaylistGroup,
			skillv2.DefaultModeCouplingWeight)
		newMu := skillv2.ApplyCrossModeLeak(s.Mu, ownerPrior.Mu, ownerNew.Mu, w)
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
			"groups_shifted", leaked,
		)
	}
	return nil
}
