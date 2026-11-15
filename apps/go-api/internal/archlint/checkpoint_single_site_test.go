// Package archlint — checkpoint_single_site_test.go : garde-rail « le CHECKPOINT d'une base DuckDB
// s'écrit dans le paquet qui le centralise ».
//
// POURQUOI. Un CHECKPOINT lancé n'importe où l'est sans le bail d'écriture de la base, sans
// journal ni métrique, et sans savoir si un lot d'écriture est en cours : le CHECKPOINT périodique
// du profil desktop (duckdb.CheckpointHeld) prend le bail, compte ses échecs et se journalise. Le
// paquet central est internal/platform/duckdb.
//
// INTERDIT (hors tests) : un littéral de chaîne Go (guillemets ou backquote) qui contient une
// instruction CHECKPOINT, dans un fichier de cmd/, internal/, pkg/ ou scripts/ qui n'est pas dans
// internal/platform/duckdb/, sauf entrée de l'allowlist. Formes reconnues (voir isCheckpointSQL) :
// « CHECKPOINT », « FORCE CHECKPOINT », « CHECKPOINT nom », une instruction après « ; » dans un script
// (« CREATE TABLE ...; CHECKPOINT; »), « CALL checkpoint() », « force_checkpoint », et en minuscules
// quand un point-virgule en fait une instruction. Seuls les littéraux SQL comptent : un commentaire,
// un identifiant Go (CheckpointHeld), un message (« CHECKPOINT: %v »), un nom de drapeau ou de tâche
// (« checkpoint », « checkpoint shared_social ») ne sont pas des instructions. Limite assumée : un
// « checkpoint nom » tout en minuscules, sans point-virgule, se confondrait avec un nom de tâche.
//
// L'allowlist est le décompte EXACT des sites existants, fichier par fichier, avec la raison de sa
// présence : un fichier qui en ajoute un dépasse son décompte et échoue ; un fichier qui n'en a plus
// (ou moins) échoue aussi, l'entrée devenue inutile se baisse ou se retire. Elle ne fait que baisser :
// un site migré vers le paquet central retire sa ligne. Une entrée nouvelle exige une date et une
// justification.
package archlint

import (
	"go/ast"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// checkpointCentralDir : le paquet qui centralise les CHECKPOINT.
const checkpointCentralDir = "internal/platform/duckdb/"

var checkpointRoots = []string{"cmd", "internal", "pkg", "scripts"}

// Formes SQL d'un CHECKPOINT dans un littéral. Une instruction se reconnaît à sa POSITION (début du
// littéral, d'une ligne ou après un point-virgule) et à sa forme, pas à la seule présence du mot : un
// message (« CHECKPOINT: %v », « vidé + CHECKPOINT OK »), le nom d'un drapeau (« checkpoint ») ou
// d'une tâche (« checkpoint shared_social ») n'est pas une instruction.
var (
	// En majuscules (la forme SQL du dépôt) : « CHECKPOINT », « FORCE CHECKPOINT; », « CHECKPOINT nom »,
	// « INSERT ...; CHECKPOINT » (le littéral, ou une ligne, ou une instruction après « ; », s'arrête là).
	checkpointUpper = regexp.MustCompile(`(?m)(?:^|;)\s*(?:FORCE\s+)?CHECKPOINT(?:\s+[A-Za-z_][A-Za-z0-9_]*)?\s*(?:;|$)`)
	// Toute casse, mais avec un point-virgule qui fait de lui une instruction : « checkpoint; »,
	// « CREATE TABLE t (id INT); checkpoint ».
	checkpointTerminated = regexp.MustCompile(`(?i)(?:(?:^|;)\s*(?:force\s+)?checkpoint\s*;|;\s*(?:force\s+)?checkpoint\s*$)`)
	// « CALL checkpoint() », « CALL force_checkpoint('nom') », « PRAGMA force_checkpoint ».
	checkpointCall = regexp.MustCompile(`(?i)\b(?:force_checkpoint|checkpoint\()`)
)

// isCheckpointSQL dit si le littéral est (ou contient) une instruction CHECKPOINT.
func isCheckpointSQL(v string) bool {
	return checkpointUpper.MatchString(v) || checkpointTerminated.MatchString(v) || checkpointCall.MatchString(v)
}

// checkpointAllowance : le décompte exact des CHECKPOINT d'un fichier et la raison de leur présence.
type checkpointAllowance struct {
	count int
	why   string // raison datée : sans elle, l'entrée n'a pas lieu d'être
}

// checkpointAllowlist : fichier -> CHECKPOINT tolérés. Recompte réel au 2026-11-02 (lot L20, revue
// adversariale : l'ancienne détection ne voyait que le littéral « CHECKPOINT » seul et comptait à tort
// des noms de drapeau). Exact : un fichier qui en a plus échoue, un fichier qui en a moins aussi.
var checkpointAllowlist = map[string]checkpointAllowance{
	"cmd/levelup/cmd_backfill_h5_kill_mechanics.go":                             {1, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/levelup/cmd_compact_passes.go":                                         {1, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/levelup/cmd_compact_passes_rewrite.go":                                 {1, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/levelup/cmd_consolidate_aliases.go":                                    {1, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/levelup/cmd_rebuild_pme.go":                                            {2, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/levelup/cmd_repair_highlight_phantoms.go":                              {1, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/rebuild_shared_social/main.go":                                         {1, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/recompute_perfnote/main.go":                                            {1, "outil en ligne de commande qui ouvre sa base seul, serveur arrêté : aucun bail concurrent à respecter (2026-11-02)"},
	"cmd/server/background_tasks.go":                                            {1, "CHECKPOINT périodique de shared_social (checkpointSharedSocial, ADR 0022) : handle du pool process-wide, la tâche sociale est la seule à les traiter (2026-11-02)"},
	"cmd/server/shutdown.go":                                                    {1, "CHECKPOINT final de shared_social à l'arrêt, workers arrêtés (ADR 0021, 2026-11-02)"},
	"cmd/wal_forensic_compare/main.go":                                          {5, "outil forensique hors serveur : fabrique volontairement des WAL non checkpointés dans des bases jetables (ADR 0021 phase 3.3, 2026-11-02)"},
	"internal/games/halo_infinite/migrations/steps_shared_drop_weapon_kills.go": {1, "script de migration de boot : le CHECKPOINT referme la suppression des tables (2026-11-02)"},
	"internal/migration/compaction.go":                                          {1, "CompactSupersededPasses : compaction lancée par la commande levelup compact-passes, base hors service (2026-11-02)"},
	"internal/migration/declared_keys_repair.go":                                {1, "repairDeclaredKeys : réparation de clés déclarées dans le même passage de migration de boot, avant l'ouverture au service (2026-11-02)"},
	"internal/migration/registry.go":                                            {1, "checkpointPostMigration : CHECKPOINT après les migrations de boot, avant l'ouverture au service (2026-11-02)"},
	"internal/ops/media.go":                                                     {1, "IndexMedia : réindexation lancée par la commande levelup index-media (2026-11-02)"},
	"internal/ops/media_capture_recalc.go":                                      {2, "recalcul des heures de capture lancé par la commande levelup (2026-11-02)"},
	"internal/ops/media_hls.go":                                                 {1, "checkpointBestEffort : flush immédiat d'une écriture d'état de transcodage, journalisé s'il échoue (2026-11-02)"},
	"internal/ops/media_store.go":                                               {1, "checkpointMediaSchemaChange : CHECKPOINT après un changement de schéma de la base média (2026-11-02)"},
	"internal/ops/seed_demo_prestige_social.go":                                 {1, "génération de la base démo : sans CHECKPOINT le WAL de shared_social peut être perdu (ADR 0022, 2026-11-02)"},
	"internal/persist/shared_social_persister.go":                               {3, "CommitWithCheckpoint : écritures sociales critiques à flush immédiat (ADR 0022, 2026-11-02)"},
	"internal/platform/dblease/writer.go":                                       {1, "LeasedWriter.CommitWithCheckpoint : CHECKPOINT sous le bail que ce paquet vient de prendre (ADR 0013, 2026-11-02)"},
	"scripts/repair_tracks_table/main.go":                                       {2, "script de réparation ponctuel, base hors service (2026-11-02)"},
}

// checkpointSites rend « fichier:ligne » de chaque instruction CHECKPOINT d'un fichier.
func checkpointSites(src goSource) []string {
	var sites []string
	ast.Inspect(src.File, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok {
			return true
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil || !isCheckpointSQL(v) {
			return true
		}
		sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(lit.Pos()).Line))
		return true
	})
	return sites
}

// checkpointProblems compare les sites de sources à allowlist (excès et entrées devenues inutiles).
func checkpointProblems(sources []goSource, allowlist map[string]checkpointAllowance) []string {
	perFile := map[string][]string{}
	for _, src := range sources {
		if strings.HasPrefix(src.Rel, checkpointCentralDir) {
			continue
		}
		if sites := checkpointSites(src); len(sites) > 0 {
			perFile[src.Rel] = sites
		}
	}
	var out []string
	for file, sites := range perFile {
		if allowed := allowlist[file].count; len(sites) > allowed {
			out = append(out, file+" : "+strconv.Itoa(len(sites))+" CHECKPOINT (toléré : "+strconv.Itoa(allowed)+") "+strings.Join(sites, ", "))
		}
	}
	for file, allowance := range allowlist {
		if allowed := allowance.count; len(perFile[file]) < allowed {
			out = append(out, file+" : l'allowlist tolère "+strconv.Itoa(allowed)+" CHECKPOINT, il n'en reste que "+
				strconv.Itoa(len(perFile[file]))+" : baisser ou retirer la ligne")
		}
	}
	sort.Strings(out)
	return out
}

func TestCheckpointOnlyInTheCentralPackage(t *testing.T) {
	problems := checkpointProblems(goSourcesUnder(t, checkpointRoots...), checkpointAllowlist)
	if len(problems) > 0 {
		t.Errorf("CHECKPOINT hors de %s (le centraliser : duckdb.CheckpointHeld ou un helper de ce paquet) :\n  %s",
			checkpointCentralDir, strings.Join(problems, "\n  "))
	}
}

// TestCheckpointAllowlistEntriesAreJustified : chaque entrée porte un compte positif et une raison
// datée.
func TestCheckpointAllowlistEntriesAreJustified(t *testing.T) {
	for file, a := range checkpointAllowlist {
		if a.count <= 0 || !strings.Contains(a.why, "2026-") {
			t.Errorf("%s : entrée sans compte positif ou sans raison datée (%+v)", file, a)
		}
	}
}

func TestCheckpointGuardDetectsDefects(t *testing.T) {
	cases := []struct {
		name, rel, src string
		want           int
	}{
		{"instruction hors du paquet central", "cmd/server/x.go", "package main\nvar q = \"CHECKPOINT\"", 1},
		{"minuscules et point-virgule", "internal/ops/x.go", "package ops\nvar q = `checkpoint;`", 1},
		{"FORCE CHECKPOINT", "internal/ops/x.go", "package ops\nvar q = \"FORCE CHECKPOINT\"", 1},
		{"CREATE puis CHECKPOINT dans le même littéral", "cmd/x/main.go", "package main\nvar q = `CREATE TABLE t (id INTEGER); CHECKPOINT;`", 1},
		{"INSERT puis CHECKPOINT sans point-virgule final", "cmd/x/main.go", "package main\nvar q = \"INSERT INTO t VALUES (1); CHECKPOINT\"", 1},
		{"CHECKPOINT d'une base nommée", "cmd/x/main.go", "package main\nvar q = \"CHECKPOINT ma_base\"", 1},
		{"CALL checkpoint()", "internal/ops/x.go", "package ops\nvar q = \"CALL checkpoint()\"", 1},
		{"force_checkpoint", "internal/ops/x.go", "package ops\nvar q = \"PRAGMA force_checkpoint\"", 1},
		{"script sur plusieurs lignes", "internal/ops/x.go", "package ops\nvar q = `\n\t\tDROP TABLE t;\n\t\tCHECKPOINT;\n`", 1},
		{"nom de tâche : rien à signaler", "cmd/server/x.go", "package main\nvar n = \"checkpoint shared_social\"", 0},
		{"nom de drapeau : rien à signaler", "cmd/x/main.go", "package main\nvar n = \"checkpoint\"", 0},
		{"message de journal : rien à signaler", "internal/ops/x.go", "package ops\nvar m = \"CHECKPOINT final au shutdown échoué\"", 0},
		{"message d'erreur : rien à signaler", "internal/ops/x.go", "package ops\nvar m = \"CHECKPOINT: %v\"", 0},
		{"commentaire : rien à signaler", "internal/ops/x.go", "package ops\n// CHECKPOINT; checkpoint()\nvar a = 1", 0},
		{"nom de fonction Go : rien à signaler", "internal/ops/x.go", "package ops\nfunc CheckpointHeld() {}\nvar _ = CheckpointHeld", 0},
		{"paquet central : toléré", checkpointCentralDir + "x.go", "package duckdb\nvar q = \"CHECKPOINT\"", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := parseGoSnippet(t, tc.rel, tc.src)
			got := 0
			if !strings.HasPrefix(src.Rel, checkpointCentralDir) {
				got = len(checkpointSites(src))
			}
			if got != tc.want {
				t.Fatalf("sites = %d, attendu %d", got, tc.want)
			}
		})
	}
}

// TestCheckpointProblems_UnusedEntryAndExcessFail : un fichier de l'allowlist qui n'a plus de
// CHECKPOINT échoue (entrée devenue inutile), un fichier qui en ajoute au-delà de son compte aussi.
func TestCheckpointProblems_UnusedEntryAndExcessFail(t *testing.T) {
	allow := map[string]checkpointAllowance{
		"cmd/a/main.go": {1, "outil (2026-11-02)"},
		"cmd/b/main.go": {1, "outil (2026-11-02)"},
	}
	sources := []goSource{
		parseGoSnippet(t, "cmd/a/main.go", "package main\nvar a = \"CHECKPOINT\"\nvar b = \"CHECKPOINT\""),
		parseGoSnippet(t, "cmd/b/main.go", "package main\nvar a = 1"),
	}
	problems := checkpointProblems(sources, allow)
	if len(problems) != 2 || !strings.Contains(problems[0], "cmd/a/main.go") || !strings.Contains(problems[1], "cmd/b/main.go") {
		t.Fatalf("problèmes = %v ; attendu l'excès de a et l'entrée inutile de b", problems)
	}
	ok := checkpointProblems(sources[:1], map[string]checkpointAllowance{"cmd/a/main.go": {2, "outil (2026-11-02)"}})
	if len(ok) != 0 {
		t.Errorf("compte exact : aucun problème attendu, obtenu %v", ok)
	}
}
