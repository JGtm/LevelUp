//go:build research

package reapparition

// univers.go — L'UNIVERS DES NOMS DE COMPOSANT, BALAYE DANS L'IMAGE.
//
// POURQUOI UN NEGATIF A BESOIN DE CECI. Dire « le film n'ecrit pas le minuteur de reapparition
// d'un vehicule » a partir de `ecs_table.tsv` ne prouve rien : la table ne connait que les
// archetypes rencontres dans les films decodes. Le balayage de l'IMAGE, lui, rend TOUS les noms
// de composant que le moteur embarque, qu'un film les ait portes ou non. Un vocabulaire absent
// de cet univers est absent du moteur, pas seulement du corpus.
//
// LA DEFINITION EST ETROITE ET ECRITE : une chaine C isolee d'une section de donnees, faite de
// minuscules, de chiffres et de tirets, terminee par `-component` ou par `-component-<n>` (les
// instances multiples que la note de methode decrit, § 6). Elle laisse dehors les noms ECS
// ecrits sans le suffixe (`biped-action`, `simulation-state`) : le rapport le dit, et le
// vocabulaire se cherche alors dans le pool complet des chaines, pas ici.

import "sort"

const suffixeComposant = "-component"

// NomsDeComposants rend, triee et dedupliquee, la liste des noms de composant de l'image.
func (e *Executable) NomsDeComposants() []string {
	vus := map[string]bool{}
	for i := range e.sections {
		s := &e.sections[i]
		if s.execCode {
			continue
		}
		c := s.contenu
		deb := 0
		for off := 0; off < len(c); off++ {
			if c[off] != 0 {
				continue
			}
			if n := off - deb; n >= len(suffixeComposant) {
				if mot := string(c[deb:off]); estNomDeComposant(mot) {
					vus[mot] = true
				}
			}
			deb = off + 1
		}
	}
	out := make([]string, 0, len(vus))
	for n := range vus {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// estNomDeComposant applique la definition ecrite en tete de fichier.
func estNomDeComposant(s string) bool {
	if len(s) < len(suffixeComposant)+2 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-'
		if !ok {
			return false
		}
	}
	if finit(s, suffixeComposant) {
		return true
	}
	// forme instanciee : `...-component-<n>`
	j := len(s)
	for j > 0 && s[j-1] >= '0' && s[j-1] <= '9' {
		j--
	}
	if j == len(s) || j == 0 || s[j-1] != '-' {
		return false
	}
	return finit(s[:j-1], suffixeComposant)
}

func finit(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}
