//go:build gamefiles

package gamefilescorpus

// GARDE-RAIL DE LA CHAÎNE uslg -> lexique versionné — il exige le jeu installé et se déclare
// absent sinon.
//
// Le témoin hors ligne (cmd/mapcallouts-build/lexique_test.go) vérifie le FICHIER PRODUIT.
// Celui-ci vérifie la CHAÎNE : que le décodeur de la brique `himap`, relancé sur les fichiers du
// jeu, reproduit exactement ce fichier (`callouts_lexique.csv`, une donnée de référence de
// LevelUp, d'où la place de ce test hors de la brique). Les deux ensemble ferment la boucle — un
// décodeur qui dérive fait rougir celui-ci, un lexique régénéré de travers fait rougir l'autre.

import (
	"encoding/csv"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/himap"
	"levelup/go-api/internal/i18n"
)

// racineJeu rend l'installation du jeu, ou déclare le test absent si elle ou le module des
// tags globaux manque.
func racineJeu(t *testing.T) string {
	t.Helper()
	r, err := himap.DeployRoot()
	if err != nil {
		t.Skipf("pas d installation : %v", err)
	}
	if _, err := himap.VocabulaireLieux(r); err != nil && errors.Is(err, fs.ErrNotExist) {
		t.Skipf("globals absent : %v", err)
	}
	return r
}

// langueDuJeu traduit un code de langue rendu par himap vers la langue de l'application.
func langueDuJeu(t *testing.T, code string) i18n.Locale {
	t.Helper()
	l, ok := i18n.FromHaloTag(code)
	if !ok {
		t.Fatalf("code de langue %q du jeu absent de config/locales.toml", code)
	}
	return l
}

// TestLexiqueLieuxReproduitLeFichierVersionne — L'ANTI-RÉGRESSION. Le décodeur relancé sur
// les fichiers du jeu doit rendre EXACTEMENT le lexique versionné : mêmes clés, mêmes
// textes. Une divergence signifie soit un décodeur cassé, soit une mise à jour du jeu —
// dans les deux cas il faut regarder avant de republier, jamais laisser passer.
func TestLexiqueLieuxReproduitLeFichierVersionne(t *testing.T) {
	lex, err := himap.LexiqueLieux(racineJeu(t))
	if err != nil {
		t.Fatalf("LexiqueLieux : %v", err)
	}
	fichier := litLexiqueVersionne(t)
	en, fr := langueDuJeu(t, himap.CodeLangueEN), langueDuJeu(t, himap.CodeLangueFR)
	if len(fichier) == 0 {
		t.Skip("lexique versionné absent")
	}
	manquants, divergents := 0, 0
	for sid, attendu := range fichier {
		got, ok := lex[sid]
		if !ok {
			manquants++
			if manquants <= 5 {
				t.Errorf("StringId %08X du fichier absent du décodage", sid)
			}
			continue
		}
		if got[himap.CodeLangueEN] != attendu[en] || got[himap.CodeLangueFR] != attendu[fr] {
			divergents++
			if divergents <= 5 {
				t.Errorf("StringId %08X : décodé (%q, %q) != fichier (%q, %q)",
					sid, got[himap.CodeLangueEN], got[himap.CodeLangueFR], attendu[en], attendu[fr])
			}
		}
	}
	// L'inverse aussi : une entrée décodée que le fichier ignore = fichier périmé.
	nouvelles := 0
	for sid, l := range lex {
		if l[himap.CodeLangueEN] == "" || l[himap.CodeLangueFR] == "" {
			continue
		}
		if _, ok := fichier[sid]; !ok {
			nouvelles++
		}
	}
	if manquants != 0 || divergents != 0 || nouvelles != 0 {
		t.Fatalf("lexique versionné (%d entrées) vs décodage : %d manquants, %d divergents, %d nouvelles "+
			"— régénérer par `mapcallouts-build --lexique` après vérification", len(fichier), manquants, divergents, nouvelles)
	}
	t.Logf("%d entrées reproduites à l'identique", len(fichier))
}

// litLexiqueVersionne relit le CSV de référence sans dépendre du paquet qui l'écrit.
func litLexiqueVersionne(t *testing.T) map[uint32]i18n.Text {
	t.Helper()
	p := filepath.Join("..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "callouts_lexique.csv")
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("lexique illisible : %v", err)
	}
	out := map[uint32]i18n.Text{}
	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}
		sid, err := strconv.ParseUint(strings.TrimPrefix(row[0], "0x"), 16, 32)
		if err != nil {
			t.Fatalf("ligne %d : string_id %q", i+1, row[0])
		}
		out[uint32(sid)] = i18n.FromColumns(row[1], row[2])
	}
	return out
}
