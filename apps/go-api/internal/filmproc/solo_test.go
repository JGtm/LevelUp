package filmproc

// solo_test.go — LE VERROU DOIT TENIR, ET NE JAMAIS COINCER.
//
// Deux exigences opposees, et le test porte les deux : un second decodage doit etre REFUSE tant
// que le premier travaille (c'est la protection), et un verrou laisse par un processus TUE doit
// se reprendre TOUT SEUL (sinon la protection devient une panne — et le cas nominal ici est
// justement la mort violente : la sentinelle memoire tue, l'operateur aussi).

import (
	"errors"
	"os"
	"path/filepath"
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

func TestSoloRepreudUnVerrouPerime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, soloFileName)
	// Un verrou laisse par un processus TUE : le fichier existe, son battement est vieux.
	if err := os.WriteFile(path, []byte(`{"tool":"mort","pid":1}`), 0o644); err != nil {
		t.Fatalf("preparation : %v", err)
	}
	vieux := time.Now().Add(-2 * soloStale)
	if err := os.Chtimes(path, vieux, vieux); err != nil {
		t.Fatalf("preparation (dates) : %v", err)
	}

	l, err := AcquireSolo(dir, "test-repreneur", "match-a")
	if err != nil {
		t.Fatalf("un verrou PERIME n'a pas ete repris : %v — la protection deviendrait une panne "+
			"des la premiere coupure par la sentinelle memoire", err)
	}
	l.Release()
}

func TestSoloNeReprendPasUnVerrouVivant(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, soloFileName)
	if err := os.WriteFile(path, []byte(`{"tool":"vivant","pid":2}`), 0o644); err != nil {
		t.Fatalf("preparation : %v", err)
	}
	// Battement FRAIS (l'ecriture vient d'avoir lieu) : le detenteur travaille.
	if _, err := AcquireSolo(dir, "test-intrus", "match-b"); !errors.Is(err, ErrDecodeBusy) {
		t.Fatalf("un verrou FRAIS a ete repris : %v", err)
	}
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
