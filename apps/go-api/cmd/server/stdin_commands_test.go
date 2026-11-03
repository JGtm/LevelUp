package main

import (
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// recordCommands branche un enregistreur sur les commandes données.
func recordCommands(names ...string) (*stdinCommands, *[]string, *sync.Mutex) {
	var mu sync.Mutex
	var got []string
	c := &stdinCommands{}
	for _, n := range names {
		c.Handle(n, func() {
			mu.Lock()
			defer mu.Unlock()
			got = append(got, n)
		})
	}
	return c, &got, &mu
}

// TestDrainStdin_ExecuteChaqueLigneEtRendLaMainAEOF : une ligne = une commande, la dernière peut ne pas
// finir par un saut de ligne, l'EOF rend nil (c'est l'ordre d'arrêt, décidé par l'appelant).
func TestDrainStdin_ExecuteChaqueLigneEtRendLaMainAEOF(t *testing.T) {
	cmds, got, mu := recordCommands("sync")
	in := "sync\r\n\n  sync  \nsync"
	if err := drainStdin(context.Background(), strings.NewReader(in), cmds); err != nil {
		t.Fatalf("EOF doit rendre nil : %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*got) != 3 {
		t.Fatalf("%d commandes exécutées, attendu 3 (CRLF, espaces et dernière ligne sans saut compris) : %v", len(*got), *got)
	}
}

// TestDrainStdin_InconnueOuTropLongue : une commande inconnue est ignorée ; une ligne sans fin ne fait
// grossir aucun tampon et n'exécute rien ; la commande qui suit est exécutée.
func TestDrainStdin_InconnueOuTropLongue(t *testing.T) {
	cmds, got, mu := recordCommands("sync")
	long := strings.Repeat("x", 10*maxStdinLine)
	in := "inconnue\n" + long + "\nsync\n" + long // dernière ligne trop longue, sans saut : écartée
	if err := drainStdin(context.Background(), strings.NewReader(in), cmds); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*got) != 1 || (*got)[0] != "sync" {
		t.Fatalf("exécutées = %v, attendu seulement sync", *got)
	}
}

type brokenReader struct{ lines string }

func (b *brokenReader) Read(p []byte) (int, error) {
	if b.lines != "" {
		n := copy(p, b.lines)
		b.lines = b.lines[n:]
		return n, nil
	}
	return 0, io.ErrClosedPipe
}

// TestDrainStdin_ErreurDeLecture : une erreur de lecture est rendue à l'appelant, les lignes lues avant sont exécutées.
func TestDrainStdin_ErreurDeLecture(t *testing.T) {
	cmds, got, mu := recordCommands("sync")
	err := drainStdin(context.Background(), &brokenReader{lines: "sync\n"}, cmds)
	if err == nil {
		t.Fatal("l'erreur de lecture doit être rendue")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*got) != 1 {
		t.Fatalf("la ligne lue avant l'erreur est exécutée : %v", *got)
	}
}

// TestWatchStdin_UneCommandeNEstPasUnOrdreDArret : `sync` s'exécute sans arrêter le serveur ; seule la
// fermeture de stdin demande l'arrêt, une seule fois.
func TestWatchStdin_UneCommandeNEstPasUnOrdreDArret(t *testing.T) {
	b := newTestBootServer()
	ran := make(chan struct{}, 1)
	b.commands.Handle(stdinCommandSync, func() { ran <- struct{}{} })
	r, w := io.Pipe()
	b.watchStdin(r)

	if _, err := io.WriteString(w, "sync\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("la commande sync n'a pas été exécutée")
	}
	select {
	case <-b.stopRequested():
		t.Fatal("une commande ne doit pas demander l'arrêt")
	case <-time.After(50 * time.Millisecond):
	}
	_ = w.Close()
	waitClosed(t, b.stopRequested(), "arrêt demandé par EOF stdin")
	if got := b.shutdownReason(); got != shutdownReasonStdin {
		t.Errorf("cause = %q, attendu %q", got, shutdownReasonStdin)
	}
}
