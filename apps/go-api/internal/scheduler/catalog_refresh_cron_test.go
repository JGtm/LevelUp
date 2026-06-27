package scheduler

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// forgeRegistry construit un registre déterministe à 2 titres ACTIFS : halo_infinite
// (avec CapForge → drainable) + un titre modèle Halo 5 sans CapForge (catalogue
// metadata-side, à skipper). Isole les tests d'un éventuel SetDefaultRegistry global.
func forgeRegistry() *titlePkg.Registry {
	reg := titlePkg.NewRegistry() // halo_infinite déjà enregistré (actif, CapForge)
	reg.Register(&titlePkg.TitleDescriptor{
		Slug:         "title_no_forge",
		Name:         "Title Sans Catalogue UGC",
		Provider:     "title_no_forge",
		Status:       titlePkg.StatusActive,
		Capabilities: []titlePkg.Capability{titlePkg.CapMatchmaking}, // pas de CapForge
	})
	return reg
}

func TestCatalogRefreshCron_RunOnce_CallsRunner(t *testing.T) {
	called := 0
	gotTitle := ""
	c := NewCatalogRefreshCron(func(_ context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error) {
		called++
		gotTitle = titleSlug
		return domain.CatalogUGCDrainResult{Playlists: 3, Maps: 5}, nil
	}, "halo_infinite", 0)
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())

	if called != 1 {
		t.Fatalf("runner appelé %d fois, want 1 (seul halo_infinite a CapForge)", called)
	}
	if gotTitle != "halo_infinite" {
		t.Errorf("titre = %q, want halo_infinite", gotTitle)
	}
}

// TestCatalogRefreshCron_RunOnce_SkipsTitleWithoutForge vérifie la brique
// title-aware : un titre actif SANS CapForge (modèle Halo 5, catalogue résolu
// metadata-side) est skippé proprement — aucun drain lancé pour lui — tandis que
// halo_infinite (avec la cap) reste traité.
func TestCatalogRefreshCron_RunOnce_SkipsTitleWithoutForge(t *testing.T) {
	var drained []string
	c := NewCatalogRefreshCron(func(_ context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error) {
		drained = append(drained, titleSlug)
		return domain.CatalogUGCDrainResult{}, nil
	}, "", 0)
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())

	if len(drained) != 1 || drained[0] != "halo_infinite" {
		t.Fatalf("titres drainés = %v, want [halo_infinite] uniquement (title_no_forge skippé)", drained)
	}
}

func TestCatalogRefreshCron_RunOnce_ErrorNoPanic(t *testing.T) {
	c := NewCatalogRefreshCron(func(_ context.Context, _ string) (domain.CatalogUGCDrainResult, error) {
		return domain.CatalogUGCDrainResult{}, errors.New("boom")
	}, "", 0)
	c.registry = forgeRegistry()
	c.RunOnce(context.Background()) // l'erreur d'un titre n'interrompt pas le cycle
}

func TestCatalogRefreshCron_Defaults(t *testing.T) {
	c := NewCatalogRefreshCron(nil, "", 0)
	if c.interval != DefaultCatalogRefreshInterval {
		t.Errorf("interval défaut = %v, want %v", c.interval, DefaultCatalogRefreshInterval)
	}
	if c.registry == nil {
		t.Error("registry défaut ne doit pas être nil (DefaultRegistry)")
	}
	c.RunOnce(context.Background()) // runner nil → no-op, pas de panic
}
