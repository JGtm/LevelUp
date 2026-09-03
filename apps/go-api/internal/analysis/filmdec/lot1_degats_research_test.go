package filmdec

// lot1_degats_research_test.go — LOT 1 : damage_aftermath (type 0, octet 0xC0, 872k), LE VRAI
// enregistrement de degat — source, magnitude, VICTIME. Grammaire du workflow
// damage-aftermath-reader (31/08, decompilation parallele + verification adverse + synthese),
// domaines des 3 references d'en-tete lus dans l'exe (descripteur 0x144724f80, vtable+0x58 :
// ref0 domaine 1, ref1 domaine 1, ref2 domaine 7).
//
// LE JUGE EST DISCRIMINANT, ET C'EST LE POINT : contrairement au vecteur de visee (non
// discriminant a 30 bits), on decode l'evenement EN ENTIER puis la TRAME de records, et on
// compte les masques 1..7 des deltas lies aboutis. Bon cadrage -> ~99 % (le discriminant
// eprouve du depot) ; un seul bit faux -> la trame desynchronise, le taux s'effondre. Un
// TEMOIN a offset faux mesure le plancher sur le flux reel.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule.

import (
	"os"
	"testing"
)

// Les decodeurs lot1RefDom, lot1RefDom1, lot1DmgResult, lot1Dequant et lot1DecodeDamageAftermath
// ont ete PRODUCTIONISES (Lot 2 du plan precision) : voir weapon_hits_decode.go. Cet instrument
// les APPELLE tels quels — une seule copie.

// TestLot1Degats decode damage_aftermath EN ENTIER et JUGE le cadrage par l'oracle de trame.
func TestLot1Degats(t *testing.T) {
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
	var (
		paquets, type0                       int
		trameOK, trameKO, deltasLies, masqOK int
		ctrlDeltas, ctrlMasq                 int
		sources                              = map[uint64]int{}
		victimes                             = map[uint64]int{}
		degats                               = map[uint64]int{}
		part0                                = map[uint64]int{}
		part1                                = map[uint64]int{}
		avecVictime, negatifs                int
		dmgSum                               float64
	)
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
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xC0 {
				continue
			}
			paquets++
			br := NewBitReader(pay)
			br.Skip(2) // config + continuation
			if br.ReadBits(7) != 0 {
				continue // type 1 (damage_section_response), pas type 0
			}
			type0++
			// 3 references d'en-tete du type 0 : dom1 (sonde), dom1 (sonde), dom7. Les deux
			// domaine-1 sont les ENTITES participantes (blesse / responsable).
			if id0, ok0 := lot1RefDom1(br); ok0 {
				part0[id0]++
			}
			if id1, ok1 := lot1RefDom1(br); ok1 {
				part1[id1]++
			}
			lot1RefDom(br, 7)
			// La charge damage_aftermath.
			r := lot1DecodeDamageAftermath(br)
			if r.hasSource {
				sources[r.sourceID]++
			}
			degats[r.dmgRaw]++
			dmgSum += r.dmgClear
			if r.negatif {
				negatifs++
			}
			if r.hasVictim {
				avecVictime++
				victimes[r.victimIdx]++
			}
			// Bit de CONTINUATION de la liste d'evenements : 1 = un autre evenement suit
			// (on ne teste la trame que sur les paquets a evenement unique), 0 = la trame commence.
			if br.ReadBit() {
				continue
			}
			// LE JUGE : la trame doit se fermer proprement (recEnd atteint) comme une trame de
			// tick, et aller LOIN. Metrique discriminante = taux de fermeture + profondeur,
			// compares a un TEMOIN a offset faux (le meme decodage decale de +3 bits).
			pos := br.BitPos()
			w := NewWorld(reg)
			w.Restore(snap)
			recs, decErr := DecodeFrameRecords(br, w, DefaultFrameConfig())
			deltasLies += len(recs) // profondeur atteinte avant desync/fin
			if decErr == nil {
				trameOK++
				masqOK += len(recs)
			} else {
				trameKO++
			}
			// TEMOIN NEGATIF : le meme, decale de +3 bits (mauvais cadrage volontaire).
			if p := pos + 3; p+16 < len(pay)*8 {
				w2 := NewWorld(reg)
				w2.Restore(snap)
				cbr := NewBitReader(pay)
				cbr.Skip(p)
				crecs, cerr := DecodeFrameRecords(cbr, w2, DefaultFrameConfig())
				ctrlDeltas += len(crecs)
				if cerr == nil {
					ctrlMasq++
				}
			}
		}
	}
	t.Logf("== 0xC0 : %d paquets · type 0 (damage_aftermath) : %d ==", paquets, type0)
	t.Logf("SOURCE (tag du degat) : %d distinctes · DEGAT (code R5) : %s", len(sources), lot1TopU64(degats, 8))
	t.Logf("DEGAT EN CLAIR : magnitude dq sur [0,16] (32 niveaux) · moyenne %.2f · negatifs (soin, Kscale=-1) : %d / %d",
		dmgSum/float64(max(1, type0)), negatifs, type0)
	t.Logf("PARTICIPANTS (refs d'en-tete, domaine 1) : ref0 %d distincts, ref1 %d distincts — les entites blesse/responsable",
		len(part0), len(part1))
	// Les valeurs sont-elles dans la plage des slots bipedes (~512..615, croisable killsource)
	// ou des handles arbitraires (non resolubles hors ligne) ?
	minMax := func(m map[uint64]int) (uint64, uint64, int) {
		mn, mx, dansPlage := uint64(1<<20), uint64(0), 0
		for v := range m {
			if v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
			if v >= 500 && v <= 700 {
				dansPlage += m[v]
			}
		}
		return mn, mx, dansPlage
	}
	mn0, mx0, dp0 := minMax(part0)
	mn1, mx1, dp1 := minMax(part1)
	t.Logf("  ref0 : min=%d max=%d · dans plage bipede [500,700] : %d evenements", mn0, mx0, dp0)
	t.Logf("  ref1 : min=%d max=%d · dans plage bipede [500,700] : %d evenements", mn1, mx1, dp1)
	t.Logf("VICTIME (ref finale domaine 0) : presente %d / %d (%.1f %%), %d distinctes",
		avecVictime, type0, lot1Pct(avecVictime, type0), len(victimes))
	nEvt := trameOK + trameKO
	profReel := float64(deltasLies) / float64(max(1, nEvt))
	profCtrl := float64(ctrlDeltas) / float64(max(1, nEvt))
	// LE DISCRIMINANT EST LA PROFONDEUR, pas le taux de fermeture : au BON cadrage la trame
	// est une vraie trame de tick (~2.4 records/paquet, cf. 0xA0 ~2.9) ; a un offset FAUX
	// elle tombe aussitot sur un faux marqueur de fin (~0.2 record/paquet — fermeture triviale,
	// d'ou un taux de "fermeture" trompeusement HAUT au temoin).
	t.Logf("JUGE — trame (evenement unique) : %d records / %d paquets = %.2f/paquet · fermee %.1f %%",
		deltasLies, nEvt, profReel, lot1Pct(trameOK, nEvt))
	t.Logf("TEMOIN NEGATIF (+3 bits) : %d records / %d = %.2f/paquet · fermee %.1f %% (triviale)",
		ctrlDeltas, nEvt, profCtrl, lot1Pct(ctrlMasq, nEvt))
	ok := nEvt >= 50 && profReel >= 1.0 && deltasLies >= 3*ctrlDeltas
	t.Logf("VERDICT (profondeur reelle >= 1/paquet ET >= 3x le temoin) : %s", lot1Verdict(ok))
}
