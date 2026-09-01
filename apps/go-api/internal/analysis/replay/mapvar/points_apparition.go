package mapvar

// points_apparition.go — LES POINTS D'APPARITION D'OBJET RAMASSABLE d'une variante, et la
// nature que la mesure leur donne.
//
// CE FICHIER NE REMPLACE PAS `socles.go`, IL LE COMPLETE. Les trois `type_id` de socle d'ARME
// (`power`, `rack`, `powerup`) restent la propriete exclusive de `PadFamilyOf` et alimentent
// `PadSpots` sans changer d'un iota : ils partent au client par des chemins livres (datation
// des occupations, tableau de la page match), et ce lot n'y touche pas. Les points d'ici sont
// une SECONDE liste, publiee a cote.
//
// D'OU VIENT LA LISTE. De la recette `himap.EstPointDApparition`, qui interroge le catalogue
// Forge du jeu : 16 types retenus sur 4 235 tags `food`, dont 3 sont les socles d'arme deja
// connus — restent 13 points ici. Elle n'est pas recopiee a la main —
// le garde-rail `himap.TestRecetteRedonneLaTableDesPoints` la re-derive depuis les fichiers du
// jeu et echoue si cette table derive.
//
// POURQUOI UNE TABLE FIGEE PLUTOT QUE LA RECETTE A L'EXECUTION : la generation du catalogue est
// HORS LIGNE et doit rester reproductible sur une machine sans le jeu installe. La recette est
// la source, la table est son empreinte, le garde-rail est le lien.
//
// LA NATURE EST UNE MESURE, PAS UNE LECTURE. La chaine de tags ne descend pas jusqu'a l'objet
// engendre (elle s'arrete au `fosp`, references irresolvables). Le typage vient du CANAL NATIF
// des ramassages, sur Catalyst : 199 ramassages, 66 non-arme tous nommes par le manifeste du
// titre, 41 apparies sous le metre a un point catalogue.
//
//	TEMOIN : le taux de base du film — 66 non-arme sur 199 = 33,2 %. Un point qui rend 33 % de
//	non-arme ne porte aucune information.
//
//	CE QUI VALIDE LA METHODE : les trois socles PROUVES tombent exactement ou ils doivent sans
//	avoir servi a calibrer quoi que ce soit — `power` rend 0,0 % de non-arme, `rack` rend
//	36,4 % (le taux de base, donc du bruit), `powerup` rend 100 % de `powerup_overshield`.
//
// CE QUE CETTE TABLE N'EST PAS : une preuve pour les natures fines. Le critere ECRIT AVANT la
// mesure (« part non-arme au-dessus du taux de base ») est ce qui separe `SpawnKindEquipment`
// et `SpawnKindGrenade` de `SpawnKindUnknown`. Le choix ENTRE grenade et equipement, lui, est
// juge dans les seules non-armes (15 sur 16 grenades pour l'un, 11 sur 11 equipements pour
// l'autre) et repose sur UNE carte et UN film : c'est une hypothese etayee, pas un fait.

import "sort"

// SpawnKind est la nature d'un point d'apparition, telle que la mesure la donne.
type SpawnKind string

const (
	// SpawnKindGrenade — point mesure a 84,2 % de non-arme (taux de base 33,2 %), dont
	// 15 grenades sur 16.
	SpawnKindGrenade SpawnKind = "grenade"
	// SpawnKindEquipment — point mesure a 91,7 % de non-arme, dont 11 equipements sur 11.
	SpawnKindEquipment SpawnKind = "equipment"
	// SpawnKindUnknown — la recette le reconnait comme point d'apparition, mais aucune mesure
	// ne dit CE QU'IL fait naitre. Publie tel quel : un point dont la nature est inconnue vaut
	// mieux qu'un point range de force dans une categorie.
	//
	// LE CAS `0x0CD504B0` EXPLIQUE POURQUOI CETTE VALEUR EXISTE. Une premiere mesure, qui ne
	// comptait que les ramassages non-arme, l'avait type « grenade » sur trois observations.
	// Le controle symetrique montre qu'il recoit DIX armes : sa part de non-arme est 23,1 %,
	// SOUS le taux de base. Ce n'est pas un point d'equipement, et une mesure partielle
	// l'aurait publie comme tel.
	SpawnKindUnknown SpawnKind = "unknown"
)

// spawnPointTypes — LES 13 POINTS D'APPARITION : les 16 types de la recette MOINS les 3 socles, avec la nature que la mesure leur donne.
//
// Les trois socles d'ARME (`0x5F379533`, `0x6253CFC0`, `0x5E86D110`) sont ABSENTS de cette
// table ALORS QUE LA RECETTE LES RETIENT : ils appartiennent a `padFamilies` et sortent par
// `PadSpots`. Les inscrire ici les ferait publier DEUX FOIS, et casserait la non-regression que
// ce lot doit tenir.
// LA TABLE EST CLEE EN uint32, ET C'EST DELIBERE : plus de la moitie de ces `type_id` depassent
// 0x7FFFFFFF et ne s'ecrivent pas en `int32` sans passer par leur complement a deux. Un
// `-1376852264` dans une table ne se reconnait dans aucun dump ; `0xADEEE6D8` si. La conversion
// se fait au seul point d'entree, `SpawnKindOf`.
var spawnPointTypes = map[uint32]SpawnKind{
	0x0CD504B0: SpawnKindUnknown,   // mesure : 23,1 % de non-arme, SOUS le taux de base
	0x11CBFF52: SpawnKindUnknown,   // non observe sur le film de mesure
	0x2BEF1E2D: SpawnKindUnknown,   // mesure : n = 2, trop peu
	0x3BF9FAF3: SpawnKindUnknown,   // non observe
	0x5F3FA667: SpawnKindUnknown,   // mesure : 0,0 % de non-arme sur n = 4
	0x682B8B57: SpawnKindUnknown,   // non observe
	0x76110919: SpawnKindUnknown,   // non observe
	0xADEEE6D8: SpawnKindGrenade,   // 84,2 % de non-arme, dont 15 grenades sur 16
	0xAEDF9CF0: SpawnKindUnknown,   // mesure : n = 3, pas de majorite
	0xE42158DF: SpawnKindEquipment, // 91,7 % de non-arme, dont 11 equipements sur 11
	0xF5AD870C: SpawnKindUnknown,   // non observe
	0xF8EC7E1E: SpawnKindUnknown,   // mesure : n = 2, trop peu
	0xF93240BA: SpawnKindUnknown,   // non observe
}

// SpawnKindOf rend la nature d'un `type_id` de point d'apparition, et FAUX si ce type n'en est
// pas un. Un socle d'ARME rend FAUX ici : il sort par `PadFamilyOf`.
func SpawnKindOf(typeID int32) (SpawnKind, bool) {
	k, ok := spawnPointTypes[uint32(typeID)]
	return k, ok
}

// SpawnPointTypeCount rend le nombre de types de la table — le garde-rail s'en sert pour dire
// combien il compare, plutot que d'affirmer un nombre en dur dans son message.
func SpawnPointTypeCount() int { return len(spawnPointTypes) }

// SpawnPointTypeIDs rend les types de la table, tries — ordre deterministe pour les gardes.
func SpawnPointTypeIDs() []int32 {
	out := make([]int32, 0, len(spawnPointTypes))
	for id := range spawnPointTypes {
		out = append(out, int32(id))
	}
	sort.Slice(out, func(i, j int) bool { return uint32(out[i]) < uint32(out[j]) })
	return out
}

// SpawnPoint est UN point d'apparition d'objet ramassable de la carte.
type SpawnPoint struct {
	// Pos est la position de l'objet REPRESENTATIF, meme repere que PadSpot.Pos.
	Pos Vec3
	// TypeID est le type brut du representant, publie a cote de la nature — jamais remplace
	// par elle (regle du depot : on ne stocke pas une resolution qui peut s'ameliorer).
	TypeID int32
	// Kind est la nature mesuree, ou `unknown`.
	Kind SpawnKind
	// InstanceID est l'identifiant que le JEU donne au representant.
	InstanceID int32
	// Objects est le nombre d'objets FUSIONNES dans ce point.
	Objects int
	// Mixed dit que des objets de NATURES DIFFERENTES ont fusionne ici — meme drapeau et meme
	// raison que `PadSpot.Mixed`.
	//
	// SANS LUI L'INFORMATION ETAIT IRRECUPERABLE : le regroupement garde le type du
	// representant et JETTE ceux des absorbes. Sur le corpus, 234 des 1 934 points sont des
	// fusions ; si l'une d'elles absorbait un point de grenade dans un point d'equipement, le
	// catalogue publierait une nature fausse sans que rien ne le signale. Le producteur peut
	// desormais le DIRE plutot que de le taire.
	Mixed bool
}

// SpawnPoints rend les points d'apparition d'une variante, dans le MEME ordre deterministe et
// avec le MEME regroupement que `PadSpots` — meme rayon `PadSpotMergeM`, meme tri spatial.
// Deux fonctions qui rangent des positions de la meme carte ne doivent pas les ranger
// differemment : un diff du catalogue se lirait deux fois moins bien.
func SpawnPoints(v *Variant) []SpawnPoint {
	if v == nil {
		return nil
	}
	objs := make([]Object, 0, 64)
	for _, o := range v.Objects {
		if _, ok := SpawnKindOf(o.TypeID); ok {
			objs = append(objs, o)
		}
	}
	sort.SliceStable(objs, func(i, j int) bool {
		return lessPadSpot(objs[i], objs[j])
	})
	out := make([]SpawnPoint, 0, len(objs))
	for _, o := range objs {
		kind, _ := SpawnKindOf(o.TypeID)
		if i := nearestSpawnPoint(out, o.Pos); i >= 0 {
			out[i].Objects++
			out[i].Mixed = out[i].Mixed || out[i].Kind != kind
			continue
		}
		out = append(out, SpawnPoint{
			Pos: o.Pos, TypeID: o.TypeID, Kind: kind, InstanceID: o.InstanceID, Objects: 1,
		})
	}
	return out
}

// nearestSpawnPoint rend l'index du premier point a moins de PadSpotMergeM, ou -1.
func nearestSpawnPoint(pts []SpawnPoint, p Vec3) int {
	for i := range pts {
		if Dist3(pts[i].Pos, p) < PadSpotMergeM {
			return i
		}
	}
	return -1
}
