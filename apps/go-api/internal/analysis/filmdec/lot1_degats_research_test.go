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

// lot1RefDom consomme une reference gardee du domaine dom (sans sonde). Rend (index, presente).
func lot1RefDom(br *BitReader, dom int) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	idx := br.ReadBits(uint(lot1RefDomWidths[dom]))
	br.Skip(2)
	return idx, true
}

// lot1RefDom1 consomme une reference du domaine 1 (AVEC sonde : R(1) sonde ; largeur 9 si
// sonde==1, sinon 13 ; puis R(2) generation).
func lot1RefDom1(br *BitReader) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	w := 13
	if br.ReadBit() { // sonde
		w = 9
	}
	idx := br.ReadBits(uint(w))
	br.Skip(2)
	return idx, true
}

// lot1DmgResult : ce que damage_aftermath rend d'exploitable.
type lot1DmgResult struct {
	sourceID  uint64
	hasSource bool
	dmgRaw    uint64  // R(5) magnitude principale (code 0..31)
	dmgClear  float64 // magnitude en clair : dq(dmgRaw, 0..16), signee (soin si porte d'echelle)
	dmg2Raw   uint64  // R(5) second scalaire
	negatif   bool    // porte d'echelle : Kscale = -1.0 => magnitude negee (soin)
	victimIdx uint64
	hasVictim bool
}

// lot1Dequant : la dequantification de l'exe (FUN_1406d84b4, flagA=0 -> N niveaux ; flagB=1 ->
// bornes exactes). N = 1<<width ; code 0 -> vmin ; code N-1 -> vmax ; sinon interpolation.
func lot1Dequant(raw uint64, width uint, vmin, vmax float64) float64 {
	N := float64(uint64(1) << width)
	if raw == 0 {
		return vmin
	}
	if raw == uint64(N)-1 {
		return vmax
	}
	return vmin + (float64(raw-1)+0.5)*(vmax-vmin)/(N-2)
}

// lot1DecodeDamageAftermath consomme la charge damage_aftermath EXACTEMENT (grammaire du
// workflow). Les dequantifications float sont 0 bit ; seuls les codes sont lus.
func lot1DecodeDamageAftermath(br *BitReader) lot1DmgResult {
	var r lot1DmgResult
	// (1) source : R(1) porte ; si 1 : R(32) (id de tag global)
	if br.ReadBit() {
		r.sourceID = br.ReadBits(32)
		r.hasSource = true
	}
	// (2) +0x10 : R(1) ; si 1 : R(5)
	if br.ReadBit() {
		br.Skip(5)
	}
	br.Skip(19)       // (3) +0x14 : R(19)
	if br.ReadBit() { // (4) porte +0x20
		br.Skip(19 + 12) // R(19) + R(12)
	}
	br.Skip(5 + 5 + 6) // (5) bloc vecteur : R(5)+R(5)+R(6)
	if br.ReadBit() {  // (6) porte +0x40
		br.Skip(5)
	}
	// (7) 15 drapeaux R(1) ; le 15e (bit 28) garde un R(32)
	var last bool
	for i := 0; i < 15; i++ {
		last = br.ReadBit()
	}
	if last { // (8) si bit 28 : R(32)
		br.Skip(32)
	}
	br.Skip(1)                 // (9) bit 19 : R(1)
	br.Skip(3)                 // (10) +0x5c : R(3)
	r.dmgRaw = br.ReadBits(5)  // (11) magnitude principale R(5), dequant [0,16]
	r.dmg2Raw = br.ReadBits(5) // (12) second scalaire R(5), dequant [0,3]
	r.dmgClear = lot1Dequant(r.dmgRaw, 5, 0, 16)
	if br.ReadBit() { // (13) porte d'echelle : Kscale = -1.0 (DAT_143cd84ec) => magnitude negee
		r.negatif = true
		r.dmgClear = -r.dmgClear
	}
	br.Skip(4)         // (14) +0x52 : R(4)
	if !br.ReadBit() { // (15) +0x54 : R(1) polarite INVERSEE ; si bit==0 : R(10)
		br.Skip(10)
	}
	f58 := br.ReadBits(4) // (16) +0x58 : R(4)
	if br.ReadBit() {     // (17) porte +0x60 : R(1) ; si 1 : R(32)
		br.Skip(32)
	}
	if f58 == 1 { // (18) si F58==1 : R(8)
		br.Skip(8)
	}
	br.Skip(4)        // (19) +0x70 : R(4)
	if br.ReadBit() { // (20) VICTIME : R(1) porte ; si 1 : R(13) idx + R(2) selecteur
		r.victimIdx = br.ReadBits(13)
		br.Skip(2)
		r.hasVictim = true
	}
	return r
}

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
