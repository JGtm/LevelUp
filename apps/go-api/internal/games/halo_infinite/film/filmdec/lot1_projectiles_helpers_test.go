package filmdec

// lot1_projectiles_helpers_test.go — decodeurs et collecteurs de l'instrument
// lot1_projectiles_research_test.go (scinde pour le seuil de 500 lignes). Voir l'en-tete de ce
// fichier-la pour la grammaire lue dans l'exe, les seuils et les mesures.

import "testing"

// La reference d'en-tete unique (slot 0) est du DOMAINE 5 (commutateur 0x1408096ec :
// lea eax,[rdx+5] pour l'index 0, INT3 pour tout index > 0). Sa LARGEUR est
// ceil(log2(capacite du pool)) — une valeur RUNTIME (table DAT_1451f98d0 vide dans l'image
// statique, FUN_1406d310c). On la CALIBRE par film : la largeur juste est celle qui rend
// absents les slots 1 et 2 (le commutateur asserte pour tout slot > 0, donc leurs bits de
// presence valent toujours 0 dans un flux valide).

// projRef0 lit la reference d'en-tete du slot 0 (domaine 5) a une largeur DONNEE : R(1)
// presence ; si presente R(width) index + R(2) generation. Rend (index, presente).
func projRef0(br *BitReader, width uint) (int, bool) {
	if !br.ReadBit() {
		return -1, false
	}
	idx := br.ReadBits(width)
	br.Skip(2)
	return int(idx), true
}

// projEvt : un evenement projectile horodate, decode a l'en-tete + variant-name.
type projEvt struct {
	ts       uint64
	impact   bool   // false = detonate (0xC2 t5), true = impact_effect (0xC3 t6/7)
	ref0     int    // ref0 domaine 5 (brut) ; -1 si absente
	has1     bool   // slot 1 present (doit etre ~jamais : sanity de l'en-tete)
	has2     bool   // slot 2 present (idem)
	variant  uint64 // variant-name R(32)
	hasVar   bool   // variant-name presente (porte==0)
	variant3 uint64 // variant-name lue a +3 bits (temoin d'offset)
}

// projDecodeHeader consomme les 3 slots de reference (domaine 5, largeur width) et rend ref0 +
// presence des slots 1/2. Le BitReader est positionne APRES l'en-tete, au debut de la charge.
func projDecodeHeader(br *BitReader, width uint) (ref0 int, has1, has2 bool) {
	ref0, _ = projRef0(br, width)
	_, has1 = projRef0(br, width)
	_, has2 = projRef0(br, width)
	return ref0, has1, has2
}

// projPacketType rend true et le type (5/6/7) si pay est un evenement projectile, sinon false.
func projPacketType(pay []byte) (impact bool, ok bool) {
	if pay[0] != 0xC2 && pay[0] != 0xC3 {
		return false, false
	}
	br := NewBitReader(pay)
	br.Skip(2)
	typ := br.ReadBits(7)
	switch {
	case pay[0] == 0xC2 && typ == 5:
		return false, true
	case pay[0] == 0xC3 && (typ == 6 || typ == 7):
		return true, true
	}
	return false, false
}

// projCalibrateWidth sweepe la largeur du domaine 5 sur tous les paquets projectile et rend
// celle qui rend le plus souvent ABSENTS les slots 1 et 2 (invariant du commutateur). Publie
// le balayage.
func projCalibrateWidth(t *testing.T, dir string, n int) uint {
	t.Helper()
	type acc struct{ tot, bothAbsent int }
	by := map[uint]*acc{}
	widths := []uint{6, 7, 8, 9, 10, 11, 12, 13}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(data)
			if _, ok := projPacketType(pay); !ok {
				continue
			}
			for _, w := range widths {
				a := by[w]
				if a == nil {
					a = &acc{}
					by[w] = a
				}
				a.tot++
				br := NewBitReader(pay)
				br.Skip(9) // 2 (config) + 7 (type)
				_, h1, h2 := projDecodeHeader(br, w)
				if !h1 && !h2 {
					a.bothAbsent++
				}
			}
		}
	}
	best, bestRate := widths[0], -1.0
	line := "   balayage largeur dom5 (slots 1+2 absents) :"
	for _, w := range widths {
		a := by[w]
		r := lot1Pct(a.bothAbsent, a.tot)
		line += " w" + itoa(int(w)) + "=" + itoa(int(r)) + "%"
		if r > bestRate {
			best, bestRate = w, r
		}
	}
	t.Logf("M0 CALIBRAGE largeur ref0 (domaine 5) :")
	t.Logf("%s", line)
	t.Logf("   largeur retenue = %d (%.1f %% de slots 1+2 absents)", best, bestRate)
	return best
}

// projVariantAfterGate lit "variant-name" R(32) apres la porte de charge (commune aux deux
// evenements) : R(1) porte ; si porte!=0 -> pas de variante ; sinon [R(1) g ; si g : R(32)]
// puis R(32). Rend aussi la valeur lue a +3 bits (temoin d'offset) prise AVANT consommation.
func projVariantAfterGate(br *BitReader) (variant uint64, has bool, witness uint64) {
	witness = br.peekBits3(3 + 32) // valeur decalee de 3 bits, meme longueur nominale (temoin)
	if br.ReadBit() {              // porte : variant-name absente si porte==1
		return 0, false, witness
	}
	if br.ReadBit() { // FUN_14080d69c : g ; si g : R(32)
		br.Skip(32)
	}
	return br.ReadBits(32), true, witness
}

// peekBits3 lit n bits a partir de 3 bits APRES la position courante, sans avancer — le temoin
// d'offset (+3) de l'oracle de tag. Prend les 32 bits de poids faible.
func (b *BitReader) peekBits3(n uint) uint64 {
	save := b.pos
	b.pos += 3
	v := b.ReadBits(n - 3)
	b.pos = save
	return v & 0xffffffff
}

// projScan rejoue keyframe + tick-frames (comme sondeScanDamage) et rend les evenements
// projectile decodes, plus la base bipede la plus resolvante pour ref0.
func projScan(t *testing.T, dir string, reg *Registry, n int, width uint) ([]projEvt, int) {
	t.Helper()
	cfg := DefaultFrameConfig()
	hit := map[int]int{}
	var evs []projEvt
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
			ev, ok := projDecodeOne(pay, pk.TimestampUS, width)
			if !ok {
				continue
			}
			evs = append(evs, ev)
			if ev.ref0 >= 0 {
				for _, b := range lot1chBases {
					if lot1chIsBiped(w, b, ev.ref0) {
						hit[b]++
					}
				}
			}
		}
	}
	return evs, lot1ArgmaxBase(hit)
}

// projDecodeOne decode un paquet 0xC2 (detonate, type 5) ou 0xC3 (impact, types 6/7).
func projDecodeOne(pay []byte, ts uint64, width uint) (projEvt, bool) {
	impact, ok := projPacketType(pay)
	if !ok {
		return projEvt{}, false
	}
	br := NewBitReader(pay)
	br.Skip(9) // 2 (config) + 7 (type)
	ev := projEvt{ts: ts, impact: impact}
	ev.ref0, ev.has1, ev.has2 = projDecodeHeader(br, width)
	if !impact {
		br.Skip(6) // FUN_140809454 (detonate uniquement)
	}
	ev.variant, ev.hasVar, ev.variant3 = projVariantAfterGate(br)
	return ev, true
}

// projFireVariant : un tir long horodate avec son variant_name ET son WeaponID (deux
// decodeurs eprouves du meme paquet 0xD2), pour nommer une variante par une arme.
type projFireVariant struct {
	ts      uint64
	att     uint64
	wid     uint64
	variant uint64
	has     bool
}

// projCollectFireVariants decode les tirs longs 0xD2 t36 : attaquant (dom1), variant_name
// (grammaire type 36) et WeaponID (offsets fixes). Un seul decodeur par champ.
func projCollectFireVariants(t *testing.T, dir string, n int) []projFireVariant {
	t.Helper()
	var out []projFireVariant
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 {
				continue
			}
			fv, ok := projDecodeFireVariant(pay, pk.TimestampUS)
			if ok {
				out = append(out, fv)
			}
		}
	}
	return out
}

// projDecodeFireVariant lit att + variant_name (grammaire type 36 de sondeScanFireArme) et le
// WeaponID (decodeFireEvent, offsets fixes) d'un paquet 0xD2 long.
func projDecodeFireVariant(pay []byte, ts uint64) (projFireVariant, bool) {
	br := NewBitReader(pay)
	br.Skip(2)
	if br.ReadBits(7) != 36 {
		return projFireVariant{}, false
	}
	fv := projFireVariant{ts: ts}
	if a, ok := lot1RefDom1(br); ok {
		fv.att = a
	}
	for range 2 { // ref1 dom8, ref2 dom7
		if br.ReadBit() {
			br.Skip(15)
		}
	}
	estCourt := br.ReadBit()
	br.Skip(1)
	br.Skip(8)
	if br.ReadBit() {
		br.Skip(5)
	}
	if !br.ReadBit() {
		br.Skip(2)
	}
	if br.ReadBit() {
		br.Skip(32)
	}
	fv.variant = br.ReadBits(32)
	if estCourt {
		return projFireVariant{}, false
	}
	fe, okF := decodeFireEvent(pay)
	if okF {
		fv.wid = fe.WeaponID
	}
	fv.has = true
	return fv, true
}
