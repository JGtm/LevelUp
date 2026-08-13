package replay

import (
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// entreeObjectifs construit une entrée de catalogue en mémoire : deux points de spawn et
// une zone boîte, le strict nécessaire du builder.
func entreeObjectifs() MapObjectivesEntry {
	hx, hy := 2.0, 3.0
	return MapObjectivesEntry{
		MapID: "m",
		Objectives: []mapvar.Objective{
			{Role: mapvar.RoleFlagSpawn, Pos: mapvar.Vec3{X: 1, Y: 2, Z: 3}, TeamIndex: 0, InstanceID: 1},
			{Role: mapvar.RoleFlagSpawn, Pos: mapvar.Vec3{X: -1, Y: 2, Z: 3}, TeamIndex: 1, InstanceID: 2},
			{
				Role: mapvar.RoleStrongholdZone, Pos: mapvar.Vec3{X: 5, Y: 5, Z: 1}, TeamIndex: 0, InstanceID: 3,
				Shape: &mapvar.Shape{
					Family: mapvar.ShapeBox, HalfX: &hx, HalfY: &hy, UpZ: 2, DownZ: 1,
					Forward: mapvar.Vec3{X: 0, Y: 1}, Up: mapvar.Vec3{Z: 1},
				},
			},
		},
	}
}

// TestBuildMapObjectives_DedupEtOrdre — un rôle demandé deux fois (deux entrées de table
// qui matchent le même mode) ne sert ses objets qu'UNE fois ; la première spécification
// gagne, y compris son drapeau neutral.
func TestBuildMapObjectives_DedupEtOrdre(t *testing.T) {
	mo := BuildMapObjectives(entreeObjectifs(), []ObjectiveRoleSpec{
		{Role: mapvar.RoleStrongholdZone, Neutral: true},
		{Role: mapvar.RoleStrongholdZone, Neutral: false}, // dupliqué : ignoré
		{Role: mapvar.RoleFlagSpawn},
	})
	if mo == nil || len(mo.Zones) != 1 || len(mo.Markers) != 2 {
		t.Fatalf("attendu 1 zone / 2 marqueurs, reçu %+v", mo)
	}
	if mo.Zones[0].Team != TeamNeutral {
		t.Errorf("la première spécification (neutral) doit gagner: %+v", mo.Zones[0])
	}
	// L'ordre des marqueurs est le tri spatial (x croissant), pas l'ordre du fichier.
	if mo.Markers[0].X != -1 || mo.Markers[1].X != 1 {
		t.Errorf("ordre spatial attendu, reçu %+v", mo.Markers)
	}
}

// TestBuildMapObjectives_RienAServir — aucun objet des rôles demandés = nil, jamais un
// calque vide (le champ du document doit rester ABSENT).
func TestBuildMapObjectives_RienAServir(t *testing.T) {
	if mo := BuildMapObjectives(entreeObjectifs(), []ObjectiveRoleSpec{{Role: mapvar.RoleOddballSpawn}}); mo != nil {
		t.Errorf("attendu nil, reçu %+v", mo)
	}
	if mo := BuildMapObjectives(entreeObjectifs(), nil); mo != nil {
		t.Errorf("specs vides : attendu nil, reçu %+v", mo)
	}
}
