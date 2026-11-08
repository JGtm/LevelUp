package archlint

// film_racine_depot_unique_test.go — LA BRIQUE NE REMONTE VERS LA RACINE DU DEPOT HOTE QU'A UN
// ENDROIT (regle 6 du depot).
//
// Les tests de la brique qui lisent des donnees du depot hote situent sa racine par
// `libs/film/internal/depottest` (remontee jusqu'au `go.work`, saut hors depot). Sept copies de
// cette remontee (`racineDepot`, une par paquet, et un homonyme qui cherchait `apps/web`) ont ete
// fondues dans ce paquet : ce test refuse qu'une copie renaisse, sous ce nom ou sous un autre.
//
// # CE QU'IL EXIGE
//
// Dans tout `libs/film` (production, tests et instruments `research`), aucun fichier hors de
// `internal/depottest/depottest.go` ne cite `"go.work"` dans son code (commentaires exclus) ni ne
// declare une fonction `racineDepot`.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Ajouter a un `_test.go` de `libs/film/replay` une fonction qui teste
// `os.Stat(filepath.Join(dir, "go.work"))`.

import (
	"strings"
	"testing"
)

// maisonDeLaRacineDuDepot : le seul fichier de la brique qui remonte vers la racine du depot hote.
const maisonDeLaRacineDuDepot = racineBriqueFilm + "/internal/depottest/depottest.go"

func TestLaBriqueRemonteVersLeDepotParUneSeuleAide(t *testing.T) {
	var violations []string
	maison := false
	fichiersGo(t, surDisque(t, racineBriqueFilm), func(rel string, data []byte) {
		code := stripComments(string(data))
		if rel == maisonDeLaRacineDuDepot {
			maison = strings.Contains(code, `"go.work"`)
			return
		}
		if strings.Contains(code, `"go.work"`) {
			violations = append(violations, rel+" : remontee vers go.work")
		}
		if strings.Contains(code, "func racineDepot(") {
			violations = append(violations, rel+" : fonction racineDepot")
		}
	})
	if !maison {
		t.Fatalf("%s ne remonte plus vers go.work : le garde-rail ne garde plus rien", maisonDeLaRacineDuDepot)
	}
	if len(violations) > 0 {
		t.Errorf("remontee vers la racine du depot hors de %s — appeler depottest.Racine(t) :\n  %s",
			maisonDeLaRacineDuDepot, strings.Join(violations, "\n  "))
	}
}
