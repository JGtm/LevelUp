// Package domain — combat_profile.go : types du profil combat 3 axes.
//
// Ref : .ai/PLAN_COMBAT_PROFILE_WIRING.md
// Le profil combat est exposé dans Synthesis, Squad (par joueur) et Session Compare.
package domain

// CombatStyle est le descripteur textuel d'un axe du profil combat.
// Valeurs : voir constantes ci-dessous (trois axes × trois niveaux).
type CombatStyle string

const (
	// Axe offensif (avg_oc) — conversion dégâts→kill, du plus dispersé au plus létal.
	// Bornes : <0.78 / 0.78 / 0.81 / 0.85 / >0.90 (cf. analysis/combat_yield.go).
	CombatStyleOffensiveDisperse    CombatStyle = "disperse"    // dégâts dispersés, convertit peu
	CombatStyleOffensiveIrregulier  CombatStyle = "irregulier"  // finish irrégulier
	CombatStyleOffensiveEquilibre   CombatStyle = "equilibre"   // profil mixte
	CombatStyleOffensivePrecis      CombatStyle = "precis"      // finisseur : convertit bien
	CombatStyleOffensiveChirurgical CombatStyle = "chirurgical" // létal, gaspille très peu

	// Axe défensif (avg_dr) — survie, du plus fragile au plus encaissant.
	// Bornes : <1.20 / 1.20 / 1.35 / 1.50 / >1.65.
	CombatStyleDefensiveFragile      CombatStyle = "fragile"      // meurt vite
	CombatStyleDefensiveExpose       CombatStyle = "expose"       // encaisse peu avant de tomber
	CombatStyleDefensiveSolide       CombatStyle = "solide"       // profil moyen
	CombatStyleDefensiveResistant    CombatStyle = "resistant"    // encaisse beaucoup
	CombatStyleDefensiveInebranlable CombatStyle = "inebranlable" // survie de très haut niveau

	// Axe activité (avg_pace_ratio = pace_joueur/pace_lobby) — engagement absolu.
	// Bornes : <0.80 / 0.80 / 0.92 / 1.08 / >1.25. Nil si paces indisponibles.
	CombatStyleActivityPassif   CombatStyle = "passif"   // nettement sous le rythme du lobby
	CombatStyleActivityDiscret  CombatStyle = "discret"  // en retrait
	CombatStyleActivityMesure   CombatStyle = "mesure"   // au rythme du lobby
	CombatStyleActivityActif    CombatStyle = "actif"    // au-dessus du rythme du lobby
	CombatStyleActivityAgressif CombatStyle = "agressif" // nettement au-dessus
)

// CombatProfileBlock agrège OC, DR, PaceRatio et les descripteurs de style.
// Renvoyé dans SynthesisPageV2Response et KPIStats (par joueur dans Squad).
type CombatProfileBlock struct {
	AvgOC      float64 `json:"avg_oc"`
	AvgDR      float64 `json:"avg_dr"`
	MatchCount int     `json:"match_count"`

	// AvgPaceRatio = engagement absolu (pace_joueur / pace_lobby moyen ; 1.0 = au
	// rythme du lobby). Nil si les paces d'engagement ne sont pas disponibles.
	AvgPaceRatio *float64 `json:"avg_pace_ratio,omitempty"`

	// DmgPerKill / DmgPerDeath : dégâts moyens par frag / par mort, agrégés
	// (Σ damage_dealt / Σ kills, Σ damage_taken / Σ deaths) sur la fenêtre. Nil
	// si dénominateur nul. Affichés à côté du rendement/résistance (parité KPI
	// card Explorer YieldTile).
	DmgPerKill  *float64 `json:"dmg_per_kill,omitempty"`
	DmgPerDeath *float64 `json:"dmg_per_death,omitempty"`

	// Descripteurs textuels — nil si MatchCount < minMatchesForCombatStyle (15).
	StyleOffensive *CombatStyle `json:"style_offensive,omitempty"`
	StyleDefensive *CombatStyle `json:"style_defensive,omitempty"`
	// StyleActivity nil si AvgPaceRatio nil.
	StyleActivity *CombatStyle `json:"style_activity,omitempty"`
}
