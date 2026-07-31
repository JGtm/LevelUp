package filmdec

import "fmt"

// DebugRec is one verbose record trace step (diagnostics only).
type DebugRec struct {
	Idx       int
	Type      int
	IDBits    uint32 // raw id low bits as read
	Slot      uint32
	TypeIndex uint32
	StartBit  int
	EndBit    int
	NComps    int
	DesyncAt  int
	IsBiped   bool
	Comps     []CompResult // component breakdown (biped records only)
	Mask      uint64
}

// FrameBipedScan is one clean biped delta found by scanning a frame at arbitrary bit
// offsets — used to see WHERE each player's delta really starts (over-read detection).
type FrameBipedScan struct {
	Bit   int
	Slot  uint32
	End   int
	Comps int
}

// ScanFrameBipeds scans the whole frame for clean biped deltas (non-overlapping),
// returning every landing. If the main loop's first biped record over-reads, the
// swallowed players appear here as landings INSIDE that record's [start,end) range.
func ScanFrameBipeds(buf []byte, w *World, cfg FrameConfig, bipeds map[uint32]bool) []FrameBipedScan {
	savedPos, savedDP := posCaptureHook, dynPrecHook
	posCaptureHook, dynPrecHook = nil, nil
	defer func() { posCaptureHook, dynPrecHook = savedPos, savedDP }()
	frameLen := len(buf) * 8
	var out []FrameBipedScan
	for b := 0; b < frameLen-24; b++ {
		rec, end, ok := TryDeltaAt(buf, b, w, cfg)
		if ok && bipeds[rec.Slot] && len(rec.Trace.Comps) >= 1 {
			out = append(out, FrameBipedScan{Bit: b, Slot: rec.Slot, End: end, Comps: len(rec.Trace.Comps)})
			b = end - 1
		}
	}
	return out
}

// DebugFrameResult captures a full verbose decode of one FRAME plus a probe of the
// bits AFTER the decoder stopped: how many CLEAN biped deltas remain unreached. This
// is the oracle for "does the packet actually contain all players, and am I stopping
// early?". stopReason ∈ {"recEnd","desync","eof","guard"}.
type DebugFrameResult struct {
	Recs           []DebugRec
	StopReason     string
	StopBit        int
	FrameBits      int
	BipedDeltas    int // clean biped deltas the main loop decoded
	AfterStopBits  int // bits left after the stop
	AfterBipeds    int // clean biped deltas found by scanning the bits AFTER the stop
	AfterFirstBit  int // bit of the first clean biped delta after the stop (-1 if none)
	AfterFirstSlot uint32
}

// DebugDecodeFrame decodes a FRAME verbosely (same loop as DecodeFrameRecords, NO chain
// inference), then scans the leftover bits for clean biped deltas — revealing whether
// the packet holds more player records than the main loop reached (i.e. an early/false
// recEnd or an over-reading record swallowing successors). bipeds lists the biped slots
// (from the world dump) to recognise biped records.
func DebugDecodeFrame(buf []byte, w *World, cfg FrameConfig, bipeds map[uint32]bool) DebugFrameResult {
	res := DebugFrameResult{FrameBits: len(buf) * 8, AfterFirstBit: -1}
	br := NewBitReader(buf)
	guard := 0
	for br.BitPos() < res.FrameBits {
		guard++
		if guard > 4096 {
			res.StopReason, res.StopBit = "guard", br.BitPos()
			return res.finishProbe(buf, w, cfg, bipeds)
		}
		startPos := br.BitPos()
		if cfg.HasExtraFields {
			br.Skip(32)
		}
		typ := readRecordType(br)
		if typ == recEnd {
			res.StopReason, res.StopBit = "recEnd", startPos
			return res.finishProbe(buf, w, cfg, bipeds)
		}
		id := readRecordID(br, cfg.IDLowBits, cfg.IDBase)
		slot := id & 0x3fffffff
		d := DebugRec{Idx: len(res.Recs), Type: typ, Slot: slot, StartBit: startPos, DesyncAt: -1, IsBiped: bipeds[slot]}
		switch typ {
		case recNew:
			t := TraverseEntity(br, w.Reg, cfg.NewDefaultStateBits)
			d.TypeIndex, d.DesyncAt, d.NComps = t.TypeIndex, t.DesyncAt, len(t.Comps)
			if t.DesyncAt == -1 {
				w.BindFull(id, t.TypeIndex)
			}
		case recDel:
			br.Skip(32)
			w.Unbind(slot)
		case recDelta:
			t := decodeDelta(br, w, slot)
			d.TypeIndex, d.DesyncAt, d.NComps = t.TypeIndex, t.DesyncAt, len(t.Comps)
			if d.IsBiped {
				d.Comps, d.Mask = t.Comps, t.Mask
			}
		}
		d.EndBit = br.BitPos()
		res.Recs = append(res.Recs, d)
		if d.IsBiped && d.DesyncAt == -1 && d.Type == recDelta {
			res.BipedDeltas++
		}
		if d.DesyncAt != -1 {
			res.StopReason, res.StopBit = "desync", d.EndBit
			return res.finishProbe(buf, w, cfg, bipeds)
		}
	}
	res.StopReason, res.StopBit = "eof", br.BitPos()
	return res.finishProbe(buf, w, cfg, bipeds)
}

// finishProbe scans the bits from StopBit to end for clean biped deltas — the count of
// player records the main loop FAILED to reach.
func (res DebugFrameResult) finishProbe(buf []byte, w *World, cfg FrameConfig, bipeds map[uint32]bool) DebugFrameResult {
	res.AfterStopBits = res.FrameBits - res.StopBit
	savedPos, savedDP := posCaptureHook, dynPrecHook
	posCaptureHook, dynPrecHook = nil, nil
	defer func() { posCaptureHook, dynPrecHook = savedPos, savedDP }()
	last := -1
	for b := res.StopBit; b < res.FrameBits-24; b++ {
		rec, end, ok := TryDeltaAt(buf, b, w, cfg)
		if ok && bipeds[rec.Slot] && len(rec.Trace.Comps) >= 1 {
			res.AfterBipeds++
			if res.AfterFirstBit < 0 {
				res.AfterFirstBit, res.AfterFirstSlot = b, rec.Slot
			}
			_ = last
			last = end
			b = end - 1 // skip past this match to avoid counting overlaps
		}
	}
	return res
}

// String renders a compact dump.
func (res DebugFrameResult) String() string {
	s := fmt.Sprintf("frame %d bits | stop=%s@%d | biped-deltas-reached=%d | après-stop: %d bits, %d biped-deltas propres (1er @%d slot%d)\n",
		res.FrameBits, res.StopReason, res.StopBit, res.BipedDeltas, res.AfterStopBits, res.AfterBipeds, res.AfterFirstBit, res.AfterFirstSlot)
	for _, d := range res.Recs {
		tag := ""
		if d.IsBiped {
			tag = " <BIPED>"
		}
		s += fmt.Sprintf("  #%2d type=%d slot=%-5d ti=%-2d bits=%d..%d (%d) comps=%d desync=%d%s\n",
			d.Idx, d.Type, d.Slot, d.TypeIndex, d.StartBit, d.EndBit, d.EndBit-d.StartBit, d.NComps, d.DesyncAt, tag)
	}
	return s
}
