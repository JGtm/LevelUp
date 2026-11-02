package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/notifications"
	"levelup/go-api/internal/notifications/native"
	"levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/scheduler"
)

// TestNativeRelay_DeLaCloche_A_LEvenementStdout : une notification de la cloche, réglage actif, sort
// sur stdout en UNE ligne `notification` (titre et corps composés dans la langue du joueur, type).
func TestNativeRelay_DeLaCloche_A_LEvenementStdout(t *testing.T) {
	var out bytes.Buffer
	events := newDesktopEvents(&out)
	relay := &native.Relay{}
	cfg := settings.Defaults()
	cfg.NativeNotifications = true
	cfg.Lang = "en"
	installNativeRelay(relay, events, fakeSettings{cfg: cfg})

	// Le même chemin que le service de notification : le relais derrière un Fanout.
	notifications.Fanout(relay).Forward(context.Background(),
		&notifications.Notification{Category: notifications.CategoryMatchSynced})

	lines := lignes(&out)
	if len(lines) != 1 {
		t.Fatalf("%d lignes sur stdout, attendu 1 :\n%s", len(lines), out.String())
	}
	var ev struct {
		Event string `json:"event"`
		Title string `json:"title"`
		Body  string `json:"body"`
		Kind  string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("ligne illisible %q : %v", lines[0], err)
	}
	if ev.Event != "notification" || ev.Kind != "sync" || ev.Title != "Sync complete" || ev.Body == "" {
		t.Fatalf("événement = %+v", ev)
	}
}

// TestNativeRelay_ReglageCoupe_RienSurStdout : par défaut (réglage coupé) aucune ligne n'est écrite.
func TestNativeRelay_ReglageCoupe_RienSurStdout(t *testing.T) {
	var out bytes.Buffer
	relay := &native.Relay{}
	installNativeRelay(relay, newDesktopEvents(&out), fakeSettings{cfg: settings.Defaults()})
	relay.Forward(context.Background(), &notifications.Notification{Category: notifications.CategoryMatchSynced})
	if out.Len() != 0 {
		t.Fatalf("réglage coupé : %q écrit sur stdout", out.String())
	}
}

// recordingShell enregistre les réglages écrits.
type recordingShell struct {
	mu  sync.Mutex
	got []shellSettings
}

func (r *recordingShell) ShellSettings(s shellSettings) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = append(r.got, s)
}

func (r *recordingShell) all() []shellSettings {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]shellSettings(nil), r.got...)
}

// mutableSettings : des réglages que le test change entre deux tics.
type mutableSettings struct {
	mu  sync.Mutex
	cfg *settings.AppSettings
	err error
}

func (m *mutableSettings) Load() (*settings.AppSettings, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg, m.err
}

func (m *mutableSettings) set(cfg *settings.AppSettings, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg, m.err = cfg, err
}

// TestShellSettings_EcritsAuDemarrageEtASeulementChaqueChangement : la coquille reçoit les réglages une
// première fois, puis seulement quand ils changent ; des réglages illisibles ne changent rien.
func TestShellSettings_EcritsAuDemarrageEtASeulementChaqueChangement(t *testing.T) {
	store := &mutableSettings{cfg: settings.Defaults()}
	w := &recordingShell{}
	ticks := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { runShellSettingsLoop(ctx, store, w, ticks, 0); close(done) }()

	waitFor(t, func() bool { return len(w.all()) == 1 }, "l'écriture initiale")
	if first := w.all()[0]; first.CloseToTray || first.NativeNotifications || first.StartWithWindows || first.Locale != "en" {
		t.Fatalf("réglages initiaux = %+v, attendu tout coupé et la langue par défaut", first)
	}

	ticks <- time.Now() // rien n'a changé : rien n'est écrit
	cfg := settings.Defaults()
	cfg.CloseToTray, cfg.Lang = true, "fr"
	store.set(cfg, nil)
	ticks <- time.Now()
	waitFor(t, func() bool { return len(w.all()) == 2 }, "l'écriture du changement")
	if got := w.all()[1]; !got.CloseToTray || got.Locale != "fr" {
		t.Fatalf("après changement = %+v", got)
	}

	store.set(nil, errors.New("fichier illisible"))
	ticks <- time.Now()
	store.set(cfg, nil)
	ticks <- time.Now() // revenu identique : rien n'est réécrit
	ticks <- time.Now()
	cancel()
	<-done
	if n := len(w.all()); n != 2 {
		t.Fatalf("%d écritures, attendu 2 : ni la panne ni le retour à l'identique ne s'écrivent", n)
	}
}

func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("délai dépassé en attendant : %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// fakeManualSyncer compte les cycles manuels ; `release` les retient jusqu'à ce que le test les libère.
type fakeManualSyncer struct {
	calls    atomic.Int32
	release  chan struct{}
	triggers chan string
}

func (f *fakeManualSyncer) RunOnceTrigger(_ context.Context, trigger string) *scheduler.RunOnceResult {
	f.calls.Add(1)
	f.triggers <- trigger
	<-f.release
	return &scheduler.RunOnceResult{Total: 1, Synced: 1}
}

// TestManualSync_UnCycleManuelALaFoisEtSuiviParLeGroupeDAttente : la commande lance le déclencheur manuel
// de l'ordonnanceur, jamais deux cycles en même temps, et l'arrêt attend le cycle.
func TestManualSync_UnCycleManuelALaFoisEtSuiviParLeGroupeDAttente(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st := &bootState{schedulerCtx: ctx}
	f := &fakeManualSyncer{release: make(chan struct{}), triggers: make(chan string, 4)}
	run := newManualSync(st, f)

	run()
	if got := <-f.triggers; got != "manual" {
		t.Fatalf("déclencheur = %q, attendu manual", got)
	}
	run() // un cycle tourne déjà : ignoré
	run()
	close(f.release)
	waitFor(t, func() bool {
		done := make(chan struct{})
		go func() { st.schedulerWG.Wait(); close(done) }()
		select {
		case <-done:
			return true
		case <-time.After(20 * time.Millisecond):
			return false
		}
	}, "la fin du cycle (groupe d'attente rendu)")
	if n := f.calls.Load(); n != 1 {
		t.Fatalf("%d cycles lancés, attendu 1", n)
	}

	// Le cycle fini, une nouvelle commande en lance un autre ; arrêt demandé : plus aucun.
	run()
	waitFor(t, func() bool { return f.calls.Load() == 2 }, "le second cycle")
	cancel()
	run()
	if n := f.calls.Load(); n != 2 {
		t.Fatalf("un cycle a été lancé pendant l'arrêt : %d", n)
	}
	st.schedulerWG.Wait()
}
