package replayartifact

// gate_test.go — LA PORTE UNIQUE DES OCTETS D'UN ARTEFACT : un seul format (`.json.gz`), un `.json` brut vaut
// « absent », listage par un seul prédicat de nom, copie, suppression.

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
)

const courtTest = "000d5950"

var docTest = []byte(`{"schemaVersion":108,"matchId":"000d5950-1234-4abc-9def-0123456789ab","tracks":[{"xuid":"1"}]}`)

func cheminCanonique(dir string) string { return filepath.Join(dir, courtTest+title.ReplayArtifactExt) }

// cheminBrut : le `.json` qu'une version ancienne rangeait à la place de l'artefact.
func cheminBrut(dir string) string { return filepath.Join(dir, courtTest+".json") }

func gzipTest(t *testing.T, raw []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func ecrireTest(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestEcritureCompresseRelitIdentique : l'écriture range un fichier gzip sous le nom canonique, et la lecture
// rend, octet pour octet, le JSON écrit.
func TestEcritureCompresseRelitIdentique(t *testing.T) {
	dir := t.TempDir()
	written, stored, err := Write(context.Background(), cheminCanonique(dir), docTest)
	if err != nil {
		t.Fatal(err)
	}
	if written != cheminCanonique(dir) {
		t.Errorf("chemin écrit = %s", written)
	}
	onDisk, err := os.ReadFile(written)
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk) < 2 || onDisk[0] != 0x1f || onDisk[1] != 0x8b || bytes.Equal(onDisk, docTest) {
		t.Fatalf("le fichier rangé n'est pas du gzip")
	}
	if !bytes.Equal(onDisk, stored) {
		t.Error("les octets rendus ne sont pas ceux du fichier")
	}
	c, err := Read(written)
	if err != nil || !bytes.Equal(c.JSON, docTest) || c.StoredBytes != len(onDisk) {
		t.Fatalf("relecture = %d octets, taille stockée %d, %v", len(c.JSON), c.StoredBytes, err)
	}
}

// TestEcritureEstDeterministe : deux écritures du même JSON rendent les mêmes octets (la somme d'un artefact
// ne dépend pas de l'heure d'écriture).
func TestEcritureEstDeterministe(t *testing.T) {
	a, err := encode(docTest)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := encode(docTest)
	if !bytes.Equal(a, b) {
		t.Fatal("l'encodage n'est pas déterministe")
	}
}

// TestUnJsonBrutVautAbsent : un `.json` laissé par une version ancienne n'est pas un artefact, même sans
// `.json.gz` à côté : ni lu, ni vu par la présence, ni listé.
func TestUnJsonBrutVautAbsent(t *testing.T) {
	dir := t.TempDir()
	ecrireTest(t, cheminBrut(dir), docTest)
	avant := observability.LoadCounter(CounterUnreadable)
	if _, err := ReadBytes(cheminCanonique(dir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("lecture par le chemin canonique : %v, attendu l'absence", err)
	}
	if Exists(cheminCanonique(dir)) {
		t.Error("Exists voit un artefact là où il n'y a qu'un .json brut")
	}
	if _, err := Locate(cheminCanonique(dir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Locate : %v, attendu l'absence", err)
	}
	if got, err := List(dir); err != nil || len(got) != 0 {
		t.Errorf("List = %+v, %v : un .json brut n'est pas listé", got, err)
	}
	if observability.LoadCounter(CounterUnreadable) != avant {
		t.Error("une absence n'est pas un fichier illisible")
	}
}

// TestAbsentEtIllisible : l'absence est nominale (pas de compteur), un fichier corrompu est compté.
func TestAbsentEtIllisible(t *testing.T) {
	dir := t.TempDir()
	avant := observability.LoadCounter(CounterUnreadable)
	if _, err := ReadBytes(cheminCanonique(dir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("artefact absent : %v", err)
	}
	if Exists(cheminCanonique(dir)) {
		t.Error("Exists sur un artefact absent")
	}
	if observability.LoadCounter(CounterUnreadable) != avant {
		t.Error("une absence n'est pas un fichier illisible")
	}
	tronque := gzipTest(t, docTest)
	ecrireTest(t, cheminCanonique(dir), tronque[:len(tronque)/2])
	if _, err := ReadBytes(cheminCanonique(dir)); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("un gzip tronqué doit donner une erreur, pas l'absence : %v", err)
	}
	if observability.LoadCounter(CounterUnreadable) <= avant {
		t.Error("le fichier illisible n'est pas compté")
	}
}

// TestListeUnSeulPredicat : un seul passage, un seul prédicat de nom ; ni marque, ni en-tête, ni sauvegarde, ni
// `.json` brut, ni dossier ; trié par forme courte.
func TestListeUnSeulPredicat(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"aaaa0001.json.gz", "bbbb0002.json", "cccc0003.json", "cccc0003.json.gz",
		"aaaa0001.derived.json", "aaaa0001.meta", "dddd0004.json.ancien-11juillet", "dddd0004.json.tmp"} {
		ecrireTest(t, filepath.Join(dir, n), []byte("x"))
	}
	if err := os.Mkdir(filepath.Join(dir, "rasters"), 0o755); err != nil {
		t.Fatal(err)
	}
	ecrireTest(t, filepath.Join(dir, "rasters", "aaaa0001.json"), []byte("x"))
	got, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"aaaa0001": "aaaa0001.json.gz", "cccc0003": "cccc0003.json.gz"}
	if len(got) != len(want) {
		t.Fatalf("%d artefacts listés, attendu %d : %+v", len(got), len(want), got)
	}
	for i, e := range got {
		if want[e.Short] != e.Name {
			t.Errorf("%s : fichier %s, attendu %s", e.Short, e.Name, want[e.Short])
		}
		if i > 0 && got[i-1].Short >= e.Short {
			t.Errorf("liste non triée : %s avant %s", got[i-1].Short, e.Short)
		}
	}
	if _, err := List(filepath.Join(dir, "absent")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("dossier absent : %v", err)
	}
}

// TestCopieRecompresse : la copie lit par la porte et réécrit sous le nom canonique.
func TestCopieRecompresse(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	ecrireTest(t, cheminCanonique(src), gzipTest(t, docTest))
	if err := Copy(context.Background(), cheminCanonique(src), cheminCanonique(dst)); err != nil {
		t.Fatal(err)
	}
	if lu, err := ReadBytes(cheminCanonique(dst)); err != nil || !bytes.Equal(lu, docTest) {
		t.Fatalf("copie : %d octets, %v", len(lu), err)
	}
	if err := Copy(context.Background(), cheminCanonique(t.TempDir()), cheminCanonique(dst)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("copie d'un artefact absent : %v", err)
	}
}

// TestRemoveRetireLArtefact : la suppression retire l'artefact et rend l'absence nominale.
func TestRemoveRetireLArtefact(t *testing.T) {
	dir := t.TempDir()
	ecrireTest(t, cheminCanonique(dir), gzipTest(t, docTest))
	if err := Remove(cheminCanonique(dir)); err != nil {
		t.Fatal(err)
	}
	if Exists(cheminCanonique(dir)) {
		t.Error("l'artefact doit avoir disparu")
	}
	if err := Remove(cheminCanonique(dir)); err != nil {
		t.Errorf("l'absence n'est pas une erreur : %v", err)
	}
}

// TestLectureRefuseUnFichierVideOuSansEnteteGzip : un fichier vide, ou qui n'est pas du gzip, est une erreur
// journalisée et comptée.
func TestLectureRefuseUnFichierVideOuSansEnteteGzip(t *testing.T) {
	cas := []struct {
		nom    string
		octets []byte
		ok     bool
	}{
		{"json sans entete gzip", docTest, false},
		{"vide", nil, false},
		{"gz valide", nil, true}, // octets posés ci-dessous
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			dir := t.TempDir()
			octets := c.octets
			if c.ok {
				octets = gzipTest(t, docTest)
			}
			ecrireTest(t, cheminCanonique(dir), octets)
			avant := observability.LoadCounter(CounterUnreadable)
			_, err := ReadBytes(cheminCanonique(dir))
			if c.ok {
				if err != nil {
					t.Fatalf("lecture refusée : %v", err)
				}
				return
			}
			if err == nil || errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("une erreur de lecture est attendue : %v", err)
			}
			if observability.LoadCounter(CounterUnreadable) <= avant {
				t.Error("le fichier refusé n'est pas compté")
			}
		})
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
