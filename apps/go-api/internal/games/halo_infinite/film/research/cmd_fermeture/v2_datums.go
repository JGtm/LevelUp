//go:build research

package main

// v2_datums.go — LE SLOT REJETE CONTRE LE BLOC DE TYPE 1 (item 1.2 de la carte v2).
//
// Chaque chunk porte, juste avant son image-cle, un bloc de type 1 : la table de datums du jeu a
// cet instant (`grammar.LireBlocDeDatums`). Pour chaque paquet dont la vue B sort sur un
// en-tete REJETE, la carte demande au bloc du MEME chunk ce qu il sait du slot — la mesure de
// `TestBloc521Rejets` (un seul film), etendue au corpus et aux deux sorties — et
// au bloc du chunk SUIVANT si l eid y est ne entre-temps :
//
//	etat au bloc du chunk     vivant / trace (generation ou drapeau poses) / vide / absent
//	                          (slot au-dela de la table) / sans bloc (chunk sans bloc lisible)
//	naissance                 « NEW lu dans le chunk » : un record NEW de ce slot a ete lu et
//	                          traverse dans ce chunk, avant ce paquet ou dans lui ;
//	                          « NEW lu desynchronise · <classe> » : un NEW de ce slot a ete lu dans
//	                          ce chunk mais sa traversee a desynchronise (le slot n est pas lie) ;
//	                          la classe que les blocs donnent suit ;
//	                          « naissance non lue » : le bloc SUIVANT porte une allocation de ce
//	                          slot sous la generation de l eid (vivante, ou deja liberee), que le
//	                          bloc du chunk ne portait pas, et aucun NEW n a ete lu ;
//	                          « naissance non lue, generation 0 » : le bloc suivant porte la
//	                          generation 0 sans drapeau la ou le bloc du chunk portait une autre
//	                          generation. `FUN_142f2e598` pose `gen = (gen + 1) & 3` a chaque
//	                          allocation : le slot a ete alloue sous la generation 0, puis libere ;
//	                          « vivant au bloc du chunk » : deja vivant sous cette generation au
//	                          debut du chunk ;
//	                          « libere avant le chunk (meme generation) » : l entree du chunk porte
//	                          cette generation, deja liberee — un delta d une entite morte ;
//	                          « realloue sous une autre generation » : le slot a change entre les
//	                          deux blocs, mais pas pour cet eid ;
//	                          « aucune allocation » : les deux blocs ne montrent rien pour cet eid ;
//	                          « non mesurable » : pas de bloc lisible au chunk suivant.
//
// LIMITE ECRITE : sur un slot JAMAIS alloue au bloc du chunk (generation 0, aucun drapeau), une
// entite de generation 0 nee ET morte avant le bloc suivant laisse la meme entree. Elle tombe en
// « aucune allocation ».

import (
	"errors"
	"io"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// masqueDeSlot extrait le slot d un eid ; decalageDeGeneration, sa tete de generation.
const (
	masqueDeSlot         = 0x3fffffff
	decalageDeGeneration = 30
)

// blocDuChunk est la table de datums d un chunk ; `present` faux : aucun bloc lisible.
type blocDuChunk struct {
	present bool
	entrees []grammar.DatumEntry
}

// entree rend l entree d un slot, et dit si le bloc la porte.
func (b blocDuChunk) entree(slot uint32) (grammar.DatumEntry, bool) {
	if !b.present || int(slot) >= len(b.entrees) {
		return grammar.DatumEntry{}, false
	}
	return b.entrees[slot], true
}

// collecteurV2 recoit le detail de chaque paquet, dans l ordre du film, et le range.
type collecteurV2 struct {
	fc      *grammar.FilmContext
	m       *mesureV2
	suivant map[int]int
	chunk   int
	neufs   map[uint32]bool
	// neufsDesync : les slots d un NEW lu dans le chunk dont la traversee a desynchronise.
	neufsDesync map[uint32]bool
	blocs       map[int]blocDuChunk
	// id et paquets : le film et `fermeture_paquets.tsv` (nil hors de `-paquets`) ; err : la
	// premiere erreur d ecriture de ce fichier.
	id      string
	paquets io.Writer
	err     error
}

// nouveauCollecteurV2 ouvre la collecte d un film.
func nouveauCollecteurV2(fc *grammar.FilmContext, id string, paquets io.Writer) *collecteurV2 {
	nums := fc.ChunkNumbers()
	suivant := make(map[int]int, len(nums))
	for i := 0; i+1 < len(nums); i++ {
		suivant[nums[i]] = nums[i+1]
	}
	return &collecteurV2{fc: fc, m: nouvelleMesureV2(), suivant: suivant, chunk: -1,
		neufs: map[uint32]bool{}, neufsDesync: map[uint32]bool{}, blocs: map[int]blocDuChunk{}, id: id, paquets: paquets}
}

// voir est le rappel de `grammar.FrameClosureDetaillee`.
func (c *collecteurV2) voir(p grammar.PaquetDeCarte) {
	c.changerDeChunk(p.Chunk)
	for _, s := range p.NeufsLus {
		c.neufs[s] = true
	}
	for _, s := range p.NeufsDesynchronises {
		c.neufsDesync[s] = true
	}
	c.m.compterPaquet(p)
	if c.paquets != nil && c.err == nil {
		c.err = ecrirePaquet(c.paquets, c.id, p)
	}
	if !p.Sortie.EstUnRejet() {
		return
	}
	slot := p.EIDRejete & masqueDeSlot
	k := cleRejet{sortie: p.Sortie.String(), paquet: classeDePaquet(p),
		etat: etatAuBloc(c.bloc(c.chunk), slot), naissance: c.naissance(p.EIDRejete)}
	ajouter(c.m.rejets, k, p.UtilesEnJeu)
}

// changerDeChunk oublie les NEW lus et les blocs des chunks passes.
func (c *collecteurV2) changerDeChunk(n int) {
	if n == c.chunk {
		return
	}
	c.chunk = n
	c.neufs = map[uint32]bool{}
	c.neufsDesync = map[uint32]bool{}
	for k := range c.blocs {
		if k != n && k != c.suivant[n] {
			delete(c.blocs, k)
		}
	}
}

// bloc rend le bloc de type 1 du chunk `n`, lu une fois.
func (c *collecteurV2) bloc(n int) blocDuChunk {
	if b, ok := c.blocs[n]; ok {
		return b
	}
	b, err := lireBlocDuChunk(c.fc, n)
	switch {
	case errors.Is(err, errSansBloc):
		c.m.sansBloc++
	case err != nil:
		c.m.blocsIllisibles++
	}
	c.blocs[n] = b
	return b
}

// errSansBloc : le chunk ne porte aucun paquet de type 1.
var errSansBloc = errors.New("aucun bloc de type 1")

// lireBlocDuChunk lit le DERNIER bloc de type 1 d un chunk (la regle de `TestBloc521Rejets`).
func lireBlocDuChunk(fc *grammar.FilmContext, n int) (blocDuChunk, error) {
	data, pks, ok := fc.ChunkAt(n)
	if !ok {
		return blocDuChunk{}, errSansBloc
	}
	out, err := blocDuChunk{}, errSansBloc
	for _, pk := range pks {
		if pk.Type != grammar.PacketTypeDatums {
			continue
		}
		b, e := grammar.LireBlocDeDatums(pk.Payload(data))
		if e != nil {
			err = e
			continue
		}
		out, err = blocDuChunk{present: true, entrees: b.Entrees}, nil
	}
	return out, err
}

// etatAuBloc classe le slot dans un bloc : la classification de `TestBloc521Rejets`.
func etatAuBloc(b blocDuChunk, slot uint32) string {
	if !b.present {
		return "sans bloc"
	}
	e, ok := b.entree(slot)
	switch {
	case !ok:
		return "absent"
	case e.Vivante():
		return "vivant"
	case e.Gen != 0 || e.Drapeaux != 0:
		return "trace"
	}
	return "vide"
}

// naissance dit si l eid rejete est ne dans le chunk sans que sa naissance soit lue (cf. l en-tete).
func (c *collecteurV2) naissance(eid uint32) string {
	slot, gen := eid&masqueDeSlot, uint8(eid>>decalageDeGeneration) //nolint:gosec // deux bits
	if c.neufs[slot] {
		return "NEW lu dans le chunk"
	}
	parBlocs := c.naissanceParBlocs(slot, gen)
	if c.neufsDesync[slot] {
		return "NEW lu desynchronise · " + parBlocs
	}
	return parBlocs
}

// naissanceParBlocs classe un eid d apres le bloc du chunk et celui du chunk suivant.
func (c *collecteurV2) naissanceParBlocs(slot uint32, gen uint8) string {
	n, ok := c.suivant[c.chunk]
	if !ok {
		return "non mesurable"
	}
	apres := c.bloc(n)
	if !apres.present {
		return "non mesurable"
	}
	cur, _ := c.bloc(c.chunk).entree(slot)
	nx, _ := apres.entree(slot)
	nxAlloue, curAlloue := alloueSous(nx, gen), alloueSous(cur, gen)
	switch {
	case nxAlloue && !curAlloue:
		return "naissance non lue"
	case gen == 0 && nx.Gen == 0 && nx.Drapeaux == 0 && cur.Gen != 0:
		return "naissance non lue, generation 0"
	case curAlloue && cur.Vivante():
		return "vivant au bloc du chunk"
	case curAlloue:
		return "libere avant le chunk (meme generation)"
	case nx.Gen != cur.Gen || nx.Drapeaux != cur.Drapeaux:
		return "realloue sous une autre generation"
	}
	return "aucune allocation"
}

// alloueSous dit si une entree porte une allocation (vivante ou liberee) sous la generation `gen`.
// Une entree `generation 0, drapeaux 0` est celle d un slot jamais alloue (cf. l en-tete).
func alloueSous(e grammar.DatumEntry, gen uint8) bool {
	return e.Gen == gen && (e.Gen != 0 || e.Drapeaux != 0)
}
