package himap

import (
	"image"
	"image/color"
	"testing"
)

// toileAvecTache rend une image transparente de w x h avec un carre opaque en (x,y,cote).
func toileAvecTache(w, h, x, y, cote int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for j := y; j < y+cote; j++ {
		for i := x; i < x+cote; i++ {
			img.Set(i, j, color.RGBA{200, 200, 200, 255})
		}
	}
	return img
}

func TestCadreUtileBorneLaMatiereAvecSaMarge(t *testing.T) {
	// 1 m/px : la marge de 6 m vaut 6 px, ce qui rend le calcul lisible a l'oeil.
	img := toileAvecTache(400, 300, 100, 80, 40)
	r, ok := CadreUtile(img, 1.0)
	if !ok {
		t.Fatal("matiere presente mais CadreUtile la declare vide")
	}
	want := image.Rect(94, 74, 146, 126)
	if r != want {
		t.Fatalf("cadre utile = %v, attendu %v", r, want)
	}
}

// LE GAIN EST L'OBJET MEME DU FICHIER : une carte dont la matiere occupe 10 % de la largeur
// doit sortir un cadre bien plus petit que la grille. Sans cette assertion, une implementation
// qui rendrait toujours l'image entiere passerait le test precedent en changeant la marge.
func TestCadreUtileRognereellementUneImageVide(t *testing.T) {
	img := toileAvecTache(2000, 2000, 900, 900, 200)
	r, ok := CadreUtile(img, 1.0)
	if !ok {
		t.Fatal("matiere presente mais declaree vide")
	}
	if r.Dx() >= img.Bounds().Dx()/2 {
		t.Fatalf("aucun gain de cadrage : %d px de large pour une matiere de 200 px", r.Dx())
	}
}

func TestCadreUtileEstBorneParLImage(t *testing.T) {
	// Matiere collee au bord : la marge ne doit pas sortir de l'image.
	img := toileAvecTache(100, 100, 0, 0, 100)
	r, ok := CadreUtile(img, 0.5) // marge = 12 px
	if !ok {
		t.Fatal("matiere presente mais declaree vide")
	}
	if r != img.Bounds() {
		t.Fatalf("cadre = %v, attendu l'image entiere %v", r, img.Bounds())
	}
}

func TestCadreUtileSurImageVide(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 50, 50))
	r, ok := CadreUtile(img, 1.0)
	if ok {
		t.Fatal("une image vide n'a pas de cadre utile")
	}
	if r != img.Bounds() {
		t.Fatalf("sur image vide, le cadre rendu doit rester l'image entiere, or %v", r)
	}
}

// LE PIEGE QUE CE TEST GARDE : rogner sans translater le calage decalerait TOUTES les positions
// du rejeu de la taille du rognage. L'echelle, elle, ne bouge pas.
func TestCalageRogneTranslateLOrigineEtPasLEchelle(t *testing.T) {
	const mpp = 0.092
	x0, y1 := 10.0, 200.0
	gx, gy := CalageRogne(x0, y1, mpp, image.Rect(100, 50, 900, 700))
	if wantX := 10.0 + 100*mpp; gx != wantX {
		t.Errorf("originX = %v, attendu %v", gx, wantX)
	}
	if wantY := 200.0 - 50*mpp; gy != wantY {
		t.Errorf("originY = %v, attendu %v", gy, wantY)
	}
	// Un rognage a l'origine ne bouge rien.
	if sx, sy := CalageRogne(x0, y1, mpp, image.Rect(0, 0, 10, 10)); sx != x0 || sy != y1 {
		t.Errorf("rognage a l'origine : calage %v/%v, attendu %v/%v", sx, sy, x0, y1)
	}
}
