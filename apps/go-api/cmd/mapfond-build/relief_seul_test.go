package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
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

	"github.com/JGtm/halo-infinite-map-images/himap"

	"github.com/JGtm/halo-infinite-files-reader/hinavmesh"
)

// Le cadre des essais : 300 x 300 pixels de 0,1 m, le coin haut gauche en (-1, 12). Le fond porte de la
// matière sur ses 40 premières colonnes (x de -1 à 3), le maillage est un carré de 1 m en (0, 0), le
// rendu un plateau à 1,5 m sur x de -1 à 3 et y de -1 à 3.
const (
	coteEssai   = 300
	pasEssai    = 0.1
	origineXEss = -1.0
	origineYEss = 12.0
)

// calageEssai est le calage du fond publié des essais.
func calageEssai() himap.CalageFond {
	return himap.CalageFond{OrigineX: origineXEss, OrigineY: origineYEss, MetresParPixel: pasEssai, Largeur: coteEssai, Hauteur: coteEssai}
}

// renduEssai cuit en mémoire un plateau à 1,5 m ; `cellule` est le pas de la grille (celui du calage
// publié pour un rendu cohérent).
func renduEssai(cellule float64) *himap.Rendu {
	r := himap.NewRendu([2]float64{origineXEss, origineYEss - coteEssai*pasEssai}, [2]float64{origineXEss + coteEssai*pasEssai, origineYEss}, cellule)
	plateau := &himap.Mesh{
		Vertices:  [][3]float64{{-1, -1, 1.5}, {3, -1, 1.5}, {3, 3, 1.5}, {-1, 3, 1.5}},
		Triangles: [][3]int{{0, 1, 2}, {0, 2, 3}},
	}
	r.AddMesh(plateau, himap.Instance{Scale: [3]float64{1, 1, 1}, Forward: [3]float64{1, 0, 0}, Left: [3]float64{0, 1, 0}, Up: [3]float64{0, 0, 1}})
	return r
}

// publieUnFondEssai écrit dans `dir` un fond publié (`carte.png`, matière sur les 40 premières colonnes)
// et son sidecar, plan des murs et image plan compris ; rend le contenu du sidecar écrit.
func publieUnFondEssai(t *testing.T, dir string) []byte {
	t.Helper()
	fond := image.NewRGBA(image.Rect(0, 0, coteEssai, coteEssai))
	for y := 0; y < coteEssai; y++ {
		for x := 0; x < 40; x++ {
			fond.SetRGBA(x, y, color.RGBA{R: 50, G: 50, B: 50, A: 255})
		}
	}
	if _, err := ecritPNG(filepath.Join(dir, "carte.png"), fond); err != nil {
		t.Fatal(err)
	}
	s := mapfondsidecar.MapBackground{
		MapBackground: replay.MapBackground{
			SchemaVersion: replay.MapBackgroundSchemaVersion, Module: "carte", Image: "carte.png", Source: "t",
			GeneratedAt: time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC), Style: "encre",
			MapNames: []string{"Carte"},
			Calibration: replay.MapBackgroundCalibration{MetersPerPixel: pasEssai, OriginX: origineXEss, OriginY: origineYEss,
				WidthPx: coteEssai, HeightPx: coteEssai, Convention: himap.ConventionCalage},
			Degradations: []string{"une dégradation de la cuisson"},
		},
		PlanImage: "carte.plan.png",
		Plan:      &replaydoc.MapBackgroundPlan{Source: "t", Vertices: 4, Faces: 1, Edges: [][4]float64{{0, 0, 1, 0}, {1, 0, 1, 1}}},
	}
	chemin := filepath.Join(dir, "carte.json")
	if err := mapfondsidecar.Write(chemin, s); err != nil {
		t.Fatal(err)
	}
	octets, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	return octets
}

// cuisEssai rend une `cuire` qui rend le rendu et le maillage donnés.
func cuisEssai(r *himap.Rendu, m *hinavmesh.Maillage) func() (carteCuite, error) {
	return func() (carteCuite, error) {
		return carteCuite{c: cible{cle: "carte", ancres: [][3]float64{{0.5, 0.5, 1.5}}, maillage: m}, rendu: r}, nil
	}
}

// sansLeRelief rend le sidecar `chemin` réécrit sans ses deux champs du relief.
func sansLeRelief(t *testing.T, chemin string) []byte {
	t.Helper()
	s, err := mapfondsidecar.Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	s.ReliefImage, s.ReliefData = "", ""
	copie := filepath.Join(t.TempDir(), "copie.json")
	if err := mapfondsidecar.Write(copie, *s); err != nil {
		t.Fatal(err)
	}
	octets, err := os.ReadFile(copie)
	if err != nil {
		t.Fatal(err)
	}
	return octets
}

// lisLeRelief relit l'image et les données du relief écrits dans `dir`.
func lisLeRelief(t *testing.T, dir string, s *mapfondsidecar.MapBackground) (*image.NRGBA, himap.DonneesRelief) {
	t.Helper()
	f, err := os.Open(filepath.Join(dir, s.ReliefImage))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		t.Fatalf("image du relief de type %T, attendu NRGBA", img)
	}
	fz, err := os.Open(filepath.Join(dir, s.ReliefData))
	if err != nil {
		t.Fatal(err)
	}
	defer fz.Close()
	zr, err := gzip.NewReader(fz)
	if err != nil {
		t.Fatalf("les données du relief ne sont pas du gzip : %v", err)
	}
	var d himap.DonneesRelief
	if err := json.NewDecoder(zr).Decode(&d); err != nil {
		t.Fatal(err)
	}
	if d.SchemaVersion != himap.SchemaVersionRelief {
		t.Errorf("schemaVersion = %d, attendu %d", d.SchemaVersion, himap.SchemaVersionRelief)
	}
	return nrgba, d
}

// -relief-seul n'ajoute au sidecar que ses deux champs : tout le reste (fond, calage, plan, image plan,
// statistiques, date) reste identique à l'octet. Le relief est dans le calage du fond publié, et il a
// des pixels connus portant le drapeau « sur la carte ».
func TestRefaitLeReliefNeChangeQueLesDeuxChampsDuSidecar(t *testing.T) {
	dir := t.TempDir()
	avant := publieUnFondEssai(t, dir)
	fondAvant, err := os.ReadFile(filepath.Join(dir, "carte.png"))
	if err != nil {
		t.Fatal(err)
	}
	e := &environnement{sortieDir: dir}
	var b bilanSeul
	err = e.refaitLeRelief(context.Background(), "carte", []string{"Carte"}, &b,
		cuisEssai(renduEssai(pasEssai), carresDeTest([][2]float64{{0, 0}})))
	if err != nil {
		t.Fatalf("refaitLeRelief : %v", err)
	}
	chemin := filepath.Join(dir, "carte.json")
	apres, err := mapfondsidecar.Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	if apres.ReliefImage != "carte.relief.png" || apres.ReliefData != "carte.relief.json.gz" {
		t.Errorf("champs du relief = %q, %q", apres.ReliefImage, apres.ReliefData)
	}
	if !bytes.Equal(sansLeRelief(t, chemin), avant) {
		t.Error("un autre champ du sidecar a change")
	}
	if fondApres, _ := os.ReadFile(filepath.Join(dir, "carte.png")); !bytes.Equal(fondApres, fondAvant) {
		t.Error("l'image du fond a change")
	}
	img, donnees := lisLeRelief(t, dir, apres)
	if img.Bounds().Dx() != coteEssai || img.Bounds().Dy() != coteEssai {
		t.Errorf("image du relief de %v, attendu %dx%d", img.Bounds(), coteEssai, coteEssai)
	}
	cal := donnees.Calibration
	if cal.MetresParPixel != pasEssai || cal.OrigineX != origineXEss || cal.OrigineY != origineYEss ||
		cal.Largeur != coteEssai || cal.Hauteur != coteEssai {
		t.Errorf("calage du relief = %+v, attendu celui du sidecar publié", cal)
	}
	surCarte, inconnus := 0, 0
	for k := 0; k < len(img.Pix); k += 4 {
		switch {
		case img.Pix[k+3] == 0 && img.Pix[k]|img.Pix[k+1]|img.Pix[k+2] != 0:
			t.Fatalf("pixel %d : alpha 0 et couleurs non nulles", k/4)
		case img.Pix[k+3] == 0:
			inconnus++
		case img.Pix[k+2]&himap.DrapeauSurCarte != 0:
			surCarte++
		}
	}
	if surCarte == 0 || inconnus == 0 {
		t.Errorf("pixels sur la carte = %d, hauteur inconnue = %d : le relief ne couvre ni la carte ni son dehors", surCarte, inconnus)
	}
	if len(b.Bilans) != 1 || b.Bilans[0].Suivi.Action != actionReliefSeul {
		t.Errorf("bilans = %+v", b.Bilans)
	}
}

// retireLePlanDuSidecar réécrit le sidecar publié de `dir` sans plan des murs ni image plan : une carte
// qui n'a pas de maillage n'a jamais eu de plan.
func retireLePlanDuSidecar(t *testing.T, dir string) {
	t.Helper()
	chemin := filepath.Join(dir, "carte.json")
	s, err := mapfondsidecar.Load(chemin)
	if err != nil {
		t.Fatal(err)
	}
	s.Plan, s.PlanImage = nil, ""
	if err := mapfondsidecar.Write(chemin, *s); err != nil {
		t.Fatal(err)
	}
}

// Une carte sans maillage et sans plan publié reçoit un relief, avec la dégradation « sans-maillage » (D-7).
func TestRefaitLeReliefSansMaillageDegrade(t *testing.T) {
	dir := t.TempDir()
	publieUnFondEssai(t, dir)
	retireLePlanDuSidecar(t, dir)
	e := &environnement{sortieDir: dir}
	var b bilanSeul
	if err := e.refaitLeRelief(context.Background(), "carte", nil, &b, cuisEssai(renduEssai(pasEssai), nil)); err != nil {
		t.Fatalf("refaitLeRelief : %v", err)
	}
	s, err := mapfondsidecar.Load(filepath.Join(dir, "carte.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, donnees := lisLeRelief(t, dir, s)
	if len(donnees.Degradations) != 1 || donnees.Degradations[0] != himap.DegradationSansMaillage {
		t.Errorf("degradations = %v, attendu [%s]", donnees.Degradations, himap.DegradationSansMaillage)
	}
}

// Un fond dont le sidecar publié porte un plan a un maillage : une cuisson qui n'en trouve pas (blob absent
// ou illisible sur le poste) met la carte en échec nommé, rien n'est écrit et le code de sortie n'est pas
// nul. Sans la garde, la carte recevrait un relief « sans-maillage » et le code 0.
// Mutation jouée : retirer verifieLeMaillageDuPlanPublie de refaitLeRelief.
func TestRefaitLeReliefSansMaillageAvecPlanPublieEstUnEchecQuiNEcritRien(t *testing.T) {
	dir := t.TempDir()
	avant := publieUnFondEssai(t, dir)
	e := &environnement{sortieDir: dir}
	var b bilanSeul
	err := e.refaitLeRelief(context.Background(), "carte", nil, &b, cuisEssai(renduEssai(pasEssai), nil))
	if !errors.Is(err, errPlanPublieSansMaillage) {
		t.Fatalf("erreur = %v, attendu errPlanPublieSansMaillage", err)
	}
	if errors.Is(err, os.ErrNotExist) {
		t.Fatal("une carte sans maillage n'est pas une carte laissée intacte")
	}
	if apres, _ := os.ReadFile(filepath.Join(dir, "carte.json")); !bytes.Equal(apres, avant) {
		t.Error("le sidecar a été réécrit")
	}
	for _, nom := range []string{"carte.relief.png", "carte.relief.json.gz"} {
		if _, err := os.Stat(filepath.Join(dir, nom)); !os.IsNotExist(err) {
			t.Errorf("%s écrit malgré l'échec (%v)", nom, err)
		}
	}
	e.noteResultat(context.Background(), &b, "carte", nil, err)
	if b.Echecs != 1 {
		t.Errorf("échecs = %d, ajoutés = %d : attendu 1 échec", b.Echecs, b.Ajoutes)
	}
	if code := termineUnGesteSeul(context.Background(), e, rapportsDemandes{}, "relief seul termine", b); code == 0 {
		t.Error("code de sortie 0 malgré une carte en échec")
	}
}

// La cuisson complète a la même garde : le sidecar déjà publié dans le dossier de sortie porte un plan, la
// cuisson n'a pas de maillage ; ni le fond ni le sidecar ne sont réécrits. Une carte sans sidecar publié
// (premier jet) ou sans plan publié passe : c'est une carte qui n'a vraiment pas de maillage.
// Mutation jouée : retirer verifieLeMaillageAvantPublication de publie.
func TestPublieSansMaillageAvecPlanPublieNEcritRien(t *testing.T) {
	dir := t.TempDir()
	avant := publieUnFondEssai(t, dir)
	fondAvant, err := os.ReadFile(filepath.Join(dir, "carte.png"))
	if err != nil {
		t.Fatal(err)
	}
	e := &environnement{sortieDir: dir, style: himap.StyleFondParDefaut, reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}
	c := cible{cle: "carte", noms: []string{"Carte"}, ancres: [][3]float64{{0.5, 0.5, 1.5}}, sourcePlan: sourcePlanNatif}
	ba := e.publie(context.Background(), c, renduEssai(pasEssai), himap.BilanCuisson{})
	if !errors.Is(ba.Err, errPlanPublieSansMaillage) {
		t.Fatalf("erreur = %v, attendu errPlanPublieSansMaillage", ba.Err)
	}
	if apres, _ := os.ReadFile(filepath.Join(dir, "carte.json")); !bytes.Equal(apres, avant) {
		t.Error("le sidecar a été réécrit")
	}
	if fondApres, _ := os.ReadFile(filepath.Join(dir, "carte.png")); !bytes.Equal(fondApres, fondAvant) {
		t.Error("le fond publié a été réécrit")
	}
	if _, err := os.Stat(filepath.Join(dir, "carte.relief.json.gz")); !os.IsNotExist(err) {
		t.Errorf("relief écrit malgré l'échec (%v)", err)
	}
	if code := resume(context.Background(), []bilanAsset{ba}, e); code != 1 {
		t.Errorf("échecs de la cuisson = %d, attendu 1", code)
	}

	retireLePlanDuSidecar(t, dir)
	if ba := e.publie(context.Background(), c, renduEssai(pasEssai), himap.BilanCuisson{}); ba.Err != nil {
		t.Errorf("carte sans plan publié refusée : %v", ba.Err)
	}
	if ba := (&environnement{sortieDir: t.TempDir(), style: himap.StyleFondParDefaut, reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}).
		publie(context.Background(), c, renduEssai(pasEssai), himap.BilanCuisson{}); ba.Err != nil {
		t.Errorf("carte sans sidecar publié refusée : %v", ba.Err)
	}
}

// Une cuisson à une échelle différente de celle du fond publié met la carte en échec nommé : rien n'est
// écrit, ni fichier du relief ni sidecar.
func TestRefaitLeReliefEchelleDifferenteEstUnEchecQuiNEcritRien(t *testing.T) {
	dir := t.TempDir()
	avant := publieUnFondEssai(t, dir)
	e := &environnement{sortieDir: dir}
	var b bilanSeul
	err := e.refaitLeRelief(context.Background(), "carte", nil, &b, cuisEssai(renduEssai(2*pasEssai), carresDeTest([][2]float64{{0, 0}})))
	if !errors.Is(err, himap.ErrEchelleDifferente) {
		t.Fatalf("erreur = %v, attendu ErrEchelleDifferente", err)
	}
	if apres, _ := os.ReadFile(filepath.Join(dir, "carte.json")); !bytes.Equal(apres, avant) {
		t.Error("le sidecar a été réécrit")
	}
	for _, nom := range []string{"carte.relief.png", "carte.relief.json.gz"} {
		if _, err := os.Stat(filepath.Join(dir, nom)); !os.IsNotExist(err) {
			t.Errorf("%s écrit malgré l'échec (%v)", nom, err)
		}
	}
	e.noteResultat(context.Background(), &b, "carte", nil, err)
	if b.Echecs != 1 {
		t.Errorf("échecs = %d, attendu 1 : l'échelle différente n'est pas une carte laissée intacte", b.Echecs)
	}
}

// Sans sidecar ou sans fond publié, la carte est laissée intacte et la cuisson n'est même pas lancée.
func TestRefaitLeReliefSansSidecarLaisseLaCarteIntacte(t *testing.T) {
	e := &environnement{sortieDir: t.TempDir()}
	cuite := false
	var b bilanSeul
	err := e.refaitLeRelief(context.Background(), "carte", nil, &b, func() (carteCuite, error) {
		cuite = true
		return carteCuite{}, nil
	})
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("erreur = %v, attendu os.ErrNotExist", err)
	}
	if cuite {
		t.Error("la carte a été cuite alors qu'elle n'a pas de sidecar")
	}
}

// Une cuisson en échec n'est pas une carte laissée intacte, même quand son erreur vient d'un fichier absent
// de l'outillage (variante, maillage).
func TestRefaitLeReliefUneCuissonEnEchecNEstPasUneCarteIntacte(t *testing.T) {
	dir := t.TempDir()
	publieUnFondEssai(t, dir)
	e := &environnement{sortieDir: dir}
	var b bilanSeul
	err := e.refaitLeRelief(context.Background(), "carte", nil, &b, func() (carteCuite, error) {
		return carteCuite{}, os.ErrNotExist
	})
	if err == nil || errors.Is(err, os.ErrNotExist) {
		t.Fatalf("erreur = %v, attendu une erreur qui n'est pas os.ErrNotExist", err)
	}
}

// Une cuisson complète refait le relief dans le même geste que le fond : `publie` écrit le fond, le plan
// et le relief, et le calage du relief est celui du sidecar écrit.
func TestPublieRefaitLeReliefAvecLeFond(t *testing.T) {
	dir := t.TempDir()
	// Les données d'un relief d'une cuisson d'avant sont remplacées, pas gardées.
	if err := os.WriteFile(filepath.Join(dir, "carte.relief.json.gz"), []byte("perime"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := &environnement{sortieDir: dir, style: himap.StyleFondParDefaut, reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}
	c := cible{cle: "carte", noms: []string{"Carte"}, ancres: [][3]float64{{0.5, 0.5, 1.5}},
		maillage: carresDeTest([][2]float64{{0, 0}}), sourcePlan: sourcePlanNatif}
	// Le cadre publié est celui de la matière, rogné : la cellule du rendu est celle du calage.
	ba := e.publie(context.Background(), c, renduEssai(pasEssai), himap.BilanCuisson{})
	if ba.Err != nil {
		t.Fatalf("publie : %v", ba.Err)
	}
	s, err := mapfondsidecar.Load(filepath.Join(dir, "carte.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.ReliefImage != "carte.relief.png" || s.ReliefData != "carte.relief.json.gz" {
		t.Fatalf("champs du relief = %q, %q", s.ReliefImage, s.ReliefData)
	}
	img, donnees := lisLeRelief(t, dir, s)
	cal := s.Calibration
	if donnees.Calibration.Largeur != cal.WidthPx || donnees.Calibration.Hauteur != cal.HeightPx ||
		donnees.Calibration.OrigineX != cal.OriginX || donnees.Calibration.OrigineY != cal.OriginY ||
		img.Bounds().Dx() != cal.WidthPx || img.Bounds().Dy() != cal.HeightPx {
		t.Errorf("le relief n'est pas dans le calage du fond publié : relief %+v (image %v), fond %+v", donnees.Calibration, img.Bounds(), cal)
	}
}

// Un relief impossible à calculer retire ses deux champs du sidecar et ses fichiers : le fond est publié
// sans relief, jamais avec celui d'un autre fond.
func TestEcritSidecarSansReliefRetireLeReliefPerime(t *testing.T) {
	dir := t.TempDir()
	for _, nom := range []string{"carte.relief.png", "carte.relief.webp", "carte.relief.json.gz"} {
		if err := os.WriteFile(filepath.Join(dir, nom), []byte("perime"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e := &environnement{sortieDir: dir, style: himap.StyleFondParDefaut, reglages: &reglagesFond{Cartes: map[string]reglageCarte{}}}
	cal := himap.CalageFond{MetresParPixel: 0.1, Largeur: 3, Hauteur: 3}
	if err := e.ecritSidecar(context.Background(), cible{cle: "carte"}, himap.BilanCuisson{}, cal, annexesFond{}); err != nil {
		t.Fatal(err)
	}
	s, err := mapfondsidecar.Load(filepath.Join(dir, "carte.json"))
	if err != nil {
		t.Fatal(err)
	}
	if s.ReliefImage != "" || s.ReliefData != "" {
		t.Errorf("champs du relief = %q, %q, attendu vides", s.ReliefImage, s.ReliefData)
	}
	for _, nom := range []string{"carte.relief.png", "carte.relief.webp", "carte.relief.json.gz"} {
		if _, err := os.Stat(filepath.Join(dir, nom)); !os.IsNotExist(err) {
			t.Errorf("%s périmé non retiré (%v)", nom, err)
		}
	}
}
