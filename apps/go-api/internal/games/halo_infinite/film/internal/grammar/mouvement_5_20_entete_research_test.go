//go:build research

package grammar

// mouvement_5_20_entete_research_test.go — L EN-TETE REJETE, LU BIT A BIT (lot 5.20.3).
//
// Le lot 5.19 a dumpe SIX temoins de l en-tete sur lequel la vue B sort en rejet, et les six
// lisaient la meme chose : « prefixe 1 = DELTA, slot 1792 (0x700), tag 1 ». Un identifiant
// CONSTANT d un paquet a l autre n est pas un identifiant. Cet instrument le mesure sur TOUS
// les rejets du film au lieu de six, et il publie :
//
//	(a) la distribution des 48 premiers bits a la position de l en-tete rejete — si elle est
//	    concentree, ce n est pas un record mais une STRUCTURE ;
//	(b) la distribution du triplet lu (prefixe, idLow, tag) ;
//	(c) la DISTANCE en bits entre l en-tete rejete et la FIN du payload — une distance
//	    concentree dit que la position est fixe par rapport a la queue, pas au flux ;
//	(d) ce que les MEMES bits donneraient a d autres largeurs d identifiant (11 a 15), parce
//	    que la largeur du jeu est `FUN_1406d310c(DAT_144706100)` et non un litteral.
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestEntete520$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"testing"
)

// e520Rejet est UN en-tete rejete, lu tel quel.
type e520Rejet struct {
	Bit      int
	Mots     [2]uint64 // les 48 premiers bits a la position de l en-tete (32 + 16)
	Prefixe  uint64
	Low      uint64
	Tag      uint64
	DepuisLa int // bits entre l en-tete et la fin du payload
}

// e520Lire rend l en-tete rejete d un paquet, ou `ok` faux si le paquet ne rejette pas.
func e520Lire(pay []byte, cfg FrameConfig, mar d519Marche) (e520Rejet, bool) {
	var r e520Rejet
	if _, ok := t519SlotRejete(pay, cfg, mar); !ok {
		return r, false
	}
	enTete := 1 + cfg.IDLowBits + 2
	if cfg.HasExtraFields {
		enTete += 32
	}
	at := mar.m.FinVueB - enTete
	if at < 0 || at+48 > len(pay)*8 {
		return r, false
	}
	r.Bit = at
	r.Mots[0] = kfReadBits(pay, at, 32)
	r.Mots[1] = kfReadBits(pay, at+32, 16)
	p := at
	if cfg.HasExtraFields {
		p += 32
	}
	r.Prefixe = kfReadBits(pay, p, 1)
	r.Low = kfReadBits(pay, p+1, cfg.IDLowBits)
	r.Tag = kfReadBits(pay, p+1+cfg.IDLowBits, 2)
	r.DepuisLa = len(pay)*8 - at
	return r, true
}

// e520Top publie les `n` cles les plus frequentes d un histogramme.
func e520Top(t *testing.T, titre string, h map[string]int, n int) {
	t.Helper()
	cles := make([]string, 0, len(h))
	for k := range h {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if h[cles[i]] != h[cles[j]] {
			return h[cles[i]] > h[cles[j]]
		}
		return cles[i] < cles[j]
	})
	total := 0
	for _, v := range h {
		total += v
	}
	t.Logf("%s : %d classes sur %d occurrences", titre, len(h), total)
	for i, k := range cles {
		if i >= n {
			break
		}
		t.Logf("   %-44s %6d (%.1f %%)", k, h[k], m533bPart(h[k], total))
	}
}

// TestEntete520 lit l en-tete rejete de TOUS les paquets fautifs du film.
func TestEntete520(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	bits, triplets, distances, largeurs := map[string]int{}, map[string]int{}, map[string]int{}, map[string]int{}
	n := 0
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				//nolint:gosec // slot, TI et Gen viennent du walker, bornes par construction
				w.BindImageCle(uint32(r.Gen), uint32(r.Slot), uint32(r.TI))
			}
		}
		LierTableDeDatums(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, tc.cfg); debut < 0 {
					continue
				}
			}
			mar := t519Marcher(pay, w, tc.cfg, debut)
			r, ok := e520Lire(pay, tc.cfg, mar)
			if !ok {
				continue
			}
			n++
			bits[fmt.Sprintf("%08x %04x", r.Mots[0], r.Mots[1])]++
			triplets[fmt.Sprintf("prefixe %d · low %5d · tag %d", r.Prefixe, r.Low, r.Tag)]++
			distances[fmt.Sprintf("%d bits avant la fin", r.DepuisLa)]++
			p := r.Bit
			if tc.cfg.HasExtraFields {
				p += 32
			}
			for _, wid := range []int{11, 12, 13, 14, 15} {
				low := kfReadBits(pay, p+1, wid)
				tag := kfReadBits(pay, p+1+wid, 2)
				largeurs[fmt.Sprintf("W=%2d low %5d tag %d", wid, low, tag)]++
			}
		}
	}
	t.Logf("REJETS LUS : %d (largeur d identifiant du cadre : %d bits, base %d)",
		n, tc.cfg.IDLowBits, tc.cfg.IDBase)
	e520Top(t, "(a) les 48 bits a la position de l en-tete", bits, 20)
	e520Top(t, "(b) le triplet lu", triplets, 20)
	e520Top(t, "(c) distance a la fin du payload", distances, 20)
	e520Top(t, "(d) les memes bits a d autres largeurs", largeurs, 25)
}
