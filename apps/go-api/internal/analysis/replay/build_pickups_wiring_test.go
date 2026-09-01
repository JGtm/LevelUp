package replay

// build_pickups_wiring_test.go — LE CÂBLAGE DES DEUX CATALOGUES DE NOMMAGE (revue adversariale
// du schéma 31, P2-2).
//
// LE DÉFAUT QUE CE TEST ATTRAPE, ET POURQUOI AUCUN AUTRE NE LE VOYAIT.
//
// `buildPickups` reçoit ses deux tables de résolution par deux chemins différents du site
// d'appel (`build.go`) :
//
//	clk.families  = opt.Labels.EquipmentFamilies   -> les objets d'équipement
//	weaponKeys    = opt.Labels.Keys                -> les familles d'arme
//
// Les deux sont des `map[uint32]string`. **Les intervertir COMPILE.** Et comme les deux espaces
// d'identifiants sont DISJOINTS (mesuré le 2026-09-01 : zéro identifiant commun, deux films),
// l'interversion ne produit pas de faux noms — elle produit `family` VIDE partout, ce qui est
// exactement ce que `omitempty` rend invisible. Tous les gates seraient restés verts :
// `document_pickups_naming_test.go` appelle `buildPickups` DIRECTEMENT et ne voit donc jamais
// le site d'appel ; le golden n'affiche pas les ramassages ; le contrat ne juge que la forme.
//
// Ce test passe donc par `BuildFromPositions` — LA fonction de production, celle qui porte le
// câblage — et non par `buildPickups`. C'est la même leçon que `build_score_test.go` : prouver
// qu'un calque est JUSTE ne prouve pas qu'il est APPELÉ, ni qu'il est appelé avec les bons
// arguments.
//
// AUCUNE GARDE D'ENVIRONNEMENT : ce test tourne en CI, sur des entrées synthétiques.

import (
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// Les deux identifiants du câblage, choisis DISJOINTS comme ils le sont en production, et
// écrits en littéraux : ce sont de vraies valeurs du corpus (`0xbcabbe43` = grenade à
// fragmentation du manifeste, `0x767db96d` = une famille d'arme du canal i43..i46).
const (
	wirIDEquipement = 0xbcabbe43
	wirIDArme       = 0x767db96d
)

// TestBuildWiresEachCatalogToItsOwnKind — chaque nature reçoit la famille du BON catalogue,
// à travers le chemin de production.
//
// L'INVERSION QUI LE PROUVE : intervertir les deux arguments du site d'appel de `build.go`
// (`clk.families` <-> `weaponKeys`) fait tomber ce test avec « family vide » sur les deux
// natures — vérifié le 2026-09-01. Sans lui, l'interversion passait tous les gates.
func TestBuildWiresEachCatalogToItsOwnKind(t *testing.T) {
	// Deux mini-catalogues DISJOINTS et réalistes : un objet d'équipement, une arme.
	labels := LabelCatalog{
		EquipmentFamilies: map[uint32]string{wirIDEquipement: "grenade_frag"},
		Keys:              map[uint32]string{wirIDArme: "hinf_ma40_ar"},
	}
	// Les ramassages tombent DANS la fenêtre des positions (origine 2 s), sinon ils seraient
	// écartés comme antérieurs à la première frame et le test ne mesurerait rien.
	pickups := []filmdec.BipedPickup{
		{TimestampUS: 2_000_000, Slot: 1, CatalogID: wirIDArme, Class: 0},       // arme
		{TimestampUS: 2_100_000, Slot: 1, CatalogID: wirIDEquipement, Class: 2}, // grenade
	}

	doc := BuildFromPositions("m", "halo_infinite", positionsPourOrigine(), nil, Options{
		FilmClockOriginUS: 1_000_000,
		Pickups:           pickups,
		Labels:            labels,
	})

	if len(doc.Pickups) != 2 {
		t.Fatalf("le document porte %d ramassage(s), attendu 2 — le calque n'est pas câblé",
			len(doc.Pickups))
	}
	// LITTÉRAUX, jamais les constantes du code testé : permuter `PickupWeapon` et
	// `PickupGrenade` en production doit faire tomber ce test aussi.
	for _, c := range []struct {
		i          int
		kind, veut string
	}{
		{0, "weapon", "hinf_ma40_ar"},
		{1, "grenade", "grenade_frag"},
	} {
		got := doc.Pickups[c.i]
		if string(got.Kind) != c.kind {
			t.Errorf("ramassage %d : kind = %q, attendu %q", c.i, got.Kind, c.kind)
		}
		if got.Family != c.veut {
			t.Errorf("ramassage %d (%s) : family = %q, attendu %q — LES DEUX CATALOGUES SONT "+
				"PROBABLEMENT INTERVERTIS au site d'appel de build.go (une famille vide est le "+
				"symptôme exact : les deux espaces d'identifiants sont disjoints)",
				c.i, c.kind, got.Family, c.veut)
		}
	}
	if doc.Coverage == nil || doc.Coverage.Pickups == nil {
		t.Fatal("document sans coverage.pickups : la couverture du calque n'est pas câblée")
	}
	if n := doc.Coverage.Pickups.UnknownFamilies; n != 0 {
		t.Errorf("unknownFamilies = %d, attendu 0 : les deux identifiants sont catalogués", n)
	}
}

// TestBuildLeavesFamilyEmptyWithoutCatalogs — SANS catalogue, le document publie les ramassages
// et les compte comme non résolus.
//
// C'EST LE SENS INVERSE, et il compte autant : il vérifie que l'absence de nom n'efface pas
// l'événement, et que le compteur DIT l'absence au lieu de la taire. Sans lui, un câblage qui
// ne passerait AUCUN catalogue serait indiscernable du bon, `family` étant `omitempty`.
func TestBuildLeavesFamilyEmptyWithoutCatalogs(t *testing.T) {
	doc := BuildFromPositions("m", "halo_infinite", positionsPourOrigine(), nil, Options{
		FilmClockOriginUS: 1_000_000,
		Pickups: []filmdec.BipedPickup{
			{TimestampUS: 2_000_000, Slot: 1, CatalogID: wirIDArme, Class: 0},
			{TimestampUS: 2_100_000, Slot: 1, CatalogID: wirIDEquipement, Class: 2},
		},
	})
	if len(doc.Pickups) != 2 {
		t.Fatalf("le document porte %d ramassage(s), attendu 2 : un ramassage anonyme reste publié",
			len(doc.Pickups))
	}
	for i, p := range doc.Pickups {
		if p.Family != "" {
			t.Errorf("ramassage %d : family = %q sans aucun catalogue, attendu vide", i, p.Family)
		}
	}
	if n := doc.Coverage.Pickups.UnknownFamilies; n != 2 {
		t.Errorf("unknownFamilies = %d, attendu 2 — le compteur doit DIRE l'absence", n)
	}
}
