//go:build integration

package killcollector

// cartes_des_films_integration_test.go — LA VRAIE CARTE DE CHAQUE FILM DE FIXTURE (2026-09-27).
//
// LE DEFAUT FERME. `positions_integration`, `isolation_facts_integration`, `backfill_cout_integration`
// et `collector_ouvriers_integration` cablaient la resolution de carte sur TOUS les noms du
// catalogue (`allCatalogNames`, l iteration d une MAP Go) : la premiere entree qui resolvait
// gagnait, donc chaque film etait decode sous une carte TIREE AU HASARD d un passage a l autre — et
// depuis que la carte decide des largeurs de la marche des morts, sous les largeurs d une autre
// carte. Ils decodent desormais sous la carte du film, lue dans cette table, de facon deterministe.
//
// PROVENANCE, SANS BASE OUVERTE : la colonne `map_name` des extraits de registre COMMIS
// (`.ai/V7.5/replay2d/registre_film/oracle_lotB_overtime.tsv`, `oracle_vague6_registry.tsv`) et
// les references du decodeur (`killsource_test.go`). Un film absent de cette table n a pas de
// carte resolue : le collecteur le met de cote, jamais un decodage au plus proche.

import "testing"

// cartesDesFilmsDeFixture : film (identifiant court) -> nom de carte au catalogue de bornes.
var cartesDesFilmsDeFixture = map[string]string{
	// les references du decodeur
	"000d5950": "Cliffhanger", "9b191a7f": "Bazaar", "78919882": "High Ground", "fccc61cd": "Launch Site",
	"b1ad85eb": "Domicile",
	// le bas du cout (10 a 14 chunks), films CTF / Arena qui ecrivent des morts
	"cf040013": "Fortress", "bf5ced1b": "Illusion", "008e1bba": "Critical Dewpoint", "58864b3c": "Domicile",
	"3685373c": "Fortress", "846044ba": "Behemoth", "a17e61a2": "Dynasty", "e94163af": "Bazaar",
	"b8d1fe0c": "Recharge", "a32ee8d2": "Cliffside",
	// le cout median (29 chunks) et le plus gros film du cache (69 chunks)
	"e624c2a4": "Behemoth", "e85d7bad": "Recharge", "1c4c63c2": "Refuge",
}

// cartesDesFixtures : le resolveur de carte des tests d integration, sur cette table.
func cartesDesFixtures() cartesParMatch {
	m := cartesParMatch{}
	for film, carte := range cartesDesFilmsDeFixture {
		m[film] = []string{carte}
	}
	return m
}

// TestCartesDesFilmsDeFixtureSontAuCatalogue : chaque carte de la table se resout au catalogue
// COMMIS, largeurs comprises — une faute de frappe mettrait le film de cote sans que le test qui
// l emploie le dise.
func TestCartesDesFilmsDeFixtureSontAuCatalogue(t *testing.T) {
	cat := catalogueDuDepot(t)
	for film, carte := range cartesDesFilmsDeFixture {
		e, err := cat.Lookup(carte)
		if err != nil {
			t.Errorf("%s : carte %q hors catalogue : %v", film, carte, err)
			continue
		}
		if e.AxisWidths[0] == 0 || e.AxisWidths[1] == 0 || e.AxisWidths[2] == 0 {
			t.Errorf("%s : carte %q sans largeurs d axe", film, carte)
		}
	}
}
