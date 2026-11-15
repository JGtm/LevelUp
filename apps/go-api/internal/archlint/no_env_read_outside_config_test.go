// Package archlint — no_env_read_outside_config_test.go : l'environnement se lit dans
// `internal/config` et dans `cmd/`, nulle part ailleurs.
//
// POURQUOI. Une variable d'environnement lue par un paquet profond n'apparaît dans aucune
// configuration résolue : elle ne se voit ni dans le profil de déploiement, ni dans les tests de
// configuration, ni dans la documentation, et deux paquets peuvent la lire chacun à leur façon
// (c'est ainsi que `LEVELUP_OAUTH_CLIENT_ID` était lu à deux endroits, avec la même règle recopiée).
// La règle : `os.Getenv` et `os.LookupEnv` n'existent que dans `internal/config` (source unique de
// la configuration) et dans `cmd/` (points d'entrée, hors de ce balayage). Un paquet qui a besoin
// d'une valeur la reçoit de la configuration, ou appelle la fonction de `internal/config` qui la lit.
//
// PORTÉE. Tous les fichiers non-test de `internal/` et `pkg/`. Les commentaires et les chaînes ne
// comptent pas : le garde-rail lit l'arbre syntaxique, il cherche l'appel.
//
// ALLOWLIST. `lecturesEnvAutorisees` est la dette gelée à la date de la règle (2026-11-11) : une
// entrée par fichier, avec la raison pour laquelle la lecture n'a pas rejoint `internal/config`.
// Une entrée ne s'ajoute pas sans justification datée ; elle se retire quand la lecture part. Le
// garde-rail refuse aussi une entrée qui ne lit plus rien (liste périmée).
package archlint

import (
	"go/ast"
	"path"
	"sort"
	"strings"
	"testing"
)

// lectureEnvInterdite dit si un appel `os.Getenv(...)` / `os.LookupEnv(...)` apparaît dans la
// source ; rend le nombre d'appels. nomOS est le nom local du paquet `os` (alias compris), ""
// quand le fichier n'importe pas `os`.
func lectureEnvInterdite(src goSource) int {
	nomOS := ""
	for _, imp := range src.File.Imports {
		if imp.Path.Value != `"os"` {
			continue
		}
		nomOS = "os"
		if imp.Name != nil {
			nomOS = imp.Name.Name
		}
	}
	if nomOS == "" || nomOS == "_" || nomOS == "." {
		return 0
	}
	n := 0
	ast.Inspect(src.File, func(nd ast.Node) bool {
		appel, ok := nd.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := appel.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if ok && id.Name == nomOS && (sel.Sel.Name == "Getenv" || sel.Sel.Name == "LookupEnv") {
			n++
		}
		return true
	})
	return n
}

// lecturesEnvAutorisees : fichiers de `internal/` et `pkg/` qui lisent encore l'environnement
// eux-mêmes, avec la raison. Gelée le 2026-11-11 (plan architecture, étape 2) ; ne doit que
// rétrécir.
var lecturesEnvAutorisees = map[string]string{
	"internal/api/middleware/slog_logger.go":                         "2026-11-11 : seuil de lenteur des requêtes (LEVELUP_SLOW_REQUEST_MS) lu par le middleware de journal ; réglage de diagnostic, pas une donnée de déploiement ; à rapatrier avec l'étape cadre HTTP.",
	"internal/api/wire/registry_replay_notify.go":                    "2026-11-11 : base publique des liens de notification (LEVELUP_PUBLIC_BASE_URL) ; donnée de déploiement lue au câblage ; à porter par AppConfig.",
	"internal/domain/title/repo_root.go":                             "2026-11-11 : racine du dépôt (LEVELUP_REPO_ROOT), résolue avant toute configuration (le chargeur de configuration s'en sert) ; point d'amorçage, ne peut pas passer par AppConfig.",
	"internal/games/halo_infinite/haloadapter/asset_name_fetcher.go": "2026-11-11 : résolution des noms d'assets au sync (LEVELUP_SYNC_RESOLVE_ASSETS) ; réglage de sync lu à l'usage ; à porter par AppConfig.",
	"internal/media/binpath.go":                                      "2026-11-11 : dossier de ffmpeg (EnvFFmpegDir), chemin système résolu avant la configuration ; lecteur unique gardé par media_binpath_single_resolver_test.go.",
	"internal/observability/logging/config.go":                       "2026-11-11 : configuration du journal (LEVELUP_LOGS_*, LEVELUP_LOG_FORMAT, LEVELUP_LOG_MAX_LINE, LEVELUP_LOG_COLOR) ; le paquet est importé par internal/config, il ne peut pas l'importer.",
	"pkg/duckdbbackup/config.go":                                     "2026-11-11 : configuration de sauvegarde (BACKUP_ENABLED, RESTIC_*) du paquet public pkg/duckdbbackup, autonome : il ne doit pas importer internal/config ; secrets restic lus ici uniquement.",
	"internal/platform/auth/azure_credentials.go":                    "2026-11-11 : secret du client Azure (variable SPNKR, voir le garde-rail de ce paquet), lecteur unique gardé par auth/sentinel_test.go ; un secret ne passe pas par AppConfig.",
	"internal/platform/duckdb/db.go":                                 "2026-11-11 : bornes de mémoire, de threads et de pool DuckDB (envOr, envIntOr) lues à l'ouverture de chaque connexion ; le paquet n'importe pas AppConfig (config l'importe).",
	"internal/platform/schedtask/runner.go":                          "2026-11-11 : racine système de Windows (SystemRoot) pour retrouver schtasks.exe ; donnée du système, pas de LevelUp.",
	"internal/platform/syslocale/syslocale_posix.go":                 "2026-11-11 : TZ, LANGUAGE et LC_* du système (fuseau et langue de l'hôte) ; donnée du système, pas de LevelUp.",
	"internal/sync/engine_highlight_events.go":                       "2026-11-11 : fenêtre de reprise des films et reprise des films frais (LEVELUP_FILM_RETRY_WINDOW_HOURS, LEVELUP_FRESH_FILM_RETRY) ; réglages de sync lus à l'usage ; à porter par AppConfig.",
	"internal/sync/snapshot/snapshot_readiness_eval.go":              "2026-11-11 : délai de grâce de l'instantané (LEVELUP_SNAPSHOT_GRACE_HOURS) ; réglage lu à l'usage ; à porter par AppConfig.",
	"internal/testfixtures/external_chunks.go":                       "2026-11-11 : dossiers de films de test hors dépôt (LEVELUP_TEST_FILM_DATA_DIR, LEVELUP_TEST_FILM_MANIFESTS_DIR) ; paquet de fixtures réservé aux tests, jamais lié au serveur.",
}

func TestAucuneLectureEnvHorsConfig(t *testing.T) {
	var violations []string
	vues := map[string]bool{}
	for _, src := range goSourcesUnder(t, "internal", "pkg") {
		if strings.HasPrefix(src.Rel, "internal/config/") {
			continue
		}
		n := lectureEnvInterdite(src)
		if n == 0 {
			continue
		}
		vues[src.Rel] = true
		if _, ok := lecturesEnvAutorisees[src.Rel]; !ok {
			violations = append(violations, src.Rel)
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("%d fichier(s) lisent l'environnement (os.Getenv / os.LookupEnv) hors internal/config :\n  - %s\n"+
			"Lire la valeur par une fonction ou un champ de internal/config. Une entrée de lecturesEnvAutorisees "+
			"exige une raison datée.", len(violations), strings.Join(violations, "\n  - "))
	}
	var perimees []string
	for rel, raison := range lecturesEnvAutorisees {
		if strings.TrimSpace(raison) == "" || !strings.HasPrefix(raison, "2026-") {
			t.Errorf("lecturesEnvAutorisees[%s] : la raison doit commencer par une date (AAAA-MM-JJ) et être non vide", rel)
		}
		if !vues[rel] {
			perimees = append(perimees, path.Clean(rel))
		}
	}
	sort.Strings(perimees)
	if len(perimees) > 0 {
		t.Errorf("entrées de lecturesEnvAutorisees qui ne lisent plus l'environnement (à retirer) :\n  - %s",
			strings.Join(perimees, "\n  - "))
	}
}

// TestAucuneLectureEnvHorsConfig_AutoPreuve : le détecteur voit les deux appels, sous alias
// comme sous le nom courant, et ignore un commentaire, une chaîne et un autre paquet.
func TestAucuneLectureEnvHorsConfig_AutoPreuve(t *testing.T) {
	cas := []struct {
		nom string
		src string
		n   int
	}{
		{"getenv", "package p\nimport \"os\"\nfunc f() string { return os.Getenv(\"X\") }\n", 1},
		{"lookupenv", "package p\nimport \"os\"\nfunc f() { _, _ = os.LookupEnv(\"X\") }\n", 1},
		{"alias", "package p\nimport sys \"os\"\nfunc f() string { return sys.Getenv(\"X\") }\n", 1},
		{"deux", "package p\nimport \"os\"\nfunc f() { _ = os.Getenv(\"A\"); _ = os.Getenv(\"B\") }\n", 2},
		{"commentaire", "package p\nimport \"os\"\n// os.Getenv(\"X\")\nvar s = \"os.Getenv(\\\"X\\\")\"\nvar _ = os.Args\n", 0},
		{"autre paquet", "package p\nimport \"x/os\"\nfunc f() string { return os.Getenv(\"X\") }\n", 0},
		{"sans import", "package p\nfunc f() {}\n", 0},
	}
	for _, c := range cas {
		if got := lectureEnvInterdite(parseGoSnippet(t, "p.go", c.src)); got != c.n {
			t.Errorf("%s : %d appel(s) vus, attendu %d", c.nom, got, c.n)
		}
	}
}
