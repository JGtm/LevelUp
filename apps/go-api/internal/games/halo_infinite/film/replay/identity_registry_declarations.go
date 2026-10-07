package replay

// identity_registry_declarations.go — LE CORPS D'UN INDEX QUE PLUSIEURS OCCUPANTS SE RELAIENT, NOMME
// PAR LES DECLARATIONS BOT_METADATA QUAND L'ENTITE SE TAIT.
//
// # LE TROU QU'IL FERME
//
// La lecture par entite (identity_registry_entites.go) lie le corps d'un index partage a l'entite
// `ti=9` qui vit a sa creation. Les entites ne se lisent qu'aux images-cles (une toutes les ~20 s) :
// un bot dont toute la presence tombe entre deux images-cles n'a AUCUNE entite, la lecture se tait,
// et son corps restait anonyme — ou nomme, par le repli de l'occupation du siege, du nom de l'humain
// qui tient l'index plus tard.
//
// # LA LECTURE
//
// BOT_METADATA est reecrit en tete de chaque chunk et a chaque changement : ses declarations sont
// datees a la frame pres (`BotIdentity.Declarations`). La lecture couvre les index qu'au moins deux
// bots declarent, et ceux qu'un humain de la table tient et qu'au moins un bot declare. Le corps
// d'index `pi` cree a `t`, dont la vie court sur [from, to], appartient au bot `b` quand :
//
//   - l'humain de `pi`, s'il y en a un, est ABSENT sur toute la creation et toute la vie : il a au
//     moins une entite de cet index, et aucune de ses entites (celles qu'aucune declaration de bot ne
//     revendique) ne vit a un instant de [t, to] (fenetre large) ;
//   - et l'une des deux conditions :
//     a. a `t`, `b` est le SEUL bot distinct de `pi` declare, et une de ses declarations couvre a la
//     fois `t` et toute la vie ;
//     b. a `t`, AUCUN bot de `pi` n'est declare (le film ecrit la creation avant le paquet
//     BOT_METADATA qui declare son bot), une seule declaration de l'index croise [t, to] — celle de
//     `b`, nee apres `t` —, elle couvre la vie jusqu'a son terme, et aucune image-cle porteuse ne
//     tombe entre la creation et sa naissance : BOT_METADATA est reecrit en tete de chaque chunk, un
//     bot deja la a cette image-cle y serait declare.
//
// Sinon la lecture se tait et les voies d'avant reprennent. C'est une lecture du film, pas un
// repli : aucun seuil, aucune fenetre choisie.
//
// L'UNICITE SE LIT A LA CREATION, PAS PARMI LES SEULS BOTS QUI COUVRENT LA VIE : deux bots declares
// a l'instant ou le corps nait ne designent pas son occupant, meme quand un seul des deux couvre
// toute la vie : rien ne dit lequel des deux est ne dans ce corps. La lecture se tait.
//
// L'ENTITE GARDE LA PRIORITE : quand l'entite vivante a la creation designe l'occupant, c'est elle
// qui nomme ([creationReport.poser] la lit d'abord).
//
// LA VIE ENTIERE, PAS SEULEMENT LA CREATION : un remplacant cree pendant que la declaration de son
// predecesseur court encore tomberait dans celle-ci. Exiger que la declaration couvre aussi toute la
// vie le refuse des que sa vie deborde la declaration du predecesseur : une vie qui deborde n'est pas
// prouvee etre celle de ce bot, et la lecture se tait.

import (
	"math"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// lectureParDeclaration porte, pour chaque index que la lecture couvre, ses bots dont le `bid` est
// publie ; les index qu'un humain de la table tient ; et le balayage des entites, qui prouve
// l'absence de cet humain et date les images-cles porteuses.
type lectureParDeclaration struct {
	bots    map[int][]BotIdentity
	humains map[int]bool
	scan    grammar.PlayerEntityScan
}

// lireLesDeclarations construit la lecture : les index qu'au moins deux bots distincts declarent
// ([bidsParIndex]), et ceux qu'un humain de la table tient et qu'au moins un bot declare. Un index
// d'un seul bot sans humain n'y entre pas : l'index le nomme deja (tableau de l'API, siege).
func lireLesDeclarations(bots []BotIdentity, idx types.PlayerIndexTable,
	scan grammar.PlayerEntityScan) lectureParDeclaration {
	_, partages := bidsParIndex(bots)
	humains := indexToXUIDOf(idx.ByXUID)
	m := lectureParDeclaration{bots: map[int][]BotIdentity{}, humains: map[int]bool{}, scan: scan}
	for _, b := range bots {
		_, tenu := humains[b.FilmIndex]
		if b.Bid() == "" || (!tenu && !partages[b.FilmIndex]) {
			continue
		}
		m.bots[b.FilmIndex] = append(m.bots[b.FilmIndex], b)
		m.humains[b.FilmIndex] = m.humains[b.FilmIndex] || tenu
	}
	return m
}

// botDeclareSurLaVie rend le `bid` du bot de l'index `pi` que les declarations designent pour le
// corps cree a `tUS` dont la vie est `l` (cf. l'en-tete). Faux sinon, et pour un instant anterieur a
// l'origine du film (negatif).
func (m lectureParDeclaration) botDeclareSurLaVie(pi int, tUS int64, l lifeSpan) (string, bool) {
	de, a := min(tUS, l.from), max(tUS, l.to)
	if de < 0 || len(m.bots[pi]) == 0 {
		return "", false
	}
	if m.humains[pi] && !m.humainAbsent(pi, uint64(de), uint64(a)) {
		return "", false
	}
	t := uint64(tUS)
	switch declares := m.botsDeclaresA(pi, t); len(declares) {
	case 1:
		if m.declarationDuBotCouvre(pi, declares[0], uint64(de), uint64(a)) {
			return declares[0], true
		}
		return "", false
	case 0:
		return m.botDeclareApresLaCreation(pi, t, uint64(a))
	default:
		return "", false
	}
}

// botsDeclaresA rend les `bid` DISTINCTS des bots de l'index `pi` declares a l'instant `t`.
func (m lectureParDeclaration) botsDeclaresA(pi int, t uint64) []string {
	var out []string
	for _, b := range m.bots[pi] {
		if declarationCouvre(b.Declarations, t, t) && !contientBid(out, b.Bid()) {
			out = append(out, b.Bid())
		}
	}
	return out
}

// declarationDuBotCouvre dit qu'une declaration du bot `bid` de l'index `pi` couvre tout [de, a].
func (m lectureParDeclaration) declarationDuBotCouvre(pi int, bid string, de, a uint64) bool {
	for _, b := range m.bots[pi] {
		if b.Bid() == bid && declarationCouvre(b.Declarations, de, a) {
			return true
		}
	}
	return false
}

// botDeclareApresLaCreation est la condition b de l'en-tete : aucun bot declare a la creation `t`,
// une seule declaration de l'index croise [t, a], nee apres `t`, qui couvre la vie jusqu'a `a`, et
// aucune image-cle porteuse entre `t` et sa naissance. Faux sans balayage : les images-cles ne sont
// pas connues.
func (m lectureParDeclaration) botDeclareApresLaCreation(pi int, t, a uint64) (string, bool) {
	if !m.scan.Scanned {
		return "", false
	}
	elu, croisees := "", 0
	var decl [2]uint64
	for _, b := range m.bots[pi] {
		for _, d := range b.Declarations {
			if d[0] <= a && finDeDeclaration(d) > t {
				elu, decl, croisees = b.Bid(), d, croisees+1
			}
		}
	}
	if croisees != 1 || decl[0] <= t || !declarationCouvre([][2]uint64{decl}, decl[0], a) ||
		m.imageCleEntre(t, decl[0]) {
		return "", false
	}
	return elu, true
}

// imageCleEntre dit qu'une image-cle porteuse tombe STRICTEMENT entre `de` et `a`.
func (m lectureParDeclaration) imageCleEntre(de, a uint64) bool {
	for _, k := range m.scan.KeyframesUS {
		if k > de && k < a {
			return true
		}
	}
	return false
}

// humainAbsent dit que l'humain de l'index `pi` est absent sur [de, a] : il a au moins une entite de
// cet index — une entite qu'aucune declaration d'un bot de l'index ne revendique —, aucune n'est
// instable, et la fenetre large d'aucune ne touche [de, a]. Faux sans balayage.
func (m lectureParDeclaration) humainAbsent(pi int, de, a uint64) bool {
	if !m.scan.Scanned {
		return false
	}
	siennes := 0
	for _, e := range m.scan.Entities {
		if e.Index != pi || m.entiteDUnBot(pi, e) {
			continue
		}
		if e.Unstable || fenetreLargeDe(m.scan, e).touche(de, a) {
			return false
		}
		siennes++
	}
	return siennes > 0
}

// entiteDUnBot dit qu'une declaration d'un bot de l'index `pi` croise la fenetre stricte de l'entite.
func (m lectureParDeclaration) entiteDUnBot(pi int, e grammar.PlayerEntity) bool {
	for _, b := range m.bots[pi] {
		if entiteDeclareeParLeBot(m.scan, e, b.Declarations) {
			return true
		}
	}
	return false
}

// touche dit que la fenetre large (bornes exclues, ouverte au bord du film) contient un instant de
// [de, a].
func (f fenetreLarge) touche(de, a uint64) bool {
	return (f.ouverteAvant || a > f.de) && (f.ouverteApres || de < f.a)
}

// finDeDeclaration rend la fin exclue d'une declaration `[debut, fin)`, `fin == 0` = jusqu'au bout.
func finDeDeclaration(d [2]uint64) uint64 {
	if d[1] == 0 {
		return math.MaxUint64
	}
	return d[1]
}

// declarationCouvre dit si une declaration `[debut, fin)` (`fin == 0` = declare jusqu'au bout)
// contient tout l'intervalle [de, a].
func declarationCouvre(decls [][2]uint64, de, a uint64) bool {
	for _, d := range decls {
		if d[0] <= de && a < finDeDeclaration(d) {
			return true
		}
	}
	return false
}
