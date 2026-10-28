//go:build gamefiles

package himap

// GARDE-RAIL DU DÉCODEUR uslg — il exige le jeu installé et se déclare absent sinon.
//
// Il vérifie que le vocabulaire des lieux (tag `locs`) est lu en entier et que la liste de
// chaînes retenue nomme chaque lieu en anglais et en français. La preuve que le décodage
// reproduit le lexique VERSIONNÉ (une donnée de LevelUp) vit dans le paquet de tests de
// LevelUp `internal/gamefilescorpus` (uslg_gamefiles_test.go) : la brique ne lit aucune donnée
// du dépôt.

import (
	"os"
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
		if !ok || l[CodeLangueEN] == "" || l[CodeLangueFR] == "" {
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
