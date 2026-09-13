// Package archlint — no_local_msr_axes_test.go : ratchet G.3 (finitions v7.5,
// 2026-09-13).
//
// CE QU'IL INTERDIT. Nommer un index `idx_msr_*` de `match_skill_rank` en
// LITTÉRAL GO hors de sa source unique, `internal/platform/duckdb/indexcheck`.
// C'est la signature d'une CARTE D'AXES recopiée : la liste des index surveillés,
// leurs expressions de scan forcé et leurs prédicats de lookup.
//
// POURQUOI. Cette carte a deux consommateurs — la sonde data-health périodique
// (`internal/scheduler/data_health_msr_index.go`) et l'outil de réparation
// (`cmd/repair_msr_index`). Recopiée, elle diverge en silence dès qu'une migration
// ajoute un index : la sonde continue de rendre « sain » sur un axe qu'elle ne
// regarde plus, et l'outil ne reconstruit pas celui qui manque. C'est la leçon
// CLAUDE.md n°6 (centraliser + garde-rail) appliquée avant la 3e copie.
//
// CE QU'IL N'INTERDIT PAS. Les CRÉATEURS d'index (migrations, `sync/schema.go`)
// écrivent `idx_msr_*` DANS une DDL SQL, pas comme identifiant Go entre
// guillemets : ils ne matchent pas. Les commentaires non plus. Le motif vise
// exactement la forme « liste d'identifiants d'index en Go ».
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

// msrIndexLiteralRE — un nom d'index `idx_msr_*` écrit comme littéral Go entre
// guillemets. La DDL des migrations vit dans des raw strings à backticks
// (`CREATE INDEX IF NOT EXISTS idx_msr_playlist ON …`) et n'est pas concernée.
var msrIndexLiteralRE = regexp.MustCompile(`"idx_msr_[a-z_]+"`)

// msrAxesOwner — la source unique. Tout autre fichier qui nomme un index en
// littéral Go recopie la carte.
const msrAxesOwner = "internal/platform/duckdb/indexcheck/"

func TestCarteDesAxesMatchSkillRankNonRecopiee(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	internalRoot := filepath.Dir(filepath.Dir(thisFile))
	goAPIRoot := filepath.Dir(internalRoot)

	var violations []string
	trouveChezLeProprietaire := false
	for _, sub := range []string{"internal", "cmd"} {
		root := filepath.Join(goAPIRoot, sub)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			proprietaire := strings.HasPrefix(rel, msrAxesOwner)
			for i, line := range strings.Split(string(data), "\n") {
				if !msrIndexLiteralRE.MatchString(line) {
					continue
				}
				if proprietaire {
					trouveChezLeProprietaire = true
					continue
				}
				// Les tests ont le droit d'ATTENDRE un nom d'index (assertion sur
				// le résultat rendu par indexcheck) ; ils ne portent pas la carte.
				if strings.HasSuffix(rel, "_test.go") {
					continue
				}
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s/: %v", sub, err)
		}
	}

	if len(violations) > 0 {
		t.Errorf("carte des axes `match_skill_rank` RECOPIÉE hors de %s (%d ligne(s)) :\n  - %s\n"+
			"Utiliser indexcheck.MatchSkillRankAxes() / indexcheck.RunAll — une carte recopiée "+
			"cesse en silence de surveiller un index que la migration vient d'ajouter.",
			msrAxesOwner, len(violations), strings.Join(violations, "\n  - "))
	}

	// Le ratchet doit MORDRE : s'il ne trouve plus la carte chez son propriétaire,
	// c'est que le motif ne correspond plus à rien et qu'il est devenu décoratif.
	if !trouveChezLeProprietaire {
		t.Errorf("aucun littéral `\"idx_msr_*\"` trouvé sous %s — le motif du ratchet "+
			"ne décrit plus la carte d'axes : le corriger, pas le laisser vert à vide.", msrAxesOwner)
	}
}
