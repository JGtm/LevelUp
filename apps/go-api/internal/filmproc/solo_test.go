package filmproc

// solo_test.go — LE VERROU DOIT TENIR, ET NE JAMAIS COINCER.
//
// Deux exigences opposees : un second decodage doit etre REFUSE tant que le premier travaille
// (c'est la protection), et un verrou laisse par un processus TUE doit se reprendre TOUT SEUL
// (sinon la protection devient une panne — et le cas nominal ici est justement la mort violente :
// la sentinelle memoire tue, l'operateur aussi). Depuis le verrou OS (J2.6, 2026-09-26), la
// seconde est une propriete du NOYAU, prouvee sur un vrai processus tue par
// `platform/filelock.TestTryLock_LibereALaMortDuProcessus` ; ce fichier porte la premiere, et le
// fait qu'un detenteur ne rende jamais que SON verrou.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSoloRefuseUnSecondDecodage(t *testing.T) {
	dir := t.TempDir()
	premier, err := AcquireSolo(dir, "test-un", "match-a")
	if err != nil {
		t.Fatalf("premier verrou refuse : %v", err)
	}
	defer premier.Release()

	second, err := AcquireSolo(dir, "test-deux", "match-b")
	if err == nil {
		second.Release()
		t.Fatal("un SECOND decodage a ete autorise pendant que le premier tient le verrou — " +
			"c'est exactement le sinistre du 2026-08-31 (deux boucles de replay-build en parallele)")
	}
	if !errors.Is(err, ErrDecodeBusy) {
		t.Errorf("erreur inattendue : %v (attendu ErrDecodeBusy)", err)
	}
	// LE MESSAGE DOIT NOMMER LE DETENTEUR : sans lui, l'operateur ne sait pas quoi attendre ni
	// quoi arreter, et il ira tuer le mauvais processus.
	for _, attendu := range []string{"test-un", "match-a"} {
		if got := err.Error(); !contains(got, attendu) {
			t.Errorf("le refus ne nomme pas %q : %s", attendu, got)
		}
	}
}

func TestSoloRendLeVerrouALaLiberation(t *testing.T) {
	dir := t.TempDir()
	premier, err := AcquireSolo(dir, "test-un", "match-a")
	if err != nil {
		t.Fatalf("premier verrou refuse : %v", err)
	}
	premier.Release()
	premier.Release() // deux fois : le chemin defer + le chemin d'erreur peuvent tous deux passer

	second, err := AcquireSolo(dir, "test-deux", "match-b")
	if err != nil {
		t.Fatalf("le verrou n'a pas ete rendu : %v", err)
	}
	second.Release()
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// vieillirLeVerrou date d'une heure chaque fichier du dossier du verrou : l'etat d'un detenteur
// VIVANT mais gele (pause, veille), dont aucun battement n'a ete ecrit.
func vieillirLeVerrou(t *testing.T, dir string) {
	t.Helper()
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	vieux := time.Now().Add(-time.Hour)
	for _, e := range entrees {
		if err := os.Chtimes(filepath.Join(dir, e.Name()), vieux, vieux); err != nil {
			t.Fatal(err)
		}
	}
}

// TestAcquireSolo_ReleaseNeLibereQueSonVerrou — un detenteur ne rend que SON verrou. Avec le
// battement de coeur, un detenteur gele se faisait deposseder (verrou « perime » repris par B),
// puis son Release effacait le fichier de B : C decodait pendant que B travaillait. Quel que soit
// le chemin qui mene B au verrou, apres le Release de A, B doit toujours exclure C.
func TestAcquireSolo_ReleaseNeLibereQueSonVerrou(t *testing.T) {
	dir := t.TempDir()
	a, err := AcquireSolo(dir, "outil-a", "match-a")
	if err != nil {
		t.Fatalf("verrou de A : %v", err)
	}
	vieillirLeVerrou(t, dir)
	b, err := AcquireSolo(dir, "outil-b", "match-b")
	if err != nil {
		if !errors.Is(err, ErrDecodeBusy) {
			t.Fatalf("refus de B : %v, attendu ErrDecodeBusy", err)
		}
		a.Release()
		if b, err = AcquireSolo(dir, "outil-b", "match-b"); err != nil {
			t.Fatalf("verrou de B apres le Release de A : %v", err)
		}
	} else {
		a.Release()
	}
	defer b.Release()
	a.Release()

	if c, err := AcquireSolo(dir, "outil-c", "match-c"); err == nil {
		c.Release()
		t.Fatal("le Release de A a libere le verrou de B : C decode pendant que B travaille")
	}
}

// TestAcquireSolo_RefusNommeLeDernierDetenteur — le refus nomme le detenteur ACTUEL, jamais un
// precedent : l'operateur doit savoir quel processus attendre ou arreter.
func TestAcquireSolo_RefusNommeLeDernierDetenteur(t *testing.T) {
	dir := t.TempDir()
	a, err := AcquireSolo(dir, "outil-a", "match-a")
	if err != nil {
		t.Fatalf("verrou de A : %v", err)
	}
	a.Release()
	b, err := AcquireSolo(dir, "outil-b", "match-b")
	if err != nil {
		t.Fatalf("verrou de B : %v", err)
	}
	defer b.Release()

	_, err = AcquireSolo(dir, "outil-c", "match-c")
	if !errors.Is(err, ErrDecodeBusy) {
		t.Fatalf("err = %v, attendu ErrDecodeBusy", err)
	}
	for _, attendu := range []string{"outil-b", "match-b"} {
		if !strings.Contains(err.Error(), attendu) {
			t.Errorf("le refus ne nomme pas %q : %s", attendu, err)
		}
	}
	for _, ancien := range []string{"outil-a", "match-a"} {
		if strings.Contains(err.Error(), ancien) {
			t.Errorf("le refus nomme l'ancien detenteur %q : %s", ancien, err)
		}
	}
}

// TestAcquireSoloWait_AttendPuisPrend — le regime d'attente prend le verrou des qu'il est rendu.
func TestAcquireSoloWait_AttendPuisPrend(t *testing.T) {
	dir := t.TempDir()
	tenu, err := AcquireSolo(dir, "outil-a", "match-a")
	if err != nil {
		t.Fatalf("verrou de A : %v", err)
	}
	go func() {
		time.Sleep(300 * time.Millisecond)
		tenu.Release()
	}()
	l, err := AcquireSoloWait(context.Background(), dir, "outil-b", "match-b", 5*time.Second)
	if err != nil {
		t.Fatalf("attendu le verrou une fois rendu, obtenu %v", err)
	}
	l.Release()
}
