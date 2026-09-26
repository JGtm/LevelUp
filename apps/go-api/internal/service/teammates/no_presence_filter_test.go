package teammates

// no_presence_filter_test.go — garde-rail (ratchet) de l'ADR 0033, volet SQL.
//
// CE QU'IL INTERDIT. Toute colonne de PRESENCE dans les requetes qui construisent la
// population escouade. Quitter un match n'est pas quitter la session : un plantage de jeu
// ou de PC ne doit jamais retirer un match de la session d'une composition. Le pendant Go
// (types sans champ de presence, scenario du crash) est `composition_presence_test.go`.
//
// POURQUOI UN GREP ET PAS UN TEST DE COMPORTEMENT. Le comportement est deja verrouille par
// le scenario du crash ; ce qu'un test de comportement ne peut PAS attraper, c'est une
// requete ajoutee demain a la meme couche avec un `AND p.present_at_completion` « pour ne
// compter que les matchs finis ensemble » — elle passerait tous les tests existants et
// rejouerait exactement le defaut rapporte deux fois.
//
// ALLOWLIST : VIDE au 2026-09-09, et elle doit le rester. Une entree exige une
// justification datee ET une relecture de l'ADR 0033 : il n'existe aucun cas connu ou la
// presence doit filtrer une population escouade. Compter les departs pour les AFFICHER est
// un autre sujet, et cela ne passe pas par ces fichiers.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// presenceColumns : les colonnes de `match_participants` qui disent la presence.
var presenceColumns = []string{
	"present_at_beginning",
	"present_at_completion",
	"joined_in_progress",
	"left_in_progress",
	"last_leave_time",
}

// squadPopulationSources : les fichiers qui produisent la population escouade. Chemins
// relatifs a la racine du module Go (apps/go-api).
var squadPopulationSources = []string{
	"internal/platform/duckdb/queries_squad.go",
	"internal/platform/duckdb/squad_repo.go",
	"internal/platform/duckdb/squad_repo_synthesis.go",
	"internal/platform/duckdb/squad_repo_mapstats.go",
}

// presenceAllowlist : fichiers autorises a mentionner une colonne de presence, avec la
// date et le motif. VIDE — cf. en-tete.
var presenceAllowlist = map[string]string{}

func TestNoPresenceFilterInSquadPopulation(t *testing.T) {
	root := goAPIRoot(t)
	scanned := 0
	for _, rel := range squadPopulationSources {
		path := filepath.Join(root, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			// Un fichier renomme doit casser le ratchet, pas le desarmer en silence.
			t.Fatalf("source de population escouade illisible (%s) : %v — si le fichier a "+
				"ete renomme, mettre a jour squadPopulationSources", rel, err)
		}
		scanned++
		if motif, ok := presenceAllowlist[rel]; ok {
			t.Logf("%s : allowliste — %s", rel, motif)
			continue
		}
		src := strings.ToLower(string(data))
		for _, col := range presenceColumns {
			if strings.Contains(src, col) {
				t.Errorf("%s mentionne %q. La population escouade ne filtre JAMAIS sur la "+
					"presence : quitter un match n'est pas quitter la session (ADR 0033).",
					rel, col)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("aucune source scannee — le ratchet ne verifie rien")
	}
}

// goAPIRoot remonte de ce paquet (internal/service/teammates) a la racine du module.
func goAPIRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(packageDir(t), "..", "..", "..")
}
