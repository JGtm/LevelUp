package halo_infinite

// killsource_icons_test.go — LE RÉFÉRENTIEL ÉNUMÉRÉ EST LA RÉSOLUTION DES ÉLIMINATIONS, PARCOURUE.
//
// L'invariant : pour TOUTE source de dégât du dictionnaire, la clé et l'image que `KillSourceIcon`
// rend pour une élimination sont exactement celles de l'entrée du référentiel pour cette clé.

import (
	"sort"
	"testing"

	"levelup/go-api/internal/port"

	"github.com/JGtm/halo-infinite-film-decoder/killicon"
)

var _ port.KillSourceIconCatalog = (*AssetURLAdapter)(nil)

func TestKillSourceIconsByKey_MemeResolutionQueLesEliminations(t *testing.T) {
	a := NewAssetURLAdapter()
	entries := map[string]string{}
	for _, e := range a.KillSourceIconsByKey() {
		entries[e.WeaponKey] = e.ImageURL
	}
	if len(entries) == 0 {
		t.Fatal("le référentiel est vide : le dictionnaire embarqué porte des clés d'arme")
	}
	verifiees := 0
	for _, tag := range killicon.ResolvedTags() {
		ic, ok := a.KillSourceIcon(tag)
		if !ok || ic.WeaponKey == "" || ic.ImageURL == "" {
			continue
		}
		verifiees++
		url, present := entries[ic.WeaponKey]
		if !present {
			t.Errorf("source %d : KillSourceIcon rend la clé %q, absente du référentiel", tag, ic.WeaponKey)
			continue
		}
		if url != ic.ImageURL {
			t.Errorf("source %d, clé %q : image du référentiel %q, image de l'élimination %q — une clé, une image",
				tag, ic.WeaponKey, url, ic.ImageURL)
		}
	}
	if verifiees == 0 {
		t.Fatal("aucune source vérifiée : le test ne prouve rien")
	}
}

func TestKillSourceIconsByKey_UneEntreeParCleTrieeAvecImage(t *testing.T) {
	list := NewAssetURLAdapter().KillSourceIconsByKey()
	keys := make([]string, 0, len(list))
	seen := map[string]bool{}
	for _, e := range list {
		if e.WeaponKey == "" || e.ImageURL == "" {
			t.Errorf("entrée sans clé ou sans image : %+v", e)
		}
		if !e.Tinted {
			t.Errorf("clé %q : l'atlas du fil d'éliminations est un masque à teindre", e.WeaponKey)
		}
		if seen[e.WeaponKey] {
			t.Errorf("clé %q servie deux fois", e.WeaponKey)
		}
		seen[e.WeaponKey] = true
		keys = append(keys, e.WeaponKey)
	}
	if !sort.StringsAreSorted(keys) {
		t.Errorf("entrées non triées par clé : %v", keys)
	}
}
