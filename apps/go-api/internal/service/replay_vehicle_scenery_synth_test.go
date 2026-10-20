package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// replay_vehicle_scenery_synth_test.go — LA MATIERE DANS LE CADRE (revue RR-M7-05 du 2026-09-24).
// Au parc, aucun decor n est decide par un vide DANS l image : les douze de Starboard tombent hors
// du cadre. Ces tests posent donc un fond SYNTHETIQUE, au format publie (PNG RGBA + sidecar), sur
// lequel les deux cas que le parc n exerce pas deviennent des faits : un vide large dans le cadre
// masque, un trou de reconstruction plus etroit que la fermeture (2 x 4 m) ne masque pas. Remplacer
// la fermeture par le masque brut fait rougir le second.

// cleFondSynthetique : la cle du fond de test, servie comme un map_id Forge.
const cleFondSynthetique = "m7-fond-synthetique"

// fondSynthetique : 40 m x 40 m a 0,5 m par pixel, origine (0 ; 40). Matiere partout, sauf un vide
// de 12 m x 12 m au coin nord-ouest (x < 12, y > 28) et une saignee de 2 m (20 <= x < 22) entre
// y = 0 et y = 20.
func fondSynthetique(t *testing.T, root string) {
	t.Helper()
	const n, mpp = 80, 0.5
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for py := 0; py < n; py++ {
		for px := 0; px < n; px++ {
			x, y := (float64(px)+0.5)*mpp, 40-(float64(py)+0.5)*mpp
			vide := (x < 12 && y > 28) || (x >= 20 && x < 22 && y < 20)
			if !vide {
				img.SetNRGBA(px, py, color.NRGBA{R: 90, G: 90, B: 90, A: 255})
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	res := title.NewPathResolver(root)
	ecrire(t, res.MapBackgroundImageFilePath(title.DefaultSlug, cleFondSynthetique+".png"), buf.String())
	ecrire(t, res.MapBackgroundMetaPath(title.DefaultSlug, cleFondSynthetique),
		`{"schemaVersion":1,"module":"`+cleFondSynthetique+`","mapNames":["M7 synthetique"],`+
			`"image":"`+cleFondSynthetique+`.png","source":"test","generatedAt":"2026-09-24T00:00:00Z",`+
			`"style":"encre","calibration":{"metersPerPixel":0.5,"originX":0,"originY":40,`+
			`"widthPx":80,"heightPx":80,"convention":"x = originX + (px+0.5)*mpp"}}`)
}

func decorSynthetique(t *testing.T, root string, vies ...replay.VehicleTrack) *replay.VehicleScenery {
	t.Helper()
	svc := NewReplayService(title.DefaultSlug, root, &mapNamesStub{mapID: cleFondSynthetique}).(*replayService)
	doc := docJoue(0, vies...)
	svc.resolveVehicleScenery(context.Background(), doc, "m7", svc.matchMapKeys(context.Background(), "m7"))
	return doc.VehicleScenery
}

func TestVehicleScenery_MatiereDansLeCadre(t *testing.T) {
	root := t.TempDir()
	fondSynthetique(t, root)
	v := decorSynthetique(t, root,
		viePosee(1, "warthog", 4, 36, 1),   // dans le cadre, au milieu du vide de 12 m : masque
		viePosee(2, "warthog", 21, 10, 1),  // dans la saignee de 2 m : la fermeture la bouche, affiche
		viePosee(3, "warthog", 30, 10, 1),  // sur la matiere : affiche
		viePosee(4, "warthog", 60, 10, 1),  // hors du cadre : masque
		viePosee(5, "warthog", 30, 10, -5), // sur la matiere, 5 m sous le sol foule : masque
	)
	if v == nil || v.Zone != sceneryZoneMap || v.Candidates != 5 || v.InPlayArea != 2 {
		t.Fatalf("verdict = %+v", v)
	}
	want := map[uint32]string{
		1: sceneryReasonOffPlayArea, 4: sceneryReasonOffPlayArea, 5: sceneryReasonBelowPlayedFloor,
	}
	got := raisons(v)
	if len(got) != len(want) {
		t.Fatalf("masquees = %v, attendu %v", got, want)
	}
	for slot, r := range want {
		if got[slot] != r {
			t.Errorf("vie %d : raison %q, attendu %q", slot, got[slot], r)
		}
	}
}

// TestVehicleScenery_CacheDuMasque : le masque d un fond n est decode qu une fois, et un fond
// RECUIT (fichier change) est relu — jamais servi perime depuis le cache.
func TestVehicleScenery_CacheDuMasque(t *testing.T) {
	root := t.TempDir()
	fondSynthetique(t, root)
	res := title.NewPathResolver(root)
	chemin := res.MapBackgroundImageFilePath(title.DefaultSlug, cleFondSynthetique+".png")
	meta := res.MapBackgroundMetaPath(title.DefaultSlug, cleFondSynthetique)

	premier, err := sceneryMaskFor(chemin, meta)
	if err != nil {
		t.Fatal(err)
	}
	second, err := sceneryMaskFor(chemin, meta)
	if err != nil {
		t.Fatal(err)
	}
	if premier != second {
		t.Fatal("le meme fond a ete decode deux fois : le cache ne sert pas")
	}
	// Un fond recuit : la matiere couvre tout, l image change de taille et de date.
	plein := image2Plein(t)
	if err := os.WriteFile(chemin, plein, 0o644); err != nil {
		t.Fatal(err)
	}
	demain := time.Now().Add(24 * time.Hour)
	if err := os.Chtimes(chemin, demain, demain); err != nil {
		t.Fatal(err)
	}
	recuit, err := sceneryMaskFor(chemin, meta)
	if err != nil {
		t.Fatal(err)
	}
	if recuit == premier || !recuit.Praticable(4, 36) {
		t.Fatal("fond recuit : le cache a servi l ancien masque")
	}
}

// image2Plein : un PNG 80 x 80 entierement praticable.
func image2Plein(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 80, 80))
	for py := 0; py < 80; py++ {
		for px := 0; px < 80; px++ {
			img.SetNRGBA(px, py, color.NRGBA{A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
