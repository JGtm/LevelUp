package duckdb

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain"
)

func metaDesVariantes(t *testing.T) *DB {
	t.Helper()
	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	ctx := context.Background()
	for _, s := range []string{
		`CREATE TABLE maps_catalog (title_slug VARCHAR, map_asset_id VARCHAR, name_canonical VARCHAR)`,
		`CREATE TABLE asset_translations (
			asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
			description VARCHAR, fetched_at TIMESTAMP)`,
		// Cartes de base : Thunderhead (traduite), Live Fire (sans traduction).
		`INSERT INTO maps_catalog VALUES ('halo_infinite','base-thunder','Thunderhead')`,
		`INSERT INTO maps_catalog VALUES ('halo_infinite','base-livefire','Live Fire')`,
		`INSERT INTO asset_translations VALUES ('base-thunder','map','fr-FR','Tonnerre','',now())`,
		// Même nom dans un autre titre : ne doit pas être pris.
		`INSERT INTO maps_catalog VALUES ('halo_5','autre-titre','Thunderhead')`,
	} {
		if _, err := meta.Exec(ctx, s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
	return meta
}

var suffixesDeTest = []string{"- Ranked", "Heavies"}

func TestBasesDeCartes(t *testing.T) {
	pdb := &PlayerDB{Metadata: metaDesVariantes(t), TitleSlug: "halo_infinite"}
	noms := map[string]string{
		"var-heavies":   "Thunderhead Heavies",
		"var-ranked":    "Live Fire - Ranked",
		"base-thunder":  "Thunderhead",
		"var-orpheline": "Inconnue Heavies", // aucune carte de base au catalogue
	}
	bases := basesDeCartes(context.Background(), pdb, noms, suffixesDeTest)

	if got := bases["var-heavies"]; got.EN != "Thunderhead" || got.FR != "Tonnerre" {
		t.Errorf("variante Heavies : base %+v, attendu Thunderhead / Tonnerre", got)
	}
	// Base sans traduction FR : le nom EN.
	if got := bases["var-ranked"]; got.EN != "Live Fire" || got.FR != "Live Fire" {
		t.Errorf("variante Ranked : base %+v, attendu Live Fire / Live Fire", got)
	}
	for _, id := range []string{"base-thunder", "var-orpheline"} {
		if _, ok := bases[id]; ok {
			t.Errorf("%s ne doit pas être rattachée : %+v", id, bases[id])
		}
	}
}

func TestBasesDeCartes_SansSuffixeOuSansMetadata(t *testing.T) {
	noms := map[string]string{"var-heavies": "Thunderhead Heavies"}
	pdb := &PlayerDB{Metadata: metaDesVariantes(t), TitleSlug: "halo_infinite"}
	if got := basesDeCartes(context.Background(), pdb, noms, nil); got != nil {
		t.Errorf("sans suffixe : aucune variante attendue, reçu %+v", got)
	}
	if got := basesDeCartes(context.Background(), &PlayerDB{TitleSlug: "halo_infinite"}, noms, suffixesDeTest); got != nil {
		t.Errorf("sans metadata : aucune variante attendue, reçu %+v", got)
	}
}

// Le pipeline de filtres ne voit plus que la carte de base : options et filtrage lisent ces noms.
func TestRattacherCartesDeBase_Filtres(t *testing.T) {
	pdb := &PlayerDB{Metadata: metaDesVariantes(t), TitleSlug: "halo_infinite"}
	s := func(v string) *string { return &v }
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", MapID: s("var-heavies"), MapName: s("Thunderhead Heavies"), MapNameFR: s("Du lourd sur Thunderhead")},
		{MatchID: "m2", MapID: s("base-thunder"), MapName: s("Thunderhead"), MapNameFR: s("Tonnerre")},
		{MatchID: "m3", MapID: nil, MapName: s("Thunderhead Heavies"), MapNameFR: s("Thunderhead Heavies")},
	}
	rattacherCartesDeBase(context.Background(), pdb, suffixesDeTest, rows)

	if *rows[0].MapName != "Thunderhead" || *rows[0].MapNameFR != "Tonnerre" || *rows[0].MapID != "var-heavies" {
		t.Errorf("la variante doit prendre le nom de la base et garder son identifiant : %+v", rows[0])
	}
	if *rows[1].MapNameFR != "Tonnerre" {
		t.Errorf("la carte de base ne bouge pas : %+v", rows[1])
	}
	if *rows[2].MapName != "Thunderhead Heavies" {
		t.Errorf("sans identifiant de carte, la ligne reste telle quelle : %+v", rows[2])
	}
}

// L'historique garde son nom affiché et reçoit le libellé de la base pour le seul filtre.
func TestRattacherCartesDeBaseHistorique(t *testing.T) {
	pdb := &PlayerDB{Metadata: metaDesVariantes(t), TitleSlug: "halo_infinite"}
	s := func(v string) *string { return &v }
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", MapID: s("var-heavies"), MapName: s("Thunderhead Heavies"), MapNameFR: s("Du lourd sur Thunderhead")},
		{MatchID: "m2", MapID: s("base-thunder"), MapName: s("Thunderhead"), MapNameFR: s("Tonnerre")},
	}
	rattacherCartesDeBaseHistorique(context.Background(), pdb, suffixesDeTest, rows)

	if rows[0].MapFilterLabel == nil || *rows[0].MapFilterLabel != "Tonnerre" {
		t.Errorf("libellé de filtre de la variante : %v, attendu Tonnerre", rows[0].MapFilterLabel)
	}
	if *rows[0].MapNameFR != "Du lourd sur Thunderhead" {
		t.Errorf("le nom affiché ne doit pas changer : %q", *rows[0].MapNameFR)
	}
	if rows[1].MapFilterLabel != nil {
		t.Errorf("la carte de base n'a pas de libellé de filtre propre : %q", *rows[1].MapFilterLabel)
	}
}

// À nom égal au catalogue, l'identifiant le plus petit l'emporte : le libellé ne dépend pas de
// l'ordre de lecture.
func TestIdsDeCartesParNom_PlusPetitIdentifiantEnCasDeDoublon(t *testing.T) {
	meta := metaDesVariantes(t)
	if _, err := meta.Exec(context.Background(),
		`INSERT INTO maps_catalog VALUES ('halo_infinite','aaa-thunder-doublon','Thunderhead')`); err != nil {
		t.Fatalf("seed doublon: %v", err)
	}
	pdb := &PlayerDB{Metadata: meta, TitleSlug: "halo_infinite"}
	ids := idsDeCartesParNom(context.Background(), pdb, map[string]struct{}{"Thunderhead": {}})
	if got := ids["Thunderhead"]; got != "aaa-thunder-doublon" {
		t.Errorf("identifiant retenu = %q, attendu aaa-thunder-doublon", got)
	}
}

// Catalogue absent (base non migrée) : aucune variante rattachée, aucune panique.
func TestBasesDeCartes_CatalogueAbsent(t *testing.T) {
	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	pdb := &PlayerDB{Metadata: meta, TitleSlug: "halo_infinite"}
	noms := map[string]string{"var": "Thunderhead Heavies"}
	if got := basesDeCartes(context.Background(), pdb, noms, suffixesDeTest); got != nil {
		t.Errorf("catalogue absent : aucune variante attendue, reçu %+v", got)
	}
}
