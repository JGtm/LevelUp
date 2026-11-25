package main

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func ecrireDoc(t *testing.T, chemin, contenu string, gz bool) {
	t.Helper()
	f, err := os.Create(chemin)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if gz {
		z := gzip.NewWriter(f)
		if _, err := z.Write([]byte(contenu)); err != nil {
			t.Fatal(err)
		}
		if err := z.Close(); err != nil {
			t.Fatal(err)
		}
		return
	}
	if _, err := f.WriteString(contenu); err != nil {
		t.Fatal(err)
	}
}

func TestLireSonsJouesCompteLesSonsDeTousLesDocuments(t *testing.T) {
	dir := t.TempDir()
	ecrireDoc(t, filepath.Join(dir, "a.json.gz"), `{"matchId":"x","chatter":[{"t":1,"sound":"0000000a"},{"t":2,"sound":"0000000a","name":"n"},{"t":3}]}`, true)
	ecrireDoc(t, filepath.Join(dir, "b.json"), `{"chatter":[{"t":1,"sound":"0000000b"}]}`, false)
	ecrireDoc(t, filepath.Join(dir, "c.json"), `{"tracks":[]}`, false) // aucun calque de répliques
	ecrireDoc(t, filepath.Join(dir, "casse.json"), `{"chatter":[`, false)
	ecrireDoc(t, filepath.Join(dir, "notes.txt"), `pas un document`, false)
	got, err := lireSonsJoues(dir, journalDeTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.Films != 3 || got.Sons["0000000a"] != 2 || got.Sons["0000000b"] != 1 || len(got.Sons) != 2 {
		t.Errorf("sons joués = %+v : trois documents lus, un illisible écarté, une réplique sans son ignorée", got)
	}
}

func TestLireSonsJouesRefuseUnDossierSansDocument(t *testing.T) {
	if _, err := lireSonsJoues(t.TempDir(), journalDeTest(t)); err == nil {
		t.Error("un dossier sans document lisible doit être refusé : une liste vide viderait la table")
	}
}
