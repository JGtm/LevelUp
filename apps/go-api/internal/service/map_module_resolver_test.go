package service

import (
	"context"
	"testing"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/testutil"
)

// Une carte intégrée a une clé de module (par son nom retenu) ; une carte Forge, dont le catalogue des
// bornes ne connaît que le canevas, n'en a pas ; un titre sans catalogue des bornes n'a pas de résolveur.
func TestMapModuleResolverFor_Fixture(t *testing.T) {
	root := calloutsFixture(t)
	resolve, ok := MapModuleResolverFor(root, title.DefaultSlug)
	if !ok {
		t.Fatal("le titre a un catalogue des bornes : un résolveur est attendu")
	}
	got := resolve(context.Background(), []analysis.MapNameCount{
		{MapID: "a", Name: "Cliffhanger"},
		{MapID: "b", Name: "Dynasty"},
		{MapID: "c", Name: "Inconnue"},
	})
	if got["a"] != "ridgeline" {
		t.Errorf("Cliffhanger : clé %q, attendu ridgeline", got["a"])
	}
	if _, ok := got["b"]; ok {
		t.Errorf("Dynasty (carte Forge, canevas fo08_wetland sans entrée de zones) n'a pas de clé : %q", got["b"])
	}
	if _, ok := got["c"]; ok {
		t.Error("un nom inconnu n'a pas de clé")
	}
	if _, ok := MapModuleResolverFor(root, "titre_sans_catalogue"); ok {
		t.Error("un titre sans catalogue des bornes n'a pas de résolveur")
	}
}

// Sur le catalogue versionné réel : les republications d'une carte intégrée partagent sa clé, les
// cartes Forge (Shogun, Houseki) n'en ont pas.
func TestMapModuleResolverFor_DonneesReelles(t *testing.T) {
	root, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du dépôt introuvable : %v", err)
	}
	resolve, ok := MapModuleResolverFor(root, title.DefaultSlug)
	if !ok {
		t.Fatal("le catalogue des bornes versionné est absent")
	}
	got := resolve(context.Background(), []analysis.MapNameCount{
		{MapID: "bz1", Name: "Bazaar"}, {MapID: "bz2", Name: "Bazaar"},
		{MapID: "lf", Name: "Live Fire - Ranked"},
		{MapID: "sh1", Name: "Shogun"}, {MapID: "sh2", Name: "Shogun"},
		{MapID: "ho", Name: "Houseki"},
	})
	for id, want := range map[string]string{"bz1": "ctf_bazaar", "bz2": "ctf_bazaar", "lf": "sgh_interlock"} {
		if got[id] != want {
			t.Errorf("%s : clé %q, attendu %q", id, got[id], want)
		}
	}
	for _, id := range []string{"sh1", "sh2", "ho"} {
		if k, ok := got[id]; ok {
			t.Errorf("%s : carte Forge, aucune clé attendue, reçu %q", id, k)
		}
	}
}
