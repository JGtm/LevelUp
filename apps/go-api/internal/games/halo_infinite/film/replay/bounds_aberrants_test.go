package replay

// bounds_aberrants_test.go — LES BORNES NE SE LAISSENT PLUS DEFINIR PAR UN POINT FAUX.
//
// LE DEFAUT MESURE (2026-09-08). `boundsOf` etait un min/max BRUT : sur l artefact `81c02726`
// (Isolement, Bases), UN echantillon sur 16 064 — (x=-78.6, y=+46.38, z=-325.4), quand le sol
// joue est a 117.8 en mediane — fixait a lui seul MinX, MaxY et MinZ. En aval, le client
// ecartait le fond de carte (`coversPlayedArea`) alors que l image couvre 99 % des positions,
// et cadrait la scene sur un point fantome.
//
// CE QUE CES TESTS FIXENT : le rejet EXISTE, il porte sur le bon point, il ne mord PAS sur le
// jeu legitime, et il s abstient quand il n a pas de quoi juger.

import "testing"

// pisteDe fabrique une piste depuis des triplets (x, y, z).
func pisteDe(pts [][3]float32) Track {
	out := make([]Point, 0, len(pts))
	for i, p := range pts {
		out = append(out, Point{T: i, X: p[0], Y: p[1], Z: p[2]})
	}
	return Track{Points: out}
}

// nuageRegulier rend n points repartis sur un terrain plat de 40 m, sol a 100 m.
func nuageRegulier(n int) [][3]float32 {
	out := make([][3]float32, 0, n)
	for i := 0; i < n; i++ {
		f := float32(i%40) - 20
		out = append(out, [3]float32{f, f / 2, 100 + float32(i%5)})
	}
	return out
}

func TestBoundsOf_EcarteLEchantillonAberrant(t *testing.T) {
	pts := nuageRegulier(1000)
	// LE POINT D ISOLEMENT, a l echelle de ce nuage : 440 m sous le sol joue.
	pts = append(pts, [3]float32{-78.6, 46.38, -325.4})

	b, ecartes := boundsOf([]Track{pisteDe(pts)})

	if ecartes != 1 {
		t.Fatalf("ecartes = %d, attendu 1", ecartes)
	}
	if b.MinX < -21 {
		t.Errorf("MinX = %v : le point aberrant definit encore la borne", b.MinX)
	}
	if b.MaxY > 11 {
		t.Errorf("MaxY = %v : le point aberrant definit encore la borne", b.MaxY)
	}
	if b.MinZ < 99 {
		t.Errorf("MinZ = %v : le point aberrant definit encore la borne", b.MinZ)
	}
}

func TestBoundsOf_NuagePropreInchange(t *testing.T) {
	pts := nuageRegulier(1000)

	b, ecartes := boundsOf([]Track{pisteDe(pts)})

	if ecartes != 0 {
		t.Fatalf("ecartes = %d sur un nuage propre, attendu 0", ecartes)
	}
	if b.MinX != -20 || b.MaxX != 19 {
		t.Errorf("bornes X = [%v, %v], attendu [-20, 19]", b.MinX, b.MaxX)
	}
}

func TestBoundsOf_LeJeuLegitimeSurvit(t *testing.T) {
	// UN ECART DE 9 ETENDUES : c est le plus grand ecart LEGITIME mesure sur le parc (9.5,
	// artefact `9e8fb31b`). Le seuil est a 12 — il doit passer, sans quoi on effacerait un
	// perchoir ou une chute dans un vide reel.
	pts := nuageRegulier(1000)
	// Etendue centrale de Z sur ce nuage : ~4 m. Neuf etendues = 36 m sous le p1.
	pts = append(pts, [3]float32{0, 0, 100 - 36})

	_, ecartes := boundsOf([]Track{pisteDe(pts)})

	if ecartes != 0 {
		t.Fatalf("ecartes = %d : un ecart legitime a ete pris pour un artefact", ecartes)
	}
}

func TestBoundsOf_SousLePlancherAucunRejet(t *testing.T) {
	// Trop peu de points pour que des centiles veuillent dire quoi que ce soit : on ne juge pas.
	pts := nuageRegulier(50)
	pts = append(pts, [3]float32{-500, 500, -900})

	b, ecartes := boundsOf([]Track{pisteDe(pts)})

	if ecartes != 0 {
		t.Fatalf("ecartes = %d sous le plancher d echantillons, attendu 0", ecartes)
	}
	if b.MinX != -500 {
		t.Errorf("MinX = %v : sous le plancher, les bornes restent BRUTES", b.MinX)
	}
}

func TestBoundsOf_TerrainPlatNeRejettePasLeDeplacement(t *testing.T) {
	// Tout le monde sur le meme plan : l etendue de Z est NULLE. Sans plancher d etendue, le
	// moindre pas en Z passerait pour un artefact.
	pts := make([][3]float32, 0, 1000)
	for i := 0; i < 1000; i++ {
		pts = append(pts, [3]float32{float32(i % 30), float32(i % 20), 12})
	}
	pts = append(pts, [3]float32{15, 10, 14}) // deux metres plus haut : une marche, pas un bug

	_, ecartes := boundsOf([]Track{pisteDe(pts)})

	if ecartes != 0 {
		t.Fatalf("ecartes = %d sur un terrain plat, attendu 0", ecartes)
	}
}

func TestBoundsOf_PisteVide(t *testing.T) {
	b, ecartes := boundsOf(nil)
	if ecartes != 0 {
		t.Fatalf("ecartes = %d sans piste, attendu 0", ecartes)
	}
	if b.MinX <= b.MaxX {
		t.Errorf("bornes = %+v : sans point, l etendue doit rester le signal d absence", b)
	}
}
