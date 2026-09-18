//go:build research

package grammar

// niveaux_registre_research_test.go — LOT 5.1.7-a : LE `level` QUE CHAQUE FILM ECRIT, PAR
// COMPOSANT QUI EN FAIT UNE LARGEUR.
//
// `param_4` est le `level` du registre du film. Le registre est PAR FILM (chunk 0) : la table
// `ecs_table.tsv` n en fige qu UN. Cet instrument lit le registre de chaque film nomme et rend
// le niveau des huit composants dont un deserialiseur branche dessus — pour savoir si la
// grandeur est CONSTANTE d un build a l autre, ou si elle varie (auquel cas un ratchet sur une
// seule table ne garde rien).
//
// LECTURE SEULE, sans decodage de paquet (chunk 0 seulement) :
//
//	NIV_FILMS='<abs>/film_chunks/084a804d;<abs>/film_chunks/a349fea8' \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestNiveauxDuRegistre$' -v

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// nivCibles : (archetype, composant) dont un deserialiseur lit `param_4`.
var nivCibles = []struct {
	ti  int
	nom string
}{
	{40, compObjectParentState}, {40, "unit-actor-control-component"},
	{40, "unit-actor-state-component"}, {40, compForwardUpDynPrec},
	{35, compObjectParentState}, {35, "unit-actor-control-component"},
	{35, "unit-actor-state-component"}, {35, "unit-malleable-property-component"},
	{35, "biped-malleable-property-component"}, {35, "biped-slide-component"},
	{38, compObjectParentState}, {42, compObjectParentState}, {37, compObjectParentState},
}

func TestNiveauxDuRegistre(t *testing.T) {
	films := os.Getenv("NIV_FILMS")
	if films == "" {
		t.Skip("instrument de mesure : NIV_FILMS requis (chemins separes par ;)")
	}
	for _, dir := range strings.Split(films, ";") {
		if dir = strings.TrimSpace(dir); dir == "" {
			continue
		}
		nivUnFilm(t, dir)
	}
}

func nivUnFilm(t *testing.T, dir string) {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	fc := NewFilmContext(film)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre de %s : %v", dir, err)
	}
	var b strings.Builder
	for _, c := range nivCibles {
		a, ok := reg.Archetype(c.ti)
		if !ok {
			continue
		}
		for i, nom := range a.Components {
			if nom != c.nom {
				continue
			}
			if b.Len() > 0 {
				b.WriteString(" | ")
			}
			b.WriteString("ti=" + itoaNiv(c.ti) + " i" + itoaNiv(i) + " " +
				nomCourtNiv(nom) + "=" + itoaNiv(int(a.Level(i))))
		}
	}
	t.Logf("%s : %s", filepath.Base(dir), b.String())
}

func itoaNiv(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

// nomCourtNiv raccourcit le nom pour que la ligne tienne.
func nomCourtNiv(n string) string {
	n = strings.TrimSuffix(n, "-component")
	return strings.TrimPrefix(strings.TrimPrefix(n, "object-"), "unit-")
}
