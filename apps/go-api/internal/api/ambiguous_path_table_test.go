// ambiguous_path_table_test.go — la table UNIQUE des formes de chemin ambigu que toutes les
// gardes qui décident sur un chemin doivent refuser (garde-rail C10 : une garde qui lit le
// chemin décodé ou nettoyé ne doit jamais laisser passer un chemin que chi route autrement).
//
// Exportée (fichier de test du paquet api) pour que les tests de la pile complète
// (paquet api_test, routeur assemblé) passent exactement la même table que les tests des
// gardes (ambiguous_path_guards_test.go).
package api

import "strings"

// AmbiguousForm est une manière d'écrire un séparateur et un segment « .. » qui fait lire
// au chemin décodé (`URL.Path`) autre chose que ce que chi route (`URL.RawPath`, ou le
// chemin littéral avant nettoyage).
type AmbiguousForm struct {
	Name string
	// Sep remplace chaque "/" entre segments, Dot chaque segment "..".
	Sep, Dot string
	// OneSegment : le résultat reste un seul segment pour chi (séparateur encodé), donc
	// peut tenir dans un paramètre de chemin.
	OneSegment bool
}

// AmbiguousForms : séparateurs encodés (%2F, %5C, casses mêlées), points encodés (%2e%2e),
// segments vides (//), segments « . » et « .. » en clair.
var AmbiguousForms = []AmbiguousForm{
	{Name: "%2F", Sep: "%2F", Dot: "..", OneSegment: true},
	{Name: "%2f", Sep: "%2f", Dot: "..", OneSegment: true},
	{Name: "%5C", Sep: "%5C", Dot: "..", OneSegment: true},
	{Name: "%5c", Sep: "%5c", Dot: "..", OneSegment: true},
	{Name: "%2e%2e+%2F", Sep: "%2F", Dot: "%2e%2e", OneSegment: true},
	{Name: "%2E%2e+%2f (casses mêlées)", Sep: "%2f", Dot: "%2E%2e", OneSegment: true},
	{Name: "%2E%2E en segment clair", Sep: "/", Dot: "%2E%2E"},
	{Name: "//", Sep: "//", Dot: ".."},
	{Name: "/./", Sep: "/./", Dot: ".."},
	{Name: "/../ en clair", Sep: "/", Dot: ".."},
}

// Smuggle écrit les segments sous la forme f : chaque ".." devient f.Dot, les segments
// sont joints par f.Sep. Les appelants mettent au moins un ".." dans segs : c'est lui qui
// rend le chemin ambigu quelle que soit la forme.
func Smuggle(f AmbiguousForm, segs ...string) string {
	out := make([]string, len(segs))
	for i, s := range segs {
		if s == ".." {
			s = f.Dot
		}
		out[i] = s
	}
	return strings.Join(out, f.Sep)
}

// SmuggleParam rend une valeur de paramètre de chemin ambiguë autour de id : un seul segment
// pour chi quelle que soit la forme (séparateur encodé autour d'un « .. », ou le « .. » seul
// quand la forme sépare en clair).
func SmuggleParam(f AmbiguousForm, id string) string {
	if f.OneSegment {
		return Smuggle(f, id, "..", id)
	}
	return f.Dot
}
