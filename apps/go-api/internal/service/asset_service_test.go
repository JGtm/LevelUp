package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// mockAssetMetaRepo implémente port.AssetMetaRepository pour les tests.
type mockAssetMetaRepo struct {
	maps    []canonical.AssetMeta
	weapons []canonical.AssetMeta
	err     error
}

func (m *mockAssetMetaRepo) ListMapsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return m.maps, m.err
}

func (m *mockAssetMetaRepo) ListWeaponsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return m.weapons, m.err
}

func (m *mockAssetMetaRepo) ListMedalsByTitle(_ context.Context, _, _ string) ([]canonical.AssetMeta, error) {
	return nil, m.err
}

func TestAssetService_ListMaps_EnrichesImageURL(t *testing.T) {
	repo := &mockAssetMetaRepo{
		maps: []canonical.AssetMeta{
			{ID: "map-001", NameEN: "Aquarius", NameFR: "Aquarius"},
			{ID: "map-002", NameEN: "Breaker", NameFR: "Breaker"},
		},
	}
	svc := NewAssetService(repo)

	items, err := svc.ListMaps(context.Background(), "halo_infinite", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2", len(items))
	}
	want := "/api/v1/assets/maps/halo_infinite/map-001/image"
	if items[0].ImageURL != want {
		t.Errorf("ImageURL=%q, want %q", items[0].ImageURL, want)
	}
	want2 := "/api/v1/assets/maps/halo_infinite/map-002/image"
	if items[1].ImageURL != want2 {
		t.Errorf("ImageURL=%q, want %q", items[1].ImageURL, want2)
	}
}

func TestAssetService_ListMaps_RepoError(t *testing.T) {
	repo := &mockAssetMetaRepo{err: errors.New("db fail")}
	svc := NewAssetService(repo)

	_, err := svc.ListMaps(context.Background(), "halo_infinite", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestAssetService_ListMaps_EmptyResult(t *testing.T) {
	repo := &mockAssetMetaRepo{maps: nil}
	svc := NewAssetService(repo)

	items, err := svc.ListMaps(context.Background(), "unknown_title", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len=%d, want 0", len(items))
	}
}

func TestAssetService_ListWeapons_NoImageURL(t *testing.T) {
	repo := &mockAssetMetaRepo{
		weapons: []canonical.AssetMeta{
			{ID: "100", NameEN: "BR75 Battle Rifle", NameFR: "Fusil BR75"},
		},
	}
	svc := NewAssetService(repo)

	items, err := svc.ListWeapons(context.Background(), "halo_infinite", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d, want 1", len(items))
	}
	if items[0].ImageURL != "" {
		t.Errorf("ImageURL=%q, want empty (B2 gap — no weapon_id→file mapping in V1)", items[0].ImageURL)
	}
}

func TestAssetService_ListWeapons_RepoError(t *testing.T) {
	repo := &mockAssetMetaRepo{err: errors.New("db fail")}
	svc := NewAssetService(repo)

	_, err := svc.ListWeapons(context.Background(), "halo_infinite", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ── Lot rr/L4 (2026-09-23) : une carte par VISUEL dans le tiroir ────────────────
//
// Le dépôt rend UNE LIGNE PAR ASSET (décision D15, test
// TestListMapsByTitle_DedupeParAssetIDPasParNom) ; plusieurs assets distincts (versions
// republiées, copies Forge) partagent pourtant le même visuel. Les fixtures couvrent
// les trois populations mesurées sur le catalogue réel : homonymes (même nom anglais,
// même libellé FR), nom anglais absent (identifiant brut : aucune image, exclu) et
// libellés FR divergents pour une même image. Identifiants fictifs.

// imageParNomAnglais imite un adapter dont l'image est clé par le nom anglais ; un nom
// sans image connue (ici : préfixe "uuid-") rend "" et sort du tiroir.
func imageParNomAnglais(_ string, nameEN string) string {
	if nameEN == "" || strings.HasPrefix(nameEN, "uuid-") {
		return ""
	}
	return "/static/maps/t/" + nameEN + ".jpg"
}

func listMapsAvecImages(t *testing.T, maps []canonical.AssetMeta, search string) []canonical.AssetMeta {
	t.Helper()
	svc := NewAssetService(&mockAssetMetaRepo{maps: maps}).WithMapImageURL(imageParNomAnglais)
	items, err := svc.ListMaps(context.Background(), "titre", search)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return items
}

func TestAssetService_ListMaps_UneCarteParVisuel(t *testing.T) {
	items := listMapsAvecImages(t, []canonical.AssetMeta{
		{ID: "e0000003", NameEN: "Solution", NameFR: "Solution"},
		{ID: "a0000001", NameEN: "Absolution", NameFR: "Absolution"},
		{ID: "70000002", NameEN: "Solution", NameFR: "Solution"},
		{ID: "30000001", NameEN: "Solution", NameFR: "Solution"},
	}, "")
	if len(items) != 2 {
		t.Fatalf("len=%d, want 2 (Absolution + une seule Solution) : %+v", len(items), items)
	}
	if items[0].NameEN != "Absolution" || items[1].NameEN != "Solution" {
		t.Fatalf("ordre = %q, %q ; want Absolution, Solution", items[0].NameEN, items[1].NameEN)
	}
	// Libellés égaux : le plus petit identifiant départage (ordre total, stable).
	if items[1].ID != "30000001" {
		t.Errorf("représentant de Solution = %q, want 30000001 (plus petit ID)", items[1].ID)
	}
}

func TestAssetService_ListMaps_LibelleFRTraduitRetenu(t *testing.T) {
	items := listMapsAvecImages(t, []canonical.AssetMeta{
		{ID: "10000000", NameEN: "Starboard", NameFR: "Starboard"},
		{ID: "90000000", NameEN: "Starboard", NameFR: "Tribord"},
		{ID: "50000000", NameEN: "Starboard", NameFR: "Starboard"},
		{ID: "70000000", NameEN: "Starboard", NameFR: "Tribord"},
	}, "")
	if len(items) != 1 {
		t.Fatalf("len=%d, want 1 : %+v", len(items), items)
	}
	// Le libellé TRADUIT prime sur un ID plus petit (décision Q27 du 2026-09-23).
	if items[0].NameFR != "Tribord" || items[0].ID != "70000000" {
		t.Errorf("représentant = %q/%q, want Tribord/70000000", items[0].NameFR, items[0].ID)
	}
}

func TestAssetService_ListMaps_LibelleFRPresentAvantAbsent(t *testing.T) {
	items := listMapsAvecImages(t, []canonical.AssetMeta{
		{ID: "10000000", NameEN: "Curfew", NameFR: ""},
		{ID: "20000000", NameEN: "Curfew", NameFR: "Curfew"},
	}, "")
	if len(items) != 1 || items[0].ID != "20000000" {
		t.Fatalf("got %+v, want une seule carte 20000000 (libellé FR présent)", items)
	}
}

func TestAssetService_ListMaps_TriParNom(t *testing.T) {
	// Sans builder d'image, l'URL est dérivée de l'ID : deux assets homonymes restent deux
	// cartes (visuels distincts), triées par nom anglais puis par ID. Chasm/Gouffre et
	// Forest/Forêt fixent la clé : l'ordre anglais (Chasm, Forest) contredit l'ordre des
	// libellés FR (Forêt, Gouffre) — un tri par NameFR échoue ici.
	svc := NewAssetService(&mockAssetMetaRepo{maps: []canonical.AssetMeta{
		{ID: "c", NameEN: "Recharge", NameFR: "Recharge"},
		{ID: "d", NameEN: "Forest", NameFR: "Forêt"},
		{ID: "b", NameEN: "aquarius", NameFR: "aquarius"},
		{ID: "z", NameEN: "Behemoth", NameFR: "Behemoth"},
		{ID: "e", NameEN: "Chasm", NameFR: "Gouffre"},
		{ID: "a", NameEN: "Behemoth", NameFR: "Behemoth"},
	}})
	items, err := svc.ListMaps(context.Background(), "titre", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got []string
	for _, m := range items {
		got = append(got, m.NameEN+"/"+m.ID)
	}
	want := "aquarius/b Behemoth/a Behemoth/z Chasm/e Forest/d Recharge/c"
	if strings.Join(got, " ") != want {
		t.Errorf("ordre = %q, want %q", strings.Join(got, " "), want)
	}
}

// Invariant du tiroir sur un catalogue mêlant les trois populations : aucune image en
// double, aucun asset sans image, et l'instantané du dépôt n'est pas réordonné (le
// StaticAssetMetaRepo sert sa tranche en mémoire telle quelle quand search est vide).
func TestAssetService_ListMaps_AucuneImageEnDouble(t *testing.T) {
	maps := []canonical.AssetMeta{
		{ID: "40000000", NameEN: "Streets", NameFR: "Streets"},
		{ID: "20000000", NameEN: "Streets", NameFR: "Streets"},
		{ID: "uuid-1", NameEN: "uuid-1"},
		{ID: "60000000", NameEN: "Perilous", NameFR: "Périlleux"},
		{ID: "10000000", NameEN: "Perilous", NameFR: "Perilous"},
		{ID: "30000000", NameEN: "Aquarius", NameFR: "Aquarius"},
	}
	ordreAvant := make([]string, len(maps))
	for i := range maps {
		ordreAvant[i] = maps[i].ID
	}
	items := listMapsAvecImages(t, maps, "")
	vues := map[string]bool{}
	for _, m := range items {
		if m.ImageURL == "" || vues[m.ImageURL] {
			t.Fatalf("image vide ou en double : %+v", items)
		}
		vues[m.ImageURL] = true
	}
	if len(items) != 3 {
		t.Fatalf("len=%d, want 3 (Aquarius, Perilous, Streets) : %+v", len(items), items)
	}
	if items[1].NameFR != "Périlleux" {
		t.Errorf("Perilous : libellé retenu %q, want Périlleux", items[1].NameFR)
	}
	for i := range maps {
		if maps[i].ID != ordreAvant[i] {
			t.Fatalf("l'instantané du dépôt a été réordonné : %v -> position %d = %q", ordreAvant, i, maps[i].ID)
		}
	}
}

func TestAssetService_ListMaps_RechercheAvantRegroupement(t *testing.T) {
	// La recherche filtre d'abord (dépôt) ; le regroupement porte sur ce qui reste. Deux
	// versions d'un même visuel portent deux libellés FR traduits différents : le
	// représentant GLOBAL serait « Flanc droit » (plus petit ID), qui ne contient pas
	// « tribord ». Regrouper avant de chercher rendrait donc 0 carte ; chercher d'abord
	// rend la version dont le libellé correspond à la saisie.
	repo := NewStaticAssetMetaRepo([]canonical.AssetMeta{
		{ID: "10000000", NameEN: "Starboard", NameFR: "Flanc droit"},
		{ID: "90000000", NameEN: "Starboard", NameFR: "Tribord"},
		{ID: "40000000", NameEN: "Aquarius", NameFR: "Aquarius"},
	}, nil)
	svc := NewAssetService(repo).WithMapImageURL(imageParNomAnglais)
	items, err := svc.ListMaps(context.Background(), "titre", "tribord")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "90000000" || items[0].NameFR != "Tribord" {
		t.Fatalf("got %+v, want une seule carte 90000000/Tribord", items)
	}
}

func TestOneCardPerImage_EntreeIntacte(t *testing.T) {
	// Le helper ne réordonne ni ne modifie la tranche reçue : il rend une tranche neuve.
	in := []canonical.AssetMeta{
		{ID: "2", NameEN: "Solution", NameFR: "Solution", ImageURL: "/s.jpg"},
		{ID: "1", NameEN: "Solution", NameFR: "Solution", ImageURL: "/s.jpg"},
		{ID: "3", NameEN: "Perilous", NameFR: "Périlleux", ImageURL: "/p.jpg"},
		{ID: "0", NameEN: "Perilous", NameFR: "Perilous", ImageURL: "/p.jpg"},
	}
	avant := append([]canonical.AssetMeta(nil), in...)
	out := oneCardPerImage(in)
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2 : %+v", len(out), out)
	}
	for i := range in {
		if in[i] != avant[i] {
			t.Fatalf("entrée modifiée en position %d : %+v, want %+v", i, in[i], avant[i])
		}
	}
}

func TestOneCardPerImage_SansImageJamaisFusionnees(t *testing.T) {
	// Une URL vide n'est pas une identité visuelle : deux cartes sans image restent deux.
	out := oneCardPerImage([]canonical.AssetMeta{
		{ID: "b", NameEN: "Sans image B"},
		{ID: "a", NameEN: "Sans image A"},
		{ID: "c", NameEN: "Avec image", ImageURL: "/x.jpg"},
	})
	if len(out) != 3 {
		t.Fatalf("len=%d, want 3 : %+v", len(out), out)
	}
}
