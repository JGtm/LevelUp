package mapfondsidecar

// images_publiees_test.go — LES FONDS PUBLIES ET LEURS IMAGES PLAN.
//
// Une image plan ne sert que si elle se superpose a l'image du fond : memes dimensions, memes que
// le calage du sidecar. Ce test lit les fonds PUBLIES (donnee versionnee) : un plan sans image,
// une image sans plan, ou une image de la mauvaise taille est une cuisson interrompue, et le
// client afficherait un plan decale ou rien.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"

	"golang.org/x/image/webp"
)

// dimensionsWebP lit les dimensions d'une image sans la decoder en entier.
func dimensionsWebP(t *testing.T, chemin string) (int, int) {
	t.Helper()
	f, err := os.Open(chemin)
	if err != nil {
		t.Fatalf("image absente : %v", err)
	}
	defer f.Close()
	cfg, err := webp.DecodeConfig(f)
	if err != nil {
		t.Fatalf("image illisible (%s) : %v", chemin, err)
	}
	return cfg.Width, cfg.Height
}

func TestChaqueImagePlanSuperposeAuFond(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	dir := filepath.Join(racine, "data", "titles", "halo_infinite", "reference", "map_backgrounds")
	sidecars, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(sidecars) == 0 {
		t.Skipf("aucun fond publie (%v)", err)
	}
	avecPlan := 0
	for _, chemin := range sidecars {
		cle := strings.TrimSuffix(filepath.Base(chemin), ".json")
		s, err := Load(chemin)
		if err != nil {
			t.Errorf("%s : %v", cle, err)
			continue
		}
		if (s.Plan != nil) != (s.PlanImage != "") {
			t.Errorf("%s : plan %v mais image plan %q — l'un ne va pas sans l'autre", cle, s.Plan != nil, s.PlanImage)
			continue
		}
		if s.PlanImage == "" {
			continue
		}
		avecPlan++
		if nom, ok := CleDeImagePlan(s.PlanImage); !ok || nom != cle || filepath.Base(s.PlanImage) != s.PlanImage {
			t.Errorf("%s : nom d'image plan %q hors convention", cle, s.PlanImage)
			continue
		}
		wp, hp := dimensionsWebP(t, filepath.Join(dir, s.PlanImage))
		wf, hf := dimensionsWebP(t, filepath.Join(dir, s.Image))
		cal := s.Calibration
		if wp != wf || hp != hf || wp != cal.WidthPx || hp != cal.HeightPx {
			t.Errorf("%s : image plan %dx%d, fond %dx%d, calage %dx%d — elles ne se superposent pas",
				cle, wp, hp, wf, hf, cal.WidthPx, cal.HeightPx)
		}
	}
	t.Logf("%d fonds sur %d portent un plan et son image", avecPlan, len(sidecars))
}
