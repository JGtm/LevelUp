package narrative

// objective_roles.go — LA CLASSIFICATION PAR RÔLE (prendre / défendre / tenir) des
// colonnes de `match_objective_stats`, pour l'agrégat d'usage de la page Sessions
// (lot S2 du chantier session-usage, HANDOFF_SESSION_USAGE_BDD_2026-09-04 §5).
//
// # POURQUOI ICI, ET PAS UNE LISTE PARALLÈLE
//
// Les noms de colonnes de la table vivent en SOURCE UNIQUE dans ce package
// (objective_participation.go : constantes non exportées, consommées par les tables
// de poids et de durées d'où la couche repo GÉNÈRE ses SUM). Une classification par
// rôle écrite ailleurs recopierait ces noms en chaînes libres — la 3e copie interdite.
// Les rôles se définissent donc SUR le même vocabulaire, dans le même package, et le
// garde-rail TestObjectiveRoles_PartitionDesColonnesDAction verrouille la cohérence :
// prendre ∪ défendre ∪ écartées == exactement les clés des tables de poids, sans
// recouvrement ; tenir == exactement l'union des colonnes de durée par famille.
//
// # LA RÈGLE DE CLASSEMENT
//
//   - PRENDRE : les actions qui S'EMPARENT de l'objectif ou font marquer son camp —
//     captures, vols, prises de zone, ramassages de crâne, dépôts/vols de graines,
//     conversions/extractions réussies, VIP adverse abattu (l'action qui score) et
//     ses assists. `flag_returners_killed` est un soutien d'ATTAQUE : on abat ceux
//     qui ramènent le drapeau qu'on vient de faire tomber.
//   - DÉFENDRE : les actions qui PROTÈGENT l'objectif de son camp — retours et
//     sécurisations de drapeau, porteurs adverses abattus (drapeau, crâne, graine),
//     sécurisations et frags défensifs de zone, conversions adverses refusées,
//     frags réalisés EN ÉTANT le VIP (défendre l'objectif que l'on incarne).
//   - TENIR : les DURÉES (secondes) passées sur/avec l'objectif — exactement
//     ObjectiveFamilyHoldColumns, réutilisée telle quelle (aucune liste nouvelle).
//
// # CE QUI N'ENTRE DANS AUCUN RÔLE, ET POURQUOI
//
//   - `zone_scoring_ticks`, `skull_scoring_ticks` : proportionnels au temps de
//     tenue mais exprimés en TICKS — les additionner aux secondes de « tenir »
//     mélangerait deux unités dans une même somme.
//
// Les colonnes déjà écartées des tables de poids (flag_grabs, kills_as_flag_carrier,
// longest_*, times_selected_as_vip...) restent hors rôles par construction : les
// rôles ne classent QUE le vocabulaire retenu par les tables de poids.

// ObjectiveRole identifie un rôle d'objectif de l'agrégat de session.
type ObjectiveRole string

// Les trois rôles publiés par l'agrégat de session (clés stables, traduites côté web).
const (
	ObjectiveRoleTake   ObjectiveRole = "take"
	ObjectiveRoleDefend ObjectiveRole = "defend"
	ObjectiveRoleHold   ObjectiveRole = "hold"
)

// objectiveRoleTakeColumns / objectiveRoleDefendColumns : la partition des colonnes
// d'action (⊆ clés de ObjectiveFamilyActionWeights — garde-rail dédié).
var objectiveRoleTakeColumns = []string{
	objectiveColFlagCaptures,
	objectiveColFlagCaptureAssists,
	objectiveColFlagSteals,
	objectiveColFlagReturnersKilled,
	objectiveColZoneCaptures,
	objectiveColZoneOffensiveKills,
	objectiveColSkullGrabs,
	objectiveColPowerSeedsDeposited,
	objectiveColPowerSeedsStolen,
	objectiveColExtractionInitiationsCompleted,
	objectiveColExtractionConversionsCompleted,
	objectiveColSuccessfulExtractions,
	objectiveColVipKills,
	objectiveColVipAssists,
}

var objectiveRoleDefendColumns = []string{
	objectiveColFlagReturns,
	objectiveColFlagSecures,
	objectiveColFlagCarriersKilled,
	objectiveColZoneSecures,
	objectiveColZoneDefensiveKills,
	objectiveColSkullCarriersKilled,
	objectiveColPowerSeedCarriersKilled,
	objectiveColExtractionConversionsDenied,
	ObjectiveColKillsAsVIP,
}

// objectiveRoleExcludedActionColumns : colonnes d'action volontairement HORS rôle
// (unité en ticks — cf. l'en-tête). Le garde-rail exige que la partition
// prendre + défendre + écartées couvre EXACTEMENT les clés des tables de poids.
var objectiveRoleExcludedActionColumns = []string{
	objectiveColZoneScoringTicks,
	objectiveColSkullScoringTicks,
}

// GrandeurFlagGrabsNet — LES PRISES NETTES DE DRAPEAU, première grandeur d'un
// rôle qui ne soit PAS une colonne de `match_objective_stats`.
//
// # POURQUOI ELLE EXISTE, ET POURQUOI `flag_grabs` N'Y ENTRE PAS
//
// Le compteur officiel `flag_grabs` est écarté des tables de poids depuis
// l'origine, et à juste titre : il compte CHAQUE ramassage, donc aussi le
// JONGLAGE (lancer le drapeau devant soi pour courir plus vite, puis le
// reprendre). Le faire entrer dans « prendre » aurait récompensé le jonglage —
// mesure du 2026-09-13 : 40 % du compteur en est, et sur un film le premier au
// brut tombe TROISIÈME au net.
//
// La grandeur NETTE, elle, dit bien ce que le rôle « prendre » veut dire. Elle
// entre donc dans le rôle, et le compteur brut reste dehors.
//
// # POURQUOI ELLE N'EST PAS UNE COLONNE DE PLUS DANS LES TABLES CI-DESSUS
//
// Elle ne vient pas de la même table : elle est lue du FILM et vit dans
// `match_flag_grabs_net` (append-only, vue `_latest`), pas dans
// `match_objective_stats` que l'API alimente. Trois choses le rendent NON
// négociable :
//
//   - `ObjectiveRoleColumns` est la source d'où la couche repo GÉNÈRE ses SUM
//     sur `match_objective_stats_latest` — y glisser ce nom produirait un
//     `SUM(flag_grabs_net)` sur une table qui n'a pas la colonne ;
//   - `ObjectiveFamilyActionWeights` pilote l'index de participation, dont le
//     garde-rail exige que chaque clé soit une colonne réelle du schéma ;
//   - la partition prendre ∪ défendre ∪ écartées == clés des tables de poids
//     resterait fausse.
//
// D'où la séparation : les COLONNES d'un côté, les GRANDEURS AUTRES de l'autre,
// et [ObjectiveRoleGrandeurs] qui rend l'union pour les lecteurs qui affichent.
const GrandeurFlagGrabsNet = "flag_grabs_net"

// objectiveRoleExtraGrandeurs : par rôle, les grandeurs qui ne sont pas des
// colonnes de `match_objective_stats`. Une seule à ce jour.
var objectiveRoleExtraGrandeurs = map[ObjectiveRole][]string{
	ObjectiveRoleTake: {GrandeurFlagGrabsNet},
}

// objectiveExtraGrandeurFamily : la famille de mode à laquelle chaque grandeur
// hors colonne appartient. Sans elle, une grandeur de CTF apparaîtrait sur la
// grille d'un mode à zones.
var objectiveExtraGrandeurFamily = map[string]ObjectiveFamily{
	GrandeurFlagGrabsNet: FamilyCTF,
}

// ObjectiveRoleExtraGrandeurs retourne les grandeurs HORS COLONNE d'un rôle,
// dans un ordre déterministe (copie). Vide pour les rôles qui n'en ont pas.
//
// ⚠ NE JAMAIS les passer à un générateur de SQL sur `match_objective_stats` :
// ces noms ne sont pas des colonnes de cette table (cf. [GrandeurFlagGrabsNet]).
func ObjectiveRoleExtraGrandeurs(role ObjectiveRole) []string {
	return append([]string(nil), objectiveRoleExtraGrandeurs[role]...)
}

// ObjectiveExtraGrandeurFamily retourne la famille d'une grandeur hors colonne,
// et false si le nom n'en est pas une.
func ObjectiveExtraGrandeurFamily(grandeur string) (ObjectiveFamily, bool) {
	fam, ok := objectiveExtraGrandeurFamily[grandeur]
	return fam, ok
}

// ObjectiveRoleGrandeurs retourne TOUTES les grandeurs d'un rôle : ses colonnes
// de `match_objective_stats` d'abord, puis les grandeurs mesurées ailleurs.
//
// C'est la lecture des AFFICHEURS (grilles d'objectif, cartes de rôle). Les
// producteurs de SQL, eux, appellent [ObjectiveRoleColumns] — et l'écart entre
// les deux fonctions EST la frontière entre « ce que la table porte » et « ce
// que le rôle veut dire ».
func ObjectiveRoleGrandeurs(role ObjectiveRole) []string {
	return append(ObjectiveRoleColumns(role), objectiveRoleExtraGrandeurs[role]...)
}

// ObjectiveRoleColumns retourne les colonnes de `match_objective_stats` du rôle,
// dans un ordre déterministe (copie — l'appelant peut la garder). Le rôle « tenir »
// est dérivé d'ObjectiveFamilyHoldColumns (source unique des durées), jamais d'une
// liste propre.
func ObjectiveRoleColumns(role ObjectiveRole) []string {
	switch role {
	case ObjectiveRoleTake:
		return append([]string(nil), objectiveRoleTakeColumns...)
	case ObjectiveRoleDefend:
		return append([]string(nil), objectiveRoleDefendColumns...)
	case ObjectiveRoleHold:
		return objectiveHoldColumnsUnion()
	default:
		return nil
	}
}

// AllObjectiveRoles retourne les trois rôles dans l'ordre de publication.
func AllObjectiveRoles() []ObjectiveRole {
	return []ObjectiveRole{ObjectiveRoleTake, ObjectiveRoleDefend, ObjectiveRoleHold}
}

// objectiveHoldColumnsUnion : l'union DÉDOUBLONNÉE des colonnes de durée par famille,
// dans l'ordre stable des familles (time_in_zones_seconds est partagée par KOTH et
// Strongholds — une seule occurrence).
func objectiveHoldColumnsUnion() []string {
	seen := map[string]bool{}
	var out []string
	for _, fam := range AllObjectiveFamilies() {
		for _, c := range ObjectiveFamilyHoldColumns[fam] {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}
