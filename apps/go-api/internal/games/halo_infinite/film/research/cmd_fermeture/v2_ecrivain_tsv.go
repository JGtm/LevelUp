//go:build research

package main

// v2_ecrivain_tsv.go — LES SORTIES DU LOT L0 DE LA CARTE V2 (`v2_ecrivain.go`) : trois TSV par
// film, la section du resume, le denominateur fixe lu sur `-denominateur-fixe` et la ligne par
// paquet de `-paquets`.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// optionsV2 : ce que la ligne de commande donne a la carte v2.
type optionsV2 struct {
	// fixe : le denominateur fixe consolide, par film (`-denominateur-fixe`) ; vide : non donne.
	fixe map[string]int
	// paquets : ecrire `fermeture_paquets.tsv`.
	paquets bool
}

// lireDenominateurFixe lit un TSV dont l en-tete porte les colonnes `film` et `fixe`
// (`r_comb2_denominateurs.tsv`). Chemin vide : aucun denominateur fixe.
func lireDenominateurFixe(chemin string) (map[string]int, error) {
	out := map[string]int{}
	if chemin == "" {
		return out, nil
	}
	f, err := os.Open(chemin) //nolint:gosec // chemin donne par l operateur, lu seulement
	if err != nil {
		return nil, fmt.Errorf("-denominateur-fixe : %w", err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	colFilm, colFixe := -1, -1
	for sc.Scan() {
		champs := strings.Split(sc.Text(), "\t")
		if colFilm < 0 {
			for i, c := range champs {
				switch c {
				case "film":
					colFilm = i
				case "fixe":
					colFixe = i
				}
			}
			if colFilm < 0 || colFixe < 0 {
				return nil, errors.New("-denominateur-fixe : colonnes `film` et `fixe` absentes de l en-tete")
			}
			continue
		}
		if len(champs) <= max(colFilm, colFixe) {
			continue
		}
		n, err := strconv.Atoi(champs[colFixe])
		if err != nil {
			return nil, fmt.Errorf("-denominateur-fixe : film %s : %w", champs[colFilm], err)
		}
		out[champs[colFilm]] = n
	}
	return out, sc.Err()
}

// tsvL0 : les TSV du lot L0, crees par [ouvrirRapportV2].
func tsvL0(r *rapportV2) []struct {
	f           **os.File
	nom, entete string
} {
	out := []struct {
		f           **os.File
		nom, entete string
	}{
		{&r.ecrivain, "fermeture_ecrivain.tsv", "film\tbuild\tmesure\tregle\tpaquets\tutiles"},
		{&r.denominateurs, "fermeture_denominateurs.tsv", "film\tbuild\tutiles_lus\tutiles_fermes_au_bit\t" +
			"utiles_fermes\tpart_variable\tfixe_donne\tfixe\tpart_fixe"},
		{&r.temoinsDecales, "fermeture_temoins_decales.tsv", "film\tbuild\tpaquets_fermes\tdecalage\t" +
			"refermes\tpart"},
	}
	if r.opts.paquets {
		out = append(out, struct {
			f           **os.File
			nom, entete string
		}{&r.paquets, "fermeture_paquets.tsv", "film\tchunk\tpaquet\tliste\tsortie_vueB\tferme_au_bit\t" +
			"ferme\tpremiere_regle\tregles\tcause\tutiles_lus\treste_apres_entete_rejete\tdeborde"})
	}
	return out
}

// ecrireEcrivain ecrit `fermeture_ecrivain.tsv` pour un film.
func (r *rapportV2) ecrireEcrivain(id, build string, m *mesureV2) error {
	e := m.l0
	w := r.ecrivain
	if _, err := fmt.Fprintf(w, "%s\t%s\tpaquets\tfermes au bit\t%d\t%d\n%s\t%s\tpaquets\tfermes\t%d\t%d\n",
		id, build, e.paquetsFermesAuBit, e.utilesFermesAuBit, id, build, e.paquetsFermes, e.utilesFermes); err != nil {
		return err
	}
	for _, k := range nomsTries(e.retires) {
		if _, err := fmt.Fprintf(w, "%s\t%s\tretires (premiere regle)\t%s\t%d\t%d\n", id, build, k,
			e.retires[k].paquets, e.retires[k].enJeu); err != nil {
			return err
		}
	}
	for v := 1; v < grammar.NombreDInvariants; v++ {
		nom := grammar.InvariantEcrivain(v).String()
		if _, err := fmt.Fprintf(w, "%s\t%s\tfermes au bit portant\t%s\t%d\t-\n%s\t%s\tnon fermes portant\t%s\t%d\t-\n",
			id, build, nom, e.portees[v], id, build, nom, e.nonFermes[v]); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%s\t%s\tlistes d evenements\tlocalisees\t%d\t-\n%s\t%s\tlistes d evenements\tnon localisees\t%d\t-\n"+
		"%s\t%s\tlistes d evenements\tlues (fin de la vue A)\t%d\t-\n",
		id, build, e.listesLocalisees, id, build, e.listesNonLocalisees, id, build, e.listesLues)
	return err
}

// fixeDe rend le denominateur fixe d un film : le maximum du fixe donne et des utiles lus par
// cette marche (la regle « Pourcentages » du plan : le fixe monte quand une marche lit plus loin).
func (r *rapportV2) fixeDe(id string, lus int) (donne, fixe int) {
	donne = r.opts.fixe[id]
	return donne, max(donne, lus)
}

// ecrireDenominateurs ecrit `fermeture_denominateurs.tsv` pour un film.
func (r *rapportV2) ecrireDenominateurs(id, build string, m *mesureV2) error {
	e := m.l0
	donne, fixe := r.fixeDe(id, e.utilesLus)
	_, err := fmt.Fprintf(r.denominateurs, "%s\t%s\t%d\t%d\t%d\t%s\t%d\t%d\t%s\n", id, build, e.utilesLus,
		e.utilesFermesAuBit, e.utilesFermes, pourcent(float64(e.utilesFermes), float64(e.utilesLus)),
		donne, fixe, pourcent(float64(e.utilesFermes), float64(fixe)))
	return err
}

// ecrireTemoinsDecales ecrit `fermeture_temoins_decales.tsv` pour un film.
func (r *rapportV2) ecrireTemoinsDecales(id, build string, m *mesureV2) error {
	e := m.l0
	for b, n := range e.temoins {
		if _, err := fmt.Fprintf(r.temoinsDecales, "%s\t%s\t%d\t%+d\t%d\t%s\n", id, build, e.temoinsFermes,
			decalageDuBit(b), n, pourcent(float64(n), float64(e.temoinsFermes))); err != nil {
			return err
		}
	}
	return nil
}

// ecrirePaquet ecrit la ligne d un paquet dans `fermeture_paquets.tsv`.
func ecrirePaquet(w io.Writer, id string, p grammar.PaquetDeCarte) error {
	liste := "sans"
	switch {
	case p.ListeNonLocalisee:
		liste = "non localisee"
	case p.ListeLocalisee:
		liste = "localisee"
	case p.ListeLue:
		liste = "lue"
	}
	reste := -1
	if p.Sortie.EstUnRejet() {
		reste = p.Bits - p.FinVueB
	}
	_, err := fmt.Fprintf(w, "%s\t%d\t%d\t%s\t%s\t%t\t%t\t%s\t%s\t%s\t%d\t%d\t%t\n", id, p.Chunk, p.Index,
		liste, p.Sortie, p.FermeeAuBit, p.Fermee, p.Invariant, nomsDesRegles(p.Invariants),
		cellule(p.Cause), p.UtilesLus, reste, p.Deborde)
	return err
}

// ecrireSectionL0 ecrit, dans le resume, les mesures du lot L0 par build et sur le corpus.
func (r *rapportV2) ecrireSectionL0(w io.Writer) {
	fmt.Fprintln(w, "## Lot L0 — fermes au bit pres contre fermes (regles de l ecrivain)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Build | Fermes au bit | Fermes | Retires | Utiles lus | Utiles fermes au bit | "+
		"Utiles fermes | Part variable | Fixe | Part fixe |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|")
	builds := append(nomsTries(r.parBuild), "**corpus**")
	for _, b := range builds {
		cb := r.corpus
		if c, ok := r.parBuild[b]; ok {
			cb = c
		}
		e := cb.l0
		fixe := r.fixeDuBuild(b)
		fmt.Fprintf(w, "| %s | %d | %d | %d | %d | %d | %d | %s | %d | %s |\n", b, e.paquetsFermesAuBit,
			e.paquetsFermes, e.paquetsFermesAuBit-e.paquetsFermes, e.utilesLus, e.utilesFermesAuBit,
			e.utilesFermes, pourcent(float64(e.utilesFermes), float64(e.utilesLus)), fixe,
			pourcent(float64(e.utilesFermes), float64(fixe)))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Corpus : fermetures au bit retirees, par premiere regle")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Regle | Paquets | Utiles |")
	fmt.Fprintln(w, "|---|---|---|")
	for _, k := range nomsTries(r.corpus.l0.retires) {
		fmt.Fprintf(w, "| %s | %d | %d |\n", k, r.corpus.l0.retires[k].paquets, r.corpus.l0.retires[k].enJeu)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Corpus : temoins decales (vue C relue depuis un depart decale de k bits)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| k | Refermes | Part des fermes |")
	fmt.Fprintln(w, "|---|---|---|")
	e := r.corpus.l0
	for b, n := range e.temoins {
		fmt.Fprintf(w, "| %+d | %d | %s |\n", decalageDuBit(b), n, pourcent(float64(n), float64(e.temoinsFermes)))
	}
	fmt.Fprintln(w)
}

// fixeDuBuild somme le denominateur fixe des films d un build (« **corpus** » : tous).
func (r *rapportV2) fixeDuBuild(build string) int {
	s := 0
	for _, f := range r.films {
		if build == "**corpus**" || f.build == build {
			_, fixe := r.fixeDe(f.id, f.m.l0.utilesLus)
			s += fixe
		}
	}
	return s
}

// lireOptionsV2 rassemble les options de la carte v2 donnees sur la ligne de commande.
func lireOptionsV2(cheminFixe string, paquets bool) (optionsV2, error) {
	fixe, err := lireDenominateurFixe(cheminFixe)
	return optionsV2{fixe: fixe, paquets: paquets}, err
}
