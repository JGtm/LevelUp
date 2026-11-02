// desktop_port_test.go — le port stable du mode desktop (ADR 0045 D-9) : repris s'il est libre,
// sinon remplacé par un port libre écrit dans le fichier, avec un avertissement.
package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// portLibre rend un port libre de la boucle locale (libéré avant le retour).
func portLibre(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return listenerPort(ln)
}

func lirePort(t *testing.T, chemin string) string {
	t.Helper()
	raw, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("fichier de port illisible : %v", err)
	}
	return strings.TrimSpace(string(raw))
}

func TestBindDesktopPort_PremierLancementEcritLePort(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "desktop", "port")
	ln, err := bindDesktopPort(context.Background(), "127.0.0.1", chemin, true)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if got, want := lirePort(t, chemin), strconv.Itoa(listenerPort(ln)); got != want {
		t.Errorf("fichier de port = %q, attendu le port lié %s", got, want)
	}
}

func TestBindDesktopPort_ReprendLePortGardeQuandIlEstLibre(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "desktop", "port")
	garde := portLibre(t)
	if err := writeStablePort(chemin, garde); err != nil {
		t.Fatal(err)
	}
	ln, err := bindDesktopPort(context.Background(), "127.0.0.1", chemin, true)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if got := listenerPort(ln); got != garde {
		t.Errorf("port lié = %d, attendu le port gardé %d", got, garde)
	}
}

func TestBindDesktopPort_PortOccupeRepliEtReecrit(t *testing.T) {
	logs := captureSlog(t)
	chemin := filepath.Join(t.TempDir(), "desktop", "port")
	occupant, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupant.Close()
	pris := listenerPort(occupant)
	if err := writeStablePort(chemin, pris); err != nil {
		t.Fatal(err)
	}

	ln, err := bindDesktopPort(context.Background(), "127.0.0.1", chemin, true)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if got := listenerPort(ln); got == pris || got == 0 {
		t.Fatalf("port lié = %d : le repli doit choisir un autre port libre que %d", got, pris)
	}
	if got, want := lirePort(t, chemin), strconv.Itoa(listenerPort(ln)); got != want {
		t.Errorf("fichier de port = %q, attendu le port de repli %s", got, want)
	}
	if out := logs.String(); !strings.Contains(out, "level=WARN") || !strings.Contains(out, "port de repli") {
		t.Errorf("avertissement « port de repli » absent du journal :\n%s", out)
	}
}

func TestBindDesktopPort_ContenuInvalideRepliSansPanique(t *testing.T) {
	for _, contenu := range []string{"", "abc", "0", "70000", "-5", "80 80"} {
		chemin := filepath.Join(t.TempDir(), "port")
		if err := os.WriteFile(chemin, []byte(contenu), 0o644); err != nil {
			t.Fatal(err)
		}
		ln, err := bindDesktopPort(context.Background(), "127.0.0.1", chemin, true)
		if err != nil {
			t.Fatalf("contenu %q : %v", contenu, err)
		}
		if got, want := lirePort(t, chemin), strconv.Itoa(listenerPort(ln)); got != want {
			t.Errorf("contenu %q : fichier = %q, attendu le port lié %s", contenu, got, want)
		}
		ln.Close()
	}
}

// Une instance sans le verrou d'instance ne réécrit pas le port de celle qui tourne.
func TestBindDesktopPort_SansPersistanceLeFichierResteIntact(t *testing.T) {
	chemin := filepath.Join(t.TempDir(), "desktop", "port")
	occupant, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupant.Close()
	pris := listenerPort(occupant)
	if err := writeStablePort(chemin, pris); err != nil {
		t.Fatal(err)
	}
	ln, err := bindDesktopPort(context.Background(), "127.0.0.1", chemin, false)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if got := lirePort(t, chemin); got != strconv.Itoa(pris) {
		t.Errorf("fichier de port = %q, attendu %d intact", got, pris)
	}
}

// Hors -mode desktop : le port configuré tel quel, sans fichier de port ; occupé, c'est une erreur.
func TestBindConfiguredPort_OccupeEstUneErreur(t *testing.T) {
	occupant, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupant.Close()
	if ln, err := bindConfiguredPort(context.Background(), occupant.Addr().String()); err == nil {
		ln.Close()
		t.Fatal("port occupé lié sans erreur : le serveur hébergé n'a pas de repli")
	}
}
