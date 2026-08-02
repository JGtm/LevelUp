// Package archlint — no_raw_kill_scope_literal_test.go : ratchet J4R-3.
//
// Interdit tout littéral brut de PORTÉE de `shared.match_kill_events` (`"kill-feed"`,
// `"credit-seul"`, `"highlight-events"` en position de valeur `read_path` / `read_origin`) hors
// de son propriétaire `internal/domain/killscope`.
//
// POURQUOI CE TEST EXISTE — il répare une promesse non tenue. Le commentaire de
// `migration/steps_shared_kill_events_from_pairs.go` annonçait que sa copie privée des deux
// valeurs était « verrouillée par un test d'égalité — cf.
// steps_shared_kill_events_from_pairs_test.go ». Ce fichier n'a jamais existé (constat J4R-3,
// relevé par DEUX relecteurs indépendants). La copie était donc libre de dériver.
//
// CE QUE LA DÉRIVE COÛTERAIT, ET POURQUOI ELLE SERAIT MUETTE : c'est sur `read_path` que se
// décide la PRÉSÉANCE entre producteurs (le film gagne, toujours). Une copie qui diverge d'un
// caractère ne lève aucune erreur — le filtre de préséance cesse simplement de reconnaître les
// passes de l'autre écrivain, et un producteur crédit réécrit par-dessus une passe de film. Le
// nombre de lignes ne bouge pas, aucun nom ne change : seule la source du dégât fatal disparaît
// de la lecture.
//
// Ce ratchet couvre AUSSI le retour en arrière que `killcollector.covertParUnFilm` documente
// (J4R-4-bis) : la préséance se testait autrefois EN NÉGATIF (`read_path <> 'highlight-events'`),
// donc toute voie autre que la sienne passait pour un film. Écrire une telle comparaison exige
// un littéral — que ce test refuse.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// killScopeAllowlist : VIDE, et elle doit le rester.
//
// Contrairement aux ratchets de dette (allowlist décroissante), il n'y a rien à migrer ici : les
// quatre écrivains lisent déjà `domain/killscope`. Une entrée ajoutée ici serait une seconde
// source de vérité pour une valeur qui décide de la préséance — la justifier par écrit ET par
// une date, ou ne pas l'ajouter.
var killScopeAllowlist = map[string]bool{}

// killScopeRE matche les trois valeurs de portée en littéral Go OU SQL (simples quotes).
//
// Les trois chaînes sont distinctives : elles n'apparaissent pas dans du texte courant. La prose
// française du dépôt écrit « kill-feed » sans guillemets droits (commentaires) — non matchée,
// puisque les lignes de commentaire sont sautées avant le test.
var killScopeRE = regexp.MustCompile(`["']((kill-feed)|(credit-seul)|(highlight-events))["']`)

func TestNoRawKillScopeLiteral(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile)) // .../internal
	apiRoot := filepath.Dir(internalRoot)                // .../apps/go-api

	var violations []string
	for _, racine := range []string{internalRoot, filepath.Join(apiRoot, "cmd")} {
		err := filepath.WalkDir(racine, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			rel, _ := filepath.Rel(apiRoot, path)
			rel = filepath.ToSlash(rel)
			// Le propriétaire des valeurs, et lui seul, a le droit de les écrire.
			if strings.HasPrefix(rel, "internal/domain/killscope/") {
				return nil
			}
			if killScopeAllowlist[rel] {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				// `--` : les DDL du dépôt sont des chaînes brutes Go dont les commentaires SQL
				// citent les valeurs pour documenter la colonne. C'est de la prose, pas une
				// seconde source de vérité — même traitement que `//`.
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") ||
					strings.HasPrefix(trimmed, "--") {
					continue
				}
				if killScopeRE.MatchString(line) {
					violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+trimmed)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", racine, err)
		}
	}
	if len(violations) > 0 {
		t.Errorf("littéral brut de portée `match_kill_events` interdit (J4R-3) — utiliser "+
			"`killscope.ReadPathLiveFeed` / `killscope.ReadPathCreditBackfill` / "+
			"`killscope.OriginCreditOnly` (paquet `internal/domain/killscope`, feuille sans "+
			"import : importable depuis persist, migration, killcollector et ops sans cycle). "+
			"Une copie qui dérive d'un caractère rend la préséance film aveugle, SANS erreur "+
			"ni compteur :\n  %s", strings.Join(violations, "\n  "))
	}
}
