package filmdec

// lot1_cadrage_research_test.go — LOT 1 DU PLAN « PERCER LA TRAME » : LES INSTRUMENTS DE
// CADRAGE (suite de lot1_familles_trame_research_test.go, scinde pour tenir le seuil de
// 500 lignes). Trois instruments, dont DEUX NEGATIFS PUBLIES :
//
//	TestLot1EnteteParPaquet      cadrage par paquet — NEGATIF : les k gagnants divergent
//	                             a en-tete identique, seule la voie par famille est fiable.
//	TestLot1PremierRecordSousK   contenu du 1er record sous le k gagnant de la famille —
//	                             la RETRACTATION du « DEL de tete » vient de lui.
//	TestLot1InferenceParFamille  balayage avec inference de chaine — NON CONCLUANT par
//	                             defaut d instrument (trames vides gagnantes, lecture
//	                             au-dela du payload), publie tel quel.
//
// Meme garde LOT1_TRAME_FILM, meme protocole (un film par process, verrou pris).

import (
	"fmt"
	"math/bits"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestLot1EnteteParPaquet — LE CADRAGE PAR PAQUET, et la correlation en-tete <-> k.
//
// Le balayage PAR FAMILLE a etabli que l'amorce varie (k=2 pour 0xA0, k=6 pour 0xC2, k=8
// pour 0xD2). Ce test cherche la STRUCTURE de l'en-tete : pour CHAQUE paquet, la plus
// petite amorce k qui rend un decodage acceptable, puis la correlation entre k et les bits
// de tete du payload.
//
// ACCEPTATION d'un k pour un paquet (criteres ecrits avant la mesure) :
//   - DecodeFrameRecords atteint le marqueur de fin proprement (pas de desync) ;
//   - au moins un record, dont au moins un traverse (DesyncAt == -1) ;
//   - la marche COUVRE le paquet : dernier bit atteint >= 50 % des bits du payload
//     (sans ce garde, une « trame vide » lue a k=2 accepterait des paquets de 20 Ko).
//     Exception : payload <= 2 octets, ou une trame vide est acceptable.
//
// PUBLIE : par famille, l'histogramme des k gagnants ; pour les familles a k variable, les
// 16 premiers bits des paquets groupes par k — c'est la que la structure de l'en-tete se lit.
func TestLot1EnteteParPaquet(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	type famAgg struct {
		ks    map[int]int    // k gagnant -> paquets
		bitsK map[string]int // "k=N tete=bbbbbbbb bbbbbbbb" -> paquets
		aucun int
	}
	fams := map[byte]*famAgg{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			f := fams[pay[0]]
			if f == nil {
				f = &famAgg{ks: map[int]int{}, bitsK: map[string]int{}}
				fams[pay[0]] = f
			}
			gagnant := -1
			for k := 1; k <= 20 && gagnant < 0; k++ {
				cfg := DefaultFrameConfig()
				cfg.PacketPreambleBits = k
				w := NewWorld(reg)
				w.Restore(snap)
				br := NewBitReader(pay)
				recs, decErr := DecodeFrameRecords(br, w, cfg)
				if decErr != nil {
					continue
				}
				if len(pay) <= 2 {
					gagnant = k // trame courte : la fermeture propre suffit
					break
				}
				walked, end := 0, 0
				for i := range recs {
					if recs[i].DesyncAt == -1 {
						walked++
					}
					if recs[i].Trace.EndBit > end {
						end = recs[i].Trace.EndBit
					}
				}
				if len(recs) >= 1 && walked >= 1 && end*2 >= len(pay)*8 {
					gagnant = k
				}
			}
			if gagnant < 0 {
				f.aucun++
				continue
			}
			f.ks[gagnant]++
			tete := fmt.Sprintf("k=%-2d tete=%08b", gagnant, pay[0])
			if len(pay) > 1 {
				tete += fmt.Sprintf(" %08b", pay[1])
			}
			f.bitsK[tete]++
		}
	}
	var octets []int
	for o := range fams {
		octets = append(octets, int(o))
	}
	sort.Ints(octets)
	for _, o := range octets {
		f := fams[byte(o)]
		total := f.aucun
		for _, v := range f.ks {
			total += v
		}
		if total < 10 {
			continue
		}
		t.Logf("  0x%02X : %5d paquets · k gagnants : %s · aucun k : %d",
			o, total, lot1DistK(f.ks), f.aucun)
		if len(f.ks) > 1 { // k variable : publier les tetes par k
			for _, l := range lot1TopN(f.bitsK, 6) {
				t.Logf("         %s", l)
			}
		}
	}
}

// TestLot1PremierRecordSousK — sous le k gagnant du balayage PAR FAMILLE (le seul signal
// robuste : masques 1..7), que contient le PREMIER record ? Si les archetypes et les masques
// sont coherents (entites liees, composants plausibles), le cadrage de la famille est LANDE
// et la structure se nomme. Recensement pur, sans seuil. Familles et k issus de
// TestLot1AmorceParFamille : 0xA0->2 (temoin), 0xC2->6, 0xD2->8, 0xD3->6 (faible, publie
// pour observation), 0xC0->5 (hypothese « vue 1 vide de 3 bits apres l'amorce », cf. la
// mesure multi-vues).
func TestLot1PremierRecordSousK(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	familleK := map[byte]int{0xA0: 2, 0xC2: 6, 0xD2: 8, 0xD3: 6, 0xC0: 5}
	type agg struct {
		packets int
		r0      map[string]int // description du 1er record -> paquets
		r1      map[string]int // description du 2e record -> paquets
	}
	fams := map[byte]*agg{}
	desc := func(r *FrameRecord, w *World) string {
		switch {
		case r.Type == recEnd:
			return "end"
		case r.Type == recDel:
			return fmt.Sprintf("del slot~%d", r.Slot/64*64) // bande de 64 pour grouper
		case r.Type == recNew:
			return fmt.Sprintf("new ti=%d ok=%v", r.TypeIndex, r.DesyncAt == -1)
		default:
			_, bound := w.ArchetypeForSlot(r.Slot)
			return fmt.Sprintf("delta ti=%d lie=%v ok=%v masque=%d",
				r.TypeIndex, bound, r.DesyncAt == -1, bits.OnesCount64(r.Trace.Mask))
		}
	}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			k, vise := familleK[pay[0]]
			if !vise {
				continue
			}
			f := fams[pay[0]]
			if f == nil {
				f = &agg{r0: map[string]int{}, r1: map[string]int{}}
				fams[pay[0]] = f
			}
			f.packets++
			cfg := DefaultFrameConfig()
			cfg.PacketPreambleBits = k
			w := NewWorld(reg)
			w.Restore(snap)
			br := NewBitReader(pay)
			recs, _ := DecodeFrameRecords(br, w, cfg)
			if len(recs) == 0 {
				f.r0["trame-vide"]++
				continue
			}
			f.r0[desc(&recs[0], w)]++
			if len(recs) > 1 {
				f.r1[desc(&recs[1], w)]++
			}
		}
	}
	var octets []int
	for o := range fams {
		octets = append(octets, int(o))
	}
	sort.Ints(octets)
	for _, o := range octets {
		f := fams[byte(o)]
		t.Logf("  0x%02X (k=%d) : %d paquets", o, familleK[byte(o)], f.packets)
		for _, l := range lot1TopN(f.r0, 8) {
			t.Logf("      1er record : %s", l)
		}
		for _, l := range lot1TopN(f.r1, 5) {
			t.Logf("      2e  record : %s", l)
		}
	}
}

// TestLot1InferenceParFamille — le balayage d'amorce REJOUE AVEC L'INFERENCE DE CHAINE,
// qui voit a travers les transitoires (le premier record de 0xC2/0xD2/0xD3 est un delta sur
// un slot NON LIE : la marche sequentielle meurt dessus, l'inference essaie les 50
// archetypes et ne retient que les non-ambigus confirmes par la suite de la chaine).
//
// CRITERES (ecrits avant mesure) :
//
//	I1 — le k gagnant par famille (part de fins de chaine atteintes SANS depassement du
//	     payload) doit CONFIRMER celui du balayage sans inference (0xC2->6, 0xD2->8) ;
//	     divergence = publiee, c'est le balayage avec inference qui fait foi (il decode
//	     davantage, donc contraint plus).
//	I2 — recensement des archetypes INFERES des transitoires de tete (0xD2/0xD3) au k
//	     gagnant : c'est l'identite de ce qui meurt/charge la trame. Sans seuil.
func TestLot1InferenceParFamille(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	prev := inferChain
	SetInferChain(true)
	defer SetInferChain(prev)

	amorces := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12}
	cibles := map[byte]bool{0xA0: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xC7: true,
		0xD2: true, 0xD3: true, 0xE5: true, 0xE9: true}
	type cell struct {
		packets, finPropre, records, walked, inferes, deborde int
	}
	mesure := map[byte]map[int]*cell{}
	tisInferes := map[byte]map[string]int{} // famille -> "ti=N" du 1er record infere (k gagnant provisoire)
	kProvisoire := map[byte]int{0xC2: 6, 0xD2: 8, 0xD3: 6, 0xC0: 5, 0xC3: 6, 0xC7: 5, 0xE5: 8, 0xE9: 5, 0xA0: 2}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		wBase := NewWorld(reg)
		pks := WalkPackets(data)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		snap := wBase.Snapshot()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			if !cibles[pay[0]] {
				continue
			}
			if pay[0] == 0xA0 && pk.Index%20 != 0 {
				continue // temoin : 1 paquet sur 20 suffit, l'inference est chere
			}
			frameLen := len(pay) * 8
			for _, k := range amorces {
				cfg := DefaultFrameConfig()
				cfg.PacketPreambleBits = k
				if mesure[pay[0]] == nil {
					mesure[pay[0]] = map[int]*cell{}
				}
				cl := mesure[pay[0]][k]
				if cl == nil {
					cl = &cell{}
					mesure[pay[0]][k] = cl
				}
				cl.packets++
				w := NewWorld(reg)
				w.Restore(snap)
				br := NewBitReader(pay)
				br.Skip(k)
				recs, inferred, hitEnd := decodeInferLoop(br, pay, w, cfg)
				deb := br.BitPos() > frameLen
				if deb {
					cl.deborde++
				}
				if hitEnd && !deb {
					cl.finPropre++
				}
				cl.records += len(recs)
				cl.inferes += inferred
				for i := range recs {
					if recs[i].DesyncAt == -1 {
						cl.walked++
					}
				}
				if k == kProvisoire[pay[0]] && len(recs) > 0 && !deb {
					r0 := recs[0]
					if r0.Type == recDelta && r0.DesyncAt == -1 {
						if tisInferes[pay[0]] == nil {
							tisInferes[pay[0]] = map[string]int{}
						}
						tisInferes[pay[0]][fmt.Sprintf("ti=%d", r0.TypeIndex)]++
					}
				}
			}
		}
	}
	var octets []int
	for o := range mesure {
		octets = append(octets, int(o))
	}
	sort.Ints(octets)
	for _, o := range octets {
		type score struct {
			k       int
			fin     float64
			walked  int
			inferes int
		}
		var sc []score
		for _, k := range amorces {
			cl := mesure[byte(o)][k]
			if cl == nil || cl.packets == 0 {
				continue
			}
			sc = append(sc, score{k, lot1Pct(cl.finPropre, cl.packets), cl.walked, cl.inferes})
		}
		sort.Slice(sc, func(i, j int) bool { return sc[i].fin > sc[j].fin })
		parts := make([]string, 0, len(sc))
		for _, s := range sc {
			parts = append(parts, fmt.Sprintf("k=%d fin=%.0f%%(w=%d,i=%d)", s.k, s.fin, s.walked, s.inferes))
		}
		t.Logf("  0x%02X : %s", o, strings.Join(parts, " · "))
		if len(sc) > 0 && sc[0].fin > 0 {
			t.Logf("         I1 : k gagnant (fin de chaine propre) = %d (%.0f %%)", sc[0].k, sc[0].fin)
		}
		if m := tisInferes[byte(o)]; len(m) > 0 {
			t.Logf("         I2 : 1er record propre au k provisoire (%d) : %s",
				kProvisoire[byte(o)], strings.Join(lot1TopN(m, 8), " · "))
		}
	}
}

func lot1DistK(m map[int]int) string {
	var ks []int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	parts := make([]string, 0, len(ks))
	for _, k := range ks {
		parts = append(parts, fmt.Sprintf("k=%d(x%d)", k, m[k]))
	}
	return strings.Join(parts, " ")
}

// lot1TopN rend les n entrees les plus frequentes, formatees « cle xCOMPTE ».
func lot1TopN(m map[string]int, n int) []string {
	type kv struct {
		k string
		v int
	}
	var s []kv
	for key, v := range m {
		s = append(s, kv{key, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	if len(s) > n {
		s = s[:n]
	}
	out := make([]string, 0, len(s))
	for _, e := range s {
		out = append(out, fmt.Sprintf("%s x%d", e.k, e.v))
	}
	return out
}
