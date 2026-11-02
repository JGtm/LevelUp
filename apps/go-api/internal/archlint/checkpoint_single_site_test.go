// Package archlint — checkpoint_single_site_test.go : garde-rail « le CHECKPOINT d'une base DuckDB
// s'écrit dans le paquet qui le centralise ».
//
// POURQUOI. Un CHECKPOINT lancé n'importe où l'est sans le bail d'écriture de la base, sans
// journal ni métrique, et sans savoir si un lot d'écriture est en cours : le CHECKPOINT périodique
// du profil desktop (duckdb.CheckpointHeld) prend le bail, compte ses échecs et se journalise. Le
// paquet central est internal/platform/duckdb.
//
// INTERDIT (hors tests) : une instruction SQL CHECKPOINT (littéral de chaîne « CHECKPOINT »,
// « FORCE CHECKPOINT ») dans un fichier de cmd/, internal/, pkg/ ou
// scripts/ qui n'est pas dans internal/platform/duckdb/, sauf entrée de l'allowlist.
//
// L'allowlist est le décompte des sites existants à la pose du ratchet : un fichier qui en
// ajoute un nouveau dépasse son décompte et échoue. Elle ne fait que baisser : un site migré vers
// le paquet central retire sa ligne. Une entrée nouvelle exige une date et une justification.
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

// checkpointStatement reconnaît un littéral qui EST une instruction CHECKPOINT (pas un message de
// journal qui en parle).
var checkpointStatement = regexp.MustCompile(`(?i)^\s*(force\s+)?checkpoint\s*;?\s*$`)

// checkpointAllowlist : fichier -> nombre de CHECKPOINT tolérés. Sites existants au 2026-11-02, jour
// de la pose du ratchet (lot L20.g) : outils en ligne de commande qui ouvrent leur base seuls,
// CHECKPOINT d'arrêt ou d'après-écriture du serveur (ADR 0021, ADR 0022), migrations.
var checkpointAllowlist = map[string]int{
	"cmd/backfill-world-player-stats/main.go":       1,
	"cmd/cleanup_media_index/main.go":               2,
	"cmd/levelup/cmd_backfill_h5_kill_mechanics.go": 1,
	"cmd/levelup/cmd_compact_passes.go":             1,
	"cmd/levelup/cmd_compact_passes_rewrite.go":     1,
	"cmd/levelup/cmd_consolidate_aliases.go":        1,
	"cmd/levelup/cmd_rebuild_pme.go":                2,
	"cmd/levelup/cmd_repair_highlight_phantoms.go":  1,
	"cmd/purge_foreign_lusr_chain/purge.go":         1,
	"cmd/purge_player_media/main.go":                1,
	"cmd/rebuild_pme_art/main.go":                   2,
	"cmd/rebuild_shared_social/main.go":             1,
	"cmd/recompute_perfnote/main.go":                1,
	"cmd/server/background_tasks.go":                1,
	"cmd/server/shutdown.go":                        2,
	"cmd/world-aliases-persist/main.go":             1,
	"internal/migration/compaction.go":              1,
	"internal/migration/declared_keys_repair.go":    1,
	"internal/migration/registry.go":                1,
	"internal/ops/media.go":                         1,
	"internal/ops/media_capture_recalc.go":          2,
	"internal/ops/media_hls.go":                     1,
	"internal/ops/media_store.go":                   1,
	"internal/ops/seed_demo_prestige_social.go":     1,
	"internal/persist/shared_social_persister.go":   3,
	"internal/platform/dblease/writer.go":           1,
	"scripts/repair_tracks_table/main.go":           2,
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
		if err != nil || !checkpointStatement.MatchString(v) {
			return true
		}
		sites = append(sites, src.Rel+":"+strconv.Itoa(src.Fset.Position(lit.Pos()).Line))
		return true
	})
	return sites
}

func checkpointProblems(sources []goSource) []string {
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
		if allowed := checkpointAllowlist[file]; len(sites) > allowed {
			out = append(out, file+" : "+strconv.Itoa(len(sites))+" CHECKPOINT (toléré : "+strconv.Itoa(allowed)+") "+strings.Join(sites, ", "))
		}
	}
	for file, allowed := range checkpointAllowlist {
		if len(perFile[file]) < allowed {
			out = append(out, file+" : l'allowlist tolère "+strconv.Itoa(allowed)+" CHECKPOINT, il n'en reste que "+
				strconv.Itoa(len(perFile[file]))+" : baisser ou retirer la ligne")
		}
	}
	sort.Strings(out)
	return out
}

func TestCheckpointOnlyInTheCentralPackage(t *testing.T) {
	problems := checkpointProblems(goSourcesUnder(t, checkpointRoots...))
	if len(problems) > 0 {
		t.Errorf("CHECKPOINT hors de %s (le centraliser : duckdb.CheckpointHeld ou un helper de ce paquet) :\n  %s",
			checkpointCentralDir, strings.Join(problems, "\n  "))
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
		{"nom de tâche qui contient le mot : rien à signaler", "cmd/server/x.go", "package main\nvar n = \"checkpoint shared_social\"", 0},
		{"message de journal : rien à signaler", "internal/ops/x.go", "package ops\nvar m = \"CHECKPOINT final au shutdown échoué\"", 0},
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
