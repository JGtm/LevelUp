//go:build research

package grammar

// reapparition_37_bassin_film_lecteurs_test.go — LES LECTEURS DU LOT 3.7, EXTRAITS.
//
// Ils vivaient dans `reapparition_37_bassin_film_research_test.go`, qui a franchi le seuil de
// 500 lignes de `CLAUDE.md` (regle 5) au moment ou la jointure a l index de `ti=11 i0` y est
// entree. La coupe suit la responsabilite : le fichier frere porte la PASSE (corpus, marche d un
// record, publication), celui-ci porte les QUATRE GRAMMAIRES relevees au lot 3.7 et la JOINTURE.
//
// LES LARGEURS SONT CELLES DE L ECRIVAIN, et leur provenance est ecrite en tete du fichier
// frere, adresse par adresse. Aucune n est devinee ; aucune n est en production.

import (
	"fmt"
	"sort"
	"testing"
)

// reap37LireVolumes : i13 — R(13) de compte, puis un bit par volume.
func reap37LireVolumes(br *Lecteur) {
	n := br.ReadBits(reap37KillVolumeCptBits)
	for k := uint64(0); k < n; k++ {
		br.ReadBit()
	}
}

// reap37LireLetterbox : i14 au niveau 2 — R(1), R(16), quatre portes INVERSEES [+R(7)], quatre
// portes directes [+R(16)].
func reap37LireLetterbox(br *Lecteur) {
	br.ReadBit()
	br.ReadBits(reap37LetterboxQuantBits)
	for k := 0; k < reap37LetterboxSlots; k++ {
		if !br.ReadBit() { // FUN_142efd284 : le bit A UN veut dire ABSENT
			br.ReadBits(reap37LetterboxOptBits)
		}
	}
	for k := 0; k < reap37LetterboxSlots; k++ {
		if br.ReadBit() {
			br.ReadBits(reap37LetterboxQuantBits)
		}
	}
}

// reap37LireBassin : i15 — le masque des 64 fentes, puis chaque fente presente.
func reap37LireBassin(br *Lecteur) (uint64, []reap37Fente) {
	masque := br.ReadBits(reap37BassinFentes)
	var out []reap37Fente
	for k := 0; k < reap37BassinFentes; k++ {
		if masque&(uint64(1)<<uint(k)) == 0 {
			continue
		}
		f := reap37Fente{Index: k, Tag: br.ReadBits(reap37BassinTagBits)}
		if f.Tag == 0 {
			out = append(out, f)
			continue
		}
		borne := RoundTimerMax
		if f.Tag == 1 {
			borne = reap37BorneHeure
		}
		f.QA = uint16(br.ReadBits(reap37MinuteurBits)) //nolint:gosec // 16 bits lus
		f.QB = uint16(br.ReadBits(reap37MinuteurBits)) //nolint:gosec // 16 bits lus
		f.A = DequantEndpoint(uint64(f.QA), 0, borne, reap37MinuteurBits, false, true)
		f.B = DequantEndpoint(uint64(f.QB), 0, borne, reap37MinuteurBits, false, true)
		f.Queue = uint8(br.ReadBits(reap37MinuteurQueueBits)) //nolint:gosec // 5 bits lus
		if f.Tag == 1 {
			q := br.ReadBits(reap37MinuteurBits)
			f.Trois = DequantEndpoint(q, 0, borne, reap37MinuteurBits, false, true)
			f.HasTrois = true
		}
		out = append(out, f)
	}
	return masque, out
}

func reap37PublierBassin(t *testing.T, l []reap37Lecture, bloquants map[string]int) {
	t.Helper()
	fermes := 0
	for _, x := range l {
		if x.Ferme {
			fermes++
		}
	}
	t.Logf("  %d record(s) de moteur marches, %d FERMES (l oracle de justesse)", len(l), fermes)
	noms := make([]string, 0, len(bloquants))
	for n := range bloquants {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	for _, n := range noms {
		t.Logf("  BLOQUANT %-60s %d record(s)", n, bloquants[n])
	}
	if len(l) == 0 {
		return
	}
	sort.Slice(l, func(a, b int) bool { return l[a].TimestampUS < l[b].TimestampUS })
	base := l[0].TimestampUS
	for _, x := range l {
		vivantes := 0
		for _, f := range x.Fentes {
			if f.Tag != 0 {
				vivantes++
			}
		}
		etat := ""
		switch {
		case x.Queue != "":
			etat = "  [queue inconnue : " + x.Queue + "]"
		case !x.Ferme:
			etat = "  [NON FERME]"
		}
		t.Logf("  ti=%-2d t=%8.1f s  masque=%#016x  %d fente(s) declaree(s), %d vivante(s)%s",
			x.TI, float64(x.TimestampUS-base)/1e6, x.Masque, len(x.Fentes), vivantes, etat)
		for _, f := range x.Fentes {
			if f.Tag == 0 {
				t.Logf("        fente %2d : ETEINTE", f.Index)
				continue
			}
			trois := ""
			if f.HasTrois {
				trois = fmt.Sprintf("  c=%.3f s", f.Trois)
			}
			t.Logf("        fente %2d : tag=%d  a=%9.3f s  b=%9.3f s  queue=%2d  (qa=%5d qb=%5d)%s",
				f.Index, f.Tag, f.A, f.B, f.Queue, f.QA, f.QB, trois)
		}
	}
}
