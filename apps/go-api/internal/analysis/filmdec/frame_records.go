package filmdec

import "fmt"

// FRAME-delta decoder (L3) — port of the type-0 packet record loop FUN_1406cd128 +
// dispatch FUN_1406cbaa0 + delta FUN_141f86b58. Grammar extracted via Ghidra
// (workflow frame-decoder-grammar). See .ai/PLAN_FILM_KILLFEED_V3.md / HANDOFF.
//
// Per-record header (CONFIRMED, the two traps a naive port falls into):
//   - record TYPE is a PREFIX CODE, not a flat R(2):
//       R(1) ; if 1 -> DELTA (1 bit) ; if 0 -> R(2) in {0=end,1=new,2=del,3=delta}.
//   - record ID is table-driven, not a fixed R(7):
//       low = R(idLowBits) + idBase ; tag = R(2) placed in bits 30-31.
//       id = (tag<<30) | (low & 0x3fffffff) ; slot = id & 0x3fffffff.
//
// Per-type body:
//   NEW   (1): [if hasExtraFields && R(1): R(8)] then R(6) typeIndex + default-state
//              + gate + mask + components  == TraverseEntity (default-state IS in stream).
//   DEL   (2): [if hasExtraFields && R(1): R(8)] then R(32).
//   DELTA (3): NO typeIndex, NO default-state (archetype + base state from RAM/World):
//              mask + present components only.
//
// An optional per-record R(32) prefix precedes every record iff hasExtraFields
// (cVar1 = FUN_14076cea8, a global config flag; hypothesised false for offline
// Theater films).

const (
	recEnd   = 0 // type 0: end of record loop
	recNew   = 1 // type 1: new entity (default-state in stream)
	recDel   = 2 // type 2: deletion
	recDelta = 3 // type 3: delta (common case; archetype + base state from World)
)

// FrameConfig holds the parameters that are NOT present in the payload and come from
// the game's runtime context. They must be calibrated empirically (sweep) — see
// open_risks in the spec. Defaults are the starting hypotheses.
type FrameConfig struct {
	HasExtraFields bool // cVar1: per-record R(32) prefix + R(8) NEW/DEL guards. Theater offline: likely false.
	// IDLowBits est la largeur du champ id bas. C'est une valeur de RUNTIME (FUN_1406d3140
	// -> FUN_1406d310c sur DAT_1451f98d0/d4, peuplee au chargement de carte) : elle DIFFERE
	// d'un film a l'autre. 11 sur 000d5950 (calibre) ; 21 bits d'en-tete = idLow 14 sur le
	// film de la capture live (mesure : cmd/tmp_hdrtruth). Le 13 par defaut n'est qu'une
	// hypothese de depart, PAS une constante du format.
	IDLowBits           int
	IDBase              uint32
	NewDefaultStateBits int // default-state width for a NEW record's archetype (runtime; brute-forced).
	// PacketPreambleBits est l'AMORCE DE PAQUET : les bits consommes sur le bitreader du
	// payload AVANT le premier record. Source (desassemblage) : le frame-processor
	// FUN_142987460 fait `DAT_144706104 = FUN_1406cf008(reader)` — un R(1) — puis boucle
	// sur ses 3 vues, chacune appelant vtable[0x40] = FUN_1406cd128 (la boucle de records).
	// Le reader est cree sur le payload EXACT du paquet (FUN_14298816c : memcpy de
	// *(param_2+4) octets, puis FUN_1406d5cc0(reader,3) qui repositionne byte-ptr au debut),
	// donc ce bit est bien le bit 0 du payload. N'est applique que si le lecteur est au
	// bit 0 (un appel en cours de flux n'est pas un debut de paquet).
	PacketPreambleBits int
}

// DefaultPacketPreambleBits est l'amorce de paquet consommee avant le premier record.
//
// POURQUOI 2 ALORS QUE LE DESASSEMBLAGE N'EN MONTRE QU'UN. Le desassemblage etablit UN bit :
// FUN_142987460 fait `DAT_144706104 = FUN_1406cf008(reader)`, et FUN_1406cf008 est un R(1)
// (`*(p+0x2c) += 1`). Ce bit vaut 1 dans 100,00 % des 30 418 payloads de 000d5950 — signature
// d'un drapeau de configuration constant. Le SECOND bit, lui, n'est PAS localise dans le
// desassemblage : il est etabli par la MESURE, et cette distinction doit rester visible.
//
// Trois grammaires donnent le meme en-tete total de 20 bits sur le PREMIER record, et ne se
// separent qu'a partir du second : amorce 2 + idLow 11 · amorce 1 + idLow 12 · amorce 1 +
// prefixe long + idLow 10. Le temoin qui les departage est la FORME DU MASQUE, invariant par
// carte et par film : le compteur de composants tient sur 3 bits, donc un record epars porte
// au plus 7 composants, et la verite terrain (138 390 records bipedes captures sur le
// desassembleur reel) en compte 99,86 % dans cette plage, mode a 4.
//
//	part de masques 1..7 sur les DELTA de slots bindes :
//	  amorce 0, idLow 11 (avant)  : 14,65 %   (n = 33 154)   <- sous le niveau du hasard
//	  amorce 2, idLow 11          : 84,81 %   (n = 33 662)   <- retenu
//	  amorce 1, idLow 12          : 51,67 %   (n =  5 135)
//	  amorce 2, idLow 10          :  5,24 %
//	  niveau du hasard mesure     : 10,67 %   (critere evalue a des decalages arbitraires)
//	  verite terrain              : 99,86 %
//
// NECESSAIRE MAIS PAS SUFFISANT — a ne pas oublier. Apres cette correction, i22 lit encore
// 92,46 % de comptes de grenades impossibles (le compteur doit valoir 4 ; il est uniforme sur
// 0..7). Il reste donc au moins une faute dans le CORPS des records. Cette amorce est gardee
// parce qu'elle repose sur trois temoins independants d'i22 et concordants — le R(1) du
// desassemblage, le bit 0 a 100 %, et la forme des masques — et non parce qu'elle "ameliore"
// un chiffre.
const DefaultPacketPreambleBits = 2

// DefaultFrameConfig is the starting hypothesis for an offline Theater film.
//
// IDLowBits 13 reste une hypothese de depart a calibrer (c'est une valeur de RUNTIME : 11 sur
// 000d5950, 14 sur le film de la capture live). L'amorce, elle, est une propriete du FORMAT et
// non du runtime : elle a donc sa place ici en dur.
func DefaultFrameConfig() FrameConfig {
	return FrameConfig{HasExtraFields: false, IDLowBits: 13, IDBase: 0, NewDefaultStateBits: 0,
		PacketPreambleBits: DefaultPacketPreambleBits}
}

// FrameRecord is one decoded record of a type-0 FRAME packet.
type FrameRecord struct {
	Type      int
	ID        uint32
	Slot      uint32 // ID & 0x3fffffff
	TypeIndex uint32
	Trace     EntityTrace
	DesyncAt  int
	// HeaderBit : position bit (dans le payload) du PREMIER bit de l'en-tete du record.
	// Necessaire pour confronter la marche sequentielle a l'oracle Rosette, dont la colonne
	// recordHeaderBit donne la position VRAIE de chaque record capture : sans elle on ne peut
	// pas localiser le point de rupture de la marche.
	HeaderBit int
}

// readRecordType ports the record-type PREFIX CODE: R(1); set -> DELTA; else R(2).
func readRecordType(br *BitReader) int {
	if br.ReadBit() {
		return recDelta
	}
	return int(br.ReadBits(2))
}

// readRecordID ports FUN_1406d3140(_,_,7,_): low = R(idLowBits)+idBase ; tag = R(2)
// in bits 30-31. The slot (id & 0x3fffffff) indexes the World; the tag is generation.
func readRecordID(br *BitReader, idLowBits int, idBase uint32) uint32 {
	var low uint32
	if idLowBits > 0 {
		low = uint32(br.ReadBits(uint(idLowBits)))
	}
	low += idBase
	tag := uint32(br.ReadBits(2))
	return (tag << 30) | (low & 0x3fffffff)
}

// decodeDelta ports FUN_141f86b58: a delta carries NO typeIndex and NO default-state.
// The archetype is resolved from the World (entity created earlier); the base state
// lives in RAM. The bitstream content is the BASELINE selector + the component mask +
// the present components (the iterator FUN_14076cb60 == consumeMask + the shared loop).
//
// BASELINE (FUN_1406cdc04) : [R(1) ; si 1 -> R(7)] est lu ENTRE le tag d'id et le masque.
// Il manquait a ce port ; sa presence est prouvee contre l'oracle Rosette du film 000d5950
// (mode `oracle` de cmd/tmp_ecsschema) : eid + indices du masque + position de fin d'en-tete
// concordent 54760/54760 AVEC ce bit, et le bit vaut 0 sur 54760/54760 (jamais le R(7)).
func decodeDelta(br *BitReader, w *World, slot uint32) EntityTrace {
	setAccumSlot(slot) // cible d'accumulation i0 pour ce record (no-op si pas de World accumulateur)
	t := EntityTrace{HeldWeapon: noVariant, DesyncAt: -1}
	if br.ReadBit() { // baseline selector
		br.Skip(7)
	}
	typeIndex, ok := w.ArchetypeForSlot(slot)
	if !ok {
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	t.TypeIndex = typeIndex
	arch, ok := w.Reg.Archetype(int(typeIndex))
	if !ok {
		t.DesyncAt = 0
		t.EndBit = br.BitPos()
		return t
	}
	t.Mask = consumeMask(br) // FIRST and only header read of a delta (no R6/default-state/gate).
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// TryDeltaAt attempts to decode a single DELTA record starting at bit `bitpos` of buf,
// resolving the archetype from w. Returns the record, the bit position immediately after
// it, and whether it decoded cleanly (a recDelta header + a fully-traversed delta,
// DesyncAt==-1). Used by the resync scanner (scanForTargetDelta) to relocate the next
// target record after a desync WITHOUT binding the intervening unknown slots. A fresh
// reader is created so the caller's stream position is untouched.
func TryDeltaAt(buf []byte, bitpos int, w *World, cfg FrameConfig) (FrameRecord, int, bool) {
	br := NewBitReader(buf)
	br.Skip(bitpos)
	if br.Remaining() < 24 {
		return FrameRecord{}, bitpos, false
	}
	if cfg.HasExtraFields {
		br.Skip(32)
	}
	typ := readRecordType(br)
	if typ != recDelta {
		return FrameRecord{}, bitpos, false
	}
	id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
	slot := id & 0x3fffffff
	rec := FrameRecord{Type: typ, ID: id, Slot: slot, DesyncAt: -1}
	rec.Trace = decodeDelta(br, w, slot)
	rec.TypeIndex = rec.Trace.TypeIndex
	rec.DesyncAt = rec.Trace.DesyncAt
	return rec, br.BitPos(), rec.DesyncAt == -1
}

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
	prevHook := posCaptureHook
	posCaptureHook = func(s PositionSample) {
		if !capHas {
			capPos, capHas = s.Vec, true // first (i0) position of the trial record
		}
	}
	defer func() { posCaptureHook = prevHook }()
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

// DecodeFrameInfer decodes a FRAME like DecodeFrameRecords but, on a delta for an
// UNBOUND slot (a transient entity absent from the binding dump), it INFERS the slot's
// archetype (inferUnboundArchetype) and skips its body accordingly — decoding the frame
// CORRECTLY past the transient instead of scanning/guessing. This is the offline
// "decode-correctly" alternative to the resync heuristic: no false positives, because it
// only advances when the inference is unambiguous AND confirmed by a clean bound
// successor. Falls back to a desync (returns) when inference is ambiguous/none.
func DecodeFrameInfer(buf []byte, w *World, cfg FrameConfig) ([]FrameRecord, int) {
	br := NewBitReader(buf)
	out, inferred, _ := decodeInferLoop(br, buf, w, cfg)
	return out, inferred
}

// decodeInferLoop is the core of DecodeFrameInfer operating on a SUPPLIED BitReader,
// so several replication "views" of one packet (frame-processor FUN_142987460 = a
// leading config bit then 3 view record-loops) can be decoded in sequence sharing one
// reader. Returns the records, the number of inferred transients, and hitEnd = whether
// it stopped on a clean end-of-records marker (true) vs a desync/EOF (false).
func decodeInferLoop(br *BitReader, buf []byte, w *World, cfg FrameConfig) ([]FrameRecord, int, bool) {
	var out []FrameRecord
	inferred := 0
	frameLen := len(buf) * 8
	guard := 0
	// stall appends the desynced record and attempts a validated resync recovery
	// (inferResyncTargets set). Returns true = stop (unrecoverable desync), false =
	// recovered (br repositioned, keep looping).
	stall := func(startPos int, rec FrameRecord) bool {
		out = append(out, rec)
		if inferResyncTargets != nil {
			if next, ok := validatedResync(buf, startPos+1, w, cfg); ok {
				br.SetBitPos(next)
				return false
			}
		}
		return true
	}
	for br.BitPos() < frameLen {
		guard++
		if guard > 8192 {
			return out, inferred, false
		}
		startPos := br.BitPos()
		if cfg.HasExtraFields {
			br.Skip(32)
		}
		typ := readRecordType(br)
		if typ == recEnd {
			return out, inferred, true
		}
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		slot := id & 0x3fffffff
		rec := FrameRecord{Type: typ, ID: id, Slot: slot, DesyncAt: -1}
		switch typ {
		case recNew:
			bodyStart := br.BitPos()
			rec.Trace = TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
			rec.TypeIndex, rec.DesyncAt = rec.Trace.TypeIndex, rec.Trace.DesyncAt
			repaired := false
			if rec.DesyncAt != -1 && inferChain && inferRepair {
				if t, end, ok := repairUnportedComponent(buf, bodyStart, recNew, slot, rec.Trace, w, cfg); ok {
					rec.Trace, rec.TypeIndex, rec.DesyncAt, repaired = t, t.TypeIndex, -1, true
					br.SetBitPos(end)
				}
			}
			switch {
			case rec.DesyncAt != -1:
				if stall(startPos, rec) {
					return out, inferred, false
				}
				continue
			case repaired:
				w.BindSoft(id, rec.TypeIndex)
			default:
				w.BindFull(id, rec.TypeIndex)
			}
		case recDel:
			br.Skip(32)
			w.Unbind(slot)
		case recDelta:
			if _, bound := w.ArchetypeForSlot(slot); bound {
				bodyStart := br.BitPos()
				rec.Trace = decodeDelta(br, w, slot)
				rec.TypeIndex, rec.DesyncAt = rec.Trace.TypeIndex, rec.Trace.DesyncAt
				if rec.DesyncAt != -1 && inferChain && inferRepair {
					if t, end, ok := repairUnportedComponent(buf, bodyStart, recDelta, slot, rec.Trace, w, cfg); ok {
						rec.Trace, rec.DesyncAt = t, -1
						br.SetBitPos(end)
					}
				}
				if rec.DesyncAt != -1 {
					if stall(startPos, rec) {
						return out, inferred, false
					}
					continue
				}
			} else {
				var (
					ti   uint32
					end  int
					uniq bool
					ok   bool
				)
				if inferChain {
					ti, end, uniq, ok = inferChainArchetype(buf, br.BitPos(), w, cfg)
				} else {
					ti, end, ok = inferUnboundArchetype(buf, br.BitPos(), w, cfg)
				}
				if !ok {
					rec.DesyncAt = 0
					if stall(startPos, rec) {
						return out, inferred, false
					}
					continue
				}
				if inferChain && uniq {
					w.BindSoft(id, ti)
				}
				rec.TypeIndex = ti
				inferred++
				br.SetBitPos(end)
			}
		default:
			rec.DesyncAt = 0
			if stall(startPos, rec) {
				return out, inferred, false
			}
			continue
		}
		out = append(out, rec)
	}
	return out, inferred, false
}

// inferResyncTargets, when non-nil, enables validated-resync recovery in
// DecodeFrameInfer: on an unresolvable desync, scan forward for a landing where a
// clean delta on one of these slots decodes AND the chain walker confirms continued
// clean decoding (a hard-bound clean delta downstream or a flush end-of-frame). This
// reaches biped records stranded in a desync tail with far fewer false positives than
// raw DecodeFrameResync (which accepts any clean target delta). Nil = no recovery.
var inferResyncTargets map[uint32]bool

// SetInferResyncTargets sets (non-nil) or disables (nil) validated-resync recovery.
func SetInferResyncTargets(t map[uint32]bool) { inferResyncTargets = t }

// inferResyncStat counts validated-resync recoveries (diagnostics).
var inferResyncStat int

// InferResyncCount returns the number of validated-resync recoveries performed.
func InferResyncCount() int { return inferResyncStat }

// validatedResync scans buf forward from bit `from` for the first landing where a
// clean delta on a target slot decodes AND the chain walker confirms that decoding
// continues cleanly from just after it (reaching a hard-bound clean delta or a flush
// end-of-frame within the walk budget). Returns the landing bit (the START of the
// confirming target record, so the caller re-decodes it WITH capture hooks live) and
// ok. The confirmation is the same structural test chain inference uses, so a
// coincidental target-slot delta that does not lead to sustained clean decode is
// rejected — the discriminator raw resync lacked.
func validatedResync(buf []byte, from int, w *World, cfg FrameConfig) (int, bool) {
	savedPos, savedDP := posCaptureHook, dynPrecHook
	posCaptureHook, dynPrecHook = nil, nil
	defer func() { posCaptureHook, dynPrecHook = savedPos, savedDP }()

	frameLen := len(buf) * 8
	c := &chainCtx{buf: buf, frameLen: frameLen, w: w, cfg: cfg,
		budget: chainTrialBudget, minComps: chainConfirmMinComps, needConfirms: 1,
		overlay: map[uint32]uint32{}}
	for b := from; b < frameLen-24; b++ {
		rec, after, ok := TryDeltaAt(buf, b, w, cfg)
		if !ok || !inferResyncTargets[rec.Slot] || len(rec.Trace.Comps) < 1 {
			continue
		}
		// The landing itself is a clean hard-bound target delta; require the CONTINUATION
		// to confirm so a coincidental clean delta (followed by garbage) is rejected.
		if c.budget <= 0 {
			return 0, false
		}
		if c.confirmChainAt(after, chainMaxDepth-1, chainMaxRecords) {
			inferResyncStat++
			return b, true
		}
	}
	return 0, false
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
		savedPos, savedDP, savedAcc := posCaptureHook, dynPrecHook, accumWorld
		posCaptureHook, dynPrecHook, accumWorld = nil, nil, nil
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
		posCaptureHook, dynPrecHook, accumWorld = savedPos, savedDP, savedAcc
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
	br := NewBitReader(buf)
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
// from all views are concatenated. Uses chain inference per view when inferChain is on.
// Returns the records and the number of views that decoded before a desync stopped it.
func DecodeFrameViews(buf []byte, w *World, cfg FrameConfig, nViews int, skipLeadBits int) ([]FrameRecord, int) {
	br := NewBitReader(buf)
	br.Skip(skipLeadBits)
	frameLen := len(buf) * 8
	var all []FrameRecord
	viewsDone := 0
	for v := 0; v < nViews && br.BitPos() < frameLen-3; v++ {
		start := br.BitPos()
		// Chain inference per view so view 0 decodes past transients to its END marker
		// (else it desyncs and views 1/2 are never reached). hitEnd = reached a clean
		// end-of-records marker; only then does a next view start cleanly after it.
		recs, _, hitEnd := decodeInferLoop(br, buf, w, cfg)
		all = append(all, recs...)
		if !hitEnd {
			return all, viewsDone // desynced inside this view — cannot trust the boundary
		}
		viewsDone++
		if br.BitPos() == start { // no progress (immediate end) — stop
			break
		}
	}
	return all, viewsDone
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
	br := NewBitReader(buf)
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
		savedHook := posCaptureHook
		posCaptureHook = nil
		next := scanForTargetDelta(buf, startPos+1, w, cfg, targets, accept)
		posCaptureHook = savedHook
		if next < 0 {
			return out
		}
		rec2, endPos, _ := TryDeltaAt(buf, next, w, cfg) // re-decodes with capture on -> real sample
		out = append(out, rec2)
		br = NewBitReader(buf)
		br.Skip(endPos)
	}
	return out
}

// decodeDeltaWithArch decodes a delta body (mask + present components) at br using an
// EXPLICIT archetype (not resolved from the World). Used by unbound-slot archetype
// inference. Mirrors decodeDelta minus the World lookup.
func decodeDeltaWithArch(br *BitReader, arch Archetype, typeIndex uint32) EntityTrace {
	t := EntityTrace{HeldWeapon: noVariant, DesyncAt: -1, TypeIndex: typeIndex}
	t.Mask = consumeMask(br)
	traverseComponentLoop(br, arch, &t)
	t.EndBit = br.BitPos()
	return t
}

// boundDeltaCleanAt reports whether a DELTA record on an ALREADY-BOUND slot decodes
// cleanly starting at bit `p` — the confirmation signal that an inferred archetype
// aligned the stream (a real bound entity, e.g. a biped, follows the transient).
func boundDeltaCleanAt(buf []byte, p int, w *World, cfg FrameConfig) bool {
	br := NewBitReader(buf)
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

// inferUnboundArchetype tries every registered archetype for the unbound-slot delta at
// bit `bitpos`; a candidate qualifies if its body decodes cleanly AND the NEXT record is
// a clean delta on a bound slot (alignment confirmed). Returns the typeIndex + end bit
// iff EXACTLY ONE archetype qualifies (unambiguous), else ok=false. Hooks are suppressed
// during the trials (no spurious samples). This is the offline "decode-correctly" path
// for transient entities absent from the binding dump.
// inferRequireBoundSuccessor gates the strong confirmation (next record = clean bound
// delta). Default true (correct, but blocked by transient CHAINS). Set false to infer on
// unambiguity ALONE — reaches through chains; a wrong choice merely desyncs the frame
// (stops), it does NOT fabricate a biped position (unlike the resync scan).
var inferRequireBoundSuccessor = true

// SetInferStrict toggles the bound-successor confirmation for unbound-slot inference.
func SetInferStrict(v bool) { inferRequireBoundSuccessor = v }

// inferChain routes unbound-slot inference through the recursive CHAIN resolver
// (frame_chain_infer.go), which sees through sequences of transients that block the
// single-step confirmation. Default false (single-step, historical behaviour).
var inferChain = false

// SetInferChain toggles recursive chain inference for unbound-slot deltas.
func SetInferChain(v bool) { inferChain = v }

// inferRepair enables component-width inference (repairUnportedComponent) on records
// that desync on an un-ported component. Off by default: it is expensive and mostly
// rescues non-biped transients (its true value is the per-component width observations
// it accumulates for porting). Requires inferChain.
var inferRepair = false

// SetInferRepair toggles component-width repair for desynced records.
func SetInferRepair(v bool) { inferRepair = v }

func inferUnboundArchetype(buf []byte, bitpos int, w *World, cfg FrameConfig) (uint32, int, bool) {
	saved := posCaptureHook
	savedDP := dynPrecHook
	posCaptureHook, dynPrecHook = nil, nil
	defer func() { posCaptureHook, dynPrecHook = saved, savedDP }()

	var winTi uint32
	winEnd, matches := -1, 0
	for ti := range w.Reg.Archetypes {
		arch := w.Reg.Archetypes[ti]
		br := NewBitReader(buf)
		br.Skip(bitpos) // bitpos = delta body start (mask), already past type+id
		t := decodeDeltaWithArch(br, arch, uint32(ti))
		if t.DesyncAt != -1 {
			continue
		}
		if !inferRequireBoundSuccessor || boundDeltaCleanAt(buf, br.BitPos(), w, cfg) {
			matches++
			winTi, winEnd = uint32(ti), br.BitPos()
			if matches > 1 {
				return 0, bitpos, false // ambiguous
			}
		}
	}
	if matches == 1 {
		return winTi, winEnd, true
	}
	return 0, bitpos, false
}

// DecodeFrameRecords decodes the type-0 FRAME record loop until type==0, maintaining
// the World (entity-id -> archetype, held-weapon cache). Returns the records decoded;
// on a component desync it returns the records so far plus an error (the bit position
// of the next record can no longer be trusted).
func DecodeFrameRecords(br *BitReader, w *World, cfg FrameConfig) ([]FrameRecord, error) {
	var out []FrameRecord
	if cfg.PacketPreambleBits > 0 && br.BitPos() == 0 {
		br.Skip(cfg.PacketPreambleBits) // amorce de paquet (cf. FrameConfig.PacketPreambleBits)
	}
	for {
		hdrBit := br.BitPos()
		if cfg.HasExtraFields {
			br.Skip(32) // optional per-record R(32) prefix, discarded.
		}
		typ := readRecordType(br)
		if typ == recEnd {
			return out, nil // clean end of frame.
		}
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		slot := id & 0x3fffffff
		setAccumSlot(slot) // attribue les samples i0 (et l'accumulation) au slot du record courant
		rec := FrameRecord{Type: typ, ID: id, Slot: slot, DesyncAt: -1, HeaderBit: hdrBit}

		switch typ {
		case recNew:
			if cfg.HasExtraFields && br.ReadBit() {
				br.Skip(8)
			}
			// NEW = R(6) typeIndex + default-state (in stream) + gate + mask + comps.
			rec.Trace = TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
			rec.TypeIndex = rec.Trace.TypeIndex
			rec.DesyncAt = rec.Trace.DesyncAt
			// Bind only on a FULLY clean traversal. Binding on a desyncing NEW (using the
			// R6 typeIndex, which is read before the default-state) was tried and REGRESSED
			// clean frames 18060->6718: the decoder still produces false-cleans (unported
			// components skipped at approximate widths preserve a calibrated total), so a
			// NEW reached after a false-clean is bit-misaligned and its typeIndex binds the
			// slot to a WRONG archetype, overwriting correct world_dump bindings and
			// corrupting all of that slot's future DELTAs. Opportunistic binding is unsafe
			// until the default-state deser is bit-exact (handoff L3 "T3" wall).
			if rec.DesyncAt == -1 {
				w.BindFull(id, rec.TypeIndex)
				if rec.Trace.HeldWeapon != noVariant {
					w.SetHeldWeapon(slot, rec.Trace.HeldWeapon)
				}
			}
		case recDel:
			if cfg.HasExtraFields && br.ReadBit() {
				br.Skip(8)
			}
			br.Skip(32) // unconditional R(32) for DEL.
			w.Unbind(slot)
		case recDelta:
			// Rejet DUR sur l'eid COMPLET (generation comprise) quand le mode strict est actif :
			// FUN_1406caad8 compare entry[+0x00] a l'eid ENTIER et renvoie 3 sans lire le corps,
			// ce qui fait `break` a FUN_1406cd128. Un delta qui echoue ce test est donc la preuve
			// d'une lecture fausse, pas un record a decoder.
			if !w.GenerationMatches(id) {
				rec.Trace = EntityTrace{HeldWeapon: noVariant, DesyncAt: 0, EndBit: br.BitPos()}
				rec.DesyncAt = 0
				break
			}
			rec.Trace = decodeDelta(br, w, slot)
			rec.TypeIndex = rec.Trace.TypeIndex
			rec.DesyncAt = rec.Trace.DesyncAt
			if rec.DesyncAt == -1 && rec.Trace.HeldWeapon != noVariant {
				w.SetHeldWeapon(slot, rec.Trace.HeldWeapon)
			}
		default:
			return out, fmt.Errorf("invalid record type %d at bit %d", typ, br.BitPos())
		}

		out = append(out, rec)
		if rec.DesyncAt != -1 {
			return out, fmt.Errorf("desync record slot=%d typeIdx=%d at component i%d (bit %d)",
				slot, rec.TypeIndex, rec.DesyncAt, br.BitPos())
		}
	}
}
