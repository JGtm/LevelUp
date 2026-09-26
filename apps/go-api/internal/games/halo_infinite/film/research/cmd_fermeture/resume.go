//go:build research

package main

// resume.go — LE RESUME MARKDOWN, ecrit a la fin du corpus : par build, la part des paquets
// fermes par vue et la part des RECORDS UTILES fermes (le declencheur de la spec de la
// representation intermediaire, §9 : >= 95 % sur chaque build) ; puis le classement des causes
// d arret du corpus, avec le gain potentiel de chacune.

import (
	"cmp"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// seuilDeclencheur est le seuil de la spec (§9) : la part des records utiles fermes, par build.
const seuilDeclencheur = 0.95

// terminer ferme les TSV et ecrit le resume.
func (r *rapport) terminer(top int) error {
	if err := r.fermer(); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(r.dir, "fermeture_resume.md")) //nolint:gosec // repertoire verifie
	if err != nil {
		return fmt.Errorf("creation du resume : %w", err)
	}
	r.ecrireResume(f, top)
	return f.Close()
}

// ecrireResume ecrit le resume Markdown.
func (r *rapport) ecrireResume(w io.Writer, top int) {
	fmt.Fprintf(w, "# Carte de fermeture des trames delta — %s\n\n", time.Now().Format("2006-01-02"))
	fmt.Fprintf(w, "Films mesures : %d ; echecs : %d. Table ECS : `%s`. Contexte d instrument "+
		"(largeurs d axe lues dans le film, profil par defaut, sans calibration killsource).\n\n",
		r.mesures, r.echecs, r.tab.chemin)
	fmt.Fprintln(w, "## Par build")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Build | Films | Paquets fermes | Vue A | Vue B | Vue C | Records utiles fermes | "+
		"Entrees de controle lues | Declencheur (>= 95 %) | Pic memoire max |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|")
	for _, build := range nomsTries(r.parBuild) {
		cb := r.parBuild[build]
		verdict := "non atteint"
		if cb.utiles == 0 {
			verdict = "aucun record utile lu"
		}
		if cb.utiles > 0 && float64(cb.utilesFermes) >= seuilDeclencheur*float64(cb.utiles) {
			verdict = "ATTEINT"
		}
		fmt.Fprintf(w, "| %s | %d | %s | %s | %s | %s | %s | %d | %s | %s |\n", build, cb.films,
			part(cb.fermes, cb.paquets), part(cb.vuesFermees[grammar.VueMessages], cb.atteints[grammar.VueMessages]),
			part(cb.vuesFermees[grammar.VueEntites], cb.atteints[grammar.VueEntites]),
			part(cb.vuesFermees[grammar.VueControle], cb.atteints[grammar.VueControle]),
			part(cb.utilesFermes, cb.utiles), cb.entrees, verdict, mio(cb.picMax))
	}
	fmt.Fprintln(w)
	r.ecrireClassement(w, top)
}

// ecrireClassement ecrit les causes d arret du corpus, par paquets bloques decroissants.
func (r *rapport) ecrireClassement(w io.Writer, top int) {
	fmt.Fprintln(w, "## Causes d arret (premiere cause de chaque paquet non ferme)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Gain potentiel = records utiles LUS et non fermes dans les paquets que la cause "+
		"arrete : BORNE SUPERIEURE (une autre cause peut suivre ; les records d apres l arret ne "+
		"sont pas lus du tout).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Rang | Cause | Archetype | Index | Statut | Usage produit | Paquets bloques | "+
		"Gain potentiel | Builds |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|")
	cles := nomsTries(r.parBloquant)
	slices.SortStableFunc(cles, func(a, b string) int {
		return cmp.Or(cmp.Compare(r.parBloquant[b].paquets, r.parBloquant[a].paquets), strings.Compare(a, b))
	})
	for rang, cle := range cles {
		if top > 0 && rang >= top {
			break
		}
		b := r.parBloquant[cle]
		ti := "-"
		if b.ti >= 0 {
			ti = fmt.Sprintf("%d", b.ti)
		}
		fmt.Fprintf(w, "| %d | %s | %s | %s | %s | %s | %d | %d | %s |\n", rang+1, cle, ti,
			strings.Join(nomsTries(b.index), ", "), cellule(b.statut), cellule(b.usage), b.paquets,
			b.enJeu, strings.Join(nomsTries(b.builds), ", "))
	}
}

// part rend `fermes/total (xx,x %)`.
func part(fermes, total int) string {
	if total == 0 {
		return "0/0"
	}
	pc := strings.Replace(fmt.Sprintf("%.1f", 100*float64(fermes)/float64(total)), ".", ",", 1)
	return fmt.Sprintf("%d/%d (%s %%)", fermes, total, pc)
}

// mio rend des octets en mebioctets.
func mio(o uint64) string { return fmt.Sprintf("%.0f Mio", float64(o)/(1<<20)) }

// cellule neutralise une valeur de table pour une cellule Markdown.
func cellule(v string) string {
	if v == "" {
		return "-"
	}
	return strings.ReplaceAll(v, "|", "/")
}

// nomsTries rend les cles d une table, triees.
func nomsTries[V any](m map[string]V) []string { return slices.Sorted(maps.Keys(m)) }

// clesTriees rend les cles entieres d une table, triees.
func clesTriees[V any](m map[int]V) []int { return slices.Sorted(maps.Keys(m)) }
