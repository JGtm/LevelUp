package mapfondsidecar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain/replaydoc"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

func sidecarDeTest(plan *replaydoc.MapBackgroundPlan) Sidecar {
	return Sidecar{
		MapBackground: replay.MapBackground{
			SchemaVersion: replay.MapBackgroundSchemaVersion,
			Module:        "ctf_bazaar",
			Image:         "ctf_bazaar.png",
			Source:        "test",
			Style:         "encre",
			GeneratedAt:   time.Date(2026, 8, 12, 23, 0, 51, 920448200, time.UTC),
			Calibration:   replay.MapBackgroundCalibration{MetersPerPixel: 0.034, WidthPx: 10, HeightPx: 12, Convention: "c"},
		},
		Plan: plan,
	}
}

// Le plan survit a une relecture-reecriture : c'est ce que fait `mapfond-webp -convertir`.
func TestSidecar_LePlanSurvitAUneReecriture(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "ctf_bazaar.json")
	plan := &replaydoc.MapBackgroundPlan{Source: "navmesh module any", Vertices: 4, Faces: 2, Edges: [][4]float64{{0, 0, 1, 0}, {1, 0, 1, 1}}}
	if err := Write(chemin, sidecarDeTest(plan)); err != nil {
		t.Fatal(err)
	}
	s, err := Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	s.Image = "ctf_bazaar.webp"
	if err := Write(chemin, *s); err != nil {
		t.Fatal(err)
	}
	relu, err := Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if relu.Image != "ctf_bazaar.webp" {
		t.Errorf("image = %q", relu.Image)
	}
	if !relu.GeneratedAt.Equal(s.GeneratedAt) {
		t.Errorf("generatedAt = %v, attendu %v", relu.GeneratedAt, s.GeneratedAt)
	}
	if relu.Plan == nil || len(relu.Plan.Edges) != 2 || relu.Plan.Vertices != 4 || relu.Plan.Faces != 2 {
		t.Fatalf("plan perdu a la reecriture : %+v", relu.Plan)
	}
}

// Sans plan, le fichier n'a pas de champ `plan`.
func TestSidecar_SansPlanPasDeChamp(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "x.json")
	if err := Write(chemin, sidecarDeTest(nil)); err != nil {
		t.Fatal(err)
	}
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), `"plan"`) {
		t.Errorf("champ plan ecrit sans plan : %s", blob)
	}
	plan, err := LoadPlan(chemin)
	if err != nil || plan != nil {
		t.Errorf("LoadPlan = %+v, %v ; attendu nil, nil", plan, err)
	}
}
