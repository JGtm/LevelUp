// desktop_events_test.go — le contrat des événements stdout (ADR 0045 D-2) : une ligne JSON par
// événement, jamais entrelacée, et rien du tout hors -mode desktop.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"levelup/go-api/internal/config"
)

// syncBuf est un tampon sûr en concurrence : le test lit pendant que le serveur écrit.
type syncBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// lignes rend les lignes écrites dans buf.
func lignes(buf fmt.Stringer) []string {
	out := strings.TrimRight(buf.String(), "\n")
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

func TestDesktopEvents_FormesExactes(t *testing.T) {
	var buf bytes.Buffer
	e := newDesktopEvents(&buf)
	e.Listening(51234)
	e.Ready(51234)
	e.Failed(bootFailureLocked)
	e.Failed(bootFailureMigrationFailed)
	e.Failed(bootFailureSchemaNewer)
	e.Failed(bootFailurePort)
	e.Stopping()
	e.Stopped(true)
	e.Stopped(false)
	e.Notification("Synchronisation terminée", "Tes dernières parties sont à jour.", "sync")
	e.ShellSettings(shellSettings{CloseToTray: true, Locale: "fr"})

	want := []string{
		`{"event":"listening","port":51234}`,
		`{"event":"ready","port":51234}`,
		`{"event":"failed","state":"locked"}`,
		`{"event":"failed","state":"migration_failed"}`,
		`{"event":"failed","state":"schema_newer"}`,
		`{"event":"failed","state":"port"}`,
		`{"event":"stopping"}`,
		`{"event":"stopped","clean":true}`,
		`{"event":"stopped","clean":false}`,
		`{"event":"notification","title":"Synchronisation terminée","body":"Tes dernières parties sont à jour.","kind":"sync"}`,
		`{"event":"shell_settings","native_notifications":false,"close_to_tray":true,"start_with_windows":false,"locale":"fr"}`,
	}
	got := lignes(&buf)
	if len(got) != len(want) {
		t.Fatalf("%d lignes, attendu %d :\n%s", len(got), len(want), buf.String())
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ligne %d = %s, attendu %s", i, got[i], want[i])
		}
	}
}

// Un écrivain désactivé (nil, hors -mode desktop) ne fait rien et ne panique pas.
func TestDesktopEvents_NilNeFaitRien(t *testing.T) {
	var e *desktopEvents
	e.Listening(1)
	e.Ready(1)
	e.Failed(bootFailureLocked)
	e.Stopping()
	e.Stopped(true)
}

// Deux goroutines qui écrivent en même temps ne mélangent jamais leurs octets : chaque ligne reste
// un JSON entier.
func TestDesktopEvents_UneLigneJamaisEntrelacee(t *testing.T) {
	var buf bytes.Buffer
	e := newDesktopEvents(&buf)
	var wg sync.WaitGroup
	const par, n = 8, 200
	for g := 0; g < par; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < n; i++ {
				e.Listening(40000 + i)
			}
		}()
	}
	wg.Wait()
	got := lignes(&buf)
	if len(got) != par*n {
		t.Fatalf("%d lignes, attendu %d", len(got), par*n)
	}
	for _, l := range got {
		var ev struct {
			Event string `json:"event"`
			Port  int    `json:"port"`
		}
		if err := json.Unmarshal([]byte(l), &ev); err != nil || ev.Event != eventListening || ev.Port < 40000 {
			t.Fatalf("ligne entrelacée ou illisible : %q (%v)", l, err)
		}
	}
}

// refuseStart annonce `failed` avec l'état posé sur la porte, avant de garder le port.
func TestRefuseStart_AnnonceFailed(t *testing.T) {
	var buf syncBuf
	b, sigCh, _ := startSignalWatch(t)
	b.events = newDesktopEvents(&buf)

	done := make(chan int, 1)
	go func() { done <- b.refuseStart(t.Context(), bootFailureLocked, os.ErrExist) }()
	waitFailed(t, b.gate)
	deadline := time.Now().Add(2 * time.Second)
	for buf.String() == "" && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := strings.TrimSpace(buf.String()); got != `{"event":"failed","state":"locked"}` {
		t.Errorf("événement = %q, attendu failed locked", got)
	}
	sigCh <- os.Interrupt
	<-done
}

// openRouter annonce `ready` sur le port lié, une fois le routeur branché ; un arrêt demandé avant
// l'ouverture n'annonce rien.
func TestOpenRouter_AnnonceReadySurLePortLie(t *testing.T) {
	g := newBootGate(&config.AppConfig{})
	var buf bytes.Buffer
	b := newBootServer(&http.Server{Handler: g}, g, func(int) {})
	b.events, b.port = newDesktopEvents(&buf), 51999

	b.openRouter(context.Background(), routerStub())
	if got := strings.TrimSpace(buf.String()); got != `{"event":"ready","port":51999}` {
		t.Errorf("événement = %q, attendu ready 51999", got)
	}

	var stopped bytes.Buffer
	b2 := newBootServer(&http.Server{Handler: g}, newBootGate(&config.AppConfig{}), func(int) {})
	b2.events, b2.port = newDesktopEvents(&stopped), 51999
	b2.requestShutdown("test")
	b2.openRouter(context.Background(), routerStub())
	if stopped.Len() != 0 {
		t.Errorf("ready annoncé alors que le routeur n'est pas branché : %q", stopped.String())
	}
}
