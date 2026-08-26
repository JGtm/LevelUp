// Package archlint — no_unbounded_film_loop_test.go : ratchet de L'EXECUTEUR BORNE
// (lot RUNNER, 2026-08-26).
//
// # CE QUE CE RATCHET EMPECHE, ET POURQUOI IL EXISTE
//
// Trois sinistres, le meme mecanisme : une boucle qui enchaine des films a travers la
// construction d'artefact, dans UN SEUL processus. Le 2026-08-20 (`backfill-replay`, six heures
// de spirale GC puis `errno=1450`), le 2026-08-24 (`51101d1d`, 7,9 Go en 2,6 s), et le
// 2026-08-26 — celui-la sur la machine DE TRAVAIL de l'utilisateur, deux fois de suite, la
// seconde alors meme qu'un processus par film avait ete mis en place mais SANS plafond ni
// priorite basse.
//
// La lecon tient en une phrase : **la construction d'artefact est un amplificateur, et tout
// appelant qui l'enchaine doit passer par `internal/filmproc`** — un processus par film, un
// plafond DUR par processus, une priorite CPU basse.
//
// # CE QUE CE RATCHET COMPTE, ET CE QU'IL NE PEUT PAS COMPTER
//
// Il compte les SITES D'APPEL de la construction d'artefact, pas les boucles. Detecter « une
// boucle » statiquement demanderait une analyse de flot que ce garde-fou n'a pas — et qui se
// ferait contourner par une fonction intermediaire. Le choix est donc l'inverse, et il est plus
// sur : **tout site d'appel doit etre DECLARE ici avec sa justification datee**, disant sous
// quel regime il decode. Un site neuf fait rougir le test, et son auteur doit ecrire pourquoi
// il est borne — ce qui est exactement la question qu'on veut lui poser.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// filmBuildCalls : les appels surveilles — les deux portes de la construction d'artefact.
//
// `killsource.Decode` n'y est PAS : il decode le fil des morts, pas l'artefact, et son cout
// n'est pas du meme ordre (cf. l'exemption datee de `killcollector` ci-dessous, qui l'enchaine
// deliberement).
var filmBuildCalls = []string{
	"BuildMatch(",
	"BuildFromFilm(",
}

// filmBuildAllowedCallers : les sites d'appel AUTORISES, chemin relatif a apps/go-api.
//
// TOUTE ENTREE PORTE SON REGIME DE DECODAGE. « un film par processus » veut dire que ce site
// ne decode qu'un film et que le processus meurt ensuite ; « un a la fois » veut dire que le
// processus vit mais ne decode jamais deux films en parallele ET arme sa propre sentinelle.
var filmBuildAllowedCallers = map[string]string{
	"internal/replaybuild/replaybuild.go": "2026-08-26 — la DEFINITION de BuildMatch ; elle " +
		"appelle BuildFromFilm, elle ne boucle sur rien",
	"cmd/levelup/cmd_backfill_replay_child.go": "2026-08-26 — ENFANT de la passe bornee " +
		"(backfill_child.go) : un film par processus, sentinelle armee, le parent compte les issues",
	"cmd/replay-build/main.go": "2026-08-26 — CLI unitaire : un film par invocation, le " +
		"processus meurt ensuite. Aucune boucle",
	"cmd/replay-worker/job.go": "2026-08-26 — ouvrier : un job a la fois, sentinelle armee PAR " +
		"JOB (memlimit.go) et desarmee entre deux. Processus long-vivant, jamais deux films en vol",
	"internal/api/wire/registry_replay_build.go": "2026-08-26 — action admin sur UN match, " +
		"declenchee a la main. Aucune boucle",
	"cmd/zone-attribution/measure.go": "2026-08-26 — ENFANT de la passe bornee (passe.go) : " +
		"un film par processus, plafond de MESURE (2 Gio) et priorite basse via internal/filmproc",
	"internal/analysis/replay/build.go": "2026-08-26 — la DEFINITION de BuildFromFilm ; elle " +
		"ne boucle sur rien",

	// LA SEULE ENTREE QUI N'EST PAS UNE EXEMPTION MAIS UNE DETTE, et elle est ecrite comme
	// telle plutot que maquillee en regime sain.
	//
	// `buildAll` enchaine jusqu'a `maxPerCycle` = 5 films a travers BuildMatch, DANS LE
	// PROCESSUS DU SERVEUR, sans sentinelle memoire. C'est exactement la forme du sinistre du
	// 2026-08-20 — quatre petits films cuits, effondrement sur le CINQUIEME.
	//
	// CE QUI LIMITE LE RISQUE AUJOURD'HUI, et il faut le dire pour ne pas alarmer a tort :
	// l'etape n'existe qu'en environnement NON-PRODUCTION (le wiring ne l'installe pas sur le
	// VPS — « le VPS web ne decode JAMAIS »), elle est best-effort, et le cap de 5 borne le
	// nombre, pas le pic.
	//
	// POURQUOI CE LOT NE LA CORRIGE PAS : l'executeur canonique ne s'y transpose PAS tel quel.
	// Sa sentinelle mene a un arret du processus, ce qui est interdit la ou des handles
	// d'ecriture DuckDB sont tenus (ADR 0013/0019/0030) ; et re-executer le binaire du SERVEUR
	// en enfant n'a pas de sens. Il faut une conception a part — donc un arbitrage, pas une
	// initiative d'executeur. Consigne au registre le 2026-08-26.
	"internal/sync/replayartifacts/artifacts.go": "2026-08-26 — DETTE CONNUE, pas une " +
		"exemption : boucle de 5 films max dans le processus du serveur, sans sentinelle. " +
		"Local uniquement, best-effort. Migration = lot a part (l'arret du processus est " +
		"interdit la ou des handles d'ecriture sont tenus)",
}

func TestNoUnboundedFilmLoop(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile))) // internal/archlint -> internal -> apps/go-api

	var violations []string
	seen := map[string]bool{}
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			// LES COMMENTAIRES NE SONT PAS DES APPELS : plusieurs fichiers CITENT `BuildMatch`
			// pour expliquer leur regime, et les compter ferait rougir le test sur de la prose.
			if t := strings.TrimSpace(line); strings.HasPrefix(t, "//") {
				continue
			}
			for _, call := range filmBuildCalls {
				if !strings.Contains(line, call) {
					continue
				}
				seen[rel] = true
				if _, allowed := filmBuildAllowedCallers[rel]; allowed {
					continue
				}
				violations = append(violations,
					rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk apps/go-api: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("site(s) de construction d'artefact NON DECLARE(S) (%d).\n"+
			"La construction d'artefact est un AMPLIFICATEUR memoire : elle a sature la machine "+
			"trois fois (2026-08-20, 08-24, 08-26).\n"+
			"Si ce site enchaine des films, il DOIT passer par internal/filmproc (un processus "+
			"par film, plafond dur, priorite basse) — cf. cmd/zone-attribution/passe.go.\n"+
			"S'il ne decode qu'un film, declare-le dans filmBuildAllowedCallers avec sa "+
			"justification DATEE :\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}

	// L'ALLOWLIST NE DOIT PAS POURRIR. Une entree qui ne designe plus aucun appel est une
	// permission qui survit a son motif — et la prochaine lecture la prendrait pour une regle.
	for rel := range filmBuildAllowedCallers {
		if !seen[rel] {
			t.Errorf("entree d'allowlist PERIMEE : %q n'appelle plus la construction "+
				"d'artefact — la retirer plutot que la laisser autoriser un site qui n'existe plus", rel)
		}
	}
}
