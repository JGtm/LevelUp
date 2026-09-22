//go:build research

package grammar

// mouvement_5_11_6_reste_research_test.go — LES BITS REELS QUE PERSONNE NE LIT (lot 5.11.6).
//
// # CE QUE LES DEUX PREMIERS INSTRUMENTS ONT ETABLI
//
// Le cadrage des paquets est SAIN (ecart constant de 16 octets = l en-tete, 0 octet de reste en
// fin de chunk). Mais le dernier record que la marche rend sur `dad793c7` — toujours un `ti=6`
// statborg a masque vide — CONSOMME 92 BITS alors qu il n en reste que 35 a 37 : il est
// FABRIQUE a partir de zeros lus au-dela de la fin du payload.
//
// Consequence : sur 95,7 % des paquets de ce film, **35 a 37 bits REELS** sont attribues a un
// record qui n existe pas. C est la population que le lot 5.11 n a pas regardee, et c est la
// que ce fichier cherche le champ du saut.
//
// # LA DEFINITION DU RESTE, ET ELLE EST PRUDENTE
//
// `reste` = les bits entre la fin du DERNIER record ENTIEREMENT CONTENU dans le payload et la
// fin du payload. Un record qui deborde ne compte pas comme lu : ce qu il a « consomme » est en
// partie du vide.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m5116Reste est le residu reel d UN paquet.
type m5116Reste struct {
	rel      float64
	bits     int
	finLue   int
	nBits    int
	valeur   string
	nBiped   int
	nRecs    int
	nDeborde int
}

// TestMouvement5116Reste dumpe et croise les bits reels non lus, dedans contre dehors.
func TestMouvement5116Reste(t *testing.T) {
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
	a, b := m511Bornes()
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	var dedans, hors []m5116Reste
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
			r := m5116Reste{rel: float64(pk.TimestampUS-t0) / 1e6, bits: len(pay) * 8,
				finLue: debut, nRecs: len(recs)}
			for _, rec := range recs {
				if rec.Trace.EndBit > r.bits {
					r.nDeborde++
					continue
				}
				if rec.Trace.EndBit > r.finLue {
					r.finLue = rec.Trace.EndBit
				}
				if rec.TypeIndex == BipedTypeIndex {
					r.nBiped++
				}
			}
			r.nBits = r.bits - r.finLue
			r.valeur = m511Bits(pay, r.finLue, r.nBits)
			if r.rel >= a && r.rel <= b {
				dedans = append(dedans, r)
			} else {
				hors = append(hors, r)
			}
		}
	}
	m5116RendreReste(t, a, b, dedans, hors)
}

// m5116RendreReste publie le croisement dedans/dehors et le dump de la fenetre.
func m5116RendreReste(t *testing.T, a, b float64, dedans, hors []m5116Reste) {
	t.Helper()
	larD, larH := map[int]int{}, map[int]int{}
	valD, valH := map[string]int{}, map[string]int{}
	for _, r := range dedans {
		larD[r.nBits]++
		valD[r.valeur]++
	}
	for _, r := range hors {
		larH[r.nBits]++
		valH[r.valeur]++
	}
	t.Logf("RESTE REEL (bits entre la fin du dernier record CONTENU et la fin du payload)")
	t.Logf("FENETRE [%.3f ; %.3f] s : %d paquets dedans, %d dehors", a, b, len(dedans), len(hors))
	m5116Histo(t, "  largeur dedans", larD, len(dedans))
	m5116Histo(t, "  largeur hors  ", larH, len(hors))
	var seuls []string
	for v, n := range valD {
		if valH[v] == 0 {
			seuls = append(seuls, fmt.Sprintf("%q x%d", v, n))
		}
	}
	sort.Strings(seuls)
	if len(seuls) == 0 {
		t.Logf("  AUCUNE valeur de reste exclusive a la fenetre (%d distinctes dedans, %d "+
			"dehors)", len(valD), len(valH))
	} else {
		t.Logf("  %d VALEUR(S) DE RESTE EXCLUSIVE(S) A LA FENETRE :", len(seuls))
		for i, s := range seuls {
			if i >= 25 {
				t.Logf("    ... (%d de plus)", len(seuls)-25)
				break
			}
			t.Logf("    %s", s)
		}
	}
	m5116Top(t, "  restes dedans", valD, 6)
	m5116Top(t, "  restes hors  ", valH, 6)
	t.Logf("DUMP DE LA FENETRE :")
	for i, r := range dedans {
		if i >= 40 {
			t.Logf("  ... (%d de plus)", len(dedans)-40)
			break
		}
		t.Logf("  t=%8.3f s · %4d bits · %d records (%d bipede, %d debordent) · lu jusqu a "+
			"%4d · RESTE %2d bits = %s", r.rel, r.bits, r.nRecs, r.nBiped, r.nDeborde,
			r.finLue, r.nBits, r.valeur)
	}
	t.Logf("TEMOIN, 12 paquets hors fenetre :")
	pas := len(hors) / 12
	if pas < 1 {
		pas = 1
	}
	var n int
	for i := 0; i < len(hors) && n < 12; i += pas {
		n++
		r := hors[i]
		t.Logf("  t=%8.3f s · %4d bits · %d records (%d bipede, %d debordent) · lu jusqu a "+
			"%4d · RESTE %2d bits = %s", r.rel, r.bits, r.nRecs, r.nBiped, r.nDeborde,
			r.finLue, r.nBits, r.valeur)
	}
	_ = strings.TrimSpace("")
}
