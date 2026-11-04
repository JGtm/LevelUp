package filmrun

// runner_annulation_test.go — L'ANNULATION DU CONTEXTE TUE L'ENFANT.
//
// La cuisson d'un film tourne dans un enfant (`exec.CommandContext`) sous le contexte du sync, lui-même
// dérivé du contexte d'arrêt du serveur : à l'arrêt, l'enfant doit mourir et `Run` rendre la main
// aussitôt, sans attendre la fin du décodage (ADR 0045 D-4, lot L12.3). L'enfant est le binaire de
// test lui-même (même patron que runner_child_test.go).

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// dortAideEnfant : l'enfant annonce son identifiant puis dort. Faux : ce processus n'est pas un
// enfant, et [TestAideEnfantDort] ne fait rien.
var dortAideEnfant = flag.Bool("filmproc-aide-dort", false,
	"usage interne des tests : l'enfant re-exécuté annonce son pid puis dort")

// TestAideEnfantDort EST L'ENFANT qui ne finit jamais de lui-même.
func TestAideEnfantDort(t *testing.T) {
	if !*dortAideEnfant {
		t.Skip("pas un enfant : ce cas ne s'exécute que re-exécuté par TestRunAnnuleTueLEnfant")
	}
	fmt.Printf("pid=%d\n", os.Getpid())
	time.Sleep(time.Hour)
}

// journalSur est un io.Writer sûr en concurrence : le test lit le journal pendant que le lanceur
// l'écrit.
type journalSur struct {
	mu sync.Mutex
	sb strings.Builder
}

func (j *journalSur) Write(p []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.sb.Write(p)
}

func (j *journalSur) String() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.sb.String()
}

// pidAnnonce attend la ligne `pid=N` de l'enfant dans le journal.
func pidAnnonce(t *testing.T, j *journalSur) int {
	t.Helper()
	limite := time.Now().Add(30 * time.Second)
	for time.Now().Before(limite) {
		for _, ligne := range strings.Split(j.String(), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(ligne), "pid="); ok {
				pid, err := strconv.Atoi(v)
				if err != nil {
					t.Fatalf("pid illisible %q : %v", v, err)
				}
				return pid
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("l'enfant n'a pas annoncé son pid en 30 s ; journal :\n%s", j.String())
	return 0
}

// TestRunAnnuleTueLEnfant : annuler le contexte pendant que l'enfant vit le tue, `Run` rend la main
// sans attendre l'heure de sommeil, et le processus n'existe plus.
func TestRunAnnuleTueLEnfant(t *testing.T) {
	var journal journalSur
	r, err := NewRunner(t.TempDir(), &journal)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	ctx, annuler := context.WithCancel(context.Background())
	defer annuler()

	fini := make(chan Result, 1)
	go func() {
		fini <- r.Run(ctx, []string{"-test.run=^TestAideEnfantDort$", "-filmproc-aide-dort"})
	}()
	pid := pidAnnonce(t, &journal)
	if !processAlive(pid) {
		t.Fatalf("l'enfant %d n'existe pas avant l'annulation", pid)
	}

	debut := time.Now()
	annuler()
	select {
	case res := <-fini:
		if res.Issue == IssueOK {
			t.Errorf("issue %v : un enfant tué n'est pas un succès", res.Issue)
		}
		t.Logf("Run rendu %s après l'annulation (issue %v)", time.Since(debut).Round(time.Millisecond), res.Issue)
	case <-time.After(15 * time.Second):
		t.Fatal("Run n'a pas rendu la main 15 s après l'annulation : l'enfant n'est pas tué à l'arrêt")
	}

	limite := time.Now().Add(10 * time.Second)
	for processAlive(pid) {
		if time.Now().After(limite) {
			t.Fatalf("l'enfant %d vit encore après l'annulation", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
