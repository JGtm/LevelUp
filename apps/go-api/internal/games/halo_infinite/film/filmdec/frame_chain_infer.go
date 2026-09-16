package filmdec

import "sort"

// Recursive CHAIN inference for unbound-slot deltas.
//
// Single-step inference (inferUnboundArchetype) fails on CHAINS of transients: its
// confirmation (next record = clean delta on a bound slot) never comes when the next
// record is itself an unbound transient. The chain resolver explores the record stream
// RECURSIVELY: at each unbound-slot delta it forks on every archetype whose body
// decodes cleanly — deduped by body END BIT, because only the resulting alignment
// matters for skipping a transient, not which archetype produced it — and a path
// CONFIRMS when it reaches a clean, >=1-component delta on a HARD-bound slot (a
// world_dump binding or a clean main-loop NEW; SOFT inferred bindings are never
// anchors, so a wrong inference cannot self-confirm). The first transient's
// resolution is accepted iff exactly ONE distinct body-end leads to a confirmed
// path — ambiguity desyncs, preserving the zero-false-positive property of the
// single-step version.
//
// Mid-chain NEW records are attempted only for the biped archetype (its default-state
// deser is bit-exact); any other NEW fails the path: with NewDefaultStateBits=0 a
// non-biped NEW is a misparse and would be a false-clean generator.

// Constants calibrated on the Cliffhanger film (tmp_chaindbg / tmp_traj sweeps); the
// rationale for each is inline. They are constants, not knobs: the sweeps that chose
// them are done.
const (
	chainMaxDepth    = 6      // max unbound-delta inference levels along one path
	chainMaxRecords  = 24     // max records traversed along one path
	chainTrialBudget = 200000 // max archetype body trials per top-level call
	// A DEEP-tier confirming delta must carry >=1 present component: an empty 4-bit
	// delta is too weak an anchor (it decodes clean at far too many offsets).
	chainConfirmMinComps = 1
)

// ChainRepairedCount returns how many desynced records component-width inference
// rescued (diagnostics). Compteur de l OBSERVATEUR depuis le lot 2.2.f.

// CompWidthObservations returns the accumulated stub-width observations per
// component name (width -> occurrences). Table de l OBSERVATEUR depuis le lot 2.2.f — c est
// « la table sans verrou » que l en-tete de `decode_gate.go` nomme.

// chainCompMaxStub bounds the stub-width sweep of component repair (bits).
const chainCompMaxStub = 640

// repairUnportedComponent rescues a record that desynced on an un-ported (or
// partially-ported) component: it sweeps the component's EXTRA stub width (the
// unportedStubWidth mechanism, so partial consumers like simulation-state keep their
// known prefix), keeps the widths whose full-record redecode is clean, and accepts the
// unique downstream-confirmed alignment via resolveAlignment. On success the record is
// re-decoded WITH capture hooks (so position/velocity samples of the components after
// the repaired one are emitted) and the clean trace + end bit are returned.
//
// tr is the failed trace (its last component entry names the culprit). recType selects
// the redecode path (recNew -> TraverseEntity, recDelta -> decodeDelta on slot). The
// whole repair runs with capture hooks suppressed: the components BEFORE the failure
// already emitted their samples during the original decode (re-emitting would
// duplicate them), and on the biped path position/velocity (i0/i1) always precede the
// repairable components (i55+), so nothing positional is lost after the stub.
func repairUnportedComponent(buf []byte, bodyStart, recType int, slot uint32, tr EntityTrace, w *World, cfg FrameConfig) (EntityTrace, int, bool) {
	if len(tr.Comps) == 0 || tr.Comps[len(tr.Comps)-1].Ported {
		return tr, 0, false
	}
	name := tr.Comps[len(tr.Comps)-1].Name
	if _, preset := cfg.Profil.Grammaire.largeurBouchon(name); preset {
		return tr, 0, false // an external harness already stubs it; do not fight it
	}
	defer cfg.Obs.neutraliserCaptures()()

	// LA TABLE DE BOUCHONS EST CELLE DE CE BALAYAGE (lot 2.3) : `repairUnportedComponent`
	// essaie des largeurs sur SON cadre, et ce cadre meurt avec lui. Avant, la table etait une
	// variable de paquet que chaque essai ecrivait — donc un etat que tout le processus voyait.
	cfg.Profil.Grammaire.LargeursBouchon = map[string]int{}
	frameLen := len(buf) * 8
	redecode := func() EntityTrace {
		br := NewBitReader(buf)
		br.poserCadre(cfg)
		br.Skip(bodyStart)
		if recType == recNew {
			return TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
		}
		return decodeDelta(br, w, slot)
	}

	byEnd := map[int][]int{}
	var order []int
	for stub := 0; stub <= chainCompMaxStub; stub++ {
		cfg.Profil.Grammaire.LargeursBouchon[name] = stub
		t := redecode()
		if t.DesyncAt != -1 || t.EndBit > frameLen {
			continue
		}
		if _, dup := byEnd[t.EndBit]; !dup {
			order = append(order, t.EndBit)
		}
		byEnd[t.EndBit] = append(byEnd[t.EndBit], stub)
	}
	delete(cfg.Profil.Grammaire.LargeursBouchon, name)
	sort.Ints(order)
	if len(order) == 0 {
		return tr, 0, false
	}

	c := &chainCtx{buf: buf, frameLen: frameLen, w: w, cfg: cfg,
		budget: chainTrialBudget, overlay: map[uint32]uint32{}}
	winEnd, _, ok := resolveAlignment(c, order)
	if !ok {
		return tr, 0, false
	}
	stub := byEnd[winEnd][0]
	cfg.Profil.Grammaire.LargeursBouchon[name] = stub
	t := redecode()
	delete(cfg.Profil.Grammaire.LargeursBouchon, name)
	if t.DesyncAt != -1 || t.EndBit != winEnd {
		return tr, 0, false
	}
	cfg.Obs.compterReparation(name, byEnd[winEnd])
	return t, winEnd, true
}

// chainTombstone marks a slot DELed earlier along the current path.
const chainTombstone = ^uint32(0)

// Les compteurs d issue de l inference de chaine sont des champs de l OBSERVATEUR depuis le lot
// 2.2.f, et ils appartiennent au BALAYAGE depuis le lot 2.3 : `ChainStats` / `ResetChainStats`,
// qui lisaient et remettaient a zero ceux du PROCESSUS, ont disparu avec lui — un instrument lit
// desormais les compteurs de l observateur qu il a pose.

// deltaBodyTrial decodes a delta BODY (mask + components) at bit `bitpos` with an
// explicit archetype, STRICTLY: the traversal must be clean and the body must not run
// past the frame (ReadBits past the end yields zeros and would fabricate clean
// decodes). Returns the end bit and the present-component count.
//
// A mask-range check (reject masks addressing components beyond the archetype's count)
// was tried and dropped: it measurably rejected TRUE alignments (tmp_chaindbg: 717 of
// 902 single-step wins lost to it). traverseComponentLoop already ignores mask bits
// beyond len(arch.Components), so an over-wide mask is harmless, not a misparse tell.
func deltaBodyTrial(buf []byte, bitpos int, arch Archetype, ti uint32, frameLen int,
	ctx ContexteDeLecture) (end, comps int, ok bool) {
	br := NewBitReader(buf)
	br.PoserContexte(ctx)
	br.Skip(bitpos)
	t := EntityTrace{DesyncAt: -1, TypeIndex: ti}
	t.Mask = consumeMask(br)
	traverseComponentLoop(br, arch, &t)
	if t.DesyncAt != -1 || br.BitPos() > frameLen {
		return 0, 0, false
	}
	return br.BitPos(), len(t.Comps), true
}

// chainCtx carries the shared state of one top-level chain resolution. overlay holds
// path-local bindings (mid-chain biped NEWs) and tombstones (mid-chain DELs); entries
// are undone on backtrack, the World itself is never mutated during trials. minComps
// is the present-component floor a confirming bound delta must carry (0 for the
// immediate tier — parity with boundDeltaCleanAt — and chainConfirmMinComps for the
// deep tier, where empty 4-bit deltas would be too weak an anchor).
type chainCtx struct {
	buf      []byte
	frameLen int
	w        *World
	cfg      FrameConfig
	budget   int
	minComps int
	// needConfirms is how many hard-bound confirmations a walk must collect before
	// returning true (1 normally; raised to break ties between candidate alignments —
	// a wrong alignment rarely survives several substantive hard-bound deltas).
	needConfirms int
	overlay      map[uint32]uint32
}

// cleanBodyEnds returns the DISTINCT end bits of all archetypes whose body decodes
// cleanly at `body`, ascending. Deduping by end collapses the archetype fan-out to the
// handful of alignments that actually differ.
func (c *chainCtx) cleanBodyEnds(body int) []int {
	seen := map[int]bool{}
	var ends []int
	for ti := range c.w.Reg.Archetypes {
		if c.budget <= 0 {
			break
		}
		c.budget--
		end, _, ok := deltaBodyTrial(c.buf, body, c.w.Reg.Archetypes[ti], uint32(ti), c.frameLen,
			c.cfg.contexte())
		if ok && !seen[end] {
			seen[end] = true
			ends = append(ends, end)
		}
	}
	sort.Ints(ends)
	return ends
}

// confirmChainAt walks records from bit `pos`, forking on unbound-slot deltas, and
// reports whether SOME path reaches a confirmation (clean >=1-component delta on a
// HARD-bound slot). overlay mutations are undone before returning.
func (c *chainCtx) confirmChainAt(pos, depth, recs int) bool {
	if recs <= 0 || c.budget <= 0 || pos >= c.frameLen {
		return false
	}
	br := NewBitReader(c.buf)
	br.poserCadre(c.cfg) // EN TETE (lots 2.2.a et 2.3)
	br.Skip(pos)
	if c.cfg.HasExtraFields {
		br.Skip(32)
	}
	switch readRecordType(br) {
	case recEnd:
		return c.endOfFrameConfirms(br.BitPos())
	case recDel:
		slot := readRecordID(br, c.cfg.IDLowBits, c.cfg.IDBase) & 0x3fffffff
		br.Skip(32) // unconditional R(32) for DEL — carries no discrimination
		if br.BitPos() > c.frameLen {
			return false
		}
		return c.withOverlay(slot, chainTombstone, br.BitPos(), depth, recs)
	case recNew:
		id := readRecordID(br, c.cfg.IDLowBits, c.cfg.IDBase)
		t := TraverseEntity(br, c.w.Reg, c.cfg.NewDefaultStateBits)
		// Only the biped default-state deser is bit-exact; any other NEW would be a
		// misparse with NewDefaultStateBits=0, i.e. a false-clean generator.
		if t.TypeIndex != bipedDefaultStateTypeIndex || t.DesyncAt != -1 || br.BitPos() > c.frameLen {
			return false
		}
		return c.withOverlay(id&0x3fffffff, t.TypeIndex, br.BitPos(), depth, recs)
	case recDelta:
		return c.chainDelta(br, depth, recs)
	}
	return false
}

// endOfFrameConfirms reports whether landing on an end-of-frame marker at bit pos
// (marker already consumed) is accepted as a TERMINAL confirmation: the marker must
// sit flush with the payload end (only zero padding after it). Garbage alignments
// rarely terminate exactly at the payload boundary, so this is strong positional
// evidence — it unlocks chains at the tail of a frame, where no bound delta can
// follow. A flush end satisfies an escalated walk's REMAINING requirement too: it is
// the strongest structural check available (any earlier misalignment cascades and
// virtually never re-lands flush on the marker).
func (c *chainCtx) endOfFrameConfirms(pos int) bool {
	rem := c.frameLen - pos
	if rem < 0 || rem > 15 {
		return false
	}
	br := NewBitReader(c.buf)
	br.Skip(pos)
	for i := 0; i < rem; i++ {
		if br.ReadBit() {
			return false
		}
	}
	return true
}

// withOverlay sets overlay[slot]=v, recurses at pos, and restores the previous entry.
func (c *chainCtx) withOverlay(slot, v uint32, pos, depth, recs int) bool {
	prev, had := c.overlay[slot]
	c.overlay[slot] = v
	ok := c.confirmChainAt(pos, depth, recs-1)
	if had {
		c.overlay[slot] = prev
	} else {
		delete(c.overlay, slot)
	}
	return ok
}

// chainDelta handles a DELTA record inside a chain walk: confirmation on a clean
// HARD-bound delta, neutral continuation on overlay/soft-bound slots, recursion (fork
// by distinct body end) on unbound slots.
func (c *chainCtx) chainDelta(br *BitReader, depth, recs int) bool {
	slot := readRecordID(br, c.cfg.IDLowBits, c.cfg.IDBase) & 0x3fffffff
	body := br.BitPos()
	ti, bound := uint32(0), false
	if ov, inOverlay := c.overlay[slot]; inOverlay {
		if ov == chainTombstone {
			return false // delta on a slot DELed earlier along this path: invalid path
		}
		ti, bound = ov, true
	} else {
		ti, bound = c.w.ArchetypeForSlot(slot)
	}
	if bound {
		arch, ok := c.w.Reg.Archetype(int(ti))
		if !ok {
			return false
		}
		c.budget--
		end, comps, clean := deltaBodyTrial(c.buf, body, arch, ti, c.frameLen, c.cfg.contexte())
		if !clean {
			return false
		}
		if _, inOverlay := c.overlay[slot]; !inOverlay && c.w.HardBound(slot) && comps >= c.minComps {
			// CONFIRMED: clean, substantive delta on a hard-bound slot. Tie-breaking
			// walks require several such confirmations before accepting.
			if c.needConfirms <= 1 {
				return true
			}
			c.needConfirms--
			ok := c.confirmChainAt(end, depth, recs-1)
			c.needConfirms++
			return ok
		}
		return c.confirmChainAt(end, depth, recs-1)
	}
	if depth <= 0 {
		return false
	}
	for _, end := range c.cleanBodyEnds(body) {
		if c.confirmChainAt(end, depth-1, recs-1) {
			return true
		}
	}
	return false
}

// inferChainArchetype resolves the unbound-slot delta whose BODY starts at `bitpos`,
// allowing CHAINS of unbound transients before the confirming hard-bound delta.
//
// Two tiers, by increasing looseness so recall never regresses below single-step:
//  1. IMMEDIATE — the record right after the candidate body is a clean delta on a
//     hard-bound slot (single-step semantics, but deduped by alignment instead of by
//     archetype, so same-width archetype collisions no longer count as ambiguous).
//  2. DEEP — only when tier 1 finds nothing: the recursive walk (fork on unbound
//     deltas, path-local NEW/DEL overlay) must confirm within chainMaxDepth levels.
//
// In both tiers the winner must be the UNIQUE confirmed alignment (ambiguity fails —
// zero fabricated positions). Returns the winning typeIndex (uniqueTi=false when
// several archetypes share the winning alignment — the skip is still exact, but the
// slot must not be soft-bound to an arbitrary pick), the body end bit, and ok.
func inferChainArchetype(buf []byte, bitpos int, w *World, cfg FrameConfig) (ti uint32, end int, uniqueTi, ok bool) {
	defer cfg.Obs.neutraliserCapturePosition()()

	frameLen := len(buf) * 8
	byEnd := map[int][]uint32{}
	var order []int
	for i := range w.Reg.Archetypes {
		e, _, clean := deltaBodyTrial(buf, bitpos, w.Reg.Archetypes[i], uint32(i), frameLen,
			cfg.contexte())
		if !clean {
			continue
		}
		if _, dup := byEnd[e]; !dup {
			order = append(order, e)
		}
		byEnd[e] = append(byEnd[e], uint32(i))
	}
	sort.Ints(order)

	c := &chainCtx{buf: buf, frameLen: frameLen, w: w, cfg: cfg,
		budget: chainTrialBudget, overlay: map[uint32]uint32{}}
	winEnd, immediate, ok2 := resolveAlignment(c, order)
	if !ok2 {
		return 0, bitpos, false, false
	}
	cfg.Obs.compterIssueDeChaine(immediate)
	tis := byEnd[winEnd]
	return tis[0], winEnd, len(tis) == 1, true
}

// resolveAlignment picks the unique confirmed alignment among candidate ends, by
// increasing strictness: tier 1 (immediate bound successor), tier 2 (deep walk), then
// tie-breaking by requiring 2 then 3 hard confirmations along the walk. immediate
// reports whether tier 1 alone resolved it (diagnostics).
func resolveAlignment(c *chainCtx, order []int) (winEnd int, immediate, ok bool) {
	c.minComps, c.needConfirms = 0, 1
	cands := confirmedEnds(c, order, 0, 1)
	if len(cands) == 1 {
		return cands[0], true, true
	}
	if len(cands) == 0 {
		c.minComps, c.needConfirms = chainConfirmMinComps, 1
		cands = confirmedEnds(c, order, chainMaxDepth-1, chainMaxRecords)
		if len(cands) == 1 {
			return cands[0], false, true
		}
		if len(cands) == 0 {
			c.cfg.Obs.compterEchecDeChaine(c.budget <= 0)
			return 0, false, false
		}
	}
	// Tied alignments. Escalate the DISCRIMINATION over the tied set only, one rung
	// at a time: first the deep walk at a single confirmation (kills candidates whose
	// immediate successor confirmed but whose continuation dies), then walks required
	// to collect 2 then 3 confirmations (flush-end counts as final). A rung that
	// eliminates EVERY candidate is skipped rather than taken as evidence (short
	// frames often cannot satisfy it at all).
	c.minComps = chainConfirmMinComps
	if c.minComps < 1 {
		c.minComps = 1
	}
	for nc := 1; nc <= 3 && len(cands) > 1; nc++ {
		c.needConfirms = nc
		next := confirmedEnds(c, cands, chainMaxDepth-1, chainMaxRecords)
		if len(next) > 0 {
			cands = next
		}
	}
	if len(cands) == 1 {
		return cands[0], false, true
	}
	c.cfg.Obs.compterAmbiguiteDeChaine()
	return 0, false, false
}

// confirmedEnds returns the subset of candidate ends whose walk confirms under the
// current ctx parameters.
func confirmedEnds(c *chainCtx, ends []int, depth, recs int) []int {
	var out []int
	for _, e := range ends {
		if c.confirmChainAt(e, depth, recs) {
			out = append(out, e)
		}
	}
	return out
}
