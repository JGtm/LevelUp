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
// court sur [from, to], appartient au bot de `pi` quand, aucun humain de la table ne tenant `pi`,
// les deux conditions tiennent ENSEMBLE :
//   - a `t`, UN SEUL bot distinct de `pi` est declare ;
//   - une declaration de ce bot couvre a la fois `t` et toute la vie.
//
// Sinon la lecture se tait et les voies d'avant reprennent. C'est une lecture du film, pas un
// repli : aucun seuil, aucune fenetre choisie.
//
// L'UNICITE SE LIT A LA CREATION, PAS PARMI LES SEULS BOTS QUI COUVRENT LA VIE : deux bots declares
// a l'instant ou le corps nait ne designent pas son occupant, meme quand un seul des deux couvre
// toute la vie : rien ne dit lequel des deux est ne dans ce corps. La lecture se tait.
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

// lireBotsDesIndexPartages construit la table sur les index partages que rend [bidsParIndex] (la
// seule mesure de « index declare par au moins deux bots distincts »), privee de ceux qu'un humain
// de la table tient. Un index d'un seul bot n'y entre pas : l'index le nomme deja (tableau de
// l'API, siege).
func lireBotsDesIndexPartages(bots []BotIdentity, idx types.PlayerIndexTable) botsDesIndexPartages {
	_, partages := bidsParIndex(bots)
	humains := indexToXUIDOf(idx.ByXUID)
	out := botsDesIndexPartages{}
	for _, b := range bots {
		if _, tenu := humains[b.FilmIndex]; tenu || !partages[b.FilmIndex] || b.Bid() == "" {
			continue
		}
		out[b.FilmIndex] = append(out[b.FilmIndex], b)
	}
	return out
}

// botDeclareSurLaVie rend le `bid` du bot de l'index `pi` que les declarations designent pour le
// corps cree a `tUS` dont la vie est `l`. Les deux conditions sont exigees ensemble :
//   - a l'instant de la creation, UN SEUL bot distinct de l'index est declare ([seulBotDeclareA]) :
//     deux bots declares a cet instant, rien ne les departage ;
//   - une declaration de CE bot couvre a la fois la creation et toute la vie.
//
// Faux sinon, et pour un instant anterieur a l'origine du film (negatif).
func (m botsDesIndexPartages) botDeclareSurLaVie(pi int, tUS int64, l lifeSpan) (string, bool) {
	de, a := min(tUS, l.from), max(tUS, l.to)
	if de < 0 {
		return "", false
	}
	elu, seul := m.seulBotDeclareA(pi, uint64(tUS))
	if !seul {
		return "", false
	}
	for _, b := range m[pi] {
		if b.Bid() == elu && declarationCouvre(b.Declarations, uint64(de), uint64(a)) {
			return elu, true
		}
	}
	return "", false
}

// seulBotDeclareA rend le `bid` du bot de l'index `pi` declare a l'instant `t`, quand il est le
// SEUL bot distinct a l'etre. Faux quand aucun bot ou plusieurs bots distincts y sont declares.
func (m botsDesIndexPartages) seulBotDeclareA(pi int, t uint64) (string, bool) {
	elu := ""
	for _, b := range m[pi] {
		if !declarationCouvre(b.Declarations, t, t) {
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
