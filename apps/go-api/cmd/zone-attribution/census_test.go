package main

// census_test.go — les deux pieces PURES du recensement.
//
// Ce que ces cas protegent : la lecture du schema d'un artefact (dont le cas « fichier absent »,
// qui est le plus frequent — la plupart des matchs n'ont pas d'artefact cuit) et le format des
// schemas, qui est la SEULE sortie du recensement qu'un lecteur puisse mal interpreter.
//
// Le reste de `runCensus` est de l'orchestration : base, disque, racine du depot. Il se juge en
// le lancant, pas en le simulant — un faux registre ne dirait rien de plus que ce que ces deux
// fonctions disent deja.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactSchema(t *testing.T) {
	dir := t.TempDir()
	cas := []struct {
		nom     string
		contenu string
		veut    int
		ok      bool
	}{
		{"artefact valide", `{"schemaVersion":20,"matchId":"abcd"}`, 20, true},
		// Un artefact SANS le champ n'est pas illisible : il rend zero, et le compte des
		// artefacts le retient quand meme — c'est un artefact, simplement d'avant le champ.
		{"sans schemaVersion", `{"matchId":"abcd"}`, 0, true},
		{"json casse", `{"schemaVersion":`, 0, false},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			p := filepath.Join(dir, c.nom+".json")
			if err := os.WriteFile(p, []byte(c.contenu), 0o644); err != nil {
				t.Fatalf("ecriture : %v", err)
			}
			got, ok := artifactSchema(p)
			if got != c.veut || ok != c.ok {
				t.Errorf("artifactSchema = (%d, %v), attendu (%d, %v)", got, ok, c.veut, c.ok)
			}
		})
	}
	// LE CAS LE PLUS FREQUENT : la plupart des matchs n'ont aucun artefact cuit. L'absence doit
	// etre un faux PROPRE, jamais une erreur qui arreterait le recensement.
	if got, ok := artifactSchema(filepath.Join(dir, "jamais-cuit.json")); ok || got != 0 {
		t.Errorf("artefact absent : (%d, %v), attendu (0, false)", got, ok)
	}
}

func TestFormatSchemas(t *testing.T) {
	cas := []struct {
		nom  string
		in   map[int]int
		veut string
	}{
		// Aucun artefact cuit : un tiret, jamais « 0 » — qui se lirait comme un schema 0.
		{"aucun", map[int]int{}, "-"},
		{"un seul", map[int]int{20: 12}, "20x12"},
		// TRIE PAR VERSION CROISSANTE : sans tri, l'ordre viendrait de l'iteration de map, donc
		// changerait d'un lancement a l'autre et rendrait deux recensements incomparables.
		{"plusieurs, tries", map[int]int{20: 7, 6: 1}, "6x1, 20x7"},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if got := formatSchemas(c.in); got != c.veut {
				t.Errorf("formatSchemas = %q, attendu %q", got, c.veut)
			}
		})
	}
}
