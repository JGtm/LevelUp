//go:build research

package main

// v2_resume.go — LES SECTIONS DE LA CARTE V2 DANS LE RESUME : elles s ajoutent APRES celles du
// mode `fermeture`, qui ne changent pas. Les chiffres sont colles, sans interpretation : les
// verdicts appartiennent au rapport de la campagne.

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// ordreDesSorties et ordreDesRestes : l ordre des colonnes, fixe.
var (
	ordreDesSorties = []string{grammar.SortieVueBTerminateur.String(), grammar.SortieVueBRejetHorsDatum.String(),
		grammar.SortieVueBRejetDeVue.String(), grammar.SortieVueBAutre.String()}
	ordreDesRestes = []string{"0-7 (non nuls)", "8-63", ">= 64", "negatif"}
)

// ventilation resume les paquets « hors cadre » d une table.
type ventilation struct {
	total, enJeu, vide  int
	parSortie, parReste map[string]int
}

func ventiler(t map[cleHorsCadre]*compte) ventilation {
	v := ventilation{parSortie: map[string]int{}, parReste: map[string]int{}}
	for k, c := range t {
		v.total += c.paquets
		v.enJeu += c.enJeu
		v.parSortie[k.sortie] += c.paquets
		v.parReste[k.reste] += c.paquets
		if k.vueC == "vide" {
			v.vide += c.paquets
		}
	}
	return v
}

// cellules rend la ligne de ventilation : hors cadre, chaque sortie, vue C vide, chaque reste.
func (v ventilation) cellules() string {
	parts := []string{fmt.Sprintf("%d", v.total)}
	for _, s := range ordreDesSorties {
		parts = append(parts, part(v.parSortie[s], v.total))
	}
	parts = append(parts, part(v.vide, v.total))
	for _, rs := range ordreDesRestes {
		parts = append(parts, part(v.parReste[rs], v.total))
	}
	parts = append(parts, fmt.Sprintf("%d", v.enJeu))
	return strings.Join(parts, " | ")
}

// enteteVentilation : l en-tete des tables de ventilation.
const enteteVentilation = "Hors cadre | Sortie B : terminateur | rejet hors datum | rejet de vue | autre | " +
	"Vue C vide | Reste 0-7 non nuls | 8-63 | >= 64 | negatif | Utiles en jeu |"

// ecrireSection ecrit les sections de la carte v2.
func (r *rapportV2) ecrireSection(w io.Writer, top int) {
	fmt.Fprintln(w, "## Carte v2 — « vue C : terminateur hors cadre » ventile (item 1.1)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Sortie de la vue B : comment la boucle de records s est arretee avant la vue C ; vue C : "+
		"vide (son terminateur seul) ou non ; reste : bits du payload derriere le terminateur de la vue C.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Par build")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Build | Films | "+enteteVentilation)
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, b := range nomsTries(r.parBuild) {
		cb := r.parBuild[b]
		fmt.Fprintf(w, "| %s | %d | %s |\n", b, cb.films, ventiler(cb.horsCadre).cellules())
	}
	fmt.Fprintf(w, "| **corpus** | %d | %s |\n\n", r.corpus.films, ventiler(r.corpus.horsCadre).cellules())
	fmt.Fprintln(w, "### Par film")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Film | Build | "+enteteVentilation)
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, f := range r.filmsTries() {
		fmt.Fprintf(w, "| %s | %s | %s |\n", f.id, f.build, ventiler(f.m.horsCadre).cellules())
	}
	fmt.Fprintln(w)
	r.ecrireCroisements(w, top)
	r.ecrireSortiesDuResume(w)
	r.ecrireRejetsDuResume(w, top)
	r.ecrireEntreesDuResume(w)
	r.ecrireChunk3DuResume(w)
	r.ecrireBorneDuResume(w)
}

// filmsTries rend les films par build puis par identifiant.
func (r *rapportV2) filmsTries() []filmV2 {
	out := slices.Clone(r.films)
	slices.SortFunc(out, func(a, b filmV2) int {
		if c := strings.Compare(a.build, b.build); c != 0 {
			return c
		}
		return strings.Compare(a.id, b.id)
	})
	return out
}

// ecrireCroisements ecrit, sur le corpus, le croisement sortie x vue C x reste et le dernier
// composant lu.
func (r *rapportV2) ecrireCroisements(w io.Writer, top int) {
	fmt.Fprintln(w, "### Corpus : sortie x vue C x reste")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Sortie B | Vue C | Reste | Paquets | Utiles en jeu |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|")
	for i, k := range clesHorsCadre(r.corpus.horsCadre) {
		if top > 0 && i >= top {
			break
		}
		c := r.corpus.horsCadre[k]
		fmt.Fprintf(w, "| %s | %s | %s | %d | %d |\n", k.sortie, k.vueC, k.reste, c.paquets, c.enJeu)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "### Corpus : dernier record et dernier composant lus avant la fin de la vue B")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Sortie B | Dernier lu | Paquets | Utiles en jeu |")
	fmt.Fprintln(w, "|---|---|---|---|")
	for i, k := range clesDernier(r.corpus.dernier) {
		if top > 0 && i >= top {
			break
		}
		c := r.corpus.dernier[k]
		fmt.Fprintf(w, "| %s | %s | %d | %d |\n", k.sortie, cellule(k.dernier), c.paquets, c.enJeu)
	}
	fmt.Fprintln(w)
}

// ecrireSortiesDuResume ecrit les sorties de la vue B sur TOUS les paquets qui l atteignent.
func (r *rapportV2) ecrireSortiesDuResume(w io.Writer) {
	fmt.Fprintln(w, "## Sorties de la vue B, tous paquets qui l atteignent")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Build | Sortie B | Paquets | Fermes | Hors cadre | Autres causes |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|")
	builds := append(nomsTries(r.parBuild), "**corpus**")
	for _, b := range builds {
		cb := r.corpus
		if c, ok := r.parBuild[b]; ok {
			cb = c
		}
		for _, s := range nomsTries(cb.sorties) {
			v := cb.sorties[s]
			fmt.Fprintf(w, "| %s | %s | %d | %s | %s | %s |\n", b, s, v.paquets, part(v.fermes, v.paquets),
				part(v.horsCadre, v.paquets), part(v.autres, v.paquets))
		}
	}
	fmt.Fprintln(w)
}

// ecrireRejetsDuResume ecrit les rejets confrontes au bloc de type 1 (item 1.2).
func (r *rapportV2) ecrireRejetsDuResume(w io.Writer, top int) {
	fmt.Fprintln(w, "## Sorties par rejet contre le bloc de type 1 (item 1.2)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Etat du slot rejete dans le bloc de type 1 de SON chunk ; naissance : le bloc du chunk "+
		"SUIVANT porte-t-il une allocation de cet eid que le chunk ne portait pas, sans NEW lu ?")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Sortie B | Paquet | Etat au bloc | Naissance | Paquets | Utiles en jeu |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|")
	for i, k := range clesRejets(r.corpus.rejets) {
		if top > 0 && i >= top {
			break
		}
		c := r.corpus.rejets[k]
		fmt.Fprintf(w, "| %s | %s | %s | %s | %d | %d |\n", k.sortie, k.paquet, k.etat, k.naissance, c.paquets, c.enJeu)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Build | Rejets | dont paquet hors cadre | dont ferme | Vivant au bloc | Naissance non lue | Non mesurable |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|")
	for _, b := range append(nomsTries(r.parBuild), "**corpus**") {
		cb := r.corpus
		if c, ok := r.parBuild[b]; ok {
			cb = c
		}
		var tot, hc, fe, viv, nee, nm int
		for k, c := range cb.rejets {
			n := c.paquets
			tot += n
			hc += compterSi(k.paquet == "hors cadre", n)
			fe += compterSi(k.paquet == "ferme", n)
			viv += compterSi(k.etat == "vivant", n)
			nee += compterSi(k.naissance == "naissance non lue", n)
			nm += compterSi(k.naissance == "non mesurable", n)
		}
		fmt.Fprintf(w, "| %s | %d | %s | %s | %s | %s | %s |\n", b, tot, part(hc, tot), part(fe, tot),
			part(viv, tot), part(nee, tot), part(nm, tot))
	}
	fmt.Fprintln(w)
}

// compterSi rend `n` si la condition tient, 0 sinon.
func compterSi(ok bool, n int) int {
	if ok {
		return n
	}
	return 0
}

// ecrireEntreesDuResume ecrit le denominateur des entrees de controle utiles (item 1.3).
func (r *rapportV2) ecrireEntreesDuResume(w io.Writer) {
	fmt.Fprintln(w, "## Entrees de controle utiles (item 1.3)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Utile = entree `kind 0` qui porte le bloc de 0x68 octets, la seule que le tir continu lit. "+
		"Denominateur ESTIME = utiles fermees + moyenne par vue C fermee du film x paquets non fermes. "+
		"PAR CONSTRUCTION, la part estimee d un film egale sa part de vues C fermees (U / (U + U/F x N) = "+
		"F / (F + N)) : cette estimation ne dit rien de plus que les paquets.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Build | Paquets | Vues C fermees | Entrees fermees | Utiles fermees | Moyenne par vue C fermee | "+
		"Denominateur estime | Part estimee | Utiles lues hors fermeture (non prouvees) |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|")
	for _, b := range append(nomsTries(r.parBuild), "**corpus**") {
		e := r.corpus.entrees
		if c, ok := r.parBuild[b]; ok {
			e = c.entrees
		}
		// Le denominateur d un build est la SOMME des denominateurs de ses films (chaque film a sa
		// propre moyenne), pas la moyenne du build appliquee au build.
		d := r.denominateurDe(b)
		fmt.Fprintf(w, "| %s | %d | %s | %d | %d | %.2f | %.0f | %s | %d |\n", b, e.paquets,
			part(e.vuesCFermees, e.paquets), e.entreesFermees, e.utilesFermees, e.moyenne(), d,
			pourcent(float64(e.utilesFermees), d), e.utilesLuesHorsFermeture)
	}
	fmt.Fprintln(w)
}

// denominateurDe somme les denominateurs estimes des films d un build (ou du corpus).
func (r *rapportV2) denominateurDe(build string) float64 {
	d := 0.0
	for _, f := range r.films {
		if build == "**corpus**" || f.build == build {
			d += f.m.entrees.denominateurEstime()
		}
	}
	return d
}

// pourcent rend `a / b` en pourcentage, virgule decimale.
func pourcent(a, b float64) string {
	if b == 0 {
		return "-"
	}
	return strings.Replace(fmt.Sprintf("%.1f %%", 100*a/b), ".", ",", 1)
}

// ecrireChunk3DuResume ecrit, par film, le compte declare du chunk des temps forts (item 1.5).
func (r *rapportV2) ecrireChunk3DuResume(w io.Writer) {
	fmt.Fprintln(w, "## Chunk des temps forts : compte declare contre evenements trouves (item 1.5)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Film | Build | Chunk | Paquets type 9 | Declares | Trouves | Ecart | Kills | Deaths | "+
		"Medailles | Mode | Fil des morts | Refus |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, f := range r.filmsTries() {
		c := f.m.chunk3
		fmt.Fprintf(w, "| %s | %s | %d | %d | %d | %d | %d | %d | %d | %d | %d | %d | %s |\n", f.id, f.build,
			c.chunk, c.paquets9, c.declares, c.trouves, c.trouves-c.declares, c.kills, c.deaths, c.medailles,
			c.modes, c.filDesMorts, cellule(c.refus))
	}
	fmt.Fprintln(w)
}

// ecrireBorneDuResume ecrit le mode borne par build (item 1.6).
func (r *rapportV2) ecrireBorneDuResume(w io.Writer) {
	fmt.Fprintln(w, "## Mode borne : lectures au-dela de la fin du payload (item 1.6)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Build | Records lus (vue B) | Records debordants | Composants debordants | "+
		"NEW propres debordants | Paquets avec debordement | Terminateurs de vue B lus au-dela | Chunks consultes sans bloc de type 1 | Blocs de type 1 illisibles |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|")
	for _, b := range append(nomsTries(r.parBuild), "**corpus**") {
		c := r.corpus
		if cb, ok := r.parBuild[b]; ok {
			c = cb
		}
		x := c.borne
		fmt.Fprintf(w, "| %s | %d | %s | %d | %d | %d | %d | %d | %d |\n", b, x.records, part(x.debordants, x.records),
			x.composants, x.neufs, x.paquets, x.terminateursAuDela, c.sansBloc, c.blocsIllisibles)
	}
	fmt.Fprintln(w)
}
