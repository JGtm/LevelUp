package killsource

// index_motif.go — LE TROISIEME LIEN DIRECT `index de joueur <-> joueur`, ET LE SEUL QUI VOIE
// LES REMPLACANTS (lot 5.2b.1, 2026-09-20).
//
// # LE DEFAUT QUE CE FICHIER FERME, MESURE SUR `b1ad85eb`
//
// La table de `chunk_00` (film_table.go) est ecrite A L OUVERTURE du film : elle porte les huit
// sieges du depart. Quand un joueur QUITTE et qu un autre le REMPLACE en cours de match, le
// remplacant n y est pas — et le roster du decodeur reste borne a ce que la table dit. Sur
// `b1ad85eb` : table = 8 sieges occupes (index 0..7), plus un bot a l index 8 par BOT_METADATA,
// donc `nPlay = 9`. Or le kill-feed NOMME un neuvieme humain, `Claudors`, que la table ignore, et
// que la bijection ne peut placer nulle part : les neuf indices sont deja epingles.
//
// CONSEQUENCE MESUREE : HUIT dead-states a tag `jpt!` valide portent l indice 10 (celui de
// `Claudors`, cf. ci-dessous), `selectCredible` les refuse tous, le compteur `OutOfRoster` vaut 8
// et l alerte dure qui en decoulait rendait `LineByLinePublishable() == false` POUR TOUT LE
// MATCH — donc `publishable = FALSE` sur les 77 lignes ecrites en base, donc un kill-feed sans
// aucune arme cote produit (Q21b filtre sur `publishable`).
//
// # LA LECTURE, ET POURQUOI C EST LA MEME QUE `PlayerIndexTable`
//
// Le xuid d un joueur figure dans les chunks de REPLICATION sur 8 octets petit-boutiste, et les
// CINQ BITS qui le precedent portent son index de joueur. C est la lecture que
// `weaponv3.ResolveXuidToPI` porte depuis le chantier `filmdec-killweapon`, celle que
// `replay.ScanPlayerIndices` publie sous le nom `PlayerIndexTable`, et celle que la ventilation
// des tirs (`sync/killcollector`) emploie deja. ELLE N EST PAS RECOPIEE ICI : ce fichier appelle
// le meme resolveur, sur les memes chunks, avec la meme exigence de concordance.
//
// « L INDEX C EST L INDEX » (decision utilisateur du 2026-09-07) : une seule table d identite par
// film, tout lien DIRECT employe a 100 %, l inference en repli NOMME et COMPTE. La table de
// `chunk_00` et ce motif sont DEUX lectures du MEME fait — elles se completent, et quand elles se
// contredisent c est COMPTE (`MotifContradit`), jamais arbitre en silence : la table de
// `chunk_00` garde la main, elle est la plus eprouvee (314 accords sur 322 sieges, 30 films).
//
// # CE QUI EST CHERCHE, ET CE QUI NE L EST PAS
//
// Les xuids cherches viennent du FILM SEUL — les sieges de `chunk_00` et le kill-feed — jamais
// d une base. C est ce qui garde ce paquet decodable hors ligne, `cmd/killsource` compris. Un
// participant que NI la table NI le kill-feed ne nomme reste donc inconnu de ce decodeur : sur
// `b1ad85eb` c est l index 9, un joueur sans kill ni mort (mesure du 2026-09-20). Il n est
// rattrape par aucune de ces trois lectures, AUCUN dead-state ne le designe, et il n a donc
// aucun effet sur les lignes publiees.

import (
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// indexParMotif : ce que la lecture du motif de xuid rend au roster.
type indexParMotif struct {
	// nomParIndex : `index de joueur -> nom`, pour les xuids dont la lecture est UNANIME.
	nomParIndex map[int]string
	// lectures : chunks de replication qui ont livre au moins un index.
	lectures int
	// desaccords : xuids lus a DEUX index differents selon le chunk. Non publies — ce n est pas
	// un desaccord a arbitrer, c est le signe que la lecture est fausse (meme regle que
	// `replay.ScanPlayerIndices`).
	desaccords int
	// absents : xuids cherches dont le motif n apparait dans aucun chunk de replication.
	absents int
}

// lireIndexParMotif cherche, pour chaque xuid que le FILM nomme, l index de joueur ecrit devant
// son motif dans les chunks de replication.
//
// LE CHUNK 0 ET LE DERNIER SONT EXCLUS, et c est mesure par le chantier voisin : applique au
// registre ou au chunk des highlights, le resolveur rend 0 pour TOUS les xuids — le motif s y
// trouve dans un contexte qui n est pas celui d un enregistrement de joueur. Les inclure
// ecraserait la bonne lecture.
func lireIndexParMotif(f *film, slots []types.PlayerSlot, kf *killFeed) indexParMotif {
	out := indexParMotif{nomParIndex: map[int]string{}}
	nomDuXUID, xuids := xuidsNommesParLeFilm(slots, kf)
	if len(xuids) == 0 {
		return out
	}
	nums := grammar.FilmChunkNumbers(f.src)
	if len(nums) < 2 {
		return out
	}
	vus := make(map[uint64]map[int]int, len(xuids))
	for _, c := range nums[:len(nums)-1] {
		raw, _, ok := grammar.FilmChunkAt(f.src, c)
		if !ok {
			continue
		}
		lus := weaponv3.ResolveXuidToPI(xuids, raw)
		if len(lus) == 0 {
			continue
		}
		out.lectures++
		for x, pi := range lus {
			if vus[x] == nil {
				vus[x] = map[int]int{}
			}
			vus[x][pi]++
		}
	}
	for _, x := range xuids {
		switch par := vus[x]; {
		case len(par) == 0:
			out.absents++
		case len(par) > 1:
			out.desaccords++
		default:
			for pi := range par {
				out.retenir(pi, nomDuXUID[x])
			}
		}
	}
	return out
}

// retenir : un index ne peut porter qu UN nom. Deux xuids lus au meme index sont une collision
// — les deux sont laches, comme `replay.injectiveOrEmpty` lache une table non injective : un
// index partage rangerait les morts de deux joueurs sous le meme nom sans que rien ne le dise.
func (m *indexParMotif) retenir(pi int, nom string) {
	if nom == "" || pi < 0 || pi >= 32 {
		return
	}
	if deja, pris := m.nomParIndex[pi]; pris {
		if deja != nom {
			delete(m.nomParIndex, pi)
			m.desaccords++
		}
		return
	}
	m.nomParIndex[pi] = nom
}

// xuidsNommesParLeFilm : les xuids que le film nomme, avec leur nom, en ordre stable.
//
// DEUX SOURCES, ET AUCUNE BASE : les sieges de `chunk_00` (qui nomment les joueurs du depart,
// remplacants exclus) et le kill-feed (qui nomme les joueurs qui tuent ou meurent, donc les
// remplacants qui jouent). Leur union est la population que ce decodeur peut esperer placer.
func xuidsNommesParLeFilm(slots []types.PlayerSlot, kf *killFeed) (map[uint64]string, []uint64) {
	nom := make(map[uint64]string, len(slots)+len(kf.xuidDe))
	for _, s := range slots {
		if s.XUID != 0 && s.Gamertag != "" {
			nom[s.XUID] = s.Gamertag
		}
	}
	for n, x := range kf.xuidDe {
		if x != 0 && n != "" {
			nom[x] = n
		}
	}
	xuids := make([]uint64, 0, len(nom))
	for x := range nom {
		xuids = append(xuids, x)
	}
	sort.Slice(xuids, func(i, j int) bool { return xuids[i] < xuids[j] })
	return nom, xuids
}
