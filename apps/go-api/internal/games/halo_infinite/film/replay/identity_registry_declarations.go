package replay

// identity_registry_declarations.go — LE CORPS D'UN INDEX QUE PLUSIEURS BOTS SE RELAIENT, NOMME PAR
// LES DECLARATIONS BOT_METADATA QUAND L'ENTITE SE TAIT.
//
// # LE TROU QU'IL FERME
//
// La lecture par entite (identity_registry_entites.go) lie le corps d'un index partage a l'entite
// `ti=9` qui vit a sa creation. Les entites ne se lisent qu'aux images-cles (une toutes les ~20 s) :
// un bot dont toute la presence tombe entre deux images-cles n'a AUCUNE entite, la lecture se tait,
// et son corps restait anonyme — `bidsParIndex` s'abstient sur un index que plusieurs bots
// declarent, `botNamesBySeat` aussi.
//
// # LA LECTURE
//
// BOT_METADATA est reecrit en tete de chaque chunk et a chaque changement : ses declarations sont
// datees a la frame pres (`BotIdentity.Declarations`). Le corps d'index `pi` cree a `t`, dont la vie
// court sur [from, to], appartient au bot de `pi` dont UNE declaration couvre a la fois `t` et
// toute la vie, quand ce bot est UNIQUE et qu'aucun humain de la table ne tient `pi`. Sinon la
// lecture se tait et les voies d'avant reprennent. C'est une lecture du film, pas un repli : aucun
// seuil, aucune fenetre choisie.
//
// L'ENTITE GARDE LA PRIORITE : le corps d'un bot peut preceder sa premiere declaration (le film
// ecrit la creation avant le paquet BOT_METADATA) ; la declaration se tait alors, l'entite non.
//
// LA VIE ENTIERE, PAS SEULEMENT LA CREATION : la creation du corps peut preceder la declaration de
// son bot ; un remplacant cree pendant que la declaration de son predecesseur court encore
// tomberait dans celle-ci. Exiger que la declaration couvre aussi toute la vie le refuse des que
// sa vie deborde la declaration du predecesseur : une vie qui deborde n'est pas prouvee etre celle
// de ce bot, et la lecture se tait.

import (
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// botsDesIndexPartages : pour chaque index que PLUSIEURS bots declarent et qu'aucun humain de la
// table ne tient, ses bots dont le `bid` est publie.
type botsDesIndexPartages map[int][]BotIdentity

// lireBotsDesIndexPartages construit la table. Un index d'un seul bot n'y entre pas : l'index le
// nomme deja (tableau de l'API, siege).
func lireBotsDesIndexPartages(bots []BotIdentity, idx types.PlayerIndexTable) botsDesIndexPartages {
	tenus := map[int]bool{}
	for _, i := range idx.ByXUID {
		tenus[i] = true
	}
	parIndex := map[int][]BotIdentity{}
	bids := map[int]map[string]bool{}
	for _, b := range bots {
		bid := b.Bid()
		if bid == "" || tenus[b.FilmIndex] {
			continue
		}
		parIndex[b.FilmIndex] = append(parIndex[b.FilmIndex], b)
		if bids[b.FilmIndex] == nil {
			bids[b.FilmIndex] = map[string]bool{}
		}
		bids[b.FilmIndex][bid] = true
	}
	out := botsDesIndexPartages{}
	for i, bs := range parIndex {
		if len(bids[i]) >= 2 {
			out[i] = bs
		}
	}
	return out
}

// botDeclareSurLaVie rend le `bid` de l'unique bot de l'index `pi` dont une declaration couvre la
// creation `tUS` du corps et toute la vie `l`. Faux quand aucun bot ou plusieurs bots distincts
// la couvrent.
func (m botsDesIndexPartages) botDeclareSurLaVie(pi int, tUS int64, l lifeSpan) (string, bool) {
	de, a := min(tUS, l.from), max(tUS, l.to)
	if de < 0 {
		return "", false
	}
	elu := ""
	for _, b := range m[pi] {
		if !declarationCouvre(b.Declarations, uint64(de), uint64(a)) {
			continue
		}
		if elu != "" && elu != b.Bid() {
			return "", false
		}
		elu = b.Bid()
	}
	return elu, elu != ""
}

// declarationCouvre dit si une declaration `[debut, fin)` (`fin == 0` = declare jusqu'au bout)
// contient tout l'intervalle [de, a].
func declarationCouvre(decls [][2]uint64, de, a uint64) bool {
	for _, d := range decls {
		fin := d[1]
		if fin == 0 {
			fin = math.MaxUint64
		}
		if d[0] <= de && a < fin {
			return true
		}
	}
	return false
}
