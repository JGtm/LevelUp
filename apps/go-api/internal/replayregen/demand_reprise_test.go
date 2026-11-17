package replayregen

import (
	"context"
	"errors"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
)

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
