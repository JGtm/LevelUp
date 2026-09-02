package filmdec

// Component registry (ECS archetype schema) parser. The registry lives in the
// film's chunk_00 (zlib-compressed; inflates to ~1.97 MB). It is an array of
// fixed-size archetype blocks; block #N holds the ORDERED component list of
// archetype #N — exactly the order FUN_14076cb60 iterates (and the bit index the
// presence-mask FUN_1406d7610 gates). Verified empirically: block 35 @0x08e300 =
// the BIPED/player archetype (object-position-dynamic-precision at i0, … ,
// weapon-state-type-info ×4 = HELD WEAPON at i43..46, …).
//
// Slot layout (260 bytes): [u32 kind LE][u32 flags LE][name ASCII, NUL-padded].
// Block layout: archetypeBlockSlots slots; the component list is the leading run
// of non-empty-name slots, the rest is zero padding.

import (
	"bytes"
	"encoding/binary"
)

const (
	registrySlotSize    = 260
	archetypeBlockSlots = 64
	archetypeBlockSize  = registrySlotSize * archetypeBlockSlots // 0x4100
)

// Étiquettes de composant citées à plus de deux endroits du décodage. Les autres
// restent en littéral : les centraliser toutes créerait une table d'indirection sans
// lecteur. Le risque que porte une étiquette dupliquée — deux branches qui ne
// consomment pas le même nombre de bits — est déjà couvert par
// TestCaptureConsumesSameBitsAsDispatch (capture_test.go), pas par un test de grep.
const (
	compObjectBodyVitality  = "object-body-vitality-component"
	compWeaponStateTypeInfo = "weapon-state-type-info"
)

// Archetype is one ECS archetype: an ordered list of component names. The slice
// index is the iterator/mask bit index used by the component loop.
type Archetype struct {
	Index      int      // block number = archetype index in the registry
	Components []string // ordered component names (mask bit i -> Components[i])
	// Flags[i] = le champ flags (u32 @ slot+4) du composant i, utilisé comme niveau de
	// précision L (largeur d'axe = quantAxisWidth(L)) par le traverseur générique.
	//
	// CE N'EST PAS la source des largeurs de la position absolue d'un biped (i0) : le
	// registre est BIT-À-BIT IDENTIQUE d'un film à l'autre (FNV des 1067 slots noms+flags
	// = a413610cd08e4355 sur Cliffhanger comme sur Catalyst) alors que les largeurs d'i0
	// changent de carte en carte (13/13/14 vs 15/15/15). Le niveau d'i0 est câblé au site
	// d'appel (MOV R9D,0x10) et les largeurs dérivent des bornes du BSP de la carte.
	// Découpage réel d'i0 : DetectI0Layout (i0_layout.go), lu dans le bitstream.
	Flags []uint32
}

// Level returns the precision level (flags) of component i, or 0 if out of range.
func (a Archetype) Level(i int) uint32 {
	if i < 0 || i >= len(a.Flags) {
		return 0
	}
	return a.Flags[i]
}

// component returns the name at iterator index i, or "" if out of range.
func (a Archetype) component(i int) string {
	if i < 0 || i >= len(a.Components) {
		return ""
	}
	return a.Components[i]
}

// indicesOf returns every iterator index whose component name equals name.
func (a Archetype) indicesOf(name string) []int {
	var out []int
	for i, c := range a.Components {
		if c == name {
			out = append(out, i)
		}
	}
	return out
}

// Registry is the parsed set of archetype blocks from chunk_00.
type Registry struct {
	Archetypes []Archetype
	// fingerprint est l'empreinte FNV-1a des slots non vides, calculee pendant la passe de
	// lecture (registry_fingerprint.go) — la seule qui voie le champ `kind`, que le parse ne
	// retient pas. Se lit par RegistryFingerprint.
	fingerprint uint64
}

// Archetype returns archetype #idx, or (zero, false) if idx is out of range.
func (r *Registry) Archetype(idx int) (Archetype, bool) {
	if idx < 0 || idx >= len(r.Archetypes) {
		return Archetype{}, false
	}
	return r.Archetypes[idx], true
}

// ParseRegistryChunk parses every fixed-size archetype block of an ALREADY-INFLATED chunk_00.
//
// IT NO LONGER INFLATES (lot 1 of PLAN_CUISSON_PERF, 2026-09-02). Decompression happens once per
// film, in `filmsource`: the cooking path hands over `film.Chunk(<registre>)`, and the single-chunk
// readers (research tools, tests) hand over `filmdec.ReadFilmChunk(dir, 0)`, which inflates
// through the same decompressor. Feeding it a still-compressed buffer now yields an EMPTY registry
// rather than an error — callers must inflate first; `internal/archlint` forbids a second
// `zlib.NewReader` inside `filmdec` so the rule cannot silently come back.
//
// The error return is kept: the signature is used in a dozen files, and the parse itself will grow
// error cases (a registry whose block size does not divide the buffer is already suspicious).
func ParseRegistryChunk(data []byte) (*Registry, error) {
	return parseRegistry(data), nil
}

func parseRegistry(data []byte) *Registry {
	reg := &Registry{}
	fp := registryHasher()
	nBlocks := len(data) / archetypeBlockSize
	for b := 0; b < nBlocks; b++ {
		base := b * archetypeBlockSize
		arch := Archetype{Index: b}
		for s := 0; s < archetypeBlockSlots; s++ {
			off := base + s*registrySlotSize
			name := slotName(data, off)
			if name == "" {
				break // start of zero padding -> end of this archetype's list
			}
			arch.Components = append(arch.Components, name)
			arch.Flags = append(arch.Flags, binary.LittleEndian.Uint32(data[off+4:])) // flags @ slot+4 = level
			fp.addSlot(data, off, name)
		}
		reg.Archetypes = append(reg.Archetypes, arch)
	}
	reg.fingerprint = fp.sum()
	warnUnknownRegistry(reg.fingerprint, nBlocks, fp.slots)
	return reg
}

// slotName extracts the NUL-terminated ASCII name at slot offset off+8.
func slotName(data []byte, off int) string {
	start := off + 8
	end := off + registrySlotSize
	if start >= len(data) {
		return ""
	}
	if end > len(data) {
		end = len(data)
	}
	raw := data[start:end]
	if z := bytes.IndexByte(raw, 0); z >= 0 {
		raw = raw[:z]
	}
	for _, c := range raw { // reject non-printable (not a real name slot)
		if c < 0x20 || c > 0x7e {
			return ""
		}
	}
	return string(raw)
}
