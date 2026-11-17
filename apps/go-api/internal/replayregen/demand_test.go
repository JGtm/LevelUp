package replayregen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/halo-infinite-film-decoder/finalise"
)

// memDemandStore : le magasin des demandes, en mémoire ; saves compte les écritures.
type memDemandStore struct {
	mu      sync.Mutex
	demands []domain.ReplayGenerationDemand
	saves   int
	loadErr error
}

func (m *memDemandStore) LoadDemands(context.Context) ([]domain.ReplayGenerationDemand, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.ReplayGenerationDemand(nil), m.demands...), m.loadErr
}

func (m *memDemandStore) SaveDemands(_ context.Context, d []domain.ReplayGenerationDemand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.demands = append([]domain.ReplayGenerationDemand(nil), d...)
	m.saves++
	return nil
}

func (m *memDemandStore) snapshot() []domain.ReplayGenerationDemand {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.ReplayGenerationDemand(nil), m.demands...)
}

// demandFixture : une file reliée avec un faux téléchargement de film et un magasin de demandes en mémoire.
type demandFixture struct {
	*fixture
	store    *memDemandStore
	fetchN   atomic.Int32
	releaseN atomic.Int32
	mu       sync.Mutex
	order    []string // identifiants courts, dans l'ordre des reconstructions
	fetchErr func(d domain.ReplayGenerationDemand) error
	released []string
}

// newDemandFixture relie un service neuf sur la fixture du lot, avec FetchFilm, ReleaseFilm et un magasin.
func newDemandFixture(t *testing.T, maxDemands int) *demandFixture {
	t.Helper()
	f := &demandFixture{fixture: newFixture(t), store: &memDemandStore{}}
	f.cancel()
	f.wg.Wait()
	baseRebuild := f.deps.Rebuild
	f.deps.Rebuild = func(ctx context.Context, slug, id string) error {
		f.mu.Lock()
		f.order = append(f.order, id[:8])
		f.mu.Unlock()
		return baseRebuild(ctx, slug, id)
	}
	f.deps.FetchFilm = func(_ context.Context, d domain.ReplayGenerationDemand) error {
		n := f.inFly.Add(1)
		defer f.inFly.Add(-1)
		for {
			m := f.maxFly.Load()
			if n <= m || f.maxFly.CompareAndSwap(m, n) {
				break
			}
		}
		f.fetchN.Add(1)
		time.Sleep(3 * time.Millisecond)
		if f.fetchErr != nil {
			return f.fetchErr(d)
		}
		return nil
	}
	f.deps.ReleaseFilm = func(_ context.Context, _, id string) {
		f.releaseN.Add(1)
		f.mu.Lock()
		f.released = append(f.released, id)
		f.mu.Unlock()
	}
	f.deps.Demands = f.store
	f.deps.MaxDemands = maxDemands
	f.rebind(t)
	return f
}

// rebind relie un service neuf aux dépendances courantes de la fixture.
func (f *demandFixture) rebind(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.svc = New()
	f.svc.Bind(ctx, &f.wg, f.deps)
	t.Cleanup(func() { cancel(); f.wg.Wait() })
}

func (f *demandFixture) orderSnapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.order...)
}

// demandFor : la demande du match numéro n (identifiant complet, forme courte `%08x`).
func demandFor(n int) domain.ReplayGenerationDemand {
	return domain.ReplayGenerationDemand{
		TitleSlug: testSlug, MatchID: fmt.Sprintf("%08x-0000-0000-0000-000000000001", 0xd00+n),
		PlayerSlug: "joueur", XUID: "2533274800000001", Gamertag: "Joueur", RequestedAt: time.Unix(1700000000+int64(n), 0).UTC(),
	}
}

func shortOf(n int) string { return fmt.Sprintf("%08x", 0xd00+n) }

func (f *demandFixture) status(n int) (domain.ReplayGenerationStatus, bool) {
	d := demandFor(n)
	return f.svc.DemandStatus(d.TitleSlug, d.MatchID)
}

func (f *demandFixture) waitFailed(t *testing.T, n int) domain.ReplayGenerationStatus {
	t.Helper()
	var st domain.ReplayGenerationStatus
	waitFor(t, fmt.Sprintf("demande %d en échec", n), func() bool {
		var ok bool
		st, ok = f.status(n)
		return ok && st.State != domain.ReplayGenerationQueued && st.State != domain.ReplayGenerationRunning
	})
	return st
}

// waitSettled attend que la demande n soit sortie de la file (rangée en `ready`, qui ne se garde pas).
func (f *demandFixture) waitSettled(t *testing.T, n int) {
	t.Helper()
	waitFor(t, fmt.Sprintf("demande %d terminée", n), func() bool { _, ok := f.status(n); return !ok })
}

// captureLogs redirige slog vers un tampon le temps du test.
func captureLogs(t *testing.T) *lockedBuffer {
	t.Helper()
	buf := &lockedBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// (1) un doublon rend l'état de la demande en vol, sans seconde entrée ni seconde reconstruction.
func TestDemand_DoublonIdempotent(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.heavy.Store(false) // la demande reste en attente
	st, created, err := f.svc.Enqueue(demandFor(1))
	if err != nil || !created || st.State != domain.ReplayGenerationQueued || st.Position != 1 {
		t.Fatalf("première demande : st=%+v created=%v err=%v", st, created, err)
	}
	st2, created2, err := f.svc.Enqueue(demandFor(1))
	if err != nil || created2 || st2.State != domain.ReplayGenerationQueued {
		t.Fatalf("doublon : st=%+v created=%v err=%v", st2, created2, err)
	}
	// la forme courte de l'identifiant désigne le même match
	short := demandFor(1)
	short.MatchID = shortOf(1)
	if _, created3, _ := f.svc.Enqueue(short); created3 {
		t.Fatal("la forme courte doit désigner la même demande")
	}
	if n := len(f.store.snapshot()); n != 1 {
		t.Fatalf("magasin = %d demandes, attendu 1", n)
	}
	f.heavy.Store(true)
	f.waitSettled(t, 1)
	if got := f.calls.Load(); got != 1 {
		t.Fatalf("Rebuild appelé %d fois, attendu 1", got)
	}
}

// (2) la onzième demande est refusée, rien n'est écrit pour elle.
func TestDemand_PlafondRefuseEtNEcritRien(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.heavy.Store(false)
	for i := 1; i <= DefaultMaxDemands; i++ {
		if _, _, err := f.svc.Enqueue(demandFor(i)); err != nil {
			t.Fatalf("demande %d : %v", i, err)
		}
	}
	savesAvant := f.store.saves
	_, created, err := f.svc.Enqueue(demandFor(DefaultMaxDemands + 1))
	if !errors.Is(err, domain.ErrReplayGenerationQueueFull) || created {
		t.Fatalf("onzième demande : created=%v err=%v, attendu ErrReplayGenerationQueueFull", created, err)
	}
	if f.store.saves != savesAvant || len(f.store.snapshot()) != DefaultMaxDemands {
		t.Fatalf("le refus a écrit au magasin (saves %d -> %d, %d demandes)", savesAvant, f.store.saves, len(f.store.snapshot()))
	}
	if _, ok := f.status(DefaultMaxDemands + 1); ok {
		t.Fatal("la demande refusée ne doit avoir aucun état")
	}
}

// Enqueue sur un service non relié.
func TestDemand_NonReliee(t *testing.T) {
	_, _, err := New().Enqueue(demandFor(1))
	if !errors.Is(err, domain.ErrReplayGenerationUnbound) {
		t.Fatalf("err = %v, attendu ErrReplayGenerationUnbound", err)
	}
}

// (3) un film à la fois : lot et demandes mêlés, jamais deux téléchargements ou reconstructions en parallèle.
func TestDemand_UnFilmALaFoisLotEtDemandesMeles(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.seed(t, 4)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 4; i++ {
		if _, _, err := f.svc.Enqueue(demandFor(i)); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Millisecond)
	}
	for i := 1; i <= 4; i++ {
		f.waitSettled(t, i)
	}
	waitFor(t, "lot terminé", func() bool { return !f.svc.Status().Running })
	if m := f.maxFly.Load(); m > 1 {
		t.Fatalf("%d films en parallèle, attendu 1 au plus", m)
	}
	if got := f.calls.Load(); got != 8 {
		t.Fatalf("Rebuild appelé %d fois, attendu 8 (4 du lot, 4 demandes)", got)
	}
}

// (4) une demande arrivée pendant le premier film du lot passe avant le deuxième candidat.
func TestDemand_PasseAvantLeCandidatSuivantDuLot(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.seed(t, 3)
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	var first sync.Once
	prev := f.deps.Rebuild
	f.deps.Rebuild = func(ctx context.Context, slug, id string) error {
		first.Do(func() { started <- struct{}{}; <-release })
		return prev(ctx, slug, id)
	}
	f.rebind(t)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	<-started
	if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
		t.Fatal(err)
	}
	close(release)
	waitFor(t, "quatre reconstructions", func() bool { return f.calls.Load() == 4 })
	order := f.orderSnapshot()
	if len(order) != 4 || order[1] != shortOf(1) {
		t.Fatalf("ordre des reconstructions = %v, la demande (%s) devait passer en deuxième", order, shortOf(1))
	}
}

// (5) la porte du mode jeu et la synchronisation retiennent la demande, signalées dans l'état.
func TestDemand_PortesModeJeuEtSynchronisation(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.heavy.Store(false)
	if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "attente de la partie signalée", func() bool {
		st, _ := f.status(1)
		return st.State == domain.ReplayGenerationQueued && st.WaitingForGame
	})
	if f.fetchN.Load() != 0 || f.calls.Load() != 0 {
		t.Fatal("rien ne doit partir pendant une partie")
	}
	f.syncing.Store(true)
	f.heavy.Store(true)
	time.Sleep(60 * time.Millisecond)
	if f.fetchN.Load() != 0 || f.calls.Load() != 0 {
		t.Fatal("rien ne doit partir pendant un cycle de synchronisation")
	}
	f.syncing.Store(false)
	f.waitSettled(t, 1)
	if f.fetchN.Load() != 1 || f.calls.Load() != 1 {
		t.Fatalf("fetch=%d rebuild=%d, attendu 1 et 1", f.fetchN.Load(), f.calls.Load())
	}
}

// (6) un lot en pause retient aussi les demandes ; sa reprise les relâche.
func TestDemand_LotEnPauseRetientLesDemandes(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.seed(t, 2)
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	var first sync.Once
	prev := f.deps.Rebuild
	f.deps.Rebuild = func(ctx context.Context, slug, id string) error {
		first.Do(func() { started <- struct{}{}; <-release })
		return prev(ctx, slug, id)
	}
	f.rebind(t)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	<-started
	if _, err := f.svc.Pause(); err != nil {
		t.Fatal(err)
	}
	st, _, err := f.svc.Enqueue(demandFor(1))
	if err != nil || st.State != domain.ReplayGenerationQueued || !st.Paused {
		t.Fatalf("demande pendant la pause : st=%+v err=%v", st, err)
	}
	close(release)
	time.Sleep(80 * time.Millisecond)
	if f.fetchN.Load() != 0 {
		t.Fatal("la demande ne doit pas partir pendant la pause du lot")
	}
	if _, err := f.svc.Resume(); err != nil {
		t.Fatal(err)
	}
	f.waitSettled(t, 1)
}

// (7) l'ajout écrit le fichier, la sortie le réécrit, la reprise remet les demandes dans l'ordre.
func TestDemand_PersistanceEtReprise(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.heavy.Store(false)
	for i := 1; i <= 2; i++ {
		if _, _, err := f.svc.Enqueue(demandFor(i)); err != nil {
			t.Fatal(err)
		}
	}
	if got := f.store.snapshot(); len(got) != 2 || got[0].MatchID != demandFor(1).MatchID || got[1].MatchID != demandFor(2).MatchID {
		t.Fatalf("magasin après ajout = %+v", got)
	}
	// « redémarrage » : un service neuf, le magasin garde les demandes
	f.cancel()
	f.wg.Wait()
	f.heavy.Store(true)
	f.rebind(t)
	f.svc.ResumeDemands()
	f.waitSettled(t, 1)
	f.waitSettled(t, 2)
	if order := f.orderSnapshot(); len(order) != 2 || order[0] != shortOf(1) || order[1] != shortOf(2) {
		t.Fatalf("ordre de reprise = %v", order)
	}
	waitFor(t, "magasin vidé", func() bool { return len(f.store.snapshot()) == 0 })
}

// (7 bis) une reprise écarte une demande incomplète et réécrit le fichier.
func TestDemand_RepriseEcarteLesDemandesIncompletes(t *testing.T) {
	f := newDemandFixture(t, 0)
	logs := captureLogs(t)
	bad := demandFor(9)
	bad.MatchID = ""
	f.store.demands = []domain.ReplayGenerationDemand{bad}
	f.svc.ResumeDemands()
	if _, ok := f.svc.DemandStatus(bad.TitleSlug, demandFor(9).MatchID); ok {
		t.Fatal("une demande sans match ne doit pas reprendre")
	}
	if !strings.Contains(logs.String(), "incomplète") {
		t.Fatalf("la demande écartée doit se journaliser : %s", logs.String())
	}
	if len(f.store.snapshot()) != 0 {
		t.Fatal("le fichier doit être réécrit sans la demande écartée")
	}
}

// (7 ter) un fichier illisible : journal Warn et file vide.
func TestDemand_RepriseFichierIllisible(t *testing.T) {
	f := newDemandFixture(t, 0)
	logs := captureLogs(t)
	f.store.loadErr = errors.New("json cassé")
	f.svc.ResumeDemands()
	if !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "illisibles") {
		t.Fatalf("un fichier illisible doit se journaliser en Warn : %s", logs.String())
	}
	if _, ok := f.status(1); ok || f.fetchN.Load() != 0 {
		t.Fatal("la file doit rester vide")
	}
}

// (8) un arrêt pendant la reconstruction laisse la demande au fichier, non terminée.
func TestDemand_ArretPendantLaReconstructionGardeLaDemande(t *testing.T) {
	f := newDemandFixture(t, 0)
	started := make(chan struct{}, 1)
	f.deps.Rebuild = func(ctx context.Context, _, _ string) error {
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}
	f.rebind(t)
	if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
		t.Fatal(err)
	}
	<-started
	f.cancel()
	f.wg.Wait()
	if got := f.store.snapshot(); len(got) != 1 || got[0].MatchID != demandFor(1).MatchID {
		t.Fatalf("la demande doit rester au fichier après l'arrêt : %+v", got)
	}
	if st, ok := f.status(1); !ok || st.State != domain.ReplayGenerationQueued {
		t.Fatalf("après l'arrêt la demande doit se lire en file : %+v ok=%v", st, ok)
	}
}

// (9) le classement des issues, de bout en bout pour le film disparu et l'essai répété d'un décodage occupé.
func TestDemand_FilmDisparuEtDecodageOccupe(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.fetchErr = func(d domain.ReplayGenerationDemand) error {
		if d.MatchID == demandFor(1).MatchID {
			return fmt.Errorf("manifeste: %w", domain.ErrReplayFilmGone)
		}
		return nil
	}
	var busy atomic.Int32
	prev := f.deps.Rebuild
	f.deps.Rebuild = func(ctx context.Context, slug, id string) error {
		if busy.Add(1) <= 2 {
			return fmt.Errorf("%w: tenu", ErrBusy)
		}
		return prev(ctx, slug, id)
	}
	f.rebind(t)
	if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
		t.Fatal(err)
	}
	if st := f.waitFailed(t, 1); st.State != domain.ReplayGenerationFilmUnavailable {
		t.Fatalf("film disparu : %+v", st)
	}
	if f.calls.Load() != 0 || f.releaseN.Load() != 0 {
		t.Fatal("un film disparu ne reconstruit rien et ne libère rien")
	}
	if _, _, err := f.svc.Enqueue(demandFor(2)); err != nil {
		t.Fatal(err)
	}
	f.waitSettled(t, 2)
	if busy.Load() != 3 {
		t.Fatalf("Rebuild appelé %d fois, attendu 3 (deux ErrBusy puis le succès)", busy.Load())
	}
	if f.releaseN.Load() != 1 {
		t.Fatalf("ReleaseFilm = %d, attendu 1 après le succès", f.releaseN.Load())
	}
}

func TestClassifyDemand(t *testing.T) {
	cas := []struct {
		nom      string
		err      error
		failedIn string
		state    string
		reason   string
		retry    bool
	}{
		{"succès", nil, "", domain.ReplayGenerationReady, "", false},
		{"film disparu", fmt.Errorf("x: %w", domain.ErrReplayFilmGone), domain.ReplayGenPhaseDownloading, domain.ReplayGenerationFilmUnavailable, "", false},
		{"sans jeton", fmt.Errorf("x: %w", domain.ErrReplayNoToken), domain.ReplayGenPhaseDownloading, domain.ReplayGenerationFailed, domain.ReplayGenReasonTokenUnavailable, true},
		{"film non fini", finalise.ErrFilmNonFinalise, domain.ReplayGenPhaseDownloading, domain.ReplayGenerationFailed, domain.ReplayGenReasonFilmNotReady, true},
		{"carte hors catalogue", replaybuild.ErrMapNotInCatalog, domain.ReplayGenPhaseBuilding, domain.ReplayGenerationFailed, domain.ReplayGenReasonMapNotInCatalog, false},
		{"clé de film inconnue", replaybuild.ErrUnknownFilmKey, domain.ReplayGenPhaseBuilding, domain.ReplayGenerationFailed, domain.ReplayGenReasonUnknownFilmKey, false},
		{"mode non géré", replaybuild.ErrModeNonGere, domain.ReplayGenPhaseBuilding, domain.ReplayGenerationNotSupported, domain.ReplayGenReasonModeUnmanaged, false},
		{"échec de téléchargement", errors.New("réseau"), domain.ReplayGenPhaseDownloading, domain.ReplayGenerationFailed, domain.ReplayGenReasonFetchFailed, true},
		{"échec de construction", errors.New("enfant"), domain.ReplayGenPhaseBuilding, domain.ReplayGenerationFailed, domain.ReplayGenReasonBuildFailed, true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			st := classifyDemand(c.err, c.failedIn)
			if st.State != c.state || st.Reason != c.reason || st.Retryable != c.retry {
				t.Fatalf("classement = %+v, attendu state=%s reason=%s retry=%v", st, c.state, c.reason, c.retry)
			}
		})
	}
}

// (10) un échec se remplace par une nouvelle demande ; un rejeu devenu à jour sort en `ready` sans rien lancer.
func TestDemand_EchecPuisNouvelleDemandeEtRejeuDejaAJour(t *testing.T) {
	f := newDemandFixture(t, 0)
	var failOnce atomic.Bool
	failOnce.Store(true)
	f.fetchErr = func(domain.ReplayGenerationDemand) error {
		if failOnce.Swap(false) {
			return errors.New("réseau coupé")
		}
		return nil
	}
	if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
		t.Fatal(err)
	}
	st := f.waitFailed(t, 1)
	if st.State != domain.ReplayGenerationFailed || st.Reason != domain.ReplayGenReasonFetchFailed || !st.Retryable {
		t.Fatalf("échec attendu fetch_failed réessayable : %+v", st)
	}
	if _, created, err := f.svc.Enqueue(demandFor(1)); err != nil || !created {
		t.Fatalf("nouvelle demande après échec : created=%v err=%v", created, err)
	}
	f.waitSettled(t, 1)
	if f.calls.Load() != 1 {
		t.Fatalf("Rebuild = %d, attendu 1", f.calls.Load())
	}
	// rejeu déjà à jour au tour : ni téléchargement ni reconstruction
	writeArtifact(t, f.paths, shortOf(2), true)
	fetchAvant, callsAvant := f.fetchN.Load(), f.calls.Load()
	if _, _, err := f.svc.Enqueue(demandFor(2)); err != nil {
		t.Fatal(err)
	}
	f.waitSettled(t, 2)
	if f.fetchN.Load() != fetchAvant || f.calls.Load() != callsAvant {
		t.Fatal("un rejeu à jour ne doit déclencher ni téléchargement ni reconstruction")
	}
}

// (11) les issues gardées sont bornées : la plus ancienne sort.
func TestDemand_IssuesBornees(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.fetchErr = func(domain.ReplayGenerationDemand) error { return errors.New("réseau coupé") }
	total := maxResults + 8
	for i := 1; i <= total; i++ {
		if _, _, err := f.svc.Enqueue(demandFor(i)); err != nil {
			t.Fatal(err)
		}
		f.waitFailed(t, i)
	}
	f.svc.mu.Lock()
	n, order := len(f.svc.results), len(f.svc.resultOrder)
	f.svc.mu.Unlock()
	if n != maxResults || order != maxResults {
		t.Fatalf("issues gardées = %d (ordre %d), attendu %d", n, order, maxResults)
	}
	if _, ok := f.status(1); ok {
		t.Fatal("la plus ancienne issue doit être sortie")
	}
	if _, ok := f.status(total); !ok {
		t.Fatal("la plus récente doit être gardée")
	}
}

// Le film n'est libéré qu'après une reconstruction réussie.
func TestDemand_FilmLibereApresSuccesSeulement(t *testing.T) {
	f := newDemandFixture(t, 0)
	f.deps.Rebuild = func(context.Context, string, string) error { return errors.New("enfant") }
	f.rebind(t)
	if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
		t.Fatal(err)
	}
	if st := f.waitFailed(t, 1); st.Reason != domain.ReplayGenReasonBuildFailed {
		t.Fatalf("attendu build_failed : %+v", st)
	}
	if f.releaseN.Load() != 0 {
		t.Fatal("aucun film ne doit être libéré après un échec")
	}
}
