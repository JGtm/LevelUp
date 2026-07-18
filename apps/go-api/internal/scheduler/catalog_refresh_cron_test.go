package scheduler

import (
	"context"
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

// cronRecord retourne le CronStatusRecord d'un cron nommé (fatal si absent).
func cronRecord(t *testing.T, name string) observability.CronStatusRecord {
	t.Helper()
	for _, r := range observability.CronStatusSnapshot() {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("cron %q absent du snapshot de statut", name)
	return observability.CronStatusRecord{}
}

// TestCatalogRefreshCron_PartialFailure_ReportedToCronStatus (F10, décision D1) :
// un cycle où un titre échoue rapporte l'erreur RÉELLE à ReportCronRun → le statut
// du cron devient un ÉCHEC AVEC CAUSE (LastError renseigné, consecutive_failures
// incrémenté), et un cycle réussi remet le compteur à zéro. Avant F10 le cron était
// « toujours vert » (ReportCronRun(..., nil, ...) inconditionnel).
func TestCatalogRefreshCron_PartialFailure_ReportedToCronStatus(t *testing.T) {
	observability.ResetCronStatus()
	t.Cleanup(observability.ResetCronStatus)

	fail := true
	c := NewCatalogRefreshCron(func(_ context.Context, _ string) (domain.CatalogUGCDrainResult, error) {
		if fail {
			return domain.CatalogUGCDrainResult{}, errors.New("drain boom")
		}
		return domain.CatalogUGCDrainResult{}, nil
	}, "", 0).WithCatalogAdapterCheck(catalogPresentForInfinite)
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())
	rec := cronRecord(t, "catalog_refresh")
	if rec.ConsecutiveFailures != 1 {
		t.Fatalf("échec partiel : ConsecutiveFailures = %d, want 1 (failure visible)", rec.ConsecutiveFailures)
	}
	if !strings.Contains(rec.LastError, "halo_infinite") || !strings.Contains(rec.LastError, "boom") {
		t.Errorf("LastError = %q, want cause avec titre + erreur réelle", rec.LastError)
	}

	c.RunOnce(context.Background())
	if rec := cronRecord(t, "catalog_refresh"); rec.ConsecutiveFailures != 2 {
		t.Fatalf("2e échec : ConsecutiveFailures = %d, want 2 (incrémente)", rec.ConsecutiveFailures)
	}

	fail = false
	c.RunOnce(context.Background())
	if rec := cronRecord(t, "catalog_refresh"); rec.ConsecutiveFailures != 0 || rec.LastError != "" {
		t.Errorf("après succès : ConsecutiveFailures=%d LastError=%q, want 0 / vide", rec.ConsecutiveFailures, rec.LastError)
	}
}

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

// catalogPresentForInfinite est un CatalogAdapterChecker de test : seul halo_infinite
// a un catalog adapter résolvable (équivalent de Catalog(slug) == nil) ; tout autre
// titre (ex. title_no_forge) renvoie false (équivalent ErrTitleNotResolved). C'est le
// gate RÉEL injecté qui remplace le proxy CapForge.
func catalogPresentForInfinite(titleSlug string) bool {
	return titleSlug == "halo_infinite"
}

func TestCatalogRefreshCron_RunOnce_CallsRunner(t *testing.T) {
	called := 0
	gotTitle := ""
	c := NewCatalogRefreshCron(func(_ context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error) {
		called++
		gotTitle = titleSlug
		return domain.CatalogUGCDrainResult{Playlists: 3, Maps: 5}, nil
	}, "halo_infinite", 0).WithCatalogAdapterCheck(catalogPresentForInfinite)
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())

	if called != 1 {
		t.Fatalf("runner appelé %d fois, want 1 (seul halo_infinite a un catalog adapter)", called)
	}
	if gotTitle != "halo_infinite" {
		t.Errorf("titre = %q, want halo_infinite", gotTitle)
	}
}

// TestCatalogRefreshCron_RunOnce_SkipsTitleWithoutCatalogAdapter vérifie le gate
// RÉEL (injecté) : un titre actif dont le catalog adapter n'est PAS résolvable
// (équivalent Catalog(slug) == ErrTitleNotResolved, modèle Halo 5 résolu
// metadata-side) est skippé proprement — aucun drain lancé pour lui — tandis que
// halo_infinite (adapter présent) reste traité. C'est le test « présence d'adapter »
// qui remplace le proxy CapForge.
func TestCatalogRefreshCron_RunOnce_SkipsTitleWithoutCatalogAdapter(t *testing.T) {
	var drained []string
	c := NewCatalogRefreshCron(func(_ context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error) {
		drained = append(drained, titleSlug)
		return domain.CatalogUGCDrainResult{}, nil
	}, "", 0).WithCatalogAdapterCheck(catalogPresentForInfinite)
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())

	if len(drained) != 1 || drained[0] != "halo_infinite" {
		t.Fatalf("titres drainés = %v, want [halo_infinite] uniquement (title sans adapter skippé)", drained)
	}
}

// TestCatalogRefreshCron_RunOnce_FallbackCapForge vérifie le fallback rétro-compat :
// sans checker injecté (hasCatalogAdapter nil), le gate retombe sur le proxy CapForge.
// Comportement prod identique (halo_infinite a CapForge → drainé ; title_no_forge ne
// l'a pas → skippé).
func TestCatalogRefreshCron_RunOnce_FallbackCapForge(t *testing.T) {
	var drained []string
	c := NewCatalogRefreshCron(func(_ context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error) {
		drained = append(drained, titleSlug)
		return domain.CatalogUGCDrainResult{}, nil
	}, "", 0) // pas de WithCatalogAdapterCheck → fallback CapForge
	c.registry = forgeRegistry()

	c.RunOnce(context.Background())

	if len(drained) != 1 || drained[0] != "halo_infinite" {
		t.Fatalf("titres drainés (fallback CapForge) = %v, want [halo_infinite] uniquement", drained)
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
