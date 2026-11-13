package service

// replay_map_background_image_plan_test.go — L'IMAGE PLAN TRAVERSE LE SERVICE.
//
// Son nom vit dans le sidecar (`planImage`) ; elle se sert sous la meme cle que le fond, avec les
// memes gardes sur le nom de fichier. Une carte sans `planImage` n'a pas de plan a servir : meme
// absence que sans fond.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// fondAvecImagePlan pose un fond de Cliffhanger dont le sidecar porte `planImage`.
func fondAvecImagePlan(t *testing.T, planImage string, ecritImage bool) (string, *replayService) {
	t.Helper()
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	res := title.NewPathResolver(root)
	sidecar := strings.Replace(sidecarAvecPlan, `"plan":{`, `"planImage":"`+planImage+`","plan":{`, 1)
	ecrire(t, res.MapBackgroundMetaPath(title.DefaultSlug, "ridgeline"), sidecar)
	if ecritImage {
		ecrire(t, res.MapBackgroundImageFilePath(title.DefaultSlug, planImage), "RIFFplan")
	}
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	return root, NewReplayService(title.DefaultSlug, root, repo, nil).(*replayService)
}

func TestMapBackgroundPlanImage_ServiParCarteEtParMatch(t *testing.T) {
	_, svc := fondAvecImagePlan(t, "ridgeline.plan.webp", true)
	ctx := context.Background()

	blob, mime, err := svc.MapBackgroundPlanImageForMap(ctx, "asset-cliffhanger")
	if err != nil || string(blob) != "RIFFplan" || mime != mimeImageWebP {
		t.Fatalf("par carte : %q %q %v", blob, mime, err)
	}
	blob, mime, err = svc.MapBackgroundPlanImage(ctx, "match-1")
	if err != nil || string(blob) != "RIFFplan" || mime != mimeImageWebP {
		t.Fatalf("par match : %q %q %v", blob, mime, err)
	}
	if fond, _, err := svc.MapBackgroundImageForMap(ctx, "asset-cliffhanger"); err != nil || string(fond) == "RIFFplan" {
		t.Errorf("l'image du fond ne doit pas etre le plan : %q %v", fond, err)
	}
	bg, err := svc.MapBackgroundForMap(ctx, "asset-cliffhanger")
	if err != nil || bg.PlanImage != "ridgeline.plan.webp" {
		t.Errorf("planImage servi = %v, %v", bg, err)
	}
}

func TestMapBackgroundPlanImage_AbsenceEstUnFondAbsent(t *testing.T) {
	ctx := context.Background()
	// Sidecar sans planImage.
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo, nil)
	if _, _, err := svc.MapBackgroundPlanImageForMap(ctx, "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("sans planImage : %v, attendu ErrMapBackgroundNotAvailable", err)
	}
	// planImage declare mais fichier absent.
	_, svc2 := fondAvecImagePlan(t, "ridgeline.plan.webp", false)
	if _, _, err := svc2.MapBackgroundPlanImageForMap(ctx, "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("fichier absent : %v, attendu ErrMapBackgroundNotAvailable", err)
	}
	// Carte inconnue.
	inconnue := &mapNamesStub{}
	if _, _, err := NewReplayService(title.DefaultSlug, root, inconnue, nil).MapBackgroundPlanImageForMap(ctx, "inconnue"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("carte inconnue : %v", err)
	}
}

// Le nom de l'image plan est un contenu de fichier : meme garde que l'image du fond (nom simple,
// extension en liste blanche).
func TestMapBackgroundPlanImage_RefuseUnNomDangereux(t *testing.T) {
	for _, nom := range []string{"../ridgeline.webp", "ridgeline.plan.exe", "sous/dossier.webp"} {
		_, svc := fondAvecImagePlan(t, strings.ReplaceAll(nom, `\`, ""), false)
		if _, _, err := svc.MapBackgroundPlanImageForMap(context.Background(), "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("%q accepte : %v", nom, err)
		}
	}
}
