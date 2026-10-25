package domain

import (
	"sort"
	"strings"
)

// MediaAudience est l'ensemble des auteurs dont une requête peut montrer les médias :
// la règle qui le calcule (joueur de la page + co-membres de ses groupes, réduits à
// ce que le lecteur consulte lui-même) vit dans authz.MediaAuthorXUIDs.
//
// Un auteur se nomme comme media_files.player_slug le porte : slug OU gamertag du
// profil, comparés sans casse. La valeur zéro n'autorise personne — une audience
// non résolue ferme la galerie au lieu de l'ouvrir.
type MediaAudience struct {
	authors map[string]bool
}

// NewMediaAudience construit l'audience des auteurs nommés ; les noms vides sont ignorés.
func NewMediaAudience(names ...string) MediaAudience {
	a := MediaAudience{authors: make(map[string]bool, len(names))}
	for _, n := range names {
		if k := strings.ToLower(strings.TrimSpace(n)); k != "" {
			a.authors[k] = true
		}
	}
	return a
}

// Allows dit si les médias de l'auteur `author` sont visibles.
func (a MediaAudience) Allows(author string) bool {
	return a.authors[strings.ToLower(strings.TrimSpace(author))]
}

// Authors rend les auteurs autorisés, en minuscules et triés (vide si personne).
func (a MediaAudience) Authors() []string {
	out := make([]string, 0, len(a.authors))
	for k := range a.authors {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
