package skill_v2

// mode_correlation.go : Phase 4 — corrélation cross-mode (TS2 §11 lite).
//
// Modèle simplifié vs full TS2 §11 :
//   - Pas de variable globale `s_p^{global}` partagée
//   - Pas de factor graph étendu
//   - À la place : leak post-update — une fraction `w_d` (cap 0.3-0.4) du
//     déplacement de μ dans le mode m est répercutée sur les μ des AUTRES
//     modes que le joueur a déjà joués.
//
// Avantages : pas de re-architecture du factor graph, contrôle direct du
// degré de "fuite" via la constante. Inconvénients : ne capture pas les
// corrélations négatives entre modes (un fort en slayer pourrait être faible
// en BTB), et c'est asymétrique (un seul update propage, pas un cluster).
//
// **Cap requis par décision produit** : w_d ≤ 0.4 strict. La motivation est
// d'éviter qu'un joueur "chaos" (modes high-deaths fun-mode) voie son skill
// slayer / objectif inflated par ses kills chaos. Cf. .ai/LUSR_V2_HANDOFF.md.

// Phase4ModeCouplingMaxWeight est la borne supérieure absolue du w_d. Le code
// CLAMP toute valeur au-dessus à 0.4 pour respecter la contrainte produit.
const Phase4ModeCouplingMaxWeight = 0.4

// DefaultModeCouplingWeight est la valeur conservative par défaut. La
// re-estimation Phase 5 (TTT batch) pourra l'ajuster dans [0, 0.4].
const DefaultModeCouplingWeight = 0.3

// ApplyCrossModeLeak retourne le nouveau μ d'un mode "secondaire" après leak.
//
//	new_mu_other = old_mu_other + w_d · (new_mu_primary - old_mu_primary)
//
// w_d est clampé à [0, Phase4ModeCouplingMaxWeight].
//
// σ n'est PAS modifié — on déplace l'ancre, on ne change pas la confiance.
// Justification : ce leak n'ajoute pas de nouvelle observation directe sur le
// skill dans l'autre mode, juste une expectation de corrélation.
func ApplyCrossModeLeak(oldMuOther, oldMuPrimary, newMuPrimary, weight float64) float64 {
	w := weight
	if w < 0 {
		w = 0
	}
	if w > Phase4ModeCouplingMaxWeight {
		w = Phase4ModeCouplingMaxWeight
	}
	delta := newMuPrimary - oldMuPrimary
	return oldMuOther + w*delta
}
