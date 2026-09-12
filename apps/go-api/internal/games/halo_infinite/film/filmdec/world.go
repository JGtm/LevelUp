package filmdec

// World tracks entity-id -> archetype (and the last resolved position) ACROSS FRAME records.
// A FRAME delta (type-3) carries NO typeIndex: it must resolve its archetype from the
// entity created earlier by a NEW record (or by the keyframe init). This mirrors the
// game's runtime datum store (FUN_1406cb5f0: per-id slot table, stride 0xa0, the
// archetype index living at slot+8). We materialize it as a simple per-slot map.
//
// Key = id & 0x3fffffff (the slot index). The full id also carries a 2-bit
// generation/namespace tag in bits 30-31, but the decoder keys ONLY on the slot index:
// a slot's identity is reset whenever a NEW record rebinds it (BindFull/BindSoft rewrite
// the whole slotState, PosValid=false). No separate generation guard is maintained — a
// well-formed stream only changes a slot's occupant via such a rebind.
//
// UN CACHE « ARME EN MAIN » A VECU ICI, ET IL A ETE RETIRE le 2026-08-18 (regle 0 code mort).
// `SetHeldWeapon` / `HeldWeapon` recopiaient la variante `weapon-state-type-info` (i43-i46) de
// chaque record propre dans le slot. Le cache n avait AUCUN appelant, et la mesure qui aurait pu
// lui en donner un l a REFUTE : sur trois films CTF, 68 284 lectures d arme tenue, dont 96 a
// 100 % valent la variante NULLE sur les slots que le pont bipede nomme, et zero occurrence du
// marqueur de portage du drapeau (plan `.ai/V7.5/replay2d/PLAN_OBJECTIFS_VIVANTS_2E_LECTURE.md`,
// phase 0, item 0.2). Le canal du porteur publie par la phase 1 passe par les IMAGES-CLES.
type World struct {
	Reg   *Registry
	slots map[uint32]slotState
}

type slotState struct {
	TypeIndex uint32
	FullID    uint32 // eid COMPLET (generation 2 bits comprise) tel que pose par le NEW / le keyframe
	Soft      bool   // inferred binding (chain inference), not a dump/NEW ground truth
	// GenAny : la GENERATION de cette liaison est inconnue (aucun keyframe ni NEW ne l'a
	// posee ; cf. BindWildcard). Le test strict de generation est alors neutralise pour ce
	// slot — sans quoi la liaison serait inutilisable, l'eid complet du flux ne pouvant pas
	// etre devine.
	GenAny bool
	// Pos / PosValid : dernière position ABSOLUE RÉSOLUE de l'entité (i0 object-position).
	// Le deser i0 (FUN_1406cfe44) réplique la position en DELTA (prev + delta) sur la majorité
	// des frames ; seuls les chemins ABSOLUS (keyframe/predFlag==1/fallback) posent une valeur
	// fraîche. On mémorise donc la position par slot pour accumuler les deltas (sinon un delta
	// isolé n'est pas une coordonnée). PosValid=false = pas encore de seed absolu (trou attendu
	// avant la 1re absolue d'un slot). Un rebind (BindFull/BindSoft) repart PosValid=false,
	// ce qui purge déjà la position d'un slot recyclé — pas de garde de génération séparée.
	Pos      [3]float32
	PosValid bool
}

// NewWorld creates an empty World bound to a parsed archetype registry.
func NewWorld(reg *Registry) *World {
	return &World{Reg: reg, slots: map[uint32]slotState{}}
}

// BindFull registers (or rebinds) the entity created by a NEW record: slot -> archetype.
// L'eid COMPLET (generation en bits 30-31) est memorise : le jeu teste `entry[+0x00] != eid` sur
// l'eid entier avant de lire le corps d'un delta (FUN_1406caad8 -> `return 3`), donc la generation
// est une contrainte de validite gratuite. Elle n'est appliquee que si SetStrictGeneration(true).
func (w *World) BindFull(id, typeIndex uint32) {
	w.slots[id&0x3fffffff] = slotState{TypeIndex: typeIndex, FullID: id}
}

// strictGeneration exige que l'eid COMPLET d'un delta (tag de generation inclus) corresponde a
// celui pose au binding, comme le fait `FUN_1406caad8` (`entry[0] != eid` -> return 3, corps NON
// lu, boucle abandonnee). Defaut false : les bindings issus des keyframes portent une generation
// qui peut avoir change depuis, et le mode strict doit rester un A/B mesurable.
var strictGeneration = false

// SetStrictGeneration active le test sur l'eid complet (generation comprise).
func SetStrictGeneration(v bool) { strictGeneration = v }

// GenerationMatches indique si l'eid complet `id` correspond a celui memorise pour son slot.
// Toujours vrai quand le mode strict est desactive ou quand aucune generation n'a ete memorisee.
func (w *World) GenerationMatches(id uint32) bool {
	if !strictGeneration {
		return true
	}
	s, ok := w.slots[id&0x3fffffff]
	return ok && (s.GenAny || s.FullID == id)
}

// BindWildcard enregistre une liaison slot -> archetype dont la GENERATION est INCONNUE.
// Seul usage legitime : un slot qu'aucun keyframe et aucun record NEW propre ne declare,
// mais dont l'archetype est etabli par une contrainte structurelle externe (par exemple une
// entite creee ET detruite entre deux keyframes, dont la plage de slots et la fenetre de vie
// sont deduites de l'ordre d'allocation des keyframes). Le test strict de generation est
// neutralise pour ce slot, l'eid complet ne pouvant pas etre devine.
func (w *World) BindWildcard(slot, typeIndex uint32) {
	w.slots[slot&0x3fffffff] = slotState{
		TypeIndex: typeIndex, FullID: slot & 0x3fffffff, GenAny: true,
	}
}

// BindSoft registers an INFERRED slot -> archetype binding (chain inference). Soft
// bindings decode subsequent deltas like hard ones, but are NOT confirmation anchors
// for further inference (a soft anchor could self-confirm a wrong chain).
func (w *World) BindSoft(id, typeIndex uint32) {
	w.slots[id&0x3fffffff] = slotState{TypeIndex: typeIndex, FullID: id, Soft: true}
}

// HardBound reports whether slot carries a NON-inferred binding (world dump or clean
// NEW record) — the only bindings chain inference may confirm against.
func (w *World) HardBound(slot uint32) bool {
	s, ok := w.slots[slot]
	return ok && !s.Soft
}

// Unbind removes a slot (DEL record).
func (w *World) Unbind(slot uint32) { delete(w.slots, slot) }

// ArchetypeForSlot returns the archetype typeIndex bound to a slot, if any.
func (w *World) ArchetypeForSlot(slot uint32) (uint32, bool) {
	s, ok := w.slots[slot]
	return s.TypeIndex, ok
}

// SetPos mémorise la position ABSOLUE résolue d'un slot (seed absolu ou prev+delta). No-op si
// le slot n'est pas tracké (un delta sur un slot inconnu n'a pas d'entité à accumuler).
func (w *World) SetPos(slot uint32, v [3]float32) {
	if s, ok := w.slots[slot]; ok {
		s.Pos, s.PosValid = v, true
		w.slots[slot] = s
	}
}

// PosOf retourne la dernière position absolue résolue d'un slot (ok=false si aucun seed encore).
func (w *World) PosOf(slot uint32) ([3]float32, bool) {
	if s, ok := w.slots[slot]; ok && s.PosValid {
		return s.Pos, true
	}
	return [3]float32{}, false
}

// Bound reports how many slots are currently tracked (diagnostics).
func (w *World) Bound() int { return len(w.slots) }

// cloneSlots returns a shallow copy of the slot table (for per-frame rollback).
func (w *World) cloneSlots() map[uint32]slotState {
	m := make(map[uint32]slotState, len(w.slots))
	for k, v := range w.slots {
		m[k] = v
	}
	return m
}

// restoreSlots replaces the slot table (rollback after a desynced frame so a
// partial/garbage NEW/DEL does not corrupt a PERSISTENT World).
func (w *World) restoreSlots(m map[uint32]slotState) { w.slots = m }

// WorldSnapshot is an opaque copy of a World's slot table for rollback across a
// desynced frame (see World.Snapshot / World.Restore). Exported so decode tools can
// run a PERSISTENT World (NEW binds, DEL unbinds across frames, mirroring the game's
// runtime datum store) and roll back only the frames that desync.
type WorldSnapshot struct{ slots map[uint32]slotState }

// Snapshot captures the current slot table (shallow copy) for a later Restore.
func (w *World) Snapshot() WorldSnapshot { return WorldSnapshot{slots: w.cloneSlots()} }

// Restore rewinds the slot table to a prior Snapshot (e.g. to discard a desynced
// frame's partial/garbage NEW/DEL bindings without dropping earlier clean bindings).
func (w *World) Restore(s WorldSnapshot) { w.restoreSlots(s.slots) }
