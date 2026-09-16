package filmdec

// Keyframe/delta entity traversal: drives the mask-gated component loop
// (FUN_14076cb60) over an archetype's ordered component list (from the registry),
// dispatching each present component to its ported bit-consumer. The goal is to
// reach the held-weapon component (weapon-state-type-info) and read its variant-name.
//
// STATUS: iterative validation harness. consumeByName wires only the desers that
// are bit-exact-confirmed; unconfirmed ones return ported=false so the traversal
// stops cleanly (DesyncAt) instead of silently mis-aligning — that tells us exactly
// which deser to port next.

// noVariant is the engine sentinel for an unread/absent variant-name.
const noVariant uint32 = 0xFFFFFFFF

// CompResult is one decoded component slot in a record.
type CompResult struct {
	Index    int    // archetype iterator index (= mask bit)
	Name     string // component name from the registry
	Variant  uint32 // variant-name for obje/weapon components (else noVariant)
	Ported   bool   // false => no bit-exact deser; traversal must stop here
	StartBit int
	// Payload porte la VALEUR décodée du composant, pour les seuls composants de
	// captureNames (cf. capture.go) : BodyVitality, ShieldVitality, RespawnTimer,
	// RoundTimer. nil partout ailleurs — le décodeur reste un sauteur de bits par défaut.
	Payload any
}

// EntityTrace is the result of traversing one new-entity record.
type EntityTrace struct {
	TypeIndex   uint32
	DefaultBits int
	Gate        bool
	Mask        uint64
	Comps       []CompResult
	Dead        *DeadState // captured object-dead-state heavy form (nil if no dead-state component present)
	DesyncAt    int        // iterator index of the first un-ported present component (-1 if all consumed)
	EndBit      int
}

// bipedDefaultStateTypeIndex is the keyframe typeIndex of the biped archetype
// (#35), the one whose default-state is deserialised by FUN_140F44C38
// (consumeBipedDefaultState). For this typeIndex TraverseEntity uses the bit-exact
// deser instead of a fixed Skip(defaultStateBits).
const bipedDefaultStateTypeIndex = 35

// objectArchetypeCount est le nombre d'archétypes OBJET enregistrés dans la table ECS du .exe
// (DAT_144e61d88, peuplée par le registrar FUN_140e453b4 : ti 0x00..0x31, COUNT=0x32=50). Un record
// NEW avec R(6) typeIndex >= 50 est IMPOSSIBLE (FUN_1408f1aa4 retourne erreur 3 sur descripteur NULL).
// ⇒ garde-rail : un ti >= 50 est la PREUVE d'une désync (curseur en zone de données), pas un vrai record.
// (Découverte autoritative .exe, 2026-07-03.)
const objectArchetypeCount = 50

// LE DEFAULT-STATE DU BIPEDE (typeIndex==35) passe par le déser bit-exact
// consumeBipedDefaultState (FUN_140F44C38) pour la part qu'il couvre réellement, puis saute
// le résidu mesuré jusqu'à defaultStateBits. Voir consumeBipedDefaultState / default_state.go.
//
// MEASURED FACT (cmd/tmp_defstate_measure, cmd/tmp_mpp): on the Hydra record
// FUN_140F44C38 (vtable[0x60], the ONLY vtable slot that receives the bitreader RSI
// — confirmed by asm at FUN_141f86704 @141f868c2 `MOV R9, RSI`) consumes 120 bits of
// the 380-bit default-state, NOT 32. The earlier 32-bit figure was wrong because the
// port treated FUN_14080cfe8 (the object-multiplayer-properties block) as 0 bits;
// the asm at FUN_140F44C38 @140f44d0e (`MOV RDX, RDI` = bitreader) proves it consumes
// the stream. consumeMultiplayerPropertiesBlock now ports it bit-exact (+88 bits).
//
// The remaining 260 bits are read by FUN_140F44C38 leaf paths whose widths are
// populated at map-load from the film's replication/precision config (DAT_1445cc9e0
// per-axis widths, DAT_144632be0 index width, DAT_145121140) and read 0 statically —
// the same limitation already affecting the traversal PrecisionDescriptor. On
// the calibration record those config-gated paths form a regular 96-bit block (a
// quantized vec3/quat, "0x3FC,0" x4 @bit194256) plus dense words: NOT a free loop,
// but driven by runtime widths not recoverable from the .exe. vtable[0x88] gets no
// bitreader and the FUN_141f86704 config tail is <=2 bits, so they cannot host it.
// Until those runtime widths are sourced from the film header, the 120-bit bit-exact
// prefix runs and the residue is skipped to preserve the calibrated total.
//
// LA BASCULE `useBipedDefaultStateDeser` A DISPARU le 2026-09-05 (lot E, item E.2) : le
// drapeau valait `true` et son setter n'avait aucun appelant, donc le chemin « legacy
// pure-Skip » était inatteignable. Le déser bit-exact est désormais le seul chemin du
// typeIndex 35, ce qu'il était déjà en fait.

// TraverseEntity decodes one new-entity record header (R6 typeIndex + default-state
// + R1 gate + presence mask) and walks the archetype's present components via
// consumeByName. It stops at the first un-ported present component (DesyncAt). The
// held-weapon variant is captured when a weapon-state-type-info slot is reached.
//
// For the biped archetype (typeIndex==35) the leading part of the default-state is
// deserialised bit-exact via consumeBipedDefaultState (FUN_140F44C38) + the
// config-gated FUN_141f86704 tail; any residue up to defaultStateBits is skipped
// (the residue deser is not yet identified — see the note above consumeBipedDefaultState's
// call site). For other archetypes the legacy fixed Skip(defaultStateBits) is kept.
func TraverseEntity(br *BitReader, reg *Registry, defaultStateBits int) EntityTrace {
	t := EntityTrace{DesyncAt: -1, DefaultBits: defaultStateBits}
	t.TypeIndex = uint32(br.ReadBits(6))
	if t.TypeIndex >= objectArchetypeCount {
		// typeIndex objet cappé < 50 (.exe) : au-delà = désync (curseur en zone de données), pas un record.
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	if t.TypeIndex == bipedDefaultStateTypeIndex {
		// VALIDÉ BIT-EXACT en live (CE breakpoint sur FUN_140f44c38 : rep biped = 166 ou 198
		// bits selon la donnée ; consumeBipedDefaultState consomme EXACTEMENT 198 sur le record
		// capturé). Le default-state = la longueur EXACTE du deser, PAS un skip vers le "380"
		// (qui était un faux-propre incluant des bits de la boucle de composants). Le résidu-skip
		// historique est SUPPRIMÉ.
		consumeBipedDefaultState(br)     // FUN_140F44C38, self-délimité, bit-exact
		consumeBipedDefaultStateTail(br) // FUN_141f86704 config-gated tail (default 0)
		// LA SURCHARGE DE CALIBRATION `defaultStateBitsByTI` A DISPARU ICI le 2026-09-05
		// (lot E, item E.2) : la table n'etait peuplee que par `SetDefaultStateBitsForTI`,
		// un reglage sans appelant. Elle restait vide, la branche etait inatteignable.
	} else if fn, ok := defaultStateDeserByTI[t.TypeIndex]; ok && br.p.Grammaire.DeserEtatParArchetype {
		fn(br) // deser vtable[0x60] porté bit-exact (cf. default_state_arch.go)
	} else {
		br.Skip(defaultStateBits) // fallback : stub 0-bit (défaut) ou largeur globale de calibration
	}
	// t.Gate : R(1) réel AVANT le masque dans le record NEW (pré-boucle, cf FUN_1408f1aa4).
	// N'existe PAS dans le path DELTA (decodeDelta appelle consumeMask seul). Le retirer casse
	// le décodage (désync) — c'est un vrai bit, pas un double-gate.
	t.Gate = br.ReadBit()
	t.Mask = consumeMask(br)

	arch, ok := reg.Archetype(int(t.TypeIndex))
	if !ok {
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	traverseComponentLoop(br, arch, &t)
	if t.DesyncAt == -1 && br.p.Grammaire.BitsDeQueueRecordNew > 0 {
		br.Skip(br.p.Grammaire.BitsDeQueueRecordNew) // queue terminale d un record NEW (calibration image-cle)
	}
	t.EndBit = br.BitPos()
	return t
}

// traverseComponentLoop walks an archetype's ORDERED component list and decodes each
// present component (mask bit set) via consumeByName, in iteration order. It is the
// shared body between the keyframe/NEW path (TraverseEntity, which prepends the
// R6 typeIndex + default-state + gate header) and the FRAME delta path (decodeDelta,
// which has NO header — just mask + components). Stops cleanly at the first un-ported
// present component (DesyncAt) and captures the held-weapon variant when reached.
// LES SIX BASCULES DE GRAMMAIRE DE CE FICHIER ONT QUITTE LE PAQUET AU LOT 2.3
// (`filmComponentCorruptionCheck`, `newRecordTailBits`, `useArchDefaultStateDeser`,
// `simStateComplete`, `calibratedWidth`, `unportedStubWidth`, avec leurs six reglages publics).
// Elles vivent dans [GrammaireBalayage], que le lecteur de bits porte : un instrument qui en
// pose une la pose pour SON balayage, plus pour le processus. Leur provenance et leur defaut
// sont ecrits a leur champ.

// LA SURCHARGE DE CALIBRATION `defaultStateBitsByTI` A DISPARU le 2026-09-05 (lot E, item E.2),
// avec son enregistreur `SetDefaultStateBitsForTI` et les deux branches qu'elle gardait
// (`TraverseEntity` et la sonde `TraverseKeyframeBipedAt`) : le setter n'avait aucun appelant,
// la table restait donc vide et les deux branches étaient inatteignables. Les archétypes
// non-biped passent, comme avant, par leur déserialiseur porté (`defaultStateDeserByTI`) ou par
// le repli `br.Skip(defaultStateBits)`.

// consumeCorruptionCheck lit le sentinel per-composant du mode film : R(1) garde ; si 1, R(32).
func consumeCorruptionCheck(br *BitReader) {
	if br.p.Grammaire.ControleDeCorruption && br.ReadBit() {
		br.ReadBits(32) // sentinel attendu 0xbcddcba
	}
}

// consumeSimStateHandleTail porte FUN_14076e494(br, dst, LEVEL=0x10, 0, 0, param_6=0) — la
// QUEUE d'i60, RÉSOLUE le 2026-08-17 (lot R7-b) après avoir été longtemps portée « largeur
// inconnue, désync propre ».
//
//	cVar1 = FUN_14076f91c()   garde RUNTIME (DAT_144e61ea0 / DAT_145121140), 0 bit
//	                          = `BitReader.fullPrecision`, déjà modélisée ici.
//	cVar1 != 0 : FUN_1411b259c -> FUN_1406d676c(br, br, dst, 0x60)   = R(96) brut.
//	cVar1 == 0 (retail, dominant) : FUN_14076e524(dst, br, idxOut, LEVEL=0x10) =
//	          R(1) porte d'index ; si 0 -> R(DAT_144632be0) index de région ;
//	          puis FUN_140cc5128 = 3 axes aux largeurs de la ligne LEVEL=16.
//
// C'est EXACTEMENT le lecteur absolu de `consumeAbsoluteWithGate`, MOINS son bit precHigh
// (ici la garde est runtime, pas un bit du flux) et MOINS son R(2) « fini » de queue — que
// FUN_14076e494 n'appelle pas.
func consumeSimStateHandleTail(br *BitReader) {
	if fullPrecisionGate(br) { // FUN_14076f91c vrai -> copie brute
		br.ReadBits(rawVec3Bits) // FUN_1406d676c(..., 0x60)
		return
	}
	idx := -1
	if !br.ReadBit() { // FUN_14076e524 : porte d'index ; 0 -> lit l'index de région
		idx = int(br.ReadBits(br.worldObjectPrecision().IndexW))
	}
	for i := 0; i < 3; i++ {
		br.ReadBits(absAxisWFor(br, idx, i)) // FUN_140cc5128 axe i
	}
}

// consume140c1e79c porte FUN_140c1e79c (direction+magnitude d'i60) :
//
//	R(1) gate ; si bit==0 -> R(19) packed dir (FUN_1406d8288 dequant, 0 bit)
//	PUIS TOUJOURS R(8) magnitude (FUN_1406d84b4 width=8 @140c1e80f ; FUN_1406d8678 dequant).
//
// CONFIRMÉ via disasm : le bloc LAB_140c1e7f2 charge `MOV dword [RSP+0x20],0x8` avant
// CALL 1406d84b4. L'ancien port oubliait cette magnitude R(8) (lumpée dans simStateExtra).
func consume140c1e79c(br *BitReader) {
	if !br.ReadBit() { // gate==0 -> packed dir
		br.ReadBits(19)
	}
	br.ReadBits(8) // magnitude R(8)
}

// consumeSimulationState porte i60 (FUN_142ED6D88, vérifié via décompile) :
//
//	R(1) flag (FUN_1406cf008) ; si 0 -> FUN_14058c250 (0 bit).
//	si 1 : 2×FUN_1407f2058 (R(1)[R5]) + 4×FUN_142ee2194 (R16) + 2×R(2) inline
//	       + 4×FUN_142ee2194 (R16) + FUN_140c1e79c (R1[R19]+R8)
//	       + queue : FUN_140501798 predicate (0 bit) puis FUN_14076e494.
//
// LE PREDICAT EST VRAI PAR CONSTRUCTION (établi le 2026-08-17, lot R7-b), et c'est ce qui
// débloque la queue. `FUN_140c1e79c(br, ?, out=RBP+0x2c, dir=RBP+0x38)` (disasm @142ed6f9b)
// décode une DIRECTION unitaire en `+0x38` puis appelle `FUN_1406d8678(dir, angle, out)`,
// qui construit en `+0x2c` un vecteur PERPENDICULAIRE à la direction (produit vectoriel avec
// un vecteur de base, normalisé, puis rotation de Rodrigues autour de `dir` par l'angle R(8),
// et normalisation finale FUN_1404fec88). `FUN_140501798(+0x2c, +0x38)` teste ensuite
// ‖v1‖²≈1, ‖v2‖²≈1 et v1·v2≈0 — constantes lues dans le binaire : DAT_143cd8374 = 1.0,
// DAT_143cd8370 = 0.0, tolérance DAT_143cd84bc = 1e-3. Une base orthonormée construite
// satisfait les trois : la queue est donc LUE, elle n'est pas conditionnelle en pratique.
func consumeSimulationState(br *BitReader) {
	if !br.ReadBit() { // FUN_1406cf008 flag ; 0 -> 0 bit
		return
	}
	consumeGate0R(br, 5) // FUN_1407f2058 #1
	consumeGate0R(br, 5) // FUN_1407f2058 #2
	for i := 0; i < 4; i++ {
		br.ReadBits(16) // FUN_142ee2194 = R(16)
	}
	br.ReadBits(2) // inline R(2) #1
	br.ReadBits(2) // inline R(2) #2
	for i := 0; i < 4; i++ {
		br.ReadBits(16) // FUN_142ee2194 = R(16)
	}
	consume140c1e79c(br)          // FUN_140c1e79c = R(1)[R19]+R8
	consumeSimStateHandleTail(br) // FUN_14076e494, predicat vrai par construction
}

func traverseComponentLoop(br *BitReader, arch Archetype, t *EntityTrace) {
	traverseComponentLoopFrom(br, arch, t, 0)
}

// traverseComponentLoopFrom walks the component loop starting at index `from` —
// the resume path of component-width inference (frame_chain_infer.go), which skips
// a failed component by a candidate width and re-decodes the remainder.
func traverseComponentLoopFrom(br *BitReader, arch Archetype, t *EntityTrace, from int) {
	for i := from; i < len(arch.Components); i++ {
		if t.Mask&(uint64(1)<<(uint(i)&63)) == 0 {
			continue // component absent from the mask: NO bits consumed.
		}
		start := br.BitPos()
		// CALIBRATION OVERRIDE: runtime-precision components (i0 position, i21 aiming,
		// ...) read per-axis widths populated at map-load and absent from the static
		// .exe, so their ported desers consume the wrong bit count in the delta
		// predicted-precision path. When a CE delta capture has measured the real
		// per-component width (CONSTANT per map), skip by it instead of calling the
		// buggy deser. The dead-state is intentionally NOT calibrated (must be decoded).
		// i5 object-shield-vitality était cité ici À TORT (corrigé le 2026-07-26) :
		// FUN_140d50cbc n'utilise que des largeurs littérales 8/16/12 et des bornes
		// .rdata constantes — il ne dépend d'aucune précision runtime.
		if w, ok := br.p.Grammaire.largeurCalibree(arch.Components[i]); ok {
			br.Skip(w)
			consumeCorruptionCheck(br)
			t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Ported: true, StartBit: start})
			continue
		}
		variant, dead, payload, ported := consumeByNameCapturing(br, arch.Components[i], t.TypeIndex, arch.Level(i))
		if dead != nil {
			t.Dead = dead
		}
		if !ported {
			// CALIBRATION HOOK: an un-ported component normally desyncs (we can't trust
			// the bit position past a component whose width we don't know). If a probe has
			// set a stub width for this component name, consume that many bits and keep
			// going — lets a harness brute-force a missing tail deser's width by record-
			// chaining (the only un-ported component on the delta-biped path is i63
			// biped-action-component, the LAST component, AFTER the weapon). Default: off.
			if w, ok := br.p.Grammaire.largeurBouchon(arch.Components[i]); ok {
				br.Skip(w)
				consumeCorruptionCheck(br)
				t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Variant: variant, Ported: true, StartBit: start})
				continue
			}
			t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Variant: variant, Ported: false, StartBit: start})
			t.DesyncAt = i
			return
		}
		consumeCorruptionCheck(br)
		t.Comps = append(t.Comps, CompResult{Index: i, Name: arch.Components[i], Variant: variant,
			Ported: ported, StartBit: start, Payload: payload})
	}
}

// consumeMask mirrors FUN_1406d7610: R(1) gate ; if 0 -> R(3) count + count×R(6)
// index (sparse set) ; if 1 -> R(64) dense mask.
func consumeMask(br *BitReader) uint64 {
	if !br.ReadBit() {
		count := uint32(br.ReadBits(3))
		var mask uint64
		for i := uint32(0); i < count; i++ {
			idx := uint(br.ReadBits(6))
			mask |= uint64(1) << (idx & 63)
		}
		return mask
	}
	return br.ReadBits(64)
}
