package replayartifact

// gate_test.go — LA PORTE UNIQUE DES OCTETS D'UN ARTEFACT : lecture des deux formes (détectées par les octets
// magiques), parc mixte, listage par un seul prédicat de nom, copie, suppression.

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
func cheminBrut(dir string) string {
	return filepath.Join(dir, courtTest+title.ReplayArtifactLegacyExt)
}

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
	a, err := Encode(docTest)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Encode(docTest)
	if !bytes.Equal(a, b) {
		t.Fatal("l'encodage n'est pas déterministe")
	}
}

// TestLitUnJsonAncien : un artefact brut d'un parc ancien se lit par son chemin canonique comme par le sien.
func TestLitUnJsonAncien(t *testing.T) {
	dir := t.TempDir()
	ecrireTest(t, cheminBrut(dir), docTest)
	for _, p := range []string{cheminCanonique(dir), cheminBrut(dir)} {
		lu, err := ReadBytes(p)
		if err != nil || !bytes.Equal(lu, docTest) {
			t.Errorf("ReadBytes(%s) = %d octets, %v", filepath.Base(p), len(lu), err)
		}
	}
	if !Exists(cheminCanonique(dir)) {
		t.Error("Exists ne voit pas l'artefact brut")
	}
}

// TestDetecteParLesOctetsMagiques : le nom ne décide rien.
func TestDetecteParLesOctetsMagiques(t *testing.T) {
	dir := t.TempDir()
	ecrireTest(t, cheminBrut(dir), gzipTest(t, docTest)) // du gzip sous le nom brut
	if lu, err := ReadBytes(cheminBrut(dir)); err != nil || !bytes.Equal(lu, docTest) {
		t.Errorf("gzip sous un nom .json : %d octets, %v", len(lu), err)
	}
	// Du JSON brut sous le nom compressé n'est pas un artefact : refusé (cf. TestLectureRefuseUnFichierVideOuSansEnteteGzip).
}

// TestParcMixte : le compressé gagne, et le doublon est compté.
func TestParcMixte(t *testing.T) {
	dir := t.TempDir()
	ancien, recent := []byte(`{"schemaVersion":1}`), []byte(`{"schemaVersion":2}`)
	ecrireTest(t, cheminBrut(dir), ancien)
	ecrireTest(t, cheminCanonique(dir), gzipTest(t, recent))
	avant := observability.LoadCounter(CounterDuplicate)
	lu, err := ReadBytes(cheminCanonique(dir))
	if err != nil || !bytes.Equal(lu, recent) {
		t.Fatalf("le .json.gz doit gagner, lu : %s (%v)", lu, err)
	}
	if observability.LoadCounter(CounterDuplicate) <= avant {
		t.Error("le doublon n'est pas compté")
	}
	if f, err := Locate(cheminBrut(dir)); err != nil || f.Path != cheminCanonique(dir) {
		t.Errorf("Locate depuis le chemin brut = %+v, %v", f, err)
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

// TestEcritureRetireLeBrutAncien : réécrire un artefact qui existait en brut ne laisse pas deux fichiers.
func TestEcritureRetireLeBrutAncien(t *testing.T) {
	dir := t.TempDir()
	ecrireTest(t, cheminBrut(dir), []byte(`{"schemaVersion":1}`))
	written, _, err := Write(context.Background(), cheminCanonique(dir), docTest)
	if err != nil {
		t.Fatal(err)
	}
	if !HasLegacySibling(written) {
		t.Fatal("Write ne touche pas au .json ancien : c'est RemoveLegacy qui le retire")
	}
	RemoveLegacy(context.Background(), written)
	if _, err := os.Stat(cheminBrut(dir)); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("le .json ancien doit disparaître : %v", err)
	}
}

// TestListeUnSeulPredicat : un seul passage, un seul prédicat de nom ; ni marque, ni en-tête, ni sauvegarde, ni
// dossier ; en parc mixte le compressé gagne ; trié par forme courte.
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
	want := map[string]string{"aaaa0001": "aaaa0001.json.gz", "bbbb0002": "bbbb0002.json", "cccc0003": "cccc0003.json.gz"}
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
	ecrireTest(t, cheminBrut(src), docTest)
	if err := Copy(context.Background(), cheminCanonique(src), cheminCanonique(dst)); err != nil {
		t.Fatal(err)
	}
	if lu, err := ReadBytes(cheminCanonique(dst)); err != nil || !bytes.Equal(lu, docTest) {
		t.Fatalf("copie : %d octets, %v", len(lu), err)
	}
	if _, err := os.Stat(cheminCanonique(dst)); err != nil {
		t.Errorf("la copie doit être rangée sous le nom compressé : %v", err)
	}
}

// TestRemoveRetireLesDeuxFormes.
func TestRemoveRetireLesDeuxFormes(t *testing.T) {
	dir := t.TempDir()
	ecrireTest(t, cheminBrut(dir), docTest)
	ecrireTest(t, cheminCanonique(dir), gzipTest(t, docTest))
	if err := Remove(cheminCanonique(dir)); err != nil {
		t.Fatal(err)
	}
	if Exists(cheminCanonique(dir)) {
		t.Error("l'artefact doit avoir disparu sous ses deux formes")
	}
	if err := Remove(cheminCanonique(dir)); err != nil {
		t.Errorf("l'absence n'est pas une erreur : %v", err)
	}
}

// TestEcritureSansEcraserNePoseJamaisParDessusUnCompresse : la pose exclusive ne remplace pas un `.json.gz`
// existant (un serveur actif a pu en poser un plus récent), et ne laisse aucun temporaire.
func TestEcritureSansEcraserNePoseJamaisParDessusUnCompresse(t *testing.T) {
	dir := t.TempDir()
	recent := []byte(`{"schemaVersion":2}`)
	ecrireTest(t, cheminCanonique(dir), gzipTest(t, recent))
	avant, _ := os.ReadFile(cheminCanonique(dir))
	_, _, created, err := WriteIfAbsent(context.Background(), cheminCanonique(dir), docTest)
	if err != nil || created {
		t.Fatalf("created = %v, err = %v : l'existant doit être respecté sans erreur", created, err)
	}
	if apres, _ := os.ReadFile(cheminCanonique(dir)); !bytes.Equal(apres, avant) {
		t.Fatal("le .json.gz existant a été écrasé")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("un temporaire a été laissé : %d entrées", len(entries))
	}
	other := t.TempDir()
	w, stored, created, err := WriteIfAbsent(context.Background(), cheminCanonique(other), docTest)
	if err != nil || !created || w != cheminCanonique(other) {
		t.Fatalf("première pose : %s, %v, %v", w, created, err)
	}
	if c, err := Read(w); err != nil || !bytes.Equal(c.JSON, docTest) || c.StoredBytes != len(stored) {
		t.Fatalf("relecture de la pose : %v", err)
	}
}

// TestLectureRefuseUnFichierVideOuSansEnteteGzip : un `.json.gz` qui n'est pas du gzip, et tout fichier vide, sont des
// erreurs journalisées et comptées ; un `.json` brut reste accepté.
func TestLectureRefuseUnFichierVideOuSansEnteteGzip(t *testing.T) {
	cas := []struct {
		nom, fichier string
		octets       []byte
		ok           bool
	}{
		{"gz sans entete", courtTest + ".json.gz", docTest, false},
		{"gz vide", courtTest + ".json.gz", nil, false},
		{"brut vide", courtTest + ".json", nil, false},
		{"brut valide", courtTest + ".json", docTest, true},
		{"gz valide", courtTest + ".json.gz", nil, true}, // octets posés ci-dessous
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			dir := t.TempDir()
			octets := c.octets
			if c.nom == "gz valide" {
				octets = gzipTest(t, docTest)
			}
			ecrireTest(t, filepath.Join(dir, c.fichier), octets)
			avant := observability.LoadCounter(CounterUnreadable)
			_, err := ReadBytes(filepath.Join(dir, c.fichier))
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
