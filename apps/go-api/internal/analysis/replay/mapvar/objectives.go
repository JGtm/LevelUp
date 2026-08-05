// Package mapvar — objectives.go : résolution des labels de mode de jeu et
// classification des objets d'objectif.
//
// ORIGINE DE LA TABLE (méthode, pas devinette) :
// les labels sont stockés sous forme de hash int32. La fonction de hachage a été
// identifiée par correspondance directe : murmur3_x86_32("stockpile_socket", seed=0)
// == 2110778921, valeur lue telle quelle dans ctf_breaker.mvar sur les 10 objets
// que la table de noms du même fichier appelle explicitement
// "stockpile_blue_socket_01".."stockpile_red_socket_05". La table ci-dessous a
// ensuite été peuplée par recherche exhaustive d'identifiants snake_case dont le
// murmur3 retombe sur un hash observé.
//
// GARDE-FOU : une telle recherche produit des collisions fortuites (mesuré :
// ~5 attendues sur 1,0e8 candidats × 210 cibles). Seules les correspondances
// SÉMANTIQUEMENT cohérentes avec le domaine Halo sont retenues ici ; les
// collisions absurdes ("seed_post_off_of", "weapon_attrition_assault_the") sont
// rejetées. Un hash non listé reste INCONNU — on ne devine pas de libellé.
package mapvar

// labelNames : hash murmur3_x86_32(seed=0) → nom du label de mode de jeu.
//
// Vérification de non-régression : mapvar_test.go recalcule le murmur3 de chaque
// nom et exige qu'il retombe sur sa clé. La table ne peut donc pas dériver.
var labelNames = map[int32]string{
	// --- Capture du drapeau ---
	-2087265038: "ctf_include",
	494243972:   "ctf_exclude",
	2045313975:  "ctf_neutral_include",
	1138535599:  "ctf_multi_exclude",
	1493717729:  "flag_spawn",
	-713178115:  "flag_delivery",
	// --- Stockpile ---
	1755892232:  "stockpile_include",
	1757610942:  "stockpile_exclude",
	2110778921:  "stockpile_socket",
	-1831893171: "stockpile_navpoint",
	// --- Autres modes ---
	1397847647:  "oddball_spawn",
	1611667447:  "oddball_include",
	-1354035140: "strongholds_include",
	412386272:   "strongholds_zone",
	-1574286981: "assault_include",
	-138058499:  "assault_exclude",
	-534119345:  "assault_bomb",
	1384999457:  "extraction_zone",
	-903313158:  "extraction_include",
	1525356238:  "infection_include",
	-626192115:  "infection_exclude",
	1673870030:  "elimination_include",
	1838764749:  "elimination_exclude",
	1192059526:  "skull_weapon",
	// --- Filtres d'activation hors PvP d'arène (craqués 2026-08-01/02) ---
	2140598169:  "firefight_include",
	248451123:   "minigame_include",
	-1875636905: "forge_include",
}

// LabelName retourne le nom d'un label, ou "" s'il n'est pas résolu.
func LabelName(hash int32) string { return labelNames[hash] }

// Role est le rôle d'objectif d'un objet, tel que le rejeu doit l'afficher.
type Role string

// Rôles reconnus. Un rôle n'est attribué QUE si un label de rôle explicite est
// porté par l'objet — jamais par heuristique de position ou de type_id.
const (
	RoleFlagSpawn         Role = "flag_spawn"         // point d'apparition du drapeau
	RoleFlagDelivery      Role = "flag_delivery"      // point de livraison / socle de capture
	RoleStockpileSocket   Role = "stockpile_socket"   // socle de dépôt Stockpile
	RoleStockpileNavpoint Role = "stockpile_navpoint" // repère de zone Stockpile
	RoleStrongholdZone    Role = "strongholds_zone"   // zone de Bastion
	RoleExtractionZone    Role = "extraction_zone"    // zone d'Extraction
	RoleOddballSpawn      Role = "oddball_spawn"      // apparition du crâne (Oddball)
	RoleAssaultBomb       Role = "assault_bomb"       // apparition de la bombe (Assaut)
)

// roleByLabel : labels qui portent un RÔLE (l'objet EST cet objectif).
// Les labels *_include / *_exclude ne sont PAS des rôles : ce sont des filtres
// d'activation par mode de jeu. Les confondre attribuerait un rôle à des dizaines
// d'objets de décor.
var roleByLabel = map[string]Role{
	"flag_spawn":         RoleFlagSpawn,
	"flag_delivery":      RoleFlagDelivery,
	"stockpile_socket":   RoleStockpileSocket,
	"stockpile_navpoint": RoleStockpileNavpoint,
	"strongholds_zone":   RoleStrongholdZone,
	"extraction_zone":    RoleExtractionZone,
	"oddball_spawn":      RoleOddballSpawn,
	// Mesuré sur les 199 variantes : 5 occurrences sur 5 cartes, soit EXACTEMENT
	// une par carte — la même signature que le crâne d'Oddball. C'est l'objet, pas
	// un filtre de mode.
	"assault_bomb": RoleAssaultBomb,
}

// Objective est un objet d'objectif identifié.
//
// AUCUN champ de nom de zone (A / B / C) : la donnée n'existe pas dans le
// fichier. Les trois zones Bastion d'une même carte portent le même type_id,
// les mêmes labels et le même hachage ; elles ne diffèrent que par position,
// dimensions et team_index. La lettre est attribuée à l'exécution — si elle
// devient nécessaire, elle viendra de là et se posera dans un champ DISTINCT,
// jamais devinée depuis l'ordre du fichier sans témoin.
type Objective struct {
	Role       Role     `json:"role"`
	TypeID     int32    `json:"type_id"`
	Pos        Vec3     `json:"pos"`
	Forward    Vec3     `json:"forward"`
	TeamIndex  int      `json:"team_index"` // -1 si neutre / non affecté
	InstanceID int32    `json:"instance_id"`
	Labels     []string `json:"labels"` // labels résolus
	Unresolved []int32  `json:"unresolved_labels,omitempty"`
	ObjectIdx  int      `json:"object_index"`
	// Shape est ABSENT quand l'objectif est ponctuel — le rendu doit alors
	// afficher un point. Jamais de disque par défaut (cf. shape.go).
	Shape *Shape `json:"shape,omitempty"`
}

// Objectives extrait les objets porteurs d'un rôle d'objectif.
func (v *Variant) Objectives() []Objective {
	out := make([]Objective, 0, 16)
	for _, o := range v.Objects {
		role, names, unknown := classify(o.Labels)
		if role == "" {
			continue
		}
		out = append(out, Objective{
			Role:       role,
			TypeID:     o.TypeID,
			Pos:        o.Pos,
			Forward:    o.Forward,
			TeamIndex:  o.TeamIndex,
			InstanceID: o.InstanceID,
			Labels:     names,
			Unresolved: unknown,
			ObjectIdx:  o.Index,
			Shape:      o.Shape(),
		})
	}
	return out
}

// UnresolvedLabels retourne les hashs de label présents dans la variante que la
// table ne sait pas nommer. Sert à mesurer honnêtement la couverture.
func (v *Variant) UnresolvedLabels() map[int32]int {
	out := map[int32]int{}
	for _, o := range v.Objects {
		for _, l := range o.Labels {
			if labelNames[l] == "" {
				out[l]++
			}
		}
	}
	return out
}

func classify(labels []int32) (Role, []string, []int32) {
	var role Role
	names := make([]string, 0, len(labels))
	var unknown []int32
	for _, h := range labels {
		name := labelNames[h]
		if name == "" {
			unknown = append(unknown, h)
			continue
		}
		names = append(names, name)
		if r, ok := roleByLabel[name]; ok && role == "" {
			role = r
		}
	}
	return role, names, unknown
}
