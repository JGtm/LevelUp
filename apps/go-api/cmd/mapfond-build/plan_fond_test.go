package main

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-film-decoder/replay"

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// carresDeTest bat un maillage de carres de 1 m dont le coin bas gauche est donne.
func carresDeTest(coins [][2]float64) *hinavmesh.Maillage {
	m := &hinavmesh.Maillage{}
	for _, c := range coins {
		base := int32(len(m.Sommets))
		m.Sommets = append(m.Sommets,
			hinavmesh.Point{X: c[0], Y: c[1]}, hinavmesh.Point{X: c[0] + 1, Y: c[1]},
			hinavmesh.Point{X: c[0] + 1, Y: c[1] + 1}, hinavmesh.Point{X: c[0], Y: c[1] + 1})
		m.Faces = append(m.Faces, hinavmesh.Face{Sommets: []int32{base, base + 1, base + 2, base + 3}})
	}
	return m
}

// Le predicat de matiere suit la convention du calage : xMonde = origineX + (px + 0,5) x mpp et
// yMonde = origineY - (py + 0,5) x mpp ; hors du cadre, pas de matiere.
func TestMatiereDeImageSuitLeCalage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.SetRGBA(1, 2, color.RGBA{R: 9, G: 9, B: 9, A: 255})
	cal := himap.CalageFond{OrigineX: 10, OrigineY: 20, MetresParPixel: 1, Largeur: 4, Hauteur: 4}
	m := matiereDeImage(img, cal)
	if !m(11.5, 17.5) {
		t.Error("le centre du pixel (1, 2) porte de la matiere")
	}
	for nom, p := range map[string][2]float64{
		"pixel vide": {10.5, 19.5}, "a gauche": {9.5, 17.5}, "au-dessus": {11.5, 21}, "a droite": {14.5, 17.5}, "en dessous": {11.5, 15.5},
	} {
		if m(p[0], p[1]) {
			t.Errorf("%s : matiere annoncee", nom)
		}
	}
	if compteMatiere(img) != 1 {
		t.Errorf("compteMatiere = %d, attendu 1", compteMatiere(img))
	}
}

// Le plan ecarte la composante sans ancre ni matiere et garde celle qui porte l'ancre ; ses aretes
// sont coupees au cadre et arrondies au millimetre.
func TestCalculePlanGardeLaComposanteAncree(t *testing.T) {
	m := deuxCarres()
	cadre := himap.CalageFond{OrigineX: -1, OrigineY: 12, MetresParPixel: 0.1, Largeur: 300, Hauteur: 300}
	pf, err := calculePlan("t", m, [][3]float64{{0.5, 0.5, 0}}, cadre, func(x, y float64) bool { return x < 5 })
	if err != nil || pf == nil {
		t.Fatalf("calculePlan : %v, %v", pf, err)
	}
	if pf.Retenues.Gardees != 1 || pf.Retenues.Ecartees != 1 {
		t.Errorf("composantes = %d gardees / %d ecartees, attendu 1 / 1", pf.Retenues.Gardees, pf.Retenues.Ecartees)
	}
	if len(pf.Plan.Edges) != 4 || pf.Plan.Vertices != 8 || pf.Plan.Faces != 2 {
		t.Errorf("plan = %+v, attendu les 4 aretes du seul carre garde", pf.Plan)
	}
	if pf.Image.Bounds().Dx() != 300 || pf.Image.Bounds().Dy() != 300 {
		t.Errorf("image plan de %v, attendu 300x300", pf.Image.Bounds())
	}
	if pf, err := calculePlan("t", nil, nil, cadre, nil); pf != nil || err != nil {
		t.Errorf("sans maillage : %v, %v, attendu nil, nil", pf, err)
	}
}

// deuxCarres : deux faces carrees de 1 m, l'une en (0, 0), l'autre loin en (8, 8).
func deuxCarres() *hinavmesh.Maillage {
	return carresDeTest([][2]float64{{0, 0}, {8, 8}})
}

// -plan-seul refait le plan et l'image plan d'un sidecar publie : le fond et le calage ne
// bougent pas, le sidecar porte le plan et le nom de l'image plan, et l'image plan est ecrite.
func TestRefaitLePlanGardeLeFondEtPoseLImagePlan(t *testing.T) {
	dir := t.TempDir()
	fond := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 40; x++ {
			fond.SetRGBA(x, y, color.RGBA{R: 50, G: 50, B: 50, A: 255})
		}
	}
	f, err := os.Create(filepath.Join(dir, "carte.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, fond); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	avant := mapfondsidecar.MapBackground{MapBackground: replay.MapBackground{
		SchemaVersion: replay.MapBackgroundSchemaVersion, Module: "carte", Image: "carte.png", Source: "t",
		GeneratedAt: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), Style: "encre",
		Calibration: replay.MapBackgroundCalibration{MetersPerPixel: 0.1, OriginX: -1, OriginY: 12, WidthPx: 300, HeightPx: 300, Convention: himap.ConventionCalage},
	}}
	chemin := filepath.Join(dir, "carte.json")
	if err := mapfondsidecar.Write(chemin, avant); err != nil {
		t.Fatal(err)
	}
	e := &environnement{sortieDir: dir}
	var b bilanPlanSeul
	err = e.refaitLePlan(context.Background(),
		carteDuPlan{cle: "carte", noms: []string{"Carte"}, source: "t", ancres: [][3]float64{{0.5, 0.5, 0}}, maillage: deuxCarres()}, &b)
	if err != nil {
		t.Fatalf("refaitLePlan : %v", err)
	}
	apres, err := mapfondsidecar.Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if apres.Plan == nil || len(apres.Plan.Edges) != 4 || apres.PlanImage != "carte.plan.png" {
		t.Errorf("sidecar = plan %+v, image plan %q", apres.Plan, apres.PlanImage)
	}
	if apres.Image != "carte.png" || apres.Calibration != avant.Calibration || !apres.GeneratedAt.Equal(avant.GeneratedAt) {
		t.Errorf("le calage ou le fond a bouge : %+v", apres.MapBackground)
	}
	if _, err := os.Stat(filepath.Join(dir, "carte.plan.png")); err != nil {
		t.Errorf("image plan absente : %v", err)
	}
	if len(b.Bilans) != 1 || b.Bilans[0].Suivi.Action != actionPlanSeul {
		t.Errorf("bilans = %+v", b.Bilans)
	}
}

// Un fond dont les dimensions ne sont pas celles du calage est refuse : le plan ne se
// superposerait pas.
func TestRefaitLePlanRefuseUnFondDeAutreTaille(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "carte.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	s := mapfondsidecar.MapBackground{MapBackground: replay.MapBackground{
		SchemaVersion: replay.MapBackgroundSchemaVersion, Module: "carte", Image: "carte.png", Source: "t",
		Calibration: replay.MapBackgroundCalibration{MetersPerPixel: 0.1, WidthPx: 300, HeightPx: 300, Convention: himap.ConventionCalage},
	}}
	if err := mapfondsidecar.Write(filepath.Join(dir, "carte.json"), s); err != nil {
		t.Fatal(err)
	}
	e := &environnement{sortieDir: dir}
	var b bilanPlanSeul
	if err := e.refaitLePlan(context.Background(), carteDuPlan{cle: "carte", source: "t", maillage: deuxCarres()}, &b); err == nil {
		t.Fatal("un fond de 10x10 pour un calage de 300x300 a ete accepte")
	}
}
