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

// registreBlocsAttendus : le registre du film de reference fait 50 blocs — parseRegistry le borne
// a sa fin structurelle (cf. `b9390a9f5`, « le registre fait 50 blocs, pas 118 »).
const registreBlocsAttendus = 50

// TestRegistreReelDeLaMiniBobine — LE REGISTRE REEL, CONFRONTE INCONDITIONNELLEMENT.
//
// IL LIT LA MINI-BOBINE VERSIONNEE, pas le cache de developpement. Jusqu'au 2026-09-06 (lot E,
// item E.6) ce test pointait un CHEMIN ABSOLU de la machine de l'auteur et se `t.Skipf` ailleurs :
// il ne gardait donc rien en CI ni chez quiconque. La bobine de `killsource` porte le chunk_00 du
// meme film 000d5950, versionne (435 Ko) — son absence est une panne du depot, pas une condition
// d'execution, d'ou le `t.Fatal`.
//
// CE QU'IL DIT QUE LE GOLDEN NE DIT PAS. Le golden des familles
// (`golden_minibobine_test.go`) rend des empreintes : un rouge y signifie « quelque chose a
// change ». Les assertions ci-dessous NOMMENT ce qui casserait en premier si le decoupage du
// registre bougeait — l'empreinte du registre, le nombre de blocs, et les indices de composant du
// bipede que tout le decodage suppose (i0 position, i4 vitalite, i11 dead-state, 43-46 armes
// portees).
func TestRegistreReelDeLaMiniBobine(t *testing.T) {
	// ReadFilmChunk DECOMPRESSE : ParseRegistryChunk ne le fait plus (lot 1).
	raw, err := ReadFilmChunk(bobineFamilles, 0)
	if err != nil {
		t.Fatalf("chunk_00 de la mini-bobine versionnee illisible (%s) : %v", bobineFamilles, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if fp := RegistryFingerprint(reg); fp != KnownRegistryFingerprint {
		t.Errorf("empreinte du registre = %#016x, connue %#016x : la bobine ne porte plus le "+
			"registre de reference du chantier", fp, KnownRegistryFingerprint)
	}
	if len(reg.Archetypes) != registreBlocsAttendus {
		t.Errorf("%d archetypes, attendu %d (le registre est borne a sa fin structurelle)",
			len(reg.Archetypes), registreBlocsAttendus)
	}
	biped, ok := reg.Archetype(BipedTypeIndex)
	if !ok {
		t.Fatalf("aucun archetype #%d (il y en a %d)", BipedTypeIndex, len(reg.Archetypes))
	}
	for _, c := range []struct {
		i    int
		veut string
	}{
		{0, "object-position-dynamic-precision-component"},
		{4, "object-body-vitality-component"},
		{11, "object-dead-state-component"},
	} {
		if got := biped.component(c.i); got != c.veut {
			t.Errorf("biped i%d = %q, attendu %q", c.i, got, c.veut)
		}
	}
	held := biped.indicesOf(compWeaponStateTypeInfo)
	if len(held) != 4 || held[0] != 43 || held[1] != 44 || held[2] != 45 || held[3] != 46 {
		t.Errorf("indices d'arme portee = %v, attendu [43 44 45 46]", held)
	}
	if len(biped.Components) != archetypeBlockSlots {
		t.Errorf("le bipede porte %d composants, attendu %d", len(biped.Components), archetypeBlockSlots)
	}
}
