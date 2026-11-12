package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
	go_sync "levelup/go-api/internal/sync"
)

// TestTryManualSyncCooldown : 1er déclenchement passe, 2e dans la fenêtre bloqué (retry>0), clé
// différente indépendante.
func TestTryManualSyncCooldown(t *testing.T) {
	s := NewSyncJobService(&config.AppConfig{}, nil, nil, nil).WithManualSyncCooldown(time.Hour)

	if retry, ok := s.tryManualSyncCooldown("k"); !ok || retry != 0 {
		t.Fatalf("1er appel doit passer sans retry, got retry=%v ok=%v", retry, ok)
	}
	retry, ok := s.tryManualSyncCooldown("k")
	if ok || retry <= 0 {
		t.Fatalf("2e appel doit être bloqué avec retry>0, got retry=%v ok=%v", retry, ok)
	}
	if retry > time.Hour {
		t.Fatalf("retry borné par le cooldown (1h), got %v", retry)
	}
	if _, ok := s.tryManualSyncCooldown("autre"); !ok {
		t.Fatal("une clé différente doit passer indépendamment")
	}
}

// TestTryManualSyncCooldown_Disabled : cooldown=0 → toujours autorisé.
func TestTryManualSyncCooldown_Disabled(t *testing.T) {
	s := NewSyncJobService(&config.AppConfig{}, nil, nil, nil).WithManualSyncCooldown(0)
	for i := 0; i < 3; i++ {
		if retry, ok := s.tryManualSyncCooldown("k"); !ok || retry != 0 {
			t.Fatalf("cooldown désactivé doit toujours passer (appel %d), got retry=%v ok=%v", i, retry, ok)
		}
	}
}

// Le sync manuel exige une session ; le refus tombe avant tout autre garde-fou.
func TestStartDeltaSync_SansSession(t *testing.T) {
	f := newSyncJobsFixture(t)
	_, err := f.svc.StartDeltaSync(context.Background(), port.SyncCaller{}, "jgtm")
	requireSyncJobError(t, err, port.SyncErrAuthRequired)
	_, err = f.svc.StartSyncAll(context.Background(), port.SyncCaller{})
	requireSyncJobError(t, err, port.SyncErrAuthRequired)
}

// Une session sans jetons Halo passe : l'authentification vient du pool.
func TestStartSyncAll_SessionSansJetonsHalo(t *testing.T) {
	f := newSyncJobsFixture(t)
	job, err := f.svc.StartSyncAll(context.Background(), port.SyncCaller{Authenticated: true})
	if err != nil {
		t.Fatalf("StartSyncAll: %v", err)
	}
	f.jobs.waitTerminal(t, job.JobID)
}

// Le cooldown couvre la TENTATIVE : un 2e appel dans la fenêtre est refusé même si le 1er l'a été par le
// gate. Le refus porte le délai restant.
func TestStartDeltaSync_Cooldown(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.svc.WithManualSyncCooldown(time.Hour)
	f.gate.refuseAll = true

	_, err := f.svc.StartDeltaSync(context.Background(), sessionCaller(), "jgtm")
	requireSyncJobError(t, err, port.SyncErrAlreadyRunning)

	_, err = f.svc.StartDeltaSync(context.Background(), sessionCaller(), "jgtm")
	je := requireSyncJobError(t, err, port.SyncErrCooldown)
	if je.RetryAfter <= 0 || je.RetryAfter > time.Hour {
		t.Errorf("RetryAfter %v, attendu dans ]0, 1h]", je.RetryAfter)
	}
}

func TestStartDeltaSync_Refus(t *testing.T) {
	cases := []struct {
		name    string
		slug    string
		arrange func(f *syncJobsFixture)
		kind    port.SyncJobErrorKind
	}{
		{"job déjà actif", "jgtm", func(f *syncJobsFixture) { f.jobs.Create(domain.JobTypeInitialSync, "jgtm") },
			port.SyncErrDeltaAlreadyActive},
		{"joueur inconnu", "inconnu", nil, port.SyncErrPlayerNotFound},
		{"joueur déjà en vol via une autre source", "jgtm", func(f *syncJobsFixture) { f.gate.refuseAll = true },
			port.SyncErrAlreadyRunning},
		{"profils illisibles", "jgtm", func(f *syncJobsFixture) { f.profiles.err = errors.New("boom") },
			port.SyncErrProfilesLoad},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSyncJobsFixture(t)
			if c.arrange != nil {
				c.arrange(f)
			}
			before := f.jobs.count()
			_, err := f.svc.StartDeltaSync(context.Background(), sessionCaller(), c.slug)
			requireSyncJobError(t, err, c.kind)
			if f.jobs.count() != before {
				t.Error("un refus ne doit créer aucun job")
			}
			if len(f.runner.options()) != 0 {
				t.Error("un refus ne doit lancer aucune sync")
			}
		})
	}
}

// Le profil se cherche parmi TOUS les titres.
func TestStartDeltaSync_Succes(t *testing.T) {
	f := newSyncJobsFixture(t)
	post := &fakePostSync{}
	f.svc.WithPostSyncRunner(post)
	f.runner.result = domain.SyncResult{MatchesInserted: 2}

	job, err := f.svc.StartDeltaSync(context.Background(), sessionCaller(), "h5only")
	if err != nil {
		t.Fatalf("StartDeltaSync: %v", err)
	}
	done := f.jobs.waitTerminal(t, job.JobID)
	if done.Status != domain.JobStatusSucceeded || !strings.HasPrefix(*done.CurrentStep, "inserted=2") {
		t.Fatalf("job %+v", done)
	}
	if got := f.profiles.filters[0]; len(got) != 0 {
		t.Errorf("la recherche du profil porte sur tous les titres, filtre reçu %v", got)
	}
	waitFor(t, func() bool { return len(post.trace()) == 2 })
	waitFor(t, func() bool { return f.gate.releasedCount() == 1 })
	if got := f.runner.options(); len(got) != 1 || got[0].MaxMatches != domain.DefaultSyncOptions().MaxMatches {
		t.Errorf("options %+v : attendu les options par défaut", got)
	}
}

// Sync de tous : tous coalescés = succès (aucun échec), les comptes figurent au résumé.
func TestStartSyncAll_ToutCoalesce(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.gate.refuseAll = true
	job, err := f.svc.StartSyncAll(context.Background(), sessionCaller())
	if err != nil {
		t.Fatalf("StartSyncAll: %v", err)
	}
	done := f.jobs.waitTerminal(t, job.JobID)
	if done.Status != domain.JobStatusSucceeded || !strings.Contains(*done.CurrentStep, "players=2 succeeded=0 failed=0 coalesced=2") {
		t.Fatalf("job %+v résumé %v", done, done.CurrentStep)
	}
	if len(f.runner.options()) != 0 {
		t.Error("aucun joueur coalescé ne doit être synchronisé")
	}
}

// Sync de tous : un échec fait échouer le job sans interrompre les autres joueurs ; un joueur déjà en
// vol est sauté.
func TestStartSyncAll_EchecEtCoalescence(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.gate.refuse = map[string]bool{"H5Only": true}
	f.runner.err = errors.New("boom")
	job, err := f.svc.StartSyncAll(context.Background(), sessionCaller())
	if err != nil {
		t.Fatalf("StartSyncAll: %v", err)
	}
	done := f.jobs.waitTerminal(t, job.JobID)
	if done.Status != domain.JobStatusFailed || !strings.Contains(*done.CurrentStep, "players=2 succeeded=0 failed=1 coalesced=1") {
		t.Fatalf("job %+v résumé %v", done, done.CurrentStep)
	}
	if done.ProgressPct == nil {
		t.Error("la progression doit être renseignée")
	}
	if f.gate.releasedCount() != 1 {
		t.Errorf("la réservation du joueur synchronisé doit être libérée, libérations : %d", f.gate.releasedCount())
	}
}

func TestStartSyncAll_Refus(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.profiles.players = nil
	_, err := f.svc.StartSyncAll(context.Background(), sessionCaller())
	requireSyncJobError(t, err, port.SyncErrNoPlayers)

	f.profiles.err = errors.New("boom")
	_, err = f.svc.StartSyncAll(context.Background(), sessionCaller())
	requireSyncJobError(t, err, port.SyncErrProfilesLoad)
}

// L'arrêt du serveur interrompt la sync de tous avant le joueur suivant.
func TestStartSyncAll_ArretDuServeur(t *testing.T) {
	f := newSyncJobsFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f.svc.WithServerContext(ctx)
	job, err := f.svc.StartSyncAll(context.Background(), sessionCaller())
	if err != nil {
		t.Fatalf("StartSyncAll: %v", err)
	}
	done := f.jobs.waitTerminal(t, job.JobID)
	if !strings.Contains(*done.CurrentStep, "succeeded=0 failed=0 coalesced=0") || len(f.runner.options()) != 0 {
		t.Errorf("aucun joueur ne devait démarrer : %v", done.CurrentStep)
	}
}

// TestNewPooledEngine_UsesInjectedBuilder : avec un EngineBuilder injecté, newPooledEngine délègue au
// builder (même mécanique que l'auto-sync) et lui transmet (gamertag, xuid) — il ne reconstruit PAS un
// moteur de sync initiale.
func TestNewPooledEngine_UsesInjectedBuilder(t *testing.T) {
	sentinel := &go_sync.SyncEngine{} // jamais exécuté : on teste la sélection, pas RunDelta
	var gotGT, gotXUID string
	s := NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil).
		WithEngineBuilder(func(_ context.Context, gt, xuid string) *go_sync.SyncEngine {
			gotGT, gotXUID = gt, xuid
			return sentinel
		})

	if got := s.newPooledEngine(context.Background(), "DemoPlayer", "xuid-1"); got != sentinel {
		t.Fatal("newPooledEngine doit retourner le moteur construit par le builder injecté")
	}
	if gotGT != "DemoPlayer" || gotXUID != "xuid-1" {
		t.Fatalf("(gamertag, xuid) non transmis au builder: gt=%q xuid=%q", gotGT, gotXUID)
	}
}

// TestNewPooledEngine_FallbackWhenNoBuilder : sans builder (tests / bootstrap sans pool) → repli sur le
// moteur de la sync initiale, non-nil, sans panique (settingsStore nil toléré).
func TestNewPooledEngine_FallbackWhenNoBuilder(t *testing.T) {
	s := NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil)
	if got := s.newPooledEngine(context.Background(), "DemoPlayer", "xuid-1"); got == nil {
		t.Fatal("le repli doit retourner un moteur non-nil")
	}
}

// TestNewEngineFor_WiresPrestigeHook : garde anti-régression. Quand le service a reçu un hook Prestige,
// le SyncEngine construit pour la sync initiale doit le porter ; sans hook, il n'en porte pas.
func TestNewEngineFor_WiresPrestigeHook(t *testing.T) {
	s := NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir()}, nil, nil, nil)
	if s.newEngineFor("halo_infinite", "DemoPlayer", "xuid-1", nil).HasPrestigeHook() {
		t.Error("hook Prestige câblé alors que WithPrestigeHook n'a pas été appelé")
	}
	s.WithPrestigeHook(func(context.Context, string, string) {})
	if !s.newEngineFor("halo_infinite", "DemoPlayer", "xuid-1", nil).HasPrestigeHook() {
		t.Fatal("le moteur de la sync initiale ne porte pas le hook Prestige : il ne tournera jamais sur ce chemin")
	}
}

// TestNewEngineFor_SharedProviderIsTitleScoped : le titre par défaut reçoit le provider du boot ; un
// autre titre n'en reçoit JAMAIS le provider du boot, et reste sans provider quand aucun Manager ne peut
// le servir.
func TestNewEngineFor_SharedProviderIsTitleScoped(t *testing.T) {
	boot := sharedprovider.FromInMemoryDB((*sql.DB)(nil), ":memory:")
	s := NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir(), SharedProvider: boot}, nil, nil, nil)

	for _, slug := range []string{"", title.DefaultSlug} {
		if got := s.newEngineFor(slug, "DemoPlayer", "xuid-1", nil).SharedProviderForTest(); got != boot {
			t.Errorf("titre %q : attendu le provider du boot, got %v", slug, got)
		}
	}

	eOther := s.newEngineFor("synthetic_test_title", "DemoPlayer", "xuid-1", nil)
	if got := eOther.SharedProviderForTest(); got == boot {
		t.Fatal("autre titre : le provider du boot (titre par défaut) a été câblé sur le moteur")
	}
	if eOther.HasSharedProvider() {
		t.Error("autre titre sans Manager : le moteur doit rester sans provider (mode hérité)")
	}
}
