package mappings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadAssetsFromBytes_HappyPath(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50

[assets.challenge_tier.heroic]
labels = { en = "Heroic", fr = "Héroïque" }
color_token = "challenge.heroic"
display_order = 20
`)
	set, err := LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	if set.TitleSlug() != "halo_infinite" {
		t.Errorf("TitleSlug = %q, want halo_infinite", set.TitleSlug())
	}
	if set.SchemaVersion() != 1 {
		t.Errorf("SchemaVersion = %d, want 1", set.SchemaVersion())
	}

	got, ok := set.Get("mode", "ranked")
	if !ok {
		t.Fatal("Get(mode, ranked) introuvable")
	}
	if lbl, _ := got.Label("fr"); lbl != "Classé" {
		t.Errorf("Label fr = %q, want Classé", lbl)
	}
	if got.DisplayOrder != 50 {
		t.Errorf("DisplayOrder = %d, want 50", got.DisplayOrder)
	}

	tier, ok := set.Get("challenge_tier", "heroic")
	if !ok {
		t.Fatal("Get(challenge_tier, heroic) introuvable")
	}
	if tier.ColorToken != "challenge.heroic" {
		t.Errorf("ColorToken = %q, want challenge.heroic", tier.ColorToken)
	}
}

func TestLoadAssetsFromBytes_AllOfKindOrder(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.bbb]
labels = { en = "BBB", fr = "BBB" }
display_order = 30

[assets.mode.aaa]
labels = { en = "AAA", fr = "AAA" }
display_order = 10

[assets.mode.ccc]
labels = { en = "CCC", fr = "CCC" }
display_order = 20
`)
	set, err := LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	got := set.AllOfKind("mode")
	if len(got) != 3 {
		t.Fatalf("AllOfKind len = %d, want 3", len(got))
	}
	want := []string{"aaa", "ccc", "bbb"}
	for i, m := range got {
		if m.ID != want[i] {
			t.Errorf("position %d: id = %q, want %q", i, m.ID, want[i])
		}
	}
}

func TestLoadAssetsFromBytes_LabelFallback(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50
`)
	set, _ := LoadAssetsFromBytes("test.toml", doc)
	a, _ := set.Get("mode", "ranked")
	// Locale inconnue → fallback EN.
	lbl, used := a.Label("de")
	if lbl != "Ranked" || !used {
		t.Errorf("fallback de: got %q used=%v, want Ranked + true", lbl, used)
	}
}

func TestLoadAssetsFromBytes_LabelFallbackToID(t *testing.T) {
	// Cas dégradé : un set construit à la main sans labels.
	a := AssetMapping{Kind: "mode", ID: "unknown"}
	lbl, used := a.Label("fr")
	if lbl != "unknown" || !used {
		t.Errorf("fallback to id: got %q used=%v, want unknown + true", lbl, used)
	}
}

func TestLoadAssetsFromBytes_MissingMeta(t *testing.T) {
	doc := []byte(`
[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for missing meta")
	}
	if !strings.Contains(err.Error(), "title_slug manquant") {
		t.Errorf("error = %v, want title_slug manquant", err)
	}
}

func TestLoadAssetsFromBytes_MissingLabel(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked" }
display_order = 50
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for missing FR label")
	}
	if !strings.Contains(err.Error(), "label FR manquant") {
		t.Errorf("error = %v, want label FR manquant", err)
	}
}

func TestLoadAssetsFromBytes_DisplayOrderCollision(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.aaa]
labels = { en = "AAA", fr = "AAA" }
display_order = 10

[assets.mode.bbb]
labels = { en = "BBB", fr = "BBB" }
display_order = 10
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for display_order collision")
	}
	if !strings.Contains(err.Error(), "collision") {
		t.Errorf("error = %v, want collision", err)
	}
}

func TestNewAssetMappingSet_NilMap(t *testing.T) {
	set := NewAssetMappingSet("hi", 1, nil)
	if set.AllOfKind("any") != nil {
		t.Error("nil map should yield nil AllOfKind")
	}
	if len(set.Kinds()) != 0 {
		t.Error("nil map should yield 0 kinds")
	}
}

func TestAssetMappingSet_NilSafe(t *testing.T) {
	var set *AssetMappingSet
	if _, ok := set.Get("mode", "ranked"); ok {
		t.Error("nil set Get should return false")
	}
	if set.AllOfKind("mode") != nil {
		t.Error("nil set AllOfKind should return nil")
	}
	if set.Kinds() != nil {
		t.Error("nil set Kinds should return nil")
	}
}

func TestLoadAssetsFromFile_FileNotFound(t *testing.T) {
	_, err := LoadAssetsFromFile(filepath.Join(t.TempDir(), "absent.toml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("error = %v, want read prefix", err)
	}
}

// ─── Saisons : champs optionnels start_date / end_date / extra ───────────────

func TestLoadAssets_SeasonsWithDates(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.season.season1]
labels = { en = "Heroes of Reach", fr = "Heroes of Reach" }
display_order = 10
start_date = "2021-12-08T00:00:00Z"
end_date   = "2022-05-03T00:00:00Z"
extra = { csr_season_id = "CsrSeason1", short_label = "S1" }
`)
	set, err := LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	a, ok := set.Get("season", "season1")
	if !ok {
		t.Fatal("Get(season, season1) introuvable")
	}
	if a.StartDate == nil || a.StartDate.Format(time.RFC3339) != "2021-12-08T00:00:00Z" {
		t.Errorf("StartDate = %v, want 2021-12-08T00:00:00Z", a.StartDate)
	}
	if a.EndDate == nil || a.EndDate.Format(time.RFC3339) != "2022-05-03T00:00:00Z" {
		t.Errorf("EndDate = %v, want 2022-05-03T00:00:00Z", a.EndDate)
	}
	if a.Extra["csr_season_id"] != "CsrSeason1" {
		t.Errorf("Extra[csr_season_id] = %q, want CsrSeason1", a.Extra["csr_season_id"])
	}
	if a.Extra["short_label"] != "S1" {
		t.Errorf("Extra[short_label] = %q, want S1", a.Extra["short_label"])
	}
}

func TestLoadAssets_OpenSeasonNoEndDate(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.season.s_current]
labels = { en = "Current", fr = "Courante" }
display_order = 999
start_date = "2026-04-01T00:00:00Z"
`)
	set, err := LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	a, ok := set.Get("season", "s_current")
	if !ok {
		t.Fatal("Get(season, s_current) introuvable")
	}
	if a.StartDate == nil {
		t.Fatal("StartDate doit être non-nil")
	}
	if a.EndDate != nil {
		t.Errorf("EndDate = %v, want nil pour saison ouverte", a.EndDate)
	}
}

func TestLoadAssets_InvalidDateFormat(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.season.bad]
labels = { en = "Bad", fr = "Bad" }
display_order = 10
start_date = "2022-05-03"
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
	if !strings.Contains(err.Error(), "RFC 3339") {
		t.Errorf("error = %v, want RFC 3339 mention", err)
	}
}

func TestLoadAssets_EndBeforeStart(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.season.bad]
labels = { en = "Bad", fr = "Bad" }
display_order = 10
start_date = "2022-05-03T00:00:00Z"
end_date   = "2022-01-01T00:00:00Z"
`)
	_, err := LoadAssetsFromBytes("test.toml", doc)
	if err == nil {
		t.Fatal("expected error when end_date < start_date")
	}
	if !strings.Contains(err.Error(), "avant start_date") {
		t.Errorf("error = %v, want 'avant start_date' mention", err)
	}
}

func TestLoadAssets_LegacyKindsUnaffected(t *testing.T) {
	doc := []byte(`
[meta]
title_slug = "halo_infinite"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 50
`)
	set, err := LoadAssetsFromBytes("test.toml", doc)
	if err != nil {
		t.Fatalf("LoadAssetsFromBytes: %v", err)
	}
	a, _ := set.Get("mode", "ranked")
	if a.StartDate != nil {
		t.Errorf("StartDate = %v, want nil pour kind sans dates", a.StartDate)
	}
	if a.EndDate != nil {
		t.Errorf("EndDate = %v, want nil", a.EndDate)
	}
	if a.Extra != nil {
		t.Errorf("Extra = %v, want nil pour kind sans extra", a.Extra)
	}
}

func TestLoadAssetsFromFile_RealFile(t *testing.T) {
	tomlPath := filepath.Join(t.TempDir(), "assets.toml")
	doc := []byte(`
[meta]
title_slug = "smoke"
schema_version = 1

[assets.mode.ranked]
labels = { en = "Ranked", fr = "Classé" }
display_order = 10
`)
	if err := os.WriteFile(tomlPath, doc, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	set, err := LoadAssetsFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadAssetsFromFile: %v", err)
	}
	if set.TitleSlug() != "smoke" {
		t.Errorf("TitleSlug = %q", set.TitleSlug())
	}
}

// TestLoadAssetsFromFile_HaloInfiniteSeasonsCatalog cadenasse le contenu réel
// de config/titles/halo_infinite/mappings/assets.toml :
//   - >= 13 entrées dans le kind "season"
//   - toutes les saisons ont StartDate non-nil
//   - DisplayOrder strictement croissant (pas de collision par construction
//     car le validator l'aurait rejeté, mais on vérifie l'ordre logique)
//   - extra.csr_season_id présent partout (utile pour cross-réf futures)
//
// Cas d'usage : sécurise qu'un futur ajout d'opération ne casse rien et
// que le catalog reste cohérent.
func TestLoadAssetsFromFile_HaloInfiniteSeasonsCatalog(t *testing.T) {
	// Path relatif au repo : on remonte depuis le package mappings.
	// Structure : apps/go-api/internal/games/mappings → repo root + 5 niveaux.
	repoRoot := filepath.Join("..", "..", "..", "..", "..")
	tomlPath := filepath.Join(repoRoot, "config", "titles", "halo_infinite", "mappings", "assets.toml")

	set, err := LoadAssetsFromFile(tomlPath)
	if err != nil {
		t.Fatalf("LoadAssetsFromFile(%s): %v", tomlPath, err)
	}

	seasons := set.AllOfKind("season")
	if len(seasons) < 13 {
		t.Fatalf("len(seasons) = %d, want >= 13 (S1-S13 + Winter Update)", len(seasons))
	}

	prevOrder := -1
	for _, s := range seasons {
		if s.StartDate == nil {
			t.Errorf("season %q : StartDate nil (toutes les saisons doivent en avoir une)", s.ID)
		}
		if s.Extra["csr_season_id"] == "" {
			t.Errorf("season %q : extra.csr_season_id manquant", s.ID)
		}
		if s.DisplayOrder <= prevOrder {
			t.Errorf("season %q : display_order=%d non strictement croissant (précédent=%d)", s.ID, s.DisplayOrder, prevOrder)
		}
		prevOrder = s.DisplayOrder
	}

	// Au moins une saison ouverte (la courante) — fin nil.
	hasOpen := false
	for _, s := range seasons {
		if s.EndDate == nil {
			hasOpen = true
			break
		}
	}
	if !hasOpen {
		t.Errorf("aucune saison ouverte (sans end_date) dans le catalog — la saison courante doit l'être")
	}
}
