package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/port"
	go_sync "levelup/go-api/internal/sync"
)

// fakeSyncJobStore est un registre de jobs en mémoire (SyncJobStore).
type fakeSyncJobStore struct {
	mu   sync.Mutex
	jobs map[string]*domain.AsyncJobStatus
	seq  int
}

func newFakeSyncJobStore() *fakeSyncJobStore {
	return &fakeSyncJobStore{jobs: map[string]*domain.AsyncJobStatus{}}
}

func (f *fakeSyncJobStore) Create(jobType domain.JobType, playerSlug string) *domain.AsyncJobStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := fmt.Sprintf("job-%d", f.seq)
	f.jobs[id] = &domain.AsyncJobStatus{JobID: id, JobType: string(jobType), Status: domain.JobStatusQueued, PlayerSlug: playerSlug}
	cp := *f.jobs[id]
	return &cp
}

func (f *fakeSyncJobStore) FindActiveJob(jobType domain.JobType, playerSlug string) *domain.AsyncJobStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, j := range f.jobs {
		if j.JobType == string(jobType) && j.PlayerSlug == playerSlug && !j.IsTerminal() {
			cp := *j
			return &cp
		}
	}
	return nil
}

func (f *fakeSyncJobStore) FindActiveInitialSync(playerSlug string) *domain.AsyncJobStatus {
	return f.FindActiveJob(domain.JobTypeInitialSync, playerSlug)
}

func (f *fakeSyncJobStore) Update(jobID string, fn func(*domain.AsyncJobStatus)) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	j, ok := f.jobs[jobID]
	if ok {
		fn(j)
	}
	return ok
}

func (f *fakeSyncJobStore) SetStatus(jobID string, status domain.JobStatus, step *string) bool {
	return f.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.Status = status
		if step != nil {
			j.CurrentStep = step
		}
	})
}

// get rend une copie du job, ou nil.
func (f *fakeSyncJobStore) get(jobID string) *domain.AsyncJobStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	if j, ok := f.jobs[jobID]; ok {
		cp := *j
		return &cp
	}
	return nil
}

// count rend le nombre de jobs créés.
func (f *fakeSyncJobStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.jobs)
}

// waitTerminal attend la fin du job et le rend.
func (f *fakeSyncJobStore) waitTerminal(t *testing.T, jobID string) *domain.AsyncJobStatus {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if j := f.get(jobID); j != nil && j.IsTerminal() {
			return j
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("job %s non terminal après 5 s", jobID)
	return nil
}

// fakeSyncProfiles est une source de profils en mémoire (SyncProfiles) ; elle garde les filtres de titre reçus.
type fakeSyncProfiles struct {
	players []domain.PlayerSummary
	err     error
	filters [][]string
}

func (f *fakeSyncProfiles) LoadPlayers(titleFilter ...string) ([]domain.PlayerSummary, error) {
	f.filters = append(f.filters, titleFilter)
	if f.err != nil {
		return nil, f.err
	}
	var out []domain.PlayerSummary
	for _, p := range f.players {
		if len(titleFilter) == 0 || titleFilter[0] == "" || p.TitleSlug == titleFilter[0] {
			out = append(out, p)
		}
	}
	return out, nil
}

// fakeSyncGate est un go_sync.SyncGate contrôlable : refuse les joueurs de `refuse` (tous si refuseAll),
// garde les clés réservées et compte les libérations.
type fakeSyncGate struct {
	mu        sync.Mutex
	refuseAll bool
	refuse    map[string]bool
	claimed   []string
	released  int
}

func (g *fakeSyncGate) TryClaim(gamertag string) (func(), bool) { return g.TryClaimT("", gamertag) }

func (g *fakeSyncGate) TryClaimT(titleSlug, gamertag string) (func(), bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.refuseAll || g.refuse[gamertag] {
		return nil, false
	}
	g.claimed = append(g.claimed, titleSlug+"/"+gamertag)
	return func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.released++
	}, true
}

func (g *fakeSyncGate) IsInFlight(string) bool                 { return g.refuseAll }
func (g *fakeSyncGate) IsInFlightT(string, string) bool        { return g.refuseAll }
func (g *fakeSyncGate) WaitInFlight()                          {}
func (g *fakeSyncGate) BeginShutdown()                         {}
func (g *fakeSyncGate) GateSnapshot() go_sync.GateSnapshotData { return go_sync.GateSnapshotData{} }

func (g *fakeSyncGate) releasedCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.released
}

// fakeDeltaRunner est un DeltaRunner qui rend un résultat fixé et garde les options reçues.
type fakeDeltaRunner struct {
	result domain.SyncResult
	err    error

	mu      sync.Mutex
	gotOpts []domain.SyncOptions
}

func (r *fakeDeltaRunner) RunDelta(_ context.Context, opts domain.SyncOptions) (domain.SyncResult, error) {
	r.mu.Lock()
	r.gotOpts = append(r.gotOpts, opts)
	r.mu.Unlock()
	return r.result, r.err
}

func (r *fakeDeltaRunner) options() []domain.SyncOptions {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domain.SyncOptions(nil), r.gotOpts...)
}

// fakeEmitter enregistre les notifications émises.
type fakeEmitter struct {
	mu        sync.Mutex
	emitted   []notifications.EmitInput
	coalesced []notifications.EmitInput
}

func (e *fakeEmitter) Emit(_ context.Context, in notifications.EmitInput) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.emitted = append(e.emitted, in)
	return nil
}

func (e *fakeEmitter) EmitCoalesced(_ context.Context, in notifications.EmitInput, _ time.Duration) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.coalesced = append(e.coalesced, in)
	return nil
}

func (e *fakeEmitter) emittedInputs() []notifications.EmitInput {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]notifications.EmitInput(nil), e.emitted...)
}

func (e *fakeEmitter) coalescedInputs() []notifications.EmitInput {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]notifications.EmitInput(nil), e.coalesced...)
}

// waitFor attend que cond devienne vraie (5 s au plus).
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition non atteinte après 5 s")
}

// fakePostSync est un port.PostSyncRunner qui trace l'ordre capture / finalisation.
type fakePostSync struct {
	mu     sync.Mutex
	events []string
}

func (p *fakePostSync) BeforeSync(_ context.Context, slug string) port.PostSyncFinalizer {
	p.record("before:" + slug)
	return func(context.Context) { p.record("after:" + slug) }
}

func (p *fakePostSync) record(e string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
}

func (p *fakePostSync) trace() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.events...)
}

// syncJobsFixture assemble un service entièrement en mémoire : profils, jobs, gate et runners
// factices. Les runners rendent runner ; la construction réelle d'un moteur n'a jamais lieu.
type syncJobsFixture struct {
	svc      *SyncJobService
	jobs     *fakeSyncJobStore
	profiles *fakeSyncProfiles
	gate     *fakeSyncGate
	runner   *fakeDeltaRunner
}

func newSyncJobsFixture(t *testing.T) *syncJobsFixture {
	t.Helper()
	f := &syncJobsFixture{
		jobs: newFakeSyncJobStore(),
		profiles: &fakeSyncProfiles{players: []domain.PlayerSummary{
			{PlayerSlug: "jgtm", Gamertag: "JGtm", XUID: "1001", TitleSlug: "halo_infinite", InitialMaxMatches: 300},
			{PlayerSlug: "h5only", Gamertag: "H5Only", XUID: "1002", TitleSlug: "halo_5"},
		}},
		gate:   &fakeSyncGate{},
		runner: &fakeDeltaRunner{},
	}
	f.svc = NewSyncJobService(&config.AppConfig{RepoRoot: t.TempDir()}, nil, f.jobs, nil).
		WithProfiles(f.profiles).
		WithSyncGate(f.gate).
		WithManualSyncCooldown(0)
	f.svc.runnerFor = func(ctx context.Context, _, _, _ string, _ *domain.HaloTokens) (DeltaRunner, context.Context) {
		return f.runner, ctx
	}
	f.svc.pooledRunnerFor = func(context.Context, string, string) DeltaRunner { return f.runner }
	return f
}

// requireSyncJobError échoue sauf si err est un *port.SyncJobError de la nature voulue, et le rend.
func requireSyncJobError(t *testing.T, err error, kind port.SyncJobErrorKind) *port.SyncJobError {
	t.Helper()
	var je *port.SyncJobError
	if !errors.As(err, &je) || je.Kind != kind {
		t.Fatalf("attendu un refus %q, obtenu %v", kind, err)
	}
	return je
}

// sessionCaller : un appelant connecté, avec jetons Halo.
func sessionCaller() port.SyncCaller {
	return port.SyncCaller{Authenticated: true, HaloTokens: &domain.HaloTokens{SpartanToken: "spartan-test"}}
}
