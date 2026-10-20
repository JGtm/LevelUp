package main

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// construitImageSynthetique batit une image NRGBA 64x64 qui couvre les quatre profils de pixel
// que l'Etape 0 du plan impose de verifier : un aplat opaque, une zone totalement transparente,
// un aplat semi-transparent (alpha intermediaire) et un trait fin d'un seul pixel de large sur
// un aplat. Une image reelle de fond de carte melange ces quatre profils ; un encodeur qui
// perdrait un bit sur l'un d'eux romprait l'egalite octet a octet exigee par le plan.
//
// NRGBA (non premultiplie), pas RGBA : c'est le type concret que rend `png.Decode` sur un vrai
// fond de carte (RGBA avec alpha). Une image *image.RGBA construite a la main avec des valeurs
// choisies au hasard peut violer la contrainte du modele premultiplie (R/G/B <= A), ce que le
// premultiply/unpremultiply interne de l'encodeur ne peut pas restituer a l'identique — ce
// serait un artefact du test, pas une perte reelle de l'encodeur.
func construitImageSynthetique() *image.NRGBA {
	const cote = 64
	img := image.NewNRGBA(image.Rect(0, 0, cote, cote))
	traitY := cote - cote/4
	for y := 0; y < cote; y++ {
		for x := 0; x < cote; x++ {
			switch {
			case x < cote/2 && y < cote/2:
				img.Set(x, y, color.NRGBA{R: 200, G: 40, B: 40, A: 255}) // aplat opaque
			case x >= cote/2 && y < cote/2:
				img.Set(x, y, color.NRGBA{}) // zone totalement transparente
			case x < cote/2 && y >= cote/2:
				img.Set(x, y, color.NRGBA{R: 10, G: 60, B: 220, A: 120}) // semi-transparent
			case y == traitY:
				img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255}) // trait fin
			default:
				img.Set(x, y, color.NRGBA{R: 30, G: 160, B: 60, A: 255}) // aplat
			}
		}
	}
	return img
}

// TestAllerRetourWebP_ImageSynthetique_Identique est le coeur de l'Etape 0 : l'aller-retour
// PNG (en memoire) -> WebP sans perte -> RGBA doit rendre EXACTEMENT les memes pixels, sur les
// quatre profils construits ci-dessus.
func TestAllerRetourWebP_ImageSynthetique_Identique(t *testing.T) {
	img := construitImageSynthetique()

	res, encode, err := allerRetourWebP(img)
	if err != nil {
		t.Fatalf("aller-retour: %v", err)
	}
	if !res.Identique {
		t.Fatalf("aller-retour NON identique sur l'image synthetique (zones transparentes, trait fin, aplats)")
	}
	if len(encode) == 0 {
		t.Fatalf("encodage webp vide")
	}
	if res.OctetsWebP != len(encode) {
		t.Fatalf("OctetsWebP=%d, len(encode)=%d : incoherent", res.OctetsWebP, len(encode))
	}
	// Duree >= 0 seulement : sur une image 64x64, l'encodage/decodage peut descendre sous la
	// resolution de l'horloge (mesure a 0s sur certains systemes). La mesure fine se fait dans
	// le gate reel, sur les fonds de plusieurs Mo.
	if res.DureeEncode < 0 || res.DureeDecode < 0 {
		t.Fatalf("duree negative: encode=%v decode=%v", res.DureeEncode, res.DureeDecode)
	}
}

// TestVersRGBA_PreserveRect verifie que la canonicalisation ne deforme pas le cadre de l'image
// (Rect) : le plan exige l'egalite de Rect en plus de celle des Pix.
func TestVersRGBA_PreserveRect(t *testing.T) {
	img := construitImageSynthetique()
	out := versRGBA(img)
	if out.Rect != img.Rect {
		t.Fatalf("rect modifie par versRGBA: got %v want %v", out.Rect, img.Rect)
	}
}

// TestGainPct couvre le calcul de gain utilise par le mode -verifier, y compris le cas
// degenere (fichier vide) qui ne doit jamais diviser par zero.
func TestGainPct(t *testing.T) {
	cas := []struct {
		avant, apres int
		veut         float64
	}{
		{avant: 1000, apres: 500, veut: 50},
		{avant: 1000, apres: 1000, veut: 0},
		{avant: 0, apres: 0, veut: 0},
	}
	for _, c := range cas {
		got := gainPct(c.avant, c.apres)
		if got != c.veut {
			t.Fatalf("gainPct(%d,%d)=%v, veut %v", c.avant, c.apres, got, c.veut)
		}
	}
}

// TestConvertitUnFond_EcritWebpMetAJourSidecarSupprimePNG exerce le mode -convertir de bout en
// bout sur un repertoire temporaire synthetique — jamais sur `data/`. Il prouve que le WebP est
// ecrit, que le sidecar porte desormais le nom du WebP (D3) et que le PNG est supprime, SANS
// executer l'outil sur le parc reel (hors perimetre de l'Etape 0, decision D10 a venir).
func TestConvertitUnFond_EcritWebpMetAJourSidecarSupprimePNG(t *testing.T) {
	dir := t.TempDir()
	cheminPNG := filepath.Join(dir, "test_module.png")
	cheminJSON := filepath.Join(dir, "test_module.json")

	img := construitImageSynthetique()
	f, err := os.Create(cheminPNG)
	if err != nil {
		t.Fatalf("creation png: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("encodage png: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("fermeture png: %v", err)
	}

	sidecar := replay.MapBackground{
		SchemaVersion: replay.MapBackgroundSchemaVersion,
		Module:        "test_module",
		Image:         "test_module.png",
		Source:        "test",
		GeneratedAt:   time.Now().UTC(),
		Style:         "test",
	}
	blob, err := json.MarshalIndent(sidecar, "", "  ")
	if err != nil {
		t.Fatalf("serialisation sidecar: %v", err)
	}
	if err := os.WriteFile(cheminJSON, blob, 0o644); err != nil {
		t.Fatalf("ecriture sidecar: %v", err)
	}

	info, err := os.Stat(cheminPNG)
	if err != nil {
		t.Fatalf("stat png: %v", err)
	}
	fond := fondPNG{chemin: cheminPNG, octets: info.Size()}

	if err := convertitUnFond(context.Background(), fond); err != nil {
		t.Fatalf("convertitUnFond: %v", err)
	}

	if _, err := os.Stat(cheminPNG); !os.IsNotExist(err) {
		t.Fatalf("le PNG aurait du etre supprime (err=%v)", err)
	}
	cheminWebP := filepath.Join(dir, "test_module.webp")
	if _, err := os.Stat(cheminWebP); err != nil {
		t.Fatalf("le webp n'a pas ete ecrit: %v", err)
	}

	relu, err := replay.LoadMapBackground(cheminJSON)
	if err != nil {
		t.Fatalf("relecture sidecar: %v", err)
	}
	if relu.Image != "test_module.webp" {
		t.Fatalf("sidecar Image non mis a jour: got %q, veut test_module.webp", relu.Image)
	}
}

// TestSelectionnePNG_TrieParTailleDecroissanteEtLimite verifie l'echantillonnage utilise par le
// gate de l'Etape 0 (`-echantillon=5`) : les fichiers les plus gros d'abord, la liste tronquee.
func TestSelectionnePNG_TrieParTailleDecroissanteEtLimite(t *testing.T) {
	dir := t.TempDir()
	tailles := map[string]int{"petit.png": 10, "moyen.png": 100, "gros.png": 1000}
	for nom, taille := range tailles {
		if err := os.WriteFile(filepath.Join(dir, nom), make([]byte, taille), 0o644); err != nil {
			t.Fatalf("ecriture %s: %v", nom, err)
		}
	}

	tous, err := selectionnePNG(dir, 0)
	if err != nil {
		t.Fatalf("selectionnePNG(0): %v", err)
	}
	if len(tous) != 3 {
		t.Fatalf("attendu 3 fichiers, obtenu %d", len(tous))
	}
	if filepath.Base(tous[0].chemin) != "gros.png" {
		t.Fatalf("premier fichier attendu gros.png, obtenu %s", filepath.Base(tous[0].chemin))
	}

	deux, err := selectionnePNG(dir, 2)
	if err != nil {
		t.Fatalf("selectionnePNG(2): %v", err)
	}
	if len(deux) != 2 {
		t.Fatalf("attendu 2 fichiers avec echantillon=2, obtenu %d", len(deux))
	}
	if filepath.Base(deux[1].chemin) != "moyen.png" {
		t.Fatalf("second fichier attendu moyen.png, obtenu %s", filepath.Base(deux[1].chemin))
	}
}
