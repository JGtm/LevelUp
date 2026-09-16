package filmdec

// bit_leaf_readers.go — LES PRIMITIVES DE LECTURE PARTAGEES (portes, options, entiers de
// largeur variable), avec l'adresse de leur deserialiseur en commentaire.
//
// Sorties de `unit_weaponstate.go` par deplacement pur au lot 2.7 (scission des fichiers de
// plus de 500 lignes) : aucune ligne de logique n'a change. Elles y etaient nees parce que
// les composants d'unite ont ete portes les premiers ; elles servent aujourd'hui tout le
// paquet, et un fichier de composants n'est pas leur place.

// ---------------------------------------------------------------------------
// Shared leaf readers (resolved deser primitives, addresses in comments).
// ---------------------------------------------------------------------------

// consumeGateR reads a 1-bit presence gate; if SET (bit==1), reads width bits.
// Mirrors the idiom "R(1); if bit==1: R(width)" — CONFIRMED bit-exact by decompile
// for FUN_140c50d1c (w=8), FUN_140cec0a0 (w=8), FUN_140e82b84 (w=12): each yields a
// 0xFFFF sentinel with NO payload on the bit==0 branch and reads the body on bit==1.
//
// NOTE: FUN_1406d1024 (w=6) is NOT in this group despite a similar shape — its gate
// polarity is INVERTED (payload present on bit==0). Use consumeGate0R for it.
func consumeGateR(br *BitReader, width uint) {
	if br.ReadBit() {
		br.ReadBits(width)
	}
}

// consumeGate0R reads a 1-bit gate; if CLEAR (bit==0), reads width bits (else a
// 0xFFFFFFFF sentinel, no payload). Mirrors FUN_1406d1024 (w=6), whose polarity is
// the INVERSE of consumeGateR/FUN_140c50d1c. CONFIRMED by decompile: the "-1 < lVar3"
// branch (top bit clear == gate 0) is the one that advances the bit counter by width;
// the bit==1 branch returns 0xffffffff without reading. Same polarity as consumeID2
// (FUN_1406d00ec). Used by the held-weapon i43 float block (consume1407f0550) and the
// i48 biped-desired-ability-set deser (consumeBipedDesiredAbilitySet).
func consumeGate0R(br *BitReader, width uint) {
	if !br.ReadBit() {
		br.ReadBits(width)
	}
}

// consumeOpt32 mirrors FUN_14080d69c: R(1) gate; if set R(32) (FUN_14080d6f0).
func consumeOpt32(br *BitReader) {
	at := br.BitPos()
	if br.ReadBit() {
		v := br.ReadBits(32) // FUN_14080d6f0 = R(32)
		br.obs.publishUnitRef(UnitRefRead{
			Kind: UnitRefWord32, StartBit: at, EndBit: br.BitPos(),
			Present: true, Val: uint32(v),
		})
		return
	}
	br.obs.publishUnitRef(UnitRefRead{Kind: UnitRefWord32, StartBit: at, EndBit: br.BitPos()})
}

// consumeID2 mirrors FUN_1406d00ec: R(1); if bit==0 R(2); else nothing.
func consumeID2(br *BitReader) {
	if !br.ReadBit() {
		br.ReadBits(2)
	}
}

// readVarWidthInt porte FUN_1406d3140 : l entier a largeur variable du flux.
//
// `param3` est le `param_3` du jeu — la CATEGORIE, qui choisit la plage dans la table de
// `FUN_140d10bb0` et donc la largeur du champ de valeur (cf. `varwidth.go`). Cout en bits :
// [sonde R(1) quand param3 == 1] + `varWidthBits(param3)` bits de valeur + R(2) de queue.
//
// LA SONDE BASCULE LA CATEGORIE, elle ne fait pas que couter un bit : quand param3 vaut 1 et
// que le bit lu vaut 1, le jeu prend l entree 4 de la table (9 bits au lieu de 13). C est la
// correction du 2026-09-15 ; le portage precedent lisait 13 bits dans les deux cas.
//
// LES RETOURS ONT ETE AJOUTES LE 2026-08-30 (sonde i26) SANS CHANGER UN BIT : la valeur et
// les deux bits de queue etaient consommes puis jetes.
func readVarWidthInt(br *BitReader, param3 int) (val uint64, tail uint64) {
	if param3 == varWidthProbeCategory && br.ReadBit() { // FUN_1406cf008, param_3 == 1 SEULEMENT
		param3 = varWidthProbeSlot
	}
	if w := varWidthBits(param3); w > 0 {
		val = br.ReadBits(w)
	}
	tail = br.ReadBits(2) // 2 bits de queue (generation du handle)
	return val, tail
}

// consume1408f0ac4 porte FUN_1408f0ac4 : une porte R(1), puis l entier a largeur variable de
// `FUN_1406d3140` dans la CATEGORIE `param3` que l appelant du jeu pousse dans R8D.
//
// LE PARAMETRE EST LE `param_3` DU JEU, ET IL EST OBLIGATOIRE DEPUIS LE 2026-09-15 : chaque
// site d appel porte le sien, relu sur le desassemblage (tableau au §4 du plan). Un defaut
// implicite remettrait 13 bits partout, c est-a-dire le defaut que ce lot corrige.
// `FUN_1406cb0cc`, appele ensuite par le jeu, ne consomme AUCUN bit (controle de config).
func consume1408f0ac4(br *BitReader, param3 int) (bool, uint64) {
	val, _, present := consume1408f0ac4Probe(br, param3)
	return present, val
}

// consume1408f0ac4Probe est FUN_1408f0ac4 avec sa CATEGORIE explicite, et rend les trois
// valeurs que la sonde d i26 lit. Memes bits que `consume1408f0ac4`, rien de plus ni de moins.
func consume1408f0ac4Probe(br *BitReader, param3 int) (val, tail uint64, present bool) {
	probe := param3 == varWidthProbeCategory
	at := br.BitPos()
	if br.ReadBit() {
		val, tail = readVarWidthInt(br, param3) // FUN_1406d3140
		// FUN_1406cb0cc consumes 0 bits (config check).
		br.obs.publishUnitRef(UnitRefRead{
			Kind: UnitRefVarWidth, StartBit: at, EndBit: br.BitPos(), Present: true,
			Val: uint32(val), Tail: uint32(tail), Probe: probe,
		})
		return val, tail, true
	}
	br.obs.publishUnitRef(UnitRefRead{
		Kind: UnitRefVarWidth, StartBit: at, EndBit: br.BitPos(), Probe: probe,
	})
	return 0, 0, false
}

// consume1411b1ac0 mirrors FUN_1411b1ac0 -> FUN_140e82b84: R(1); if set R(12).
func consume1411b1ac0(br *BitReader) { consumeGateR(br, 12) }
