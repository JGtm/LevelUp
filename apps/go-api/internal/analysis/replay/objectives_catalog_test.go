package replay

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/testutil"
)

func zoneObjective(idx int, instance int32, pos mapvar.Vec3, withShape bool) mapvar.Objective {
	o := mapvar.Objective{
		Role:       mapvar.RoleStrongholdZone,
		InstanceID: instance,
		ObjectIdx:  idx,
		TeamIndex:  mapvar.TeamUnset,
		Pos:        pos,
	}
	if withShape {
		r := 3.0
		o.Shape = &mapvar.Shape{
			Family: mapvar.ShapeCylinder, Radius: &r,
			UpZ: 2, DownZ: 0.5,
			Forward: mapvar.Vec3{X: 1}, Up: mapvar.Vec3{Z: 1},
		}
	}
	return o
}

func writeCatalog(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "map_objectives.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("ecriture fixture: %v", err)
	}
	return p
}

// TestCatalogueRefuseUneAutreVersionDeSchema garde le verrou de version. Un catalogue
// v1 n'a pas de champ `shape` : le lire comme un v2 rendrait toutes ses zones
// « ponctuelles », c'est-a-dire silencieusement absentes du croisement.
func TestCatalogueRefuseUneAutreVersionDeSchema(t *testing.T) {
	p := writeCatalog(t, `{"schema_version":1,"title_slug":"halo_infinite","maps":{}}`)
	if _, err := LoadMapObjectives(p); err == nil {
		t.Fatal("catalogue en v1 : attendu une erreur de version, obtenu nil")
	}
	p2 := writeCatalog(t, `{"schema_version":2,"title_slug":"halo_infinite","maps":{}}`)
	if _, err := LoadMapObjectives(p2); err != nil {
		t.Fatalf("catalogue en v2 : %v", err)
	}
}

// TestLookupCarteInconnueEstUnCasNominal : le catalogue couvre 34 cartes sur la centaine
// jouee. L'appelant doit pouvoir DISTINGUER « carte non couverte » d'une panne de
// lecture, sinon il ne peut pas degrader proprement.
func TestLookupCarteInconnueEstUnCasNominal(t *testing.T) {
	p := writeCatalog(t, `{"schema_version":2,"maps":{}}`)
	c, err := LoadMapObjectives(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Lookup("aucune-carte"); !errors.Is(err, ErrUnknownMap) {
		t.Errorf("err = %v, attendu ErrUnknownMap", err)
	}
	var nilCat *MapObjectivesCatalog
	if _, err := nilCat.Lookup("x"); !errors.Is(err, ErrUnknownMap) {
		t.Errorf("catalogue nil : err = %v, attendu ErrUnknownMap", err)
	}
}

// TestZonesOfRoleFiltreEtCompteCeQuIlEcarte garde les compteurs de couverture. Une zone
// sans forme n'est pas testable : la taire ferait passer 2 zones sur 3 pour la carte
// entiere.
func TestZonesOfRoleFiltreEtCompteCeQuIlEcarte(t *testing.T) {
	e := MapObjectivesEntry{Objectives: []mapvar.Objective{
		zoneObjective(0, 11, mapvar.Vec3{X: 5}, true),
		zoneObjective(1, 12, mapvar.Vec3{X: 9}, false), // sans forme
		{Role: mapvar.RoleFlagSpawn, ObjectIdx: 2, Pos: mapvar.Vec3{X: 1}},
	}}
	set := e.ZonesOfRole(mapvar.RoleStrongholdZone)
	if len(set.Zones) != 1 {
		t.Fatalf("zones = %d, attendu 1", len(set.Zones))
	}
	if set.Zones[0].InstanceID != 11 {
		t.Errorf("instance retenue = %d, attendu 11", set.Zones[0].InstanceID)
	}
	if set.Pointless != 1 {
		t.Errorf("Pointless = %d, attendu 1 (la zone sans forme)", set.Pointless)
	}
	if set.Degenerate != 0 {
		t.Errorf("Degenerate = %d, attendu 0", set.Degenerate)
	}
	if set.Carried {
		t.Errorf("Carried = true sur une carte au schema courant")
	}
}

// TestZonesOfRoleCompteLesFormesDegenerees separe « pas de forme » de « forme
// inutilisable » : les deux disparaissent du croisement, mais seule la seconde est une
// anomalie a expliquer.
func TestZonesOfRoleCompteLesFormesDegenerees(t *testing.T) {
	degeneree := zoneObjective(0, 11, mapvar.Vec3{}, true)
	degeneree.Shape.Up = mapvar.Vec3{}
	set := MapObjectivesEntry{Objectives: []mapvar.Objective{degeneree}}.
		ZonesOfRole(mapvar.RoleStrongholdZone)
	if len(set.Zones) != 0 || set.Degenerate != 1 || set.Pointless != 0 {
		t.Errorf("zones=%d degenerate=%d pointless=%d, attendu 0/1/0",
			len(set.Zones), set.Degenerate, set.Pointless)
	}
}

// TestRangSpatialNeDependPasDeLOrdreDuFichier est LE garde-rail du champ SpatialRank.
// Le catalogue ne porte aucun nom de zone et l'ordre du fichier n'en est pas un : si le
// rang suivait cet ordre, la meme zone changerait de nom a la prochaine regeneration.
func TestRangSpatialNeDependPasDeLOrdreDuFichier(t *testing.T) {
	a := zoneObjective(0, 101, mapvar.Vec3{X: -10, Y: 3}, true)
	b := zoneObjective(1, 102, mapvar.Vec3{X: 0, Y: -7}, true)
	c := zoneObjective(2, 103, mapvar.Vec3{X: 12, Y: 1}, true)

	ordres := [][]mapvar.Objective{{a, b, c}, {c, b, a}, {b, c, a}}
	var reference []int32
	for i, objs := range ordres {
		set := MapObjectivesEntry{Objectives: objs}.ZonesOfRole(mapvar.RoleStrongholdZone)
		got := make([]int32, len(set.Zones))
		for j, z := range set.Zones {
			if z.SpatialRank != j {
				t.Errorf("ordre %d : SpatialRank = %d au rang %d", i, z.SpatialRank, j)
			}
			got[j] = z.InstanceID
		}
		if reference == nil {
			reference = got
			continue
		}
		for j := range got {
			if got[j] != reference[j] {
				t.Fatalf("ordre %d : rang %d = instance %d, attendu %d (le rang suit "+
					"l'ordre du fichier)", i, j, got[j], reference[j])
			}
		}
	}
	if len(reference) != 3 || reference[0] != 101 || reference[2] != 103 {
		t.Errorf("tri spatial attendu 101,102,103 en x croissant ; obtenu %v", reference)
	}
}

// TestCarteReporteeEstSignalee garde le piege documente par le producteur : sur une carte
// reportee d'un schema anterieur, une zone sans forme est NON MIGREE, pas ponctuelle.
func TestCarteReporteeEstSignalee(t *testing.T) {
	e := MapObjectivesEntry{
		CarriedFromSchema: 1,
		Objectives:        []mapvar.Objective{zoneObjective(0, 11, mapvar.Vec3{}, false)},
	}
	set := e.ZonesOfRole(mapvar.RoleStrongholdZone)
	if !set.Carried {
		t.Error("carte reportee : Carried = false, le consommateur croira a des zones ponctuelles")
	}
	if set.Pointless != 1 {
		t.Errorf("Pointless = %d, attendu 1", set.Pointless)
	}
}

// TestCatalogueLivreEstExploitable lit le catalogue REELLEMENT versionne. Il vaut mieux
// qu'une fixture : c'est lui que le croisement lira.
//
// Invariant garde (mesure du 2026-08-08 : 63 zones Bastion sur 21 cartes, 161 zones
// d'Extraction sur 32, 100 % avec forme) : un role SURFACIQUE n'a aucun objectif sans
// forme et aucune forme degeneree. Le jour ou ce n'est plus vrai, c'est une nouveaute a
// expliquer, pas un cas a rattraper en silence.
func TestCatalogueLivreEstExploitable(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	path := filepath.Join(root, "data", "titles", "halo_infinite", "reference", "map_objectives.json")
	cat, err := LoadMapObjectives(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue vide")
	}
	// pointlessConnues : les zones surfaciques SANS forme du corpus, nominatives et
	// EXPLIQUEES — pas une tolerance. La mesure corpus (mapvar/shape.go : zones Bastion
	// 430/431 avec forme, et AUCUNE famille de forme inconnue sur les 16 434 formes des
	// 199 variantes) etablit que la SOURCE ne porte pas de sac de forme pour cette zone :
	// Salvation (extraction masse du 2026-08-13), zone Bastion a l'objet 446. Les
	// consommateurs la voient deja via ZonesOfRole().Pointless et l'ecartent. Toute
	// occurrence NON listee ici reste une erreur ; si une regeneration lui trouve un jour
	// une forme, l'entree devient fausse et se retire.
	pointlessConnues := map[string]map[mapvar.Role]int{
		"cd08bc7a-7ba5-4502-be87-c58b641fc94d": {mapvar.RoleStrongholdZone: 1}, // Salvation
	}
	// RoleHill depuis le lot C-ter volet 2 : 113 collines sur 23 cartes, 100 % avec forme —
	// les 4 cartes des films (Catalyst 6, Chasm 5, Shogun 5, Solitude - Ranked 5) plus les
	// 19 autres cartes KOTH du registre, regenerees a la revue ronde 1 (R1-3).
	roles := []mapvar.Role{mapvar.RoleStrongholdZone, mapvar.RoleExtractionZone, mapvar.RoleHill}
	total := map[mapvar.Role]int{}
	for id, e := range cat.Maps {
		for _, role := range roles {
			set := e.ZonesOfRole(role)
			total[role] += len(set.Zones)
			if attendu := pointlessConnues[id][role]; set.Pointless != attendu {
				t.Errorf("carte %s, role %s : %d objectif(s) SURFACIQUE(s) sans forme (attendu %d — cf. pointlessConnues)",
					id, role, set.Pointless, attendu)
			}
			if set.Degenerate != 0 {
				t.Errorf("carte %s, role %s : %d forme(s) degeneree(s)", id, role, set.Degenerate)
			}
		}
	}
	for _, role := range roles {
		if total[role] == 0 {
			t.Errorf("aucune zone exploitable pour le role %s", role)
		}
	}
	t.Logf("catalogue livre : %d cartes, %d zones Bastion, %d zones Extraction, %d collines",
		len(cat.Maps), total[mapvar.RoleStrongholdZone], total[mapvar.RoleExtractionZone], total[mapvar.RoleHill])
}
