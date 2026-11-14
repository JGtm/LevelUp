package main

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// anneau : un carre de 10 m percé d'un trou de 2 m en son centre, en quatre faces.
func anneau() *hinavmesh.Maillage {
	m := &hinavmesh.Maillage{}
	for _, r := range [][4]float64{{0, 0, 10, 4}, {0, 6, 10, 10}, {0, 4, 4, 6}, {6, 4, 10, 6}} {
		base := int32(len(m.Sommets))
		m.Sommets = append(m.Sommets, hinavmesh.Point{X: r[0], Y: r[1]}, hinavmesh.Point{X: r[2], Y: r[1]},
			hinavmesh.Point{X: r[2], Y: r[3]}, hinavmesh.Point{X: r[0], Y: r[3]})
		m.Faces = append(m.Faces, hinavmesh.Face{Sommets: []int32{base, base + 1, base + 2, base + 3}})
	}
	return m
}

// Le masque vient du maillage : composantes gardees, marge resolue, trous interieurs remplis,
// dans la grille du fond (py vers le bas). Un trou de l'anneau est dans le masque (le rocher
// qui s'y dresse reste dessine) ; la marge de 1 m s'arrete a 1 m ; le nord du monde est en haut.
func TestMasqueSilhouetteDuMaillageRempliLesTrousEtSuitLeCalage(t *testing.T) {
	cal := himap.CalageFond{OrigineX: -2, OrigineY: 12, MetresParPixel: 0.5, Largeur: 28, Hauteur: 28}
	masque, err := masqueSilhouetteDuMaillage(anneau(), nil, cal, 1)
	if err != nil {
		t.Fatal(err)
	}
	dans := func(x, y float64) bool {
		px, py := int((x-cal.OrigineX)/cal.MetresParPixel), int((cal.OrigineY-y)/cal.MetresParPixel)
		return masque[py*cal.Largeur+px]
	}
	for nom, c := range map[string]struct {
		x, y float64
		want bool
	}{
		"centre du trou": {5, 5, true}, "face": {2, 2, true}, "marge": {-0.75, 5, true},
		"hors marge": {-1.75, 5, false}, "loin": {11.9, 11.9, false}, "au nord": {5, 10.75, true}, "au nord hors marge": {5, 11.75, false},
	} {
		if got := dans(c.x, c.y); got != c.want {
			t.Errorf("%s (%g, %g) : %v, attendu %v", nom, c.x, c.y, got, c.want)
		}
	}
	// Un maillage reduit aux faces gardees : le masque ne couvre pas les faces ecartees.
	garde := []bool{true, false, false, false}
	m2, err := masqueSilhouetteDuMaillage(anneau(), garde, cal, 0)
	if err != nil {
		t.Fatal(err)
	}
	if m2[int((cal.OrigineY-2)/cal.MetresParPixel)*cal.Largeur+int((2-cal.OrigineX)/cal.MetresParPixel)] != true ||
		m2[int((cal.OrigineY-8)/cal.MetresParPixel)*cal.Largeur+int((2-cal.OrigineX)/cal.MetresParPixel)] {
		t.Error("le masque ne suit pas les faces gardees")
	}
}

func TestEffaceHorsMasqueEtAncres(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 9, G: 9, B: 9, A: 255})
		}
	}
	masque := make([]bool, 100)
	for y := 0; y < 10; y++ {
		for x := 0; x < 5; x++ {
			masque[y*10+x] = true
		}
	}
	cal := himap.CalageFond{OrigineX: 0, OrigineY: 10, MetresParPixel: 1, Largeur: 10, Hauteur: 10}
	ancres := [][3]float64{{1.5, 5, 0}, {8.5, 5, 0}}
	if got := ancresAvecMatiere(img, cal, ancres, 1); got != 2 {
		t.Fatalf("avant : %d ancres, attendu 2", got)
	}
	if retire := effaceHorsMasque(img, masque); retire != 50 {
		t.Errorf("retire = %d, attendu 50", retire)
	}
	if got := ancresAvecMatiere(img, cal, ancres, 1); got != 1 {
		t.Errorf("apres : %d ancres, attendu 1", got)
	}
	if got := compteMatiere(img); got != 50 {
		t.Errorf("matiere restante = %d, attendu 50", got)
	}
}

// -plan-seul -rogne-publie : le fond est rogne a la silhouette dilatee, sans changer ses
// dimensions ni son calage ; le sidecar le dit, le fond est ecrit en PNG, un second passage ne
// rogne plus ; une carte que l'avis laisse n'est pas rognee.
func TestRogneLePublieALaSilhouetteSansToucherAuCalage(t *testing.T) {
	dir := t.TempDir()
	fond := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 300; x++ {
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
		SchemaVersion: replay.MapBackgroundSchemaVersion, Module: "carte", Image: "carte.png", Source: "cuisson",
		GeneratedAt: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), Style: "encre",
		Calibration: replay.MapBackgroundCalibration{MetersPerPixel: 0.1, OriginX: -1, OriginY: 12, WidthPx: 300, HeightPx: 300, Convention: himap.ConventionCalage},
		Stats:       replay.MapBackgroundStats{Anchors: 1, AnchorsInFrame: 1, AnchorsWithGround: 1},
	}}
	chemin := filepath.Join(dir, "carte.json")
	if err := mapfondsidecar.Write(chemin, avant); err != nil {
		t.Fatal(err)
	}
	a := avisChargeDeTest(t, avisDeTest)
	e := &environnement{sortieDir: dir, avis: a, rognePublie: true, reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}
	carte := carteDuPlan{cle: "carte", noms: []string{"Carte"}, source: "t", ancres: [][3]float64{{0.5, 0.5, 0}}, maillage: deuxCarres()}
	var b bilanPlanSeul
	if err := e.refaitLePlan(context.Background(), carte, &b); err != nil {
		t.Fatalf("refaitLePlan : %v", err)
	}
	apres, err := mapfondsidecar.Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if apres.Calibration != avant.Calibration || apres.Image != "carte.png" {
		t.Errorf("calage ou image changes : %+v", apres.MapBackground)
	}
	if !strings.Contains(apres.Source, marqueRogne) || !apres.GeneratedAt.After(avant.GeneratedAt) || apres.Stats.CellsClipped == 0 {
		t.Errorf("sidecar non mis a jour : source %q, date %v, cellules %d", apres.Source, apres.GeneratedAt, apres.Stats.CellsClipped)
	}
	rogne := lisPNGDeTest(t, filepath.Join(dir, "carte.png"))
	if rogne.Bounds() != fond.Bounds() {
		t.Errorf("dimensions changees : %v", rogne.Bounds())
	}
	if rogne.RGBAAt(150, 150).A != 0 {
		t.Error("matiere loin de la silhouette (150,150) restee")
	}
	if rogne.RGBAAt(15, 115).A == 0 { // le carre en (0,0) est aux pixels (10..20, 110..120)
		t.Error("matiere de la composante retenue effacee")
	}
	if len(b.Bilans) != 1 || b.Bilans[0].Suivi.Action != actionPublieRogne || b.Bilans[0].Suivi.PartRognee == nil {
		t.Fatalf("bilans = %+v", b.Bilans)
	}
	// Second passage : deja rogne, rien ne change.
	b2 := bilanPlanSeul{}
	if err := e.refaitLePlan(context.Background(), carte, &b2); err != nil {
		t.Fatal(err)
	}
	if b2.Bilans[0].Suivi.Action != actionPlanSeul {
		t.Errorf("second passage : action %q, attendu plan seul", b2.Bilans[0].Suivi.Action)
	}
}

func lisPNGDeTest(t *testing.T, chemin string) *image.RGBA {
	t.Helper()
	f, err := os.Open(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	rgba := image.NewRGBA(img.Bounds())
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return rgba
}

// Une carte que l'avis laisse (publie), ou qui cuit sans son maillage, n'est pas rognee ; un mode
// autre que silhouette est refuse plutot que rogne dans une autre forme.
func TestRognePublieALaisseLesCartesPubliees(t *testing.T) {
	a := avisChargeDeTest(t, avisDeTest)
	e := &environnement{avis: a, reglages: &reglagesFond{Cartes: map[string]reglageCarte{
		"sans": {SansNavmesh: true, GateLe: "2026-11-13"},
	}}}
	fond := image.NewRGBA(image.Rect(0, 0, 10, 10))
	cal := himap.CalageFond{OrigineX: 0, OrigineY: 10, MetresParPixel: 1, Largeur: 10, Hauteur: 10}
	pf := &planFond{Image: image.NewRGBA(image.Rect(0, 0, 10, 10))}
	for _, cle := range []string{"publiee", "sans-maillage", "sans"} {
		if _, ok, err := e.rognePublieA(context.Background(), carteDuPlan{cle: cle}, fond, cal, pf); ok || err != nil {
			t.Errorf("%s : ok=%v err=%v, attendu laissee sans erreur", cle, ok, err)
		}
	}
	if _, ok, err := e.rognePublieA(context.Background(), carteDuPlan{cle: "faces"}, fond, cal, pf); ok || err == nil {
		t.Errorf("mode faces : ok=%v err=%v, attendu une erreur", ok, err)
	}
}
