package filelock

// filelock_test.go — LE VERROU EST CELUI DU NOYAU : exclusif entre detenteurs, rendu a la mort
// du detenteur, sans battement de coeur ni reprise de verrou perime.
//
// LE BINAIRE DE TEST EST SON PROPRE ENFANT (modele `filmproc/runner_child_test.go`) : quand la
// variable [envEnfant] porte un chemin, [TestMain] ne lance aucun test — le processus prend le
// verrou, l'annonce sur sa sortie standard et attend d'etre tue. Aucun binaire externe.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// envEnfant : chemin du verrou que l'enfant doit tenir. Vide = ce processus n'est pas un enfant.
const envEnfant = "LEVELUP_FILELOCK_ENFANT"

// annonceEnfant : la ligne que l'enfant ecrit une fois le verrou tenu.
const annonceEnfant = "verrou-tenu"

func TestMain(m *testing.M) {
	if chemin := os.Getenv(envEnfant); chemin != "" {
		os.Exit(enfantTientLeVerrou(chemin))
	}
	os.Exit(m.Run())
}

// enfantTientLeVerrou : le corps de l'enfant. Il ne rend la main que s'il n'a pas pu prendre le
// verrou ; sinon il dort jusqu'a ce que le parent le tue.
func enfantTientLeVerrou(chemin string) int {
	l, err := TryLock(chemin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "enfant :", err)
		return 2
	}
	fmt.Println(annonceEnfant)
	time.Sleep(time.Hour)
	_ = l.Unlock()
	return 0
}

func TestTryLock_SecondDetenteurRefuse(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "decode.lock")
	premier, err := TryLock(chemin)
	if err != nil {
		t.Fatalf("premier verrou : %v", err)
	}
	defer func() { _ = premier.Unlock() }()

	second, err := TryLock(chemin)
	if err == nil {
		_ = second.Unlock()
		t.Fatal("un second detenteur a obtenu le verrou tenu par le premier")
	}
	if !errors.Is(err, ErrLocked) {
		t.Errorf("err = %v, attendu ErrLocked", err)
	}
}

// TestTryLock_LibereALaMortDuProcessus — le cas nominal du verrou de decodage : le detenteur est
// TUE (sentinelle memoire, operateur). Le noyau rend le verrou ; aucune reprise manuelle.
func TestTryLock_LibereALaMortDuProcessus(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "decode.lock")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	enfant := exec.Command(exe, "-test.run=^$") //nolint:gosec // le binaire de test lui-meme
	enfant.Env = append(os.Environ(), envEnfant+"="+chemin)
	sortie, err := enfant.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	enfant.Stderr = os.Stderr
	if err := enfant.Start(); err != nil {
		t.Fatalf("lancement de l'enfant : %v", err)
	}
	tue := false
	defer func() {
		if !tue {
			_ = enfant.Process.Kill()
			_ = enfant.Wait()
		}
	}()
	attendreAnnonce(t, sortie)

	if l, err := TryLock(chemin); !errors.Is(err, ErrLocked) {
		if l != nil {
			_ = l.Unlock()
		}
		t.Fatalf("verrou tenu par l'enfant vivant : err = %v, attendu ErrLocked", err)
	}
	if err := enfant.Process.Kill(); err != nil {
		t.Fatalf("kill de l'enfant : %v", err)
	}
	_ = enfant.Wait()
	tue = true

	// Le noyau rend le verrou a la fin du processus ; Windows peut le faire avec un leger delai.
	limite := time.Now().Add(10 * time.Second)
	for {
		l, err := TryLock(chemin)
		if err == nil {
			_ = l.Unlock()
			return
		}
		if !errors.Is(err, ErrLocked) || time.Now().After(limite) {
			t.Fatalf("verrou non rendu apres la mort du detenteur : %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// attendreAnnonce lit la sortie de l'enfant jusqu'a l'annonce du verrou tenu.
func attendreAnnonce(t *testing.T, sortie io.Reader) {
	t.Helper()
	lu := make(chan bool, 1)
	go func() {
		sc := bufio.NewScanner(sortie)
		for sc.Scan() {
			if strings.TrimSpace(sc.Text()) == annonceEnfant {
				lu <- true
				return
			}
		}
		lu <- false
	}()
	select {
	case ok := <-lu:
		if !ok {
			t.Fatal("l'enfant s'est arrete sans tenir le verrou")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("l'enfant n'a pas annonce le verrou en 30 s")
	}
}

func TestUnlock_Idempotent(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "decode.lock")
	l, err := TryLock(chemin)
	if err != nil {
		t.Fatalf("verrou : %v", err)
	}
	if err := l.Unlock(); err != nil {
		t.Fatalf("premier Unlock : %v", err)
	}
	if err := l.Unlock(); err != nil {
		t.Errorf("second Unlock : %v, attendu nil (idempotent)", err)
	}
	suivant, err := TryLock(chemin)
	if err != nil {
		t.Fatalf("verrou non rendu par Unlock : %v", err)
	}
	_ = suivant.Unlock()
}
