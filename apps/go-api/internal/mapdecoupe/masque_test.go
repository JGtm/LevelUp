package mapdecoupe

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay"
)

// masqueTest fabrique un masque synthétique calé en (0, 0), un mètre par cellule — les
// coordonnées monde et les indices de cellule se lisent alors directement.
func masqueTest(t *testing.T, nx, ny int, dur []bool) *Masque {
	t.Helper()
	if len(dur) != nx*ny {
		t.Fatalf("masque mal dimensionné : %d cellules pour %dx%d", len(dur), nx, ny)
	}
	return &Masque{
		Module: "temoin",
		Calage: replay.MapBackgroundCalibration{
			MetersPerPixel: 1, OriginX: 0, OriginY: float64(ny), WidthPx: nx, HeightPx: ny,
		},
		NX: nx, NY: ny, dur: dur,
	}
}

func TestPraticableSuitLeCalagePublie(t *testing.T) {
	// Une seule cellule dure, la (2, 1) : centre monde (2,5 ; 2,5) avec OriginY = 4.
	dur := make([]bool, 4*4)
	dur[1*4+2] = true
	m := masqueTest(t, 4, 4, dur)

	if !m.Praticable(2.5, 2.5) {
		t.Error("le centre de la cellule dure est déclaré non praticable")
	}
	if m.Praticable(1.5, 2.5) {
		t.Error("la cellule voisine, vide, est déclarée praticable")
	}
	if m.Praticable(2.5, 100) {
		t.Error("un point hors cadre est déclaré praticable")
	}
}

func TestCombleBoucheUnTrouInterieurEtNEnleveRien(t *testing.T) {
	// Grille pleine, un trou d'une cellule au centre.
	dur := make([]bool, 9*9)
	for k := range dur {
		dur[k] = true
	}
	dur[4*9+4] = false
	m := masqueTest(t, 9, 9, dur)

	ferme := m.Comble(1.5)
	if !ferme.dur[4*9+4] {
		t.Error("la fermeture n'a pas bouché le trou d'une cellule")
	}
	for k, avant := range m.dur {
		if avant && !ferme.dur[k] {
			t.Fatalf("la fermeture a ENLEVÉ de la matière en %d — elle ne doit qu'ajouter", k)
		}
	}
}

func TestCombleNeRelieQueCeQuiEstProche(t *testing.T) {
	// Deux blocs pleins séparés par un couloir vide de 5 cellules.
	dur := make([]bool, 12*3)
	for j := 0; j < 3; j++ {
		for i := 0; i < 12; i++ {
			dur[j*12+i] = i < 3 || i > 7
		}
	}
	m := masqueTest(t, 12, 3, dur)

	if m.Comble(1).dur[1*12+5] {
		t.Error("un rayon de 1 a comblé un couloir de 5 cellules")
	}
	if !m.Comble(3).dur[1*12+5] {
		t.Error("un rayon de 3 n'a pas comblé un couloir de 5 cellules")
	}
}

func TestCombleRayonNulRendLeMasqueTelQuel(t *testing.T) {
	dur := make([]bool, 4)
	dur[0] = true
	m := masqueTest(t, 2, 2, dur)
	if m.Comble(0) != m {
		t.Error("un rayon nul doit rendre le masque lui-même, sans copie ni calcul")
	}
}

func TestChargeMasqueRefuseUneImageQuiContreditSonCalage(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	img.Set(1, 1, color.NRGBA{R: 200, G: 200, B: 200, A: 255})
	pngPath := filepath.Join(dir, "temoin.png")
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("création du PNG : %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encodage du PNG : %v", err)
	}
	f.Close()

	metaPath := filepath.Join(dir, "temoin.json")
	ecritSidecar(t, metaPath, 3, 3)
	if _, err := ChargeMasque(pngPath, metaPath); err != nil {
		t.Fatalf("un couple cohérent doit se charger : %v", err)
	}

	ecritSidecar(t, metaPath, 4, 3)
	if _, err := ChargeMasque(pngPath, metaPath); err == nil {
		t.Fatal("une image qui contredit son calage doit être REFUSÉE, pas décalée en silence")
	}
}

func TestChargeMasqueLitLAlphaCommeMatiere(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.NRGBA{R: 10, G: 10, B: 10, A: 0})   // transparent = pas de matière
	img.Set(1, 0, color.NRGBA{R: 10, G: 10, B: 10, A: 255}) // opaque = matière
	pngPath := filepath.Join(dir, "temoin.png")
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("création du PNG : %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encodage du PNG : %v", err)
	}
	f.Close()
	metaPath := filepath.Join(dir, "temoin.json")
	ecritSidecar(t, metaPath, 2, 1)

	m, err := ChargeMasque(pngPath, metaPath)
	if err != nil {
		t.Fatalf("ChargeMasque : %v", err)
	}
	if m.dur[0] || !m.dur[1] {
		t.Errorf("alpha mal lu : dur = %v, attendu [false true]", m.dur)
	}
	if got := m.PartDure(); got != 0.5 {
		t.Errorf("PartDure = %v, attendu 0,5", got)
	}
}

func ecritSidecar(t *testing.T, path string, w, h int) {
	t.Helper()
	blob := []byte(`{"schemaVersion":1,"module":"temoin","image":"temoin.png","source":"test",
"style":"jeu","calibration":{"metersPerPixel":1,"originX":0,"originY":` +
		itoa(h) + `,"widthPx":` + itoa(w) + `,"heightPx":` + itoa(h) + `,"convention":"x"},"stats":{}}`)
	if err := os.WriteFile(path, blob, 0o644); err != nil {
		t.Fatalf("écriture du sidecar : %v", err)
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b []byte
	for v > 0 {
		b = append([]byte{byte('0' + v%10)}, b...)
		v /= 10
	}
	return string(b)
}
