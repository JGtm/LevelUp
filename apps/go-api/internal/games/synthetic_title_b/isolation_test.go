package synthetic_title_b

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/mappings"
)

// repoRoot calcule le chemin de la racine du repo en remontant depuis ce fichier.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")
}

func mustLoadFields(t *testing.T, slug string) *mappings.FieldMappingSet {
	t.Helper()
	path := filepath.Join(repoRoot(t), "config", "titles", slug, "mappings", "fields.toml")
	set, err := mappings.LoadFieldsFromFile(path)
	if err != nil {
		t.Fatalf("LoadFieldsFromFile(%s): %v", path, err)
	}
	return set
}

// TestSyntheticTitleB_FieldsTOMLIsValid garantit que le corpus synthétique
// charge proprement avec le loader strict de Phase A.
func TestSyntheticTitleB_FieldsTOMLIsValid(t *testing.T) {
	t.Parallel()
	set := mustLoadFields(t, TitleSlug)
	if set.TitleSlug() != TitleSlug {
		t.Errorf("TitleSlug = %q, want %q", set.TitleSlug(), TitleSlug)
	}
}

// TestSyntheticTitleB_HasDifferentLabelsThanHI prouve qu'un titre fictif
// peut diverger sémantiquement d'Halo Infinite tout en exposant les mêmes
// FieldKey canoniques.
func TestSyntheticTitleB_HasDifferentLabelsThanHI(t *testing.T) {
	t.Parallel()

	hiSet := mustLoadFields(t, "halo_infinite")
	bSet := mustLoadFields(t, TitleSlug)

	hiKills, _ := hiSet.Get(canonical.FieldKills)
	bKills, _ := bSet.Get(canonical.FieldKills)

	hiLabel, _ := hiKills.Label("en")
	bLabel, _ := bKills.Label("en")

	if hiLabel == bLabel {
		t.Errorf("HI et titre B partagent le même label EN pour kills (%q) — divergence attendue", hiLabel)
	}
	if hiLabel != "Kills" || bLabel != "Eliminations" {
		t.Errorf("labels EN: HI=%q B=%q, want Kills/Eliminations", hiLabel, bLabel)
	}
}

// TestSyntheticTitleB_StoresInDifferentUnits prouve que la couche supporte
// des stockages divergents : le titre B stocke ses durées en ms.
func TestSyntheticTitleB_StoresInDifferentUnits(t *testing.T) {
	t.Parallel()
	set := mustLoadFields(t, TitleSlug)
	d, ok := set.Get(canonical.FieldDurationSeconds)
	if !ok {
		t.Fatal("FieldDurationSeconds absent du titre B")
	}
	if d.StorageUnit != mappings.UnitMilliseconds {
		t.Errorf("storage_unit = %q, want milliseconds", d.StorageUnit)
	}
	if d.DisplayUnit != mappings.UnitSeconds {
		t.Errorf("display_unit = %q, want seconds", d.DisplayUnit)
	}
	out, ok := mappings.ConvertValue(120000, d.StorageUnit, d.DisplayUnit)
	if !ok || out != 120.0 {
		t.Errorf("convert 120000 ms = %v (ok=%v), want 120s", out, ok)
	}
}

// TestSyntheticTitleB_AdapterIsAgnostic prouve que les services produit
// peuvent consommer indifféremment HI et titre B via les mêmes interfaces.
//
// Critère du plan §10.3 : "les services produit n'ont aucun code-path
// conditionnel sur title_slug (sauf le bootstrap et le router de l'adapter)".
func TestSyntheticTitleB_AdapterIsAgnostic(t *testing.T) {
	t.Parallel()

	hiSet := mustLoadFields(t, "halo_infinite")
	bSet := mustLoadFields(t, TitleSlug)

	hiSemantic := halo_infinite.NewSemanticAdapter(hiSet)
	bSemantic := NewSemanticAdapter(bSet)

	// Les deux semantic adapters implémentent strictement la même interface.
	semantics := map[string]games.TitleSemanticAdapter{
		"hi": hiSemantic,
		"b":  bSemantic,
	}
	for tag, sem := range semantics {
		if sem.TitleSlug() == "" {
			t.Errorf("[%s] TitleSlug vide", tag)
		}
		if sem.SchemaVersion() <= 0 {
			t.Errorf("[%s] SchemaVersion = %d", tag, sem.SchemaVersion())
		}
		if sem.Fields() == nil {
			t.Errorf("[%s] Fields() nil", tag)
		}
	}

	// Les data adapters aussi : un consommateur appelle indifféremment.
	hiData := halo_infinite.NewDataAdapter(nil, nil)
	bData := NewDataAdapter(&FakePlayerStats{XUID: "0xB", MatchesPlayed: 5, Kills: 10, Deaths: 4})

	datas := []games.TitleDataAdapter{hiData, bData}
	for _, d := range datas {
		caps := d.Capabilities()
		if caps == nil {
			t.Errorf("[%s] Capabilities nil", d.TitleSlug())
		}
		// Match summaries doit toujours être appelable, même si vide.
		_, _ = d.LoadMatchSummaries(context.Background(), []string{"m1"})
	}
}

// TestSyntheticTitleB_ResolverIsolation prouve que registrer plusieurs titres
// dans le même resolver ne fuit pas (Data(slug_a) ne retourne pas le data de slug_b).
func TestSyntheticTitleB_ResolverIsolation(t *testing.T) {
	t.Parallel()

	hiSet := mustLoadFields(t, "halo_infinite")
	bSet := mustLoadFields(t, TitleSlug)

	resolver := games.NewStaticResolver("halo_infinite")
	resolver.RegisterSemantic(halo_infinite.NewSemanticAdapter(hiSet))
	resolver.RegisterSemantic(NewSemanticAdapter(bSet))
	resolver.RegisterData(halo_infinite.NewDataAdapter(nil, nil))
	resolver.RegisterData(NewDataAdapter(&FakePlayerStats{XUID: "0xB", Kills: 42}))

	// Round-trip : le slug B doit retourner le set B, pas HI.
	bSemantic, err := resolver.Semantic(TitleSlug)
	if err != nil {
		t.Fatalf("Semantic(B) err: %v", err)
	}
	if bSemantic.TitleSlug() != TitleSlug {
		t.Errorf("Semantic(B).TitleSlug = %q", bSemantic.TitleSlug())
	}

	// Le data adapter B retourne ses stats injectées, pas celles d'HI.
	bData, err := resolver.Data(TitleSlug)
	if err != nil {
		t.Fatalf("Data(B) err: %v", err)
	}
	stats, err := bData.LoadPlayerStats(context.Background(), "0xB", canonical.StatsScope{})
	if err != nil {
		t.Fatalf("LoadPlayerStats: %v", err)
	}
	if stats.Kills != 42 {
		t.Errorf("Kills = %d, want 42 (isolation casée — fuite cross-titres)", stats.Kills)
	}

	// Slug inconnu = ErrTitleNotResolved, pas un fallback silencieux sur le default.
	if _, err := resolver.Data("unknown_title"); err == nil {
		t.Errorf("resolver.Data(unknown) devrait retourner ErrTitleNotResolved")
	}
}
