package filmdec

// weapon_hits_decode.go — DECODEURS de production sortis des instruments de recherche (Lot 2 du
// plan PRECISION_ARME_DISTANCE). Ils vivaient dans des fichiers `_test.go` (lot1_degats,
// lot1_evenements, lot1_monde_chrono, lot1_degats_blesse) ; la passe killcollector (Lot 3) doit
// les appeler hors test. Ils sont DEPLACES ici tels quels (memes noms, meme package) : les
// instruments de recherche qui les utilisent restent inchanges — une seule copie de chaque
// decodeur, pas de duplication (CLAUDE.md regle 6).
//
// Contenu :
//   - references d'en-tete du descripteur 0x1451f98d0 (lot1RefDom / lot1RefDom1) et leurs largeurs ;
//   - la charge damage_aftermath (0xC0 type 0) : lot1DecodeDamageAftermath + dequantification ;
//   - la resolution slot base-512 des index bruts vers les slots bipedes (lot1chBases,
//     lot1chIsBiped, lot1ArgmaxBase).

// lot1RefDomWidths : largeur de l'index R(w) par domaine (table 0x1451f98d0, lot B1/E2).
var lot1RefDomWidths = map[int]int{0: 13, 1: 13, 2: 8, 3: 8, 4: 9, 5: 8, 6: 9, 7: 13, 8: 13}

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
// workflow damage-aftermath-reader). Les dequantifications float sont 0 bit ; seuls les codes
// sont lus.
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

// lot1chBases : jeu de bases candidat pour la resolution slot (identique a victime_slot pour
// comparabilite). La bande bipede se resout typiquement a 512.
var lot1chBases = []int{0, 128, 256, 384, 448, 480, 500, 508, 510, 512, 514, 516, 520, 544, 576}

// lot1chIsBiped indique si l'index brut idx, rapporte a la base, atterrit sur un slot lie a un
// bipede dans le monde reconstruit w.
func lot1chIsBiped(w *World, base, idx int) bool {
	if idx < 0 {
		return false
	}
	slot := base + idx
	if slot < 0 || slot >= 8192 {
		return false
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	return ok && ti == BipedTypeIndex
}

// lot1ArgmaxBase rend la base a l'atterrissage bipede maximal (base la plus basse en cas d'egalite).
func lot1ArgmaxBase(hits map[int]int) int {
	best, bestN := lot1chBases[0], -1
	for _, b := range lot1chBases {
		if hits[b] > bestN {
			best, bestN = b, hits[b]
		}
	}
	return best
}
