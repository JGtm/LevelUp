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
	// Buckets NON-COMBAT (pas d'arme réelle) — présents seulement sur H5 (registre).
	FragClassVehicle       = "vehicle"       // Véhicule
	FragClassTurret        = "turret"        // Tourelle
	FragClassEnvironmental = "environmental" // Environnement
	FragClassOther         = "other"         // Autre
)

// nonCombatFragClasses regroupe les classes qui ne correspondent à AUCUN outil de
// destruction identifiable : environnement (chute, explosifs de map), « autre » (UGC)
// et le résidu non attribué. Elles sont exclues du breakdown par-ARME (le sunburst les
// absorbe dans « Non attribué »). Miroir Go canonique de NON_COMBAT_WEAPON_ROLES
// (apps/web/src/features/synthesis/weaponRoleInsight.ts) — source unique côté Go,
// gardée par frag_distribution_test.go (TestIsNonCombatFragClass).
//
// V73-3.2 : véhicule et tourelle en sont SORTIS. Ce sont des outils de destruction
// réels, identifiés PAR ENGIN dans le registre d'armes (weapon_key h5_vehicle_warthog,
// h5_turret_gauss…) — ils portent donc un breakdown légitime, au lieu de disparaître
// dans « Non attribué ». Rien n'est codé par titre : un titre dont le registre ne
// déclare aucune arme de classe vehicle/turret ne produit simplement aucune row de
// cette classe (cas Halo Infinite au 2026-08-02, en attente du killsource).
var nonCombatFragClasses = map[string]bool{
	FragClassUnattributed:  true,
	FragClassEnvironmental: true,
	FragClassOther:         true,
}

// IsNonCombatFragClass indique si une classe d'arme est un bucket non-combat
// (environnement / autre / non attribué), à exclure d'un breakdown par-arme réservé
// aux outils de destruction identifiables.
func IsNonCombatFragClass(class string) bool {
	return nonCombatFragClasses[class]
}

// perWeaponFragClasses regroupe les classes dont le niveau 2 du sunburst est ventilé
// par ENGIN (weapon_key du registre) et non par rôle de combat : sur ces classes,
// `role` et `family` valent la classe elle-même dans le registre (tous les véhicules
// portent class=role=family="vehicle"), un niveau 2 par rôle serait donc un arc unique
// sans information. La clé de niveau 2 est le weapon_key ; le libellé vient de
// config/titles/{slug}/mappings/weapon_names.toml via metadata.weapon_name_labels
// (jamais de nom d'engin en dur côté Go).
var perWeaponFragClasses = map[string]bool{
	FragClassVehicle: true,
	FragClassTurret:  true,
}

// IsPerWeaponFragClass indique si le niveau 2 d'une classe se ventile par ENGIN
// (weapon_key) plutôt que par rôle de combat. Cf. perWeaponFragClasses.
func IsPerWeaponFragClass(class string) bool {
	return perWeaponFragClasses[class]
}

// WeaponClassHasAccuracy indique si une classe d'arme a une précision PERTINENTE dans un
// graphe « Précision par arme ». Faux pour les classes SANS « tir au but » — projectiles
// (grenade), mêlée, capacités spartanes, résidu non attribué — les armes montées d'engin
// (véhicule/tourelle) et les buckets non-combat (environnement/autre via
// IsNonCombatFragClass). Vrai pour les armes à tir (classes gun) et les classes non
// résolues ("" — bénéfice du doute : arme hors registre). Sans ce filtre, une grenade
// lancée (shots_fired > 0, jamais « au but ») apparaît à 0 % dans le graphe alors qu'elle
// est absente du sunburst (bug observé Sessions/Synthesis).
//
// V73-3.2 : véhicule/tourelle sont désormais listés ICI explicitement. Ils ont quitté
// nonCombatFragClasses (ils ont gagné un breakdown par engin dans le sunburst) mais leur
// exclusion du graphe de précision est INCHANGÉE — décision produit distincte, hors
// périmètre. Note : la donnée existerait (weapon_accuracy H5 porte des shots_fired pour
// ces classes), son exploitation relève d'un arbitrage produit séparé.
//
// Exporté et hébergé côté domain (déplacé depuis service) pour être partagé par le package
// service/teammates, qui ne peut pas importer son parent service. N'utilise que des
// constantes domain → aucun cycle d'import.
func WeaponClassHasAccuracy(class string) bool {
	switch class {
	case FragClassGrenade, FragClassMelee, FragClassSpartanAbility, FragClassUnattributed,
		FragClassVehicle, FragClassTurret:
		return false
	}
	return !IsNonCombatFragClass(class)
}

// Clés de rôle canoniques du niveau 2 propres aux classes API (les rôles d'arme
// registre — precision/automatic/sniper/… — restent portés par le registre).
const (
	FragRoleAssassination = "assassination" // Mêlée niv.2 (H5)
	FragRoleDirectMelee   = "direct_melee"  // Mêlée niv.2 (H5) = total_melee_kills (disjoint des assassinats)
	FragRoleGroundPound   = "ground_pound"  // Capacité spartane niv.2
	FragRoleShoulderBash  = "shoulder_bash" // Capacité spartane niv.2
)

// Clés de rôle du niveau 2 de la classe Grenade — TYPE de grenade (V72-15.2). Dérivées
// de la FAMILLE registre (frag_grenade/plasma_grenade/dynamo_grenade/splinter_grenade) des
// rows class=grenade ; grenade_other regroupe les familles non typées + le résidu de
// réconciliation vers le total API autoritatif (Σ types == kills de la classe).
const (
	FragRoleGrenadeFrag     = "grenade_frag"
	FragRoleGrenadePlasma   = "grenade_plasma"
	FragRoleGrenadeDynamo   = "grenade_dynamo"
	FragRoleGrenadeSplinter = "grenade_splinter"
	FragRoleGrenadeOther    = "grenade_other"
)

// GrenadeTypeRoleOrder fixe l'ordre canonique (déterministe) des types de grenade pour le
// tie-break du niveau 2 (à kills égaux). « Autre grenade » en dernier.
var GrenadeTypeRoleOrder = []string{
	FragRoleGrenadeFrag, FragRoleGrenadePlasma, FragRoleGrenadeDynamo,
	FragRoleGrenadeSplinter, FragRoleGrenadeOther,
}

// GrenadeTypeRoleForFamily mappe une FAMILLE d'arme registre (row class=grenade) vers son
// rôle « type de grenade » canonique. ok=false (→ FragRoleGrenadeOther) pour une famille
// non typée : dégradation propre, jamais de comparaison de slug de titre — toute famille
// grenade connue des deux titres (Infinite : frag/plasma/dynamo ; H5 : frag/plasma/splinter)
// est couverte, une famille inconnue retombe dans « Autre grenade ».
func GrenadeTypeRoleForFamily(family string) (string, bool) {
	switch family {
	case "frag_grenade":
		return FragRoleGrenadeFrag, true
	case "plasma_grenade":
		return FragRoleGrenadePlasma, true
	case "dynamo_grenade":
		return FragRoleGrenadeDynamo, true
	case "splinter_grenade":
		return FragRoleGrenadeSplinter, true
	}
	return FragRoleGrenadeOther, false
}

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

// FragRoleEntry est un arc du niveau 2 (rôle = fonction de combat, sous-mécanique, type
// de grenade, ou ENGIN pour les classes véhicule/tourelle).
type FragRoleEntry struct {
	Role  string `json:"role"` // precision|automatic|sniper|shotgun|special|sidearm|assassination|direct_melee|ground_pound|shoulder_bash|grenade_*| weapon_key d'engin (h5_vehicle_warthog…)
	Kills int    `json:"kills"`
	// Label : libellé d'affichage DÉJÀ résolu, servi UNIQUEMENT quand Role est un
	// weapon_key d'engin (classes IsPerWeaponFragClass) — sa source est
	// config/titles/{slug}/mappings/weapon_names.toml (via metadata.weapon_name_labels),
	// et non un manifeste i18n web. Motif : les noms d'engins sont propres à CHAQUE
	// titre ; les recopier dans le manifeste web frags.toml (partagé par tous les titres)
	// dupliquerait le TOML du titre et enflerait à chaque titre ajouté. Vide pour tous
	// les autres rôles, qui restent des clés canoniques traduites côté web
	// (frags.role.*) — le front applique `label || t(frags.role.<role>)`.
	Label string `json:"label,omitempty"`
}
