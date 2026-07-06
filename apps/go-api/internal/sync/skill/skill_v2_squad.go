package skill

// skill_v2_squad.go — Sprint 1.C : application des offsets de synergie d'escouade
// au runtime du shadow runner.
//
// Activation gated par env `LEVELUP_LUSR_V2_SQUAD_OFFSET=1` (OFF par défaut) :
// tant que le flag est off, le repo squad est nil et les offsets sont nuls →
// comportement strictement identique aux phases précédentes (zéro risque prod).
//
// Quand activé : pour chaque joueur, on somme les offsets de synergie avec ses
// coéquipiers présents (cf. internal/analysis/skill_v2/squad.go pour le modèle),
// on décale le μ EFFECTIF avant l'update EP, puis on retire l'offset du
// posterior. Seul le delta de l'EP s'applique au μ individuel ; σ inchangé.

import (
	"context"
	"log/slog"
	"os"
	"strings"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// lusrSquadOffsetEnvFlag contrôle la Phase 2 (squad offset). **ON par défaut**
// (décision produit 2026-05-28). NB : même actif, la correction n'a d'effet que
// si des offsets ont été estimés par cmd/lusr_v2_squad_estimate (sinon
// LoadSquadOffsets renvoie vide → no-op). Mettre "0"/"false"/"no" pour désactiver.
const lusrSquadOffsetEnvFlag = "LEVELUP_LUSR_V2_SQUAD_OFFSET"

// IsLUSRV2SquadOffsetEnabled retourne true sauf si le flag est explicitement
// désactivé ("0"/"false"/"no"). ON par défaut. Cf. lusrSquadOffsetEnvFlag.
func IsLUSRV2SquadOffsetEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(lusrSquadOffsetEnvFlag)))
	return v != "0" && v != "false" && v != "no"
}

// computeTeamSquadOffsets retourne, pour chaque joueur de `states`, la somme des
// offsets de synergie avec ses coéquipiers présents dans CE match (bornée à
// ±SquadOffsetCap). Joueurs sans offset enregistré (non trackés) → 0.
// repo nil (flag off) → slice de zéros.
func computeTeamSquadOffsets(ctx context.Context, repo port.SquadOffsetRepository,
	states []domain.SkillV2State, group string) []float64 {
	offsets := make([]float64, len(states))
	if repo == nil {
		return offsets
	}
	for i := range states {
		partnerOffsets, err := repo.LoadSquadOffsets(ctx, states[i].XUID, group)
		if err != nil {
			slog.WarnContext(ctx, "LUSR v2 squad: LoadSquadOffsets échoué — offset 0",
				"xuid", states[i].XUID, "group", group, "err", err)
			continue
		}
		if len(partnerOffsets) == 0 {
			continue
		}
		var total float64
		for j := range states {
			if i != j {
				total += partnerOffsets[states[j].XUID]
			}
		}
		off := skillv2.ClampSquadOffset(total)
		offsets[i] = off
		if off != 0 {
			slog.DebugContext(ctx, "LUSR v2 squad offset appliqué",
				"xuid", states[i].XUID, "group", group, "offset", off,
				"partners_present", len(states)-1)
		}
	}
	return offsets
}

// applyOffsetsToGaussians retourne les gaussiennes EFFECTIVES (μ + offset, σ
// inchangé) — l'entrée du modèle quand la correction squad est active.
func applyOffsetsToGaussians(g []skillv2.Gaussian, offsets []float64) []skillv2.Gaussian {
	out := make([]skillv2.Gaussian, len(g))
	for i := range g {
		out[i] = skillv2.ApplySquadOffset(g[i], offsets[i])
	}
	return out
}

// stripOffsetsFromGaussians retire l'offset du posterior effectif pour récupérer
// le μ INDIVIDUEL (l'offset était constant pour ce match ; seul le delta EP doit
// s'appliquer au skill individuel). σ inchangé.
func stripOffsetsFromGaussians(g []skillv2.Gaussian, offsets []float64) []skillv2.Gaussian {
	out := make([]skillv2.Gaussian, len(g))
	for i := range g {
		out[i] = skillv2.Gaussian{Mu: g[i].Mu - offsets[i], Sigma: g[i].Sigma}
	}
	return out
}
