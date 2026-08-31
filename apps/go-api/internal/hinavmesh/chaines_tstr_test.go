package hinavmesh

import (
	"os"
	"path/filepath"
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
// CE QUI RESTE, ET LE DIAGNOSTIC A ETE CORRIGE LE 2026-08-31. La table des noms de champs se
// revele alors trop COURTE d une entree : un membre demande l indice 98 d une table qui en porte
// 98. On a d abord lu cet ecart d un comme une ORIGINE D INDEXATION — c est FAUX. Le membre en
// question est le cinquantieme de hkPropertyId, un type qui n a pas cinquante membres : le flux
// TBDY est DESYNCHRONISE bien avant, et l indice hors bornes n est que le premier symptome
// visible. La mesure detaillee vit dans sonde_tbdy_test.go, avec la piste des drapeaux de membre
// 0x21/0x23 essayee et refutee.
//
// LE TEMOIN EXIGE LES DEUX : Isolation et Absolution doivent se decoder ENTIEREMENT. Absolution
// y arrive depuis le 2026-08-31 par trois corrections enchainees — l entree TBDY illisible
// franchie par resynchronisation mesuree, le maillage cherche par son TYPE et non par la racine
// (sa region porte un hkaiStaticTreeNavMeshQueryMediator), et le vecteur haut suppose faute
// d etre declare. Si l une des trois est retiree, ce temoin tombe.
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

	a, err := Decode(lis("78da545f-a168-4a5e-9c8d-dd379067c352.blob"))
	if err != nil {
		t.Fatalf("Absolution (TSTR/FSTR) : decodage refuse : %v", err)
	}
	if len(a.Faces) == 0 || len(a.Sommets) == 0 {
		t.Fatalf("Absolution : maillage vide — %d faces, %d sommets", len(a.Faces), len(a.Sommets))
	}
	// Le vecteur haut n est pas declare par cette generation : il est SUPPOSE, et le maillage
	// doit le dire. Une carte qui pretendrait l avoir lu serait plus inquietante que l inverse.
	if !a.HautSuppose {
		t.Fatal("Absolution : le maillage pretend avoir LU son vecteur haut, alors que sa " +
			"generation ne le declare pas")
	}
	if m.HautSuppose {
		t.Fatal("Isolation : vecteur haut declare comme suppose alors que le fichier le porte")
	}
	t.Logf("Isolation %d faces / %d sommets | Absolution %d faces / %d sommets",
		len(m.Faces), len(m.Sommets), len(a.Faces), len(a.Sommets))
}
