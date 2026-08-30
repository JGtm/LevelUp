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
// rejetées. Un hash non listé reste INCONNU — on ne devine pas de libellé. Une seule
// exception, par HASH et non par nom : la colline de KOTH (voir LabelHashHillRole), dont
// le nom n'a pas été retrouvé mais dont le rôle est établi par la géométrie et un film.
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
	// --- Total Control (craqués 2026-08-20, lot L13 volet C) ---
	//
	// La ligne de registre du 2026-08-08 (« zones de Total Control absentes des variantes de
	// CARTE, la donnée est ailleurs, ouvrir la variante de MODE ») est FAUSSE, et sa cause est
	// une seule lettre : la recherche de 2026-08-08 essayait `total_control_zone`. Le vrai nom
	// n'a pas d'underscore entre les deux mots. Les zones sont bien dans la variante de CARTE,
	// sous le motif habituel volume + filtre de mode.
	//
	// MESURE SUR LE CATALOGUE COMPLET (2026-08-25, 302 zones sur 21 entrées) — elle AMENDE le
	// relevé du 2026-08-20, qui portait sur 6 cartes et annonçait « toutes de type 850884602,
	// toutes neutres » :
	//
	//	forme        302 / 302 en portent une (aucune zone ponctuelle) — confirmé ;
	//	type_id      301 / 302 valent 850884602 ; UNE vaut -722308271 (Sylvanus, ci-dessous) ;
	//	team_index   296 / 302 sont neutres — 3 en équipe 0, 2 en équipe 2, 1 en ÉQUIPE 7.
	//	             Une équipe 7 sur un mode à deux camps finit d'établir que ce champ ne
	//	             porte rien ici : la table du titre sert le rôle en `neutral = true`.
	//
	// L'exception de Sylvanus est le seul objet du catalogue à porter `totalcontrol_zone`
	// SANS `totalcontrol_include`, et c'est aussi le seul de son type : une boîte de
	// 2,74 x 2,20 x 0,16 m, soit l'échelle d'une plaque de décor et non d'une zone de
	// contrôle. C'est un decoy probable, de la même famille que les cylindres
	// `minigame_include` qui portent le hash de la colline. Il est publié quand même : le
	// contrat de ce fichier est qu'un rôle vient d'un label de rôle EXPLICITE, jamais d'une
	// heuristique de type_id ou de dimensions — l'écarter sur sa taille serait exactement la
	// devinette que roleByLabel interdit. Consigné au registre, à trancher sur mesure (la
	// carte n'a aucun match Total Control).
	1750425936: "totalcontrol_zone",
	1589616376: "totalcontrol_include",
	// --- Filtres d'activation hors PvP d'arène (craqués 2026-08-01/02) ---
	2140598169:  "firefight_include",
	248451123:   "minigame_include",
	-1875636905: "forge_include",
	// firefight_objective est un ROLE, et c'est une RÉFUTATION du lot 5 (2026-08-08), qui
	// l'avait rangé parmi « 7 labels identifiés, aucun n'est un rôle ». Mesure du 2026-08-20 :
	// sur Oasis ET Highpower, 5 volumes type -1476457415 AVEC forme portent
	// [firefight_include, ce hash], accompagnés de 5 marqueurs ponctuels type -877512201
	// portant [firefight_include, 732028173] — exactement le motif volume+marqueur des zones
	// de Bastion et des collines. Le marqueur, lui, n'attribue aucun rôle (même règle que le
	// marqueur de Bastion) et son nom n'est pas retrouvé : il n'entre pas dans cette table.
	-1624244313: "firefight_objective",
	// --- Land Grab (cable au lot C catalogues, 2026-08-27 — ex-lot L) ---
	//
	// Les deux noms sont RESOLUS par la chasse murmur3 rejouable (TestHuntLabels, radicaux
	// landgrab/land_grab sur les .mvar versionnes du dump) : Cliffhanger porte 9 volumes
	// [landgrab_include, landgrab_zone] AVEC forme et 9 marqueurs ponctuels
	// [landgrab_include, -941529218] — exactement le motif volume+marqueur de Bastion et de
	// KOTH (le releve du 2026-08-19, en-tete des labels de KOTH ci-dessous, l'avait deja
	// nomme). Le census du catalogue (2026-08-27) compte 29 entrees porteuses du marqueur.
	// Le marqueur -941529218 n'attribue AUCUN role et son nom n'est pas retrouve : il
	// n'entre pas dans cette table (meme regle que les marqueurs de Bastion et de KOTH).
	-886053664: "landgrab_include",
	996801386:  "landgrab_zone",
}

// LabelName retourne le nom d'un label, ou "" s'il n'est pas résolu.
func LabelName(hash int32) string { return labelNames[hash] }

// LABELS DE KOTH — HASHS ETABLIS, NOMS NON RETROUVES (lot C-ter volet 2, 2026-08-19).
//
// Les collines de King of the Hill vivent dans la variante de CARTE, sous un motif identique
// a celui des autres modes du meme fichier (volume + filtre de mode + marqueur ponctuel) :
//
//	volumes    type -1476457415, boite ou cylindre, neutres, TOUS avec forme,
//	           labels [2133978317, -767961569] — 6 sur Catalyst, 5 sur Chasm, Shogun,
//	           Solitude - Ranked, Cliffhanger ; 5 de RACK sur le canevas Forge fo11/fo08 ;
//	marqueurs  type -877512201, sans forme, un par colline a sa position,
//	           labels [2133978317, -1482301937].
//
// Comparaison : Land Grab = volumes [landgrab_include, landgrab_zone] + marqueurs
// [landgrab_include, -941529218] ; Bastion = [strongholds_include, strongholds_zone] +
// [strongholds_include, -1246645531]. 2133978317 est donc le FILTRE de mode (l'« include »
// de KOTH), -767961569 le ROLE (la colline). Le nom snake_case des trois n'a pas ete
// retrouve : radicaux x suffixes (koth, hill, king_of_the_hill, kingofthehill, crown, zone,
// ... x _include/_zone/_hill/...), variantes de casse, dictionnaire de deux mots, et force
// brute sur [a-z_] jusqu'a 7 caracteres (10,9 milliards de radicaux) — aucune paire
// coherente. Ils N'ENTRENT PAS dans labelNames (une entree devinee y serait refusee par
// TestLabelTableIsSelfConsistent) : le role se pose par HASH, ci-dessous.
//
// LE ROLE ET LE FILTRE NE JOUENT PAS LE MEME ROLE. Le hash de role est AUSSI porte par deux
// cylindres `minigame_include` par carte de developpeur (Catalyst, Chasm : r = 1,4 m, un par
// camp, aux bases, type -1855279381) qui n'existent pas en KOTH. Le filtre 2133978317 sert a
// les ecarter — mais il n'est pas la marque de la colline : des cartes declarent leurs
// collines sans aucun filtre (Forbidden, Illusion). La regle exacte, et la mesure qui l'a
// corrigee le 2026-08-20, sont sur isHill (plus bas).
//
// Verifie sur le film 01e1f945 (Catalyst, 60 rampes de jauge) : la production apparie 56
// rampes a ces 6 volumes (52 avec les formes de Bastion/Extraction), et 13 des 20 periodes
// publiees changent de forme (registre_film/LOTCTER_VOLET2.md).
const (
	LabelHashHillRole    int32 = -767961569
	LabelHashHillInclude int32 = 2133978317
	// LabelHashHillMarker est documente pour l'inventaire ; il n'attribue AUCUN role (le
	// marqueur ponctuel n'est pas dessine — meme regle que le marqueur de Bastion, non resolu).
	LabelHashHillMarker int32 = -1482301937
)

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
	// RoleTotalControlZone est une zone de Total Control. ATTENTION AU VIVIER : le fichier
	// en declare 14 a 18 par carte, alors qu'une manche n'en ACTIVE que 3. Le choix des 3
	// n'est PAS dans le fichier (il ne l'est pas davantage pour la colline de KOTH) : ce
	// role publie des FORMES, pas un etat. Un consommateur qui dessinerait les 14 a 18 comme
	// autant de zones actives se tromperait — cf. l'entree Total Control d'objective_roles.toml.
	RoleTotalControlZone Role = "totalcontrol_zone"
	// RoleFirefightObjective est la zone d'objectif d'une manche de Firefight (PvE).
	RoleFirefightObjective Role = "firefight_objective"
	// RoleLandGrabZone est une zone de Land Grab. ATTENTION AU VIVIER, comme pour Total
	// Control et la colline : le fichier declare 9 zones par carte quand une VAGUE du mode
	// n'en active que 3 (trois vagues successives). Le choix des 3 actives n'est PAS dans
	// le fichier : ce role publie des FORMES, jamais un etat. Aucun film Land Grab
	// n'existe (expire) — le cablage sert les matchs FUTURS, par degradation d'absence.
	RoleLandGrabZone Role = "landgrab_zone"
	// RoleHill est la colline de King of the Hill — attribuee par le hash de ROLE
	// LabelHashHillRole (voir isHill), jamais par un nom.
	RoleHill Role = "hill"
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
	// `totalcontrol_include`, lui, N'EST PAS ici : c'est le filtre de mode, pas l'objet.
	"totalcontrol_zone":   RoleTotalControlZone,
	"firefight_objective": RoleFirefightObjective,
	// `landgrab_include` n'est pas ici non plus : filtre de mode, pas l'objet.
	"landgrab_zone": RoleLandGrabZone,
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
//
// UN OBJET PEUT PORTER PLUSIEURS RÔLES, et il produit alors PLUSIEURS objectifs — un par
// rôle, même position, même forme, même `object_index` (mesuré le 2026-08-20). Ce n'est pas
// une commodité : dans Forge, un volume se déclare pour plusieurs modes à la fois, et c'est
// exactement pourquoi les labels sont une LISTE. Sur Empyrean (asset d035fc3e), les trois
// zones de Bastion portent `[strongholds_include, strongholds_zone]` ET la paire de hashs
// de la colline ; les quatre marqueurs ponctuels de KOTH tombent sur ces trois volumes plus
// un quatrième. La carte a donc QUATRE collines. L'ancien code retenait le PREMIER rôle
// nommé et n'en publiait qu'UNE — le catalogue livré porte « Empyrean : 1 colline », et
// c'est ce choix-là, pas le fichier, qui perdait les trois autres.
//
// Les consommateurs lisent par rôle (`ZonesOfRole`) : un volume qui est zone de Bastion en
// Bastion et colline en KOTH doit apparaître dans les deux listes, jamais dans une seule.
func (v *Variant) Objectives() []Objective {
	out := make([]Objective, 0, 16)
	for _, o := range v.Objects {
		roles, names, unknown := classify(o.Labels)
		for _, role := range roles {
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

// classify rend TOUS les rôles portés par un objet, dans l'ordre des labels (la colline en
// dernier, elle se lit sur la liste entière), sans doublon — un même nom de label peut
// figurer deux fois dans le fichier.
func classify(labels []int32) ([]Role, []string, []int32) {
	names := make([]string, 0, len(labels))
	var unknown []int32
	var roles []Role
	seen := make(map[Role]bool, 2)
	add := func(r Role) {
		if !seen[r] {
			seen[r] = true
			roles = append(roles, r)
		}
	}
	for _, h := range labels {
		name := labelNames[h]
		if name == "" {
			unknown = append(unknown, h)
			continue
		}
		names = append(names, name)
		if r, ok := roleByLabel[name]; ok {
			add(r)
		}
	}
	if isHill(labels) {
		add(RoleHill)
	}
	return roles, names, unknown
}

// isHill dit si l'objet est une colline de King of the Hill.
//
// LA REGLE : il porte le hash de ROLE, et il n'est pas revendique par un AUTRE mode. Un
// objet est revendique par un autre mode quand il porte le filtre d'activation de ce mode
// (`<mode>_include`) sans porter celui de KOTH. Les hashs de KOTH restent comptes parmi les
// labels non resolus : leur nom n'est pas connu, et on ne le devine pas.
//
// POURQUOI LA PAIRE N'EST PLUS EXIGEE (mesure du 2026-08-20 ; corrige la regle posee au lot
// C-ter volet 2 le 2026-08-19). Le filtre de KOTH n'a jamais servi qu'a ECARTER un decoy
// precis, documente plus haut : deux cylindres par carte de developpeur qui portent le hash
// de role ET `minigame_include` (r = 1,4 m, un par camp, type -1855279381). Exiger la paire
// ecartait bien ce decoy — mais avec lui, toutes les collines que leur fichier declare SANS
// filtre. Mesure sur les deux cartes concernees :
//
//	Illusion (9e821f5e) : 5 volumes type 1818458590 portant le hash de role SEUL (aucun
//	   autre label, resolu ou non), et 5 marqueurs ponctuels de KOTH type -877512201
//	   [filtre KOTH + marqueur] AUX MEMES POSITIONS (3 au centimetre, les 2 autres a 0,59 m
//	   et 0,22 m). Le marqueur dit ou est la colline ; le volume dit sa forme.
//	Forbidden (87c03bfd) : les memes 5 volumes type 1818458590, formes TOUTES DIFFERENTES
//	   (7,80x4,80 / 5,40x4,20 / 6,00x5,25 / 8,17x3,64 / 7,90x2,60 m).
//
// Ces deux cartes ont de vrais matchs Arena KOTH au registre (Forbidden 3, Illusion 2, tous
// en Quick Play, variante de mode `KOTH:Arena`). Le registre du 2026-08-20 supposait un
// « prefab Forge partage » au vu d'`instance_id` identiques : c'etait un artefact — TOUS les
// objets des variantes `*_map.mvar` ont `instance_id = 0` (deja consigne au lot C-ter, §6),
// et cinq formes differentes ne sont pas un prefab partage.
func isHill(labels []int32) bool {
	role, kothInclude, foreignInclude := false, false, false
	for _, h := range labels {
		switch h {
		case LabelHashHillRole:
			role = true
		case LabelHashHillInclude:
			kothInclude = true
		default:
			if isModeIncludeLabel(labelNames[h]) {
				foreignInclude = true
			}
		}
	}
	return role && (kothInclude || !foreignInclude)
}

// isModeIncludeLabel dit si un nom de label RESOLU est un filtre d'activation par mode.
// La convention est celle de labelNames et de l'en-tete de roleByLabel : `<mode>_include`.
// Un hash NON resolu ne peut pas etre juge — on ne devine pas plus ici qu'ailleurs.
func isModeIncludeLabel(name string) bool {
	const suffix = "_include"
	return len(name) > len(suffix) && name[len(name)-len(suffix):] == suffix
}
