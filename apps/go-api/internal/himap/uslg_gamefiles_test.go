//go:build gamefiles

package himap

// GARDE-RAIL DU DÉCODEUR uslg — il exige le jeu installé et se déclare absent sinon.
//
// Le témoin hors ligne (cmd/mapcallouts-build/lexique_test.go) vérifie le FICHIER PRODUIT.
// Celui-ci vérifie la CHAÎNE : que le décodeur, relancé sur les fichiers du jeu, reproduit
// exactement ce fichier. Les deux ensemble ferment la boucle — un décodeur qui dérive fait
// rougir celui-ci, un lexique régénéré de travers fait rougir l'autre.

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func racineJeu(t *testing.T) string {
	t.Helper()
	r, err := DeployRoot()
	if err != nil {
		t.Skipf("pas d installation : %v", err)
	}
	if _, err := os.Stat(cheminGlobals(r)); err != nil {
		t.Skipf("globals absent : %v", err)
	}
	return r
}

// TestVocabulaireLieuxEstLeTableauDuTagLocs — le vocabulaire global des lieux : 778 entrées
// déclarées par le tag `locs`, toutes lues.
func TestVocabulaireLieuxEstLeTableauDuTagLocs(t *testing.T) {
	vocab, err := VocabulaireLieux(racineJeu(t))
	if err != nil {
		t.Fatalf("VocabulaireLieux : %v", err)
	}
	if len(vocab) != 778 {
		t.Fatalf("%d StringId de lieu, 778 attendus (le tag locs a changé de taille)", len(vocab))
	}
	distincts := map[uint32]bool{}
	for _, v := range vocab {
		distincts[v] = true
	}
	t.Logf("%d entrées, %d StringId distincts", len(vocab), len(distincts))
}

// TestLexiqueLieuxCouvreToutLeVocabulaire — la liste de chaînes retenue doit nommer CHAQUE
// lieu du vocabulaire global. C'est ce qui garantit qu'une carte Forge, quel que soit le
// nom qu'elle pioche, trouvera un texte.
func TestLexiqueLieuxCouvreToutLeVocabulaire(t *testing.T) {
	racine := racineJeu(t)
	vocab, err := VocabulaireLieux(racine)
	if err != nil {
		t.Fatalf("VocabulaireLieux : %v", err)
	}
	lex, err := LexiqueLieux(racine)
	if err != nil {
		t.Fatalf("LexiqueLieux : %v", err)
	}
	manquants := 0
	for _, v := range vocab {
		l, ok := lex[v]
		if !ok || l.EN == "" || l.FR == "" {
			manquants++
			if manquants <= 5 {
				t.Errorf("StringId %08X sans texte complet (%+v)", v, l)
			}
		}
	}
	if manquants != 0 {
		t.Fatalf("%d des %d noms de lieu sans texte EN/FR", manquants, len(vocab))
	}
	t.Logf("lexique : %d entrées, les %d noms de lieu du vocabulaire sont tous nommés en EN et FR",
		len(lex), len(vocab))
}

// TestLexiqueLieuxReproduitLeFichierVersionne — L'ANTI-RÉGRESSION. Le décodeur relancé sur
// les fichiers du jeu doit rendre EXACTEMENT le lexique versionné : mêmes clés, mêmes
// textes. Une divergence signifie soit un décodeur cassé, soit une mise à jour du jeu —
// dans les deux cas il faut regarder avant de republier, jamais laisser passer.
func TestLexiqueLieuxReproduitLeFichierVersionne(t *testing.T) {
	lex, err := LexiqueLieux(racineJeu(t))
	if err != nil {
		t.Fatalf("LexiqueLieux : %v", err)
	}
	fichier := litLexiqueVersionne(t)
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
		if got.EN != attendu.EN || got.FR != attendu.FR {
			divergents++
			if divergents <= 5 {
				t.Errorf("StringId %08X : décodé (%q, %q) != fichier (%q, %q)",
					sid, got.EN, got.FR, attendu.EN, attendu.FR)
			}
		}
	}
	// L'inverse aussi : une entrée décodée que le fichier ignore = fichier périmé.
	nouvelles := 0
	for sid, l := range lex {
		if l.EN == "" || l.FR == "" {
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
func litLexiqueVersionne(t *testing.T) map[uint32]LibelleLieu {
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
	out := map[uint32]LibelleLieu{}
	for i, row := range rows {
		if i == 0 || len(row) < 3 {
			continue
		}
		sid, err := strconv.ParseUint(strings.TrimPrefix(row[0], "0x"), 16, 32)
		if err != nil {
			t.Fatalf("ligne %d : string_id %q", i+1, row[0])
		}
		out[uint32(sid)] = LibelleLieu{EN: row[1], FR: row[2]}
	}
	return out
}
