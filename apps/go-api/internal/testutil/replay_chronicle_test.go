package testutil

// replay_chronicle_test.go — L'EXTRACTEUR DE CHRONIQUE EST-IL ENCORE BRANCHE ?
//
// Un extracteur muet ne casse rien : il rend une liste vide, et les deux garde-rails qui en
// dependent passent au vert en ne verifiant plus rien. C'est le pire mode de defaillance d'un
// ratchet, et ce fichier est ce qui l'empeche — sur une fixture qui porte les TROIS formes
// d'en-tete attestees ET les leurres qui les entourent dans le vrai fichier.

import (
	"fmt"
	"testing"
)

// chroniqueFixture : les trois formes reelles, et quatre leurres copies du vrai fichier.
const chroniqueFixture = `package replay
//
// v3 (2026-08-13, plan parite lot 2) : le lancer de grenade publie son LIEN.
//
// v6 doit se voir comme « a re-cuire », pas comme a jour.
//
// SCHEMA 29 (2026-08-31) — LA LUNETTE. Le palier de visee a la lunette est publie.
//
// CE QUE LA VERSION 22 PORTE, ET POURQUOI ELLE MONTE. La COURONNE VIP est publiee.
//
// CE QUE LA VERSION 31 REFUSE. De descendre un libelle.
//
// v33 vise l'anneau d'armement, pas ce canal.
//
// v22 : la reprise du backfill se fait par SchemaVersion.
//
// v14 (drapeau), v16 (zones) et v21 (proprietaire de colline) : la reprise du backfill.
`

// TestChronicleVersionsFrom : les trois formes entrent, les leurres restent dehors.
//
// LE CAS « v14 (drapeau) » EST DELIBERE et il est celui qui compte : la forme d'en-tete est
// indiscernable, en DEBUT DE LIGNE, d'une citation qui commence par une version. L'extracteur
// le compte, et c'est sans consequence — 14 est une vraie entree par ailleurs, et la liste
// dedoublonne. Ce qui serait grave (compter 6, 33, ou le « v22 : » d'une phrase) ne l'est pas.
func TestChronicleVersionsFrom(t *testing.T) {
	const want = "[3 14 22 29 31]"
	if got := fmt.Sprint(chronicleVersionsFrom(chroniqueFixture)); got != want {
		t.Errorf("versions extraites %s, attendu %s", got, want)
	}
}

// TestChronicleVersionsFromNeDevinePas : une fixture SANS aucune en-tete rend une liste vide —
// c'est ce que le lecteur disque traite comme une erreur, pas comme un succes silencieux.
func TestChronicleVersionsFromNeDevinePas(t *testing.T) {
	if got := chronicleVersionsFrom("package replay\n// rien a declarer\n"); len(got) != 0 {
		t.Errorf("un texte sans en-tete rend %v, attendu vide", got)
	}
}

// TestReplayChronicleVersionsLitLeVraiFichier : le chemin construit tombe-t-il bien sur la
// chronique du depot ? Sans cette assertion, un deplacement de fichier rendrait le helper
// muet et les deux garde-rails inertes.
func TestReplayChronicleVersionsLitLeVraiFichier(t *testing.T) {
	versions, err := ReplayChronicleVersions()
	if err != nil {
		t.Fatalf("lecture de la chronique : %v", err)
	}
	// 51 entrees au 2026-09-13 (v2 a v54, moins les numeros 32 et 51 sautes a la
	// renumerotation de deux lots paralleles). Le seuil borne par le bas : c'est la
	// PRESENCE qui est figee, pas le compte, qui monte a chaque lot.
	if len(versions) < 40 {
		t.Errorf("la chronique ne declare que %d versions (%v) — la forme des en-tetes a-t-elle "+
			"change ?", len(versions), versions)
	}
}
