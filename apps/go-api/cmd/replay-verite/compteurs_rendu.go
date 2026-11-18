package main

// compteurs_rendu.go — LA SORTIE DU MODE `-compteurs` : tableaux d'accord, defauts, transitions. Aucun identifiant
// de joueur n'y passe : seuls des rejeux sont nommes, par les 8 premiers caracteres de leur match.

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// pct rend un pourcentage a deux decimales, « n/a » sans denominateur.
func pct(a, b int) string {
	if b == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.2f %%", 100*float64(a)/float64(b))
}

// rendreAccord ecrit le tableau « accord exact, par build et par compteur » d'une mesure : exact / lus, et
// entre parentheses les joueurs que le film ne porte pas.
func rendreAccord(out io.Writer, titre string, m *mesure) {
	_, _ = fmt.Fprintf(out, "\n%s (exact / lus ; entre parentheses : valeur absente du film)\n", titre)
	_, _ = fmt.Fprintf(out, "| build | %s |\n|---|%s\n", strings.Join(nomsCompteurs, " | "), strings.Repeat("---|", nbCompteurs))
	builds := make([]string, 0, len(m.parBuild))
	for b := range m.parBuild {
		builds = append(builds, b)
	}
	sort.Strings(builds)
	var tous [nbCompteurs]casse
	for _, b := range builds {
		ligne := []string{b}
		for c := range nbCompteurs {
			ligne = append(ligne, cellule(m.parBuild[b][c]))
			for i := range tous[c] {
				tous[c][i] += m.parBuild[b][c][i]
			}
		}
		_, _ = fmt.Fprintf(out, "| %s |\n", strings.Join(ligne, " | "))
	}
	ligne := []string{"TOUS"}
	for c := range nbCompteurs {
		ligne = append(ligne, cellule(tous[c])+" "+pct(tous[c][issueExact], tous[c][issueExact]+tous[c][issueFaux]))
	}
	_, _ = fmt.Fprintf(out, "| %s |\n", strings.Join(ligne, " | "))
}

// cellule rend « exact/lus (+absents) » d'une casse.
func cellule(c casse) string {
	lus := c[issueExact] + c[issueFaux]
	s := fmt.Sprintf("%d/%d", c[issueExact], lus)
	if abs := c[issueAbsentZero] + c[issueAbsentPlus]; abs > 0 {
		s += fmt.Sprintf(" (+%d)", abs)
	}
	return s
}

// rendreDefauts ecrit les defauts que l'API ne dit pas.
func rendreDefauts(out io.Writer, titre string, m *mesure) {
	_, _ = fmt.Fprintf(out, "\n%s\n", titre)
	_, _ = fmt.Fprintf(out, "  couples joueur-manche lus : %d ; compteur nomme > eliminations de la manche : %d couples dans %d rejeux\n",
		m.couples, m.violations, len(m.rejeuxViolations))
	_, _ = fmt.Fprintf(out, "  serie (maxSpree) > eliminations du joueur : %d joueurs dans %d rejeux\n", m.series, len(m.rejeuxSeries))
	_, _ = fmt.Fprintf(out, "  grenades >= %d : %d joueurs ; egales a %d : %d dans %d rejeux\n", seuilGrenadeSuspecte,
		m.grenadesHautes, valeurGrenadeSuspecte, m.grenades16, len(m.rejeuxGrenades16))
}

// rendreRejeuxAFaute liste les rejeux a defaut, du plus charge au moins charge (8 premiers caracteres) ; apres le compte,
// la nature : c = couple ou un compteur depasse les eliminations, s = serie aberrante, g = grenades a 16.
func rendreRejeuxAFaute(out io.Writer, m *mesure) {
	ids := make([]string, 0, len(m.defauts))
	for id := range m.defauts {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(a, b int) bool {
		if m.defauts[ids[a]] != m.defauts[ids[b]] {
			return m.defauts[ids[a]] > m.defauts[ids[b]]
		}
		return ids[a] < ids[b]
	})
	_, _ = fmt.Fprintf(out, "\nrejeux a defaut (%d) : ", len(ids))
	for _, id := range ids {
		_, _ = fmt.Fprintf(out, "%s:%d%s ", id, m.defauts[id], m.natures[id])
	}
	_, _ = fmt.Fprintln(out)
}

// rendreCompteurs ecrit la sortie du mode.
func rendreCompteurs(out io.Writer, o optionsCompteurs, l *lecture) {
	avant, apres, tr := l.avant, l.apresMes, l.tr
	_, _ = fmt.Fprintf(out, "COMPTEURS NOMMES CONTRE L'API : %d rejeux, %d joueurs du tableau, %d joints a l'API\n",
		avant.rejeux, avant.joueurs, avant.joints)
	rendreAccord(out, "ACCORD EXACT"+etiquette(o, " AVANT"), avant)
	rendreDefauts(out, "DEFAUTS"+etiquette(o, " AVANT"), avant)
	if o.detail {
		rendreRejeuxAFaute(out, avant)
	}
	if o.apres == "" {
		return
	}
	rendreAccord(out, "ACCORD EXACT APRES", apres)
	rendreDefauts(out, "DEFAUTS APRES", apres)
	if o.detail {
		rendreRejeuxAFaute(out, apres)
	}
	rendreTransitions(out, tr)
}

// etiquette rend `suffixe` en comparaison, rien sinon.
func etiquette(o optionsCompteurs, suffixe string) string {
	if o.apres != "" {
		return suffixe
	}
	return ""
}

// rendreTransitions ecrit, par compteur, les joueurs dont l'accord large est gagne / perdu, par build.
func rendreTransitions(out io.Writer, tr transitions) {
	_, _ = fmt.Fprintf(out, "\nACCORD LARGE (le film absent vaut 0), JOUEURS GAGNES / PERDUS entre avant et apres\n")
	_, _ = fmt.Fprintf(out, "| build | %s |\n|---|%s\n", strings.Join(nomsCompteurs, " | "), strings.Repeat("---|", nbCompteurs))
	builds := map[string]bool{}
	for b := range tr.gagnes {
		builds[b] = true
	}
	for b := range tr.perdus {
		builds[b] = true
	}
	noms := make([]string, 0, len(builds))
	for b := range builds {
		noms = append(noms, b)
	}
	sort.Strings(noms)
	var gt, pt [nbCompteurs]int
	for _, b := range noms {
		ligne := []string{b}
		for c := range nbCompteurs {
			g, p := cellEntier(tr.gagnes[b], c), cellEntier(tr.perdus[b], c)
			gt[c] += g
			pt[c] += p
			ligne = append(ligne, fmt.Sprintf("+%d / -%d", g, p))
		}
		_, _ = fmt.Fprintf(out, "| %s |\n", strings.Join(ligne, " | "))
	}
	ligne := []string{"TOUS"}
	for c := range nbCompteurs {
		ligne = append(ligne, fmt.Sprintf("+%d / -%d", gt[c], pt[c]))
	}
	_, _ = fmt.Fprintf(out, "| %s |\n", strings.Join(ligne, " | "))
}

// cellEntier lit une case d'un tableau de compteurs absent ou non.
func cellEntier(t *[nbCompteurs]int, c int) int {
	if t == nil {
		return 0
	}
	return t[c]
}
