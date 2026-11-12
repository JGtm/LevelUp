package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/platform/dblease"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
)

// withSettings pose un app_settings.json minimal sur le service du fixture.
func (f *syncJobsFixture) withSettings(t *testing.T, canStart bool) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app_settings.json")
	body := `{"can_start_initial_sync":false,"lang":"fr"}`
	if canStart {
		body = `{"can_start_initial_sync":true,"lang":"fr"}`
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	f.svc.settingsStore = settings_platform.NewStore(path)
}

func TestCheckInitialSyncEnabled(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.withSettings(t, false)
	requireSyncJobError(t, f.svc.CheckInitialSyncEnabled(context.Background()), port.SyncErrInitialDisabled)

	f.withSettings(t, true)
	if err := f.svc.CheckInitialSyncEnabled(context.Background()); err != nil {
		t.Fatalf("sync initiale autorisée : %v", err)
	}
}

func TestStartInitialSync_Refus(t *testing.T) {
	cases := []struct {
		name    string
		caller  port.SyncCaller
		req     domain.InitialSyncStartRequest
		arrange func(f *syncJobsFixture)
		kind    port.SyncJobErrorKind
	}{
		{"max_matches trop grand", sessionCaller(), domain.InitialSyncStartRequest{PlayerSlug: "jgtm", MaxMatches: 9999}, nil,
			port.SyncErrInvalidMaxMatches},
		{"max_matches négatif", sessionCaller(), domain.InitialSyncStartRequest{PlayerSlug: "jgtm", MaxMatches: -1}, nil,
			port.SyncErrInvalidMaxMatches},
		{"job déjà actif", sessionCaller(), domain.InitialSyncStartRequest{PlayerSlug: "jgtm", MaxMatches: 100},
			func(f *syncJobsFixture) { f.jobs.Create(domain.JobTypeInitialSync, "jgtm") },
			port.SyncErrInitialAlreadyActive},
		{"sans session", port.SyncCaller{}, domain.InitialSyncStartRequest{PlayerSlug: "jgtm", MaxMatches: 100}, nil,
			port.SyncErrAuthRequired},
		{"session sans jetons Halo", port.SyncCaller{Authenticated: true}, domain.InitialSyncStartRequest{PlayerSlug: "jgtm", MaxMatches: 100}, nil,
			port.SyncErrHaloTokensMissing},
		{"joueur absent du titre", sessionCaller(), domain.InitialSyncStartRequest{PlayerSlug: "jgtm", TitleSlug: "halo_5", MaxMatches: 100}, nil,
			port.SyncErrPlayerNotFoundForTitle},
		{"joueur déjà en vol via une autre source", sessionCaller(), domain.InitialSyncStartRequest{PlayerSlug: "jgtm", MaxMatches: 100},
			func(f *syncJobsFixture) { f.gate.refuseAll = true },
			port.SyncErrAlreadyRunning},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSyncJobsFixture(t)
			if c.arrange != nil {
				c.arrange(f)
			}
			before := f.jobs.count()
			_, err := f.svc.StartInitialSync(context.Background(), c.caller, c.req)
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

// Le titre absent de la requête est celui du contexte ; le refus nomme le joueur et le titre cherchés,
// et la recherche porte sur les profils de CE titre.
func TestStartInitialSync_TitreDuContexte(t *testing.T) {
	f := newSyncJobsFixture(t)
	ctx := ctxkeys.WithTitleSlug(context.Background(), "halo_5")
	_, err := f.svc.StartInitialSync(ctx, sessionCaller(), domain.InitialSyncStartRequest{PlayerSlug: "jgtm", MaxMatches: 10})
	je := requireSyncJobError(t, err, port.SyncErrPlayerNotFoundForTitle)
	if je.PlayerSlug != "jgtm" || je.TitleSlug != "halo_5" {
		t.Errorf("refus %+v : attendu le joueur jgtm et le titre halo_5", je)
	}
	if got := f.profiles.filters; len(got) != 1 || len(got[0]) != 1 || got[0][0] != "halo_5" {
		t.Errorf("filtres de titre %v, attendu [[halo_5]]", got)
	}
}

// Le nombre de matchs : explicite = tel quel ; omis = défaut du profil, borné à 2000 ; omis sans défaut
// de profil = 200.
func TestStartInitialSync_NombreDeMatchs(t *testing.T) {
	cases := []struct {
		name    string
		slug    string
		title   string
		maxReq  int
		profMax int
		want    int
	}{
		{"explicite", "jgtm", "halo_infinite", 50, 300, 50},
		{"défaut du profil", "jgtm", "halo_infinite", 0, 300, 300},
		{"défaut du profil borné", "jgtm", "halo_infinite", 0, 5000, maxInitialSyncMatches},
		{"sans défaut de profil", "h5only", "halo_5", 0, 0, defaultInitialSyncMatches},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSyncJobsFixture(t)
			for i := range f.profiles.players {
				if f.profiles.players[i].PlayerSlug == c.slug {
					f.profiles.players[i].InitialMaxMatches = c.profMax
				}
			}
			job, err := f.svc.StartInitialSync(context.Background(), sessionCaller(),
				domain.InitialSyncStartRequest{PlayerSlug: c.slug, TitleSlug: c.title, MaxMatches: c.maxReq})
			if err != nil {
				t.Fatalf("StartInitialSync: %v", err)
			}
			f.jobs.waitTerminal(t, job.JobID)
			opts := f.runner.options()
			if len(opts) != 1 || opts[0].MaxMatches != c.want {
				t.Errorf("options de sync %+v, attendu MaxMatches=%d", opts, c.want)
			}
		})
	}
}

// Succès : le job rendu est celui de sa création, la réservation est libérée, le résumé est consigné, le
// post-sync encadre la sync et la notification match_synced part.
func TestStartInitialSync_Succes(t *testing.T) {
	f := newSyncJobsFixture(t)
	f.runner.result = domain.SyncResult{MatchesInserted: 3, MatchesSkipped: 1, MedalsInserted: 9, DurationSeconds: 1.5}
	post := &fakePostSync{}
	em := &fakeEmitter{}
	f.svc.WithPostSyncRunner(post).WithNotificationsEmitterFactory(
		func(context.Context, string) (notifications.Emitter, error) { return em, nil })

	job, err := f.svc.StartInitialSync(context.Background(), sessionCaller(),
		domain.InitialSyncStartRequest{PlayerSlug: "jgtm", TitleSlug: "halo_infinite", MaxMatches: 20})
	if err != nil {
		t.Fatalf("StartInitialSync: %v", err)
	}
	if job.Status != domain.JobStatusQueued || job.JobType != string(domain.JobTypeInitialSync) {
		t.Errorf("le job rendu doit être celui de sa création, got %+v", job)
	}
	done := f.jobs.waitTerminal(t, job.JobID)
	if done.Status != domain.JobStatusSucceeded {
		t.Fatalf("statut %q, attendu succeeded", done.Status)
	}
	if done.CurrentStep == nil || !strings.HasPrefix(*done.CurrentStep, "inserted=3 skipped=1 medals=9") {
		t.Errorf("résumé %v", done.CurrentStep)
	}
	waitFor(t, func() bool { return len(post.trace()) == 2 && len(em.emittedInputs()) == 1 })
	if got := post.trace(); got[0] != "before:jgtm" || got[1] != "after:jgtm" {
		t.Errorf("post-sync %v, attendu before puis after", got)
	}
	if got := f.gate.claimed; len(got) != 1 || got[0] != "halo_infinite/JGtm" {
		t.Errorf("réservations %v, attendu halo_infinite/JGtm", got)
	}
	waitFor(t, func() bool { return f.gate.releasedCount() == 1 })
}

// Échec : la cause ne part ni dans le job ni dans la notification, seulement un code ; le post-sync n'est
// pas finalisé.
func TestStartInitialSync_Echec(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		wantCode  string
		retryable bool
	}{
		{"échec réel", errors.New("boom interne duckdb"), "sync_failed", false},
		{"bail occupé", dblease.ErrDBLocked, "db_busy", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newSyncJobsFixture(t)
			f.runner.err = c.err
			post := &fakePostSync{}
			em := &fakeEmitter{}
			f.svc.WithPostSyncRunner(post).WithNotificationsEmitterFactory(
				func(context.Context, string) (notifications.Emitter, error) { return em, nil })

			job, err := f.svc.StartInitialSync(context.Background(), sessionCaller(),
				domain.InitialSyncStartRequest{PlayerSlug: "jgtm", TitleSlug: "halo_infinite", MaxMatches: 20})
			if err != nil {
				t.Fatalf("StartInitialSync: %v", err)
			}
			done := f.jobs.waitTerminal(t, job.JobID)
			if done.Status != domain.JobStatusFailed || done.Error == nil ||
				done.Error.Code != c.wantCode || done.Error.Retryable != c.retryable {
				t.Fatalf("job %+v, attendu échec %q (relançable : %v)", done, c.wantCode, c.retryable)
			}
			if strings.Contains(done.Error.Message, "duckdb") {
				t.Errorf("le message du job laisse fuir la cause : %q", done.Error.Message)
			}
			waitFor(t, func() bool { return len(em.coalescedInputs()) == 1 })
			if got := post.trace(); len(got) != 1 {
				t.Errorf("post-sync %v : jamais finalisé sur un échec", got)
			}
		})
	}
}
