//go:build research

package grammar

// mouvement_5_11_6_pied_research_test.go — LE PIED DE TRAME, ET LE CHAMP QUI Y BASCULE
// (lot 5.11.6).
//
// # CE QUE LES INSTRUMENTS PRECEDENTS ONT ETABLI, DANS L ORDRE
//
//  1. La marche CONSOMME 57 bits de plus que le paquet n en porte : son dernier record (un
//     `ti=6` a masque vide) est FABRIQUE a partir de zeros lus au-dela de la fin.
//  2. Le cadrage des paquets est sain (ecart constant de 16 octets = l en-tete).
//  3. Ce que la marche prend pour un second record est en fait LE PIED DE TRAME : trois bits de
//     `recEnd` suivis d un MOT FIXE, identique sur 5 066 paquets hors fenetre.
//
// # CE QUE CE FICHIER MESURE
//
// Le pied, mot par mot, sur TOUT le film : sa valeur, sa largeur, et les instants ou il change.
// Le film temoin ne porte qu un joueur et qu une action — un mot de pied qui ne prend une valeur
// differente QU A L INSTANT DU DECOLLAGE est le declencheur du saut, au sens exact de l oracle.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m5116PiedBits est la largeur du mot de pied relevee au dump : trois bits de `recEnd` puis
// trente-deux bits fixes. Le reste du paquet est du bourrage jusqu a l octet.
const m5116PiedBits = 35

// m5116FinRecEnd est la largeur du marqueur de fin de liste de records : un bit de prefixe a
// zero puis `R(2)` (cf. readRecordType).
const m5116FinRecEnd = 3

// m5116Pied est UNE lecture de pied de trame.
type m5116Pied struct {
	rel    float64
	chunk  int
	index  int
	debut  int
	mot    string
	bits   int
	nBiped int
}

// TestMouvement5116Pied publie le pied de trame de CHAQUE paquet et nomme ce qui y bascule.
func TestMouvement5116Pied(t *testing.T) {
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
	var pieds []m5116Pied
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
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			p := m5116Pied{rel: float64(pk.TimestampUS-t0) / 1e6, chunk: c, index: pk.Index,
				debut: debut, bits: len(pay) * 8}
			for _, rec := range recs {
				if rec.Trace.EndBit > p.bits {
					continue
				}
				if rec.Trace.EndBit > p.debut {
					p.debut = rec.Trace.EndBit
				}
				if rec.TypeIndex == BipedTypeIndex {
					p.nBiped++
				}
			}
			if p.bits-p.debut < m5116PiedBits {
				continue // paquet trop court : pas de pied complet, il est compte a part
			}
			p.mot = m511Bits(pay, p.debut+m5116FinRecEnd, m5116PiedBits-m5116FinRecEnd)
			pieds = append(pieds, p)
		}
	}
	m5116RendrePied(t, pieds)
}

// m5116RendrePied publie le recensement des mots de pied et la liste des instants ou le mot
// DOMINANT est abandonne — un mot rare est un evenement, et le film temoin n en a qu un.
func m5116RendrePied(t *testing.T, pieds []m5116Pied) {
	t.Helper()
	compte := map[string]int{}
	for _, p := range pieds {
		compte[p.mot]++
	}
	t.Logf("PIED DE TRAME : %d paquets portent un pied complet de %d bits (%d de `recEnd` + "+
		"%d bits de mot)", len(pieds), m5116PiedBits, m5116FinRecEnd,
		m5116PiedBits-m5116FinRecEnd)
	t.Logf("  %d valeur(s) distincte(s) du mot :", len(compte))
	type kv struct {
		v string
		n int
	}
	l := make([]kv, 0, len(compte))
	for v, n := range compte {
		l = append(l, kv{v: v, n: n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].v < l[j].v
	})
	for i, x := range l {
		if i >= 12 {
			t.Logf("    ... (%d valeur(s) de plus)", len(l)-12)
			break
		}
		t.Logf("    %s  x%d", x.v, x.n)
	}
	if len(l) == 0 {
		return
	}
	dominant := l[0].v
	t.Logf("MOT DOMINANT : %s (x%d sur %d, soit %.2f %%)", dominant, l[0].n, len(pieds),
		m533bPart(l[0].n, len(pieds)))
	t.Logf("INSTANTS OU LE PIED ABANDONNE LE MOT DOMINANT :")
	var rares int
	for _, p := range pieds {
		if p.mot == dominant {
			continue
		}
		rares++
		if rares > 40 {
			continue
		}
		t.Logf("  t=%8.3f s · chunk %d paquet %d · %d bipede dans le paquet · mot %s "+
			"(DIFFERENCES avec le dominant : %s)",
			p.rel, p.chunk, p.index, p.nBiped, p.mot, m5116Diff(dominant, p.mot))
	}
	t.Logf("  TOTAL : %d paquet(s) sur %d portent un mot NON dominant (%.3f %%)", rares,
		len(pieds), m533bPart(rares, len(pieds)))
}

// m5116Diff nomme les rangs de bits ou deux mots different.
func m5116Diff(a, b string) string {
	ab, bb := strings.ReplaceAll(a, " ", ""), strings.ReplaceAll(b, " ", "")
	n := len(ab)
	if len(bb) < n {
		n = len(bb)
	}
	var rangs []string
	for i := 0; i < n; i++ {
		if ab[i] != bb[i] {
			rangs = append(rangs, fmt.Sprintf("bit %d : %c -> %c", i, ab[i], bb[i]))
		}
	}
	if len(rangs) == 0 {
		return "aucune sur la longueur commune"
	}
	return strings.Join(rangs, ", ")
}
