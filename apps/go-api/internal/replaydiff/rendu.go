package replaydiff

// rendu.go — LES DEUX SORTIES : un tableau qu'un operateur lit, un JSON qu'un balayage agrege.
//
// LE TABLEAU EST GROUPE PAR AXE et non par ordre alphabetique de metrique : celui qui lit
// cherche « est-ce que les objectifs ont perdu quelque chose », pas « que vaut la mesure
// objectives/par-joueur/2533.../flag_captures ». Le JSON, lui, porte TOUT — c'est lui que le
// rapport de parc agrege sur 161 paires.

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ordreSens classe les sens du plus grave au moins grave : ce qui a DISPARU d'abord.
var ordreSens = map[string]int{
	SensDisparu: 0, SensPerte: 1, SensChangement: 2, SensGain: 3, SensApparu: 4,
}

// trierDifferences ordonne les ecarts : axe, puis gravite, puis metrique — un ordre STABLE,
// pour qu'un rapport rejoue deux fois soit identique deux fois.
func trierDifferences(diffs []Difference) {
	sort.SliceStable(diffs, func(i, j int) bool {
		if diffs[i].Axe != diffs[j].Axe {
			return diffs[i].Axe < diffs[j].Axe
		}
		if ordreSens[diffs[i].Sens] != ordreSens[diffs[j].Sens] {
			return ordreSens[diffs[i].Sens] < ordreSens[diffs[j].Sens]
		}
		return diffs[i].Metrique < diffs[j].Metrique
	})
}

// formater rend un nombre sans notation scientifique inutile.
func formater(v float64) string {
	if v == float64(int64(v)) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 4, 64)
}

// AfficherTableau ecrit le rapport lisible.
func AfficherTableau(w io.Writer, r Rapport, tout bool) {
	_, _ = fmt.Fprintf(w, "match %s : schema %d -> %d   (%s -> %s)\n",
		court(r.MatchID), r.SchemaAncien, r.SchemaNouveau,
		nomCourt(r.FichierAncien), nomCourt(r.FichierNouveau))
	if len(r.Differences) == 0 {
		_, _ = fmt.Fprintf(w, "  aucune difference (%d mesures identiques)\n", r.Identiques)
		return
	}
	_, _ = fmt.Fprintf(w, "  %d ecarts sur %d mesures\n", len(r.Differences), len(r.Differences)+r.Identiques)
	afficherBilans(w, r)
	axeCourant := ""
	for _, d := range r.Differences {
		if d.Axe != axeCourant {
			axeCourant = d.Axe
			_, _ = fmt.Fprintf(w, "\n  [%s]\n", axeCourant)
		}
		_, _ = fmt.Fprintf(w, "    %-9s %-58s %12s -> %-12s\n",
			d.Sens, tronquer(d.Metrique, 58), vide(d.Ancien), vide(d.Nouveau))
	}
	if tout {
		_, _ = fmt.Fprintf(w, "\n  (%d mesures identiques non listees)\n", r.Identiques)
	}
}

// afficherBilans ecrit la ligne de bilan par axe, pertes en tete.
func afficherBilans(w io.Writer, r Rapport) {
	axes := make([]string, 0, len(r.Bilans))
	for a := range r.Bilans {
		axes = append(axes, a)
	}
	sort.Slice(axes, func(i, j int) bool {
		bi, bj := r.Bilans[axes[i]], r.Bilans[axes[j]]
		if bi.Pertes != bj.Pertes {
			return bi.Pertes > bj.Pertes
		}
		return axes[i] < axes[j]
	})
	for _, a := range axes {
		b := r.Bilans[a]
		if b.Pertes == 0 && b.Gains == 0 && b.Changements == 0 {
			continue
		}
		_, _ = fmt.Fprintf(w, "    %-18s pertes=%-4d gains=%-4d changements=%-4d identiques=%d\n",
			a, b.Pertes, b.Gains, b.Changements, b.Identiques)
	}
}

// EcrireJSON depose le rapport de la paire.
func EcrireJSON(path string, r Rapport) error {
	blob, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation du rapport : %w", err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		return fmt.Errorf("ecriture du rapport %s : %w", path, err)
	}
	return nil
}

func court(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func nomCourt(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

func tronquer(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func vide(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
