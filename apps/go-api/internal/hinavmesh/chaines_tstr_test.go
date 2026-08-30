package hinavmesh

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LES DEUX ECRITURES DE LA TABLE DE CHAINES — etat au 2026-08-30, moitie du chemin fait.
//
// Trois cartes sont bloquees en « bouillie » faute de maillage lisible : Absolution, Insolence et
// Insolence Heavies. Le decodeur les refusait sur « fichier-tag sans section TST1 ».
//
// CE QUI EST ETABLI ET CORRIGE. Leur section TYPE porte exactement les memes voisins que celle
// d Isolation — TPTR, TNA1, TBDY, THSH, TPAD — mais nomme ses deux tables de chaines TSTR et FSTR
// au lieu de TST1 et FST1. Ce sont deux ecritures du meme format ; `sectionsChaines` accepte
// desormais les deux, et le decodage va bien plus loin qu avant.
//
// CE QUI RESTE. La table des noms de champs se revele alors trop COURTE d une entree : un membre
// demande l indice 98 d une table qui en porte 98. Deux pistes ont ete essayees et ECARTEES :
// prepender la chaine vide (indexation a partir de 1) decale tout et corrompt les noms de types —
// `hkPropertyId` devient `tITEM` — donc l origine n est pas en cause ; et la table n est pas
// tronquee par le decoupage, qui compte deja l entree vide finale. Il reste a etablir comment
// cette generation encode ses chaines : longueur prefixee, en-tete de section, ou table partagee
// avec TSTR.
//
// LE TEMOIN FIGE LES DEUX MOITIES : Isolation doit se decoder entierement, et Absolution doit
// echouer PLUS LOIN que la section manquante. Si quelqu un retire la reconnaissance de TSTR/FSTR,
// la seconde assertion tombe.
func TestLesDeuxEcrituresDeTableDeChaines(t *testing.T) {
	const racine = `C:/Users/Guillaume/Projects/LevelUp/.ai/re_dump/navmesh`
	lis := func(f string) []byte {
		brut, err := os.ReadFile(filepath.Join(racine, f))
		if err != nil {
			t.Skipf("blob absent du depot local (%v) — les navmesh ne sont pas versionnes", err)
		}
		return brut
	}

	m, err := Decode(lis("01af558d-53ab-4f05-ba68-92d805fc6260.blob"))
	if err != nil {
		t.Fatalf("Isolation (TST1/FST1) : decodage refuse : %v", err)
	}
	if len(m.Faces) == 0 || len(m.Sommets) == 0 {
		t.Fatalf("Isolation : maillage vide — %d faces, %d sommets", len(m.Faces), len(m.Sommets))
	}

	_, err = Decode(lis("78da545f-a168-4a5e-9c8d-dd379067c352.blob"))
	if err == nil {
		t.Log("Absolution (TSTR/FSTR) se decode entierement : le reste du chemin a ete fait, " +
			"mettre ce temoin a jour pour l exiger")
		return
	}
	if strings.Contains(err.Error(), "sans table de chaines") || strings.Contains(err.Error(), "sans section TST1") {
		t.Fatalf("Absolution : le decodeur ne reconnait plus l ecriture TSTR/FSTR — regression : %v", err)
	}
	t.Logf("Absolution (TSTR/FSTR) : sections reconnues, blocage restant en aval : %v", err)
}
