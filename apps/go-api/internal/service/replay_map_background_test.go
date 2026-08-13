package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// mapNamesStub joue le rôle du registre : il rend les identités de carte d'un match
// (map_id + noms), ou échoue.
type mapNamesStub struct {
	mapID string
	names []string
	err   error
	// vu enregistre les match_id demandés — de quoi vérifier qu'on interroge bien la base
	// et qu'on ne devine pas la carte ailleurs.
	vu []string
}

func (m *mapNamesStub) MapKeysForMatch(_ context.Context, matchID string) (port.MatchMapKeys, error) {
	m.vu = append(m.vu, matchID)
	return port.MatchMapKeys{MapID: m.mapID, Names: m.names}, m.err
}

// fondDeCarte pose, sous une racine de dépôt neuve, le catalogue de bornes et les deux
// fichiers d'un fond de carte (image + calage). Rend la racine.
//
// Le catalogue est écrit avec la MÊME forme que la donnée de référence de production
// (schemaVersion + maps indexé par nom normalisé) : un test qui invente sa propre forme
// ne dirait rien du fichier réel.
func fondDeCarte(t *testing.T, titleSlug, cleCatalogue, module string, avecImage bool) string {
	t.Helper()
	root := t.TempDir()
	res := title.NewPathResolver(root)

	catalogue := `{"schemaVersion":1,"source":"test","maps":{"` + cleCatalogue + `":{` +
		`"module":"` + module + `","min":[-10,-10,-10],"max":[10,10,10],"axisWidths":[11,11,11]}}}`
	ecrire(t, res.MapQuantBoundsPath(titleSlug), catalogue)

	sidecar := `{"schemaVersion":1,"module":"` + module + `","image":"` + module + `.png",` +
		`"source":"test","generatedAt":"2026-08-10T20:25:25Z","style":"jeu",` +
		`"calibration":{"metersPerPixel":0.092,"originX":-57.3,"originY":78.87,` +
		`"widthPx":1633,"heightPx":1627,"convention":"x = originX + (px+0.5)*mpp"},` +
		`"stats":{"anchors":14,"anchorsInFrame":14,"anchorsWithGround":14}}`
	ecrire(t, res.MapBackgroundMetaPath(titleSlug, module), sidecar)

	if avecImage {
		ecrire(t, res.MapBackgroundPath(titleSlug, module), "\x89PNG\r\n\x1a\nfaux")
	}
	return root
}

func ecrire(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// fondSousCle ajoute, sous une racine existante, les deux fichiers d'un fond (image +
// calage) sous une cle arbitraire — celle d'une carte Forge est son map_id.
func fondSousCle(t *testing.T, root, titleSlug, cle string) {
	t.Helper()
	res := title.NewPathResolver(root)
	sidecar := `{"schemaVersion":1,"module":"` + cle + `","image":"` + cle + `.png",` +
		`"source":"test","generatedAt":"2026-08-13T10:00:00Z","style":"jeu",` +
		`"calibration":{"metersPerPixel":0.092,"originX":65.6,"originY":112.43,` +
		`"widthPx":1332,"heightPx":1287,"convention":"x = originX + (px+0.5)*mpp"},` +
		`"stats":{"anchors":4,"anchorsInFrame":4,"anchorsWithGround":4}}`
	ecrire(t, res.MapBackgroundMetaPath(titleSlug, cle), sidecar)
	ecrire(t, res.MapBackgroundPath(titleSlug, cle), "\x89PNG\r\n\x1a\nfaux-forge")
}

// TestMapBackground_ChaineComplete — le contrat de F1 : match -> carte -> module -> calage.
func TestMapBackground_ChaineComplete(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	repo := &mapNamesStub{names: []string{"Cliffhanger"}}
	svc := NewReplayService(title.DefaultSlug, root, repo)

	bg, err := svc.MapBackground(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapBackground: %v", err)
	}
	if bg.Module != "ridgeline" {
		t.Errorf("module = %q, attendu ridgeline", bg.Module)
	}
	if bg.Calibration.MetersPerPixel != 0.092 || bg.Calibration.WidthPx != 1633 {
		t.Errorf("calage non transmis : %+v", bg.Calibration)
	}
	if len(repo.vu) == 0 || repo.vu[0] != "m1" {
		t.Errorf("le match n'a pas été demandé au registre : %v", repo.vu)
	}

	blob, err := svc.MapBackgroundImage(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapBackgroundImage: %v", err)
	}
	if len(blob) == 0 || string(blob[:4]) != "\x89PNG" {
		t.Errorf("les octets servis ne sont pas ceux du fichier : %q", blob)
	}
}

// TestMapBackground_ForgeParMapID — LA CLE DES CARTES FORGE EST LE map_id DU MATCH.
//
// Le décor : « Vagabond » figure au catalogue de bornes (module de CANEVAS fo08_wetland,
// nécessaire à la déquantification) mais AUCUN fond ne vit sous cette clé — un canevas est
// partagé par des dizaines de cartes. Le fond vit sous le map_id, et il doit être servi
// même quand la base ne porte AUCUN nom (map_name NULL, cas mesuré sur le seul match
// Vagabond du registre).
func TestMapBackground_ForgeParMapID(t *testing.T) {
	const mapID = "105f5d84-8de1-4908-af3a-1c4f3bf9d642"
	root := t.TempDir()
	res := title.NewPathResolver(root)
	ecrire(t, res.MapQuantBoundsPath(title.DefaultSlug),
		`{"schemaVersion":1,"source":"test","maps":{"vagabond":{`+
			`"module":"fo08_wetland","min":[-10,-10,-10],"max":[10,10,10],"axisWidths":[11,11,11]}}}`)
	fondSousCle(t, root, title.DefaultSlug, mapID)

	for _, c := range []struct {
		nom  string
		stub *mapNamesStub
	}{
		{"map_id + nom", &mapNamesStub{mapID: mapID, names: []string{"Vagabond"}}},
		{"map_id sans nom (map_name NULL)", &mapNamesStub{mapID: mapID}},
	} {
		t.Run(c.nom, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, root, c.stub)
			bg, err := svc.MapBackground(context.Background(), "m1")
			if err != nil {
				t.Fatalf("MapBackground: %v", err)
			}
			if bg.Module != mapID {
				t.Errorf("cle = %q, attendu le map_id %s", bg.Module, mapID)
			}
			if _, err := svc.MapBackgroundImage(context.Background(), "m1"); err != nil {
				t.Errorf("MapBackgroundImage: %v", err)
			}
		})
	}
}

// TestMapBackground_ForgeJamaisViaCanevas — le repli module ne sert JAMAIS un fond Forge.
//
// Un match d'une carte Forge SANS fond publié résout son nom vers le module du canevas
// (catalogue de bornes) ; aucun fond ne doit vivre là, et la réponse est l'absence propre —
// pas le fond d'une AUTRE carte bâtie sur le même canevas.
func TestMapBackground_ForgeJamaisViaCanevas(t *testing.T) {
	root := t.TempDir()
	res := title.NewPathResolver(root)
	ecrire(t, res.MapQuantBoundsPath(title.DefaultSlug),
		`{"schemaVersion":1,"source":"test","maps":{"dynasty":{`+
			`"module":"fo08_wetland","min":[-10,-10,-10],"max":[10,10,10],"axisWidths":[11,11,11]}}}`)
	// Le fond de Vagabond existe sous SON map_id ; Dynasty (même canevas) n'en a pas.
	fondSousCle(t, root, title.DefaultSlug, "105f5d84-8de1-4908-af3a-1c4f3bf9d642")

	svc := NewReplayService(title.DefaultSlug, root,
		&mapNamesStub{mapID: "cfd90b63-62fd-441a-8015-8d7804b9c3c3", names: []string{"Dynasty"}})
	if _, err := svc.MapBackground(context.Background(), "m1"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("le canevas ne doit servir aucun fond Forge : err = %v", err)
	}
}

// TestMapBackground_NomNormalise — le nom affiché passe par la MÊME normalisation que le
// build (« Aquarius - Ranked » et « Aquarius » désignent une seule carte). Si ce test
// tombe, c'est qu'une seconde règle de nommage a été écrite quelque part.
func TestMapBackground_NomNormalise(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "aquarius", "ctf_aquarius", true)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Aquarius - Ranked"}})

	bg, err := svc.MapBackground(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapBackground: %v", err)
	}
	if bg.Module != "ctf_aquarius" {
		t.Errorf("module = %q, attendu ctf_aquarius", bg.Module)
	}
}

// TestMapBackground_SecondCandidat — le registre rend plusieurs noms car aucun n'est bon
// partout : le premier qui ne résout rien ne doit pas condamner le fond.
func TestMapBackground_SecondCandidat(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "catalyst", "catalyst", true)
	svc := NewReplayService(title.DefaultSlug, root,
		&mapNamesStub{names: []string{"Catalyseur", "Catalyst"}})

	bg, err := svc.MapBackground(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapBackground: %v", err)
	}
	if bg.Module != "catalyst" {
		t.Errorf("module = %q, attendu catalyst", bg.Module)
	}
}

// TestMapBackground_Absences — chaque maillon manquant rend la MÊME sentinelle, et jamais
// une erreur 500 : une carte sans fond est un cas NORMAL (21 cartes en ont).
func TestMapBackground_Absences(t *testing.T) {
	cas := []struct {
		nom  string
		root func(t *testing.T) string
		repo port.ReplayMapNameRepo
	}{
		{
			nom:  "pas de résolution de carte câblée",
			root: func(t *testing.T) string { return fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true) },
			repo: nil,
		},
		{
			nom:  "carte inconnue en base",
			root: func(t *testing.T) string { return fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true) },
			repo: &mapNamesStub{err: errors.New("pas de ligne")},
		},
		{
			nom:  "carte absente du catalogue de modules",
			root: func(t *testing.T) string { return fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true) },
			repo: &mapNamesStub{names: []string{"Interlock"}},
		},
		{
			nom:  "catalogue de bornes absent",
			root: func(t *testing.T) string { return t.TempDir() },
			repo: &mapNamesStub{names: []string{"Cliffhanger"}},
		},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, c.root(t), c.repo)
			if _, err := svc.MapBackground(context.Background(), "m1"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
				t.Errorf("MapBackground: err = %v, attendu ErrMapBackgroundNotAvailable", err)
			}
			if _, err := svc.MapBackgroundImage(context.Background(), "m1"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
				t.Errorf("MapBackgroundImage: err = %v, attendu ErrMapBackgroundNotAvailable", err)
			}
		})
	}
}

// TestMapBackgroundImage_SansCalage — UNE IMAGE SANS CALAGE NE SE SUPERPOSE À RIEN.
//
// Le témoin départage : le PNG est bien là, seul le sidecar manque, et l'image doit
// pourtant être refusée. Sans cette règle, le client poserait un fond au hasard sur la
// carte et rien à l'écran ne dirait qu'il est faux.
func TestMapBackgroundImage_SansCalage(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	res := title.NewPathResolver(root)
	if err := os.Remove(res.MapBackgroundMetaPath(title.DefaultSlug, "ridgeline")); err != nil {
		t.Fatalf("remove sidecar: %v", err)
	}
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Cliffhanger"}})

	if _, err := svc.MapBackgroundImage(context.Background(), "m1"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("image sans calage servie : err = %v", err)
	}
	// Contre-épreuve : avec son sidecar, la même image est servie.
	root2 := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	svc2 := NewReplayService(title.DefaultSlug, root2, &mapNamesStub{names: []string{"Cliffhanger"}})
	if _, err := svc2.MapBackgroundImage(context.Background(), "m1"); err != nil {
		t.Errorf("image avec calage : %v", err)
	}
}

// TestMapBackground_ImageAbsente — le calage existe, le PNG non : le calage reste servi
// (il est exact), l'image seule manque.
func TestMapBackground_ImageAbsente(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", false)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Cliffhanger"}})

	if _, err := svc.MapBackground(context.Background(), "m1"); err != nil {
		t.Errorf("le calage doit rester servi : %v", err)
	}
	if _, err := svc.MapBackgroundImage(context.Background(), "m1"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("image absente : err = %v, attendu ErrMapBackgroundNotAvailable", err)
	}
}

// TestMapBackground_IsoleParTitre — le fond d'un titre n'est jamais servi à un autre
// (isolation par chemin FS, ADR 0008).
func TestMapBackground_IsoleParTitre(t *testing.T) {
	root := fondDeCarte(t, title.DefaultSlug, "cliffhanger", "ridgeline", true)
	svc := NewReplayService("halo_5", root, &mapNamesStub{names: []string{"Cliffhanger"}})

	if _, err := svc.MapBackground(context.Background(), "m1"); !errors.Is(err, port.ErrMapBackgroundNotAvailable) {
		t.Errorf("le fond d'un titre ne doit pas être vu par un autre : err = %v", err)
	}
}

// TestMapBackground_DonneesReelles — L'ORACLE SUR LES ASSETS VERSIONNÉS.
//
// Les tests ci-dessus éprouvent la mécanique sur des fichiers fabriqués ; celui-ci éprouve
// la CHAÎNE sur les 21 fonds réellement livrés. Il aurait attrapé, seul, une clé de
// catalogue qui ne désigne aucun module et un sidecar dont la version de schéma a bougé.
//
// IL NE SE DÉCLARE PAS `SKIP` EN SILENCE quand la donnée manque : les fonds de carte et le
// catalogue de bornes sont VERSIONNÉS (git ls-files le confirme), donc leur absence est un
// échec, pas une dispense. Seule l'impossibilité de situer la racine du dépôt fait passer.
func TestMapBackground_DonneesReelles(t *testing.T) {
	root, err := title.FindRepoRoot()
	if err != nil {
		t.Skipf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(root)
	if _, statErr := os.Stat(res.MapQuantBoundsPath(title.DefaultSlug)); statErr != nil {
		t.Fatalf("catalogue de bornes versionné absent : %v", statErr)
	}

	// Cliffhanger -> ridgeline : le seul appariement dont on possède un artefact de rejeu,
	// donc le seul que le gate visuel peut confronter à l'écran.
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Cliffhanger"}})
	bg, err := svc.MapBackground(context.Background(), "000d5950")
	if err != nil {
		t.Fatalf("fond de Cliffhanger : %v", err)
	}
	if bg.Module != "ridgeline" {
		t.Errorf("module = %q, attendu ridgeline", bg.Module)
	}
	if bg.Calibration.MetersPerPixel <= 0 || bg.Calibration.WidthPx <= 0 || bg.Calibration.HeightPx <= 0 {
		t.Errorf("calage inexploitable : %+v", bg.Calibration)
	}
	blob, err := svc.MapBackgroundImage(context.Background(), "000d5950")
	if err != nil {
		t.Fatalf("image de Cliffhanger : %v", err)
	}
	if len(blob) < 8 || string(blob[1:4]) != "PNG" {
		t.Errorf("l'octet servi n'est pas un PNG (%d octets)", len(blob))
	}

	// ET LE CADRE CONTIENT LA ZONE JOUÉE. C'est le contrôle qui dit que les deux repères
	// sont bien le même : les bornes du rejeu de 000d5950 doivent tomber dans l'image.
	// Sans lui, un fond décalé d'un facteur d'échelle passerait pour posé.
	c := bg.Calibration
	minX := c.OriginX
	maxX := c.OriginX + float64(c.WidthPx)*c.MetersPerPixel
	maxY := c.OriginY
	minY := c.OriginY - float64(c.HeightPx)*c.MetersPerPixel
	// Bornes lues dans l'artefact de référence (cf. plan piste F, état de l'art §2).
	const jouXMin, jouXMax, jouYMin, jouYMax = -7.02, 43.79, -25.14, 30.35
	if jouXMin < minX || jouXMax > maxX || jouYMin < minY || jouYMax > maxY {
		t.Errorf("la zone jouée déborde du cadre de l'image : x[%.2f;%.2f] y[%.2f;%.2f]",
			minX, maxX, minY, maxY)
	}
}

// TestMapBackground_TousLesModulesDuCatalogue — chaque module cité par le catalogue de
// bornes qui possède un fond doit livrer un calage LISIBLE. Un sous-test par carte : un
// t.Fatal dans une boucle de balayage tuerait tout le balayage (piège du chantier cartes).
func TestMapBackground_TousLesModulesDuCatalogue(t *testing.T) {
	root, err := title.FindRepoRoot()
	if err != nil {
		t.Skipf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(root)
	cat, err := filmdec.LoadMapQuantCatalog(res.MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	if len(cat.Maps) == 0 {
		t.Fatal("catalogue de bornes vide — l'oracle ne testerait rien")
	}

	avecFond := 0
	for nom, entry := range cat.Maps {
		t.Run(nom, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{nom}})
			bg, err := svc.MapBackground(context.Background(), "m")
			if errors.Is(err, port.ErrMapBackgroundNotAvailable) {
				// Cas NORMAL et documenté : toutes les cartes n'ont pas de fond cuit.
				t.Skipf("pas de fond figé pour %s (module %s)", nom, entry.Module)
			}
			if err != nil {
				t.Fatalf("fond de %s : %v", nom, err)
			}
			if bg.Module != entry.Module {
				t.Errorf("module = %q, attendu %q", bg.Module, entry.Module)
			}
			if bg.Calibration.MetersPerPixel <= 0 || bg.Calibration.WidthPx <= 0 {
				t.Errorf("calage inexploitable pour %s : %+v", nom, bg.Calibration)
			}
			if _, err := svc.MapBackgroundImage(context.Background(), "m"); err != nil {
				t.Errorf("image de %s : %v", nom, err)
			}
			avecFond++
		})
	}
	if avecFond == 0 {
		t.Error("aucune carte du catalogue n'a de fond : la chaîne ne sert rien")
	}
}

// TestMapBackground_TousLesFondsMapID — L'ORACLE DU CATALOGUE, ÉTENDU AU map_id.
//
// Le pendant Forge de TestMapBackground_TousLesModulesDuCatalogue : chaque fond versionné
// keyé par un map_id (uuid) doit être servi par la chaîne map_id-d'abord, calage lisible et
// image comprise. Il ne se déclare pas SKIP quand rien ne matche : Vagabond et Corpo sont
// publiés sous cette clé depuis le lot fonds par map_id — zéro fond uuid, c'est la chaîne
// Forge débranchée.
func TestMapBackground_TousLesFondsMapID(t *testing.T) {
	root, err := title.FindRepoRoot()
	if err != nil {
		t.Skipf("racine du dépôt introuvable : %v", err)
	}
	res := title.NewPathResolver(root)
	entrees, err := os.ReadDir(res.MapBackgroundDir(title.DefaultSlug))
	if err != nil {
		t.Fatalf("dossier des fonds versionnés : %v", err)
	}
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	n := 0
	for _, e := range entrees {
		cle := strings.TrimSuffix(e.Name(), ".json")
		if e.IsDir() || cle == e.Name() || !uuidRe.MatchString(cle) {
			continue
		}
		n++
		t.Run(cle, func(t *testing.T) {
			svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: cle})
			bg, err := svc.MapBackground(context.Background(), "m")
			if err != nil {
				t.Fatalf("fond %s : %v", cle, err)
			}
			if bg.Module != cle {
				t.Errorf("cle du sidecar = %q, attendu %q", bg.Module, cle)
			}
			if bg.Calibration.MetersPerPixel <= 0 || bg.Calibration.WidthPx <= 0 || bg.Calibration.HeightPx <= 0 {
				t.Errorf("calage inexploitable : %+v", bg.Calibration)
			}
			if _, err := svc.MapBackgroundImage(context.Background(), "m"); err != nil {
				t.Errorf("image de %s : %v", cle, err)
			}
		})
	}
	if n == 0 {
		t.Error("aucun fond keyé par map_id — la chaîne Forge ne sert rien (Vagabond et Corpo doivent y être)")
	}
}
