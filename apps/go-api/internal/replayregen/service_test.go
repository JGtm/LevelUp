package replayregen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/LevelUp/libs/film/replay"
)

const testSlug = "halo_infinite"

// fakePending : le drapeau persisté, en mémoire.
type fakePending struct {
	mu sync.Mutex
	on bool
}

func (p *fakePending) ReplayRegenPending() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.on, nil
}

func (p *fakePending) SetReplayRegenPending(on bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.on = on
	return nil
}

func (p *fakePending) get() bool { v, _ := p.ReplayRegenPending(); return v }

// writeArtifact range un artefact : à jour (schéma et couches courants) ou en retard (schéma 1, sans couches).
func writeArtifact(t *testing.T, paths *title.PathResolver, matchID string, current bool) {
	t.Helper()
	doc := map[string]any{"matchId": matchID, "schemaVersion": 1}
	if current {
		doc["schemaVersion"] = replay.SchemaVersion
		layers := map[string]string{}
		for fam, rev := range replay.RevisionsCourantesDesCouches() {
			layers[fam] = rev
		}
		doc["layers"] = layers
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	p := paths.ReplayArtifactPath(testSlug, matchID)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

type fixture struct {
	paths   *title.PathResolver
	pending *fakePending
	svc     *Service
	deps    Deps
	films   map[string]bool
	heavy   atomic.Bool
	calls   atomic.Int32
	inFly   atomic.Int32
	maxFly  atomic.Int32
	rebuild func(ctx context.Context, slug, id string) error
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// newFixture relie un service sur une racine temporaire. Par défaut la reconstruction réécrit l'artefact à jour.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{paths: title.NewPathResolver(t.TempDir()), pending: &fakePending{}, films: map[string]bool{}}
	f.heavy.Store(true)
	f.rebuild = func(_ context.Context, _, id string) error {
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	f.deps = Deps{
		Paths:  f.paths,
		Titles: func() []string { return []string{testSlug} },
		FilmPresent: func(_, id string) bool {
			return f.films[id]
		},
		Rebuild: func(ctx context.Context, slug, id string) error {
			n := f.inFly.Add(1)
			defer f.inFly.Add(-1)
			for {
				m := f.maxFly.Load()
				if n <= m || f.maxFly.CompareAndSwap(m, n) {
					break
				}
			}
			f.calls.Add(1)
			time.Sleep(5 * time.Millisecond)
			return f.rebuild(ctx, slug, id)
		},
		Pending:        f.pending,
		HeavyAllowed:   f.heavy.Load,
		GateEvery:      10 * time.Millisecond,
		BusyRetryEvery: 5 * time.Millisecond,
		BusyRetries:    3,
		RescanEvery:    time.Hour,
	}
	f.svc = New()
	ctx, cancel := context.WithCancel(context.Background())
	f.cancel = cancel
	f.svc.Bind(ctx, &f.wg, f.deps)
	t.Cleanup(func() { cancel(); f.wg.Wait() })
	return f
}

// seed range n artefacts en retard dont le film est présent.
func (f *fixture) seed(t *testing.T, n int) []string {
	t.Helper()
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("%08x", 0x100+i)
		writeArtifact(t, f.paths, ids[i], false)
		f.films[ids[i]] = true
	}
	return ids
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("délai dépassé : %s", what)
}

func TestScan_NeCompteQueLesRejeuxEnRetardDontLeFilmExiste(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 2)
	writeArtifact(t, f.paths, "a0a0a0a0", true) // à jour : exclu
	writeArtifact(t, f.paths, "5af11111", false)
	f.films["a0a0a0a0"] = true // film présent mais rejeu à jour
	// une marque de dérivation voisine n'est pas un artefact
	if err := os.WriteFile(f.paths.ReplayArtifactPath(testSlug, "00000100")+".derived.json", []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cands, err := f.svc.scan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("candidats = %d, attendu 2 (en retard avec film) : %+v", len(cands), cands)
	}
}

func TestScan_ArtefactPlusRecentQueLeBinaireNEstPasEnRetard(t *testing.T) {
	f := newFixture(t)
	p := f.paths.ReplayArtifactPath(testSlug, "f00d0001")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(fmt.Sprintf(`{"matchId":"f00d0001","schemaVersion":%d}`, replay.SchemaVersion+5)), 0o644); err != nil {
		t.Fatal(err)
	}
	f.films["f00d0001"] = true
	cands, _ := f.svc.scan(context.Background())
	if len(cands) != 0 {
		t.Fatalf("un artefact d'un schéma plus récent ne se régénère pas : %+v", cands)
	}
}

func TestStatus_LeDecompteSeFaitEnArriereEtNeLancePasLaFile(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 3)
	if st := f.svc.Status(); st.Running || st.Checked {
		t.Fatalf("premier état : %+v (décompte pas encore fait, file jamais lancée)", st)
	}
	waitFor(t, "décompte", func() bool { return f.svc.Status().Checked })
	st := f.svc.Status()
	if st.Stale != 3 || st.Running {
		t.Fatalf("état = %+v, attendu 3 en retard, file arrêtée", st)
	}
	time.Sleep(50 * time.Millisecond)
	if f.calls.Load() != 0 {
		t.Fatal("rien ne se régénère sans le démarrage de l'utilisateur")
	}
	if f.pending.get() {
		t.Fatal("le drapeau de reprise ne se pose qu'à un démarrage demandé")
	}
}

func TestStart_UnFilmALaFoisJusquAuBout(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 6)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	if !f.pending.get() {
		t.Fatal("le démarrage pose le drapeau de reprise")
	}
	waitFor(t, "fin de file", func() bool { return !f.svc.Status().Running })
	st := f.svc.Status()
	if st.Done != 6 || st.Total != 6 || st.Failed != 0 {
		t.Fatalf("état final = %+v", st)
	}
	if got := f.maxFly.Load(); got != 1 {
		t.Fatalf("films décodés en parallèle = %d, attendu 1 (un seul film à la fois)", got)
	}
	if f.calls.Load() != 6 {
		t.Fatalf("reconstructions = %d, attendu 6", f.calls.Load())
	}
	if f.pending.get() {
		t.Fatal("la fin de file efface le drapeau de reprise")
	}
	waitFor(t, "décompte final", func() bool { s := f.svc.Status(); return s.Checked && s.Stale == 0 })
}

func TestStart_Idempotent(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 3)
	f.heavy.Store(false) // la file reste en attente : un second démarrage ne la relance pas
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	st, err := f.svc.Start()
	if err != nil || !st.Running {
		t.Fatalf("second démarrage : %+v, %v", st, err)
	}
	f.heavy.Store(true)
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	if f.calls.Load() != 3 {
		t.Fatalf("reconstructions = %d, attendu 3 (une seule file)", f.calls.Load())
	}
}

func TestModeJeu_LaFileAttendLaFinDeLaPartie(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 2)
	f.heavy.Store(false)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "attente de la partie", func() bool { return f.svc.Status().WaitingForGame })
	time.Sleep(60 * time.Millisecond)
	if f.calls.Load() != 0 {
		t.Fatal("aucun film ne démarre pendant une partie")
	}
	f.heavy.Store(true)
	waitFor(t, "fin de file", func() bool { return !f.svc.Status().Running })
	if f.calls.Load() != 2 {
		t.Fatalf("reconstructions = %d, attendu 2 après la partie", f.calls.Load())
	}
}

func TestPauseEtReprise(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 4)
	gate := make(chan struct{})
	f.rebuild = func(_ context.Context, _, id string) error {
		<-gate
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "premier film en cours", func() bool { return f.inFly.Load() == 1 })
	st, err := f.svc.Pause()
	if err != nil || !st.Paused {
		t.Fatalf("pause : %+v, %v", st, err)
	}
	if f.pending.get() {
		t.Fatal("une file en pause ne reprend pas toute seule : drapeau effacé")
	}
	gate <- struct{}{} // le film en cours va à son terme
	waitFor(t, "film terminé", func() bool { return f.svc.Status().Done == 1 })
	time.Sleep(60 * time.Millisecond)
	if f.calls.Load() != 1 {
		t.Fatalf("reconstructions = %d : le film suivant ne part pas en pause", f.calls.Load())
	}
	if _, err := f.svc.Resume(); err != nil {
		t.Fatal(err)
	}
	if !f.pending.get() {
		t.Fatal("la reprise repose le drapeau")
	}
	close(gate)
	waitFor(t, "fin de file", func() bool { return !f.svc.Status().Running })
	if f.svc.Status().Done != 4 {
		t.Fatalf("état = %+v", f.svc.Status())
	}
}

func TestPauseEtReprise_SansFileRefusees(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.Pause(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("pause sans file : %v", err)
	}
	if _, err := f.svc.Resume(); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("reprise sans file : %v", err)
	}
}

func TestStart_ServiceNonRelie(t *testing.T) {
	if _, err := New().Start(); !errors.Is(err, ErrNotBound) {
		t.Fatalf("démarrage sans liaison : %v", err)
	}
}

func TestRepriseApresRedemarrage(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 2)
	f.pending.on = true // la file était active avant l'arrêt
	f.svc.ResumeIfPending()
	waitFor(t, "fin de file", func() bool { s := f.svc.Status(); return !s.Running && s.Done == 2 })
	if f.pending.get() {
		t.Fatal("la file reprise puis finie efface le drapeau")
	}
}

func TestRepriseApresRedemarrage_SansDrapeauNeFaitRien(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 2)
	f.svc.ResumeIfPending()
	time.Sleep(50 * time.Millisecond)
	if f.calls.Load() != 0 || f.svc.Status().Running {
		t.Fatal("sans drapeau, la file ne part pas")
	}
}

func TestArretDuServeur_GardeLeDrapeauPourLaReprise(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 3)
	f.rebuild = func(ctx context.Context, _, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "film en cours", func() bool { return f.inFly.Load() == 1 })
	f.cancel()
	waitFor(t, "file arrêtée", func() bool { return !f.svc.Status().Running })
	if !f.pending.get() {
		t.Fatal("un arrêt du serveur laisse le drapeau posé : la file reprend à l'ouverture suivante")
	}
	if f.svc.Status().Failed != 0 {
		t.Fatal("un film coupé par l'arrêt n'est pas un échec")
	}
}

func TestIdempotence_UnRejeuDejaRefaitSeSaute(t *testing.T) {
	f := newFixture(t)
	ids := f.seed(t, 3)
	f.heavy.Store(false)
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "liste prête", func() bool { return f.svc.Status().Total == 3 })
	writeArtifact(t, f.paths, ids[1], true) // refait entre-temps (ouverture de la page de rejeu)
	f.heavy.Store(true)
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	if f.calls.Load() != 2 {
		t.Fatalf("reconstructions = %d, attendu 2 (le rejeu déjà à jour est sauté)", f.calls.Load())
	}
	if st := f.svc.Status(); st.Done != 3 {
		t.Fatalf("état = %+v", st)
	}
}

func TestClassement_RefusVoulusEcartesEchecsComptes(t *testing.T) {
	f := newFixture(t)
	ids := f.seed(t, 3)
	f.rebuild = func(_ context.Context, _, id string) error {
		switch id {
		case ids[0]:
			return fmt.Errorf("x: %w", replaybuild.ErrMapNotInCatalog)
		case ids[1]:
			return errors.New("enfant mort")
		}
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	st := f.svc.Status()
	if st.Done != 3 || st.Failed != 1 {
		t.Fatalf("état = %+v, attendu 3 traités dont 1 échec (la carte hors catalogue est écartée)", st)
	}
	waitFor(t, "décompte final", func() bool { return f.svc.Status().Checked && !f.svc.Status().Running })
	if got := f.svc.Status().Stale; got != 1 {
		t.Fatalf("en retard après la file = %d, attendu 1 (l'échec ; l'écarté ne compte plus)", got)
	}
}

func TestOccupe_LaFileAttendEtRecommenceLeMemeFilm(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 1)
	var tries atomic.Int32
	f.rebuild = func(_ context.Context, _, id string) error {
		if tries.Add(1) < 3 {
			return fmt.Errorf("décodage en cours: %w", ErrBusy)
		}
		writeArtifact(t, f.paths, id, true)
		return nil
	}
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	if st := f.svc.Status(); st.Failed != 0 || st.Done != 1 || tries.Load() != 3 {
		t.Fatalf("état = %+v, essais = %d", st, tries.Load())
	}
}

func TestOccupe_BorneEpuiseeCompteUnEchec(t *testing.T) {
	f := newFixture(t)
	f.seed(t, 1)
	f.rebuild = func(context.Context, string, string) error { return ErrBusy }
	if _, err := f.svc.Start(); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "fin", func() bool { return !f.svc.Status().Running })
	if st := f.svc.Status(); st.Failed != 1 || f.calls.Load() != 4 {
		t.Fatalf("état = %+v, appels = %d (1 essai + 3 reprises)", st, f.calls.Load())
	}
}
