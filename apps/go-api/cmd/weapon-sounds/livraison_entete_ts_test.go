package main

// livraison_entete_ts_test.go — LE GARDE-FOU DE L'EN-TETE DU FICHIER GENERE.
//
// `apps/web/src/features/match-replay/sound/weaponSoundVariations.ts` est produit par le mode
// `livrer` et porte, en trois lignes de prose, le NOM DE SON PRODUCTEUR. Apres le portage
// Go, il annoncait toujours `_outils/livraison.py` — l'outil que la recette declare dans le
// meme lot « NE PLUS L'UTILISER » : le seul pointeur du fichier genere designait le
// producteur retire (anti-pattern « doc inversee », CLAUDE.md diagnostic n°9, constat C2 de
// la revue R1).
//
// Le fichier versionne ne peut pas etre regenere sur ce poste (les `.wav` sources du
// chantier ont disparu), et il ne le sera qu'a la prochaine livraison reelle. Ce test tient
// donc la promesse a sa place : l'en-tete du fichier versionne est, OCTET POUR OCTET, celui
// que le gabarit Go produirait.

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// livraisonTSMarqueur : la premiere ligne qui suit l'en-tete du fichier genere.
const livraisonTSMarqueur = "import type { SoundVariation } from './weaponSoundLogic'"

func TestEnTeteTSVersionneeSuitLeGabarit(t *testing.T) {
	_, ceFichier, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	// cmd/weapon-sounds -> cmd -> apps/go-api -> apps -> racine du depot
	racine := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(ceFichier)))))
	chemin := filepath.Join(racine, "apps", "web", "src", "features", "match-replay", "sound", "weaponSoundVariations.ts")
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture du fichier genere %s: %v", chemin, err)
	}
	versionne, _, okV := strings.Cut(string(brut), livraisonTSMarqueur)
	if !okV {
		t.Fatalf("%s ne contient plus %q — le gabarit et le fichier ont divergé", chemin, livraisonTSMarqueur)
	}
	attendu, _, okA := strings.Cut(livraisonTSTemplate, livraisonTSMarqueur)
	if !okA {
		t.Fatalf("livraisonTSTemplate ne contient plus %q", livraisonTSMarqueur)
	}
	if versionne != attendu {
		t.Errorf("l'en-tete du fichier genere versionne n'est pas celui que `livrer` produirait.\n"+
			"Un fichier genere qui nomme un producteur retire est une doc inversee.\n--- versionne ---\n%s\n--- gabarit ---\n%s",
			versionne, attendu)
	}
}
