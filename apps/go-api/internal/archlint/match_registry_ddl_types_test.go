package archlint

// match_registry_ddl_types_test.go — LES DEUX DDL DE match_registry DISENT LE MÊME TYPE.
//
// POURQUOI (2026-09-16). `match_registry` est déclarée à DEUX endroits : la migration de
// création title-owned (`internal/games/halo_infinite/migrations/steps_shared_core.go`) et le
// schéma de secours du moteur (`internal/sync/schema.go`). Elles ont divergé sans que rien ne
// le voie : la première annonçait `team_{0,1}_score INTEGER`, la seconde SMALLINT — et les
// bases réelles, créées par la seconde forme, rejetaient à l'INSERT tout match dont un score
// d'équipe dépasse 32 767 (Baptême du feu). Deux matchs perdus POUR TOUS LES JOUEURS.
//
// LA RÈGLE. Pour chaque colonne déclarée DANS LES DEUX DDL, le type doit être identique. Les
// colonnes propres à une seule DDL ne sont pas comparées (les deux schémas n'ont jamais eu la
// même surface : `playlist_name_fr` d'un côté, `season_id` de l'autre) — mais un plancher de
// colonnes communes garde le parseur honnête : s'il ne trouve plus rien, c'est lui qui est
// cassé, pas le code.
//
// Mutation qui doit le faire rougir : remettre `team_0_score SMALLINT` dans l'une des deux.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ddlMatchRegistry : les deux sources à comparer (chemin relatif à apps/go-api).
var ddlMatchRegistry = []string{
	"internal/games/halo_infinite/migrations/steps_shared_core.go",
	"internal/sync/schema.go",
}

// colonnesCommunesPlancher : mesuré le 2026-09-16 (26 colonnes communes). Un parseur qui rend
// moins ne compare plus rien d'utile.
const colonnesCommunesPlancher = 20

// reDebutMatchRegistry repère l'ouverture de la DDL, quelle que soit la casse et l'indentation.
var reDebutMatchRegistry = regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(IF\s+NOT\s+EXISTS\s+)?match_registry\s*\(`)

// reColonne : `nom TYPE ...` — le type est le deuxième mot, sans sa longueur éventuelle.
var reColonne = regexp.MustCompile(`^([a-z_][a-z0-9_]*)\s+([A-Za-z]+)`)

// typesMatchRegistry parse la DDL de match_registry d'un fichier et rend colonne -> type.
func typesMatchRegistry(t *testing.T, chemin string) map[string]string {
	t.Helper()
	brut, err := os.ReadFile(chemin)
	if err != nil {
		t.Fatalf("lecture de %s : %v", chemin, err)
	}
	texte := string(brut)
	pos := reDebutMatchRegistry.FindStringIndex(texte)
	if pos == nil {
		t.Fatalf("%s : aucune DDL match_registry trouvée — le parseur ou le fichier a changé", chemin)
	}
	corps := texte[pos[1]:]
	types := map[string]string{}
	profondeur := 1
	for _, ligne := range strings.Split(corps, "\n") {
		nette := strings.TrimSpace(ligne)
		profondeur += strings.Count(nette, "(") - strings.Count(nette, ")")
		if profondeur <= 0 {
			break
		}
		if nette == "" || strings.HasPrefix(nette, "--") || strings.HasPrefix(nette, "//") {
			continue
		}
		// Contraintes de table (PRIMARY KEY (...), UNIQUE (...)) : pas des colonnes.
		majuscule := strings.ToUpper(nette)
		if strings.HasPrefix(majuscule, "PRIMARY KEY") || strings.HasPrefix(majuscule, "UNIQUE") ||
			strings.HasPrefix(majuscule, "FOREIGN KEY") || strings.HasPrefix(majuscule, "CONSTRAINT") {
			continue
		}
		m := reColonne.FindStringSubmatch(nette)
		if m == nil {
			continue
		}
		types[m[1]] = strings.ToUpper(m[2])
	}
	if len(types) == 0 {
		t.Fatalf("%s : DDL match_registry parsée sans aucune colonne", chemin)
	}
	return types
}

// TestMatchRegistryDDLTypesIdentiques — LE RATCHET.
func TestMatchRegistryDDLTypesIdentiques(t *testing.T) {
	racine := racineGoAPI(t)
	migrationTypes := typesMatchRegistry(t, filepath.Join(racine, ddlMatchRegistry[0]))
	schemaTypes := typesMatchRegistry(t, filepath.Join(racine, ddlMatchRegistry[1]))

	var communes []string
	for colonne := range migrationTypes {
		if _, ok := schemaTypes[colonne]; ok {
			communes = append(communes, colonne)
		}
	}
	sort.Strings(communes)
	if len(communes) < colonnesCommunesPlancher {
		t.Fatalf("%d colonne(s) commune(s) aux deux DDL, plancher %d (mesure du 2026-09-16) — "+
			"le parseur ne compare plus rien", len(communes), colonnesCommunesPlancher)
	}

	for _, colonne := range communes {
		if migrationTypes[colonne] != schemaTypes[colonne] {
			t.Errorf("match_registry.%s : %s dans %s, %s dans %s — les deux DDL doivent dire le "+
				"MÊME type ; une divergence ne se voit qu'au premier INSERT rejeté, et le match "+
				"est alors perdu pour TOUS les joueurs (plan 2026-09-16, étape 3)",
				colonne, migrationTypes[colonne], ddlMatchRegistry[0],
				schemaTypes[colonne], ddlMatchRegistry[1])
		}
	}
}
