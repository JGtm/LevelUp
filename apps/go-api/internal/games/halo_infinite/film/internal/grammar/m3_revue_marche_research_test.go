//go:build research

package grammar

// m3_revue_marche_research_test.go — LOT M3.1, REPRISE APRES REVUE ADVERSE (2026-09-24) :
// ATTRIBUTION des ancres que la marche d image-cle du lot change, payload par payload, contre la
// marche de la base fe7079f41. La revue (constat F1) a vu des vehicules disparaitre du document
// sur trois temoins du corpus ; cet instrument dit QUELLE decision du balayeur (recalage,
// glissement, borne de fenetre, traversee d une frontiere de fenetre par la fin de table) prend ou
// perd QUELLE ancre, et de quel archetype.
//
//	M3_REVUE_FILM=<dossier des chunks> go test -tags=research -count=1 -v -timeout 30m \
//	  -run '^TestM3RevueMarche$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/weaponv3"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// m3rvRegle nomme une variante du balayeur par ses quatre interrupteurs.
type m3rvRegle struct {
	nom         string
	recalage    bool // l en-tete exact d un bipede precede l ancre elue -> il est pris
	glissement  bool // une fenetre sans candidat glisse au lieu d arreter la marche
	borneLarge  bool // `q < end && q+64 <= total` (lot) au lieu de `q+64 <= end` (base)
	trainePorte bool // la traine de sentinelles se reporte d une fenetre a la suivante
}

// m3rvScan est kfScanNext sous une regle ; `streak` porte la traine de sentinelles quand la
// regle la reporte.
func m3rvScan(buf []byte, from, prevSlot, total, maxWin int, r m3rvRegle, streak *int) (at int, fin bool) {
	at = -1
	best := kfCand{consecutive: -1, gen: 1 << 30, slot: 1 << 30, bit: 1 << 30}
	exact := -1
	end := from + maxWin
	if end > total {
		end = total
	}
	s0 := 0
	if r.trainePorte {
		s0 = *streak
	}
	sentStreak := s0
	for q := from; (r.borneLarge && q < end && q+64 <= total) || (!r.borneLarge && q+64 <= end); q++ {
		id := kfReadBits(buf, q, 32)
		if id == kfSent {
			if sentStreak++; sentStreak >= 2048 {
				fin = true
				break
			}
			continue
		}
		sentStreak = 0
		s, ti, g, ok := kfAnchorFromID(buf, q, id, prevSlot, total)
		if !ok {
			continue
		}
		if s == prevSlot+1 && g == 1 {
			return q, false
		}
		if r.recalage && exact < 0 && g == 1 && ti == BipedTypeIndex {
			exact = q
		}
		cand := kfCand{gen: g, slot: s, bit: q}
		if s == prevSlot+1 {
			cand.consecutive = 1
		}
		if at < 0 || cand.betterThan(best) {
			at, best = q, cand
		}
	}
	*streak = sentStreak
	if exact >= 0 && (at < 0 || exact <= at) {
		return exact, fin
	}
	return at, fin
}

func m3rvScanGlissant(buf []byte, from, prevSlot, total, maxWin int, r m3rvRegle) int {
	streak := 0
	for f := from; f+64 <= total; f += maxWin {
		at, fin := m3rvScan(buf, f, prevSlot, total, maxWin, r, &streak)
		if at >= 0 || fin || !r.glissement {
			return at
		}
	}
	return -1
}

// m3rvWalk est walkKeyframeWorldStats sous une regle.
func m3rvWalk(buf []byte, r m3rvRegle) []KeyframeRec {
	return m3Walk(buf, kfScanFenetreBits, func(b []byte, from, prev, total, maxWin int) int {
		return m3rvScanGlissant(b, from, prev, total, maxWin, r)
	})
}

type m3rvCle struct{ slot, ti, gen int }

func m3rvEnsemble(recs []KeyframeRec) map[m3rvCle]int {
	out := map[m3rvCle]int{}
	for _, r := range recs {
		out[m3rvCle{r.Slot, r.TI, r.Gen}] = r.Bit
	}
	return out
}

// TestM3RevueMarche : pour chaque variante, les ancres perdues et ajoutees contre la base, par
// archetype, et le detail de chaque ancre de vehicule (ti=40) touchee.
func TestM3RevueMarche(t *testing.T) {
	dir := os.Getenv("M3_REVUE_FILM")
	if dir == "" {
		t.Skip("M3_REVUE_FILM absent")
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	tiSuivi := VehicleTypeIndex
	if v, errTI := strconv.Atoi(os.Getenv("M3_REVUE_TI")); errTI == nil {
		tiSuivi = v
	}
	base := m3rvRegle{nom: "BASE"}
	regles := []m3rvRegle{
		{nom: "LOT", recalage: true, glissement: true, borneLarge: true, trainePorte: true},
		{nom: "RECAL", recalage: true},
		{nom: "GLISSE", glissement: true},
		{nom: "BORNE", borneLarge: true},
		{nom: "GLISSE+TRAINE", glissement: true, trainePorte: true},
		{nom: "LOT-533fe7d91", recalage: true, glissement: true, borneLarge: true},
		{nom: "RECAL+GLISSE+TRAINE", recalage: true, glissement: true, trainePorte: true},
	}
	type bilan struct{ perdues, ajoutees map[int]int }
	bilans := map[string]*bilan{}
	for _, r := range regles {
		bilans[r.nom] = &bilan{perdues: map[int]int{}, ajoutees: map[int]int{}}
	}
	// cohérence : prod LOT == walkKeyframeWorldStats
	for _, ch := range FilmChunkNumbers(film) {
		data, pks, ok := FilmChunkAt(film, ch)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			pay := pk.Payload(data)
			ref := m3rvEnsemble(m3rvWalk(pay, base))
			prod, _ := walkKeyframeWorldStats(pay, kfScanFenetreBits)
			for _, r := range regles {
				recs := m3rvWalk(pay, r)
				if r.nom == "LOT" && len(recs) != len(prod) {
					t.Errorf("ch%d : instrument LOT %d ancres, production %d", ch, len(recs), len(prod))
				}
				apres := m3rvEnsemble(recs)
				b := bilans[r.nom]
				for k, bit := range ref {
					if _, ok := apres[k]; !ok {
						b.perdues[k.ti]++
						if k.ti == tiSuivi && r.nom == "LOT" {
							t.Logf("   %s ch%d t=%.1fs PERDUE slot %d ti %d gen %d @%d (payload %d bits)",
								r.nom, ch, float64(pk.TimestampUS)/1e6, k.slot, k.ti, k.gen, bit, len(pay)*8)
						}
					}
				}
				for k, bit := range apres {
					if _, ok := ref[k]; !ok {
						b.ajoutees[k.ti]++
						if k.ti == tiSuivi && r.nom == "LOT" {
							t.Logf("   %s ch%d t=%.1fs AJOUTEE slot %d ti %d gen %d @%d — %s", r.nom, ch,
								float64(pk.TimestampUS)/1e6, k.slot, k.ti, k.gen, bit, m3rvVoisinage(recs, bit))
						}
					}
				}
			}
		}
	}
	for _, r := range regles {
		b := bilans[r.nom]
		t.Logf("== %-20s perdues %s | ajoutees %s", r.nom, m3rvParTI(b.perdues), m3rvParTI(b.ajoutees))
	}
}

func m3rvParTI(m map[int]int) string {
	tis := make([]int, 0, len(m))
	tot := 0
	for ti, n := range m {
		tis = append(tis, ti)
		tot += n
	}
	sort.Ints(tis)
	s := fmt.Sprintf("%d [", tot)
	for _, ti := range tis {
		s += fmt.Sprintf(" ti%d:%d", ti, m[ti])
	}
	return s + " ]"
}

// m3rvVoisinage decrit l ancre qui precede et celle qui suit la position `bit` dans une marche.
func m3rvVoisinage(recs []KeyframeRec, bit int) string {
	for i, r := range recs {
		if r.Bit != bit {
			continue
		}
		s := "premiere"
		if i > 0 {
			p := recs[i-1]
			s = fmt.Sprintf("apres slot %d ti %d gen %d @%d (+%d)", p.Slot, p.TI, p.Gen, p.Bit, bit-p.Bit)
		}
		if i+1 < len(recs) {
			n := recs[i+1]
			s += fmt.Sprintf(", avant slot %d ti %d gen %d @%d (+%d)", n.Slot, n.TI, n.Gen, n.Bit, n.Bit-bit)
		} else {
			s += ", derniere"
		}
		return s
	}
	return "?"
}

// TestM3RevueNaissancesDuSlot : les dotations de naissance que la PRODUCTION rend pour UN slot
// (temoin du plan : 81c02726, slot 534, arme des 1:59.5), avec le bilan des refus du film.
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> M3_REVUE_SLOT=<slot> \
//	  go test -tags=research -count=1 -v -run '^TestM3RevueNaissancesDuSlot$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/
func TestM3RevueNaissancesDuSlot(t *testing.T) {
	slot, err := strconv.Atoi(os.Getenv("M3_REVUE_SLOT"))
	if err != nil {
		t.Skip("M3_REVUE_SLOT absent")
	}
	tc := t516Cadre(t)
	cres, _, err := ScanBipedCreations(tc.fc)
	if err != nil {
		t.Fatalf("creations : %v", err)
	}
	births, st, err := ScanBirthLoadouts(tc.fc, cres)
	if err != nil {
		t.Fatalf("naissances : %v", err)
	}
	t.Logf("== bilan : %+v", st)
	var t0 uint64
	if chs := tc.fc.ChunkNumbers(); len(chs) > 0 {
		if _, pks, ok := tc.fc.ChunkAt(chs[0]); ok && len(pks) > 0 {
			t0 = pks[0].TimestampUS
		}
	}
	for _, c := range cres {
		if int(c.Slot) == slot {
			t.Logf("   creation slot %d gen %d a %.1f s (film)", c.Slot, c.Generation, float64(c.TimestampUS-t0)/1e6)
		}
	}
	for _, b := range births {
		if int(b.Slot) != slot {
			continue
		}
		s := ""
		for _, w := range b.Weapons {
			s += fmt.Sprintf(" [k%d %08X %s]", w.Emplacement, w.Family, weaponv3.WeaponName(w.Family))
		}
		t.Logf("   DOTATION slot %d gen %d a %.1f s (film) :%s", b.Slot, b.Generation,
			float64(b.TimestampUS-t0)/1e6, s)
	}
}
