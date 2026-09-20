//go:build research

package mouvement

// rapport.go — LA PASSE DU LOT 5.3, ET CE QU'ELLE ECRIT.
//
// LA REGLE DE PUBLICATION EST CELLE DU LOT 3.7, ET ELLE EST REUTILISEE TELLE QUELLE :
// [reapparition.Calibrer] rejoue la chaine du descripteur sur six composants dont le depot
// connait deja l'adresse de l'ecrivain. Une seule concordance qui rate et la passe s'arrete
// sans publier une seule adresse neuve — la chaine n'a pas prouve qu'elle lit cette image-la.

import (
	"fmt"
	"io"

	"levelup/go-api/internal/games/halo_infinite/film/research/reapparition"
)

// Passe resout les cibles de mouvement et ecrit le rapport sur le flux fourni.
func Passe(chemin string, w io.Writer) error {
	ex, err := reapparition.Charger(chemin)
	if err != nil {
		return err
	}
	fmt.Fprintf(w, "IMAGE : %s\n\n", chemin)
	if err := reapparition.Calibrer(ex, w); err != nil {
		return err
	}
	fmt.Fprintf(w, "== CIBLES DU LOT 5.3 (%d) ==\n\n", len(Cibles))
	for _, c := range Cibles {
		ecrireCible(ex, c, w)
	}
	return nil
}

func ecrireCible(ex *reapparition.Executable, c Cible, w io.Writer) {
	fmt.Fprintf(w, "--- %s  %s\n    POURQUOI : %s\n    LU AUJOURD'HUI : %s\n",
		c.Archetype, c.Nom, c.Pourquoi, c.LuAujourdhui)
	d := ex.Resoudre(c.Nom)
	if !d.Aboutie() {
		fmt.Fprintf(w, "    ECHEC : %s\n", d.Echec)
		for _, m := range d.Multiples {
			fmt.Fprintf(w, "      %s\n", m)
		}
		fmt.Fprintf(w, "\n")
		return
	}
	fmt.Fprintf(w, "    chaine %#x  thunk %#x  descripteur %#x  ECRIVAIN %#x  compagnon +0x28 %#x\n",
		d.ChaineVA, d.ThunkVA, d.DescripteurVA, d.EcrivainVA, d.Slots[0x28/8])
	if !d.BornesConnues {
		fmt.Fprintf(w, "    bornes : ABSENTES de .pdata — grammaire non relevee\n\n")
		return
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

// Vocabulaire ecrit, mot par mot, les chaines de l'image qui le contiennent, ET les noms de
// composant qui le contiennent. C'EST LE NEGATIF MESURE : un mot qui ne rend rien ici n'est
// dans aucune chaine de l'executable.
//
// `plafond` borne ce qui est RECOPIE, jamais ce qui est COMPTE : le total reste celui du
// balayage complet, et le rapport dit combien de lignes il a tues. Un negatif se mesure sur
// des totaux, pas sur un echantillon.
func Vocabulaire(chemin string, plafond int, w io.Writer) error {
	ex, err := reapparition.Charger(chemin)
	if err != nil {
		return err
	}
	noms := ex.NomsDeComposants()
	fmt.Fprintf(w, "IMAGE : %s\nUNIVERS DES NOMS DE COMPOSANT : %d\n\n", chemin, len(noms))
	trouvees := ex.ChainesContenant(MotsDuMouvement)
	for _, m := range MotsDuMouvement {
		ecrireMot(m, noms, trouvees[m], plafond, w)
	}
	return nil
}

func ecrireMot(mot string, noms []string, chaines []reapparition.Chaine, plafond int, w io.Writer) {
	var composants []string
	for _, n := range noms {
		if contient(n, mot) {
			composants = append(composants, n)
		}
	}
	fmt.Fprintf(w, "-- « %s » : %d chaine(s), dont %d nom(s) de composant\n",
		mot, len(chaines), len(composants))
	for _, n := range composants {
		fmt.Fprintf(w, "   COMPOSANT  %s\n", n)
	}
	for i, c := range chaines {
		if i >= plafond {
			fmt.Fprintf(w, "   ... %d chaine(s) non recopiee(s)\n", len(chaines)-plafond)
			break
		}
		fmt.Fprintf(w, "   %#x  %s\n", c.VA, c.Texte)
	}
	fmt.Fprintf(w, "\n")
}

// contient est une comparaison en minuscules ; les noms de composant le sont deja, mais la
// fonction ne le suppose pas.
func contient(s, mot string) bool {
	for i := 0; i+len(mot) <= len(s); i++ {
		if bas(s[i:i+len(mot)]) == mot {
			return true
		}
	}
	return false
}

func bas(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}
