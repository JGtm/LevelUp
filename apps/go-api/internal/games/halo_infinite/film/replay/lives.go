package replay

// lives.go — LE PONT SLOT -> JOUEUR, LU AU LIEU D'ÊTRE VOTÉ.
//
// POURQUOI CE FICHIER REMPLACE UN VOTE. Le pont vivait dans owners.go, où il était élu :
// les lancers de grenade votaient pour désigner le propriétaire d'un slot. Un vote a
// besoin d'électeurs, et il y avait 70 lancers pour 99 vies — dont le premier à 73,1 s.
// Aucune vie antérieure ne pouvait donc être nommée, quelle que soit la qualité du
// décodage : **le défaut était dans le choix de la méthode, pas dans les données**.
// Résultat mesuré de ce vote : 26 slots couverts sur 99, et 147 tirs publiés sur 519.
//
// CE QU'ON FAIT À LA PLACE. Chaque vie de biped se termine par une mort, et le film porte
// le fil des morts : une victime, datée, nommée par son XUID. On nomme donc chaque vie par
// LA MORT QUI LA TERMINE. C'est une jointure sur un fait, pas une élection.
//
// MESURES sur 000d5950 (cmd/tmp_deathnaming), toutes avec leur témoin :
//
//	vies nommées                    90 / 105        témoin (morts replacées au hasard) : 10
//	écart d'appariement             médiane 34 ms, maximum 36 ms
//	slots changeant de porteur      0 / 90          — la table slot -> joueur est licite
//	tirs rattachés                  475 / 519 = 91,5 %   contre 398 par le vote supprimé
//	arme du tir dans le loadout     405 / 418 = 96,9 %   témoin (autre slot vivant) : 3,7 %
//
// LE DERNIER CONTRÔLE EST LE PLUS IMPORTANT : il ne partage AUCUNE pièce avec ce fichier.
// L'arme vient des records de dégât du flux de trames, le loadout du balayage des familles
// dans les records de biped des images-clés. Un rapport de 26x entre le rattachement et
// son témoin ne s'obtient pas par construction.
//
// L'INDEX DE JOUEUR N'EST PLUS RÉSOLU, IL EST LU. Il fut un temps calculé par affectation de
// coût minimal sur les 8! permutations ; le film l'écrit, et `player_index.go` le lit (26
// chunks concordants sur 000d5950, table identique à celle que le calcul produisait). Le pont
// n'a donc plus aucune part de choix : deux lectures composées, et rien d'autre.

// deathMatchWindowMS est l'écart maximal accepté entre la fin d'une vie et une mort du
// fil. La médiane mesurée étant de 34 ms et le maximum de 36, cette fenêtre borne le bruit
// d'horloge, pas le signal.
const deathMatchWindowMS = 150

// lifeGapUS : au-delà de ce trou dans un même slot, on ouvre une nouvelle vie. 5 s est
// très au-dessus du pas de réplication (~16 ms) et bien en deçà du temps de réapparition
// mesuré (médiane 8,0 s).
const lifeGapUS = 5_000_000

// Death est une mort du fil, telle que le film la porte : une identité et un instant.
// L'identité est le XUID — jamais un index (cf. la règle « un ordre n'est pas une
// identité », qui a déjà produit une fausse découverte dans ce chantier).
type Death struct {
	// XUID identifie la victime. Stable, global, indépendant de tout tri.
	XUID uint64
	// Gamertag est le nom porté PAR LE FILM lui-même, dans le même enregistrement que le xuid
	// (32 octets UTF-16LE). Il n'est pas obligatoire au rattachement — celui-ci ne travaille
	// que sur le xuid — mais il rend le rejeu lisible SANS base de données, ce qui est la
	// propriété que tout ce pipeline cherche à préserver. Vide si l'enregistrement ne le porte
	// pas ; l'identité reste alors le xuid.
	Gamertag string
	// TimeMS est l'instant de la mort sur l'horloge du MATCH (origine = début du match),
	// qui n'est pas celle du film. Le décalage entre les deux est résolu par mesure.
	TimeMS int64
}

// lifeSpan est une vie de biped : les positions d'un même slot sans trou majeur.
type lifeSpan struct {
	slot     uint32
	from, to int64  // microsecondes, horloge du film
	xuid     uint64 // identité lue dans le fil des morts ; 0 = non nommée
	// bid est l'identifiant STABLE d'un BOT, forme `bid(N.0)` — la même que la base emploie.
	//
	// POURQUOI UN SECOND CHAMP PLUTÔT QU'UN xuid ÉLARGI : un bot n'a pas de xuid, et lui en
	// fabriquer un (0, un négatif, un hachage du nom) le rendrait joignable avec un humain.
	// Les deux champs sont donc EXCLUSIFS : une vie porte un xuid, ou un `bid`, ou rien.
	// `xuid == 0` reste le témoin « aucune identité de JOUEUR », que tous les lecteurs de vies
	// nommées (`ViesNommees`, `nameTracksByLives`) emploient déjà — un bot n'entre pas en base.
	bid string
	// cause dit COMMENT la vie s'est terminée. Posée à la découpe (structure), écrasée par
	// [CauseVieMort] si le fil des morts apparie sa fin.
	//
	// LA MORT PRIME SUR LA STRUCTURE, jamais l'inverse : le trou de 5 s qui suit une mort est
	// le temps de réapparition, pas une coupure de réplication. Sans cette priorité, toute
	// mort suivie d'un respawn sortirait « coupure ».
	cause string
	// nomPar dit COMMENT ON SAIT À QUI la vie appartient — une question ORTHOGONALE à la
	// précédente, et les confondre est exactement ce qui a coûté la lecture d'isolement.
	//
	// UNE FERMETURE N'EST PAS UNE FIN. `nameClosedLives` déduit une identité par élimination
	// (un autre corps est réapparu, donc ce corps-ci était celui-là) ; elle ne dit RIEN sur
	// la façon dont la vie s'est terminée. Un survivant nommé par fermeture porte donc
	// `nomPar = closure` ET `cause = film_end` : il est identifié, et il n'est pas mort.
	nomPar string
}

// Les QUATRE causes de fin d'une vie. Elles sont toutes DÉTERMINABLES sans seuil arbitraire,
// et c'est la condition pour qu'elles existent : une cause qu'on devinerait ne serait qu'un
// avis présenté comme un fait.
//
//	CauseVieMort       le fil des morts apparie la fin de la vie (à deathMatchWindowMS,
//	                   médiane mesurée 34 ms). LA SEULE QUI DISE « CE JOUEUR EST MORT ».
//	CauseVieFinFilm    la réplication du slot s'arrête et ne reprend jamais : la vie court
//	                   jusqu'au bout de ce que le film montre. C'est le cas du SURVIVANT.
//	CauseVieCoupure    la vie se ferme sans qu'aucune MORT ne la borne. DEPUIS LE LOT 1.9.13
//	                   (2026-09-15) ce n'est PLUS « un trou de plus de lifeGapUS » : un trou de
//	                   réplication est devenu une LACUNE de la même vie (cf. lives_decoupe.go),
//	                   et cette cause ne subsiste que sur ce que le film ÉCRIT d'autre — une
//	                   apparition de corps (slot recyclé) ou une fin de manche — ou sur le repli
//	                   compté des joueurs dont le film n'écrit aucune mort.
const (
	CauseVieMort    = "death"
	CauseVieFinFilm = "film_end"
	CauseVieCoupure = "cut"
)

// Les DEUX provenances d'identité d'une vie. Elles répondent à « comment sait-on à qui elle
// appartient », jamais à « comment s'est-elle terminée ».
//
//	NomParMort        le fil des morts a nommé la vie par sa victime — une LECTURE.
//	NomParFermeture   une fermeture de slot l'a nommée par élimination (closures.go) — une
//	                  DÉDUCTION, et surtout PAS UNE MORT. Confondre les deux fabrique une
//	                  mort pour un survivant qui a tiré : c'est le P0 de la ronde 2
//	                  (2026-09-07), qui a coûté toute une lecture d'isolement.
const (
	NomParMort      = "death"
	NomParFermeture = "closure"
)

// ownersFromLives compose la table slot -> index de joueur à partir des vies nommées et du
// pont index -> XUID.
//
// LA TABLE EST LICITE PARCE QU'UN SLOT NE CHANGE PAS DE PORTEUR : mesuré 0 slot sur 90 sur
// 000d5950. Si un film violait cette propriété, la table serait fausse par construction —
// d'où le compteur de collisions rendu au rapport plutôt que masqué.
// Le second retour donne slot -> XUID, c'est-à-dire l'IDENTITÉ du porteur et non son rang.
// Les deux sortent du même parcours et de la même règle de collision : les séparer ferait
// diverger deux tables censées dire la même chose.
//
// LE SLOT EN COLLISION EST DESORMAIS MARQUE, PAS SEULEMENT COMPTE (2026-09-07). La boucle garde
// le PREMIER occupant nomme et compte les suivants, mais `SlotCollisions` est un TOTAL de match :
// aucun consommateur ne pouvait savoir QUEL slot etait concerne, et tous — les marques de
// portage, les ramassages, les frags sous equipement actif, les calques d'objectif — heritaient
// donc d'un nom ARBITRAIRE (celui du premier occupant, par ordre des vies) sur ces slots-la. Le
// troisieme retour rend l'ensemble des slots ambigus, pour que « ce slot a eu deux occupants »
// cesse d'etre indiscernable de « ce slot appartient a ce joueur ».
func ownersFromLives(
	lives []lifeSpan, xuidToIndex map[uint64]int,
) (map[uint32]int, map[uint32]uint64, map[uint32]bool) {
	out := map[uint32]int{}
	byXUID := map[uint32]uint64{}
	ambigus := map[uint32]bool{}
	for _, l := range lives {
		if l.xuid == 0 {
			continue
		}
		idx, ok := xuidToIndex[l.xuid]
		if !ok {
			continue
		}
		if prev, seen := out[l.slot]; seen && prev != idx {
			ambigus[l.slot] = true
			continue // conflit : on ne tranche pas, on ne publie pas
		}
		out[l.slot] = idx
		byXUID[l.slot] = l.xuid
	}
	return out, byXUID, ambigus
}

func absI64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func minI64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
