// Package archlint — no_bare_kill_source_load_test.go : le foyer unique des kills par
// SOURCE DE DEGAT (lot « kills hors arme a feu », 2026-08-29 ; garde-rail pose le 2026-08-30).
//
// LE HELPER EXISTAIT SANS SON GARDE-RAIL, ce que CLAUDE.md regle 6 interdit explicitement :
// « a la 3e copie, centraliser dans un helper ET ajouter un garde-rail — une factorisation
// sans garde-rail re-diverge ». Le lot a bien cree le foyer
// (`internal/service/killsourceload`) et y a ramene les six surfaces, mais rien n'empechait
// la septieme d'appeler le depot en direct. C'est ce fichier.
//
// CE QUI SE PERD QUAND ON APPELLE LE DEPOT EN DIRECT, et ce n'est pas cosmetique :
//
//	la validation      `filters.Validate()` refuse le scan complet de la table partagee.
//	                   Oublie, la requete balaie tous les matchs de tous les joueurs.
//	le nominal avale   `games.ErrCapabilityNotSupported` n'est PAS une panne : c'est un
//	                   match dont le film n'a jamais ete decode. Une copie qui le remonte
//	                   comme une erreur casse la surface sur l'etat ordinaire du parc.
//	la panne criee     toute AUTRE erreur part en ERROR avec sa cause et sa surface. Une
//	                   copie qui l'avale (`_ =`, `continue`) rend un sunburst silencieusement
//	                   moins precis — la mesure a l'air juste, personne ne sait qu'il manque
//	                   des kills.
//
// PROPRIETAIRES AUTORISES — trois, et chacun pour une raison distincte : le port qui
// DECLARE la methode, le depot DuckDB qui l'IMPLEMENTE (et ses tests), et le foyer qui
// l'APPELLE. Ajouter un quatrieme nom ici demande une justification datee : ce serait
// admettre une seconde voie d'acces.
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

// killSourceLoadOwners : les seuls chemins qui ont le droit d'ecrire le nom de la methode.
var killSourceLoadOwners = []string{
	"internal/port/",                   // declaration de l'interface
	"internal/platform/duckdb/",        // implementation + ses tests de bout en bout
	"internal/service/killsourceload/", // LE foyer d'appel
}

// killSourceLoadRE matche un APPEL de la methode, pas sa simple mention : le nom suivi
// d'une parenthese ouvrante. `port.KillSourceClassRepository` cite dans une signature ou
// un champ de structure ne matche donc pas — un service a parfaitement le droit de PORTER
// le depot, il n'a pas le droit de l'appeler lui-meme.
var killSourceLoadRE = regexp.MustCompile(`LoadKillSourceClassesAggregated\s*\(`)

func TestNoBareKillSourceLoad(t *testing.T) {
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
			for _, owner := range killSourceLoadOwners {
				if strings.HasPrefix(rel, owner) {
					return nil
				}
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for i, line := range strings.Split(string(data), "\n") {
				// La prose du dépôt nomme la méthode pour expliquer le contrat ; on ne
				// traque que le code (même traitement que les autres ratchets).
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
					continue
				}
				if killSourceLoadRE.MatchString(line) {
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
		t.Errorf("appel direct au dépôt des kills par source de dégât interdit — passer par "+
			"`killsourceload.Load(ctx, repo, surface, slug, matchIDs, xuids)` "+
			"(`internal/service/killsourceload`, paquet FEUILLE, importable depuis "+
			"`service` comme depuis `service/teammates`). Une copie perd la validation "+
			"anti-scan-complet, requalifie le parc non décodé en panne, ou avale une vraie "+
			"panne — dans les trois cas le sunburst ment sans un signal :\n  %s",
			strings.Join(violations, "\n  "))
	}
}
