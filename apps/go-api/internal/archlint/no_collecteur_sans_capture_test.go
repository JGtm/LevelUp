// Package archlint — no_collecteur_sans_capture_test.go : ratchet « TOUT CHEMIN DE COLLECTE
// CABLE LA CAPTURE DES POSITIONS » (correction P0-1 de la revue de 7C, 2026-09-07).
//
// # CE QU'IL EMPECHE, ET CE QUE CA A COUTE
//
// `WithPositionCapture` n'etait appele QUE par `levelup backfill-killsource`. Ni l'etape
// post-sync du serveur ni `backfill-killsource --online` ne le cablaient : `collectPositions`
// sortait en Debug des sa deuxieme garde, et AUCUNE position n'a jamais ete produite AU FIL DU
// SYNC en production — depuis la mise en place de `kill_positions`, et pour les deux tables de
// faits d'isolement du lot 7C.
//
// LE DEFAUT ETAIT INVISIBLE : rien ne distingue, dans le code, un collecteur qui capture d'un
// collecteur qui ne capture pas. Aucun test ne rougissait, aucun log ne le disait (le refus est
// un `Debug`), et la seule facon de s'en apercevoir etait de constater qu'une table restait
// vide — ce que personne ne fait sur une table neuve.
//
// Ce ratchet rend le cablage VERIFIABLE : chaque construction de collecteur en production doit
// porter `AvecCapture` (ou `WithPositionCapture`) dans son enchainement.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// constructeurDuCollecteur : le seul point de creation d'un collecteur.
const constructeurDuCollecteur = "NewKillSourceCollector("

// cablagesDeCapture : les formes qui arment la capture.
var cablagesDeCapture = []string{"AvecCapture(", "WithPositionCapture("}

// fenetreDuCablage : le cablage doit apparaitre dans les N lignes qui suivent la construction.
//
// UNE FENETRE PLUTOT QU'UNE MEME LIGNE : les trois chemins ecrivent leur enchainement sur
// plusieurs lignes (arguments nommes, commentaires intercales). Douze lignes couvrent le plus
// long des trois avec de la marge, sans jamais atteindre la construction suivante.
const fenetreDuCablage = 12

// TestToutCollecteurCableLaCapture — le ratchet.
//
// SELF-CHECK POSITIF : au moins une construction doit etre inspectee, sinon le garde ne garde
// plus rien (le constructeur a-t-il ete renomme, ou le collecteur remplace ?).
func TestToutCollecteurCableLaCapture(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	inspectees := 0
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		// LES TESTS SONT HORS PERIMETRE : un test a le droit de construire un collecteur SANS
		// capture — c'est meme ce que fait `positions_test.go` pour prouver que la garde du
		// non-cable coupe avant toute ecriture.
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lignes := strings.Split(string(data), "\n")
		for i, ligne := range lignes {
			if !strings.Contains(ligne, constructeurDuCollecteur) {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(ligne), "//") ||
				strings.Contains(ligne, "func NewKillSourceCollector") {
				continue
			}
			inspectees++
			if cableDansLaFenetre(lignes, i) {
				continue
			}
			rel := filepath.ToSlash(mustRel(apiRoot, path))
			violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(ligne))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours apps/go-api: %v", err)
	}
	if inspectees == 0 {
		t.Fatalf("aucune construction de collecteur inspectee : %q a-t-il ete renomme ? "+
			"Le garde ne verifie plus rien", constructeurDuCollecteur)
	}
	if len(violations) > 0 {
		t.Errorf("collecteur construit SANS cabler la capture des positions (%d) — appeler "+
			"`AvecCapture` : sans elle, ni kill_positions ni les faits d isolement ne sont "+
			"jamais ecrits par ce chemin, et RIEN ne le dit (le refus est un Debug) :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// cableDansLaFenetre cherche une forme de cablage dans les lignes qui suivent la construction.
func cableDansLaFenetre(lignes []string, depuis int) bool {
	fin := depuis + fenetreDuCablage
	if fin >= len(lignes) {
		fin = len(lignes) - 1
	}
	for j := depuis; j <= fin; j++ {
		for _, forme := range cablagesDeCapture {
			if strings.Contains(lignes[j], forme) {
				return true
			}
		}
	}
	return false
}

// litteralDesDeps / champDuResolveur : le SECOND volet du garde.
//
// Le premier vérifie que chaque collecteur ARME la capture ; celui-ci vérifie que l'étape
// post-sync la RENSEIGNE. Les deux sont nécessaires et aucun ne suffit : `AvecCapture` sur des
// dépendances vides est un no-op silencieux — exactement la forme qu'aurait prise le défaut
// P0-1 si on l'avait « corrigé » sans fournir le résolveur.
const (
	litteralDesDeps   = "PostSyncDeps{"
	champDuResolveur  = "MapNames:"
	fenetreDuLitteral = 20
)

// TestPostSyncFournitLeResolveurDeCarte — le second volet.
//
// SELF-CHECK POSITIF : au moins un littéral doit être inspecté.
func TestPostSyncFournitLeResolveurDeCarte(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	inspectes := 0
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		// LES TESTS SONT HORS PERIMETRE : ils construisent des Deps partielles pour prouver
		// que les gardes coupent.
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		lignes := strings.Split(string(data), "\n")
		for i, ligne := range lignes {
			// Le littéral est composé (`killcollector.PostSyncDeps{`) ; la déclaration du
			// type, elle, s'écrit `type PostSyncDeps struct`.
			if !strings.Contains(ligne, litteralDesDeps) || strings.Contains(ligne, "type ") {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(ligne), "//") {
				continue
			}
			inspectes++
			if champPresent(lignes, i, champDuResolveur, fenetreDuLitteral) {
				continue
			}
			rel := filepath.ToSlash(mustRel(apiRoot, path))
			violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(ligne))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours apps/go-api: %v", err)
	}
	if inspectes == 0 {
		t.Fatalf("aucun littéral %q inspecté : le type a-t-il été renommé ? Le garde ne "+
			"vérifie plus rien", litteralDesDeps)
	}
	if len(violations) > 0 {
		t.Errorf("PostSyncDeps construite SANS %s (%d) — l'étape post-sync n'aura aucun "+
			"résolveur de carte, `AvecCapture` sera un no-op silencieux, et ni kill_positions "+
			"ni les faits d'isolement ne seront écrits au fil du sync :\n  %s",
			champDuResolveur, len(violations), strings.Join(violations, "\n  "))
	}
}

// champPresent cherche un champ dans les lignes qui suivent l'ouverture d'un littéral.
func champPresent(lignes []string, depuis int, champ string, fenetre int) bool {
	fin := depuis + fenetre
	if fin >= len(lignes) {
		fin = len(lignes) - 1
	}
	for j := depuis; j <= fin; j++ {
		if strings.Contains(lignes[j], champ) {
			return true
		}
	}
	return false
}
