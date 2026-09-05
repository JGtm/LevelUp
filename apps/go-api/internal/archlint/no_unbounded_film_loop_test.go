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
	"BuildBytes(",
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
	"cmd/levelup/cmd_backfill_replay_child.go": "2026-08-26 — ENFANT de la passe bornee : un " +
		"film par processus, sentinelle armee, le parent compte les issues. REGIME PRECISE LE " +
		"2026-09-03 (PLAN_CUISSON_PERF 5.4/5.7) : le parent est desormais `internal/filmproc` " +
		"(priorite CPU basse comprise, elle manquait), et cet enfant prend le VERROU SOLO EN " +
		"ATTENTE BORNEE (10 min) — une passe n'a pas de cycle suivant, elle attend son tour " +
		"plutot que d'echouer sur un chevauchement",
	"cmd/replay-build/main.go": "2026-08-31 — CLI unitaire : un film par invocation, le " +
		"processus meurt ensuite. ARME LES TROIS PROTECTIONS (verrou solo, priorite basse, " +
		"sentinelle a MeasureLimitGiB) — la justification du 2026-08-26 s'arretait a « aucune " +
		"boucle DANS le processus », ce qui n'a pas empeche un operateur d'en lancer deux en " +
		"parallele et de saturer la machine (cf. internal/filmproc/solo.go)",
	"cmd/replay-worker/job.go": "2026-08-26 — ouvrier : un job a la fois, sentinelle armee PAR " +
		"JOB (filmproc.Arm, processJob) et desarmee entre deux. Processus long-vivant, jamais " +
		"deux films en vol. REGIME COMPLETE LE 2026-09-03 (PLAN_CUISSON_PERF 5.7) : « un film a la fois DANS " +
		"ce processus » ne disait rien des AUTRES processus de la machine (serveur post-sync, " +
		"passe backfill, second ouvrier) — il prend donc le VERROU SOLO EN ATTENTE BORNEE " +
		"(10 min) autour du seul decodage, rendu avant l'envoi",
	"internal/api/wire/registry_replay_build.go": "2026-08-26 — action admin sur UN match, " +
		"declenchee a la main. Aucune boucle",
	"cmd/zone-attribution/measure.go": "2026-08-26 — ENFANT de la passe bornee (passe.go) : " +
		"un film par processus, plafond de MESURE (2 Gio) et priorite basse via internal/filmproc",
	"internal/analysis/replay/build_from_film.go": "2026-08-26 — la DEFINITION de " +
		"BuildFromFilm ; elle ne boucle sur rien. Fichier renomme au lot 1 de PLAN_CUISSON_PERF " +
		"(2026-09-02) : le decodage a quitte `build.go`, qui garde l'assemblage",
	"internal/replaychild/replaychild.go": "2026-08-26 — ENFANT BORNE du post-sync : un film " +
		"par processus, sentinelle armee, aucune base ouverte ; le PARENT range les octets. " +
		"REGIME COMPLETE LE 2026-09-03 (PLAN_CUISSON_PERF 5.7) : `Spawn` (cote parent, dans le " +
		"serveur) prend le VERROU SOLO EN REFUS IMMEDIAT avant de faire naitre l'enfant — le " +
		"post-sync ne doit rien attendre, le match manquant revient au cycle suivant et compte " +
		"en echec de ce cycle-ci. La cuisson est en outre bornee par une deadline " +
		"(min(budget restant, 15 min), item 5.5)",
	"cmd/replay-equiv/child.go": "2026-09-02 — harnais d'equivalence de la cuisson " +
		"(PLAN_CUISSON_PERF D4b) : un film par processus, sentinelle armee, verrou solo en " +
		"attente bornee, aucune base ouverte. Le PARENT (parent.go) lit le corpus et ne decode " +
		"rien — il n'enchaine jamais deux films dans un processus, c'est le motif des quatre " +
		"sinistres RAM que ce ratchet garde",
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

// sentinelleTokens : la marque d'une sentinelle memoire armee dans un paquet.
//
// UNE SEULE FORME LEGITIME DEPUIS LE LOT v2 G.1 (2026-09-05) : `filmproc.Arm`, la sentinelle
// canonique (`internal/filmproc/memguard.go`). Les deux `main` qui portaient chacun leur
// propre copie (`cmd/levelup/backfill_memlimit.go`, `cmd/replay-worker/memlimit.go`)
// l'appelaient DEJA — `filmproc` n'a aucun cout de dependance (pas d'import projet
// non-stdlib) et son callback `onExceeded` porte precisement les deux doctrines d'arret qui
// divergent legitimement (code de sortie enfant vs rapport au serveur). Un `debug.SetMemoryLimit(`
// brut hors de `internal/filmproc` est desormais une TROISIEME copie qui rouvrirait le meme
// defaut : ce ratchet ne l'accepte plus — il doit passer par `filmproc.Arm`.
var sentinelleTokens = []string{"filmproc.Arm("}

// TestPointsDEntreeDeDecodageArmentUneSentinelle — LE RATCHET NE DE LA BOMBE RAM DU 2026-08-31.
//
// # Ce qu'il empeche, et pourquoi le ratchet precedent ne suffisait pas
//
// [TestNoUnboundedFilmLoop] exige qu'un site de construction d'artefact soit DECLARE avec une
// justification. Il ne verifie pas que cette justification soit TENUE. `cmd/replay-build` etait
// declare « CLI unitaire : un film par invocation, le processus meurt ensuite. Aucune boucle » —
// c'etait vrai, et c'etait insuffisant : le binaire n'armait AUCUN plafond memoire, si bien
// qu'une boucle de shell autour de lui (et pire, deux boucles en parallele dont une en
// arriere-plan) a de nouveau sature la machine de travail de l'utilisateur. QUATRIEME sinistre,
// meme mecanisme, quatrieme remede.
//
// **La regle qui en sort : un PROCESSUS D'OPERATEUR qui decode un film arme une sentinelle, quoi
// que dise sa justification.** Une garantie qui s'arrete a la frontiere du processus ne protege
// rien contre celui qui lance le processus.
//
// # Perimetre : les points d'entree `cmd/`, et eux seuls
//
// Les sites `internal/` sont soit des DEFINITIONS (elles ne decodent que si on les appelle),
// soit des chemins qui vivent DANS LE SERVEUR — et la, la sentinelle est INTERDITE : elle mene a
// un arret de processus, et le serveur tient des bases en ecriture (cf. l'en-tete de
// `filmproc`). Leur protection est d'une autre nature (l'enfant borne de `replaychild`), et ce
// ratchet n'a pas a la confondre avec celle-ci.
func TestPointsDEntreeDeDecodageArmentUneSentinelle(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	verifies := 0
	for rel := range filmBuildAllowedCallers {
		if !strings.HasPrefix(rel, "cmd/") {
			continue // cf. l'en-tete : les sites internal/ relevent d'une autre protection
		}
		verifies++
		pkgDir := filepath.Join(apiRoot, filepath.FromSlash(filepath.Dir(rel)))
		if !sentinelleDansPaquet(t, pkgDir) {
			t.Errorf("point d'entree %q decode un film SANS sentinelle memoire.\n"+
				"Un processus d'operateur qui decode doit armer filmproc.Arm AVANT tout "+
				"decodage : « un film par invocation » ne dit rien du nombre d'invocations, "+
				"et c'est ce qui a sature la machine le 2026-08-31. "+
				"Modele : cmd/replay-build/main.go.", rel)
		}
	}
	if verifies == 0 {
		t.Fatal("aucun point d'entree cmd/ verifie — le ratchet ne prouverait rien")
	}
	t.Logf("%d point(s) d'entree cmd/ verifie(s)", verifies)
}

// sentinelleDansPaquet dit si un repertoire de paquet arme une sentinelle memoire.
func sentinelleDansPaquet(t *testing.T, dir string) bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Errorf("lecture du paquet %s: %v", dir, err)
		return false
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name())) //nolint:gosec // chemin construit depuis l'allowlist
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "//") {
				continue // une sentinelle CITEE en commentaire n'en est pas une
			}
			for _, tok := range sentinelleTokens {
				if strings.Contains(line, tok) {
					return true
				}
			}
		}
	}
	return false
}
