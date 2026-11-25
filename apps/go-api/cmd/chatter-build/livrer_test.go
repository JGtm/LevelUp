package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// voixDeTest pose un dossier d'`.ogg` encodés nommés par identifiant de média.
func voixDeTest(t *testing.T, medias map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for id, contenu := range medias {
		if err := os.WriteFile(filepath.Join(dir, id+".ogg"), []byte(contenu), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func choixDeTest() []Choix {
	return []Choix{
		{Son: "0000000a", Textes: texteFrEn("un", "one"), Voix: map[string]MediaVoix{"fr": {ID: "m1"}, "en": {ID: "m1en"}}},
		{Son: "0000000b", Textes: texteFrEn("deux", "two"), Voix: map[string]MediaVoix{"fr": {ID: "m2"}}},
	}
}

func TestLivrerCopieLesVoixEtNAnnonceQueCeQuiExiste(t *testing.T) {
	langues := languesDeTest(t)
	dossier := t.TempDir()
	// le français a les deux voix ; l'anglais a un dossier mais pas le fichier de 0a : voix annoncée seulement pour ce qui existe
	voix := sourcesVoix{"fr": voixDeTest(t, map[string]string{"m1": "ogg1", "m2": "ogg2"}), "en": voixDeTest(t, nil)}
	tab, err := livrer(choixDeTest(), voix, langues, dossier, journalDeTest(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := tab.Sons["0000000a"].Audio; !reflect.DeepEqual(got, []string{"fr"}) {
		t.Errorf("voix de 0a = %v, seul le français existe", got)
	}
	if b, _ := os.ReadFile(filepath.Join(dossier, "fr", "0000000b.ogg")); string(b) != "ogg2" {
		t.Errorf("voix copiée = %q : le fichier porte l'identifiant du son, pas celui du média", b)
	}
	if err := verifierVoix(tab, langues, dossier); err != nil {
		t.Error(err)
	}
	brut, err := os.ReadFile(filepath.Join(dossier, nomTable))
	if err != nil || !strings.Contains(string(brut), `"0000000a":{"texte":{"en":"one","fr":"un"},"audio":["fr"]}`) {
		t.Errorf("table écrite = %s, %v : une réplique par ligne", brut, err)
	}
	if d, err := clesEnDouble(brut); err != nil || len(d) != 0 {
		t.Errorf("clés en double = %v, %v", d, err)
	}
}

func TestLivrerRetireLesVoixQuePlusRienNAnnonce(t *testing.T) {
	langues := languesDeTest(t)
	dossier := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dossier, "fr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dossier, "fr", "0000ffff.ogg"), []byte("vieux"), 0o644); err != nil {
		t.Fatal(err)
	}
	voix := sourcesVoix{"fr": voixDeTest(t, map[string]string{"m1": "x", "m2": "y"})}
	if _, err := livrer(choixDeTest(), voix, langues, dossier, journalDeTest(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dossier, "fr", "0000ffff.ogg")); err == nil {
		t.Error("une voix qu'aucune entrée n'annonce doit disparaître du dossier versé")
	}
}

func TestValiderTableRefuse(t *testing.T) {
	langues := languesDeTest(t)
	ok := Table{Version: versionTable, Sons: map[string]EntreeTable{"0000000a": {Texte: texteFrEn("a", "b"), Audio: []string{"fr"}}}}
	if err := validerTable(ok, langues); err != nil {
		t.Fatalf("table valide refusée : %v", err)
	}
	cas := map[string]Table{
		"version":        {Version: 99, Sons: ok.Sons},
		"vide":           {Version: versionTable},
		"identifiant":    {Version: versionTable, Sons: map[string]EntreeTable{"xyz": ok.Sons["0000000a"]}},
		"texte manquant": {Version: versionTable, Sons: map[string]EntreeTable{"0000000a": {Texte: map[string]string{"fr": "a"}}}},
		"voix inconnue":  {Version: versionTable, Sons: map[string]EntreeTable{"0000000a": {Texte: texteFrEn("a", "b"), Audio: []string{"xx"}}}},
		"voix en double": {Version: versionTable, Sons: map[string]EntreeTable{"0000000a": {Texte: texteFrEn("a", "b"), Audio: []string{"fr", "fr"}}}},
	}
	for nom, tab := range cas {
		if err := validerTable(tab, langues); err == nil {
			t.Errorf("%s : table acceptée", nom)
		}
	}
}

func TestClesEnDoubleLesVoit(t *testing.T) {
	brut := []byte(`{"version":1,"sons":{"0000000a":{"texte":{}},"0000000b":{"texte":{}},"0000000a":{"texte":{}}}}`)
	d, err := clesEnDouble(brut)
	if err != nil || !reflect.DeepEqual(d, []string{"0000000a"}) {
		t.Errorf("doubles = %v, %v", d, err)
	}
}

func TestVerifierVoixVoitLesEcarts(t *testing.T) {
	langues := languesDeTest(t)
	dossier := t.TempDir()
	tab := Table{Version: versionTable, Sons: map[string]EntreeTable{"0000000a": {Texte: texteFrEn("a", "b"), Audio: []string{"fr"}}}}
	if err := verifierVoix(tab, langues, dossier); err == nil {
		t.Error("une voix annoncée sans fichier doit être refusée")
	}
	if err := os.MkdirAll(filepath.Join(dossier, "fr"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"0000000a.ogg", "0000000z.ogg"} {
		if err := os.WriteFile(filepath.Join(dossier, "fr", n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := verifierVoix(tab, langues, dossier); err == nil {
		t.Error("un fichier versé mais non annoncé doit être refusé")
	}
}

func TestLireSourcesVoix(t *testing.T) {
	got, err := lireSourcesVoix("fr=/a, en=/b")
	if err != nil || got["fr"] != "/a" || got["en"] != "/b" {
		t.Errorf("sources = %v, %v", got, err)
	}
	if _, err := lireSourcesVoix("fr"); err == nil {
		t.Error("une source sans dossier doit être refusée")
	}
}
