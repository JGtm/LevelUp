package archlint

// film_no_slog_in_layers_test.go — `grammar` ET `facts` NE JOURNALISENT PAS (ADR 0034 D-4, J12.3 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25).
//
// # LA REGLE
//
// Les couches de lecture rendent des valeurs et des DIAGNOSTICS types (`film/internal/constat`) ;
// seul l orchestrateur (`film/replay`, `replaybuild`, `sync/killcollector`) journalise, avec son
// contexte (`replay.JournaliserDiagnostics`). Un fichier de production de
// `film/internal/grammar/...` ou `film/internal/facts/...` n importe donc ni `log/slog` ni `log`.
//
// Le test lit les IMPORTS (go/parser) : un commentaire qui cite `slog.Warn` ne le fait pas rougir,
// un journal reel ne peut pas s ecrire sans l import.
//
// # PERIMETRE
//
// `internal/games/halo_infinite/film/internal/grammar/` et `.../internal/facts/`, sous-paquets
// compris. Fichiers non-test, hors `//go:build research` (les instruments journalisent a leur gre).
//
// # ALLOWLIST
//
// AUCUNE, et elle doit le rester.
//
// # LA MUTATION QUI DOIT ROUGIR
//
// Ajouter `import "log/slog"` et un `slog.WarnContext(ctx, "x")` dans un fichier non-test de
// `grammar` : le test nomme le fichier.

import (
	"sort"
	"strconv"
	"strings"
	"testing"
)

// importsDeJournal : les paquets de journal que les couches de lecture n importent pas.
var importsDeJournal = map[string]bool{"log/slog": true, "log": true}

func TestGrammarEtFactsNeJournalisentPas(t *testing.T) {
	var fautes []string
	fichiers := fichiersDuPerimetreFilm(t,
		"internal/games/halo_infinite/film/internal/grammar",
		"internal/games/halo_infinite/film/internal/facts",
	)
	for _, f := range fichiers {
		for _, imp := range f.ast.Imports {
			chemin, _ := strconv.Unquote(imp.Path.Value)
			if importsDeJournal[chemin] {
				fautes = append(fautes, f.rel+": importe "+chemin)
			}
		}
	}
	if len(fautes) > 0 {
		sort.Strings(fautes)
		t.Fatalf("journal dans une couche de lecture (ADR 0034 D-4) :\n  %s\n\n"+
			"Rendre un diagnostic type (`constat.Diagnostic`, note dans les diagnostics que l appelant "+
			"a ouverts) ; l orchestrateur le journalise avec son contexte.", strings.Join(fautes, "\n  "))
	}
}
