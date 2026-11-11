package replayartifact

// write_exclusive_test.go — la pose exclusive d'un artefact sur un dossier dont le système de fichiers refuse le
// lien dur (exFAT, FAT32) : repli par création exclusive, jamais de fichier partiel pris pour valide.

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"levelup/go-api/internal/observability"
)

// erreurSansLien : ce que rend un système de fichiers sans lien dur (le texte importe peu : toute erreur autre que
// « existe déjà » est lue comme un refus).
var erreurSansLien = errors.New("link: la fonction n'est pas prise en charge par ce système de fichiers")

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
