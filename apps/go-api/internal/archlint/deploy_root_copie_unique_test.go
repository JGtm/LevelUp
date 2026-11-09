package archlint

import (
	"sort"
	"strings"
	"testing"
)

// deploy_root_copie_unique_test.go — le chemin de l'installation du jeu n'existe qu'à UN endroit
// de l'espace de travail (règle 6 de CLAUDE.md : au plus deux copies d'un même littéral).
//
// POURQUOI. Le chemin `.../SteamLibrary/steamapps/common/Halo Infinite/deploy` était écrit en dur
// à trois endroits, tous pointant vers un disque qui n'existe plus : les tests se déclaraient
// « absents » et passaient au vert. `himap.DeployRoot` et `himap.LevelsDir` sont la source
// canonique ; ce garde-rail interdit le littéral partout ailleurs.
//
// POURQUOI ICI ET PAS DANS LA BRIQUE. Il vivait dans `himap` (D1 lot 2) et balayait `apps/go-api`.
// Depuis que `himap` est un module sous `libs/` (ADR 0041, lot 4), un balayage fait depuis la
// brique ne verrait plus que `libs/` : le contrat « une seule copie dans TOUT l'espace de
// travail » ne peut se tenir que d'un garde-rail enraciné sur l'espace de travail, comme les
// autres ratchets à règle générale (`workspaceGoSources`).
//
// Mutation qui doit le faire rougir : recoller `SteamLibrary` dans un `cmd/` ou dans un paquet de
// `libs/gamefiles`.

// motifsCheminDuJeu : les fragments qui n'apparaissent que dans un chemin d'installation du jeu.
var motifsCheminDuJeu = []string{"SteamLibrary", "XboxGames", "steamapps"}

// sourcesCheminDuJeuAutorisees : le fichier qui DÉCLARE les racines (la source canonique) et ce
// garde-rail lui-même (il cite les motifs). Chemins relatifs à la racine du dépôt, en slash.
var sourcesCheminDuJeuAutorisees = map[string]bool{
	"libs/himap/himap/deploy_root.go":                                true,
	"apps/go-api/internal/archlint/deploy_root_copie_unique_test.go": true,
}

// copiesDuCheminDuJeu rend, pour un fichier, les motifs d'un chemin d'installation qu'il porte
// quand il n'est pas autorisé à les porter. Fonction pure : c'est elle que l'auto-preuve exerce.
func copiesDuCheminDuJeu(rel, texte string, autorises map[string]bool) []string {
	if autorises[rel] {
		return nil
	}
	var trouves []string
	for _, m := range motifsCheminDuJeu {
		if strings.Contains(texte, m) {
			trouves = append(trouves, m)
		}
	}
	return trouves
}

func TestAucuneAutreCopieDuCheminDuJeu(t *testing.T) {
	vus := map[string]bool{}
	var violations []string
	balayerGoTexte(t, func(rel, texte string) {
		vus[rel] = true
		for _, m := range copiesDuCheminDuJeu(rel, texte, sourcesCheminDuJeuAutorisees) {
			violations = append(violations, rel+" contient "+m)
		}
	})
	for rel := range sourcesCheminDuJeuAutorisees {
		if !vus[rel] {
			t.Errorf("source autorisée %q introuvable — retirer l'entrée ou corriger le chemin", rel)
		}
	}
	if len(vus) == 0 {
		t.Fatal("aucun fichier Go balayé — le balayage s'est cassé")
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("le chemin de l'installation du jeu est écrit hors de sa source canonique "+
			"(libs/himap/himap/deploy_root.go) — passer par himap.DeployRoot et himap.LevelsDir :\n  %s",
			strings.Join(violations, "\n  "))
	}
}

// TestAucuneAutreCopieDuCheminDuJeu_AutoPreuve : une copie fictive dans un fichier quelconque de
// l'espace de travail (brique, application, test) est refusée ; la source canonique et le
// garde-rail lui-même passent ; un fichier sans motif passe.
func TestAucuneAutreCopieDuCheminDuJeu_AutoPreuve(t *testing.T) {
	copie := "const racine = `D:/SteamLibrary/steamapps/common/Halo Infinite/deploy`\n"
	for _, rel := range []string{
		"apps/go-api/cmd/mapfond-build/main.go",
		"libs/gamefiles/himodule/module.go",
		"libs/himap/himap/autre.go",
		"libs/himap/himap/deploy_root_test.go",
	} {
		got := copiesDuCheminDuJeu(rel, copie, sourcesCheminDuJeuAutorisees)
		if len(got) != 2 || got[0] != "SteamLibrary" || got[1] != "steamapps" {
			t.Errorf("%s : copie fictive attendue refusée (SteamLibrary, steamapps), obtenu %v", rel, got)
		}
	}
	if got := copiesDuCheminDuJeu("libs/himap/himap/deploy_root.go", copie, sourcesCheminDuJeuAutorisees); got != nil {
		t.Errorf("la source canonique ne doit pas être refusée, obtenu %v", got)
	}
	if got := copiesDuCheminDuJeu("apps/go-api/cmd/x/main.go", "package main\n", sourcesCheminDuJeuAutorisees); got != nil {
		t.Errorf("un fichier sans motif ne doit pas être refusé, obtenu %v", got)
	}
	if got := copiesDuCheminDuJeu("libs/x/y.go", "C:/XboxGames/Halo", nil); len(got) != 1 || got[0] != "XboxGames" {
		t.Errorf("motif XboxGames attendu refusé, obtenu %v", got)
	}
}
