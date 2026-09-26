//go:build research

package reapparition

// rapport.go — LA PASSE, ET CE QU'ELLE ECRIT.
//
// Une passe = une image, six calibrations, douze cibles. Le rapport sort en texte brut sur le
// flux fourni : c'est lui qui est recopie dans la note du lot, pas une capture d'ecran.
//
// LA REGLE DE PUBLICATION EST DANS LE CODE : `Passe` rend une erreur si une seule calibration
// echoue. Aucune adresse neuve n'est alors publiee — la chaine n'a pas prouve qu'elle lit cette
// image-la.

import (
	"fmt"
	"io"
	"sort"
)

// Passe execute la passe complete et ecrit le rapport.
func Passe(chemin string, w io.Writer) error {
	ex, err := Charger(chemin)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "IMAGE : %s\n", chemin)
	for _, s := range ex.Sections() {
		fmt.Fprintf(w, "  %s\n", s)
	}
	fmt.Fprintf(w, "\n")

	if err := Calibrer(ex, w); err != nil {
		return err
	}
	ecrireSignature(ex, w)
	ecrireCibles(ex, w)
	return nil
}

// Calibrer rejoue la chaine sur les temoins et rend une erreur des qu'un seul rate : c'est LA
// REGLE DE PUBLICATION, et elle est exportee parce qu'elle vaut pour toute passe, pas seulement
// pour celle du lot 3.7 (le lot 5.3 interroge d'autres composants de la meme image).
func Calibrer(ex *Executable, w io.Writer) error {
	fmt.Fprintf(w, "== CALIBRATION (%d temoins) ==\n", len(Calibrations))
	rates := 0
	for _, t := range Calibrations {
		d := ex.Resoudre(t.Nom)
		switch {
		case !d.Aboutie():
			fmt.Fprintf(w, "  RATE   %-58s %s\n", t.Nom, d.Echec)
			rates++
		case d.EcrivainVA != t.Attendue:
			fmt.Fprintf(w, "  RATE   %-58s lu %#x, attendu %#x (%s)\n",
				t.Nom, d.EcrivainVA, t.Attendue, t.Source)
			rates++
		default:
			fmt.Fprintf(w, "  OK     %-58s ecrivain %#x  descripteur %#x\n",
				t.Nom, d.EcrivainVA, d.DescripteurVA)
		}
	}
	fmt.Fprintf(w, "\n")
	if rates > 0 {
		return fmt.Errorf("%d calibration(s) en echec : aucune adresse neuve n'est publiee", rates)
	}
	return nil
}

// ecrireSignature mesure les slots CONSTANTS de la famille sur les temoins : un slot dont la
// valeur est la meme chez les six est une signature, un slot qui varie est une identite.
func ecrireSignature(ex *Executable, w io.Writer) {
	var descs []*Descripteur
	for _, t := range Calibrations {
		if d := ex.Resoudre(t.Nom); d.Aboutie() {
			descs = append(descs, d)
		}
	}
	fmt.Fprintf(w, "== SIGNATURE DE FAMILLE, MESUREE SUR LES TEMOINS ==\n")
	for i := 0; i < tailleDescripteur/8; i++ {
		vals := map[uint64]int{}
		for _, d := range descs {
			vals[d.Slots[i]]++
		}
		if len(vals) == 1 {
			for v := range vals {
				fmt.Fprintf(w, "  +%#04x  CONSTANT  %#x\n", i*8, v)
			}
			continue
		}
		fmt.Fprintf(w, "  +%#04x  variable  (%d valeurs distinctes sur %d temoins)\n",
			i*8, len(vals), len(descs))
	}
	fmt.Fprintf(w, "\n")
}

func ecrireCibles(ex *Executable, w io.Writer) {
	fmt.Fprintf(w, "== CIBLES DU LOT 3.7 (%d) ==\n\n", len(Cibles))
	for _, c := range Cibles {
		fmt.Fprintf(w, "--- %s  %s\n    POURQUOI : %s\n", c.Archetype, c.Nom, c.Pourquoi)
		d := ex.Resoudre(c.Nom)
		if !d.Aboutie() {
			fmt.Fprintf(w, "    ECHEC : %s\n", d.Echec)
			for _, m := range d.Multiples {
				fmt.Fprintf(w, "      %s\n", m)
			}
			fmt.Fprintf(w, "\n")
			continue
		}
		fmt.Fprintf(w, "    chaine %#x  thunk %#x  descripteur %#x  ECRIVAIN %#x\n",
			d.ChaineVA, d.ThunkVA, d.DescripteurVA, d.EcrivainVA)
		fmt.Fprintf(w, "    compagnon +0x28 %#x\n", d.Slots[0x28/8])
		if !d.BornesConnues {
			fmt.Fprintf(w, "    bornes : ABSENTES de .pdata — grammaire non relevee\n\n")
			continue
		}
		r := ex.Relever(d.Bornes)
		fmt.Fprintf(w, "    bornes %#x..%#x · %s\n", d.Bornes.Debut, d.Bornes.Fin, r.Resume())
		for _, l := range r.Largeurs {
			fmt.Fprintf(w, "      largeur %#x : R(%d)\n", l.VA, l.Bits)
		}
		for _, cible := range r.CiblesUniques {
			n := 0
			for _, a := range r.Appels {
				if a.Cible == cible {
					n++
				}
			}
			fmt.Fprintf(w, "      appel  -> %#x  (x%d)\n", cible, n)
		}
		fmt.Fprintf(w, "\n")
	}
}

// Inventaire ecrit tous les noms de composant de l'image dont le nom porte un mot du vocabulaire
// de la reapparition. C'EST LE NEGATIF MESURE : si aucun composant de vehicule n'y figure, ce
// n'est pas une impression, c'est un balayage de l'univers des noms.
func Inventaire(chemin string, mots []string, w io.Writer) error {
	ex, err := Charger(chemin)
	if err != nil {
		return err
	}
	noms := ex.NomsDeComposants()
	fmt.Fprintf(w, "== UNIVERS DES NOMS DE COMPOSANT DE L'IMAGE : %d ==\n", len(noms))
	for _, m := range mots {
		var frappes []string
		for _, n := range noms {
			if indexOf([]byte(n), []byte(m)) >= 0 {
				frappes = append(frappes, n)
			}
		}
		sort.Strings(frappes)
		fmt.Fprintf(w, "\n-- « %s » : %d\n", m, len(frappes))
		for _, n := range frappes {
			fmt.Fprintf(w, "   %s\n", n)
		}
	}
	return nil
}
