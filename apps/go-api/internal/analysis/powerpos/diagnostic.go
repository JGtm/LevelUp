package powerpos

// diagnostic.go — CE QUE LA SELECTION A FAIT, chiffre par chiffre, sans rien decider.
//
// Le verdict v1 (VERDICT_ORACLE_2026-09-20.md, section 6) n'a pu etre diagnostique qu'en
// REJOUANT le score sur les CSV, parce que la passe de mesure ne disait ni combien de
// cellules passaient le seuil, ni la taille des amas qu'elles formaient. Ce fichier rend
// ces comptes a la passe elle-meme, pour que le reglage v2 se choisisse sur eux (nombre de
// cellules d'amorce, taille des composantes avant et apres fermeture) et non a l'oeil.

import "levelup/go-api/internal/analysis/tactical"

// DiagnosticSelection porte les comptes intermediaires d'une selection.
type DiagnosticSelection struct {
	// SeuilAmorce / SeuilCroissance : les seuils appliques a la carte. En regle v1 les deux
	// valent le seuil unique.
	SeuilAmorce     float64
	SeuilCroissance float64
	// CellulesAmorce : cellules au-dessus du seuil d'amorce.
	CellulesAmorce int
	// CellulesRetenues : cellules retenues AVANT fermeture (regle v1 : = CellulesAmorce).
	CellulesRetenues int
	// CellulesApresFermeture : cellules apres la fermeture morphologique.
	CellulesApresFermeture int
	// Composantes : nombre de composantes (dans la connexite du reglage) apres fermeture ;
	// ComposantesRetenues : celles qui atteignent la taille minimale.
	Composantes         int
	ComposantesRetenues int
	// TaillesComposantes : les tailles des composantes apres fermeture, decroissantes.
	TaillesComposantes []int
	// TaillesAvantFermeture : les tailles des composantes 4-connexes AVANT fermeture,
	// decroissantes — c'est la mesure qui a manque a la v1.
	TaillesAvantFermeture []int
}

// Diagnostique rejoue la selection et rend ses comptes. Il ne rend pas de position : c'est
// `Selectionne` qui publie, ce diagnostic ne fait que compter la meme chose.
func Diagnostique(scorees []CelluleScoree, r Reglage) DiagnosticSelection {
	var d DiagnosticSelection
	if len(scorees) == 0 {
		return d
	}
	if r.QuantileAmorce > 0 {
		d.SeuilAmorce, d.SeuilCroissance = seuilsHysteresis(scorees, r)
	} else {
		d.SeuilAmorce = seuilDeCarte(scorees, r)
		d.SeuilCroissance = d.SeuilAmorce
	}
	for _, c := range scorees {
		if c.Score >= d.SeuilAmorce {
			d.CellulesAmorce++
		}
	}
	avant := retenuesDe(scorees, r)
	d.CellulesRetenues = len(avant)
	d.TaillesAvantFermeture = tailles(Composantes(avant))
	apres := Fermeture(avant, r.FermetureRayonCellules)
	d.CellulesApresFermeture = len(apres)
	comps := ComposantesConnexite(apres, r.Connexite8)
	d.TaillesComposantes = tailles(comps)
	d.Composantes = len(comps)
	for _, t := range d.TaillesComposantes {
		if t >= r.TailleMiniComposante {
			d.ComposantesRetenues++
		}
	}
	return d
}

// tailles rend les tailles des composantes (deja triees par taille decroissante).
func tailles(comps [][]tactical.Cellule) []int {
	out := make([]int, 0, len(comps))
	for _, c := range comps {
		out = append(out, len(c))
	}
	return out
}
