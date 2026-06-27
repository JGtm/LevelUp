package livesync

import (
	"testing"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	halo5 "levelup/go-api/internal/games/halo_5"
)

func TestRunnerForTitle_Dispatch(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	// halo_5 = runner live-sync dédié (réseau différé → construction sans I/O).
	if r := RunnerForTitle(halo5.TitleSlug, cfg, "JGtm", "xJG"); r == nil {
		t.Errorf("halo_5 doit avoir un runner live-sync dédié")
	}
	// halo_infinite passe par le SyncEngine → pas de runner ici.
	if r := RunnerForTitle("halo_infinite", cfg, "Madina", "x"); r != nil {
		t.Errorf("halo_infinite → RunnerForTitle doit être nil (SyncEngine)")
	}
	if r := RunnerForTitle("titre_inconnu", cfg, "X", "y"); r != nil {
		t.Errorf("titre inconnu → nil")
	}
}

func TestHandlesTitle(t *testing.T) {
	if !HandlesTitle(halo5.TitleSlug) {
		t.Error("HandlesTitle(halo_5) doit être true")
	}
	if HandlesTitle("halo_infinite") {
		t.Error("HandlesTitle(halo_infinite) doit être false (SyncEngine)")
	}
}

// registerH5WithCaps remplace le registre partagé par un registre où halo_5 est
// déclaré avec (ou sans) la capability achievements, puis restaure le built-in.
func registerH5WithCaps(t *testing.T, caps ...titlePkg.Capability) {
	t.Helper()
	reg := titlePkg.NewRegistry()
	reg.Register(&titlePkg.TitleDescriptor{
		Slug:         halo5.TitleSlug,
		Name:         "Halo 5",
		Provider:     halo5.TitleSlug,
		Status:       titlePkg.StatusActive,
		Capabilities: caps,
	})
	titlePkg.SetDefaultRegistry(reg)
	t.Cleanup(func() { titlePkg.SetDefaultRegistry(titlePkg.NewRegistry()) })
}

// TestBuildAchievementsHook_GatedOnCapability : le hook achievements n'est CONSTRUIT
// (non-nil) que si le titre déclare CapAchievements (gate title-agnostic au câblage,
// jamais slug==literal). Absente → nil → le runner ne tente jamais de fetch Xbox.
func TestBuildAchievementsHook_GatedOnCapability(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}

	// (1) Capability présente → hook construit.
	registerH5WithCaps(t, titlePkg.CapAchievements)
	if h := buildAchievementsHook(cfg, "JGtm", "xJG"); h == nil {
		t.Error("CapAchievements déclarée → hook achievements doit être non-nil")
	}

	// (2) Capability absente → hook nil (jamais appelé).
	registerH5WithCaps(t /* aucune capability */)
	if h := buildAchievementsHook(cfg, "JGtm", "xJG"); h != nil {
		t.Error("CapAchievements absente → hook achievements doit être nil")
	}

	// (3) Titre inconnu du registre → hook nil (garde défensive, pas de panic).
	titlePkg.SetDefaultRegistry(titlePkg.NewRegistry()) // built-in : halo_5 absent
	if h := buildAchievementsHook(cfg, "JGtm", "xJG"); h != nil {
		t.Error("titre absent du registre → hook achievements doit être nil")
	}
}

// TestNewHalo5Runner_WiresConvergenceProbe : le runner câblé expose bien la sonde de
// backlog d'enrichment (filet de convergence à 0 insert) — non-nil par construction.
func TestNewHalo5Runner_WiresConvergenceProbe(t *testing.T) {
	cfg := &config.AppConfig{RepoRoot: t.TempDir()}
	r := newHalo5Runner(cfg, "JGtm", "xJG")
	if r == nil {
		t.Fatal("newHalo5Runner doit construire un runner")
	}
	if r.deps.HasEnrichmentBacklog == nil {
		t.Error("le runner h5 doit câbler HasEnrichmentBacklog (filet de convergence à 0 insert)")
	}
}
