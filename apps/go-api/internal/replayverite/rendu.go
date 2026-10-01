package replayverite

// rendu.go — LE RAPPORT TEXTE D'UNE COMPARAISON.
//
// L'ordre dit ce qui compte : le verdict, les constats BLOQUANTS avec leur detail nominatif (c'est
// ce qu'un operateur attribue), les gains, puis les valeurs ABSOLUES d'apres — a titre
// d'information, elles ne decident rien.

import (
	"fmt"
	"io"
)

// Rendre ecrit le rapport d'un temoin.
func Rendre(w io.Writer, temoin string, c Comparaison) error {
	e := &ecrivain{w: w}
	e.printf("BANC DE VERITE — %s : %s\n", temoin, c.Statut)
	for _, x := range c.Bloquants() {
		e.constat(x)
	}
	for _, x := range c.Constats {
		if x.Sens == sensGain || x.Sens == sensInfo {
			e.constat(x)
		}
	}
	e.printf("  valeurs d'apres (information) :\n")
	for _, id := range clesTriees(c.Apres.Scores) {
		e.printf("    %-40s %s\n", id, formatScore(c.Apres.Scores[id]))
	}
	for _, id := range clesTriees(c.Apres.Preuves) {
		e.printf("    %-40s %d\n", id, c.Apres.Preuves[id].Valeur)
	}
	for _, id := range clesTriees(c.Apres.Violations) {
		e.printf("    %-40s %s\n", id, formatViolation(c.Apres.Violations[id]))
	}
	return e.err
}

// ecrivain retient la premiere erreur d'ecriture (une seule verification en fin de rendu).
type ecrivain struct {
	w   io.Writer
	err error
}

func (e *ecrivain) printf(format string, args ...any) {
	if e.err == nil {
		_, e.err = fmt.Fprintf(e.w, format, args...)
	}
}

// detailMaxLignes : au-dela, le detail d'un constat est tronque AU RENDU (il reste entier dans la
// Comparaison). Vingt lignes suffisent a attribuer une hausse ; une baisse massive (un correctif qui
// retire des centaines de faux) n'a pas a noyer le rapport.
const detailMaxLignes = 20

func (e *ecrivain) constat(x Constat) {
	e.printf("  [%s] %s : %s -> %s\n", x.Sens, x.Mesure, x.Avant, x.Apres)
	for i, d := range x.Detail {
		if i == detailMaxLignes {
			e.printf("      ... %d ligne(s) de plus\n", len(x.Detail)-detailMaxLignes)
			break
		}
		e.printf("      %s\n", d)
	}
}
