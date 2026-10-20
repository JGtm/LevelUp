package main

// livraison_golden_test.go — LA FIDELITE OCTET A OCTET DU MODE `livrer`, gardee par un test.
//
// # Pourquoi ce test existe
//
// L'objet du portage de `_outils/livraison.py` etait de produire les MEMES OCTETS que le
// script Python : les `hinf_*.wav` de `static/sounds/halo_infinite/` et
// `weaponSoundVariations.ts`. Cette propriete n'etait gardee par AUCUN test du depot — la
// seule preuve vivait dans un scratchpad de session, disparu avec elle. Une mutation de
// l'arrondi du mixage (`int16(v*att)` -> `int16(math.Round(v*att))`) et de la troncature
// (1,2 s -> 1,25 s) laissait la suite VERTE en changeant deux des fichiers livres (constat C3
// de la revue R1, mutation M3).
//
// # D'ou viennent les goldens, et comment les refaire
//
// `testdata/livraison/goldens/` a ete produit UNE FOIS par le script Python d'origine, sur
// l'arborescence que `livraisonEcrireJeuSynthetique` ecrit. LES REGENERER AVEC LE CODE GO
// ANNULERAIT LA PREUVE : le test ne dirait plus que « Go est d'accord avec Go ». La recette,
// si le jeu d'entrees doit changer :
//
//  1. materialiser l'arborescence dans un dossier hors depot, par un test jetable qui
//     appelle `livraisonEcrireJeuSynthetique(<racine>)` puis
//     `livraisonEcrireDepotSynthetique(<racine>/_depot)` ;
//  2. copier `_outils/livraison.py` ET `_outils/coups_lot.py` (archive Desktop du chantier,
//     hors depot, JAMAIS modifies) dans `<racine>/_outils/`, verifier les copies au `cmp` ;
//  3. `python livraison.py <racine>/_depot` depuis `<racine>/_outils` ;
//  4. recopier les `hinf_*.wav`, le `weaponSoundVariations.ts` et la sortie console (fins de
//     ligne normalisees en LF) dans `testdata/livraison/goldens/`.
//
// # La seule divergence VOULUE
//
// Les trois lignes d'en-tete du `.ts` nomment le producteur : le golden porte celui du script
// Python, le mode Go porte le sien. Le test compare donc le corps octet pour octet, et
// l'en-tete au gabarit Go — il ne les efface ni l'un ni l'autre.

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// livraisonGoldensDir : la racine des sorties de reference.
const livraisonGoldensDir = "testdata/livraison/goldens"

func TestLivrerOctetPourOctet(t *testing.T) {
	racine := t.TempDir()
	if err := livraisonEcrireJeuSynthetique(racine); err != nil {
		t.Fatalf("jeu synthetique: %v", err)
	}
	depot := filepath.Join(racine, "_depot")
	if err := livraisonEcrireDepotSynthetique(depot); err != nil {
		t.Fatalf("depot synthetique: %v", err)
	}

	console, err := livraisonCapturerSortie(func() error {
		return livrer(filepath.Join(racine, "_donnees"), racine, depot)
	})
	if err != nil {
		t.Fatalf("livrer: %v", err)
	}

	livraisonComparerArmes(t, filepath.Join(depot, "static", "sounds", "halo_infinite"))
	livraisonComparerTS(t, filepath.Join(depot, "apps", "web", "src", "features", "match-replay", "sound",
		"weaponSoundVariations.ts"))
	livraisonComparerConsole(t, console)
}

// livraisonComparerArmes verifie que la cible contient EXACTEMENT les armes de reference,
// octet pour octet, plus le seul temoin hors perimetre du miroir.
func livraisonComparerArmes(t *testing.T, cible string) {
	t.Helper()
	goldens, err := filepath.Glob(filepath.Join(livraisonGoldensDir, "hinf_*.wav"))
	if err != nil || len(goldens) == 0 {
		t.Fatalf("goldens introuvables sous %s (%v)", livraisonGoldensDir, err)
	}
	var attendus []string
	for _, g := range goldens {
		nom := filepath.Base(g)
		attendus = append(attendus, nom)
		veut, rerr := os.ReadFile(g)
		if rerr != nil {
			t.Fatalf("lecture du golden %s: %v", g, rerr)
		}
		got, rerr := os.ReadFile(filepath.Join(cible, nom))
		if rerr != nil {
			t.Errorf("%s absent de la cible: %v", nom, rerr)
			continue
		}
		if !bytes.Equal(got, veut) {
			t.Errorf("%s DIFFERE du fichier produit par _outils/livraison.py "+
				"(%d octets produits, %d attendus) — le mode `livrer` doit rendre les MEMES "+
				"octets que le script qu'il remplace", nom, len(got), len(veut))
		}
	}
	// LE MIROIR : l'arme perimee a disparu, le son d'evenement du pack est intact.
	attendus = append(attendus, "melee_kill.wav")
	sort.Strings(attendus)
	entrees, err := os.ReadDir(cible)
	if err != nil {
		t.Fatalf("lecture de la cible: %v", err)
	}
	var presents []string
	for _, e := range entrees {
		presents = append(presents, e.Name())
	}
	sort.Strings(presents)
	if strings.Join(presents, " ") != strings.Join(attendus, " ") {
		t.Errorf("contenu de la cible = %v, attendu %v (hinf_perime.wav doit avoir disparu, "+
			"melee_kill.wav doit etre intact)", presents, attendus)
	}
}

// livraisonComparerTS compare le CORPS du fichier genere au golden Python, et son en-tete au
// gabarit Go — la seule divergence voulue du portage.
func livraisonComparerTS(t *testing.T, chemin string) {
	t.Helper()
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de weaponSoundVariations.ts: %v", err)
	}
	veutBrut, err := os.ReadFile(filepath.Join(livraisonGoldensDir, "weaponSoundVariations.ts"))
	if err != nil {
		t.Fatalf("lecture du golden .ts: %v", err)
	}
	enteteGot, corpsGot, ok := strings.Cut(string(brut), livraisonTSMarqueur)
	if !ok {
		t.Fatalf("le fichier produit ne contient pas %q", livraisonTSMarqueur)
	}
	_, corpsVeut, ok := strings.Cut(string(veutBrut), livraisonTSMarqueur)
	if !ok {
		t.Fatalf("le golden ne contient pas %q", livraisonTSMarqueur)
	}
	if corpsGot != corpsVeut {
		t.Errorf("le corps de weaponSoundVariations.ts differe du golden Python.\n"+
			"--- produit ---\n%s\n--- attendu ---\n%s", corpsGot, corpsVeut)
	}
	enteteVeut, _, _ := strings.Cut(livraisonTSTemplate, livraisonTSMarqueur)
	if enteteGot != enteteVeut {
		t.Errorf("l'en-tete produit n'est pas celui du gabarit Go.\n--- produit ---\n%s\n"+
			"--- gabarit ---\n%s", enteteGot, enteteVeut)
	}
}

// livraisonComparerConsole compare la sortie console au releve du script Python. Elle attrape
// ce que les octets ne disent pas : l'ORDRE de livraison, la source retenue pour chaque arme,
// les lignes INTROUVABLE / SANS FICHIER / « variante servie par la base », et le colonnage.
func livraisonComparerConsole(t *testing.T, console string) {
	t.Helper()
	veut, err := os.ReadFile(filepath.Join(livraisonGoldensDir, "console.txt"))
	if err != nil {
		t.Fatalf("lecture du golden console: %v", err)
	}
	if console != string(veut) {
		t.Errorf("la sortie console differe de celle du script Python.\n--- produite ---\n%s\n"+
			"--- attendue ---\n%s", console, string(veut))
	}
}

// livraisonCapturerSortie execute `f` en detournant os.Stdout, et rend ce qui y a ete ecrit.
//
// LA LECTURE SE FAIT DANS UNE GOROUTINE : un tube dont personne ne vide le tampon bloque
// l'ecrivain, et le test se figerait au lieu d'echouer.
func livraisonCapturerSortie(f func() error) (string, error) {
	lecture, ecriture, err := os.Pipe()
	if err != nil {
		return "", err
	}
	ancien := os.Stdout
	os.Stdout = ecriture
	lu := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, lecture)
		lu <- buf.String()
	}()
	errF := f()
	os.Stdout = ancien
	_ = ecriture.Close()
	sortie := <-lu
	_ = lecture.Close()
	return sortie, errF
}
