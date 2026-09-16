package revision

// chronique.go — LA CHRONIQUE D UNE REVISION : ce que le godoc RACONTE, ce que le golden FIGE,
// et le rang qui relie les deux.
//
// # LE DEFAUT QUE CE LECTEUR FERME
//
// Un ratchet d empreinte ne tient que le couple (revision, empreinte) — jamais ce que la
// revision RACONTE. Constat F5 de la revue de jalon M1 : trois lots partis de la meme base `.11`
// ont empile trois blocs annoncant chacun « `.11` -> `.12` », la chronique s est arretee a `.12`
// pendant que la constante valait `.14`, et les changements de comportement portes par `.13` et
// `.14` n avaient AUCUNE entree. Rien ne rougissait.
//
// # CE QU IL LIT
//
//	godoc    `// ENTREE ` + accent grave + revision + accent grave + ` (AAAA-MM-JJ, lot) : ...`
//	golden   des lignes `revision<TAB>empreinte` ; les lignes vides et celles ouvertes par `#`
//	         sont de la prose. LA DERNIERE LIGNE DE DONNEES EST LA COURANTE.
//
// Les goldens d aujourd hui ne portent qu UNE ligne de donnees et gardent leur historique en
// commentaires ; ce lecteur les lit sans changement (une seule ligne, c est deja « la derniere
// est la courante ») et accepte l historique en LIGNES DE DONNEES que le lot 2.6.1 posera.

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Rang : la position d une revision dans sa chronique.
//
// La forme est `<prefixe>-AAAA-MM-JJ[.N]`, avec `N >= 2` : deux LOTS du meme jour se separent
// par leur rang (correctif D1 (1.2), 2026-09-14), et le premier lot du jour s ecrit SANS suffixe.
// `.1` est donc refuse — une meme position ne s ecrit pas de deux facons.
type Rang struct {
	// Date : la date ISO portee par la revision, `AAAA-MM-JJ`.
	Date string
	// N : le rang dans la journee, 1 pour une revision sans suffixe.
	N int
}

// formeRevision : la forme d une revision, prefixe exclu.
var formeRevision = regexp.MustCompile(`^([0-9]{4}-[0-9]{2}-[0-9]{2})(?:\.([0-9]+))?$`)

// ParserRevision rend le rang d une revision, ou dit pourquoi elle n a pas la forme attendue.
func ParserRevision(prefixe, revision string) (Rang, error) {
	reste, ok := strings.CutPrefix(revision, prefixe+"-")
	if !ok {
		return Rang{}, fmt.Errorf("revision %q : prefixe attendu %q (forme %s-AAAA-MM-JJ[.N])",
			revision, prefixe, prefixe)
	}
	m := formeRevision.FindStringSubmatch(reste)
	if m == nil {
		return Rang{}, fmt.Errorf("revision %q : forme attendue %s-AAAA-MM-JJ[.N]", revision, prefixe)
	}
	if m[2] == "" {
		return Rang{Date: m[1], N: 1}, nil
	}
	n, err := strconv.Atoi(m[2])
	if err != nil || n < 2 {
		return Rang{}, fmt.Errorf("revision %q : rang de journee %q invalide — le premier lot du "+
			"jour s ecrit SANS suffixe, les suivants a partir de `.2`", revision, m[2])
	}
	return Rang{Date: m[1], N: n}, nil
}

// Suit dit si `suivant` est le rang IMMEDIATEMENT apres `precedent` : le lot d apres dans la
// journee, ou le premier lot d un jour ulterieur.
func (r Rang) Suit(precedent Rang) bool {
	if r.Date == precedent.Date {
		return r.N == precedent.N+1
	}
	return r.Date > precedent.Date && r.N == 1
}

// EntreeGodoc : une entree de chronique lue dans le godoc du fichier qui porte la constante.
type EntreeGodoc struct {
	// Revision : la revision annoncee par l entree.
	Revision string
	// Date : la date entre parentheses — celle du LOT, qui peut differer de celle de la revision
	// (un rang de fusion reprend la date de la serie).
	Date string
	// Lot : ce que l entree dit du lot, tel qu ecrit, coupe a la fin de la ligne.
	Lot string
	// Ligne : le numero de ligne de l entree dans le fichier, pour les messages.
	Ligne int
}

// LigneGolden : une ligne de donnees du golden.
type LigneGolden struct {
	// Revision et Empreinte : le couple fige.
	Revision, Empreinte string
	// Ligne : le numero de ligne dans le golden.
	Ligne int
}

// Chronique : les deux cotes de la chronique d une couche, deja lus.
type Chronique struct {
	// Prefixe : `grammar`, `facts`, ...
	Prefixe string
	// Godoc : les entrees du godoc, dans l ordre du fichier.
	Godoc []EntreeGodoc
	// Golden : les lignes de donnees du golden, dans l ordre du fichier.
	Golden []LigneGolden
}

// formeEntreeGodoc construit la forme d une entree pour un prefixe donne. La ligne peut
// deborder : on ne capture que ce qui est sur la premiere ligne, le lot etant informatif.
func formeEntreeGodoc(prefixe string) *regexp.Regexp {
	return regexp.MustCompile("(?m)^// ENTREE `" + regexp.QuoteMeta(prefixe) +
		"-([0-9]{4}-[0-9]{2}-[0-9]{2}(?:\\.[0-9]+)?)` \\(([0-9]{4}-[0-9]{2}-[0-9]{2}), ([^\n]*)$")
}

// LireChronique lit le godoc et le golden d une couche.
func LireChronique(prefixe, cheminGodoc, cheminGolden string) (Chronique, error) {
	c := Chronique{Prefixe: prefixe}
	godoc, err := lignesDe(cheminGodoc)
	if err != nil {
		return Chronique{}, err
	}
	forme := formeEntreeGodoc(prefixe)
	for i, ligne := range godoc {
		m := forme.FindStringSubmatch(ligne)
		if m == nil {
			continue
		}
		lot := strings.TrimSpace(m[3])
		if fin := strings.Index(lot, ")"); fin >= 0 {
			lot = lot[:fin]
		}
		c.Godoc = append(c.Godoc, EntreeGodoc{
			Revision: prefixe + "-" + m[1], Date: m[2], Lot: lot, Ligne: i + 1,
		})
	}
	if c.Golden, err = lireGolden(cheminGolden); err != nil {
		return Chronique{}, err
	}
	return c, nil
}

// lignesDe lit un fichier versionne et rend ses lignes, fins de ligne normalisees.
func lignesDe(chemin string) ([]string, error) {
	blob, err := os.ReadFile(chemin) //nolint:gosec // chemin fourni par l appelant, resolu chez lui
	if err != nil {
		return nil, fmt.Errorf("%s illisible : %w — il est VERSIONNE, son absence est une erreur",
			chemin, err)
	}
	return strings.Split(strings.ReplaceAll(string(blob), "\r\n", "\n"), "\n"), nil
}

// lireGolden rend les lignes de donnees d un golden, dans l ordre du fichier.
func lireGolden(chemin string) ([]LigneGolden, error) {
	lignes, err := lignesDe(chemin)
	if err != nil {
		return nil, err
	}
	var donnees []LigneGolden
	for i, ligne := range lignes {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		champs := strings.Split(ligne, "\t")
		if len(champs) != 2 || champs[0] == "" || champs[1] == "" {
			return nil, fmt.Errorf("golden %s ligne %d : %q malformee — attendu "+
				"`revision<TAB>empreinte`", chemin, i+1, ligne)
		}
		donnees = append(donnees, LigneGolden{Revision: champs[0], Empreinte: champs[1], Ligne: i + 1})
	}
	if len(donnees) == 0 {
		return nil, fmt.Errorf("golden %s : aucune ligne de donnees", chemin)
	}
	return donnees, nil
}

// Courante rend la DERNIERE ligne de donnees du golden — celle qui vaut pour le code
// d aujourd hui.
func (c Chronique) Courante() LigneGolden {
	return c.Golden[len(c.Golden)-1]
}

// VerifierCouverture : la revision courante a-t-elle son entree des DEUX cotes, et est-elle bien
// la DERNIERE du golden ?
//
// Une montee sans entree ne dit pas ce qu elle change ; une montee qui n est pas en queue de
// golden veut dire que le golden n a pas ete regenere apres elle.
func (c Chronique) VerifierCouverture(revision string) error {
	var manques []string
	if !c.dansGodoc(revision) {
		manques = append(manques, fmt.Sprintf("aucune entree de godoc (entrees lues : %v) ; "+
			"forme attendue :\n  // ENTREE `%s` (AAAA-MM-JJ, lot) : ...",
			c.revisionsGodoc(), revision))
	}
	if derniere := c.Courante(); derniere.Revision != revision {
		manques = append(manques, fmt.Sprintf("la derniere ligne du golden fige %q (ligne %d), "+
			"pas la revision courante — le golden n a pas ete regenere",
			derniere.Revision, derniere.Ligne))
	}
	if len(manques) == 0 {
		return nil
	}
	return fmt.Errorf("revision %s : %s", revision, strings.Join(manques, " ; "))
}

// dansGodoc dit si une revision a son entree de godoc.
func (c Chronique) dansGodoc(revision string) bool {
	for _, e := range c.Godoc {
		if e.Revision == revision {
			return true
		}
	}
	return false
}

// revisionsGodoc rend les revisions annoncees par le godoc, dans l ordre.
func (c Chronique) revisionsGodoc() []string {
	vues := make([]string, 0, len(c.Godoc))
	for _, e := range c.Godoc {
		vues = append(vues, e.Revision)
	}
	return vues
}

// VerifierRangs : les revisions se suivent-elles, sans doublon et sans trou, a partir de
// `depuis` ?
//
// # POURQUOI UN PLANCHER, ET PAS « TOUTE LA CHRONIQUE »
//
// L historique reel porte des trous, et ils ont une cause : mesure du 2026-09-17 sur
// `grammar_rev.golden`, la serie du 2026-09-15 passe de `.6` a `.8` puis de `.8` a `.12` — des
// rangs reserves par des lots paralleles dont la fusion n a pas eu lieu. Exiger la continuite
// sur le passe demanderait de renumeroter des revisions deja ecrites dans des goldens, c
// est-a-dire d ouvrir des backlogs pour de la comptabilite. Le plancher est donc EXPLICITE :
// la couche declare a partir d ou elle tient la regle. `depuis` vide verifie tout.
func (c Chronique) VerifierRangs(depuis string) error {
	fautes := c.fautesDeRang("le godoc", c.revisionsGodoc(), c.lignesGodoc(), depuis)
	fautes = append(fautes, c.fautesDeRang("le golden", c.revisionsGolden(), c.lignesGolden(), depuis)...)
	if len(fautes) == 0 {
		return nil
	}
	return fmt.Errorf("chronique %s : %s", c.Prefixe, strings.Join(fautes, " ; "))
}

// fautesDeRang rend les ruptures de la suite des revisions d un cote de la chronique.
func (c Chronique) fautesDeRang(quoi string, revisions []string, lignes []int, depuis string) []string {
	var fautes []string
	var precedent Rang
	ouvert := depuis == ""
	for i, rev := range revisions {
		if rev == depuis {
			ouvert = true
		}
		if !ouvert {
			continue
		}
		rang, err := ParserRevision(c.Prefixe, rev)
		if err != nil {
			fautes = append(fautes, fmt.Sprintf("%s ligne %d : %v", quoi, lignes[i], err))
			continue
		}
		if precedent == (Rang{}) {
			precedent = rang
			continue
		}
		if !rang.Suit(precedent) {
			fautes = append(fautes, fmt.Sprintf("%s ligne %d : %s ne suit pas %s-%s%s — les "+
				"rangs sont strictement croissants et sans trou (un rang saute est un lot dont "+
				"personne ne saura dire ce qu il a change)",
				quoi, lignes[i], rev, c.Prefixe, precedent.Date, suffixeRang(precedent.N)))
		}
		precedent = rang
	}
	return fautes
}

// suffixeRang rend “ pour le rang 1, `.N` sinon.
func suffixeRang(n int) string {
	if n <= 1 {
		return ""
	}
	return "." + strconv.Itoa(n)
}

// revisionsGolden / lignesGolden / lignesGodoc : les projections dont [VerifierRangs] a besoin.
func (c Chronique) revisionsGolden() []string {
	out := make([]string, 0, len(c.Golden))
	for _, l := range c.Golden {
		out = append(out, l.Revision)
	}
	return out
}

func (c Chronique) lignesGolden() []int {
	out := make([]int, 0, len(c.Golden))
	for _, l := range c.Golden {
		out = append(out, l.Ligne)
	}
	return out
}

func (c Chronique) lignesGodoc() []int {
	out := make([]int, 0, len(c.Godoc))
	for _, e := range c.Godoc {
		out = append(out, e.Ligne)
	}
	return out
}
