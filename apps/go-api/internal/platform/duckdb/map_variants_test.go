package duckdb

import (
	"context"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/patterns"
	"levelup/go-api/internal/domain"
)

var reglesDeTest = analysis.MapGroupRules{Suffixes: []string{"- Ranked", "Heavies"}}

// metaDesVariantes : metadata `:memory:` avec la seule traduction FR de la carte de base Thunderhead.
func metaDesVariantes(t *testing.T) *DB {
	t.Helper()
	meta, err := OpenReadWrite(":memory:")
	if err != nil {
		t.Fatalf("OpenReadWrite metadata: %v", err)
	}
	t.Cleanup(func() { _ = meta.Close() })
	for _, s := range []string{
		`CREATE OR REPLACE TABLE asset_translations (
			asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR,
			description VARCHAR, fetched_at TIMESTAMP)`,
		`INSERT INTO asset_translations VALUES ('base-thunder','map','fr-FR','Tonnerre','',now())`,
	} {
		if _, err := meta.Exec(context.Background(), s); err != nil {
			t.Fatalf("seed %q: %v", s, err)
		}
	}
	return meta
}

// registreDesVariantes sème le registre : un match par ligne de `cartes` (map_id, map_name ; "" =
// nom NULL).
func registreDesVariantes(t *testing.T, cartes [][2]string) *PlayerDB {
	t.Helper()
	pdb := newTacticalTestPlayerDB(t)
	pdb.Metadata = metaDesVariantes(t)
	debut := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for i, c := range cartes {
		var nom any
		if c[1] != "" {
			nom = c[1]
		}
		tacExec(t, pdb, `INSERT INTO match_registry (match_id, map_id, map_name, start_time, start_time_utc)
			VALUES (?, ?, ?, ?, ?)`, "m"+string(rune('a'+i)), c[0], nom, debut.Add(time.Duration(i)*time.Hour), debut.Add(time.Duration(i)*time.Hour))
	}
	return pdb
}

// Le groupe se construit sur le registre : une base à deux identifiants, une variante sans base, un
// identifiant dont une partie des lignes n'a pas de nom.
func TestMapGroupsRepo_RegistreEtNomRetenu(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{
		{"base-thunder", "Thunderhead"},
		{"base-thunder", ""}, // ligne sans nom : le nom retenu reste Thunderhead
		{"base-thunder-bis", "Thunderhead"},
		{"var-heavies", "Thunderhead Heavies"},
		{"var-ranked", "Live Fire - Ranked"}, // aucune base au registre
		{"base-seule", "The Pit"},
	})
	g, err := NewMapGroupsRepo(pdb).WithMapGroupRules(reglesDeTest).MapGroups(context.Background())
	if err != nil {
		t.Fatalf("MapGroups: %v", err)
	}
	want := []string{"base-thunder", "base-thunder-bis", "var-heavies"}
	if got := g.Membres("var-heavies"); !reflect.DeepEqual(got, want) {
		t.Errorf("Membres(var-heavies) = %v, attendu %v", got, want)
	}
	for _, id := range []string{"var-ranked", "base-seule"} {
		if g.Regroupe(id) {
			t.Errorf("%s ne doit appartenir à aucun groupe", id)
		}
	}
}

func TestMapGroupsRepo_SansSuffixe(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{{"b", "X"}, {"v", "X Heavies"}})
	g, err := NewMapGroupsRepo(pdb).MapGroups(context.Background())
	if err != nil || !g.Vide() {
		t.Errorf("titre sans suffixe : groupe vide attendu, reçu %+v (err %v)", g, err)
	}
}

// Le pipeline de filtres ne voit plus que la carte de base : options et filtrage lisent ces noms.
func TestRattacherCartesDeBase_Filtres(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{
		{"base-thunder", "Thunderhead"},
		{"var-heavies", "Thunderhead Heavies"},
		{"var-ranked", "Live Fire - Ranked"},
	})
	s := func(v string) *string { return &v }
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", MapID: s("var-heavies"), MapName: s("Thunderhead Heavies"), MapNameFR: s("Du lourd sur Thunderhead")},
		{MatchID: "m2", MapID: s("base-thunder"), MapName: s("Thunderhead"), MapNameFR: s("Tonnerre")},
		{MatchID: "m3", MapID: nil, MapName: s("Thunderhead Heavies"), MapNameFR: s("Thunderhead Heavies")},
		// Ligne dont le nom est absent mais dont l'identifiant est une variante : rattachée par l'identifiant.
		{MatchID: "m4", MapID: s("var-heavies"), MapName: nil, MapNameFR: nil},
		{MatchID: "m5", MapID: s("var-ranked"), MapName: s("Live Fire - Ranked"), MapNameFR: s("Live Fire - Ranked")},
	}
	rattacherCartesDeBase(context.Background(), pdb, reglesDeTest, rows)

	for _, i := range []int{0, 3} {
		if *rows[i].MapName != "Thunderhead" || *rows[i].MapNameFR != "Tonnerre" || *rows[i].MapID != "var-heavies" {
			t.Errorf("ligne %d : la variante doit prendre le nom de la base et garder son identifiant : %+v", i, rows[i])
		}
	}
	if *rows[1].MapNameFR != "Tonnerre" {
		t.Errorf("la carte de base ne bouge pas : %+v", rows[1])
	}
	if *rows[2].MapName != "Thunderhead Heavies" {
		t.Errorf("sans identifiant de carte, la ligne reste telle quelle : %+v", rows[2])
	}
	if *rows[4].MapName != "Live Fire - Ranked" {
		t.Errorf("une variante sans base au registre reste seule : %+v", rows[4])
	}
}

// L'historique garde son nom affiché et reçoit le libellé de la base pour le seul filtre.
func TestRattacherCartesDeBaseHistorique(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{
		{"base-thunder", "Thunderhead"},
		{"var-heavies", "Thunderhead Heavies"},
		{"seule", "The Pit"},
	})
	s := func(v string) *string { return &v }
	rows := []domain.MatchHistoryRawRow{
		{MatchID: "m1", MapID: s("var-heavies"), MapName: s("Thunderhead Heavies"), MapNameFR: s("Du lourd sur Thunderhead")},
		{MatchID: "m2", MapID: s("base-thunder"), MapName: s("Thunderhead"), MapNameFR: s("Tonnerre")},
		{MatchID: "m3", MapID: s("seule"), MapName: s("The Pit"), MapNameFR: s("La Fosse")},
	}
	rattacherCartesDeBaseHistorique(context.Background(), pdb, reglesDeTest, rows)

	if rows[0].MapFilterLabel == nil || *rows[0].MapFilterLabel != "Tonnerre" {
		t.Errorf("libellé de filtre de la variante : %v, attendu Tonnerre", rows[0].MapFilterLabel)
	}
	if *rows[0].MapNameFR != "Du lourd sur Thunderhead" {
		t.Errorf("le nom affiché ne doit pas changer : %q", *rows[0].MapNameFR)
	}
	if rows[1].MapFilterLabel == nil || *rows[1].MapFilterLabel != "Tonnerre" {
		t.Errorf("la base porte le même libellé de filtre que ses variantes : %v", rows[1].MapFilterLabel)
	}
	if rows[2].MapFilterLabel != nil {
		t.Errorf("une carte hors groupe n'a pas de libellé de filtre propre : %q", *rows[2].MapFilterLabel)
	}
}

// Le libellé de la base vient d'un autre identifiant de même nom quand le représentant n'est pas
// traduit ; à défaut de toute traduction, c'est le nom EN.
func TestRattacherCartesDeBase_LibelleDeRepli(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{
		{"live", "Live Fire"},
		{"live", "Live Fire"}, // représentant (deux matchs), sans traduction
		{"aaa-live-bis", "Live Fire"},
		{"live-rk", "Live Fire - Ranked"},
		{"pit", "The Pit"},
		{"pit-rk", "The Pit - Ranked"}, // aucune traduction : nom EN
	})
	if _, err := pdb.Metadata.Exec(context.Background(),
		`INSERT INTO asset_translations VALUES ('aaa-live-bis','map','fr-FR','Tir nourri','',now())`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	s := func(v string) *string { return &v }
	rows := []domain.FilterMatchRow{
		{MatchID: "m1", MapID: s("live-rk"), MapName: s("Live Fire - Ranked")},
		{MatchID: "m2", MapID: s("pit-rk"), MapName: s("The Pit - Ranked")},
	}
	rattacherCartesDeBase(context.Background(), pdb, reglesDeTest, rows)
	if *rows[0].MapNameFR != "Tir nourri" {
		t.Errorf("libellé FR = %q, attendu Tir nourri (traduction d'un autre identifiant de la base)", *rows[0].MapNameFR)
	}
	if *rows[1].MapNameFR != "The Pit" {
		t.Errorf("base sans traduction : libellé = %q, attendu The Pit", *rows[1].MapNameFR)
	}
}

// La table rendue au service porte le libellé EN et FR de la base : une ligne regroupée n'affiche
// jamais le nom de sa variante.
func TestMapGroupsRepo_LibellesDeBase(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{
		{"base-thunder", "Thunderhead"},
		{"var-heavies", "Thunderhead Heavies"},
	})
	g, err := NewMapGroupsRepo(pdb).WithMapGroupRules(reglesDeTest).MapGroups(context.Background())
	if err != nil {
		t.Fatalf("MapGroups: %v", err)
	}
	if l, ok := g.Libelle("var-heavies", "fr"); !ok || l != "Tonnerre" {
		t.Errorf("libellé FR = %q, %v, attendu Tonnerre", l, ok)
	}
	if l, ok := g.Libelle("var-heavies", "en"); !ok || l != "Thunderhead" {
		t.Errorf("libellé EN = %q, %v, attendu Thunderhead", l, ok)
	}
}

// Motifs « par carte » : les lignes d'une variante portent l'identifiant de la carte de base.
func TestPatternsRepo_RegrouperLesCartes(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{
		{"base-thunder", "Thunderhead"},
		{"var-heavies", "Thunderhead Heavies"},
		{"seule", "The Pit"},
	})
	rows := []patterns.MatchRow{{MapID: "var-heavies"}, {MapID: "base-thunder"}, {MapID: "seule"}}
	NewPatternsRepo(pdb).WithMapGroupRules(reglesDeTest).regrouperLesCartes(context.Background(), rows)
	if rows[0].MapID != "base-thunder" || rows[1].MapID != "base-thunder" || rows[2].MapID != "seule" {
		t.Errorf("identifiants = %q %q %q, attendu base-thunder base-thunder seule",
			rows[0].MapID, rows[1].MapID, rows[2].MapID)
	}
}

// Le résolveur de clé de module réunit les republications d'une carte intégrée (même nom, même clé),
// sans variante ni suffixe ; les versions de la carte Forge du même registre (aucune clé) forment aussi
// un groupe.
func TestMapGroupsRepo_RepublicationsParCleDeModule(t *testing.T) {
	pdb := registreDesVariantes(t, [][2]string{
		{"bz-1", "Bazaar"}, {"bz-1", "Bazaar"}, {"bz-2", "Bazaar"},
		{"sh-1", "Shogun"}, {"sh-2", "Shogun"},
	})
	modules := func(_ context.Context, maps []analysis.MapNameCount) map[string]string {
		out := map[string]string{}
		for _, m := range maps {
			if m.Name == "Bazaar" {
				out[m.MapID] = "ctf_bazaar"
			}
		}
		return out
	}
	rules := analysis.MapGroupRules{Modules: modules}
	g, err := NewMapGroupsRepo(pdb).WithMapGroupRules(rules).MapGroups(context.Background())
	if err != nil {
		t.Fatalf("MapGroups: %v", err)
	}
	if got, want := g.Membres("bz-2"), []string{"bz-1", "bz-2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Membres(bz-2) = %v, attendu %v", got, want)
	}
	if g.Groupe("bz-2") != "bz-1" {
		t.Errorf("le représentant est l'identifiant le plus joué : %q", g.Groupe("bz-2"))
	}
	if !g.Regroupe("sh-1") || g.Groupe("sh-2") != g.Groupe("sh-1") {
		t.Error("les versions Forge de Shogun, sans clé de module, forment un groupe")
	}
}
