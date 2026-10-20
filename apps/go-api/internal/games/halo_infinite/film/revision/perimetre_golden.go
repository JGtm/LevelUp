package revision

// perimetre_golden.go — LE GOLDEN DE PERIMETRE D UNE COUCHE : la liste des paquets hashes par
// leurs octets et des couches amont qui entrent par leur valeur (lot J3.2).
//
//	paquet<TAB><dossier relatif au module>   un paquet de l arbre de la couche, ou importe
//	amont<TAB><nom de couche>                 une couche revisee rencontree
//
// Les lignes vides et celles qui commencent par `#` sont de la prose, conservee a la reecriture.
// LA REECRITURE PASSE PAR LA MEME PORTE A DEUX VERROUS que les goldens de revision ([Porte]), et
// elle ne rend jamais `ok` : sa seule sortie normale est un `t.Fatal`.

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// Etiquettes des lignes de donnees d un golden de perimetre.
const (
	etiquettePaquet = "paquet"
	etiquetteAmont  = "amont"
)

// LirePerimetre relit un golden de perimetre.
func LirePerimetre(chemin string) (Perimetre, error) {
	lignes, err := lignesDe(chemin)
	if err != nil {
		return Perimetre{}, err
	}
	var p Perimetre
	for _, ligne := range lignes {
		if ligne == "" || strings.HasPrefix(ligne, "#") {
			continue
		}
		etiquette, valeur, ok := strings.Cut(ligne, "\t")
		switch {
		case ok && etiquette == etiquettePaquet && valeur != "":
			p.Paquets = append(p.Paquets, valeur)
		case ok && etiquette == etiquetteAmont && valeur != "":
			p.Amonts = append(p.Amonts, valeur)
		default:
			return Perimetre{}, fmt.Errorf("golden de perimetre %s : ligne malformee %q — attendu "+
				"`paquet<TAB>dossier` ou `amont<TAB>couche`", chemin, ligne)
		}
	}
	return p, nil
}

// Egal dit si deux perimetres portent les memes paquets et les memes amonts, dans le meme ordre.
func (p Perimetre) Egal(autre Perimetre) bool {
	return slices.Equal(p.Paquets, autre.Paquets) && slices.Equal(p.Amonts, autre.Amonts)
}

// ReecrirePerimetre fige `per` dans le golden `chemin`, prose conservee, et rend LE MESSAGE D ECHEC
// que l appelant doit passer a `t.Fatalf`.
func (p Porte) ReecrirePerimetre(chemin string, per Perimetre) (string, error) {
	var prose []string
	if lignes, err := lignesDe(chemin); err == nil {
		for _, ligne := range lignes {
			if ligne == "" || strings.HasPrefix(ligne, "#") {
				prose = append(prose, ligne)
			}
		}
	}
	for len(prose) > 0 && prose[len(prose)-1] == "" {
		prose = prose[:len(prose)-1]
	}
	if len(prose) > 0 {
		// Une ligne vide separe la prose des donnees, a chaque reecriture.
		prose = append(prose, "")
	}
	donnees := make([]string, 0, len(per.Paquets)+len(per.Amonts))
	for _, paquet := range per.Paquets {
		donnees = append(donnees, etiquettePaquet+"\t"+paquet)
	}
	for _, a := range per.Amonts {
		donnees = append(donnees, etiquetteAmont+"\t"+a)
	}
	sortie := strings.Join(append(append(prose, donnees...), ""), "\n")
	if err := os.WriteFile(chemin, []byte(sortie), 0o600); err != nil {
		return "", fmt.Errorf("ecriture du golden %s : %w", chemin, err)
	}
	return fmt.Sprintf("1 perimetre reecrit : %s (%d paquets, amonts %v) ; relancer sans -%s pour "+
		"verifier", chemin, len(per.Paquets), per.Amonts, p.Drapeau()), nil
}
