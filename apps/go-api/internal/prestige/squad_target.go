package prestige

// squad_target.go — gestion de la cible dynamique d'un défi d'escouade collectif.
//
// Référence : Axe 4 du plan conceptuel (Baseline en mode collectif).
//
// Logique :
//   - La cible est stockée comme valeur PAR MEMBRE (target_per_member)
//   - Le total affiché = target_per_member × nombre de participants actifs
//   - Quand un membre rejoint/part, le total est recalculé
//   - Verrou : si la cible totale baisse sous la progression déjà acquise,
//     le défi ne peut pas être validé rétroactivement (vérif côté evaluator)

import "errors"

// ErrTargetWouldGoBelowProgress est retournée si retirer un membre ferait
// passer la cible totale sous la progression déjà accumulée.
var ErrTargetWouldGoBelowProgress = errors.New("prestige: cible passerait sous la progression actuelle")

// CollectiveTargetTotal calcule la cible totale d'un défi collectif.
//
//	target_total = target_per_member × nb_participants_actifs
//
// Précondition : nbActiveMembers ≥ 0. Si == 0, retourne 0.
func CollectiveTargetTotal(targetPerMember float64, nbActiveMembers int) float64 {
	if nbActiveMembers <= 0 {
		return 0
	}
	return targetPerMember * float64(nbActiveMembers)
}

// CollectiveBaseline calcule la baseline collective d'un défi en sommant
// les baselines individuelles des participants.
//
// Renvoie la somme + le nombre de participants ayant des données suffisantes
// (DataFull ou DataEstimated). Les participants en DataTracking sont ignorés
// pour le calcul anti-smurf — ils participent mais ne pèsent pas dans la baseline.
func CollectiveBaseline(baselines []Baseline) (total float64, contributingMembers int) {
	for _, b := range baselines {
		if b.DataTier == DataTracking {
			continue
		}
		total += b.Value
		contributingMembers++
	}
	return total, contributingMembers
}

// ValidateResizeForRemoval vérifie qu'un retrait de membre ne casse pas
// l'invariant "cible totale ≥ progression acquise".
//
// Retourne ErrTargetWouldGoBelowProgress si le retrait abaisserait la cible
// sous la progression actuelle (cas où la collective serait validée par
// erreur après retrait).
func ValidateResizeForRemoval(targetPerMember float64, currentActiveMembers int, currentProgress float64) error {
	newTotal := CollectiveTargetTotal(targetPerMember, currentActiveMembers-1)
	if currentProgress >= newTotal {
		return ErrTargetWouldGoBelowProgress
	}
	return nil
}
