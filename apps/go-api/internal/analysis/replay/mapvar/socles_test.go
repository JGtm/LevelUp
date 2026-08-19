package mapvar

// Tests — socles.go : LA FAMILLE, LE REGROUPEMENT, L'ORDRE.
//
// CE QUE CE FICHIER VERROUILLE, et chaque point est une clause mesuree du plan
// `.ai/V7.5/replay2d/PLAN_SOCLES_MVAR.md` :
//   - trois type_id et TROIS SEULEMENT sont des socles ; un type voisin ne l'est pas ;
//   - deux objets a moins d'un metre sont LE MEME emplacement (Catalyst : 13 objets,
//     11 emplacements) et le compte des objets fusionnes est publie ;
//   - deux objets a plus d'un metre restent DEUX emplacements ;
//   - l'ordre de sortie est spatial, donc independant de l'ordre du fichier — sans quoi le
//     catalogue versionne bougerait a chaque regeneration sans qu'aucune donnee ne change.

import "testing"

// obj fabrique un objet place, la seule chose dont PadSpots ait besoin.
func obj(typeID int32, x, y, z float64, inst int32) Object {
	return Object{TypeID: typeID, Pos: Vec3{X: x, Y: y, Z: z}, InstanceID: inst,
		TeamIndex: TeamUnset, Category: -1}
}

func TestPadFamilyOf(t *testing.T) {
	cas := []struct {
		id   int32
		fam  PadFamily
		socl bool
	}{
		{0x5F379533, PadFamilyPower, true},
		{0x6253CFC0, PadFamilyRack, true},
		{0x5E86D110, PadFamilyPowerup, true},
		// Le volume d'objectif que `map_objectives.json` publie deja : PAS un socle.
		{-1239931096, "", false},
		{0, "", false},
	}
	for _, c := range cas {
		fam, ok := PadFamilyOf(c.id)
		if ok != c.socl || fam != c.fam {
			t.Errorf("PadFamilyOf(%d) = (%q, %v), attendu (%q, %v)", c.id, fam, ok, c.fam, c.socl)
		}
	}
}

// TestPadSpotsRegroupement — le compte HONNETE : deux objets a 4,7 cm sont un emplacement.
func TestPadSpotsRegroupement(t *testing.T) {
	v := &Variant{Objects: []Object{
		obj(0x6253CFC0, 5.16, -0.003, 26.501, 10),
		obj(0x6253CFC0, 5.207, -0.003, 26.501, 11), // 4,7 cm : LE MEME emplacement
		obj(0x5F379533, -9.738, -0.003, 22.403, 12),
		obj(1, 0, 0, 0, 13), // pas un socle
	}}
	spots := PadSpots(v)
	if len(spots) != 2 {
		t.Fatalf("emplacements = %d, attendu 2 : %+v", len(spots), spots)
	}
	// L'ordre est spatial : x = -9,738 avant x = 5,16.
	if spots[0].Family != PadFamilyPower || spots[0].Objects != 1 {
		t.Errorf("premier emplacement = %+v, attendu la famille power et 1 objet", spots[0])
	}
	if spots[1].Family != PadFamilyRack || spots[1].Objects != 2 {
		t.Errorf("second emplacement = %+v, attendu la famille rack et 2 objets", spots[1])
	}
	if spots[0].Mixed || spots[1].Mixed {
		t.Errorf("aucun melange de famille attendu : %+v", spots)
	}
}

// TestPadSpotsSepares — au-dela d'un metre, deux socles restent deux socles.
func TestPadSpotsSepares(t *testing.T) {
	v := &Variant{Objects: []Object{
		obj(0x6253CFC0, 0, 0, 0, 1),
		obj(0x6253CFC0, 0, 0, 1.01, 2), // un metre et un centimetre PLUS HAUT
	}}
	if spots := PadSpots(v); len(spots) != 2 {
		t.Fatalf("emplacements = %d, attendu 2 (la 3e dimension compte) : %+v", len(spots), spots)
	}
}

// TestPadSpotsOrdreStable — le meme jeu d'objets dans deux ordres de fichier rend la MEME
// suite d'emplacements. C'est ce qui rend le catalogue versionne diffable.
func TestPadSpotsOrdreStable(t *testing.T) {
	a := []Object{
		obj(0x6253CFC0, 5.16, 0, 26.5, 10),
		obj(0x5F379533, -9.73, 0, 22.4, 11),
		obj(0x5E86D110, 0.25, 0, 21.36, 12),
	}
	b := []Object{a[2], a[0], a[1]}
	sa, sb := PadSpots(&Variant{Objects: a}), PadSpots(&Variant{Objects: b})
	if len(sa) != 3 || len(sb) != 3 {
		t.Fatalf("emplacements = %d et %d, attendu 3", len(sa), len(sb))
	}
	for i := range sa {
		if sa[i].Pos != sb[i].Pos || sa[i].Family != sb[i].Family {
			t.Fatalf("rang %d : %+v vs %+v — l'ordre depend du fichier", i, sa[i], sb[i])
		}
	}
	if sa[0].Family != PadFamilyPower || sa[1].Family != PadFamilyPowerup || sa[2].Family != PadFamilyRack {
		t.Errorf("ordre spatial attendu (x croissant) : %+v", sa)
	}
}

func TestPadSpotsVariantNil(t *testing.T) {
	if spots := PadSpots(nil); spots != nil {
		t.Errorf("PadSpots(nil) = %+v, attendu nil", spots)
	}
}

func TestDist3(t *testing.T) {
	d := Dist3(Vec3{X: 0, Y: 0, Z: 0}, Vec3{X: 3, Y: 4, Z: 12})
	if d != 13 {
		t.Errorf("Dist3 = %v, attendu 13", d)
	}
}
