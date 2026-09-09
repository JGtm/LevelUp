// Garde-rail : les DEUX factories de HomeService doivent cabler les memes
// enrichissements de contenu.
//
// Il en existe deux, et c'est voulu : HomeCtxWithAuth (registry_auth.go) sert
// l'endpoint /pages/home, HomeCtx (registry_pages_home.go) sert l'injection
// OpenGraph. Elles different LEGITIMEMENT sur l'authentification (tokens Halo,
// provider, sinks de persistance) — mais tout ce qui remplit le CONTENU des tuiles
// doit exister des deux cotes.
//
// Cet invariant a lache DEUX FOIS, chaque fois en silence :
//   - WithRoundsDecide : la tuile affichait « 181 - 186 » la ou la page du match
//     affichait « 2 - 1 » ;
//   - WithReplay (2026-09-09) : pose sur la seule jumelle OpenGraph, donc
//     `replaySvc` nil sur /pages/home et AUCUNE tuile n'a jamais porte has_replay,
//     alors que les artefacts existaient sur le disque et que la colonne « Rejeu »
//     de l'Explorer, elle, s'affichait.
//
// Le mode de panne est le meme et explique pourquoi il passe inapercu : le service
// degrade en silence sur un champ nil (dependance optionnelle, jamais une erreur).
// Ni le typage ni les tests de service ne peuvent l'attraper — seul un test de
// PARITE du cablage le peut. D'ou une lecture de source : c'est la seule facon de
// comparer deux listes d'options de construction.
package wire

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// Options presentes chez HomeCtx uniquement et dont l'absence chez HomeCtxWithAuth
// est DELIBEREE. Vide aujourd'hui : toute entree ici doit porter sa justification.
var homeCtxOnlyOptions = map[string]string{}

// blocDeFonction retourne le corps de `nom` dans `source`, de sa signature a la
// premiere accolade fermante en colonne 0 (les fonctions du paquet sont formatees
// par gofmt : l'accolade de fin est toujours seule en debut de ligne).
func blocDeFonction(t *testing.T, chemin, nom string) string {
	t.Helper()
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture %s : %v", chemin, err)
	}
	source := string(brut)
	debut := strings.Index(source, "func (r *ServiceRegistry) "+nom+"(")
	if debut < 0 {
		t.Fatalf("factory %s introuvable dans %s", nom, chemin)
	}
	reste := source[debut:]
	if fin := strings.Index(reste, "\n}\n"); fin >= 0 {
		return reste[:fin]
	}
	return reste
}

var appelOption = regexp.MustCompile(`\bWith([A-Za-z]+)\(`)

// optionsDe extrait les noms des options `With*` appelees dans un bloc de fonction.
func optionsDe(bloc string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, m := range appelOption.FindAllStringSubmatch(bloc, -1) {
		out["With"+m[1]] = struct{}{}
	}
	return out
}

func TestFactoriesHome_MemeCablageDeContenu(t *testing.T) {
	avecAuth := optionsDe(blocDeFonction(t, "registry_auth.go", "HomeCtxWithAuth"))
	openGraph := optionsDe(blocDeFonction(t, "registry_pages_home.go", "HomeCtx"))

	var manquantes []string
	for opt := range openGraph {
		if _, ok := avecAuth[opt]; ok {
			continue
		}
		if _, tolere := homeCtxOnlyOptions[opt]; tolere {
			continue
		}
		manquantes = append(manquantes, opt)
	}
	sort.Strings(manquantes)

	if len(manquantes) > 0 {
		t.Errorf(
			"HomeCtx (image OpenGraph) cable %v, pas HomeCtxWithAuth (endpoint /pages/home) :\n"+
				"l'Accueil servira ces donnees vides, en silence.\n"+
				"Cabler la ou les options manquantes dans registry_auth.go, ou les inscrire dans\n"+
				"homeCtxOnlyOptions avec la raison de leur absence.",
			manquantes,
		)
	}
}

// Sentinelle nominative : le rejeu est le cas qui a motive ce garde-rail. Meme si
// quelqu'un elargissait homeCtxOnlyOptions un jour, celui-la doit rester cable.
func TestFactoriesHome_RejeuCableSurEndpointAccueil(t *testing.T) {
	avecAuth := optionsDe(blocDeFonction(t, "registry_auth.go", "HomeCtxWithAuth"))
	if _, ok := avecAuth["WithReplay"]; !ok {
		t.Error(
			"HomeCtxWithAuth n'appelle pas WithReplay : has_replay restera faux sur toutes " +
				"les tuiles de l'Accueil et le bouton de rejeu ne s'affichera jamais, " +
				"artefacts presents ou non.",
		)
	}
}
