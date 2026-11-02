package instancelock

// instancelock_test.go : LE VERROU EST CELUI DU NOYAU. Exclusif entre processus, rendu à la mort du
// détenteur, sans battement de coeur ni reprise de verrou périmé.
//
// LE BINAIRE DE TEST EST SON PROPRE ENFANT : quand la variable envEnfant porte un chemin, TestMain
// ne lance aucun test. Le processus prend le verrou, l'annonce sur sa sortie standard et attend
// d'être tué. Aucun binaire externe.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// envEnfant : chemin du verrou que l'enfant doit tenir. Vide : ce processus n'est pas un enfant.
const envEnfant = "LEVELUP_INSTANCELOCK_ENFANT"

// annonceEnfant : la ligne que l'enfant écrit une fois le verrou tenu.
const annonceEnfant = "verrou-tenu"

func TestMain(m *testing.M) {
	if chemin := os.Getenv(envEnfant); chemin != "" {
		os.Exit(enfantTientLeVerrou(chemin))
	}
	os.Exit(m.Run())
}

// enfantTientLeVerrou : le corps de l'enfant. Il ne rend la main que s'il n'a pas pu prendre le
// verrou ; sinon il dort jusqu'à ce que le parent le tue.
func enfantTientLeVerrou(chemin string) int {
	l, err := Acquire(chemin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "enfant :", err)
		return 2
	}
	fmt.Println(annonceEnfant)
	time.Sleep(time.Hour)
	_ = l.Release()
	return 0
}

// lanceEnfant démarre le binaire de test comme détenteur du verrou et attend son annonce. Le
// détenteur est tué en fin de test s'il vit encore.
func lanceEnfant(t *testing.T, chemin string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	enfant := exec.Command(exe, "-test.run=^$") //nolint:gosec // le binaire de test lui-même
	enfant.Env = append(os.Environ(), envEnfant+"="+chemin)
	sortie, err := enfant.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	enfant.Stderr = os.Stderr
	if err := enfant.Start(); err != nil {
		t.Fatalf("lancement de l'enfant : %v", err)
	}
	t.Cleanup(func() {
		_ = enfant.Process.Kill()
		_ = enfant.Wait()
	})
	annonce := make(chan bool, 1)
	go func() {
		sc := bufio.NewScanner(sortie)
		for sc.Scan() {
			if strings.TrimSpace(sc.Text()) == annonceEnfant {
				annonce <- true
				return
			}
		}
		annonce <- false
	}()
	select {
	case ok := <-annonce:
		if !ok {
			t.Fatal("l'enfant s'est arrêté sans tenir le verrou")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("l'enfant n'a pas annoncé le verrou en 30 s")
	}
	return enfant
}

func TestAcquire_SecondDetenteurRefuse(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "levelup.lock")
	premier, err := Acquire(chemin)
	if err != nil {
		t.Fatalf("premier verrou : %v", err)
	}
	defer func() { _ = premier.Release() }()

	second, err := Acquire(chemin)
	if err == nil {
		_ = second.Release()
		t.Fatal("un second détenteur a obtenu le verrou tenu par le premier")
	}
	if !errors.Is(err, ErrInstanceDejaLancee) {
		t.Errorf("err = %v, attendu ErrInstanceDejaLancee", err)
	}
}

func TestAcquire_CreeLeDossierManquant(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "absent", "levelup.lock")
	l, err := Acquire(chemin)
	if err != nil {
		t.Fatalf("verrou sous un dossier absent : %v", err)
	}
	_ = l.Release()
}

// TestAcquire_DeuxProcessus : le détenteur est un AUTRE processus. Tant qu'il vit, le verrou est
// refusé avec l'erreur typée ; tué, le noyau le rend sans reprise manuelle.
func TestAcquire_DeuxProcessus(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "levelup.lock")
	enfant := lanceEnfant(t, chemin)

	if l, err := Acquire(chemin); !errors.Is(err, ErrInstanceDejaLancee) {
		if l != nil {
			_ = l.Release()
		}
		t.Fatalf("verrou tenu par un autre processus vivant : err = %v, attendu ErrInstanceDejaLancee", err)
	}

	if err := enfant.Process.Kill(); err != nil {
		t.Fatalf("kill de l'enfant : %v", err)
	}
	_ = enfant.Wait()

	// Le noyau rend le verrou à la fin du processus ; Windows peut le faire avec un léger délai.
	limite := time.Now().Add(10 * time.Second)
	for {
		l, err := Acquire(chemin)
		if err == nil {
			_ = l.Release()
			return
		}
		if !errors.Is(err, ErrInstanceDejaLancee) || time.Now().After(limite) {
			t.Fatalf("verrou non rendu après la mort du détenteur : %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestRelease_Idempotent(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "levelup.lock")
	l, err := Acquire(chemin)
	if err != nil {
		t.Fatalf("verrou : %v", err)
	}
	if err := l.Release(); err != nil {
		t.Fatalf("premier Release : %v", err)
	}
	if err := l.Release(); err != nil {
		t.Errorf("second Release : %v, attendu nil (idempotent)", err)
	}
	suivant, err := Acquire(chemin)
	if err != nil {
		t.Fatalf("verrou non rendu par Release : %v", err)
	}
	_ = suivant.Release()

	var nul *Lock
	if err := nul.Release(); err != nil {
		t.Errorf("Release sur un *Lock nil : %v", err)
	}
}
