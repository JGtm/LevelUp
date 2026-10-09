package replay

// contract_fixtures_budget_test.go — CE QUE LE DOSSIER DE FIXTURES PESE, ET CE QU'IL CONTIENT.
//
// Sorti de `contract_fixtures_test.go` le 2026-09-13 (lot 0.B.7) quand celui-ci a depasse les
// 500 lignes (CLAUDE.md, seuil n° 5). La coupure suit une responsabilite, pas un compte de
// lignes : ici vivent le PLAFOND du jeu et l'INVENTAIRE du dossier — ce qui pese et ce qui
// traine —, la ou l'autre fichier tient la CUISSON et la comparaison. Les deux regles que ce
// fichier fait respecter viennent de l'arbitrage du pilote du 2026-09-13 :
//
//	1. le jeu entier tient sous 3 Mio (la coupe a un point sur cinq l'y ramene : 2,01 Mio) ;
//	2. UN SEUL JEU VIVANT — le dossier ne porte que la version courante, la regeneration
//	   supprime la precedente.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// contractFixturesBudget : LE PLAFOND DE TAILLE DU JEU ENTIER, en octets compresses.
//
// POURQUOI UN PLAFOND ET PAS SEULEMENT UNE MESURE. Ces fichiers sont VERSIONNES : chaque
// montee de schema en depose un jeu complet de plus dans l'historique du depot. Une taille
// qu'on se contente de mesurer derive d'un build a l'autre sans que personne ne le decide ; un
// plafond force la decision (publier moins de calques, moins de builds) le jour ou il est
// atteint. 3 Mio est la borne du plan (lot 0.B.7), TENUE : les documents pleins pesaient
// 5,86 Mio, la coupe des points de piste (arbitrage du pilote) les ramene a 2,01 Mio. Le detail
// par fixture se lit dans TestContractFixturesTiennentDansLeBudget.
const contractFixturesBudget = 3 << 20

// TestContractFixturesTiennentDansLeBudget : le jeu entier tient-il sous le plafond, et
// combien pese chaque fixture ?
//
// LA MESURE EST LA POUR ETRE LUE (`go test -run ContractFixturesTiennent -v`) : c'est elle que
// le plan consigne. Le plafond, lui, est la pour que la question se pose le jour ou un build de
// plus le ferait sauter, au lieu d'etre decouverte dans un `git clone` de plus en plus long.
func TestContractFixturesTiennentDansLeBudget(t *testing.T) {
	dir := contractFixturesDir(t)
	fixtures := contractFixtures()
	total := 0
	for _, f := range fixtures {
		n := contractFixtureTaille(t, filepath.Join(dir, f.file()))
		total += n
		t.Logf("%-10s %-10s %9d octets", f.build, f.film, n)
	}
	t.Logf("TOTAL %d fixtures : %d octets (%.2f Mio), plafond %d octets (%.0f Mio)",
		len(fixtures), total, float64(total)/(1<<20),
		contractFixturesBudget, float64(contractFixturesBudget)/(1<<20))
	if total > contractFixturesBudget {
		t.Errorf("le jeu de fixtures pese %d octets (%.2f Mio), au-dela du plafond de %d octets "+
			"(%.0f Mio) : couper (moins de calques publies, moins de builds) ou relever le "+
			"plafond par decision ecrite, jamais par reflexe",
			total, float64(total)/(1<<20), contractFixturesBudget,
			float64(contractFixturesBudget)/(1<<20))
	}
}

// listeCourte : « : a, b » ou rien du tout. Un message qui annonce « 0 supprimee() : » pour
// dire qu'il n'a rien fait se lit comme un defaut.
func listeCourte(noms []string) string {
	if len(noms) == 0 {
		return ""
	}
	return " : " + strings.Join(noms, ", ")
}

// fixturesDuDossier : les documents de contrat PRESENTS, quelle que soit leur version.
func fixturesDuDossier(t *testing.T, dir string) []string {
	t.Helper()
	trouves, err := filepath.Glob(filepath.Join(dir, "replay_schema_*.json.gz"))
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	noms := make([]string, 0, len(trouves))
	for _, p := range trouves {
		noms = append(noms, filepath.Base(p))
	}
	return noms
}

// purgerJeuxPerimes efface les documents qui ne sont pas du jeu qu'on vient d'ecrire.
//
// UN SEUL JEU VIVANT (cf. en-tete) : le nom porte la version de schema, donc sans cette purge
// une montee de version DEPOSERAIT un jeu complet de plus a cote de l'ancien, et l'arbre
// grossirait de 2 Mio a chaque fois. L'historique git, lui, garde ce qui a existe : rien n'est
// perdu, seule la copie de travail ne garde que le present.
func purgerJeuxPerimes(t *testing.T, dir string, vivants []string) []string {
	t.Helper()
	garde := make(map[string]bool, len(vivants))
	for _, n := range vivants {
		garde[n] = true
	}
	var supprimes []string
	for _, nom := range fixturesDuDossier(t, dir) {
		if garde[nom] {
			continue
		}
		if err := os.Remove(filepath.Join(dir, nom)); err != nil {
			t.Fatalf("suppression de la fixture perimee %s : %v", nom, err)
		}
		supprimes = append(supprimes, nom)
	}
	return supprimes
}

// TestContractFixturesUnSeulJeuVivant : le dossier ne porte-t-il QUE le jeu de la version
// courante ?
//
// CE QUE CE TEST TIENT, ET QUE LA PURGE NE TIENT PAS. La purge ne s'execute qu'a la
// regeneration ; ce test s'execute a CHAQUE `go test`. Sans lui, une fixture d'une version
// anterieure recopiee a la main (ou ressuscitee par une fusion) vivrait dans l'arbre sans que
// rien ne le dise : le manifeste ne la liste pas, donc le web ne la lit pas — elle ne serait
// qu'un poids mort que personne ne voit grossir.
func TestContractFixturesUnSeulJeuVivant(t *testing.T) {
	dir := contractFixturesDir(t)
	publies := make(map[string]bool)
	for _, f := range contractFixtures() {
		publies[f.file()] = true
	}
	var intrus []string
	for _, nom := range fixturesDuDossier(t, dir) {
		if !publies[nom] {
			intrus = append(intrus, nom)
		}
	}
	if len(intrus) > 0 {
		t.Errorf("le dossier porte %d fixture(s) hors du jeu courant (schema %d)%s : le web ne "+
			"les lit pas (le manifeste ne les liste pas) et elles pesent pour rien — regenerer "+
			"avec %s=1 go test -run ContractFixtures -update, ce qui les supprime",
			len(intrus), SchemaVersion, listeCourte(intrus), contractFixturesEnv)
	}
}
