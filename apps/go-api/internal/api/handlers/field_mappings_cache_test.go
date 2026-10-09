// field_mappings_cache_test.go — cache du DTO field-mappings et ETag comparé avant
// toute construction (plan perf 2026-09-23, lot L5b, D5b.2).
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability/timing"
)

// fmCall est le résultat d'une requête field-mappings : statut, ETag, corps et
// sections timing posées pendant la requête.
type fmCall struct {
	code  int
	etag  string
	body  string
	calls map[string]int
}

// serveFieldMappings sert une requête à travers le routeur, contexte instrumenté
// comme par le middleware HTTP (timing.WithTimings).
func serveFieldMappings(t *testing.T, h *FieldMappingsHandler, query, ifNoneMatch string) fmCall {
	t.Helper()
	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) { h.Mount(sub) })
	req := httptest.NewRequest("GET", "/api/v1/titles/test_title/field-mappings"+query, nil)
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}
	ctx, tm := timing.WithTimings(req.Context())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req.WithContext(ctx))
	calls := map[string]int{}
	for _, s := range tm.Snapshot() {
		calls[s.Name] = s.Calls
	}
	return fmCall{code: w.Code, etag: w.Header().Get("ETag"), body: w.Body.String(), calls: calls}
}

func loadFieldsTOML(t *testing.T, doc string) *mappings.FieldMappingSet {
	t.Helper()
	set, err := mappings.LoadFieldsFromBytes("test.toml", []byte(doc))
	if err != nil {
		t.Fatalf("load fields: %v", err)
	}
	return set
}

func loadAssetsTOML(t *testing.T, doc string) *mappings.AssetMappingSet {
	t.Helper()
	set, err := mappings.LoadAssetsFromBytes("test.toml", []byte(doc))
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	return set
}

// TestFieldMappingsHandler_ConditionalHitSkipsBuild : la 2e requête (même version)
// sert le DTO caché — marqueur hit, aucun marqueur miss — et répond 304 à l'ETag.
func TestFieldMappingsHandler_ConditionalHitSkipsBuild(t *testing.T) {
	t.Parallel()
	h := newHandler(&stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)})

	first := serveFieldMappings(t, h, "", "")
	if first.code != http.StatusOK || first.calls["field_mappings_cache_miss"] != 1 {
		t.Fatalf("1re requête : code=%d marqueurs=%v, want 200 + miss", first.code, first.calls)
	}
	second := serveFieldMappings(t, h, "", first.etag)
	if second.code != http.StatusNotModified {
		t.Fatalf("2e requête : code=%d, want 304", second.code)
	}
	if second.calls["field_mappings_cache_hit"] != 1 || second.calls["field_mappings_cache_miss"] != 0 {
		t.Errorf("2e requête : marqueurs=%v, want hit=1 sans miss (aucune construction)", second.calls)
	}
	third := serveFieldMappings(t, h, "", "")
	if third.code != http.StatusOK || third.body != first.body || third.etag != first.etag {
		t.Errorf("3e requête sans ETag : code=%d, corps ou ETag différent du 1er", third.code)
	}
}

// TestFieldMappingsHandler_ETagChangesWhenTOMLChanges : un TOML du titre qui change
// (libellé d'un champ, puis un asset) change l'ETag ; l'ancien ETag n'obtient plus 304.
func TestFieldMappingsHandler_ETagChangesWhenTOMLChanges(t *testing.T) {
	t.Parallel()
	stub := &stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)}
	h := newHandler(stub)
	before := serveFieldMappings(t, h, "", "")

	stub.set = map[string]*mappings.FieldMappingSet{
		"test_title": loadFieldsTOML(t, strings.Replace(handlerTestTOML, "Éliminations", "Frags", 1)),
	}
	afterFields := serveFieldMappings(t, h, "", before.etag)
	if afterFields.code != http.StatusOK || afterFields.etag == before.etag {
		t.Fatalf("fields.toml modifié : code=%d etag=%s (avant %s), want 200 + nouvel ETag",
			afterFields.code, afterFields.etag, before.etag)
	}
	if !strings.Contains(afterFields.body, `"Frags"`) {
		t.Errorf("le corps ne porte pas le nouveau libellé : %s", afterFields.body)
	}

	stub.assets = map[string]*mappings.AssetMappingSet{
		"test_title": loadAssetsTOML(t, strings.Replace(handlerSeasonAssetsTOML, "Classé", "Compétitif", 1)),
	}
	afterAssets := serveFieldMappings(t, h, "", afterFields.etag)
	if afterAssets.code != http.StatusOK || afterAssets.etag == afterFields.etag {
		t.Errorf("assets.toml modifié : code=%d, ETag inchangé=%v", afterAssets.code, afterAssets.etag == afterFields.etag)
	}
}

// mutableSeasonsCatalog est un catalogue de saisons modifiable entre deux requêtes.
type mutableSeasonsCatalog struct{ entries []SeasonCatalogEntry }

func (m *mutableSeasonsCatalog) Load(_ context.Context, _ string) []SeasonCatalogEntry {
	return m.entries
}

// TestFieldMappingsHandler_ETagChangesWhenSeasonCatalogChanges : une saison
// découverte (catalogue qui change) change l'ETag, même sans aucun TOML modifié.
func TestFieldMappingsHandler_ETagChangesWhenSeasonCatalogChanges(t *testing.T) {
	t.Parallel()
	start6 := time.Date(2024, 3, 19, 0, 0, 0, 0, time.UTC)
	catalog := &mutableSeasonsCatalog{entries: []SeasonCatalogEntry{
		{ID: "season6", Label: "Spirit of Fire", Start: start6, DisplayOrder: 60},
	}}
	h := newHandler(&stubRegistry{set: mustLoad(t), assets: mustLoadAssets(t)}).WithSeasonsCatalog(catalog)
	before := serveFieldMappings(t, h, "", "")

	catalog.entries = append(catalog.entries, SeasonCatalogEntry{
		ID: "season14", Label: "Skyfall", Start: time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC), DisplayOrder: 70,
	})
	after := serveFieldMappings(t, h, "", before.etag)
	if after.code != http.StatusOK || after.etag == before.etag {
		t.Fatalf("catalogue modifié : code=%d, ETag inchangé=%v, want 200 + nouvel ETag", after.code, after.etag == before.etag)
	}
	if !strings.Contains(after.body, `"season14"`) {
		t.Errorf("la saison découverte manque au corps")
	}

	end6 := time.Date(2024, 6, 18, 0, 0, 0, 0, time.UTC)
	catalog.entries[0].End = &end6 // correction rétroactive d'une date
	corrected := serveFieldMappings(t, h, "", after.etag)
	if corrected.code != http.StatusOK || corrected.etag == after.etag {
		t.Errorf("date de saison corrigée : code=%d, ETag inchangé=%v", corrected.code, corrected.etag == after.etag)
	}
}

// TestFieldMappingsHandler_ETagStableAcrossInstances : l'ETag est le hash du corps —
// un nouveau process (nouveau handler) rend le même ETag pour le même contenu, les
// caches navigateur restent valides d'un redémarrage à l'autre.
func TestFieldMappingsHandler_ETagStableAcrossInstances(t *testing.T) {
	t.Parallel()
	a := serveFieldMappings(t, newHandler(&stubRegistry{set: mustLoad(t)}), "?locale=en", "")
	b := serveFieldMappings(t, newHandler(&stubRegistry{set: mustLoad(t)}), "?locale=en", a.etag)
	if b.code != http.StatusNotModified {
		t.Errorf("même contenu, autre instance : code=%d, want 304", b.code)
	}
}

// TestFieldMappingsHandler_UnknownLocaleNotCached : une locale hors fr/en est servie
// (repli EN) mais jamais mise en cache — le cache reste borné.
func TestFieldMappingsHandler_UnknownLocaleNotCached(t *testing.T) {
	t.Parallel()
	h := newHandler(&stubRegistry{set: mustLoad(t)})
	serveFieldMappings(t, h, "?locale=es", "")
	second := serveFieldMappings(t, h, "?locale=es", "")
	if second.code != http.StatusOK || second.calls["field_mappings_cache_miss"] != 1 {
		t.Errorf("locale inconnue : code=%d marqueurs=%v, want 200 + miss", second.code, second.calls)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if len(h.dtoByKey) != 0 {
		t.Errorf("entrées cachées = %d, want 0 pour une locale hors fr/en", len(h.dtoByKey))
	}
}
