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

// TestBuildMapObjectives_PointsOnly — un rôle `PointsOnly` ne produit AUCUNE zone : l'objet à
// forme sort en MARQUEUR, posé à son centre.
//
// LE CAS QUE CE TEST TIENT est celui du correctif du 2026-08-26 : le catalogue porte des formes
// sur des rôles qui ne sont pas des aires à tenir (livraison de drapeau, navpoint de Stockpile,
// bombe d'Assaut), et les publier en zones dessinait des bases sur un CTF. On vérifie sur le
// MÊME objet — une boîte de Bastion, la seule forme du fixture — que le drapeau change sa
// destination sans changer sa position : c'est la comparaison avec/sans qui prouve l'effet.
func TestBuildMapObjectives_PointsOnly(t *testing.T) {
	zone := BuildMapObjectives(entreeObjectifs(), []ObjectiveRoleSpec{
		{Role: mapvar.RoleStrongholdZone},
	})
	if zone == nil || len(zone.Zones) != 1 || len(zone.Markers) != 0 {
		t.Fatalf("sans PointsOnly : attendu 1 zone / 0 marqueur, reçu %+v", zone)
	}
	point := BuildMapObjectives(entreeObjectifs(), []ObjectiveRoleSpec{
		{Role: mapvar.RoleStrongholdZone, PointsOnly: true},
	})
	if point == nil || len(point.Zones) != 0 || len(point.Markers) != 1 {
		t.Fatalf("avec PointsOnly : attendu 0 zone / 1 marqueur, reçu %+v", point)
	}
	// MÊME OBJET, MÊME PLACE : seule sa présentation change.
	m, z := point.Markers[0], zone.Zones[0]
	if m.X != z.X || m.Y != z.Y || m.Z != z.Z || m.Role != z.Role || m.Team != z.Team {
		t.Errorf("le marqueur doit être le même objet au même endroit : %+v contre %+v", m, z)
	}
}

// TestBuildMapObjectives_PointsOnlyGardeLesPonctuels — un rôle `PointsOnly` qui porte À LA FOIS
// des objets à forme et des objets sans forme les sert TOUS en marqueurs, sans en perdre un.
//
// C'est exactement la configuration de `flag_delivery` sur Catalyst (2 cylindres + 2 points) :
// le défaut aurait pu se « corriger » en ÉCARTANT les objets à forme, ce qui aurait fait
// disparaître la moitié des livraisons au lieu de les redessiner.
func TestBuildMapObjectives_PointsOnlyGardeLesPonctuels(t *testing.T) {
	e := entreeObjectifs()
	r := 4.0
	e.Objectives = append(e.Objectives,
		mapvar.Objective{Role: mapvar.RoleFlagDelivery, Pos: mapvar.Vec3{X: 9, Y: 1, Z: 0}, TeamIndex: 0, InstanceID: 4},
		mapvar.Objective{
			Role: mapvar.RoleFlagDelivery, Pos: mapvar.Vec3{X: 8, Y: 1, Z: 0}, TeamIndex: 1, InstanceID: 5,
			Shape: &mapvar.Shape{
				Family: mapvar.ShapeCylinder, Radius: &r, UpZ: 2, DownZ: 1,
				Forward: mapvar.Vec3{X: 0, Y: 1}, Up: mapvar.Vec3{Z: 1},
			},
		},
	)
	mo := BuildMapObjectives(e, []ObjectiveRoleSpec{{Role: mapvar.RoleFlagDelivery, PointsOnly: true}})
	if mo == nil || len(mo.Zones) != 0 || len(mo.Markers) != 2 {
		t.Fatalf("attendu 0 zone / 2 marqueurs (le cylindre ET le point), reçu %+v", mo)
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
