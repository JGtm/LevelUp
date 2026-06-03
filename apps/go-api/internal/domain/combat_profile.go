// Package domain — combat_profile.go : types du profil combat 3 axes.
//
// Ref : .ai/PLAN_COMBAT_PROFILE_WIRING.md
// Le profil combat est exposé dans Synthesis, Squad (par joueur) et Session Compare.
package domain

// CombatStyle est le descripteur textuel d'un axe du profil combat.
// Valeurs : voir constantes ci-dessous (trois axes × trois niveaux).
type CombatStyle string

const (
	// Axe offensif (avg_oc vs OffensiveConversionP80).
	CombatStyleOffensivePrecis    CombatStyle = "precis"    // finisseur : converti bien ses dégâts en kills
	CombatStyleOffensiveEquilibre CombatStyle = "equilibre" // profil mixte
	CombatStyleOffensiveGenereux  CombatStyle = "genereux"  // setup : dégâts qui servent les coéquipiers

	// Axe défensif (avg_dr vs DefensiveResistanceP80).
	CombatStyleDefensiveResistant CombatStyle = "resistant" // encaisse beaucoup avant de tomber
	CombatStyleDefensiveSolide    CombatStyle = "solide"    // profil moyen
	CombatStyleDefensiveFragile   CombatStyle = "fragile"   // meurt vite

	// Axe activité (avg_residual_brut vs lobby).
	// Nil tant que engagement_score_brut n'est pas exposé dans canonical (Phase 4).
	CombatStyleActivityActif   CombatStyle = "actif"   // au-dessus du rythme du lobby
	CombatStyleActivityModere  CombatStyle = "modere"  // dans la moyenne du lobby
	CombatStyleActivityDiscret CombatStyle = "discret" // en retrait
)

// CombatProfileBlock agrège OC, DR, ResidualBrut et les descripteurs de style.
// Renvoyé dans SynthesisPageV2Response et KPIStats (par joueur dans Squad).
type CombatProfileBlock struct {
	AvgOC      float64 `json:"avg_oc"`
	AvgDR      float64 `json:"avg_dr"`
	MatchCount int     `json:"match_count"`

	// AvgResidualBrut est nil si engagement_score_brut n'est pas disponible
	// (Phase 4 non encore livrée — canonical.PlayerMatchEnrichment ne l'expose pas).
	AvgResidualBrut *float64 `json:"avg_residual_brut,omitempty"`

	// DmgPerKill / DmgPerDeath : dégâts moyens par frag / par mort, agrégés
	// (Σ damage_dealt / Σ kills, Σ damage_taken / Σ deaths) sur la fenêtre. Nil
	// si dénominateur nul. Affichés à côté du rendement/résistance (parité KPI
	// card Explorer YieldTile).
	DmgPerKill  *float64 `json:"dmg_per_kill,omitempty"`
	DmgPerDeath *float64 `json:"dmg_per_death,omitempty"`

	// Descripteurs textuels — nil si MatchCount < minMatchesForCombatStyle (15).
	StyleOffensive *CombatStyle `json:"style_offensive,omitempty"`
	StyleDefensive *CombatStyle `json:"style_defensive,omitempty"`
	// StyleActivity nil si AvgResidualBrut nil.
	StyleActivity *CombatStyle `json:"style_activity,omitempty"`
}
