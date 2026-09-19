//go:build research

package grammar

// niveaux_registre_research_test.go — LOT 5.1.7 : LE NIVEAU QUE CHAQUE FILM ECRIT, CONFRONTE A
// LA CONSTANTE DE L EXECUTABLE.
//
// # L HYPOTHESE QU IL TRANCHE
//
// `param_4` a DEUX sources, et le desassemblage des deux boucles le dit sans ambiguite :
//
//	FUN_142e2c690 (image-cle)  arg5 = `entree + 0x100` = le NIVEAU du registre du film
//	FUN_14076cb60 (delta)      arg5 = `vtable[0]()` du descripteur = une CONSTANTE de l executable
//
// HYPOTHESE DU MIROIR : le niveau du registre est le `vtable[0]` du build ENREGISTREUR, recopie
// dans le film au moment de l enregistrement. Si elle tient, le film porte la constante de son
// propre build, il est AUTOPORTANT, et le registre est la source sur LES DEUX chemins — la table
// de l executable courant n etant qu un miroir du build courant.
//
// CRITERE, ECRIT AVANT LA MESURE : sur les films du BUILD COURANT, `arch.Level(i)` doit valoir la
// constante de l executable pour les TREIZE composants calibres, sans exception. Un seul ecart
// sur un build courant FALSIFIE le miroir — et alors les deux chemins ont bien deux sources.
//
// LES TREIZE CONSTANTES SONT LUES CHEZ L ECRIVAIN, pas recopiees d une table : `vtable[0]` de
// chaque descripteur est une fonction de six octets (`mov eax,K ; ret`, ou `xor eax,eax ; ret`
// pour le zero), resolue par la chaine nom -> accesseur -> slot unique de `.rdata`, vtable =
// slot - 0x8 (la boucle image-cle lit le nom en `[vtable+0x8]`). Calibration 13/13 le 2026-09-18.
//
// LECTURE SEULE, sans decodage de paquet (chunk 0 seulement) :
//
//	NIV_FILMS='<abs>/film_chunks/084a804d;<abs>/film_chunks/a349fea8' \
//	  go test -tags=research ./internal/games/halo_infinite/film/internal/grammar/ \
//	  -run '^TestNiveauxDuRegistre$' -v

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// nivConstanteExe : `vtable[0]` du descripteur, LU dans `HaloInfinite.exe` le 2026-09-18.
// La cle est le nom de composant ; la valeur, la constante que la fonction rend.
var nivConstanteExe = map[string]uint32{
	compForwardUpDynPrec:                   2, // 0x141179610 mov eax,2
	compObjectParentState:                  3, // 0x14117e0e0 mov eax,3
	"unit-actor-control-component":         2, // 0x141179610
	"unit-actor-state-component":           4, // 0x140c85020 mov eax,4
	"unit-malleable-property-component":    4, // 0x140c85020
	"biped-malleable-property-component":   2, // 0x141179610
	"biped-slide-component":                1, // 0x14117b4a0 mov eax,1
	"object-maximum-vitalities-component":  3, // 0x14117e0e0
	"object-low-frequency-component":       2, // 0x141179610
	"object-frame-configuration-component": 0, // 0x1405f0ac0 xor eax,eax
	"unit-control-component":               2, // 0x141179610
	compNavpointDistanceFilters:            3, // 0x14117e0e0
	compNavpointOffscreenFilters:           2, // 0x141179610
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

// nivUnFilm confronte le registre d UN film aux treize constantes de l executable.
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
	p := fc.Profile()
	vus, accords := 0, 0
	var ecarts []string
	for _, a := range reg.Archetypes {
		for i, nom := range a.Components {
			k, ok := nivConstanteExe[nom]
			if !ok {
				continue
			}
			vus++
			lu := a.Level(i)
			if lu == k {
				accords++
				continue
			}
			ecarts = append(ecarts, fmt.Sprintf("ti=%d i%d %s : registre=%d exe=%d",
				a.Index, i, nom, lu, k))
		}
	}
	sort.Strings(ecarts)
	verdict := "MIROIR"
	if len(ecarts) > 0 {
		verdict = "ECART"
	}
	t.Logf("%-10s format=%d build=%q : %s %d/%d %s", filepath.Base(dir), p.FormatVersion(),
		p.Build(), verdict, accords, vus, strings.Join(ecarts, " · "))
}
