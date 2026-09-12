package filmdec

// lot1_degats_type1_research_test.go — LOT 1 : damage_section_response (TYPE 1, octet 0xC0,
// bit de type R(7) == 1). Le meme octet que damage_aftermath (type 0) ; seul le bit de type
// change. ENJEU : les armes LOURDES (M41 SPNKr, Hydra, Skewer, Ravager, Shock Rifle, Mangler,
// Stalker, Bulldog, Fuel Rod) n'emettent PAS de damage_aftermath (0 % de touches en type 0,
// mesure baseline lot1_attrib_arme_tir) ; leur degat est-il ICI (type 1) ?
//
// GRAMMAIRE LUE DANS L'EXE (descripteur 0x144724f78, vtable 0x143d0fa10) :
//   - domaines des 3 refs d'en-tete (vtable+0x58 = FUN_14080a048 -> switch(i)) :
//       ref0 -> domaine 1 (entite, sonde) ; ref1 -> domaine 8 ; ref2 -> domaine 7.
//       (type 0 etait dom1/dom1/dom7 : DEUX entites. type 1 n'en porte qu'UNE, ref0.)
//   - charge utile (vtable+0x68 = FUN_140968368, structure de reception 28 octets) :
//       R(5)                          // p0
//       R(1) g1 ; si g1 == 0 : R(4)   // p1, optionnel, POLARITE INVERSEE (FUN_1409684dc)
//       R(3)                          // p2 (FUN_1424d0f48)
//       R(1) g2 ; si g2 == 1 : R(19)  // p3 : direction quantifiee (FUN_14076dc04 width 0x13
//                                     //      -> FUN_1406d8288 = unpack vecteur unite, 0 bit)
//     min 10 bits, max 33 bits. AUCUN tag source R(32), AUCUNE magnitude R(5) sur [0,16],
//     AUCUNE seconde entite : type 1 n'est PAS un enregistrement de degat autoritaire, c'est
//     une REPONSE DE SECTION (quelle section touchee, depuis quelle direction).
//
// LE JUGE reste l'ORACLE DE TRAME (comme TestLot1Degats) : decoder l'evenement EN ENTIER puis
// la trame de records ; au bon cadrage la profondeur est celle d'une vraie trame de tick, a un
// bit pres (temoin +3) elle s'effondre.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne 12 chunks.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// lot1Type1Result : ce que damage_section_response rend, decode au bit pres.
type lot1Type1Result struct {
	p0     uint64 // R(5)
	p1     uint64 // R(4) si g1 == 0
	hasP1  bool
	p2     uint64 // R(3)
	dirRaw uint64 // R(19) si g2 == 1
	hasDir bool
}

// lot1DecodeDamageSectionResponse consomme la charge type 1 EXACTEMENT (FUN_140968368).
func lot1DecodeDamageSectionResponse(br *BitReader) lot1Type1Result {
	var r lot1Type1Result
	r.p0 = br.ReadBits(5)
	if !br.ReadBit() { // g1 : polarite inversee (bit 0 -> champ present)
		r.p1 = br.ReadBits(4)
		r.hasP1 = true
	}
	r.p2 = br.ReadBits(3)
	if br.ReadBit() { // g2 : direction presente
		r.dirRaw = br.ReadBits(19)
		r.hasDir = true
	}
	return r
}

// lot1Type1Evt : un damage_section_response horodate, refs d'en-tete brutes (non resolues).
type lot1Type1Evt struct {
	ts               uint64
	idx0, idx1, idx2 int // ref0 dom1 (sonde), ref1 dom8, ref2 dom7 ; -1 si absente
	pay              lot1Type1Result
}

// lot1ScanType1 rejoue les chunks (keyframe + tick-frames) et rend les evenements type 1 avec
// leurs refs, plus la base bipede la plus resolvante pour ref0 (dom1). Meme squelette que
// sondeScanDamage, cible le type 1 au lieu du type 0.
func lot1ScanType1(t *testing.T, dir string, reg *Registry, n int) ([]lot1Type1Evt, int) {
	t.Helper()
	cfg := DefaultFrameConfig()
	hit := map[int]int{}
	var evs []lot1Type1Evt
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		w := NewWorld(reg)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				w.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, w, cfg)
			}
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xC0 {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 1 {
				continue // type 0 (damage_aftermath) ou autre : ignore
			}
			ev := lot1Type1Evt{ts: pk.TimestampUS, idx0: -1, idx1: -1, idx2: -1}
			if i0, ok := lot1RefDom1(br); ok { // ref0 dom1 (sonde)
				ev.idx0 = int(i0)
			}
			if i1, ok := lot1RefDom(br, 8); ok { // ref1 dom8
				ev.idx1 = int(i1)
			}
			if i2, ok := lot1RefDom(br, 7); ok { // ref2 dom7
				ev.idx2 = int(i2)
			}
			ev.pay = lot1DecodeDamageSectionResponse(br)
			evs = append(evs, ev)
			for _, b := range lot1chBases {
				if lot1chIsBiped(w, b, ev.idx0) {
					hit[b]++
				}
			}
		}
	}
	return evs, lot1ArgmaxBase(hit)
}

// TestLot1DegatsType1 : grammaire + oracle de trame + resolution des refs.
func TestLot1DegatsType1(t *testing.T) {
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
	t.Logf("== film %s · %d chunks (damage_section_response, type 1) ==", filepath.Base(dir), n)

	var (
		type1                                    int
		ref0Abs, ref1Abs, ref2Abs                int
		ref0                                     = map[uint64]int{}
		ref1                                     = map[uint64]int{}
		ref2                                     = map[uint64]int{}
		p0Hist, p2Hist                           = map[uint64]int{}, map[uint64]int{}
		p1Present, dirPresent                    int
		trameOK, trameKO, deltasLies, ctrlDeltas int
		ctrlMasq                                 int
	)
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		pks := WalkPackets(data)
		wBase := NewWorld(reg)
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				wBase.BindFull(uint32((r.Gen<<30)|r.Slot), uint32(r.TI))
			}
		}
		cfg := DefaultFrameConfig()
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			if pay := pk.Payload(data); pay[0]&0x40 == 0 {
				br := NewBitReader(pay)
				_, _ = DecodeFrameRecords(br, wBase, cfg)
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
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 1 {
				continue
			}
			type1++
			if i0, ok := lot1RefDom1(br); ok {
				ref0[i0]++
			} else {
				ref0Abs++
			}
			if i1, ok := lot1RefDom(br, 8); ok {
				ref1[i1]++
			} else {
				ref1Abs++
			}
			if i2, ok := lot1RefDom(br, 7); ok {
				ref2[i2]++
			} else {
				ref2Abs++
			}
			r := lot1DecodeDamageSectionResponse(br)
			p0Hist[r.p0]++
			p2Hist[r.p2]++
			if r.hasP1 {
				p1Present++
			}
			if r.hasDir {
				dirPresent++
			}
			// Bit de continuation de la liste d'evenements : 1 = un autre evenement suit.
			if br.ReadBit() {
				continue
			}
			// ORACLE : la trame doit se fermer et aller loin ; temoin = +3 bits.
			pos := br.BitPos()
			w := NewWorld(reg)
			w.Restore(snap)
			recs, decErr := DecodeFrameRecords(br, w, DefaultFrameConfig())
			deltasLies += len(recs)
			if decErr == nil {
				trameOK++
			} else {
				trameKO++
			}
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
	t.Logf("TYPE 1 (damage_section_response) : %d paquets", type1)
	t.Logf("CHARGE : p0 R(5) %s · p2 R(3) %s", lot1TopU64(p0Hist, 6), lot1TopU64(p2Hist, 6))
	t.Logf("CHARGE : p1 R(4) present %d / %d (%.1f %%) · direction R(19) presente %d / %d (%.1f %%)",
		p1Present, type1, lot1Pct(p1Present, type1), dirPresent, type1, lot1Pct(dirPresent, type1))
	lot1Type1RefsLog(t, "ref0 dom1 (entite, sonde)", ref0, ref0Abs, type1)
	lot1Type1RefsLog(t, "ref1 dom8", ref1, ref1Abs, type1)
	lot1Type1RefsLog(t, "ref2 dom7", ref2, ref2Abs, type1)

	nEvt := trameOK + trameKO
	profReel := float64(deltasLies) / float64(max(1, nEvt))
	profCtrl := float64(ctrlDeltas) / float64(max(1, nEvt))
	t.Logf("JUGE — trame (evenement unique) : %d records / %d paquets = %.2f/paquet · fermee %.1f %%",
		deltasLies, nEvt, profReel, lot1Pct(trameOK, nEvt))
	t.Logf("TEMOIN NEGATIF (+3 bits) : %d records / %d = %.2f/paquet · fermee %.1f %%",
		ctrlDeltas, nEvt, profCtrl, lot1Pct(ctrlMasq, nEvt))
	// SEUIL adapte a un type RARE (55/22/42 paquets par film, la plupart en listes multiples) :
	// le discriminant reste la PROFONDEUR (trame tick-like ~3/paquet) contre l'effondrement du
	// temoin (~0/paquet). On exige >= 2 records/paquet, >= 3x le temoin, et un minimum de 4
	// paquets a evenement unique testables.
	ok := nEvt >= 4 && profReel >= 2.0 && deltasLies >= 3*(ctrlDeltas+1)
	t.Logf("VERDICT GRAMMAIRE (profondeur >= 2/paquet ET >= 3x le temoin, n>=4) : %s", lot1Verdict(ok))
}

// lot1Type1RefsLog publie la distribution d'une reference : distincts, absente, plage bipede.
func lot1Type1RefsLog(t *testing.T, label string, m map[uint64]int, abs, tot int) {
	t.Helper()
	mn, mx := uint64(1<<20), uint64(0)
	dansPlage, present := 0, 0
	for v, c := range m {
		present += c
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
		if v >= 500 && v <= 700 {
			dansPlage += c
		}
	}
	if present == 0 {
		mn = 0
	}
	t.Logf("  %s : presente %d / %d (%.1f %%) · %d distincts · min=%d max=%d · plage bipede brute [500,700] : %d",
		label, present, tot, lot1Pct(present, tot), len(m), mn, mx, dansPlage)
}

// TestLot1DegatsType1ArmesLourdes : les armes lourdes touchent-elles via le type 1 ?
// Le lien tir<->degat de type 0 passait par le RESPONSABLE (ref1 dom1). Le type 1 n'a qu'une
// entite dom1 (ref0) : on teste les DEUX hypotheses (ref0 = attaquant ; ref0 = victime, via
// coincidence temporelle) contre un temoin decale, et on chiffre la precision recuperee.
func TestLot1DegatsType1ArmesLourdes(t *testing.T) {
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
	t.Logf("== film %s · %d chunks (armes lourdes via type 1) ==", filepath.Base(dir), n)

	shots := attribCollectShots(t, dir, n)
	evs, base := lot1ScanType1(t, dir, reg, n)
	t.Logf("collecte : %d tirs longs (0xD2 t36) · %d type 1 · base bipede ref0 = %d", len(shots), len(evs), base)

	// Index temporels du type 1 : tous les ts (coincidence) et ts par ref0 (cle attaquant/victime).
	var allTs, byRef0Ts, byRef0Key []uint64
	for _, e := range evs {
		allTs = append(allTs, e.ts)
		if e.idx0 >= 0 {
			byRef0Ts = append(byRef0Ts, e.ts)
			byRef0Key = append(byRef0Key, uint64(e.idx0))
		}
	}
	sort.Slice(allTs, func(a, b int) bool { return allTs[a] < allTs[b] })
	byRef0 := lot1mtIndexByKey(byRef0Ts, byRef0Key)

	W := attribW         // 250 ms
	OFF := attribOFF     // 3 s temoin
	WideW := attribWideW // 2 s (projectiles lents)

	// M1 : fiabilite globale du lien tir -> type 1.
	fwdN, keySame, keyShift, anySame, anyShift := 0, 0, 0, 0, 0
	for _, s := range shots {
		if !s.has {
			continue
		}
		fwdN++
		if lot1mtNear(byRef0[s.att], s.ts, W) {
			keySame++
		}
		if lot1mtNear(byRef0[s.att], s.ts+OFF, W) {
			keyShift++
		}
		if lot1mtNear(allTs, s.ts, W) {
			anySame++
		}
		if lot1mtNear(allTs, s.ts+OFF, W) {
			anyShift++
		}
	}
	fl := func(v float64) float64 {
		if v < 1 {
			return 1
		}
		return v
	}
	ks, ksh := lot1Pct(keySame, fwdN), lot1Pct(keyShift, fwdN)
	as, ash := lot1Pct(anySame, fwdN), lot1Pct(anyShift, fwdN)
	t.Logf("M1 lien tir -> type 1 (tous tirs, n=%d) :", fwdN)
	t.Logf("   par CLE ref0==attaquant ±%dms : %.1f %% · temoin +%ds %.1f %% -> %.1fx",
		W/1000, ks, OFF/1_000_000, ksh, ks/fl(ksh))
	t.Logf("   par COINCIDENCE (n'importe quel type 1) ±%dms : %.1f %% · temoin +%ds %.1f %% -> %.1fx",
		W/1000, as, OFF/1_000_000, ash, as/fl(ash))

	// M2 : precision par arme via type 1, focalise sur les LOURDES. Deux colonnes : cle ref0
	// (si ref0 = attaquant) et coincidence (si ref0 = victime). Fenetre elargie pour le vol.
	type tally struct{ n, key, anyW, anyWide int }
	by := map[uint64]*tally{}
	for _, s := range shots {
		if !s.has {
			continue
		}
		ta := by[s.wid]
		if ta == nil {
			ta = &tally{}
			by[s.wid] = ta
		}
		ta.n++
		if lot1mtNear(byRef0[s.att], s.ts, W) {
			ta.key++
		}
		if lot1mtNear(allTs, s.ts, W) {
			ta.anyW++
		}
		if lot1mtNear(allTs, s.ts, WideW) {
			ta.anyWide++
		}
	}
	type row struct {
		wid                   uint64
		n, key, anyW, anyWide int
		heavy                 bool
	}
	var rows []row
	for w, ta := range by {
		rows = append(rows, row{w, ta.n, ta.key, ta.anyW, ta.anyWide, lot1IsHeavy(attribWeaponName(w))})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].heavy != rows[j].heavy {
			return rows[i].heavy // lourdes d'abord
		}
		return rows[i].n > rows[j].n
	})
	t.Logf("M2 PRECISION par arme via type 1 (cle ref0==attaquant · coincidence 250ms · coincidence 2s) :")
	hN, hKey, hAny := 0, 0, 0
	for _, r := range rows {
		if r.n < 5 {
			continue
		}
		mark := "     "
		if r.heavy {
			mark = "LOURD"
			hN += r.n
			hKey += r.key
			hAny += r.anyW
		}
		t.Logf("   [%s] %-24s : %d tirs · cle %.1f %% · coinc250 %.1f %% · coinc2s %.1f %%", mark,
			attribWeaponName(r.wid), r.n, lot1Pct(r.key, r.n), lot1Pct(r.anyW, r.n), lot1Pct(r.anyWide, r.n))
	}
	t.Logf("M2 BILAN LOURDES : %d tirs · cle ref0 %.1f %% · coincidence 250ms %.1f %%", hN,
		lot1Pct(hKey, hN), lot1Pct(hAny, hN))
	// VERDICT : le type 1 recupere-t-il les lourdes ? Il faut que le lien soit REEL (cle > temoin
	// OU coincidence > temoin) ET que les lourdes montent nettement au-dessus de leur 0 % en type 0.
	linkReal := (ks >= 1.5*fl(ksh)) || (as >= 1.5*fl(ash))
	heavyUp := hN >= 20 && lot1Pct(hAny, hN) >= 30
	t.Logf("VERDICT ARMES LOURDES (lien reel ET lourdes >= 30 %% de coincidence) : %s",
		lot1Verdict(linkReal && heavyUp))
}

// lot1IsHeavy : l'arme est-elle une arme lourde/explosif/faisceau (0 % en type 0) ?
func lot1IsHeavy(name string) bool {
	for _, k := range []string{"SPNKr", "Hydra", "Skewer", "Ravager", "Shock", "Mangler",
		"Stalker", "Bulldog", "Fuel Rod", "Rod"} {
		if strings.Contains(name, k) {
			return true
		}
	}
	return false
}
