package service

// replay_map_background_plan_test.go — LE PLAN DES MURS TRAVERSE LE SERVICE.
//
// Le sidecar d'un fond porte, a cote du calage que le decodeur de film lit, le plan des murs
// tire du maillage de navigation. Le decodeur l'ignore : c'est le service qui le relit et le
// joint a la reponse. Une carte sans plan reste servie, champ absent.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
)

const sidecarAvecPlan = `{"schemaVersion":1,"module":"ridgeline",` +
	`"mapNames":["cliffhanger","cliffhanger_ridgeline"],"image":"ridgeline.png",` +
	`"source":"test","generatedAt":"2026-08-10T20:25:25Z","style":"encre",` +
	`"calibration":{"metersPerPixel":0.092,"originX":-57.3,"originY":78.87,` +
	`"widthPx":1633,"heightPx":1627,"convention":"x = originX + (px+0.5)*mpp"},` +
	`"stats":{"anchors":14,"anchorsInFrame":14,"anchorsWithGround":14},` +
	`"plan":{"source":"navmesh module any","vertices":5,"faces":2,` +
	`"edges":[[0,0,1,0],[1,0,1,1.5],[-2.25,3,0,0]]}}`

func TestMapBackgroundForMap_ServiLePlanDesMurs(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	ecrire(t, title.NewPathResolver(root).MapBackgroundMetaPath(title.DefaultSlug, "ridgeline"), sidecarAvecPlan)
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo, nil)

	bg, err := svc.MapBackgroundForMap(context.Background(), "asset-cliffhanger")
	if err != nil {
		t.Fatalf("MapBackgroundForMap: %v", err)
	}
	if bg.Plan == nil {
		t.Fatal("le plan du sidecar n'est pas servi")
	}
	if bg.Plan.Source != "navmesh module any" || bg.Plan.Vertices != 5 || bg.Plan.Faces != 2 {
		t.Errorf("en-tete du plan alteree : %+v", bg.Plan)
	}
	want := [][4]float64{{0, 0, 1, 0}, {1, 0, 1, 1.5}, {-2.25, 3, 0, 0}}
	if len(bg.Plan.Edges) != len(want) {
		t.Fatalf("%d aretes servies, attendu %d", len(bg.Plan.Edges), len(want))
	}
	for i, a := range want {
		if bg.Plan.Edges[i] != a {
			t.Errorf("arete %d = %v, attendu %v", i, bg.Plan.Edges[i], a)
		}
	}
	if bg.Calibration.MetersPerPixel != 0.092 {
		t.Errorf("le calage ne doit pas souffrir du plan : %+v", bg.Calibration)
	}
}

func TestMapBackground_ServiLePlanParMatch(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	ecrire(t, title.NewPathResolver(root).MapBackgroundMetaPath(title.DefaultSlug, "ridgeline"), sidecarAvecPlan)
	repo := &mapNamesStub{names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo, nil)

	bg, err := svc.MapBackground(context.Background(), "match-1")
	if err != nil {
		t.Fatalf("MapBackground: %v", err)
	}
	if bg.Plan == nil || len(bg.Plan.Edges) != 3 {
		t.Fatalf("plan par match = %+v", bg.Plan)
	}
}

func TestMapBackgroundForMap_SansPlanLeChampEstAbsent(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo, nil)

	bg, err := svc.MapBackgroundForMap(context.Background(), "asset-cliffhanger")
	if err != nil {
		t.Fatalf("MapBackgroundForMap: %v", err)
	}
	if bg.Plan != nil {
		t.Errorf("plan = %+v sur un sidecar sans plan", bg.Plan)
	}
	blob, err := json.Marshal(bg)
	if err != nil {
		t.Fatal(err)
	}
	var champs map[string]json.RawMessage
	if err := json.Unmarshal(blob, &champs); err != nil {
		t.Fatal(err)
	}
	if _, present := champs["plan"]; present {
		t.Errorf("le champ plan doit etre absent de la reponse : %s", blob)
	}
}

// Un plan illisible ne prive pas le client du calage ni de l'image.
func TestMapBackgroundForMap_PlanIllisibleSertLeCalage(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	corrompu := strings.Replace(sidecarAvecPlan, `"edges":[[0,0,1,0],[1,0,1,1.5],[-2.25,3,0,0]]`, `"edges":"pas un tableau"`, 1)
	if corrompu == sidecarAvecPlan {
		t.Fatal("le sidecar de test n'a pas ete corrompu")
	}
	ecrire(t, title.NewPathResolver(root).MapBackgroundMetaPath(title.DefaultSlug, "ridgeline"), corrompu)
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo, nil)

	bg, err := svc.MapBackgroundForMap(context.Background(), "asset-cliffhanger")
	if err != nil {
		t.Fatalf("un plan illisible ne doit pas faire echouer le calage : %v", err)
	}
	if bg.Plan != nil {
		t.Errorf("plan = %+v, attendu absent", bg.Plan)
	}
	if bg.Calibration.MetersPerPixel != 0.092 {
		t.Errorf("calage perdu : %+v", bg.Calibration)
	}
}
