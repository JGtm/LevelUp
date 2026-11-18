package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain/replaydoc"
	"levelup/go-api/internal/mapfondsidecar"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
	"golang.org/x/image/webp"
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

// L'image plan `<cle>.plan.png` se convertit avec le sidecar `<cle>.json` du fond : le champ
// `planImage` prend le nom du WebP, le champ `image` et le plan ne bougent pas.
func TestConvertitUnFond_ImagePlanMetAJourPlanImage(t *testing.T) {
	dir := t.TempDir()
	cheminPlan := filepath.Join(dir, "carte.plan.png")
	f, err := os.Create(cheminPlan)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, construitImageSynthetique()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	cheminJSON := filepath.Join(dir, "carte.json")
	avant := mapfondsidecar.MapBackground{
		MapBackground: replay.MapBackground{
			SchemaVersion: replay.MapBackgroundSchemaVersion, Module: "carte", Image: "carte.webp",
			Source: "test", GeneratedAt: time.Now().UTC(), Style: "test",
		},
		PlanImage: "carte.plan.png",
		Plan:      &replaydoc.MapBackgroundPlan{Source: "t", Vertices: 4, Faces: 1, Edges: [][4]float64{{0, 0, 1, 0}}},
	}
	if err := mapfondsidecar.Write(cheminJSON, avant); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cheminPlan)
	if err != nil {
		t.Fatal(err)
	}
	if err := convertitUnFond(context.Background(), fondPNG{chemin: cheminPlan, octets: info.Size()}); err != nil {
		t.Fatalf("convertitUnFond: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "carte.plan.webp")); err != nil {
		t.Fatalf("le webp de l'image plan n'a pas ete ecrit: %v", err)
	}
	if _, err := os.Stat(cheminPlan); !os.IsNotExist(err) {
		t.Fatalf("le PNG plan aurait du etre supprime (err=%v)", err)
	}
	relu, err := mapfondsidecar.Load(cheminJSON)
	if err != nil {
		t.Fatal(err)
	}
	if relu.PlanImage != "carte.plan.webp" || relu.Image != "carte.webp" || relu.Plan == nil {
		t.Errorf("sidecar = image %q, planImage %q, plan %v", relu.Image, relu.PlanImage, relu.Plan)
	}
}

// construitImageRelief batit une image au format du relief : alpha 0 (hauteur inconnue, R, G, B nuls)
// ou 255 ; sur les pixels connus, R et G portent les deux octets de la hauteur, B les deux drapeaux.
// Les valeurs couvrent tout l'intervalle des octets, y compris 0 et 255 : c'est l'octet qui change qu'un
// encodeur qui toucherait aux couleurs sous une alpha pleine ferait voir.
func construitImageRelief() *image.NRGBA {
	const largeur, hauteur = 150, 90
	img := image.NewNRGBA(image.Rect(0, 0, largeur, hauteur))
	graine := uint32(12345)
	suivant := func() uint32 {
		graine = graine*1664525 + 1013904223
		return graine >> 8
	}
	for y := 0; y < hauteur; y++ {
		for x := 0; x < largeur; x++ {
			if (x/10+y/10)%3 == 0 { // hauteur inconnue
				continue
			}
			q := suivant() & 0xFFFF
			if x == 0 {
				q = 0xFFFF
			}
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(q & 0xFF), G: uint8(q >> 8), B: uint8(suivant() & 3), A: 255})
		}
	}
	return img
}

// strictementIdentique dit si `lu` rend exactement, pixel par pixel et canal par canal, les valeurs
// non prémultipliées de `attendu` : c'est ce que relit le navigateur du relief.
func strictementIdentique(attendu *image.NRGBA, lu image.Image) bool {
	if lu.Bounds() != attendu.Bounds() {
		return false
	}
	b := attendu.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if color.NRGBAModel.Convert(lu.At(x, y)) != attendu.NRGBAAt(x, y) {
				return false
			}
		}
	}
	return true
}

// L'image du relief ressort du WebP identique au bit près sur ses quatre canaux, alpha 0 et 255
// comprises, que la comparaison soit celle de l'outil (prémultipliée) ou stricte (non prémultipliée).
func TestAllerRetourWebP_ImageRelief_IdentiqueSurLesQuatreCanaux(t *testing.T) {
	img := construitImageRelief()
	r, encode, err := allerRetourWebP(img)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Identique {
		t.Fatal("l'aller-retour de l'outil n'est pas identique")
	}
	lu, err := webp.Decode(bytes.NewReader(encode))
	if err != nil {
		t.Fatal(err)
	}
	if !strictementIdentique(img, lu) {
		t.Error("le WebP ne rend pas les valeurs non prémultipliées du relief")
	}
}

// Le mode -convertir prend `<cle>.relief.png` : le WebP du relief est écrit, le champ `reliefImage` du
// sidecar du fond prend son nom, l'image du fond, l'image plan et le plan ne bougent pas, le PNG est
// supprimé.
func TestConvertitUnFond_ImageReliefMetAJourReliefImage(t *testing.T) {
	dir := t.TempDir()
	cheminRelief := filepath.Join(dir, "carte.relief.png")
	img := construitImageRelief()
	f, err := os.Create(cheminRelief)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	cheminJSON := filepath.Join(dir, "carte.json")
	avant := mapfondsidecar.MapBackground{
		MapBackground: replay.MapBackground{
			SchemaVersion: replay.MapBackgroundSchemaVersion, Module: "carte", Image: "carte.webp",
			Source: "test", GeneratedAt: time.Now().UTC(), Style: "test",
		},
		PlanImage:   "carte.plan.webp",
		Plan:        &replaydoc.MapBackgroundPlan{Source: "t", Vertices: 4, Faces: 1, Edges: [][4]float64{{0, 0, 1, 0}}},
		ReliefImage: "carte.relief.png",
		ReliefData:  "carte.relief.json",
	}
	if err := mapfondsidecar.Write(cheminJSON, avant); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cheminRelief)
	if err != nil {
		t.Fatal(err)
	}
	if err := convertitUnFond(context.Background(), fondPNG{chemin: cheminRelief, octets: info.Size()}); err != nil {
		t.Fatalf("convertitUnFond: %v", err)
	}
	if _, err := os.Stat(cheminRelief); !os.IsNotExist(err) {
		t.Fatalf("le PNG du relief aurait du etre supprime (err=%v)", err)
	}
	fw, err := os.Open(filepath.Join(dir, "carte.relief.webp"))
	if err != nil {
		t.Fatalf("le webp du relief n'a pas ete ecrit: %v", err)
	}
	defer fw.Close()
	lu, err := webp.Decode(fw)
	if err != nil {
		t.Fatal(err)
	}
	if !strictementIdentique(img, lu) {
		t.Error("le WebP ecrit ne rend pas le relief au bit pres")
	}
	relu, err := mapfondsidecar.Load(cheminJSON)
	if err != nil {
		t.Fatal(err)
	}
	if relu.ReliefImage != "carte.relief.webp" || relu.ReliefData != "carte.relief.json" ||
		relu.Image != "carte.webp" || relu.PlanImage != "carte.plan.webp" || relu.Plan == nil {
		t.Errorf("sidecar = image %q, planImage %q, reliefImage %q, reliefData %q, plan %v",
			relu.Image, relu.PlanImage, relu.ReliefImage, relu.ReliefData, relu.Plan)
	}
}

// Le sidecar d'une image se déduit de son nom : le fond et son sidecar, l'image plan et celle du relief
// avec le sidecar du fond, chacune par son champ.
func TestSidecarDuFond_ChoisitLeChampParLeNom(t *testing.T) {
	cas := []struct {
		fichier string
		champ   champImage
	}{
		{"carte.png", champFond}, {"carte.plan.png", champPlan}, {"carte.relief.png", champRelief},
	}
	for _, c := range cas {
		sidecar, champ := sidecarDuFond(filepath.Join("dossier", c.fichier))
		if sidecar != filepath.Join("dossier", "carte.json") || champ != c.champ {
			t.Errorf("%s : sidecar %q, champ %d ; attendu dossier/carte.json, champ %d", c.fichier, sidecar, champ, c.champ)
		}
	}
}
