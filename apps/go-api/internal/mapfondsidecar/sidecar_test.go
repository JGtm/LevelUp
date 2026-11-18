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

func sidecarDeTest(plan *replaydoc.MapBackgroundPlan) MapBackground {
	return MapBackground{
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
	ext, err := lireExtensions(chemin)
	if err != nil || ext.Plan != nil || ext.PlanImage != "" {
		t.Errorf("lireExtensions = %+v, %v ; attendu des extensions vides", ext, err)
	}
}

// L'image plan voyage avec le plan : nom ecrit, relu, et sans plan ni image le fichier n'a pas
// de champ `planImage`.
func TestSidecar_ImagePlanAllerRetour(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "x.json")
	s := sidecarDeTest(&replaydoc.MapBackgroundPlan{Source: "t", Vertices: 4, Faces: 2, Edges: [][4]float64{{0, 0, 1, 0}}})
	s.PlanImage = "x.plan.webp"
	if err := Write(chemin, s); err != nil {
		t.Fatal(err)
	}
	relu, err := Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if relu.PlanImage != "x.plan.webp" || relu.Plan == nil {
		t.Errorf("image plan ou plan perdu : %q, %+v", relu.PlanImage, relu.Plan)
	}
	if err := Write(chemin, sidecarDeTest(nil)); err != nil {
		t.Fatal(err)
	}
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "planImage") {
		t.Errorf("champ planImage ecrit sans image plan : %s", blob)
	}
}

// Le nom de l'image plan se fabrique et se relit par le meme couple de fonctions.
func TestNomImagePlanEtCle(t *testing.T) {
	for _, ext := range []string{".png", ".webp"} {
		nom := NomImagePlan("ridgeline", ext)
		if cle, ok := CleDeImagePlan(nom); !ok || cle != "ridgeline" {
			t.Errorf("CleDeImagePlan(%q) = %q, %v", nom, cle, ok)
		}
	}
	for _, nom := range []string{"ridgeline.png", "ridgeline.webp", ".plan.png", "a.b.png"} {
		if cle, ok := CleDeImagePlan(nom); ok {
			t.Errorf("CleDeImagePlan(%q) = %q, attendu un refus", nom, cle)
		}
	}
}

// Les noms des deux fichiers du relief voyagent avec le sidecar : écrits, relus, conservés par une
// réécriture (celle de `mapfond-webp -convertir`), lus par `lireExtensions`, et absents du fichier
// quand la carte n'a pas de relief.
func TestSidecar_ReliefAllerRetour(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "x.json")
	s := sidecarDeTest(&replaydoc.MapBackgroundPlan{Source: "t", Vertices: 4, Faces: 2, Edges: [][4]float64{{0, 0, 1, 0}}})
	s.PlanImage = "x.plan.webp"
	s.ReliefImage = "x.relief.webp"
	s.ReliefData = "x.relief.json"
	if err := Write(chemin, s); err != nil {
		t.Fatal(err)
	}
	relu, err := Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if relu.ReliefImage != "x.relief.webp" || relu.ReliefData != "x.relief.json" {
		t.Errorf("relief perdu : %q, %q", relu.ReliefImage, relu.ReliefData)
	}
	if relu.PlanImage != "x.plan.webp" || relu.Plan == nil {
		t.Errorf("l'ajout du relief a abîmé le plan : %q, %+v", relu.PlanImage, relu.Plan)
	}
	ext, err := lireExtensions(chemin)
	if err != nil || ext.ReliefImage != "x.relief.webp" || ext.ReliefData != "x.relief.json" {
		t.Errorf("lireExtensions = %+v, %v", ext, err)
	}

	relu.Image = "x.webp"
	if err := Write(chemin, *relu); err != nil {
		t.Fatal(err)
	}
	if apres, err := Load(chemin); err != nil || apres.ReliefImage != "x.relief.webp" || apres.ReliefData != "x.relief.json" {
		t.Errorf("relief perdu à la réécriture : %+v, %v", apres, err)
	}

	if err := Write(chemin, sidecarDeTest(nil)); err != nil {
		t.Fatal(err)
	}
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(blob), "relief") {
		t.Errorf("champ du relief écrit sans relief : %s", blob)
	}
	if ext, err := lireExtensions(chemin); err != nil || ext.ReliefImage != "" || ext.ReliefData != "" {
		t.Errorf("lireExtensions sans relief = %+v, %v", ext, err)
	}
}
