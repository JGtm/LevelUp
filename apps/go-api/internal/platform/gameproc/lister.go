package gameproc

import "strings"

// Lister énumère les noms d'exécutables des processus en cours (nom de fichier seul, sans chemin).
type Lister interface {
	Names() ([]string, error)
}

// ListerFunc adapte une fonction en [Lister].
type ListerFunc func() ([]string, error)

// Names appelle f.
func (f ListerFunc) Names() ([]string, error) { return f() }

// NormalizeName rend la forme comparable d'un nom d'exécutable : sans espaces, en minuscules (les
// noms de fichiers de Windows ne sont pas sensibles à la casse).
func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// AnyRunning dit si l'un des noms voulus figure parmi les processus. Une liste voulue vide n'est
// jamais « en cours » : un titre sans client PC (liste vide) ne déclenche rien.
func AnyRunning(running, wanted []string) bool {
	if len(wanted) == 0 {
		return false
	}
	want := make(map[string]struct{}, len(wanted))
	for _, w := range wanted {
		if n := NormalizeName(w); n != "" {
			want[n] = struct{}{}
		}
	}
	for _, r := range running {
		if _, ok := want[NormalizeName(r)]; ok {
			return true
		}
	}
	return false
}
