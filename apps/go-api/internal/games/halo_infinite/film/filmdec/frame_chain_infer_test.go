package filmdec

import "testing"

// bitWriter is the MSB-first counterpart of BitReader, for crafting synthetic frames.
type bitWriter struct {
	buf []byte
	n   int
}

func (w *bitWriter) bit(b uint64) {
	if w.n/8 >= len(w.buf) {
		w.buf = append(w.buf, 0)
	}
	if b&1 != 0 {
		w.buf[w.n/8] |= 1 << (7 - uint(w.n%8))
	}
	w.n++
}

func (w *bitWriter) bits(v uint64, n int) {
	for i := n - 1; i >= 0; i-- {
		w.bit((v >> uint(i)) & 1)
	}
}

// deltaEmpty writes a DELTA record on slot with an EMPTY presence mask (no components):
// type prefix R(1)=1, id low R(11)+tag R(2)=0, mask gate R(1)=0 + count R(3)=0.
func (w *bitWriter) deltaEmpty(slot uint32) {
	w.bit(1)                 // record type prefix -> DELTA
	w.bits(uint64(slot), 11) // id low (IDLowBits)
	w.bits(0, 2)             // id tag
	w.bit(0)                 // mask gate -> sparse
	w.bits(0, 3)             // sparse count = 0 -> empty mask
}

// end writes an end-of-records marker: R(1)=0 then R(2)=0.
func (w *bitWriter) end() { w.bit(0); w.bits(0, 2) }

// twoEmptyArchReg builds a registry with two DISTINCT empty-component archetypes so the
// decoder machinery (mask + component loop) runs with zero component reads.
func twoEmptyArchReg() *Registry {
	return &Registry{Archetypes: []Archetype{
		{Index: 0, Components: nil, Flags: nil},
		{Index: 1, Components: nil, Flags: nil},
	}}
}

var emptyCfg = FrameConfig{HasExtraFields: false, IDLowBits: 11}

func withChain(on bool, f func()) {
	prev := inferChain
	inferChain = on
	defer func() { inferChain = prev }()
	f()
}

// TestChainImmediateResolvesUnboundThenBound: an unbound-slot delta immediately
// followed by a clean HARD-bound delta is resolved by the immediate tier and skipped,
// so the bound record decodes cleanly and no record desyncs.
func TestChainImmediateResolvesUnboundThenBound(t *testing.T) {
	reg := twoEmptyArchReg()
	w := NewWorld(reg)
	w.BindFull(50, 0) // hard binding, the confirmation anchor

	var bw bitWriter
	bw.deltaEmpty(100) // unbound transient
	bw.deltaEmpty(50)  // hard-bound successor -> confirms
	bw.end()

	ResetChainStats()
	withChain(true, func() {
		recs, inferred := DecodeFrameInfer(bw.buf, w, emptyCfg)
		if inferred != 1 {
			t.Fatalf("inferred = %d, want 1", inferred)
		}
		for _, r := range recs {
			if r.DesyncAt != -1 {
				t.Fatalf("record slot=%d desynced at %d", r.Slot, r.DesyncAt)
			}
		}
		if len(recs) != 2 || recs[1].Slot != 50 {
			t.Fatalf("records = %+v, want [slot100, slot50]", recs)
		}
	})
}

// TestChainNoFalseBindWhenUnconfirmed: an unbound-slot delta with NO bound successor
// and NO flush end-of-frame must NOT resolve — the frame stalls with a desynced record
// and nothing is inferred (the zero-false-positive property).
func TestChainNoFalseBindWhenUnconfirmed(t *testing.T) {
	reg := twoEmptyArchReg()
	w := NewWorld(reg) // NOTHING bound -> no possible confirmation anchor

	var bw bitWriter
	bw.deltaEmpty(100)
	bw.deltaEmpty(101)
	bw.deltaEmpty(102)
	// Pad with 1-bits so the tail never reads as a flush (zero) end-of-frame.
	for i := 0; i < 40; i++ {
		bw.bit(1)
	}

	ResetChainStats()
	withChain(true, func() {
		recs, inferred := DecodeFrameInfer(bw.buf, w, emptyCfg)
		if inferred != 0 {
			t.Fatalf("inferred = %d, want 0 (nothing confirmable)", inferred)
		}
		if len(recs) == 0 || recs[len(recs)-1].DesyncAt == -1 {
			t.Fatalf("expected a trailing desynced record, got %+v", recs)
		}
	})
}

// TestChainFlushEndConfirms: an unbound-slot delta that lands FLUSH on the end-of-frame
// marker resolves via the terminal end-of-frame confirmation (no bound successor
// needed).
func TestChainFlushEndConfirms(t *testing.T) {
	reg := twoEmptyArchReg()
	w := NewWorld(reg)

	var bw bitWriter
	bw.deltaEmpty(100)
	bw.end()
	for bw.n%8 != 0 { // zero-pad to a byte boundary -> flush end
		bw.bit(0)
	}

	ResetChainStats()
	withChain(true, func() {
		_, inferred := DecodeFrameInfer(bw.buf, w, emptyCfg)
		if inferred != 1 {
			t.Fatalf("inferred = %d, want 1 (flush-end confirmation)", inferred)
		}
	})
}

// TestWorldSoftHardBinding: soft bindings decode deltas like hard ones but are NOT
// confirmation anchors (HardBound reports false), so a wrong inference can never
// self-confirm off another inference.
func TestWorldSoftHardBinding(t *testing.T) {
	reg := twoEmptyArchReg()
	w := NewWorld(reg)

	w.BindFull(10, 0)
	w.BindSoft(20, 1)

	if ti, ok := w.ArchetypeForSlot(10); !ok || ti != 0 {
		t.Fatalf("slot10 archetype = (%d,%v), want (0,true)", ti, ok)
	}
	if ti, ok := w.ArchetypeForSlot(20); !ok || ti != 1 {
		t.Fatalf("slot20 archetype = (%d,%v), want (1,true)", ti, ok)
	}
	if !w.HardBound(10) {
		t.Fatalf("slot10 should be hard-bound")
	}
	if w.HardBound(20) {
		t.Fatalf("slot20 is soft-bound; HardBound must be false")
	}
	if w.HardBound(99) {
		t.Fatalf("unbound slot must not be hard-bound")
	}
	// A soft binding re-bound hard by a clean NEW becomes an anchor.
	w.BindFull(20, 1)
	if !w.HardBound(20) {
		t.Fatalf("slot20 rebound hard should be an anchor")
	}
}
