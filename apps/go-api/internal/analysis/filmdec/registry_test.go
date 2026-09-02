package filmdec

import "testing"

// buildBlock renders one 64-slot archetype block from ordered component names.
func buildBlock(names []string) []byte {
	blk := make([]byte, archetypeBlockSize)
	for i, n := range names {
		if i >= archetypeBlockSlots {
			break
		}
		copy(blk[i*registrySlotSize+8:], []byte(n))
	}
	return blk
}

func TestParseRegistrySynthetic(t *testing.T) {
	a := []string{"object-position-dynamic-precision-component", "object-body-vitality-component", "weapon-state-type-info"}
	b := []string{"game-engine-team-mapping-component"}
	data := append(buildBlock(a), buildBlock(b)...)

	reg := parseRegistry(data)
	if len(reg.Archetypes) != 2 {
		t.Fatalf("got %d archetypes, want 2", len(reg.Archetypes))
	}
	if got := reg.Archetypes[0].Components; len(got) != 3 || got[0] != a[0] || got[2] != "weapon-state-type-info" {
		t.Fatalf("block0 components = %v", got)
	}
	if got := reg.Archetypes[1].Components; len(got) != 1 || got[0] != b[0] {
		t.Fatalf("block1 components = %v", got)
	}
	if idx := reg.Archetypes[0].indicesOf("weapon-state-type-info"); len(idx) != 1 || idx[0] != 2 {
		t.Fatalf("weapon-state-type-info index = %v", idx)
	}
}

// chunk00Dir is the dev cache film directory; the integration test skips when it is absent.
const chunk00Dir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func TestParseRegistryBipedBlock35(t *testing.T) {
	// ReadFilmChunk DECOMPRESSE : ParseRegistryChunk ne le fait plus (lot 1).
	raw, err := ReadFilmChunk(chunk00Dir, 0)
	if err != nil {
		t.Skipf("chunk_00 absent (%v)", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	biped, ok := reg.Archetype(35)
	if !ok {
		t.Fatalf("no archetype #35 (have %d)", len(reg.Archetypes))
	}
	if got := biped.component(0); got != "object-position-dynamic-precision-component" {
		t.Fatalf("biped i0 = %q", got)
	}
	if got := biped.component(4); got != "object-body-vitality-component" {
		t.Fatalf("biped i4 = %q", got)
	}
	if got := biped.component(11); got != "object-dead-state-component" {
		t.Fatalf("biped i11 = %q", got)
	}
	held := biped.indicesOf("weapon-state-type-info")
	if len(held) != 4 || held[0] != 43 {
		t.Fatalf("held-weapon indices = %v, want [43 44 45 46]", held)
	}
	if len(biped.Components) != 64 {
		t.Fatalf("biped component count = %d, want 64", len(biped.Components))
	}
}
