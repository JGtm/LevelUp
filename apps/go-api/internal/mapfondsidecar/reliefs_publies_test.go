package mapfondsidecar

// reliefs_publies_test.go — LES RELIEFS PUBLIES.
//
// Un relief mort est muet par construction : le service répond 404 et le web retombe sur le Plan sans
// rien dire. Ce test lit les fonds PUBLIES (donnée versionnée) et refuse un relief dont un fichier manque,
// dont les données ne se décompressent pas, dont le calage n'est plus celui du sidecar ou dont l'image n'a
// pas les dimensions du calage ; et un relief « sans maillage » sur une carte qui a un plan (cuisson
// faite sans le blob du maillage), ou un relief avec maillage sur une carte qui n'en a pas.
//
// Il ne demande PAS que chaque fond ait un relief : le parc se cuit par lots.

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"image"
	"image/png" // lecteur d'une image de relief encore en PNG avant sa conversion, et écrivain des essais
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/testutil"

	_ "golang.org/x/image/webp" // lecteur des images publiées
)

// versionSchemaReliefPubliee est la seule version des données du relief que le service sert.
const versionSchemaReliefPubliee = 1

// codeSansMaillage est la dégradation d'un relief calculé sans maillage de navigation.
const codeSansMaillage = "sans-maillage"

// donneesReliefPubliees est ce que le test lit des données du relief.
type donneesReliefPubliees struct {
	SchemaVersion int `json:"schemaVersion"`
	Calibration   struct {
		MetersPerPixel float64 `json:"metersPerPixel"`
		OriginX        float64 `json:"originX"`
		OriginY        float64 `json:"originY"`
		WidthPx        int     `json:"widthPx"`
		HeightPx       int     `json:"heightPx"`
	} `json:"calibration"`
	Degradations []string `json:"degradations"`
}

// lisDonneesRelief décompresse et décode le fichier des données d'un relief.
func lisDonneesRelief(chemin string) (donneesReliefPubliees, error) {
	var d donneesReliefPubliees
	compresse, err := os.ReadFile(chemin)
	if err != nil {
		return d, err
	}
	zr, err := gzip.NewReader(bytes.NewReader(compresse))
	if err != nil {
		return d, err
	}
	defer zr.Close()
	brut, err := io.ReadAll(zr)
	if err != nil {
		return d, err
	}
	return d, json.Unmarshal(brut, &d)
}

// dimensionsImage lit les dimensions d'une image (WebP publié, ou PNG avant sa conversion).
func dimensionsImage(chemin string) (int, int, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, err
	}
	return cfg.Width, cfg.Height, nil
}

// controleUnRelief rend les manquements du relief du sidecar `s` de la clé `cle`, et dit s'il est marqué
// « sans maillage ».
func controleUnRelief(dir, cle string, s *MapBackground) (manquements []string, sansMaillage bool) {
	if s.ReliefImage != NomImageRelief(cle, filepath.Ext(s.ReliefImage)) || !EstDonneesRelief(s.ReliefData) ||
		s.ReliefData != NomDonneesRelief(cle) {
		return []string{"noms hors convention : " + s.ReliefImage + ", " + s.ReliefData}, false
	}
	d, err := lisDonneesRelief(filepath.Join(dir, s.ReliefData))
	if err != nil {
		manquements = append(manquements, "données du relief illisibles : "+err.Error())
	} else {
		cal := s.Calibration
		if d.SchemaVersion != versionSchemaReliefPubliee {
			manquements = append(manquements, "schemaVersion "+strconv.Itoa(d.SchemaVersion))
		}
		if d.Calibration.MetersPerPixel != cal.MetersPerPixel || d.Calibration.OriginX != cal.OriginX ||
			d.Calibration.OriginY != cal.OriginY || d.Calibration.WidthPx != cal.WidthPx || d.Calibration.HeightPx != cal.HeightPx {
			manquements = append(manquements, "calage du relief différent de celui du sidecar")
		}
		for _, code := range d.Degradations {
			sansMaillage = sansMaillage || code == codeSansMaillage
		}
	}
	w, h, err := dimensionsImage(filepath.Join(dir, s.ReliefImage))
	switch {
	case err != nil:
		manquements = append(manquements, "image du relief illisible : "+err.Error())
	case w != s.Calibration.WidthPx || h != s.Calibration.HeightPx:
		manquements = append(manquements, "image du relief de "+strconv.Itoa(w)+"x"+strconv.Itoa(h)+", calage de "+
			strconv.Itoa(s.Calibration.WidthPx)+"x"+strconv.Itoa(s.Calibration.HeightPx))
	}
	return manquements, sansMaillage
}

func TestChaqueReliefPublieEstServable(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot introuvable : %v", err)
	}
	dir := filepath.Join(racine, "data", "titles", "halo_infinite", "reference", "map_backgrounds")
	sidecars, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(sidecars) == 0 {
		t.Skipf("aucun fond publie (%v)", err)
	}
	var sansPlanAvecRelief, marquesSansMaillage []string
	avecRelief := 0
	for _, chemin := range sidecars {
		cle := strings.TrimSuffix(filepath.Base(chemin), ".json")
		s, err := Load(chemin)
		if err != nil {
			t.Errorf("%s : %v", cle, err)
			continue
		}
		if (s.ReliefImage == "") != (s.ReliefData == "") {
			t.Errorf("%s : image du relief %q mais données %q — l'un ne va pas sans l'autre", cle, s.ReliefImage, s.ReliefData)
			continue
		}
		if s.ReliefImage == "" {
			continue
		}
		avecRelief++
		manquements, sansMaillage := controleUnRelief(dir, cle, s)
		for _, m := range manquements {
			t.Errorf("%s : %s", cle, m)
		}
		if s.Plan == nil {
			sansPlanAvecRelief = append(sansPlanAvecRelief, cle)
		}
		if sansMaillage {
			marquesSansMaillage = append(marquesSansMaillage, cle)
		}
	}
	sort.Strings(sansPlanAvecRelief)
	sort.Strings(marquesSansMaillage)
	if strings.Join(sansPlanAvecRelief, ",") != strings.Join(marquesSansMaillage, ",") {
		t.Errorf("reliefs marqués %q : %v ; fonds à relief sans plan des murs : %v — un relief « sans maillage » sur une carte qui a un plan "+
			"(maillage introuvable à la cuisson) ou un relief à maillage sur une carte qui n'en a pas", codeSansMaillage, marquesSansMaillage, sansPlanAvecRelief)
	}
	t.Logf("%d fonds sur %d portent un relief ; %d sans maillage", avecRelief, len(sidecars), len(marquesSansMaillage))
}

// ecritUnReliefEssai écrit dans `dir` les deux fichiers du relief de la clé "carte" : données au calage
// `cal` (JSON compressé) et image PNG de `largeur` x `hauteur` ; rend le sidecar qui les nomme.
func ecritUnReliefEssai(t *testing.T, dir string, version int, cal string, largeur, hauteur int) *MapBackground {
	t.Helper()
	var zip bytes.Buffer
	zw := gzip.NewWriter(&zip)
	if _, err := zw.Write([]byte(`{"schemaVersion":` + strconv.Itoa(version) + `,"calibration":` + cal + `,"degradations":[]}`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "carte.relief.json.gz"), zip.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "carte.relief.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, largeur, hauteur))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	s := &MapBackground{ReliefImage: "carte.relief.png", ReliefData: "carte.relief.json.gz"}
	s.Calibration.MetersPerPixel, s.Calibration.OriginX, s.Calibration.OriginY = 0.1, -1, 12
	s.Calibration.WidthPx, s.Calibration.HeightPx = 30, 20
	return s
}

// Le contrôle voit chacun des manquements qu'il annonce : sans lui, le test des fonds publiés passerait
// sur un dossier qui ne le mérite pas.
func TestControleUnReliefVoitLesManquements(t *testing.T) {
	const calageJuste = `{"metersPerPixel":0.1,"originX":-1,"originY":12,"widthPx":30,"heightPx":20}`
	cas := map[string]struct {
		version   int
		calage    string
		w, h      int
		supprimer string
		attendu   bool
	}{
		"relief juste":             {1, calageJuste, 30, 20, "", false},
		"autre version":            {2, calageJuste, 30, 20, "", true},
		"calage différent":         {1, `{"metersPerPixel":0.1,"originX":-1,"originY":12,"widthPx":30,"heightPx":21}`, 30, 20, "", true},
		"image d'une autre taille": {1, calageJuste, 30, 21, "", true},
		"image absente":            {1, calageJuste, 30, 20, "carte.relief.png", true},
		"données absentes":         {1, calageJuste, 30, 20, "carte.relief.json.gz", true},
	}
	for nom, c := range cas {
		dir := t.TempDir()
		s := ecritUnReliefEssai(t, dir, c.version, c.calage, c.w, c.h)
		if c.supprimer != "" {
			if err := os.Remove(filepath.Join(dir, c.supprimer)); err != nil {
				t.Fatal(err)
			}
		}
		manquements, _ := controleUnRelief(dir, "carte", s)
		if (len(manquements) > 0) != c.attendu {
			t.Errorf("%s : manquements = %v, attendu un manquement : %v", nom, manquements, c.attendu)
		}
	}
}
