package service

// replay_map_background_relief_test.go — LE RELIEF TRAVERSE LE SERVICE.
//
// Les noms de ses deux fichiers vivent dans le sidecar (`reliefImage`, `reliefData`) ; ils se servent
// sous la même clé que le fond, avec les mêmes gardes sur le nom de fichier. Les données ne sortent
// que si le calage qu'elles recopient est celui du sidecar : un relief périmé est un relief absent.

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// gzipTest compresse `contenu` en gzip, comme la cuisson range les données du relief.
func gzipTest(t *testing.T, contenu string) string {
	t.Helper()
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	if _, err := zw.Write([]byte(contenu)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// ecrireBrut écrit `contenu` octet pour octet. `ecrire` range un chemin en `.json.gz` comme un artefact
// de rejeu, en le compressant : les données du relief portent la même extension et sont déjà du gzip.
func ecrireBrut(t *testing.T, chemin, contenu string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(chemin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
		t.Fatal(err)
	}
}

// donneesReliefAuCalage est un fichier de données du relief projeté dans le calage du fond de
// `fondDeCarte` (0,092 m/px, origine -57,3 / 78,87, 1633 x 1627) ; `calage` le remplace pour fabriquer
// un relief périmé.
func donneesReliefAuCalage(calage string) string {
	if calage == "" {
		calage = `{"metersPerPixel":0.092,"originX":-57.3,"originY":78.87,"widthPx":1633,"heightPx":1627}`
	}
	return `{"schemaVersion":1,"calibration":` + calage + `,"floors":{"levels":[0,2.2]}}`
}

// fondAvecRelief pose un fond de Cliffhanger dont le sidecar porte `reliefImage` et `reliefData`.
// L'image n'est écrite que si `ecritImage`, les données que si `contenuDonnees` n'est pas vide ; il
// est écrit tel quel (compressé par l'appelant, ou non pour fabriquer un fichier illisible).
func fondAvecRelief(t *testing.T, image, donnees string, ecritImage bool, contenuDonnees string) (string, *replayService) {
	t.Helper()
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	res := title.NewPathResolver(root)
	chemin := res.MapBackgroundMetaPath(title.DefaultSlug, "ridgeline")
	blob, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatal(err)
	}
	sidecar := strings.Replace(string(blob), `"module":"ridgeline",`,
		`"module":"ridgeline","reliefImage":"`+image+`","reliefData":"`+donnees+`",`, 1)
	ecrire(t, chemin, sidecar)
	if ecritImage {
		ecrire(t, res.MapBackgroundImageFilePath(title.DefaultSlug, image), "RIFFrelief")
	}
	if contenuDonnees != "" {
		ecrireBrut(t, res.MapBackgroundImageFilePath(title.DefaultSlug, donnees), contenuDonnees)
	}
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	return root, NewReplayService(title.DefaultSlug, root, repo, nil).(*replayService)
}

func TestMapBackgroundRelief_ServiParCarteEtParMatch(t *testing.T) {
	donnees := donneesReliefAuCalage("")
	_, svc := fondAvecRelief(t, "ridgeline.relief.webp", "ridgeline.relief.json.gz", true, gzipTest(t, donnees))
	ctx := context.Background()

	blob, mime, err := svc.MapBackgroundReliefImageForMap(ctx, "asset-cliffhanger")
	if err != nil || string(blob) != "RIFFrelief" || mime != mimeImageWebP {
		t.Fatalf("image par carte : %q %q %v", blob, mime, err)
	}
	blob, mime, err = svc.MapBackgroundReliefImage(ctx, "match-1")
	if err != nil || string(blob) != "RIFFrelief" || mime != mimeImageWebP {
		t.Fatalf("image par match : %q %q %v", blob, mime, err)
	}
	// Les données sortent telles quelles, octet pour octet.
	for nom, lecture := range map[string]func() ([]byte, error){
		"par carte": func() ([]byte, error) { return svc.MapBackgroundReliefDataForMap(ctx, "asset-cliffhanger") },
		"par match": func() ([]byte, error) { return svc.MapBackgroundReliefData(ctx, "match-1") },
	} {
		if got, err := lecture(); err != nil || string(got) != donnees {
			t.Errorf("données %s : %q %v, attendu le fichier tel quel", nom, got, err)
		}
	}
	if fond, _, err := svc.MapBackgroundImageForMap(ctx, "asset-cliffhanger"); err != nil || string(fond) == "RIFFrelief" {
		t.Errorf("l'image du fond ne doit pas etre le relief : %q %v", fond, err)
	}
	bg, err := svc.MapBackgroundForMap(ctx, "asset-cliffhanger")
	if err != nil || bg.ReliefImage != "ridgeline.relief.webp" || bg.ReliefData != "ridgeline.relief.json.gz" {
		t.Errorf("noms du relief servis = %+v, %v", bg, err)
	}
}

func TestMapBackgroundRelief_AbsenceEstUnFondAbsent(t *testing.T) {
	ctx := context.Background()
	absent := func(nom string, err error) {
		t.Helper()
		if !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("%s : %v, attendu ErrMapBackgroundNotAvailable", nom, err)
		}
	}
	lire := func(nom string, svc port.ReplayService, carte string) {
		t.Helper()
		_, _, err := svc.MapBackgroundReliefImageForMap(ctx, carte)
		absent(nom+" (image)", err)
		_, err = svc.MapBackgroundReliefDataForMap(ctx, carte)
		absent(nom+" (données)", err)
	}

	// Sidecar sans reliefImage ni reliefData.
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	repo := &mapNamesStub{mapID: "asset-cliffhanger", names: []string{"Cliffhanger"}}
	lire("sans relief", NewReplayService(title.DefaultSlug, root, repo, nil), "asset-cliffhanger")

	// Noms déclarés, fichiers absents.
	_, svc := fondAvecRelief(t, "ridgeline.relief.webp", "ridgeline.relief.json.gz", false, "")
	lire("fichiers absents", svc, "asset-cliffhanger")

	// Carte inconnue.
	lire("carte inconnue", NewReplayService(title.DefaultSlug, root, &mapNamesStub{}, nil), "inconnue")
}

// Le nom de l'image du relief est un contenu de fichier : même garde que l'image du fond (nom simple,
// extension en liste blanche). Un fichier est écrit à chaque endroit où une lecture sans garde irait
// le chercher : sans la garde, la lecture le trouverait et le test rougirait.
// Mutation jouée : retirer le refus de imageFileNamed.
func TestMapBackgroundRelief_ImageRefuseUnNomDangereux(t *testing.T) {
	for _, nom := range []string{"../ridgeline.webp", "ridgeline.relief.exe", "sous/dossier.webp"} {
		root, svc := fondAvecRelief(t, strings.ReplaceAll(nom, `\`, ""), "ridgeline.relief.json.gz", false, "")
		res := title.NewPathResolver(root)
		ecrire(t, res.MapBackgroundImageFilePath(title.DefaultSlug, nom), "RIFFhostile")
		ecrire(t, res.MapBackgroundImageFilePath(title.DefaultSlug, filepath.Base(nom)), "RIFFhostile")
		if _, _, err := svc.MapBackgroundReliefImageForMap(context.Background(), "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("%q accepte : %v", nom, err)
		}
	}
}

// Le nom du fichier des données est lui aussi un contenu de fichier : nom simple qui finit par
// `.relief.json`. Un fichier de calage juste est écrit sous chaque nom : sans la garde, la lecture le
// servirait.
// Mutation jouée : retirer le refus de reliefDataFile.
func TestMapBackgroundRelief_DonneesRefuseUnNomDangereux(t *testing.T) {
	for _, nom := range []string{"../ridgeline.relief.json.gz", "sous/dossier.relief.json.gz",
		"ridgeline.relief.gz", "ridgeline.relief.json", ".relief.json.gz"} {
		root, svc := fondAvecRelief(t, "ridgeline.relief.webp", strings.ReplaceAll(nom, `\`, ""), false, "")
		res := title.NewPathResolver(root)
		ecrireBrut(t, res.MapBackgroundImageFilePath(title.DefaultSlug, nom), gzipTest(t, donneesReliefAuCalage("")))
		ecrireBrut(t, res.MapBackgroundImageFilePath(title.DefaultSlug, filepath.Base(nom)), gzipTest(t, donneesReliefAuCalage("")))
		if _, err := svc.MapBackgroundReliefDataForMap(context.Background(), "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("%q accepte : %v", nom, err)
		}
	}
}

// Un relief dont le calage recopié n'est plus celui du sidecar est périmé : réponse comme pour un
// relief absent, un champ du calage à la fois. L'image, elle, reste servie : seul le contrôle des
// données porte le calage.
// Mutation jouée : retirer l'un des cinq termes de calageDuRelief.estCelui.
func TestMapBackgroundRelief_CalageDifferentEstUnReliefPerime(t *testing.T) {
	calages := map[string]string{
		"échelle":     `{"metersPerPixel":0.093,"originX":-57.3,"originY":78.87,"widthPx":1633,"heightPx":1627}`,
		"origine x":   `{"metersPerPixel":0.092,"originX":-57.2,"originY":78.87,"widthPx":1633,"heightPx":1627}`,
		"origine y":   `{"metersPerPixel":0.092,"originX":-57.3,"originY":78.88,"widthPx":1633,"heightPx":1627}`,
		"largeur":     `{"metersPerPixel":0.092,"originX":-57.3,"originY":78.87,"widthPx":1605,"heightPx":1627}`,
		"hauteur":     `{"metersPerPixel":0.092,"originX":-57.3,"originY":78.87,"widthPx":1633,"heightPx":1771}`,
		"sans calage": `{}`,
	}
	for nom, calage := range calages {
		_, svc := fondAvecRelief(t, "ridgeline.relief.webp", "ridgeline.relief.json.gz", true, gzipTest(t, donneesReliefAuCalage(calage)))
		if _, err := svc.MapBackgroundReliefDataForMap(context.Background(), "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("calage différent (%s) servi : %v", nom, err)
		}
		if _, err := svc.MapBackgroundReliefData(context.Background(), "match-1"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("calage différent (%s) servi par match : %v", nom, err)
		}
		if _, _, err := svc.MapBackgroundReliefImageForMap(context.Background(), "asset-cliffhanger"); err != nil {
			t.Errorf("calage différent (%s) : l'image n'est pas soumise au contrôle : %v", nom, err)
		}
	}
}

// Des données illisibles sont tenues pour un relief absent, pas pour une panne : du JSON non compressé
// (le fichier doit être du gzip), du gzip qui ne contient pas du JSON, un schéma d'une autre version,
// un JSON valide qui dépasse la borne de taille une fois décompressé (des espaces en fin de document).
// Mutation jouée : retirer le refus de taille de decompresseDonneesRelief.
func TestMapBackgroundRelief_DonneesIllisiblesSontUnReliefAbsent(t *testing.T) {
	autreVersion := strings.Replace(donneesReliefAuCalage(""), `"schemaVersion":1`, `"schemaVersion":2`, 1)
	sansVersion := strings.Replace(donneesReliefAuCalage(""), `"schemaVersion":1,`, "", 1)
	cas := map[string]string{
		"json non compressé":         donneesReliefAuCalage(""),
		"gzip sans json":             gzipTest(t, "pas du json"),
		"schéma d'une autre version": gzipTest(t, autreVersion),
		"schéma sans version":        gzipTest(t, sansVersion),
		"au-delà de la borne":        gzipTest(t, donneesReliefAuCalage("")+strings.Repeat(" ", tailleMaxDonneesRelief)),
	}
	for nom, contenu := range cas {
		_, svc := fondAvecRelief(t, "ridgeline.relief.webp", "ridgeline.relief.json.gz", true, contenu)
		if _, err := svc.MapBackgroundReliefDataForMap(context.Background(), "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("%s : %v, attendu ErrMapBackgroundNotAvailable", nom, err)
		}
	}
}

// Une clé de fond hostile (map_id) est refusée avant tout chemin, relief compris : le sidecar du
// fond nominal porte un relief complet et la carte se retrouve par son nom : sans la garde, le repli
// par nom servirait le relief, un refus ne peut donc pas venir d'une absence.
func TestMapBackgroundRelief_RefuseUneCleHostile(t *testing.T) {
	root, _ := fondAvecRelief(t, "ridgeline.relief.webp", "ridgeline.relief.json.gz", true, gzipTest(t, donneesReliefAuCalage("")))
	for _, hostile := range clesHostiles {
		repo := &mapNamesStub{mapID: hostile, names: []string{"Cliffhanger"}}
		svc := NewReplayService(title.DefaultSlug, root, repo, nil)
		if _, _, err := svc.MapBackgroundReliefImageForMap(context.Background(), hostile); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("cle %q (image du relief) : %v", hostile, err)
		}
		if _, err := svc.MapBackgroundReliefDataForMap(context.Background(), hostile); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
			t.Errorf("cle %q (données du relief) : %v", hostile, err)
		}
	}
}

// Le sidecar lui-même (`ridgeline.json`) porte le calage du fond : un `reliefData` qui le désigne
// passerait le contrôle du calage, et le sidecar serait servi comme des données de relief. Seul le
// suffixe `.relief.json.gz` l'écarte.
func TestMapBackgroundRelief_DonneesRefuseLeSidecarLuiMeme(t *testing.T) {
	_, svc := fondAvecRelief(t, "ridgeline.relief.webp", "ridgeline.json", false, "")
	if _, err := svc.MapBackgroundReliefDataForMap(context.Background(), "asset-cliffhanger"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("le sidecar est servi comme données du relief : %v", err)
	}
}
