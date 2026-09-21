package grammar

// frame_harvest.go — LE BALAYAGE EXHAUSTIF D UNE TRAME, par opposition au decodage
// sequentiel : `ScanFrameTargets`, `DecodeFrameViews`, `DecodeFrameResync` et les
// confirmations qui decident si un candidat est un vrai record ou une coincidence.
//
// Sorti de `frame_records.go` par deplacement pur au lot 2.7 (scission des fichiers de plus
// de 500 lignes) : aucune ligne de logique n'a change. La frontiere est celle des deux
// manieres de lire une trame — suivre la chaine (frame_records.go) ou la balayer (ici).

// AcceptResyncFunc validates a resync candidate before accepting it. slot is the target
// slot, pos its decoded i0 position (hasPos=false if the record carried no position
// component). Return false to REJECT (keep scanning) — e.g. to drop a candidate whose
// position teleports impossibly from the slot's last known position (a false resync). A
// nil func accepts every clean target delta.
type AcceptResyncFunc func(slot uint32, pos [3]float32, hasPos bool) bool

// scanForTargetDelta scans buf forward from bit `from` for the first position where a
// clean DELTA record on a TARGET slot decodes (TryDeltaAt ok + slot in targets + at least
// one present component) AND passes `accept`. Returns the bit position, or -1 if none. The
// target-slot constraint (8 valid biped ids) + clean traversal + accept keep false resyncs
// rare. A local capture hook grabs each candidate's i0 position for the accept check.
func scanForTargetDelta(buf []byte, from int, w *World, cfg FrameConfig, targets map[uint32]bool, accept AcceptResyncFunc) int {
	frameLen := len(buf) * 8
	var capPos [3]float32
	var capHas bool
	prev := cfg.Obs
	capture := NouvelleObservation()
	if prev != nil {
		*capture = *prev
	}
	capture.PosCaptureHook = func(s PositionSample) {
		if !capHas {
			capPos, capHas = s.Vec, true // first (i0) position of the trial record
		}
	}
	cfg.Obs = capture
	defer func() { cfg.Obs = prev }()
	for b := from; b < frameLen-24; b++ {
		capHas = false
		rec, _, ok := TryDeltaAt(buf, b, w, cfg)
		if ok && targets[rec.Slot] && len(rec.Trace.Comps) >= 1 {
			if accept == nil || accept(rec.Slot, capPos, capHas) {
				return b
			}
		}
	}
	return -1
}

// HarvestConfirm selects how strongly ScanFrameTargets validates a candidate delta.
type HarvestConfirm int

const (
	// HarvestNextBound: the record immediately after the candidate is a clean delta on
	// a hard-bound slot (cheap; single-step). Good false-positive/cost balance.
	HarvestNextBound HarvestConfirm = iota
	// HarvestChainWalk: the chain walker must confirm the continuation (stronger, but
	// pays the recursive walk at every candidate bit — slow).
	HarvestChainWalk
)

// ScanFrameTargets exhaustively scans a FRAME for EVERY clean delta on a target slot
// whose continuation validates, and returns them in stream order. Unlike sequential
// decode (which follows the record chain and loses everything past a desync) this
// finds target records independently of chain position — the way an exhaustive bit
// scan reaches tail bipeds that sequential decode never gets to. Overlapping matches
// (a target delta decoding clean at several nearby bit offsets) are collapsed: once a
// candidate at bit b is accepted, scanning resumes past its end. Position/velocity
// samples fire through the normal hooks during the (final) decode of each accepted
// record. `targets` must be HARD-bound in w (their archetype is resolved from w).
func ScanFrameTargets(buf []byte, w *World, cfg FrameConfig, targets map[uint32]bool, mode HarvestConfirm) []FrameRecord {
	frameLen := len(buf) * 8
	var out []FrameRecord
	c := &chainCtx{buf: buf, frameLen: frameLen, w: w, cfg: cfg,
		budget: 1 << 30, minComps: chainConfirmMinComps, needConfirms: 1,
		overlay: map[uint32]uint32{}}
	b := 0
	for b < frameLen-24 {
		// Suppress hooks AND position accumulation during the trial; re-decode the accepted
		// record with hooks + accumulation live (so only accepted records seed/accumulate the
		// persistent World, not the thousands of speculative trial decodes).
		// L ACCUMULATEUR N EST PLUS A SAUVER (lot 2.3) : il vit sur le LECTEUR, et chaque essai
		// construit le sien. Seul le crochet d observation reste un etat de processus.
		restaure := cfg.Obs.neutraliserCapturePosition()
		rec, after, ok := TryDeltaAt(buf, b, w, cfg)
		confirmed := false
		if ok && targets[rec.Slot] && len(rec.Trace.Comps) >= 1 {
			switch mode {
			case HarvestChainWalk:
				confirmed = c.confirmChainAt(after, chainMaxDepth-1, chainMaxRecords)
			default:
				confirmed = harvestNextBoundClean(buf, after, w, cfg)
			}
		}
		restaure()
		if confirmed {
			rec2, end, _ := TryDeltaAt(buf, b, w, cfg) // re-decode with hooks live -> real samples
			out = append(out, rec2)
			b = end
			continue
		}
		b++
	}
	return out
}

// harvestNextBoundClean reports whether the record right after a candidate (at bit
// `pos`) is a clean delta on a HARD-bound slot OR a flush end-of-frame — the cheap
// confirmation for ScanFrameTargets.
func harvestNextBoundClean(buf []byte, pos int, w *World, cfg FrameConfig) bool {
	frameLen := len(buf) * 8
	br := LecteurSur(buf)
	br.poserCadre(cfg) // EN TETE (lots 2.2.a et 2.3)
	br.Skip(pos)
	if br.Remaining() < 24 {
		rem := frameLen - pos
		if rem < 0 || rem > 15 {
			return false
		}
		for i := 0; i < rem; i++ {
			if br.ReadBit() {
				return false
			}
		}
		return true // flush end-of-frame
	}
	if cfg.HasExtraFields {
		br.Skip(32)
	}
	if readRecordType(br) != recDelta {
		return false
	}
	slot := readRecordID(br, cfg.IDLowBits, cfg.IDBase) & 0x3fffffff
	if !w.HardBound(slot) {
		return false
	}
	t := decodeDelta(br, w, slot)
	return t.DesyncAt == -1 && br.BitPos() <= frameLen && len(t.Comps) >= 1
}

// DecodeFrameViews decodes a type-0 packet as the GAME does (frame-processor
// FUN_142987460, RE'd live): an optional leading config bit, then `nViews` sequential
// record loops (one per replication "view"), each terminated by its own end marker.
// The offline decoder previously read only ONE loop from bit 0 — potentially missing
// views 1..N (where other players may live) and mis-framing the leading bit. Records
// from all views are concatenated. Uses chain inference per view when `Grammaire.InferenceChaine` is on.
// Returns the records and the number of views that decoded before a desync stopped it.
func DecodeFrameViews(buf []byte, w *World, cfg FrameConfig, nViews int, skipLeadBits int) ([]FrameRecord, int) {
	recs, vues, _ := DecodeFrameViewsCurseur(buf, w, cfg, nViews, skipLeadBits)
	return recs, vues
}

// DecodeFrameViewsCurseur est [DecodeFrameViews] qui rend EN PLUS le CURSEUR DE BITS final du
// lecteur — la position ou la marche s est arretee dans le payload.
//
// ELLE EXISTE POUR LE GATE DU LOT 5.11.6, et ce gate est `bits non lus par paquet = le bourrage
// d octet, et rien d autre`. Tant que le curseur n etait pas observable, un decodeur pouvait
// lire 57 bits AU-DELA de la fin du paquet sans que rien ne rougisse : c est exactement ce qui se
// passait avant la garde d eid de la vue (`frame_infer.go`), et aucun compteur du depot ne le
// voyait. Le nombre de records ne suffit pas a fermer une trame ; seul le curseur le fait.
func DecodeFrameViewsCurseur(buf []byte, w *World, cfg FrameConfig, nViews int,
	skipLeadBits int) ([]FrameRecord, int, int) {
	br := LecteurSur(buf)
	br.poserCadre(cfg) // EN TETE (lots 2.2.a et 2.3)
	br.Skip(skipLeadBits)
	frameLen := len(buf) * 8
	var all []FrameRecord
	viewsDone := 0
	for v := 0; v < nViews && br.BitPos() < frameLen-3; v++ {
		w.PoserVueCourante(v) // la table de vue que la garde interrogera (lot 5.11.7)
		start := br.BitPos()
		// Chain inference per view so view 0 decodes past transients to its END marker
		// (else it desyncs and views 1/2 are never reached). hitEnd = reached a clean
		// end-of-records marker; only then does a next view start cleanly after it.
		recs, _, hitEnd := decodeInferLoop(br, buf, w, cfg)
		all = append(all, recs...)
		if !hitEnd {
			return all, viewsDone, br.BitPos() // desync : la frontiere n est pas sure
		}
		viewsDone++
		if br.BitPos() == start { // no progress (immediate end) — stop
			break
		}
	}
	return all, viewsDone, br.BitPos()
}

// DecodeFrameResync decodes a FRAME like DecodeFrameRecords but, on ANY desync (an
// un-decodable unbound-slot delta or an unported component), it SCANS FORWARD for the
// next clean delta on a `targets` slot and resyncs there — recovering target (biped)
// records that follow an un-decodable record without binding the intervening unknown
// entities. Heuristic and heavier (bit-scan per desync); for offline trajectory
// extraction only. Position samples fire through the normal capture hook. `accept` (may
// be nil) rejects false-positive resyncs by validating each candidate's decoded position.
func DecodeFrameResync(buf []byte, w *World, cfg FrameConfig, targets map[uint32]bool, accept AcceptResyncFunc) []FrameRecord {
	var out []FrameRecord
	frameLen := len(buf) * 8
	br := LecteurSur(buf)
	br.poserCadre(cfg) // EN TETE (lots 2.2.a et 2.3)
	guard := 0
	for br.BitPos() < frameLen {
		guard++
		if guard > 4096 { // runaway safety
			return out
		}
		startPos := br.BitPos()
		if cfg.HasExtraFields {
			br.Skip(32)
		}
		typ := readRecordType(br)
		if typ == recEnd {
			return out
		}
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		slot := id & 0x3fffffff
		rec := FrameRecord{Type: typ, ID: id, Slot: slot, DesyncAt: -1}
		desynced := false
		switch typ {
		case recNew:
			rec.Trace = TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
			rec.TypeIndex = rec.Trace.TypeIndex
			rec.DesyncAt = rec.Trace.DesyncAt
			if rec.DesyncAt == -1 {
				w.BindFull(id, rec.TypeIndex)
			} else {
				desynced = true
			}
		case recDel:
			br.Skip(32)
			w.Unbind(slot)
		case recDelta:
			rec.Trace = decodeDelta(br, w, slot)
			rec.TypeIndex = rec.Trace.TypeIndex
			rec.DesyncAt = rec.Trace.DesyncAt
			if rec.DesyncAt != -1 {
				desynced = true
			}
		default:
			desynced = true
		}
		if !desynced {
			out = append(out, rec)
			continue
		}
		// DESYNC — scan forward for the next clean target delta and resync there. The scan
		// trial-decodes every candidate bit, so SUPPRESS the position-capture hook during it
		// (only the accepted record must emit a sample), then restore it.
		restaureHook := cfg.Obs.neutraliserCapturePosition()
		next := scanForTargetDelta(buf, startPos+1, w, cfg, targets, accept)
		restaureHook()
		if next < 0 {
			return out
		}
		rec2, endPos, _ := TryDeltaAt(buf, next, w, cfg) // re-decodes with capture on -> real sample
		out = append(out, rec2)
		br = LecteurSur(buf)
		br.poserCadre(cfg)
		br.Skip(endPos)
	}
	return out
}

// decodeDeltaWithArch decodes a delta body (mask + present components) at br using an
// EXPLICIT archetype (not resolved from the World). Used by unbound-slot archetype
// inference. Mirrors decodeDelta minus the World lookup.
func decodeDeltaWithArch(br *Lecteur, arch Archetype, typeIndex uint32) EntityTrace {
	t := EntityTrace{DesyncAt: -1, TypeIndex: typeIndex}
	t.Mask = consumeMask(br)
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// boundDeltaCleanAt reports whether a DELTA record on an ALREADY-BOUND slot decodes
// cleanly starting at bit `p` — the confirmation signal that an inferred archetype
// aligned the stream (a real bound entity, e.g. a biped, follows the transient).
func boundDeltaCleanAt(buf []byte, p int, w *World, cfg FrameConfig) bool {
	br := LecteurSur(buf)
	br.poserCadre(cfg) // EN TETE (lots 2.2.a et 2.3)
	br.Skip(p)
	if br.Remaining() < 24 {
		return false
	}
	if cfg.HasExtraFields {
		br.Skip(32)
	}
	if readRecordType(br) != recDelta {
		return false
	}
	slot := readRecordID(br, cfg.IDLowBits, cfg.IDBase) & 0x3fffffff
	if _, ok := w.ArchetypeForSlot(slot); !ok {
		return false // successor unbound -> cannot confirm alignment
	}
	t := decodeDelta(br, w, slot)
	return t.DesyncAt == -1
}
