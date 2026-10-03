package revision

// porte.go — LA PORTE DE REGENERATION D UN GOLDEN DE REVISION.
//
// # DEUX VERROUS, ET LA RAISON DE CHACUN
//
//	le drapeau   `-update-<nom>` NOMME. Accroche au `-update` generique d un paquet, la porte
//	             laissait `go test ./...<paquet>/ -update` (sans `-run`) refiger l empreinte
//	             d une couche CASSEE en repondant `ok` (revue R1, constat P1-1).
//	la variable  `LEVELUP_UPDATE_<NOM>`. Un drapeau seul se tape par reflexe quand un gate
//	             rougit ; la variable oblige a sortir de la boucle « rouge -> -update -> vert ».
//	             Modele : `replay/document_shape_test.go`, TestDocumentShapeRegenerate, qui
//	             exige le drapeau ET `LEVELUP_CONTRACT_FIXTURES`.
//
// # UNE PORTE DE REGENERATION NE REND JAMAIS `ok`
//
// `go test` JETTE la sortie d un paquet qui PASSE : une reecriture annoncee par `t.Logf` est
// INVISIBLE avec la commande documentee (sans `-v`), si bien qu une reecriture du couple
// (revision, empreinte) se lit `ok` — exactement ce que les trois portes du chantier
// (`-update-keyframe-closure`, `-update-grammar-rev`, `-update`) refusent depuis la revue R2.
// C est pour cela que [Porte.Reecrire] rend un MESSAGE D ECHEC en cas de reussite : sa seule
// sortie normale est un `t.Fatalf`. Le type ne peut pas appeler `t.Fatalf` lui-meme — un paquet
// de production n importe pas `testing` — mais il rend l oubli visible a la lecture.
//
// # CE QUE LA PORTE N EST PAS
//
// Elle n arbitre PAS « la revision doit-elle monter ? ». C est la couche qui le decide, et le
// message de son gate ([Messages]) pose la question. La porte ne fait qu ecrire, une fois que
// les deux verrous sont ouverts.

import (
	"fmt"
	"os"
	"strings"
)

// Porte : la porte de regeneration du golden d une couche, identifiee par son nom court
// (`grammar-rev`, `facts-rev`).
type Porte struct {
	// Nom : le nom court, en minuscules, mots separes par des tirets.
	Nom string
}

// Drapeau rend le nom du drapeau `flag` que le test de la couche doit declarer.
func (p Porte) Drapeau() string { return "update-" + p.Nom }

// Variable rend le nom de la variable d environnement qui double le drapeau.
func (p Porte) Variable() string {
	return "LEVELUP_UPDATE_" + strings.ToUpper(strings.ReplaceAll(p.Nom, "-", "_"))
}

// Ouverte dit si la regeneration est autorisee, et sinon POURQUOI — la raison est faite pour
// etre passee telle quelle a `t.Skip`.
//
// `valeurEnv` est la valeur lue par l appelant (`os.Getenv(p.Variable())`) : la porte ne lit pas
// l environnement elle-meme, pour que son comportement soit testable sans variable globale.
func (p Porte) Ouverte(drapeau bool, valeurEnv string) (bool, string) {
	switch {
	case !drapeau:
		return false, fmt.Sprintf("regeneration du golden : passer -%s (et %s=1)",
			p.Drapeau(), p.Variable())
	case valeurEnv == "":
		return false, fmt.Sprintf("regeneration du golden : %s non definie (le drapeau -%s seul "+
			"ne suffit pas)", p.Variable(), p.Drapeau())
	default:
		return true, ""
	}
}

// Reecrire fige le couple (revision, empreinte) dans le golden et rend LE MESSAGE D ECHEC que
// l appelant doit passer a `t.Fatalf`.
//
// La prose du golden (lignes vides et lignes ouvertes par `#`) est conservee TELLE QUELLE, en
// tete. Les lignes de donnees suivent, dans l ordre : la derniere est reecrite si elle porte
// deja `revision` (regeneration d une empreinte a revision constante), sinon une ligne est
// AJOUTEE — c est ainsi que le golden accumule sa chronique.
func (p Porte) Reecrire(chemin, revision, empreinte string) (string, error) {
	lignes, err := lignesDe(chemin)
	if err != nil {
		return "", err
	}
	var prose []string
	var donnees []string
	for _, ligne := range lignes {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			prose = append(prose, ligne)
			continue
		}
		donnees = append(donnees, ligne)
	}
	for len(prose) > 0 && prose[len(prose)-1] == "" {
		prose = prose[:len(prose)-1]
	}
	nouvelle := revision + "\t" + empreinte
	if n := len(donnees); n > 0 && strings.HasPrefix(donnees[n-1], revision+"\t") {
		donnees[n-1] = nouvelle
	} else {
		donnees = append(donnees, nouvelle)
	}
	sortie := strings.Join(append(append(prose, donnees...), ""), "\n")
	if err := os.WriteFile(chemin, []byte(sortie), 0o600); err != nil {
		return "", fmt.Errorf("ecriture du golden %s : %w", chemin, err)
	}
	return fmt.Sprintf("1 reference(s) reecrite(s) : %s (revision %s, empreinte %s) ; "+
		"relancer sans -%s pour verifier", chemin, revision, empreinte, p.Drapeau()), nil
}
