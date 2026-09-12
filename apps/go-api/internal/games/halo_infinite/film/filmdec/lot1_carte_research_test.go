package filmdec

// lot1_carte_research_test.go — LOT 1 : LA CARTE DES COMPOSANTS DES TRAMES CADREES.
// Suite de lot1_cadrage_research_test.go (scinde pour le seuil de 500 lignes) : le
// decodage INTEGRE (un seul monde, chaque famille sous son cadrage) et le recensement
// des composants portes par les records aboutis — la carte L1-C3 du plan.

import (
	"fmt"
	"math/bits"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestLot1BalayageMondePropre — LA LEVEE DE LA RESERVE du balayage d'amorce : meme
// discriminant (masques 1..7 sur les deltas lies et aboutis), mais un monde SANS pollution
// croisee. Par chunk : monde de base = images-cles + une passe de liaisons du flux de tick
// (0xA0/0x89/0x80 a k=2, le cadrage prouve par l'oracle) ; puis pour chaque (famille, k),
// un monde RESTAURE qui ne decode QUE les paquets de cette famille, dans l'ordre.
//
// APPROXIMATION ASSUMEE, ecrite avant la mesure : le monde de base fige l'etat de FIN de la
// passe de tick du chunk (un transitoire cree au paquet 500 est deja lie quand on decode le
// paquet 100 de la famille). Le recyclage de slot intra-chunk peut donc lier un slot a un
// archetype legerement perime — un bruit qui DEGRADE le score, jamais ne l'ameliore : un
// gagnant net sous cette approximation est un gagnant a fortiori.
//
// CRITERES (ecrits avant la mesure) :
//
//	P1 — plancher n >= 30 deltas lies+aboutis ; gagnant NET si >= 15 points sur le suivant.
//	P2 — si 0xC2 (k=6) et 0xD2 (k=8) ne reproduisent PAS leur k gagnant a monde propre,
//	     le signal du balayage pollue etait un artefact : leur cadrage est RETIRE du
//	     dossier (note + Notion), pas amende.
func TestLot1BalayageMondePropre(t *testing.T) {
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
	amorces := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 16}
	cibles := map[byte]bool{0xC0: true, 0xC2: true, 0xC3: true, 0xC7: true,
		0xD2: true, 0xD3: true, 0xE5: true, 0xE9: true}
	type cell struct{ deltasLies, masquesOK int }
	mesure := map[byte]map[int]*cell{}
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
		// Passe de liaisons : le flux au cadrage prouve (bit 1 = 0), a k=2, dans wBase.
		cfg2 := DefaultFrameConfig()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg2)
			}
		}
		snap := wBase.Snapshot()
		for fam := range cibles {
			for _, k := range amorces {
				cfg := DefaultFrameConfig()
				cfg.PacketPreambleBits = k
				w := NewWorld(reg)
				w.Restore(snap)
				for _, pk := range pks {
					if pk.Type != PacketTypeDelta || pk.Size < 1 {
						continue
					}
					pay := pk.Payload(data)
					if pay[0] != fam {
						continue
					}
					br := NewBitReader(pay)
					recs, _ := DecodeFrameRecords(br, w, cfg)
					if mesure[fam] == nil {
						mesure[fam] = map[int]*cell{}
					}
					cl := mesure[fam][k]
					if cl == nil {
						cl = &cell{}
						mesure[fam][k] = cl
					}
					for i := range recs {
						r := &recs[i]
						nm := bits.OnesCount64(r.Trace.Mask)
						if r.Type == recDelta && r.DesyncAt == -1 && nm > 0 {
							cl.deltasLies++
							if nm <= 7 {
								cl.masquesOK++
							}
						}
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
			k    int
			pct  float64
			nrec int
		}
		var sc []score
		for _, k := range amorces {
			if cl := mesure[byte(o)][k]; cl != nil {
				sc = append(sc, score{k, lot1Pct(cl.masquesOK, cl.deltasLies), cl.deltasLies})
			}
		}
		sort.Slice(sc, func(i, j int) bool { return sc[i].pct > sc[j].pct })
		parts := make([]string, 0, len(sc))
		for _, s := range sc {
			parts = append(parts, fmt.Sprintf("k=%d %.1f%%(n=%d)", s.k, s.pct, s.nrec))
		}
		t.Logf("  0x%02X : %s", o, strings.Join(parts, " · "))
		var el []score
		for _, s := range sc {
			if s.nrec >= 30 {
				el = append(el, s)
			}
		}
		switch {
		case len(el) == 0:
			t.Logf("         P1 : aucune amorce a n >= 30 — NON CONCLUANT a monde propre")
		case len(el) == 1:
			t.Logf("         P1 : k=%d (%.1f %% sur %d) — SEUL au-dessus du plancher", el[0].k, el[0].pct, el[0].nrec)
		default:
			net := el[0].pct-el[1].pct >= 15
			t.Logf("         P1 : k=%d (%.1f %% sur %d, suivant k=%d %.1f %%) — %s",
				el[0].k, el[0].pct, el[0].nrec, el[1].k, el[1].pct,
				map[bool]string{true: "NET", false: "serre"}[net])
		}
	}
}

// TestLot1LargeurEnTete — LA DISTRIBUTION DES LARGEURS D'EN-TETE PAR PAQUET. Le monde
// propre montre des pelotons serres sur 0xD2/0xD3 (plusieurs k rendent de bons scores, la
// composition varie selon le film) : la signature d'un en-tete a largeur VARIABLE par
// paquet. Mesure directe : pour chaque paquet, le k qui MAXIMISE le nombre de deltas lies,
// aboutis, a masque 1..7 (ambigu si egalite) — puis l'histogramme des k par famille et les
// premiers octets par groupe de k. Recensement (sans seuil) ; base = monde enrichi par la
// passe de tick, comme TestLot1BalayageMondePropre.
func TestLot1LargeurEnTete(t *testing.T) {
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
	cibles := map[byte]bool{0xC0: true, 0xC2: true, 0xC3: true, 0xC7: true,
		0xD2: true, 0xD3: true, 0xE5: true, 0xE9: true}
	ks := map[byte]map[int]int{}       // famille -> argmax k -> paquets
	tetes := map[byte]map[string]int{} // famille -> "k=N tete=..." -> paquets
	ambigus := map[byte]int{}
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
		cfg2 := DefaultFrameConfig()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg2)
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
			meilleur, score, exaequo := -1, 0, false
			for k := 1; k <= 14; k++ {
				cfg := DefaultFrameConfig()
				cfg.PacketPreambleBits = k
				w := NewWorld(reg)
				w.Restore(snap)
				br := NewBitReader(pay)
				recs, _ := DecodeFrameRecords(br, w, cfg)
				s := 0
				for i := range recs {
					r := &recs[i]
					nm := bits.OnesCount64(r.Trace.Mask)
					if r.Type == recDelta && r.DesyncAt == -1 && nm >= 1 && nm <= 7 {
						s++
					}
				}
				switch {
				case s > score:
					meilleur, score, exaequo = k, s, false
				case s == score && s > 0 && k != meilleur:
					exaequo = true
				}
			}
			if ks[pay[0]] == nil {
				ks[pay[0]], tetes[pay[0]] = map[int]int{}, map[string]int{}
			}
			switch {
			case meilleur < 0:
				ks[pay[0]][-1]++
			case exaequo:
				ambigus[pay[0]]++
			default:
				ks[pay[0]][meilleur]++
				tete := fmt.Sprintf("k=%-2d tete=%08b", meilleur, pay[0])
				if len(pay) > 2 {
					tete += fmt.Sprintf(" %08b %08b", pay[1], pay[2])
				}
				tetes[pay[0]][tete]++
			}
		}
	}
	var octets []int
	for o := range ks {
		octets = append(octets, int(o))
	}
	sort.Ints(octets)
	for _, o := range octets {
		t.Logf("  0x%02X : argmax k (paquets) : %s · ambigus %d · sans deltas %d",
			o, lot1DistK(ks[byte(o)]), ambigus[byte(o)], ks[byte(o)][-1])
		for _, l := range lot1TopN(tetes[byte(o)], 6) {
			t.Logf("         %s", l)
		}
	}
}

// TestLot1CarteComposants — sous le cadrage ETABLI (0xC2 -> k=6, 0xD2 -> k=8, mesure sur
// 2 films), la CARTE des composants que portent les records aboutis de ces trames : c'est
// elle qui dit ou vivent l'arme, la visee et la victime. Recensement pur (L1-C3), sans seuil.
// Publie, par famille : (ti, composant) -> occurrences, sur les records DELTA aboutis.
func TestLot1CarteComposants(t *testing.T) {
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
	familleK := map[byte]int{0xC2: 6, 0xD2: 8}
	comps := map[byte]map[string]int{}
	recTIs := map[byte]map[string]int{}
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
		// UN SEUL MONDE, paquets decodes DANS L'ORDRE DU CHUNK, chaque famille sous son
		// cadrage : les transitoires vises par 0xC2/0xD2 sont crees par des records NEW a
		// l'interieur des trames de tick 0xA0 — sans les decoder aussi, leurs slots restent
		// non lies. Les familles au cadrage NON etabli sont sautees (les decoder mal
		// cadrees empoisonnerait les liaisons). C'est le prototype du decodeur integre.
		w := NewWorld(reg)
		w.Restore(snap)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			k, vise := familleK[pay[0]]
			switch {
			case vise:
				// famille cible : son cadrage etabli
			case pay[0]&0x40 == 0:
				k = DefaultPacketPreambleBits // bit 1 = 0 : cadrage historique (0xA0, 0x80, 0x89...)
			default:
				continue // bit 1 = 1 et cadrage non etabli : sauter
			}
			cfg := DefaultFrameConfig()
			cfg.PacketPreambleBits = k
			br := NewBitReader(pay)
			recs, _ := DecodeFrameRecords(br, w, cfg)
			if !vise {
				continue // les 0xA0 ne sont decodees que pour leurs liaisons
			}
			for i := range recs {
				r := &recs[i]
				if r.DesyncAt != -1 {
					continue
				}
				if recTIs[pay[0]] == nil {
					recTIs[pay[0]], comps[pay[0]] = map[string]int{}, map[string]int{}
				}
				genre := map[int]string{recNew: "new", recDel: "del", recDelta: "delta"}[r.Type]
				recTIs[pay[0]][fmt.Sprintf("%s ti=%d", genre, r.TypeIndex)]++
				for _, cp := range r.Trace.Comps {
					comps[pay[0]][fmt.Sprintf("ti=%d i%d %s", r.TypeIndex, cp.Index, cp.Name)]++
				}
			}
		}
	}
	for _, o := range []byte{0xC2, 0xD2} {
		t.Logf("  0x%02X (k=%d) — records aboutis par (genre, ti) :", o, familleK[o])
		for _, l := range lot1TopN(recTIs[o], 10) {
			t.Logf("      %s", l)
		}
		t.Logf("  0x%02X — composants presents (ti, index, nom) :", o)
		for _, l := range lot1TopN(comps[o], 25) {
			t.Logf("      %s", l)
		}
	}
}
