package replayartifact

// write_exclusive_test.go — la pose exclusive d'un artefact sur un dossier dont le système de fichiers refuse le
// lien dur (exFAT, FAT32) : repli par création exclusive, jamais de fichier partiel pris pour valide.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"

	"levelup/go-api/internal/observability"
)

// erreurSansLien : ce que rend un système de fichiers sans lien dur (le texte importe peu : toute erreur autre que
// « existe déjà » est lue comme un refus).
var erreurSansLien = fmt.Errorf("link: %w", errors.ErrUnsupported)

// sansLienDur remplace linkFile par un refus et rend le nombre d'appels.
func sansLienDur(t *testing.T) *atomic.Int32 {
	t.Helper()
	var appels atomic.Int32
	avant := linkFile
	linkFile = func(string, string) error { appels.Add(1); return erreurSansLien }
	t.Cleanup(func() { linkFile = avant })
	return &appels
}

func entreesDuDossier(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// TestPoseExclusive_LienRefuse_RepliSurLaCreationExclusive : sans lien dur, le fichier final est créé
// exclusivement, écrit et relu identique ; aucun temporaire ne reste ; le dossier est mémorisé (un seul essai de
// lien) et le repli est compté.
func TestPoseExclusive_LienRefuse_RepliSurLaCreationExclusive(t *testing.T) {
	appels := sansLienDur(t)
	dir := t.TempDir()
	avant := observability.LoadCounter(CounterLinkFallback)
	w, stored, created, err := WriteIfAbsent(context.Background(), cheminCanonique(dir), docTest)
	if err != nil || !created || w != cheminCanonique(dir) {
		t.Fatalf("première pose : %s, créé=%v, err=%v", w, created, err)
	}
	if c, err := Read(w); err != nil || !bytes.Equal(c.JSON, docTest) || c.StoredBytes != len(stored) {
		t.Fatalf("relecture : %v", err)
	}
	if got := entreesDuDossier(t, dir); len(got) != 1 {
		t.Fatalf("le dossier doit ne porter que l'artefact : %v", got)
	}
	// Une seconde pose dans le même dossier ne retente pas le lien dur.
	autre := filepath.Join(dir, "00000002.json.gz")
	if _, _, created, err = WriteIfAbsent(context.Background(), autre, docTest); err != nil || !created {
		t.Fatalf("seconde pose : créé=%v, err=%v", created, err)
	}
	if appels.Load() != 1 {
		t.Errorf("le lien dur a été tenté %d fois pour un dossier qui le refuse", appels.Load())
	}
	if got := observability.LoadCounter(CounterLinkFallback); got != avant+2 {
		t.Errorf("compteur de repli = %d, attendu %d", got, avant+2)
	}
}

// TestPoseExclusive_LienRefuse_NeRemplaceJamaisUnExistant : le repli garde la règle de la pose exclusive.
func TestPoseExclusive_LienRefuse_NeRemplaceJamaisUnExistant(t *testing.T) {
	sansLienDur(t)
	dir := t.TempDir()
	recent := gzipTest(t, []byte(`{"schemaVersion":2}`))
	ecrireTest(t, cheminCanonique(dir), recent)
	_, _, created, err := WriteIfAbsent(context.Background(), cheminCanonique(dir), docTest)
	if err != nil || created {
		t.Fatalf("créé=%v, err=%v : l'existant doit être respecté sans erreur", created, err)
	}
	if apres, _ := os.ReadFile(cheminCanonique(dir)); !bytes.Equal(apres, recent) {
		t.Fatal("le .json.gz existant a été écrasé")
	}
	if got := entreesDuDossier(t, dir); len(got) != 1 {
		t.Errorf("fichier résiduel : %v", got)
	}
}

// TestPoseExclusive_LienPrisEnCharge_NeRepliePas : un lien dur qui marche n'ouvre aucun repli.
func TestPoseExclusive_LienPrisEnCharge_NeRepliePas(t *testing.T) {
	dir := t.TempDir()
	avant := observability.LoadCounter(CounterLinkFallback)
	if _, _, created, err := WriteIfAbsent(context.Background(), cheminCanonique(dir), docTest); err != nil || !created {
		t.Fatalf("créé=%v, err=%v", created, err)
	}
	if got := observability.LoadCounter(CounterLinkFallback); got != avant {
		t.Errorf("repli compté sans refus du lien dur : %d -> %d", avant, got)
	}
}

// TestPoseExclusive_EchecDEcriture_RetireLePartiel : une écriture qui échoue en cours retire le fichier final
// partiel, rend l'erreur, et ne laisse rien.
func TestPoseExclusive_EchecDEcriture_RetireLePartiel(t *testing.T) {
	sansLienDur(t)
	avant := ecrireEtSynchroniser
	ecrireEtSynchroniser = func(f *os.File, data []byte) error {
		if _, err := f.Write(data[:len(data)/2]); err != nil {
			return err
		}
		return errors.New("disque plein")
	}
	t.Cleanup(func() { ecrireEtSynchroniser = avant })
	dir := t.TempDir()
	_, _, created, err := WriteIfAbsent(context.Background(), cheminCanonique(dir), docTest)
	if err == nil || created {
		t.Fatalf("créé=%v, err=%v : l'échec doit remonter", created, err)
	}
	if got := entreesDuDossier(t, dir); len(got) != 0 {
		t.Errorf("un fichier partiel est resté : %v", got)
	}
}

// TestPoseExclusive_LecteurConcurrent_JamaisUnPartielPourValide : pendant que le repli écrit (moitié des octets
// posés), un lecteur reçoit une erreur, jamais un document ; une fois l'écriture finie, le document entier.
func TestPoseExclusive_LecteurConcurrent_JamaisUnPartielPourValide(t *testing.T) {
	sansLienDur(t)
	moitie, reprise := make(chan struct{}), make(chan struct{})
	avant := ecrireEtSynchroniser
	ecrireEtSynchroniser = func(f *os.File, data []byte) error {
		if _, err := f.Write(data[:len(data)/2]); err != nil {
			return err
		}
		close(moitie)
		<-reprise
		_, err := f.Write(data[len(data)/2:])
		return err
	}
	t.Cleanup(func() { ecrireEtSynchroniser = avant })
	dir := t.TempDir()
	fini := make(chan error, 1)
	go func() {
		_, _, _, err := WriteIfAbsent(context.Background(), cheminCanonique(dir), docTest)
		fini <- err
	}()
	<-moitie
	if doc, err := ReadBytes(cheminCanonique(dir)); err == nil {
		t.Fatalf("un fichier partiel a été lu comme valide : %q", doc)
	}
	close(reprise)
	if err := <-fini; err != nil {
		t.Fatal(err)
	}
	if doc, err := ReadBytes(cheminCanonique(dir)); err != nil || !bytes.Equal(doc, docTest) {
		t.Fatalf("relecture finale : %v", err)
	}
}

// TestLecture_RefuseUnGzipTronque : un `.json.gz` coupé à n'importe quel octet est une erreur journalisée et
// comptée, jamais un document.
func TestLecture_RefuseUnGzipTronque(t *testing.T) {
	complet := gzipTest(t, docTest)
	for _, n := range []int{2, 10, len(complet) / 2, len(complet) - 8, len(complet) - 1} {
		dir := t.TempDir()
		ecrireTest(t, cheminCanonique(dir), complet[:n])
		avant := observability.LoadCounter(CounterUnreadable)
		doc, err := ReadBytes(cheminCanonique(dir))
		if err == nil || errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("coupé à %d octets sur %d : lu comme valide (%q) ou absent (%v)", n, len(complet), doc, err)
		}
		if observability.LoadCounter(CounterUnreadable) <= avant {
			t.Errorf("coupé à %d octets : non compté", n)
		}
	}
}

// TestLecture_CompresseIllisible_LitLeBrutVoisin : un `.json.gz` illisible (reste d'une écriture interrompue) ne
// masque pas un `.json` valide du même match ; sans brut, l'erreur remonte.
func TestLecture_CompresseIllisible_LitLeBrutVoisin(t *testing.T) {
	dir := t.TempDir()
	complet := gzipTest(t, docTest)
	ecrireTest(t, cheminCanonique(dir), complet[:len(complet)/2])
	if _, err := ReadBytes(cheminCanonique(dir)); err == nil {
		t.Fatal("sans brut voisin, un compressé tronqué doit être une erreur")
	}
	ecrireTest(t, cheminBrut(dir), docTest)
	doc, err := ReadBytes(cheminCanonique(dir))
	if err != nil || !bytes.Equal(doc, docTest) {
		t.Fatalf("le brut valide doit être lu : %v", err)
	}
}

// TestPoseExclusive_ErreurTransitoire_RepliePourCeFichierSeulement : une erreur du lien dur qui ne dit pas « non pris en
// charge » (antivirus, violation de partage) replie pour CE fichier, journalise, compte, et ne mémorise pas le dossier.
func TestPoseExclusive_ErreurTransitoire_RepliePourCeFichierSeulement(t *testing.T) {
	var appels atomic.Int32
	avant := linkFile
	linkFile = func(string, string) error {
		appels.Add(1)
		return errors.New("le processus ne peut pas accéder au fichier")
	}
	t.Cleanup(func() { linkFile = avant })
	dir := t.TempDir()
	repli := observability.LoadCounter(CounterLinkFallback)
	erreurs := observability.LoadCounter(CounterLinkError)
	for _, nom := range []string{courtTest, "00000002"} {
		_, _, created, err := WriteIfAbsent(context.Background(), filepath.Join(dir, nom+".json.gz"), docTest)
		if err != nil || !created {
			t.Fatalf("%s : créé=%v, err=%v", nom, created, err)
		}
	}
	if appels.Load() != 2 {
		t.Errorf("le lien dur a été tenté %d fois, attendu 2 : le dossier n'est pas mémorisé sur une erreur transitoire", appels.Load())
	}
	if got := observability.LoadCounter(CounterLinkFallback); got != repli+2 {
		t.Errorf("repli compté %d, attendu %d", got, repli+2)
	}
	if got := observability.LoadCounter(CounterLinkError); got != erreurs+2 {
		t.Errorf("erreur de lien compté %d, attendu %d", got, erreurs+2)
	}
}

// TestLienNonPrisEnCharge : seules les erreurs qui signifient « ce système de fichiers ne sait pas » mémorisent le
// dossier ; les codes sont ceux de l'OS courant.
func TestLienNonPrisEnCharge(t *testing.T) {
	lien := func(e error) error { return &os.LinkError{Op: "link", Old: "a", New: "b", Err: e} }
	cas := []struct {
		nom string
		err error
		ok  bool
	}{
		{"non pris en charge", lien(errors.ErrUnsupported), true},
		{"erreur quelconque", lien(errors.New("boum")), false},
		{"nil", nil, false},
	}
	if runtime.GOOS == "windows" {
		cas = append(cas, []struct {
			nom string
			err error
			ok  bool
		}{
			{"ERROR_INVALID_FUNCTION (exFAT, FAT32)", lien(syscall.Errno(1)), true},
			{"ERROR_NOT_SUPPORTED", lien(syscall.Errno(50)), true},
			{"ERROR_SHARING_VIOLATION", lien(syscall.Errno(32)), false},
			{"ERROR_ACCESS_DENIED", lien(syscall.Errno(5)), false},
		}...)
	} else {
		cas = append(cas, []struct {
			nom string
			err error
			ok  bool
		}{
			{"EPERM (FAT sous Linux)", lien(syscall.EPERM), true},
			{"ENOTSUP", lien(syscall.ENOTSUP), true},
			{"EACCES", lien(syscall.EACCES), false},
		}...)
	}
	for _, c := range cas {
		if got := lienNonPrisEnCharge(c.err); got != c.ok {
			t.Errorf("%s : lienNonPrisEnCharge = %v, attendu %v", c.nom, got, c.ok)
		}
	}
}

// TestLecture_CompresseEnCoursDEcriture_UneRelectureAvantDeConclure : le compressé est partiel à la première lecture, le
// brut voisin disparaît, et le compressé est complet à la relecture : le document est rendu, pas l'erreur du partiel.
func TestLecture_CompresseEnCoursDEcriture_UneRelectureAvantDeConclure(t *testing.T) {
	dir := t.TempDir()
	complet := gzipTest(t, docTest)
	ecrireTest(t, cheminCanonique(dir), complet[:len(complet)/2])
	ecrireTest(t, cheminBrut(dir), docTest)
	avant := apresEchecDeLecture
	apresEchecDeLecture = func(string) {
		ecrireTest(t, cheminCanonique(dir), complet)
		_ = os.Remove(cheminBrut(dir))
	}
	t.Cleanup(func() { apresEchecDeLecture = avant })
	doc, err := ReadBytes(cheminCanonique(dir))
	if err != nil || !bytes.Equal(doc, docTest) {
		t.Fatalf("la relecture doit rendre le document complet : %v", err)
	}
}

// TestLecture_CompresseVraimentIllisible_UneSeuleRelecture : sans brut et sans changement, l'erreur est rendue après UNE
// relecture, et comptée une fois.
func TestLecture_CompresseVraimentIllisible_UneSeuleRelecture(t *testing.T) {
	dir := t.TempDir()
	complet := gzipTest(t, docTest)
	ecrireTest(t, cheminCanonique(dir), complet[:len(complet)/2])
	appels := 0
	avant := apresEchecDeLecture
	apresEchecDeLecture = func(string) { appels++ }
	t.Cleanup(func() { apresEchecDeLecture = avant })
	compte := observability.LoadCounter(CounterUnreadable)
	if _, err := ReadBytes(cheminCanonique(dir)); err == nil {
		t.Fatal("un compressé tronqué sans brut doit rester une erreur")
	}
	if appels != 1 {
		t.Errorf("relectures = %d, attendu 1", appels)
	}
	if got := observability.LoadCounter(CounterUnreadable); got != compte+1 {
		t.Errorf("illisible compté %d fois, attendu 1", got-compte)
	}
}
