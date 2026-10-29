package archlint

// copies_identiques_test.go — LES COPIES JUMELLES RESTENT IDENTIQUES (plan D2-D, regle 6 de CLAUDE.md).
//
// Le decodeur de film est une brique qui n'importe rien du depot (`briques_imports_test.go`). Ce qu'il
// tire d'un paquet que LevelUp utilise aussi, il le COPIE sous son `internal/` plutot que de
// l'importer : deux copies, donc un test qui refuse qu'elles divergent. Une copie qui diverge en
// silence est la panne que la regle 6 nomme.
//
// # CE QU'IL EXIGE
//
// Pour chaque couple (original, copie), chaque fichier `.go` de l'original existe dans la copie
// et porte le MEME CONTENU, octet pour octet, a partir de la ligne qui suit la clause `package`.
// Tout ce qui precede cette ligne (commentaire de paquet, y compris l'en-tete qui nomme la copie
// jumelle) est libre. La copie ne porte aucun fichier de plus.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Changer une ligne de code (hors commentaire de paquet) de
// `film/internal/atomicfile/atomicfile.go` sans toucher `platform/atomicfile/atomicfile.go`.

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// coupleJumeau : un paquet de LevelUp et sa copie dans la brique, relatifs a `apps/go-api`.
type coupleJumeau struct{ original, copie string }

var copiesIdentiques = []coupleJumeau{
	{
		original: "internal/platform/atomicfile",
		copie:    "internal/games/halo_infinite/film/internal/atomicfile",
	},
}

func TestLesCopiesJumellesSontIdentiques(t *testing.T) {
	racine := apiRootDepuisIci(t)
	for _, c := range copiesIdentiques {
		for _, e := range ecartsDeCopie(t, racine, c) {
			t.Errorf("%s -> %s : %s", c.original, c.copie, e)
		}
	}
}

// ecartsDeCopie liste les ecarts entre l'original et sa copie.
func ecartsDeCopie(t *testing.T, racine string, c coupleJumeau) []string {
	t.Helper()
	orig := fichiersGoDuDossier(t, filepath.Join(racine, filepath.FromSlash(c.original)))
	copie := fichiersGoDuDossier(t, filepath.Join(racine, filepath.FromSlash(c.copie)))
	if len(orig) == 0 {
		t.Fatalf("%s ne porte aucun fichier Go : le test ne garde rien", c.original)
	}
	var ecarts []string
	for nom, a := range orig {
		b, ok := copie[nom]
		if !ok {
			ecarts = append(ecarts, "fichier absent de la copie : "+nom)
			continue
		}
		if !bytes.Equal(apresClausePackage(a), apresClausePackage(b)) {
			ecarts = append(ecarts, "contenu different apres la clause package : "+nom)
		}
	}
	for nom := range copie {
		if _, ok := orig[nom]; !ok {
			ecarts = append(ecarts, "fichier en trop dans la copie : "+nom)
		}
	}
	sort.Strings(ecarts)
	return ecarts
}

// fichiersGoDuDossier lit les `.go` d'un dossier (sans descendre), nom -> contenu.
func fichiersGoDuDossier(t *testing.T, dossier string) map[string][]byte {
	t.Helper()
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dossier, err)
	}
	out := map[string][]byte{}
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dossier, e.Name()))
		if err != nil {
			t.Fatalf("lecture de %s : %v", e.Name(), err)
		}
		out[e.Name()] = b
	}
	return out
}

// apresClausePackage rend le contenu qui suit la ligne `package x` (fins de ligne normalisees :
// un depot Windows peut les rendre en CRLF).
func apresClausePackage(src []byte) []byte {
	src = bytes.ReplaceAll(src, []byte("\r\n"), []byte("\n"))
	for _, ligne := range bytes.SplitAfter(src, []byte("\n")) {
		if bytes.HasPrefix(ligne, []byte("package ")) {
			return src[bytes.Index(src, ligne)+len(ligne):]
		}
	}
	return src
}

// TestCopiesIdentiques_AutoPreuve : la comparaison voit une ligne de code changee, ignore
// l'en-tete d'avant la clause package, et signale un fichier en trop.
func TestCopiesIdentiques_AutoPreuve(t *testing.T) {
	a := []byte("// Package x.\npackage x\n\nfunc F() int { return 1 }\n")
	b := []byte("// COPIE JUMELLE.\n// Package x.\npackage x\n\nfunc F() int { return 1 }\n")
	c := []byte("// Package x.\npackage x\n\nfunc F() int { return 2 }\n")
	if !bytes.Equal(apresClausePackage(a), apresClausePackage(b)) {
		t.Error("un en-tete different avant la clause package ne doit pas compter")
	}
	if bytes.Equal(apresClausePackage(a), apresClausePackage(c)) {
		t.Error("une ligne de code changee doit etre vue")
	}
}
