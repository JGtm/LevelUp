// Package domain — frag_distribution.go : DTO canonique « Répartition des frags »
// v2 (sunburst hiérarchique classe→rôle). Type title-agnostic partagé par toutes
// les surfaces (Synthesis, Match view, Timeseries, Sessions ; l'Escouade n'utilise
// que le niveau classe). Cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md §2.
//
// Provenance des données (point anti-double-source) :
//   - Classes melee/grenade/spartan_ability + total : stats canoniques API
//     (Authoritative=true).
//   - Classes shoulder/sidearm/heavy + rôles d'arme : registre d'armes
//     (WeaponKillRow, Authoritative=false = estimé).
//   - unattributed : résidu calculé (TotalKills − Σ classes), ajouté si > 0.
//
// Invariants (testés, cf. buildFragDistribution) :
//
//	(a) Σ Classes[i].Kills == TotalKills (unattributed absorbe l'écart) ;
//	(b) pour toute classe non-feuille, Σ Roles[j].Kills == Class.Kills ;
//	(c) unattributed.Kills >= 0 ;
//	(d) capability native_kill_mechanics off ⇒ aucune classe spartan_ability et
//	    Mêlée reste une feuille (byte-équivalent à l'absence de mécaniques natives).
package domain

// Clés de classe canoniques (axe manipulation) — niveau 1 du sunburst.
const (
	FragClassShoulder       = "shoulder"        // Épaule
	FragClassSidearm        = "sidearm"         // Poing
	FragClassHeavy          = "heavy"           // Lourde
	FragClassMelee          = "melee"           // Mêlée (total API)
	FragClassGrenade        = "grenade"         // Grenade (total API)
	FragClassSpartanAbility = "spartan_ability" // Capacités spartanes (H5, cap-gated)
	FragClassUnattributed   = "unattributed"    // Non attribué (résidu calculé)
)

// Clés de rôle canoniques du niveau 2 propres aux classes API (les rôles d'arme
// registre — precision/automatic/sniper/… — restent portés par le registre).
const (
	FragRoleAssassination = "assassination" // Mêlée niv.2 (H5)
	FragRoleDirectMelee   = "direct_melee"  // Mêlée niv.2 (H5) = total_melee_kills (disjoint des assassinats)
	FragRoleGroundPound   = "ground_pound"  // Capacité spartane niv.2
	FragRoleShoulderBash  = "shoulder_bash" // Capacité spartane niv.2
)

// FragKillTypeCounts porte les compteurs de kill-type NATIFS (API canonique) qui
// alimentent le niveau 1 de la FragDistribution : Mêlée, Grenade et — sous capability
// native_kill_mechanics — Capacités spartanes (ground pound + shoulder bash), plus le
// total. Struct NEUTRE (aucune dépendance à un DTO de page) pour que buildFragDistribution
// serve toutes les surfaces : Synthesis (agrégat SynthesisDetailedStats), Match view
// (ligne scoreboard native du viewer), etc. — sans dupliquer la logique (règle ≤2 copies).
type FragKillTypeCounts struct {
	Melee         int // total_melee_kills
	Grenade       int // total_grenade_kills
	Assassination int // total_assassinations (Mêlée niv.2, H5)
	GroundPound   int // total_ground_pound_kills (Capacité spartane, H5)
	ShoulderBash  int // total_shoulder_bash_kills (Capacité spartane, H5)
	Total         int // total_kills (base du résidu unattributed)
}

// FragDistribution est la répartition hiérarchique des frags d'un scope (joueur,
// match, session…). Classes ordonnées de façon déterministe ; Σ Kills == TotalKills.
type FragDistribution struct {
	TotalKills int              `json:"total_kills"`
	Classes    []FragClassEntry `json:"classes"` // ordonné ; Σ Kills == TotalKills (inclut unattributed)
}

// FragClassEntry est un arc du niveau 1 (classe = axe manipulation).
type FragClassEntry struct {
	Class         string          `json:"class"`           // shoulder|sidearm|heavy|melee|grenade|spartan_ability|unattributed
	Kills         int             `json:"kills"`           //
	Authoritative bool            `json:"authoritative"`   // true = totaux API canoniques ; false = estimé registre / résidu
	Roles         []FragRoleEntry `json:"roles,omitempty"` // nil = feuille ; sinon Σ Kills(roles) == Kills
}

// FragRoleEntry est un arc du niveau 2 (rôle = fonction de combat, ou sous-mécanique).
type FragRoleEntry struct {
	Role  string `json:"role"` // precision|automatic|sniper|shotgun|special|sidearm|assassination|direct_melee|ground_pound|shoulder_bash
	Kills int    `json:"kills"`
}
