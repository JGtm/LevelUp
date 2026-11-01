package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/config"
)

func newTestBootServer() *bootServer {
	g := newBootGate(&config.AppConfig{})
	return newBootServer(&http.Server{Handler: g, ReadHeaderTimeout: time.Second}, g, func(int) {})
}

// errReader échoue au premier Read : la lecture de stdin en erreur vaut fermeture.
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("tube cassé") }

func TestWatchStdin_EOFDemandeLArretUneFois(t *testing.T) {
	b := newTestBootServer()
	r, w := io.Pipe()
	b.watchStdin(r)

	select {
	case <-b.stopRequested():
		t.Fatal("arrêt demandé alors que stdin est encore ouvert")
	case <-time.After(50 * time.Millisecond):
	}
	if _, err := w.Write([]byte("du bruit")); err != nil { // les octets lus ne comptent pas
		t.Fatal(err)
	}
	select {
	case <-b.stopRequested():
		t.Fatal("arrêt demandé par des octets reçus : seule la fermeture compte")
	case <-time.After(50 * time.Millisecond):
	}

	_ = w.Close() // EOF
	waitClosed(t, b.stopRequested(), "arrêt demandé par EOF stdin")
	if got := b.shutdownReason(); got != shutdownReasonStdin {
		t.Errorf("cause = %q, attendu %q", got, shutdownReasonStdin)
	}
}

func TestWatchStdin_ErreurDeLectureDemandeLArret(t *testing.T) {
	b := newTestBootServer()
	b.watchStdin(errReader{})
	waitClosed(t, b.stopRequested(), "arrêt demandé par erreur de lecture")
	if got := b.shutdownReason(); got != shutdownReasonStdin {
		t.Errorf("cause = %q, attendu %q", got, shutdownReasonStdin)
	}
}

// Plusieurs sources en même temps (signal, stdin, HTTP) : un seul appel déclenche l'arrêt,
// la cause est celle de ce premier appel et les suivants ne paniquent pas sur le canal fermé.
func TestRequestShutdown_UnSeulPassage(t *testing.T) {
	b := newTestBootServer()
	var first atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for _, reason := range []string{"signal interrupt", shutdownReasonStdin, "http", shutdownReasonStdin, "http"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if b.requestShutdown(reason) {
				first.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	if n := first.Load(); n != 1 {
		t.Fatalf("%d appels ont déclenché l'arrêt, attendu exactement 1", n)
	}
	waitClosed(t, b.stopRequested(), "arrêt demandé")
	reason := b.shutdownReason()
	if reason == "" {
		t.Error("cause vide après une demande d'arrêt")
	}
	b.requestShutdown("autre")
	if got := b.shutdownReason(); got != reason {
		t.Errorf("cause changée par une demande tardive : %q -> %q", reason, got)
	}
}

// Signal puis stdin : une seule séquence d'arrêt, sans sortie forcée, cause = première demande.
func TestRequestShutdown_SignalPuisStdin(t *testing.T) {
	b, sigCh, exits := startSignalWatch(t)
	b.openRouter(context.Background(), routerStub()) // initialisation finie : pas de sortie forcée

	sigCh <- os.Interrupt
	waitClosed(t, b.stopRequested(), "arrêt demandé par signal")
	b.watchStdin(strings.NewReader("")) // EOF immédiat : seconde demande
	time.Sleep(50 * time.Millisecond)

	if got := b.shutdownReason(); !strings.HasPrefix(got, shutdownReasonSignal) {
		t.Errorf("cause = %q, attendu celle du signal (première demande)", got)
	}
	select {
	case code := <-exits:
		t.Errorf("sortie forcée (%d) sur un double déclenchement", code)
	default:
	}
}

// Stdin d'abord, signal ensuite pendant l'initialisation : le signal garde son sens de premier
// signal (arrêt propre différé, pas de sortie forcée).
func TestRequestShutdown_StdinPuisSignalPendantInit(t *testing.T) {
	b, sigCh, exits := startSignalWatch(t)
	b.watchStdin(strings.NewReader(""))
	waitClosed(t, b.stopRequested(), "arrêt demandé par stdin")

	sigCh <- os.Interrupt
	select {
	case code := <-exits:
		t.Errorf("sortie forcée (%d) sur un premier signal", code)
	case <-time.After(100 * time.Millisecond):
	}
	if got := b.shutdownReason(); got != shutdownReasonStdin {
		t.Errorf("cause = %q, attendu %q", got, shutdownReasonStdin)
	}
}
