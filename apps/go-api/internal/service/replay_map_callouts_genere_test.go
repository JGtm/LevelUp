package service

// Témoins de la résolution des zones par le CATALOGUE GÉNÉRÉ (cartes Forge rattrapées au fil de
// l'eau), par le map_id au catalogue des bornes, et par les identités déclarées des fonds.

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/port"
)

// mapIDRattrapee : une carte Forge absente du catalogue versionné (cas Lattice - Ranked).
const mapIDRattrapee = "1a6cfc2e-ec86-48e1-9464-1ce1bff6ed48"

// catalogueGenereFixture : le catalogue généré, à la forme qu'écrit le rattrapage.
func catalogueGenereFixture(mapID, en, fr string) string {
	return `{"schema_version":1,"title_slug":"halo_infinite","source":"test","maps":{},` +
		`"maps_by_id":{"` + mapID + `":{"module":"","provenance":"mvar","zones":[` +
		`{"volume_index":7,"name":"","en":"` + en + `","fr":"` + fr + `",` +
		`"x":1,"y":2,"z":3,"z_bottom":0,"z_top":6,"big":true,` +
		`"polygon":[[0,0],[6,0],[6,6],[0,6]]}]}}}`
}

// TestMapCallouts_CatalogueGenere — une carte rattrapée par le runtime est servie par son
// map_id, et une carte du catalogue versionné reste servie par lui.
func TestMapCallouts_CatalogueGenere(t *testing.T) {
	root := calloutsFixture(t)
	res := title.NewPathResolver(root)
	ecrire(t, res.MapCalloutsOverlayPath(title.DefaultSlug), catalogueGenereFixture(mapIDRattrapee, "Bridge", "Pont"))

	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: mapIDRattrapee, names: []string{"Lattice - Ranked"}})
	entry, err := svc.MapCallouts(context.Background(), "m1")
	if err != nil {
		t.Fatalf("carte rattrapée : %v", err)
	}
	if len(entry.Zones) != 1 || entry.Zones[0].FR != "Pont" || entry.Provenance != "mvar" {
		t.Fatalf("entrée servie : %+v", entry)
	}
}

// TestMapCallouts_VersionnePrimeSurGenere — une carte relue en revue n'est jamais remplacée par
// une carte rattrapée automatiquement.
func TestMapCallouts_VersionnePrimeSurGenere(t *testing.T) {
	root := calloutsFixture(t)
	res := title.NewPathResolver(root)
	ecrire(t, res.MapCalloutsOverlayPath(title.DefaultSlug), catalogueGenereFixture(mapIDForgeTest, "AUTRE", "AUTRE"))

	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: mapIDForgeTest})
	entry, err := svc.MapCallouts(context.Background(), "m1")
	if err != nil {
		t.Fatalf("MapCallouts : %v", err)
	}
	if entry.Zones[0].FR != "Grotte" {
		t.Errorf("zone servie %q : le catalogue généré a primé sur le versionné", entry.Zones[0].FR)
	}
}

// TestMapCallouts_CatalogueGenereSuitLeFilDeLEau — une carte ajoutée PENDANT que le serveur
// tourne s'affiche à la requête suivante, sans redémarrage.
func TestMapCallouts_CatalogueGenereSuitLeFilDeLEau(t *testing.T) {
	root := calloutsFixture(t)
	res := title.NewPathResolver(root)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: mapIDRattrapee})
	if _, err := svc.MapCallouts(context.Background(), "m1"); !errors.Is(err, port.ErrMapCalloutsNotAvailable) {
		t.Fatalf("avant rattrapage : err = %v, attendu une absence", err)
	}
	ecrire(t, res.MapCalloutsOverlayPath(title.DefaultSlug), catalogueGenereFixture(mapIDRattrapee, "Bridge", "Pont"))
	entry, err := svc.MapCallouts(context.Background(), "m1")
	if err != nil || entry.Zones[0].EN != "Bridge" {
		t.Fatalf("après rattrapage : %+v / %v", entry, err)
	}
}

// TestMapCallouts_CatalogueGenereIllisible — un catalogue généré abîmé ne prive pas les cartes
// versionnées de leurs zones, et ne fait servir aucune zone devinée.
func TestMapCallouts_CatalogueGenereIllisible(t *testing.T) {
	root := calloutsFixture(t)
	ecrire(t, title.NewPathResolver(root).MapCalloutsOverlayPath(title.DefaultSlug), "{abîmé")
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: mapIDForgeTest})
	if _, err := svc.MapCallouts(context.Background(), "m1"); err != nil {
		t.Errorf("carte versionnée : %v", err)
	}
	svc = NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: mapIDRattrapee})
	if _, err := svc.MapCallouts(context.Background(), "m1"); !errors.Is(err, port.ErrMapCalloutsNotAvailable) {
		t.Errorf("carte du catalogue illisible : err = %v, attendu une absence", err)
	}
}

// TestMapCallouts_MapIDAuCatalogueDesBornes — une carte que le catalogue des bornes connaît
// PAR SON MAP_ID (aucun nom exploitable) trouve son module.
func TestMapCallouts_MapIDAuCatalogueDesBornes(t *testing.T) {
	root := calloutsFixture(t)
	const idNative = "77777777-0000-0000-0000-000000000007"
	ecrire(t, title.NewPathResolver(root).MapQuantBoundsPath(title.DefaultSlug),
		`{"schemaVersion":1,"source":"test","maps":{`+
			`"`+idNative+`":{"module":"ridgeline","min":[-10,-10,-10],"max":[10,10,10],"axisWidths":[11,11,11]}}}`)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: idNative, names: []string{"Nom inconnu"}})
	entry, err := svc.MapCallouts(context.Background(), "m1")
	if err != nil || entry.Module != "ridgeline" {
		t.Fatalf("map_id au catalogue des bornes : %+v / %v", entry, err)
	}
}

// TestMapCallouts_IdentiteDeclareeParUnFond — un nom de VARIANTE qu'aucune règle de suffixe ne
// rabote (« … Sentry Defense ») trouve le module de sa carte par l'identité que le fond publié
// déclare pour elle.
func TestMapCallouts_IdentiteDeclareeParUnFond(t *testing.T) {
	root := calloutsFixture(t)
	sidecar := `{"schemaVersion":1,"module":"ridgeline","mapNames":["Ridgeline Sentry Defense"],` +
		`"image":"ridgeline.png","source":"test","generatedAt":"2026-08-10T20:25:25Z","style":"jeu",` +
		`"calibration":{"metersPerPixel":0.092,"originX":-57.3,"originY":78.87,` +
		`"widthPx":1633,"heightPx":1627,"convention":"x = originX + (px+0.5)*mpp"},` +
		`"stats":{"anchors":14,"anchorsInFrame":14,"anchorsWithGround":14}}`
	ecrire(t, title.NewPathResolver(root).MapBackgroundMetaPath(title.DefaultSlug, "ridgeline"), sidecar)
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{names: []string{"Ridgeline Sentry Defense"}})
	entry, err := svc.MapCallouts(context.Background(), "m1")
	if err != nil || entry.Module != "ridgeline" {
		t.Fatalf("identité déclarée : %+v / %v", entry, err)
	}
}

// TestCatalogueGenere_RelitQuandLeFichierChange — le cache à empreinte : une lecture tant que le
// fichier ne bouge pas, une nouvelle dès qu'il change ; un fichier absent rend fs.ErrNotExist.
func TestCatalogueGenere_RelitQuandLeFichierChange(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "generated", "map_callouts.json")
	lectures := 0
	avant := chargerCallouts
	chargerCallouts = func(p string) (*replay.MapCalloutsCatalog, error) {
		lectures++
		return avant(p)
	}
	t.Cleanup(func() { chargerCallouts = avant })

	if _, err := catalogueGenere(chemin); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("absent : err = %v, attendu fs.ErrNotExist", err)
	}
	ecrire(t, chemin, catalogueGenereFixture("aaa", "A", "A"))
	for i := 0; i < 2; i++ {
		if _, err := catalogueGenere(chemin); err != nil {
			t.Fatal(err)
		}
	}
	if lectures != 1 {
		t.Fatalf("fichier inchangé : %d lectures, attendu 1", lectures)
	}
	ecrire(t, chemin, catalogueGenereFixture("bbbbbb", "B", "B"))
	futur := time.Now().Add(time.Minute)
	if err := os.Chtimes(chemin, futur, futur); err != nil {
		t.Fatal(err)
	}
	cat, err := catalogueGenere(chemin)
	if err != nil || lectures != 2 {
		t.Fatalf("fichier changé : %d lectures (err %v), attendu 2", lectures, err)
	}
	if _, err := cat.LookupByID("bbbbbb"); err != nil {
		t.Errorf("catalogue relu périmé : %v", err)
	}
}
