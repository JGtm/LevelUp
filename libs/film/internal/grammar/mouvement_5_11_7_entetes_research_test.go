//go:build research

package grammar

// mouvement_5_11_7_entetes_research_test.go — LE PIED RELU SOUS LA GRAMMAIRE PAR VUE
// (lot 5.11.7-c).
//
// # CE QUE LA GRAMMAIRE PAR VUE A CHANGE A LA QUESTION
//
// Le lot 5.11.6 avait mesure un « mot de pied » de 32 bits, constant sur 99,54 % des paquets de
// `dad793c7`, et DEVIANT SUR UN SEUL — celui du decollage du saut (bits 27 et 30). Le lot 5.11.7
// a nomme ce mot : ce ne sont pas des drapeaux, ce sont **DEUX EN-TETES DE RECORD REJETES**, un
// par vue restante, chacun `[prefixe 1][idLow][tag 2]`.
//
// Cet instrument les DECODE, au lieu de les comparer comme des chaines de bits. La question
// « que sont les bits 27 et 30 » devient alors une question de champ :
//
//	bits  0      de l en-tete de la VUE 2 : le prefixe de type
//	bits  1..13                            : `idLow`  (13 bits sur ce film)
//	bits 14..15                            : le tag de GENERATION
//	bits 16                de la VUE 3     : le prefixe
//	bits 17..29                            : `idLow`  <- LE BIT 27 EST ICI
//	bits 30..31                            : le tag   <- LE BIT 30 EST ICI
//
// Rejouable : memes variables que `TestMouvement5116Gate`.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m5117Entete est UN en-tete de record rejete, decode.
type m5117Entete struct {
	prefixe uint64
	slot    uint64
	tag     uint64
}

func (e m5117Entete) String() string {
	return fmt.Sprintf("p%d slot %d tag %d", e.prefixe, e.slot, e.tag)
}

// m5117Pied : les deux en-tetes de rejet d un paquet.
type m5117Pied struct {
	rel    float64
	vue2   m5117Entete
	vue3   m5117Entete
	nBiped int
}

// TestMouvement5117Entetes decode les deux en-tetes de rejet de chaque paquet et publie ce qui
// change, et quand.
func TestMouvement5117Entetes(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	var pieds []m5117Pied
	var t0 uint64
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if t0 == 0 {
				t0 = pk.TimestampUS
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _, curseur := DecodeFrameViewsCurseur(pay, w, cfg, 3, debut)
			p, ok := m5117Lire(pay, recs, curseur, cfg)
			if !ok {
				continue
			}
			p.rel = float64(pk.TimestampUS-t0) / 1e6
			pieds = append(pieds, p)
		}
	}
	m5117Rendre(t, pieds)
}

// m5117Lire decode les deux en-tetes de rejet a partir de la fin de la liste de records.
//
// LA DECOUPE EST CELLE DE L ECRIVAIN : la vue 1 se clot sur `recEnd` (un prefixe a zero puis
// `R(2)`, soit trois bits), puis chaque vue restante lit `1 + idLow + 2` bits et les rejette.
func m5117Lire(pay []byte, recs []FrameRecord, curseur int, cfg FrameConfig) (m5117Pied, bool) {
	var p m5117Pied
	fin := 0
	for _, r := range recs {
		if r.Trace.EndBit > fin && r.Trace.EndBit <= len(pay)*8 {
			fin = r.Trace.EndBit
		}
		if r.TypeIndex == BipedTypeIndex {
			p.nBiped++
		}
	}
	largeur := 1 + cfg.IDLowBits + 2
	debut := fin + m5116FinRecEnd
	if fin == 0 || debut+2*largeur > curseur {
		return p, false // le paquet ne porte pas DEUX en-tetes de rejet complets
	}
	br := LecteurSur(pay)
	br.Skip(debut)
	p.vue2 = m5117LireEntete(br, cfg.IDLowBits)
	p.vue3 = m5117LireEntete(br, cfg.IDLowBits)
	return p, true
}

// m5117LireEntete lit UN en-tete de record : prefixe, `idLow`, tag de generation.
func m5117LireEntete(br *Lecteur, idLow int) m5117Entete {
	var e m5117Entete
	e.prefixe = br.ReadBits(1)
	e.slot = br.ReadBits(uint(idLow)) //nolint:gosec // idLow vaut 10..15
	e.tag = br.ReadBits(2)
	return e
}

// m5117Rendre publie le recensement des deux en-tetes et les instants ou ils changent.
func m5117Rendre(t *testing.T, pieds []m5117Pied) {
	t.Helper()
	c2, c3 := map[string]int{}, map[string]int{}
	for _, p := range pieds {
		c2[p.vue2.String()]++
		c3[p.vue3.String()]++
	}
	t.Logf("PIED RELU SOUS LA GRAMMAIRE PAR VUE : %d paquets portent DEUX en-tetes de rejet "+
		"complets", len(pieds))
	m5117Top(t, "  EN-TETE DE LA VUE 2", c2)
	m5117Top(t, "  EN-TETE DE LA VUE 3", c3)
	dom2, dom3 := m5117Dominant(c2), m5117Dominant(c3)
	t.Logf("DOMINANTS : vue 2 = %s · vue 3 = %s", dom2, dom3)
	t.Logf("INSTANTS OU L UN DES DEUX EN-TETES QUITTE SON DOMINANT :")
	var rares int
	for _, p := range pieds {
		if p.vue2.String() == dom2 && p.vue3.String() == dom3 {
			continue
		}
		rares++
		if rares > 30 {
			continue
		}
		t.Logf("  t=%8.3f s · %d bipede · vue 2 : %-22s · vue 3 : %-22s", p.rel, p.nBiped,
			p.vue2.String(), p.vue3.String())
	}
	t.Logf("  TOTAL : %d paquet(s) sur %d (%.3f %%)", rares, len(pieds),
		m533bPart(rares, len(pieds)))
}

// m5117Top publie les valeurs les plus frequentes d un recensement d en-tetes.
func m5117Top(t *testing.T, titre string, h map[string]int) {
	t.Helper()
	type kv struct {
		v string
		n int
	}
	l := make([]kv, 0, len(h))
	for v, n := range h {
		l = append(l, kv{v: v, n: n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].v < l[j].v
	})
	var parts []string
	for i, x := range l {
		if i >= 8 {
			break
		}
		parts = append(parts, fmt.Sprintf("%s x%d", x.v, x.n))
	}
	t.Logf("%s (%d distincts) : %s", titre, len(h), strings.Join(parts, " · "))
}

// m5117Dominant rend la valeur la plus frequente d un recensement.
func m5117Dominant(h map[string]int) string {
	best, n := "", -1
	for v, c := range h {
		if c > n || (c == n && v < best) {
			best, n = v, c
		}
	}
	return best
}
