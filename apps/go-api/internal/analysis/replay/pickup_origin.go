package replay

import "levelup/go-api/internal/analysis/filmdec"

// pickup_origin.go — D'OU VIENT L'OBJET QU'UN JOUEUR RAMASSE.
//
// LA QUESTION, ET POURQUOI ELLE A RESISTE DEUX LOTS. Un ramassage non-arme du canal natif dit
// QUI, QUAND et QUOI. Il ne dit pas si l'objet attendait a un point d'apparition de la carte ou
// s'il gisait au sol, lache par un mort. Trois voies geometriques ont echoue avant celle-ci :
// la distance seule (mediane 1,33 m, 46 % sous le metre — sans pouvoir de separation), la fin
// de vie d'objet (25,6 % d'injectivite), la levitation (separation 0,00 m) et la recurrence des
// naissances (elle mesure le trafic des joueurs, pas les points d'apparition). Elles sont
// consignees dans `.ai/V7.5/film_re/NOTE_ORIGINE_LEVITATION_2026-09-01.md`, section « ce qu'il
// ne faut plus retenter ».
//
// CE QUI DEBLOQUE : LA CARTE, PAS LE FILM. Le fichier de variante declare les points
// d'apparition — la recette `himap.EstPointDApparition` les reconnait sans connaitre la carte,
// et le catalogue fige les positions au centimetre. Le film n'a plus a DEVINER un point : il a
// seulement a dire si le ramassage s'est produit dessus.
//
// LES DEUX ORIGINES, ET LEUR ASYMETRIE ASSUMEE :
//
//	spawner  le ramassage a lieu a moins de PickupOriginMatchM d'un point d'apparition
//	         NON-ARME catalogue pour cette carte. C'est un fait de CARTE, connu au centimetre
//	         et des la premiere image.
//	ground   le ramassage a lieu a moins de PickupOriginMatchM d'une pose dont l'origine
//	         MESUREE est `dropped` — un objet libere par une mort (cf.
//	         EquipmentPlacement.Origin). La regle n'est pas neuve : elle REUTILISE la mesure
//	         de production, elle ne la refait pas.
//	absent   ni l'un ni l'autre. ABSTENTION EXPLICITE, jamais un repli. Un client qui lit
//	         l'absence ne doit pas conclure `ground` : il doit conclure « on ne sait pas ».
//
// L'ORDRE COMPTE ET IL EST FIXE : `spawner` l'emporte. Un objet lache pres d'un point
// d'apparition existe, et le compte des deux cas est publie pour qu'on puisse le mesurer ; mais
// entre un fait de carte au centimetre et une inference de film, le fait de carte gagne.
//
// LE MODE DECIDE, LA CARTE DECLARE. Une carte porte ses points d'apparition meme quand le mode
// ne les allume pas : sur un film Super Fiesta, les 74 points de Cliffhanger restent eteints et
// le seau `spawner` doit rester bas. C'est mesure, et c'est le contre-exemple qui donne son
// sens au chiffre.

// PickupOriginMatchM est le rayon, en metres, sous lequel un ramassage est attribue a un point
// d'apparition ou a une pose.
//
// C'EST `MapWeaponPadMatchM`, PAS UN SEUIL NEUF, et l'argument est le meme : les positions du
// catalogue sont au centimetre (32 oracles apparies, mediane 0,01 m), et le metre est une marge
// et non une tolerance. Un seuil distinct sur la meme geometrie rendrait les deux mesures
// incomparables — c'est exactement ce que le depot reproche au « magic number ».
const PickupOriginMatchM = MapWeaponPadMatchM

// PickupOriginPosMaxUS est l'ecart temporel maximal admis entre le ramassage et la position du
// ramasseur qui le localise. 100 ms : a la cadence des paquets de position c'est l'ordre de la
// frame, et un joueur ne traverse pas un metre dans cet intervalle. Au-dela, on refuse de
// localiser plutot que de placer le ramassage au mauvais endroit.
const PickupOriginPosMaxUS = 100_000

// Identifiants STABLES du document (meme regle que EquipmentPlacement.Origin et
// GroundWeapon.End) : un client qui lit une valeur inconnue la traite comme une absence.
const (
	// PickupOriginSpawner : ramasse sur un point d'apparition catalogue de la carte.
	PickupOriginSpawner = "spawner"
	// PickupOriginGround : ramasse sur une pose dont l'origine mesuree est `dropped`.
	PickupOriginGround = "ground"
)

// MapSpawnPoint est UN point d'apparition non-arme de la carte, tel que le builder le recoit.
// C'est la projection du catalogue (`MapSpawnPointSpot`) : le builder n'a pas a connaitre la
// forme du fichier fige.
type MapSpawnPoint struct {
	X, Y, Z float32
	// Kind est la nature du point (`grenade`, `equipment`, `unknown`). Elle n'entre PAS dans
	// la decision d'origine — un ramassage sur un point est un ramassage sur un point, quelle
	// que soit la nature qu'on prete au point. Elle est portee pour la couverture.
	Kind string
}

// pickupOriginJudge porte tout ce qu'il faut pour statuer sur l'origine d'un ramassage.
//
// C'EST UN STRUCT ET NON QUATRE PARAMETRES DE PLUS : `buildPickups` en avait deja cinq, le
// plafond du depot. Grouper ce qui repond a UNE question vaut mieux qu'un sixieme argument.
type pickupOriginJudge struct {
	// points : les points d'apparition NON-ARME de la carte. Vide = le juge ne rend jamais
	// `spawner`, et la couverture le dit par MapCatalogMissing.
	points []MapSpawnPoint
	// catalogKnown dit si la carte a ete TROUVEE au catalogue. Distinguer « carte absente du
	// catalogue » de « carte connue sans point » est une DECISION PRODUIT : le trou doit se
	// VOIR, et surtout il ne doit pas se combler par un telechargement pendant la cuisson —
	// la generation d'artefact reste hors ligne (doctrine du depot).
	catalogKnown bool
	// posBySlot : les positions des bipedes, par slot, pour localiser le ramasseur.
	posBySlot map[uint32][]bipedPos
	// dropped : les positions des poses dont l'origine mesuree est `dropped`, avec l'instant
	// de la pose — un objet ne peut pas etre ramasse avant d'avoir ete lache.
	dropped []droppedSpot
}

// bipedPos est le minimum utile d'une position de bipede pour ce juge.
type bipedPos struct {
	tsUS    uint64
	x, y, z float32
}

// droppedSpot est une pose `dropped` : ou, et a partir de quand.
type droppedSpot struct {
	t       int
	x, y, z float32
}

// origineDe statue sur UN ramassage. Rend la chaine vide quand rien n'est etabli.
func (j *pickupOriginJudge) origineDe(slot uint32, tsUS uint64, frame int) string {
	x, y, z, ok := j.positionDe(slot, tsUS)
	if !ok {
		return ""
	}
	// `spawner` d'abord : un fait de carte l'emporte sur une inference de film.
	for _, p := range j.points {
		if dist3([3]float32{x, y, z}, [3]float32{p.X, p.Y, p.Z}) < PickupOriginMatchM {
			return PickupOriginSpawner
		}
	}
	for _, d := range j.dropped {
		if d.t > frame {
			continue // un objet ne se ramasse pas avant d'etre lache
		}
		if dist3([3]float32{x, y, z}, [3]float32{d.x, d.y, d.z}) < PickupOriginMatchM {
			return PickupOriginGround
		}
	}
	return ""
}

// positionDe rend la position du ramasseur a l'instant du ramassage, si elle est assez proche
// dans le temps.
func (j *pickupOriginJudge) positionDe(slot uint32, tsUS uint64) (x, y, z float32, ok bool) {
	l := j.posBySlot[slot]
	if len(l) == 0 {
		return 0, 0, 0, false
	}
	best := -1
	bd := uint64(1) << 62
	for i, p := range l {
		d := p.tsUS - tsUS
		if p.tsUS < tsUS {
			d = tsUS - p.tsUS
		}
		if d < bd {
			bd, best = d, i
		}
	}
	if best < 0 || bd > PickupOriginPosMaxUS {
		return 0, 0, 0, false
	}
	return l[best].x, l[best].y, l[best].z, true
}

// newPickupOriginJudge assemble le juge a partir de ce que le builder a deja en main.
//
// IL REND TOUJOURS UN JUGE, jamais nil : un juge sans point et sans pose rend systematiquement
// l'abstention, ce qui est le comportement voulu, alors qu'un nil obligerait chaque appelant a
// se souvenir du cas. La couverture, elle, distingue bien « carte absente » de « carte vide ».
func newPickupOriginJudge(opt Options, pos []filmdec.BipedPosition,
	placements []EquipmentPlacement,
) *pickupOriginJudge {
	j := &pickupOriginJudge{
		points:       opt.SpawnPoints,
		catalogKnown: opt.MapCatalogKnown,
		posBySlot:    make(map[uint32][]bipedPos, 32),
	}
	for _, p := range pos {
		if !p.HasWorld {
			continue
		}
		j.posBySlot[p.Slot] = append(j.posBySlot[p.Slot],
			bipedPos{tsUS: p.TimestampUS, x: p.X, y: p.Y, z: p.Z})
	}
	for _, pl := range placements {
		// SEULES LES POSES `dropped` : ce sont celles que la mesure de production rattache a
		// une FIN DE VIE, donc a un objet libere par une mort. Une pose `deployed` est un
		// geste du joueur, pas un objet qui gisait ; la confondre ferait appeler `ground` un
		// mur qu'on vient de deployer.
		if pl.Origin != OriginDropped {
			continue
		}
		j.dropped = append(j.dropped, droppedSpot{t: pl.T0, x: pl.X, y: pl.Y, z: pl.Z})
	}
	return j
}
