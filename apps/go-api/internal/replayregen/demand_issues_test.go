package replayregen

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/replaybuild"

	"github.com/JGtm/halo-infinite-film-decoder/finalise"
)

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
		{"film pas encore prêt", fmt.Errorf("x: %w", domain.ErrReplayFilmNotReady), domain.ReplayGenPhaseDownloading, domain.ReplayGenerationFailed, domain.ReplayGenReasonFilmNotReady, true},
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

// Un échec réessayable garde le film (le nouvel essai le reprend sans le retélécharger).
func TestDemand_FilmGardeApresUnEchecReessayable(t *testing.T) {
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
		t.Fatal("aucun film ne doit être libéré après un échec réessayable")
	}
}

// Un échec définitif de la construction (carte hors catalogue, film illisible, mode non géré) libère le film : un
// nouvel essai ne changerait rien.
func TestDemand_FilmLibereApresUnEchecDefinitif(t *testing.T) {
	for _, tc := range []struct {
		nom   string
		err   error
		state string
	}{
		{"carte hors catalogue", replaybuild.ErrMapNotInCatalog, domain.ReplayGenerationFailed},
		{"clé de film inconnue", fmt.Errorf("lecture: %w", replaybuild.ErrUnknownFilmKey), domain.ReplayGenerationFailed},
		{"mode non géré", replaybuild.ErrModeNonGere, domain.ReplayGenerationNotSupported},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			f := newDemandFixture(t, 0)
			f.deps.Rebuild = func(context.Context, string, string) error { return tc.err }
			f.rebind(t)
			if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
				t.Fatal(err)
			}
			if st := f.waitFailed(t, 1); st.State != tc.state || st.Retryable {
				t.Fatalf("attendu un échec définitif %s : %+v", tc.state, st)
			}
			if f.releaseN.Load() != 1 {
				t.Fatalf("ReleaseFilm = %d, attendu 1 après un échec définitif", f.releaseN.Load())
			}
		})
	}
}

// Un arrêt du serveur pendant la construction garde le film : la demande recommence à l'ouverture suivante.
func TestDemand_FilmGardeApresUnArret(t *testing.T) {
	f := newDemandFixture(t, 0)
	started := make(chan struct{}, 1)
	f.deps.Rebuild = func(ctx context.Context, _, _ string) error {
		started <- struct{}{}
		<-ctx.Done()
		return fmt.Errorf("%w: %w", replaybuild.ErrMapNotInCatalog, ctx.Err())
	}
	f.rebind(t)
	if _, _, err := f.svc.Enqueue(demandFor(1)); err != nil {
		t.Fatal(err)
	}
	<-started
	f.cancel()
	f.wg.Wait()
	if f.releaseN.Load() != 0 {
		t.Fatal("un arrêt ne doit libérer aucun film")
	}
}
