package filmdec

// selectorBits is the width of the leading selector in a signed variable-width
// integer; the selector value sel gives the field width w = 8 << sel.
const selectorBits = 2

// signExtendMaxWidth is the exclusive upper bound below which a variable-width
// field is sign-extended (only widths 8 and 16; width 32 is a full int32 and
// width 64 is truncated to its low 32 bits). Mirrors the `iVar4 < 0x20` test in
// FUN_140c18a1c.
const signExtendMaxWidth = 32

// BitReader is a sequential MSB-first big-endian bit reader over a byte buffer.
// It is the behavioral equivalent of the Halo replication reader: the i-th bit
// consumed is bit (7-(i mod 8)) of byte (i div 8).
type BitReader struct {
	buf []byte
	pos int // bit offset of the next bit to read
}

// NewBitReader returns a reader positioned at the first bit of buf.
func NewBitReader(buf []byte) *BitReader { return &BitReader{buf: buf} }

// BitPos returns the bit offset of the next bit to read.
func (b *BitReader) BitPos() int { return b.pos }

// Remaining returns the number of unread bits in the buffer.
func (b *BitReader) Remaining() int { return len(b.buf)*8 - b.pos }

// ReadBits reads n bits (0..64) MSB-first and returns them right-aligned in the
// low n bits. Bits past the end of the buffer read as zero, matching the engine's
// tail padding.
// Lecture par mot de 64 bits (cf. bits_word.go) sur le domaine ou elle coincide avec la
// boucle d'origine : position courante non negative et largeur <= 64. Hors de ce domaine
// — position negative (l'indexation panique, comme avant) ou largeur > 64 (le resultat ne
// garde que les 64 DERNIERS bits lus, et le curseur avance quand meme de n) — la boucle
// d'origine reste seule maitresse.
func (b *BitReader) ReadBits(n uint) uint64 {
	if b.pos >= 0 && n <= 64 {
		r := wordBitsAt(b.buf, b.pos, n)
		b.pos += int(n)
		return r
	}
	return b.readBitsLoop(n)
}

// readBitsLoop est la lecture bit a bit d'origine : elle porte les conventions de bord que
// le chemin par mot ne couvre pas.
func (b *BitReader) readBitsLoop(n uint) uint64 {
	var r uint64
	for i := uint(0); i < n; i++ {
		var bit uint64
		if idx := b.pos >> 3; idx < len(b.buf) {
			bit = uint64(b.buf[idx]>>(7-(uint(b.pos)&7))) & 1
		}
		r = r<<1 | bit
		b.pos++
	}
	return r
}

// ReadBit reads a single bit MSB-first.
func (b *BitReader) ReadBit() bool { return b.ReadBits(1) != 0 }

// Skip advances the read position by n bits without decoding them.
func (b *BitReader) Skip(n int) { b.pos += n }

// SetBitPos moves the read position to an absolute bit offset (used to resync a
// shared reader across replication views without allocating a new BitReader).
func (b *BitReader) SetBitPos(p int) { b.pos = p }

// ReadSignedVarWidth decodes the engine's signed variable-width integer: a 2-bit
// selector sel sets the field width w = 8 << sel (8, 16, 32 or 64); w bits follow
// MSB-first. The result is sign-extended only for w in {8, 16}; w = 32 is a full
// int32 and w = 64 is truncated to its low 32 bits. Mirrors FUN_140c18a1c.
func (b *BitReader) ReadSignedVarWidth() int32 {
	sel := uint(b.ReadBits(selectorBits))
	w := uint(8) << sel
	v := uint32(b.ReadBits(w)) // low 32 bits: w=64 discards the high half, w=32 is a full dword
	if w < signExtendMaxWidth && v>>(w-1)&1 != 0 {
		v |= ^uint32(0) << w // fill the bits above the sign bit
	}
	return int32(v)
}
