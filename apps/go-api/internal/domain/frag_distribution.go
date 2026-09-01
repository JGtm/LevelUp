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
//   - Classes equipment/environmental + leur niveau 2 : SOURCE DE DÉGÂT du film
//     (KillSourceClassRow, Authoritative=false). TROISIÈME provenance, ajoutée le
//     2026-08-29. Elle existe parce que l'attribution arme-à-feu ne peut PAS voir ces
//     kills : elle part des records de dégât `0xd2` du tireur, et un répulseur, une
//     bobine ou une chute n'en émettent aucun. Ces kills tombaient donc dans
//     `unattributed` — cette provenance ne fait que DÉCOUPER ce résidu, elle ne retire
//     rien à aucune autre classe. La garantie anti-double-comptage est structurelle et
//     vit en amont (platform/duckdb : seules les clés SANS id numérique remontent).
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
	// FragClassEquipment : tué par un ÉQUIPEMENT (Halo Infinite : le répulseur). Servie
	// par la SOURCE DE DÉGÂT du film, pas par le registre d'armes ni par un compteur API
	// — cf. la 3ᵉ provenance en tête de fichier.
	FragClassEquipment = "equipment" // Équipement
)

// nonCombatFragClasses regroupe les classes qui ne correspondent à AUCUN outil de
// destruction identifiable : équipement, environnement (chute, explosifs de map),
// « autre » (UGC) et le résidu non attribué. Elles sont exclues du breakdown par-ARME.
// Source unique côté Go, gardée par frag_distribution_test.go (TestIsNonCombatFragClass).
//
// CE N'EST PAS le miroir de NON_COMBAT_WEAPON_ROLES
// (apps/web/src/features/synthesis/weaponRoleInsight.ts), contrairement à ce que disait
// ce commentaire : le set web contient EN PLUS véhicule et tourelle. La divergence est
// voulue et durable — les deux ensembles répondent à deux questions différentes :
//
//   - ICI : « cette classe a-t-elle un outil nommable dans un tableau d'armes ? » —
//     un Warthog ou une tourelle Gauss, OUI (V73-3.2, ils ont un breakdown par engin) ;
//   - CÔTÉ WEB : « cette classe dit-elle quelque chose du STYLE DE JEU du joueur ? » —
//     un frag au Warthog, NON : le compter au dénominateur de l'insight coach
//     (« tu ne prends jamais d'arme lourde ») fausserait le verdict.
//
// Le miroir EXACT de ce set côté web est NON_WEAPON_FRAG_CLASSES
// (apps/web/src/components/charts/fragDetailBreakdown.ts), qui filtre le « Détails des
// frags » ; NON_COMBAT_WEAPON_ROLES en DÉRIVE en y ajoutant les deux classes d'engin.
// Aucun garde-rail ne teste ce miroir : il n'y a pas de contrat de sérialisation entre
// les deux, seulement une intention commune, re-vérifiée à la main.
//
// V73-3.2 : véhicule et tourelle en sont SORTIS. Ce sont des outils de destruction
// réels, identifiés PAR ENGIN dans le registre d'armes (weapon_key h5_vehicle_warthog,
// h5_turret_gauss…) — ils portent donc un breakdown légitime, au lieu de disparaître
// dans « Non attribué ». Rien n'est codé par titre : un titre dont le registre ne
// déclare aucune arme de classe vehicle/turret ne produit simplement aucune row de
// cette classe (cas Halo Infinite au 2026-08-02, en attente du killsource).
// FragClassEquipment y figure (2026-08-29) : aucune ligne `weapon_kills` ne porte cette
// classe — ses entrées de registre n'ont PAS d'id numérique — donc l'y mettre ne retire
// rien à personne, et ça garde vraies les deux propriétés que ce set sert vraiment : pas
// de « précision » pour un répulseur (WeaponClassHasAccuracy), et pas d'entrée dans le
// breakdown par-arme du match, qui lit weapon_kills. Sa présence au SUNBURST ne passe pas
// par ce set : elle vient de la 3ᵉ provenance (cf. en-tête), qui a son propre chemin.
var nonCombatFragClasses = map[string]bool{
	FragClassUnattributed:  true,
	FragClassEnvironmental: true,
	FragClassOther:         true,
	FragClassEquipment:     true,
}

// IsNonCombatFragClass indique si une classe d'arme est un bucket non-combat
// (environnement / autre / non attribué), à exclure d'un breakdown par-arme réservé
// aux outils de destruction identifiables.
func IsNonCombatFragClass(class string) bool {
	return nonCombatFragClasses[class]
}

// perWeaponFragClasses regroupe les classes dont le niveau 2 du sunburst est ventilé par
// OBJET (weapon_key du registre) et non par rôle de combat : sur ces classes, `role` et
// `family` valent la classe elle-même dans le registre (tous les véhicules portent
// class=role=family="vehicle"), un niveau 2 par rôle serait donc un arc unique sans
// information. La clé de niveau 2 est le weapon_key ; le libellé vient de
// config/titles/{slug}/mappings/weapon_names.toml via metadata.weapon_name_labels
// (jamais de nom d'objet en dur côté Go).
//
// PORTÉE : ce set décrit la FORME du niveau 2, pas la PROVENANCE des frags. Depuis la
// bascule de l'arme du kill (2026-09-01), `equipment` et `environmental` s'y trouvent
// aussi — leur niveau 2 est un objet (bobine à plasma, répulseur) exactement comme celui
// d'un engin. Savoir SI une classe est servie est une autre question, tranchée dans
// fragdist.isRegistryFragClass par la provenance de la ligne : une ligne mesurée par la
// source de dégât du film sert ces classes, une ligne issue de `weapon_kills` non — sans
// quoi le bucket `h5_environmental` de Halo 5, qui porte un identifiant numérique, ferait
// apparaître une classe Environnement dans le sunburst du second titre.
var perWeaponFragClasses = map[string]bool{
	FragClassVehicle:       true,
	FragClassTurret:        true,
	FragClassEquipment:     true,
	FragClassEnvironmental: true,
}

// IsPerWeaponFragClass indique si le niveau 2 d'une classe se ventile par OBJET
// (weapon_key) plutôt que par rôle de combat. Cf. perWeaponFragClasses (et sa note de
// portée : la forme du niveau 2 ne dit pas la provenance des frags).
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
// de grenade, ou OBJET pour les classes ventilées par weapon_key : véhicule, tourelle,
// équipement, environnement).
type FragRoleEntry struct {
	Role  string `json:"role"` // precision|automatic|sniper|shotgun|special|sidearm|assassination|direct_melee|ground_pound|shoulder_bash|grenade_*| weapon_key d'objet (h5_vehicle_warthog, hinf_repulsor, hinf_coil_plasma…)
	Kills int    `json:"kills"`
	// Label : libellé d'affichage DÉJÀ résolu, servi quand Role est un weapon_key
	// d'OBJET — sa source est config/titles/{slug}/mappings/weapon_names.toml (via
	// metadata.weapon_name_labels), et non un manifeste i18n web. Motif : ces noms sont
	// propres à CHAQUE titre ; les recopier dans le manifeste web frags.toml (partagé par
	// tous les titres) dupliquerait le TOML du titre et enflerait à chaque titre ajouté.
	//
	// DEUX familles de classes le peuplent, par deux provenances distinctes :
	//   - véhicule/tourelle (IsPerWeaponFragClass), via le REGISTRE d'armes ;
	//   - équipement/environnement (2026-08-29), via la SOURCE DE DÉGÂT du film.
	// Les deux passent par fragdist.perWeaponRoles, d'où le même contrat de libellé.
	// Ce champ n'est donc PAS réservé aux classes IsPerWeaponFragClass (il l'était avant
	// le lot « kills hors arme à feu »).
	//
	// FR-first (repli EN interne côté résolveur, cf. weapon_resolver.go) : sémantique
	// INCHANGÉE depuis son introduction — un lecteur FR voit toujours ce champ. Vide pour
	// tous les autres rôles, qui restent des clés canoniques traduites côté web
	// (frags.role.*) — le front applique `label || t(frags.role.<role>)`.
	Label string `json:"label,omitempty"`
	// LabelEN : même libellé d'objet que Label, mais EN-first (repli FR interne) — ajouté
	// le 2026-08-29 (V2.1, D2) pour qu'un lecteur EN voie le nom EN de l'objet (« UNSC
	// Fusion Coil ») au lieu du FR servi par défaut (« Bobine à fusion UNSC »). Peuplé par
	// EXACTEMENT les deux mêmes provenances que Label (registre véhicule/tourelle +
	// source de dégât du film), au même point d'écriture — jamais l'un sans l'autre.
	// Vide dans les mêmes conditions que Label (aucune traduction seedée) : le web choisit
	// entre les deux SELON LA LOCALE et retombe sur un libellé générique si les deux sont
	// vides (cf. fragRoleLabel.ts, jamais une clé i18n brute à l'écran).
	LabelEN string `json:"label_en,omitempty"`
}
