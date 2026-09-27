package grammar

// generations_vivantes.go — QUELLES GENERATIONS DU HANDLE D UN SLOT BIPEDE SONT VIVANTES (lot J5.2
// du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DT-8, constat GB-1 de l audit du
// 2026-09-24).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// Les positions du bipede et ses huit canaux delta n acceptaient un record que si les deux bits de
// generation de son handle valaient 1 (`ScanFilmOptions.RequireTag1`, et trois copies en dur du
// meme filtre). Or ces deux bits sont la GENERATION du handle (`readRecordID` : « the tag is
// generation ») : quand le pool de slots bipedes reboucle — un BTB long —, le slot d un joueur mort
// est rendu a un autre corps sous la generation 2, puis 3. Ces corps n avaient AUCUNE position ni
// aucune lecture delta. Mesure du lot J5.0 (`.ai/V7.5/film_re/MESURE_GB1_2026-09-27.md`) : 123 vies
// sur 379 sans une position sur `084a804d`, 330 sur 586 sur `1c4c63c2`, dont le rejeu s arretait a
// 826 s pour un film de 1 334 s.
//
// # LE PREDICAT, ET D OU IL VIENT
//
// Une generation est VIVANTE pour un slot quand une lecture de production l a vue designer un corps :
// un record de CREATION de bipede ([ScanBipedCreations], le handle de l en-tete NEW) ou un record
// `ti=35` d IMAGE-CLE (la marche de l image-cle du contexte), sur tout le film. Un record delta est
// lu quand sa generation est vivante pour son slot. Le filtre garde le pouvoir discriminant des deux
// bits — un record d une generation qu aucune lecture n a vue reste ecarte (bruit d ancrage bit a
// bit) — sans perdre les corps de generation >= 2. La mesure J5.0 le chiffre : quelques dizaines
// d en-tetes « orphelins » par film, contre des centaines de milliers de positions recuperees, et
// AUCUNE position de production dont la vie (slot, 1) serait inconnue (19 films).
//
// # LE REPLI NOMME : `repli_generation_vivante_inconnue_tag1`
//
// Un slot dont AUCUNE generation n est connue (un corps cree puis detruit entre deux images-cles,
// sans creation acceptee) retombe sur l ancien filtre : generation 1 seulement. Il est inscrit au
// registre (`facts/fallback/registre_filmdec.go`), et COMPTE par la cuisson depuis les positions
// publiees ([GenerationsVivantes.SlotsEnRepli]) ; cette couche le nomme et ne compte rien (ADR 0034
// D-4). Un ensemble nil ou vide — le coeur pur [ScanBipedRecords] quand l appelant ne fournit rien,
// le detecteur de decoupage — met TOUS les slots dans ce repli : c est bit a bit l ancien filtre.
//
// # LES OBJETS DU MONDE : TOUTES LES GENERATIONS, EXPLICITEMENT
//
// Un vehicule ou un objet du monde emploie les quatre valeurs du tag. [ToutesLesGenerations] le dit
// en toutes lettres a l appelant (`replay/build_vehicles.go`) ; il n y a plus de booleen a desarmer.

import (
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// generationDuRepli : la seule generation que le repli `repli_generation_vivante_inconnue_tag1`
// accepte sur un slot dont aucune generation n est connue — le filtre d avant le lot J5.2.
const generationDuRepli = 1

// GenerationsVivantes dit, slot par slot, quelles generations du handle designent un corps.
// Le zero n est pas utilisable : passer par [NouvellesGenerationsVivantes] ou
// [ToutesLesGenerations] ; nil est accepte partout et vaut « aucune generation connue ».
type GenerationsVivantes struct {
	// toutes : le filtre est leve (objets du monde, instruments).
	toutes bool
	// masques : indexe par le slot ; le bit g vaut 1 quand la generation g est vivante.
	masques []uint8
}

// ToutesLesGenerations rend le filtre LEVE : un record est lu quelle que soit sa generation. C est
// le reglage des objets du monde (vehicules), dont le tag emploie ses quatre valeurs.
func ToutesLesGenerations() *GenerationsVivantes { return &GenerationsVivantes{toutes: true} }

// NouvellesGenerationsVivantes construit l ensemble a partir des vies lues. Une vie hors domaine
// (slot au-dela de 13 bits, generation au-dela de 2) est ignoree : elle ne designe aucun handle.
func NouvellesGenerationsVivantes(vies []types.LifeKey) *GenerationsVivantes {
	g := &GenerationsVivantes{masques: make([]uint8, 1<<handleSlotBits)}
	for _, v := range vies {
		if v.Slot < uint32(len(g.masques)) && v.Gen < 1<<handleGenBits {
			g.masques[v.Slot] |= 1 << v.Gen
		}
	}
	return g
}

// masque rend le masque des generations vivantes d un slot, 0 quand aucune n est connue.
func (g *GenerationsVivantes) masque(slot uint32) uint8 {
	if g == nil || slot >= uint32(len(g.masques)) {
		return 0
	}
	return g.masques[slot]
}

// Accepte dit si un record de ce handle se lit. Slot sans generation connue : le repli
// `repli_generation_vivante_inconnue_tag1` (generation 1 seulement).
func (g *GenerationsVivantes) Accepte(h types.LifeKey) bool {
	if g != nil && g.toutes {
		return true
	}
	if m := g.masque(h.Slot); m != 0 {
		return m&(1<<h.Gen) != 0
	}
	return h.Gen == generationDuRepli
}

// Connue dit si au moins une generation du slot est connue (ou si le filtre est leve).
func (g *GenerationsVivantes) Connue(slot uint32) bool {
	return g != nil && (g.toutes || g.masque(slot) != 0)
}

// SlotsEnRepli compte les slots DISTINCTS des positions publiees dont aucune generation n est
// connue : ceux dont les records ont ete lus par le repli `repli_generation_vivante_inconnue_tag1`.
// C est le compte que la cuisson publie dans `coverage.fallbacks`.
func (g *GenerationsVivantes) SlotsEnRepli(pos []BipedPosition) int {
	if g != nil && g.toutes {
		return 0
	}
	vus := map[uint32]bool{}
	for _, p := range pos {
		if !g.Connue(p.Slot) {
			vus[p.Slot] = true
		}
	}
	return len(vus)
}

// memoDesVies : ce que [FilmContext] memorise des vies du bipede. Paresseux, comme les autres
// derivations du contexte (cf. film_context.go) : calcule a la premiere demande.
type memoDesVies struct {
	creations     []BipedCreation
	stats         types.BipedCreationStats
	err           error
	creationsLues bool
	vivantes      *GenerationsVivantes
}

// CreationsDeBipede rend les records de creation de bipede du film ([ScanBipedCreations]), lus UNE
// fois. La tranche rendue est une COPIE : un appelant qui la trie ne touche pas celle du contexte.
func (c *FilmContext) CreationsDeBipede() ([]BipedCreation, types.BipedCreationStats, error) {
	if c == nil {
		return ScanBipedCreations(nil)
	}
	if !c.vies.creationsLues {
		c.vies.creations, c.vies.stats, c.vies.err = ScanBipedCreations(c)
		c.vies.creationsLues = true
	}
	return slices.Clone(c.vies.creations), c.vies.stats, c.vies.err
}

// GenerationsVivantes rend les generations vivantes du handle bipede de ce film, relevees une fois
// sur les records de creation ([FilmContext.CreationsDeBipede]) et les records `ti=35` de TOUTES
// les images-cles. Jamais nil.
func (c *FilmContext) GenerationsVivantes() *GenerationsVivantes {
	if c == nil {
		return NouvellesGenerationsVivantes(nil)
	}
	if c.vies.vivantes == nil {
		c.vies.vivantes = NouvellesGenerationsVivantes(viesConnuesDuFilm(c))
	}
	return c.vies.vivantes
}

// viesConnuesDuFilm rend les vies (slot, generation) que les deux lectures de production designent.
// Une creation illisible n est pas une erreur ici : les images-cles restent, et un slot que rien ne
// designe retombe sur le repli nomme.
func viesConnuesDuFilm(c *FilmContext) []types.LifeKey {
	var out []types.LifeKey
	if cre, _, err := c.CreationsDeBipede(); err == nil {
		for _, x := range cre {
			out = append(out, types.LifeKey{Slot: x.Slot, Gen: x.Generation})
		}
	}
	marche := c.MarcheDImageCle()
	for _, n := range c.ChunkNumbers() {
		data, pks, ok := c.ChunkAt(n)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range marche.Records(pk.Payload(data)) {
				if r.TI == BipedTypeIndex && r.Slot >= 0 && r.Gen >= 0 {
					out = append(out, types.LifeKey{Slot: uint32(r.Slot), Gen: uint32(r.Gen)}) //nolint:gosec // bornes verifiees
				}
			}
		}
	}
	return out
}
