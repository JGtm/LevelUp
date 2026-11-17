package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

const (
	genTitre  = "titre_a"
	genMatch  = "0a0a0a0a"
	genFullID = "0a0a0a0a-1111-2222-3333-444444444444"
)

type fakeGenQueue struct {
	mu       sync.Mutex
	known    map[string]domain.ReplayGenerationStatus
	enqueued []domain.ReplayGenerationDemand
	err      error
}

func (q *fakeGenQueue) Enqueue(d domain.ReplayGenerationDemand) (domain.ReplayGenerationStatus, bool, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.err != nil {
		return domain.ReplayGenerationStatus{}, false, q.err
	}
	q.enqueued = append(q.enqueued, d)
	return domain.ReplayGenerationStatus{State: domain.ReplayGenerationQueued, Position: len(q.enqueued)}, true, nil
}

func (q *fakeGenQueue) DemandStatus(_, matchID string) (domain.ReplayGenerationStatus, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	st, ok := q.known[matchID]
	return st, ok
}

type fakeGenProbe struct {
	res   port.ReplayMatchProbeResult
	err   error
	calls int
}

func (p *fakeGenProbe) ProbeReplayMatch(context.Context, string, string) (port.ReplayMatchProbeResult, error) {
	p.calls++
	return p.res, p.err
}

type genFixture struct {
	queue *fakeGenQueue
	probe *fakeGenProbe
	token bool
	caps  games.CapabilityMap
	logs  *bytes.Buffer
}

func newGenFixture(t *testing.T) *genFixture {
	t.Helper()
	f := &genFixture{
		queue: &fakeGenQueue{known: map[string]domain.ReplayGenerationStatus{}},
		probe: &fakeGenProbe{res: port.ReplayMatchProbeResult{FullID: genFullID, Known: true}},
		token: true,
		caps:  games.CapabilityMap{games.CapFilmReplayArtifact: games.CapSupported},
		logs:  &bytes.Buffer{},
	}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(f.logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return f
}

func (f *genFixture) service() *ReplayGenerationService {
	return NewReplayGenerationService(f.caps, f.queue, f.probe, func(xuid string) bool { return f.token && xuid == "xuid-page" },
		ReplayGenerationPlayer{Slug: "Joueur", XUID: "xuid-page", Gamertag: "Joueur"})
}

func (f *genFixture) status(t *testing.T) domain.ReplayGenerationStatus {
	t.Helper()
	st, err := f.service().GenerationStatus(context.Background(), genTitre, genMatch)
	if err != nil {
		t.Fatalf("GenerationStatus: %v", err)
	}
	return st
}

// Capacité absente : 503 propre (ErrCapabilityNotSupported) sur la lecture comme sur la demande, journalisé.
func TestReplayGeneration_CapaciteAbsente(t *testing.T) {
	f := newGenFixture(t)
	f.caps = games.CapabilityMap{games.CapFilmReplayArtifact: games.CapNotExposed}
	s := f.service()
	if _, err := s.GenerationStatus(context.Background(), genTitre, genMatch); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("GET : %v, attendu ErrCapabilityNotSupported", err)
	}
	if _, _, err := s.RequestGeneration(context.Background(), genTitre, genMatch); !errors.Is(err, games.ErrCapabilityNotSupported) {
		t.Fatalf("POST : %v, attendu ErrCapabilityNotSupported", err)
	}
	if f.probe.calls != 0 || len(f.queue.enqueued) != 0 {
		t.Fatal("sans capacité, ni lecture de base ni mise en file")
	}
	if !strings.Contains(f.logs.String(), "level=WARN") || !strings.Contains(f.logs.String(), "capacité") {
		t.Fatalf("le refus doit se journaliser : %s", f.logs.String())
	}
}

// Les refus : mode non géré, match inconnu, bit posé, sans jeton. L'état se lit, la demande est refusée et journalisée.
func TestReplayGeneration_RefusParEtat(t *testing.T) {
	cas := []struct {
		nom    string
		probe  port.ReplayMatchProbeResult
		token  bool
		state  string
		reason string
		err    error
	}{
		{"match inconnu", port.ReplayMatchProbeResult{}, true, domain.ReplayGenerationNotSupported, domain.ReplayGenReasonMatchUnknown, domain.ErrReplayGenerationNotSupported},
		{"mode non géré", port.ReplayMatchProbeResult{Known: true, UnmanagedMode: true}, true, domain.ReplayGenerationNotSupported, domain.ReplayGenReasonModeUnmanaged, domain.ErrReplayGenerationNotSupported},
		{"film absent", port.ReplayMatchProbeResult{Known: true, FilmAbsent: true}, true, domain.ReplayGenerationFilmUnavailable, "", domain.ErrReplayFilmUnavailable},
		{"sans jeton, film non archivé", port.ReplayMatchProbeResult{Known: true}, false, domain.ReplayGenerationNotSupported, domain.ReplayGenReasonNoToken, domain.ErrReplayGenerationNotSupported},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			f := newGenFixture(t)
			f.probe.res, f.token = c.probe, c.token
			if st := f.status(t); st.State != c.state || st.Reason != c.reason {
				t.Fatalf("état = %+v, attendu %s/%s", st, c.state, c.reason)
			}
			_, created, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch)
			if !errors.Is(err, c.err) || created {
				t.Fatalf("POST : created=%v err=%v, attendu %v", created, err, c.err)
			}
			if len(f.queue.enqueued) != 0 {
				t.Fatal("un refus ne met rien en file")
			}
			if !strings.Contains(f.logs.String(), "demande refusée") {
				t.Fatalf("le refus doit se journaliser : %s", f.logs.String())
			}
		})
	}
}

// La raison d'un refus `not_supported` se lit sur l'erreur typée.
func TestReplayGeneration_RaisonSurErreurTypee(t *testing.T) {
	f := newGenFixture(t)
	f.probe.res = port.ReplayMatchProbeResult{Known: true, UnmanagedMode: true}
	_, _, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch)
	var ns *domain.ReplayGenerationNotSupportedError
	if !errors.As(err, &ns) || ns.Reason != domain.ReplayGenReasonModeUnmanaged {
		t.Fatalf("erreur = %v, attendu ReplayGenerationNotSupportedError{mode_unmanaged}", err)
	}
}

// Rejeu déjà rangé : `ready`, rien en file, la demande rend 200 sans rien ajouter.
func TestReplayGeneration_RejeuRange(t *testing.T) {
	f := newGenFixture(t)
	f.probe.res.ArtifactPresent = true
	if st := f.status(t); st.State != domain.ReplayGenerationReady {
		t.Fatalf("état = %+v", st)
	}
	st, created, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch)
	if err != nil || created || st.State != domain.ReplayGenerationReady || len(f.queue.enqueued) != 0 {
		t.Fatalf("POST : st=%+v created=%v err=%v enqueued=%d", st, created, err, len(f.queue.enqueued))
	}
}

// Sans jeton, un film déjà à l'archive se génère sans jeton.
func TestReplayGeneration_FilmArchiveSansJeton(t *testing.T) {
	f := newGenFixture(t)
	f.token = false
	f.probe.res.FilmArchived = true
	if st := f.status(t); st.State != domain.ReplayGenerationGeneratable {
		t.Fatalf("état = %+v, attendu generatable", st)
	}
	if _, created, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch); err != nil || !created {
		t.Fatalf("POST : created=%v err=%v", created, err)
	}
}

// La demande porte l'identité complète du match et le joueur de la page, rien d'autre.
func TestReplayGeneration_DemandeLieeAuJoueurDeLaPage(t *testing.T) {
	f := newGenFixture(t)
	st, created, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch)
	if err != nil || !created || st.State != domain.ReplayGenerationQueued {
		t.Fatalf("POST : st=%+v created=%v err=%v", st, created, err)
	}
	if len(f.queue.enqueued) != 1 {
		t.Fatalf("demandes = %d", len(f.queue.enqueued))
	}
	d := f.queue.enqueued[0]
	if d.TitleSlug != genTitre || d.MatchID != genFullID || d.PlayerSlug != "Joueur" || d.XUID != "xuid-page" ||
		d.Gamertag != "Joueur" || d.RequestedAt.IsZero() {
		t.Fatalf("demande = %+v", d)
	}
}

// Doublon : une demande déjà en file ou en cours rend son état, sans nouvelle entrée ; la sonde n'est pas lue.
func TestReplayGeneration_DoublonIdempotent(t *testing.T) {
	for _, etat := range []string{domain.ReplayGenerationQueued, domain.ReplayGenerationRunning} {
		f := newGenFixture(t)
		f.queue.known[genMatch] = domain.ReplayGenerationStatus{State: etat, Position: 2, Phase: domain.ReplayGenPhaseBuilding}
		st, created, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch)
		if err != nil || created || st.State != etat {
			t.Fatalf("%s : st=%+v created=%v err=%v", etat, st, created, err)
		}
		if len(f.queue.enqueued) != 0 || f.probe.calls != 0 {
			t.Fatalf("%s : rien à ajouter ni à lire (enqueued=%d, sondes=%d)", etat, len(f.queue.enqueued), f.probe.calls)
		}
	}
}

// Un échec gardé par la file : réessayable -> la demande repart ; non réessayable -> refusée avec sa raison.
func TestReplayGeneration_EchecGarde(t *testing.T) {
	f := newGenFixture(t)
	f.queue.known[genMatch] = domain.ReplayGenerationStatus{State: domain.ReplayGenerationFailed, Reason: domain.ReplayGenReasonFetchFailed, Retryable: true}
	if st := f.status(t); st.State != domain.ReplayGenerationFailed || !st.Retryable {
		t.Fatalf("état = %+v", st)
	}
	if _, created, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch); err != nil || !created {
		t.Fatalf("réessai : created=%v err=%v", created, err)
	}

	f2 := newGenFixture(t)
	f2.queue.known[genMatch] = domain.ReplayGenerationStatus{State: domain.ReplayGenerationFailed, Reason: domain.ReplayGenReasonMapNotInCatalog}
	_, _, err := f2.service().RequestGeneration(context.Background(), genTitre, genMatch)
	var ns *domain.ReplayGenerationNotSupportedError
	if !errors.As(err, &ns) || ns.Reason != domain.ReplayGenReasonMapNotInCatalog || len(f2.queue.enqueued) != 0 {
		t.Fatalf("échec définitif : err=%v", err)
	}
}

// Un rejeu rangé l'emporte sur une issue périmée de la file.
func TestReplayGeneration_RejeuRangeEmporteSurUnEchecGarde(t *testing.T) {
	f := newGenFixture(t)
	f.queue.known[genMatch] = domain.ReplayGenerationStatus{State: domain.ReplayGenerationFailed, Retryable: true}
	f.probe.res.ArtifactPresent = true
	if st := f.status(t); st.State != domain.ReplayGenerationReady {
		t.Fatalf("état = %+v, attendu ready", st)
	}
}

// File pleine : l'erreur se propage et se journalise ; panne de lecture : idem.
func TestReplayGeneration_FilePleineEtPanneDeLecture(t *testing.T) {
	f := newGenFixture(t)
	f.queue.err = domain.ErrReplayGenerationQueueFull
	if _, _, err := f.service().RequestGeneration(context.Background(), genTitre, genMatch); !errors.Is(err, domain.ErrReplayGenerationQueueFull) {
		t.Fatalf("err = %v, attendu ErrReplayGenerationQueueFull", err)
	}
	if !strings.Contains(f.logs.String(), "refusée par la file") {
		t.Fatalf("le refus de la file doit se journaliser : %s", f.logs.String())
	}

	f2 := newGenFixture(t)
	f2.probe.err = errors.New("base illisible")
	if _, _, err := f2.service().RequestGeneration(context.Background(), genTitre, genMatch); err == nil {
		t.Fatal("une panne de lecture doit se rendre")
	}
	if !strings.Contains(f2.logs.String(), "level=ERROR") {
		t.Fatalf("la panne doit se journaliser en Error : %s", f2.logs.String())
	}
}
