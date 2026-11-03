package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"levelup/go-api/internal/platform/schedtask"
	"levelup/go-api/internal/platform/settings"
)

// fakeTasks simule le planificateur : la présence de la tâche et le journal des appels.
type fakeTasks struct {
	present   bool
	calls     []string
	queryErr  error
	ensureErr error
	removeErr error
}

func (f *fakeTasks) Query(context.Context) (schedtask.State, error) {
	f.calls = append(f.calls, "query")
	if f.queryErr != nil {
		return schedtask.StateError, f.queryErr
	}
	if f.present {
		return schedtask.StatePresent, nil
	}
	return schedtask.StateAbsent, nil
}

func (f *fakeTasks) Ensure(context.Context) error {
	f.calls = append(f.calls, "ensure")
	if f.ensureErr != nil {
		return f.ensureErr
	}
	f.present = true
	return nil
}

func (f *fakeTasks) Remove(context.Context) error {
	f.calls = append(f.calls, "remove")
	if f.removeErr != nil {
		return f.removeErr
	}
	f.present = false
	return nil
}

// writes : les appels qui modifient la tâche (hors lectures).
func (f *fakeTasks) writes() []string {
	var out []string
	for _, c := range f.calls {
		if c != "query" {
			out = append(out, c)
		}
	}
	return out
}

func newReconciler(f *fakeTasks) (*refreshReconciler, *schedtask.Observed) {
	obs := &schedtask.Observed{}
	return newRefreshReconciler(f, obs), obs
}

// TestRefreshReconciler_DemarrageReglageCoupeSansTacheNEcritRien : réglage coupé sans tâche (le défaut) : le
// démarrage lit et n'écrit rien.
func TestRefreshReconciler_DemarrageReglageCoupeSansTacheNEcritRien(t *testing.T) {
	f := &fakeTasks{present: false}
	r, obs := newReconciler(f)
	r.reconcile(context.Background(), false)
	if len(f.writes()) != 0 {
		t.Errorf("écritures %v, attendu aucune", f.writes())
	}
	if s, code := obs.Get(); s != schedtask.StateAbsent || code != "" {
		t.Errorf("état observé %q %q, attendu absent", s, code)
	}
}

// TestRefreshReconciler_DemarrageReglageActifReecritToujoursLaTache : réglage actif et tâche DÉJÀ listée : elle est
// réécrite (un seul Ensure, `/F`), car une réinstallation dans un autre dossier ou une racine changée
// laisserait une tâche périmée affichée « présente ».
func TestRefreshReconciler_DemarrageReglageActifReecritToujoursLaTache(t *testing.T) {
	f := &fakeTasks{present: true}
	r, obs := newReconciler(f)
	r.reconcile(context.Background(), true)
	if !reflect.DeepEqual(f.writes(), []string{"ensure"}) {
		t.Fatalf("écritures = %v, attendu un seul ensure", f.writes())
	}
	if s, _ := obs.Get(); s != schedtask.StatePresent {
		t.Errorf("état observé = %q, attendu present", s)
	}
	r.reconcile(context.Background(), true)
	if !reflect.DeepEqual(f.writes(), []string{"ensure"}) {
		t.Fatalf("une valeur déjà traitée ne réécrit pas : %v", f.writes())
	}
}

// TestRefreshReconciler_DemarrageRepare : réglage actif mais tâche absente = elle est posée ; réglage coupé mais
// tâche présente (reste d'une installation précédente) = elle est retirée.
func TestRefreshReconciler_DemarrageRepare(t *testing.T) {
	f := &fakeTasks{present: false}
	r, obs := newReconciler(f)
	r.reconcile(context.Background(), true)
	if !reflect.DeepEqual(f.writes(), []string{"ensure"}) {
		t.Fatalf("écritures = %v, attendu ensure", f.writes())
	}
	if s, _ := obs.Get(); s != schedtask.StatePresent {
		t.Errorf("état observé = %q, attendu present", s)
	}

	f = &fakeTasks{present: true}
	r, obs = newReconciler(f)
	r.reconcile(context.Background(), false)
	if !reflect.DeepEqual(f.writes(), []string{"remove"}) {
		t.Fatalf("écritures = %v, attendu remove", f.writes())
	}
	if s, _ := obs.Get(); s != schedtask.StateAbsent {
		t.Errorf("état observé = %q, attendu absent", s)
	}
}

// TestRefreshReconciler_UnChangementPoseEtRetire : actif puis coupé pose puis retire ; une valeur déjà traitée ne
// refait rien (aucun appel, pas même une lecture).
func TestRefreshReconciler_UnChangementPoseEtRetire(t *testing.T) {
	f := &fakeTasks{}
	r, _ := newReconciler(f)
	ctx := context.Background()
	r.reconcile(ctx, false) // démarrage : rien
	r.reconcile(ctx, true)  // activé : posée
	r.reconcile(ctx, true)  // inchangé : rien
	r.reconcile(ctx, true)
	if !reflect.DeepEqual(f.writes(), []string{"ensure"}) {
		t.Fatalf("écritures = %v, attendu un seul ensure", f.writes())
	}
	calls := len(f.calls)
	r.reconcile(ctx, true)
	if len(f.calls) != calls {
		t.Fatal("une valeur déjà traitée ne rappelle pas le planificateur")
	}
	r.reconcile(ctx, false) // coupé : retirée
	if !reflect.DeepEqual(f.writes(), []string{"ensure", "remove"}) || f.present {
		t.Fatalf("écritures = %v (présente=%v), attendu ensure puis remove", f.writes(), f.present)
	}
}

// TestRefreshReconciler_ErreurJournaliseeExposeeEtNonRetentee : un refus de schtasks est exposé (code fixe, jamais
// le texte) et n'est pas retenté à chaque tic ; basculer le réglage retente.
func TestRefreshReconciler_ErreurJournaliseeExposeeEtNonRetentee(t *testing.T) {
	f := &fakeTasks{ensureErr: errors.New("accès refusé")}
	r, obs := newReconciler(f)
	ctx := context.Background()
	r.reconcile(ctx, false)
	r.reconcile(ctx, true)
	if s, code := obs.Get(); s != schedtask.StateError || code != schedtask.ErrorCodeSchtasks {
		t.Fatalf("état = %q %q, attendu error schtasks_failed", s, code)
	}
	attempts := len(f.writes())
	r.reconcile(ctx, true)
	r.reconcile(ctx, true)
	if len(f.writes()) != attempts {
		t.Fatalf("un échec n'est pas retenté à chaque tic : %v", f.writes())
	}
	// Le réglage bascule, puis revient : une nouvelle tentative, qui réussit.
	f.ensureErr = nil
	r.reconcile(ctx, false)
	r.reconcile(ctx, true)
	if s, code := obs.Get(); s != schedtask.StatePresent || code != "" {
		t.Fatalf("après la nouvelle tentative : %q %q, attendu present", s, code)
	}
}

func TestRefreshReconciler_CheminRefuseEtPlanificateurIllisible(t *testing.T) {
	f := &fakeTasks{ensureErr: schedtask.ErrUnsupportedPath}
	r, obs := newReconciler(f)
	r.reconcile(context.Background(), false)
	r.reconcile(context.Background(), true)
	if s, code := obs.Get(); s != schedtask.StateError || code != schedtask.ErrorCodePath {
		t.Fatalf("chemin refusé : %q %q, attendu error unsupported_path", s, code)
	}

	g := &fakeTasks{queryErr: errors.New("service indisponible")}
	r2, obs2 := newReconciler(g)
	r2.reconcile(context.Background(), true)
	if s, code := obs2.Get(); s != schedtask.StateError || code != schedtask.ErrorCodeSchtasks {
		t.Fatalf("planificateur illisible : %q %q", s, code)
	}
	if len(g.writes()) != 0 {
		t.Fatalf("illisible : rien n'est écrit à l'aveugle (%v)", g.writes())
	}
}

// TestScheduledRefreshLoop_SuitLeReglage : la boucle lit le réglage au démarrage puis aux tics ; des réglages
// illisibles ne changent rien.
func TestScheduledRefreshLoop_SuitLeReglage(t *testing.T) {
	store := &mutableSettings{cfg: settings.Defaults()}
	f := &fakeTasks{}
	r, _ := newReconciler(f)
	ticks := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { runScheduledRefreshLoop(ctx, store, r, ticks, 0); close(done) }()

	// Un tic envoyé deux fois : le second n'est reçu qu'une fois le relevé du premier fini (canal sans tampon).
	tick := func() {
		ticks <- time.Now()
		ticks <- time.Now()
	}
	tick() // réglage coupé : rien
	cfg := settings.Defaults()
	cfg.ScheduledRefresh = true
	store.set(cfg, nil)
	tick() // activé : posée
	store.set(nil, errors.New("illisible"))
	tick()                              // illisible : la tâche reste
	store.set(settings.Defaults(), nil) // coupé de nouveau
	tick()
	cancel()
	<-done
	if !reflect.DeepEqual(f.writes(), []string{"ensure", "remove"}) {
		t.Fatalf("écritures = %v, attendu ensure puis remove", f.writes())
	}
}
