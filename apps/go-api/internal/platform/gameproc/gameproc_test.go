package gameproc

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
)

// scriptedLister rend une suite de listes de processus, la dernière se répétant.
type scriptedLister struct {
	steps [][]string
	errs  []error
	i     int
}

func (l *scriptedLister) Names() ([]string, error) {
	i := l.i
	if i >= len(l.steps) {
		i = len(l.steps) - 1
	}
	l.i++
	var err error
	if i < len(l.errs) {
		err = l.errs[i]
	}
	return l.steps[i], err
}

func TestAnyRunning(t *testing.T) {
	cases := []struct {
		name            string
		running, wanted []string
		want            bool
	}{
		{"présent", []string{"explorer.exe", "HaloInfinite.exe"}, []string{"HaloInfinite.exe"}, true},
		{"casse ignorée", []string{"haloinfinite.EXE"}, []string{"HaloInfinite.exe"}, true},
		{"espaces ignorés", []string{" HaloInfinite.exe "}, []string{"HaloInfinite.exe"}, true},
		{"absent", []string{"explorer.exe"}, []string{"HaloInfinite.exe"}, false},
		{"liste voulue vide : jamais en cours", []string{"explorer.exe"}, nil, false},
		{"nom voulu vide : ignoré", []string{""}, []string{"  "}, false},
		{"un des titres suffit", []string{"MCC.exe"}, []string{"HaloInfinite.exe", "MCC.exe"}, true},
	}
	for _, c := range cases {
		if got := AnyRunning(c.running, c.wanted); got != c.want {
			t.Errorf("%s : AnyRunning = %v, attendu %v", c.name, got, c.want)
		}
	}
}

// newWatcher assemble un watcher dont l'horloge est pilotée et dont les demandes de synchronisation
// sont enregistrées.
func newWatcher(l Lister, clock *time.Time, syncs *[]string) *Watcher {
	return &Watcher{
		Lister:       l,
		Mode:         NewMode(),
		Names:        func() []string { return []string{"HaloInfinite.exe"} },
		OnGameClosed: func(reason string) SyncOutcome { *syncs = append(*syncs, reason); return SyncRequested },
		Now:          func() time.Time { return *clock },
	}
}

func TestTransitions_EnterLeaveDeclencheLaSynchronisation(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC)
	var syncs []string
	w := newWatcher(&scriptedLister{steps: [][]string{
		{"explorer.exe"},
		{"explorer.exe", "HaloInfinite.exe"},
		{"explorer.exe", "HaloInfinite.exe"},
		{"explorer.exe"},
	}}, &clock, &syncs)

	w.Poll(ctx)
	if w.Mode.Running() || !w.Mode.HeavyWorkAllowed() {
		t.Fatal("sans jeu : pas de mode jeu, travail lourd permis")
	}
	w.Poll(ctx)
	if !w.Mode.Running() || w.Mode.HeavyWorkAllowed() {
		t.Fatal("jeu lancé : mode jeu, travail lourd refusé")
	}
	w.Poll(ctx)
	if len(syncs) != 0 {
		t.Fatalf("aucune synchronisation tant que le jeu tourne : %v", syncs)
	}
	w.Poll(ctx)
	if w.Mode.Running() {
		t.Fatal("jeu fermé : le mode jeu retombe")
	}
	if len(syncs) != 1 || syncs[0] != ReasonGameClosed {
		t.Fatalf("la fermeture déclenche UNE synchronisation, obtenu %v", syncs)
	}
}

func TestAntiRebond_UneFermetureProcheEstAbsorbee(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC)
	var syncs []string
	run, off := []string{"HaloInfinite.exe"}, []string{"explorer.exe"}
	w := newWatcher(&scriptedLister{steps: [][]string{run, off, run, off, run, off}}, &clock, &syncs)
	before := observability.LoadCounter(MetricSyncDebounced)

	w.Poll(ctx) // lancé
	w.Poll(ctx) // fermé : synchro
	if len(syncs) != 1 {
		t.Fatalf("première fermeture : %v", syncs)
	}
	clock = clock.Add(20 * time.Second)
	w.Poll(ctx) // relancé
	w.Poll(ctx) // refermé 20 s après : absorbé
	if len(syncs) != 1 {
		t.Fatalf("une fermeture sous l'anti-rebond ne synchronise pas : %v", syncs)
	}
	if got := observability.LoadCounter(MetricSyncDebounced) - before; got != 1 {
		t.Fatalf("compteur d'anti-rebond : +%d, attendu +1", got)
	}
	clock = clock.Add(2 * DefaultDebounce)
	w.Poll(ctx) // relancé
	w.Poll(ctx) // refermé bien après : synchro
	if len(syncs) != 2 {
		t.Fatalf("une fermeture après l'anti-rebond synchronise : %v", syncs)
	}
}

func TestReglageCoupe_PasDeModeJeuNiDeSynchronisation(t *testing.T) {
	ctx := context.Background()
	clock := time.Now()
	var syncs []string
	enabled := true
	w := newWatcher(&scriptedLister{steps: [][]string{{"HaloInfinite.exe"}}}, &clock, &syncs)
	w.Enabled = func() bool { return enabled }

	w.Poll(ctx)
	if !w.Mode.Running() {
		t.Fatal("réglage actif et jeu lancé : mode jeu")
	}
	enabled = false
	w.Poll(ctx)
	if w.Mode.Running() {
		t.Fatal("réglage coupé : le mode jeu retombe")
	}
	if len(syncs) != 0 {
		t.Fatalf("couper le réglage ne synchronise pas : %v", syncs)
	}
}

func TestEnumerationEnPanne_EtatConserveEtPanneJournaliseeUneFois(t *testing.T) {
	ctx := context.Background()
	clock := time.Now()
	var syncs []string
	boom := errors.New("instantané refusé")
	l := &scriptedLister{
		steps: [][]string{{"HaloInfinite.exe"}, nil, nil, {"explorer.exe"}},
		errs:  []error{nil, boom, boom, nil},
	}
	w := newWatcher(l, &clock, &syncs)
	before := observability.LoadCounter(MetricPollErrors)

	w.Poll(ctx)
	w.Poll(ctx)
	w.Poll(ctx)
	if !w.Mode.Running() {
		t.Fatal("une énumération en panne ne change pas l'état")
	}
	if got := observability.LoadCounter(MetricPollErrors) - before; got != 2 {
		t.Fatalf("pannes comptées : %d, attendu 2", got)
	}
	if w.lastErr != boom.Error() {
		t.Fatalf("la panne est retenue pour n'être journalisée qu'une fois : %q", w.lastErr)
	}
	w.Poll(ctx)
	if w.Mode.Running() || w.lastErr != "" {
		t.Fatal("l'énumération revenue rend l'état réel et efface la panne")
	}
	if len(syncs) != 1 {
		t.Fatalf("la fermeture constatée après la panne synchronise : %v", syncs)
	}
}

func TestTitreSansClientPC_JamaisDeJeu(t *testing.T) {
	clock := time.Now()
	var syncs []string
	w := newWatcher(&scriptedLister{steps: [][]string{{"HaloInfinite.exe"}}}, &clock, &syncs)
	w.Names = func() []string { return nil }
	w.Poll(context.Background())
	if w.Mode.Running() {
		t.Fatal("aucun nom déclaré : jamais de jeu détecté")
	}
}

func TestRun_SondeAuDemarrageEtAChaqueTic(t *testing.T) {
	clock := time.Now()
	var syncs []string
	ticks := make(chan time.Time)
	w := newWatcher(&scriptedLister{steps: [][]string{{"HaloInfinite.exe"}, {"explorer.exe"}}}, &clock, &syncs)
	w.Ticks = ticks
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { w.Run(ctx); close(done) }()

	ticks <- time.Now() // la première lecture est faite avant ce tic
	cancel()
	<-done
	if w.Mode.Running() {
		t.Fatal("deux lectures : jeu lancé puis fermé")
	}
	if len(syncs) != 1 {
		t.Fatalf("synchronisations : %v", syncs)
	}
}

func TestModeNil_ToutEstPermis(t *testing.T) {
	var m *Mode
	if m.Running() || !m.HeavyWorkAllowed() {
		t.Fatal("un mode nil n'est jamais actif")
	}
	if err := m.WaitHeavyAllowed(context.Background(), nil); err != nil {
		t.Fatalf("un mode nil ne fait jamais attendre : %v", err)
	}
}

func TestWaitHeavyAllowed_AttendLaFinDeLaPartieOuLAnnulation(t *testing.T) {
	m := NewMode()
	m.retryEvery = 5 * time.Millisecond
	m.set(true)
	deferred := 0
	before := observability.LoadCounter(MetricHeavyDeferred)

	go func() {
		time.Sleep(30 * time.Millisecond)
		m.set(false)
	}()
	if err := m.WaitHeavyAllowed(context.Background(), func() { deferred++ }); err != nil {
		t.Fatalf("la fin de la partie libère l'attente : %v", err)
	}
	if deferred != 1 {
		t.Fatalf("onDefer appelé %d fois, attendu 1", deferred)
	}
	if got := observability.LoadCounter(MetricHeavyDeferred) - before; got != 1 {
		t.Fatalf("différés comptés : %d", got)
	}

	m.set(true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.WaitHeavyAllowed(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("l'annulation interrompt l'attente : %v", err)
	}
}

// TestAvantLePremierReleve_LaPorteEstFermee : détection armée, aucun relevé encore fait — on ne sait
// pas si le jeu tourne, donc le travail lourd ne part pas ; le premier relevé, quelle qu'en soit
// l'issue (jeu absent, panne d'énumération, réglage coupé), rouvre la porte.
func TestAvantLePremierReleve_LaPorteEstFermee(t *testing.T) {
	names := func() []string { return []string{"HaloInfinite.exe"} }
	cases := []struct {
		name    string
		lister  Lister
		enabled func() bool
	}{
		{"jeu absent", ListerFunc(func() ([]string, error) { return []string{"explorer.exe"}, nil }), nil},
		{"énumération en panne", ListerFunc(func() ([]string, error) { return nil, errors.New("panne") }), nil},
		{"réglage coupé", ListerFunc(func() ([]string, error) { return []string{"HaloInfinite.exe"}, nil }), func() bool { return false }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := NewMode()
			m.ExpectFirstPoll()
			if m.HeavyWorkAllowed() {
				t.Fatal("avant le premier relevé, le travail lourd doit être refusé")
			}
			w := &Watcher{Lister: c.lister, Mode: m, Names: names, Enabled: c.enabled}
			w.Poll(context.Background())
			if !m.HeavyWorkAllowed() {
				t.Fatal("après le premier relevé (jeu absent), le travail lourd doit être permis")
			}
		})
	}
}

// TestAvantLePremierReleve_JeuDejaOuvertResteRefuse : le jeu tourne à l'ouverture de l'application ;
// le premier relevé le voit et la porte reste fermée, sans fenêtre où le travail lourd partirait.
func TestAvantLePremierReleve_JeuDejaOuvertResteRefuse(t *testing.T) {
	m := NewMode()
	m.ExpectFirstPoll()
	w := &Watcher{
		Lister: ListerFunc(func() ([]string, error) { return []string{"HaloInfinite.exe"}, nil }),
		Mode:   m,
		Names:  func() []string { return []string{"HaloInfinite.exe"} },
	}
	w.Poll(context.Background())
	if m.HeavyWorkAllowed() {
		t.Fatal("jeu déjà ouvert : le travail lourd doit rester refusé")
	}
}

func TestModeNil_ExpectFirstPollSansEffet(t *testing.T) {
	var m *Mode
	m.ExpectFirstPoll()
	if !m.HeavyWorkAllowed() {
		t.Fatal("un mode nil permet tout")
	}
}

// TestFinDePartie_CompteursSelonLIssueDeLaDemande : « déclenchée » ne se compte et ne se dit que si une
// synchronisation va avoir lieu ; auto-sync coupée = compteur « ignorée », et l'anti-rebond ne
// s'arme pas sur une demande qui n'a rien lancé.
func TestFinDePartie_CompteursSelonLIssueDeLaDemande(t *testing.T) {
	ctx := context.Background()
	clock := time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC)
	outcome := SyncDisabled
	calls := 0
	run, off := []string{"HaloInfinite.exe"}, []string{"explorer.exe"}
	w := &Watcher{
		Lister: &scriptedLister{steps: [][]string{run, off, run, off, run, off}},
		Mode:   NewMode(),
		Names:  func() []string { return []string{"HaloInfinite.exe"} },
		OnGameClosed: func(string) SyncOutcome {
			calls++
			return outcome
		},
		Now: func() time.Time { return clock },
	}
	trig := observability.LoadCounter(MetricSyncTriggered)
	skip := observability.LoadCounter(MetricSyncSkipped)

	w.Poll(ctx) // lancé
	w.Poll(ctx) // fermé, auto-sync coupée
	if d := observability.LoadCounter(MetricSyncTriggered) - trig; d != 0 {
		t.Fatalf("auto-sync coupée : aucune synchronisation déclenchée, compteur +%d", d)
	}
	if d := observability.LoadCounter(MetricSyncSkipped) - skip; d != 1 {
		t.Fatalf("auto-sync coupée : compteur « ignorée » +%d, attendu +1", d)
	}
	// Réglage rallumé 5 s plus tard : pas d'anti-rebond hérité d'une demande qui n'a rien lancé.
	outcome = SyncRequested
	clock = clock.Add(5 * time.Second)
	w.Poll(ctx)
	w.Poll(ctx)
	if calls != 2 {
		t.Fatalf("la seconde fermeture doit appeler la demande (appels=%d)", calls)
	}
	if d := observability.LoadCounter(MetricSyncTriggered) - trig; d != 1 {
		t.Fatalf("synchronisation lancée : compteur +%d, attendu +1", d)
	}
	// Demande couverte par une autre déjà en attente : ni « déclenchée » ni « ignorée ».
	outcome = SyncCovered
	clock = clock.Add(2 * DefaultDebounce)
	w.Poll(ctx)
	w.Poll(ctx)
	if d := observability.LoadCounter(MetricSyncTriggered) - trig; d != 1 {
		t.Fatalf("demande couverte : le compteur ne bouge pas (+%d)", d)
	}
}
