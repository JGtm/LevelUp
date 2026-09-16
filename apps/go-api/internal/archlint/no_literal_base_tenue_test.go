// Package archlint — no_literal_base_tenue_test.go : « BASE TENUE EN ECRITURE » SE DIT PAR UNE
// ERREUR TYPÉE, PAS PAR UN LITTÉRAL RECOPIÉ (lot 2.10.4, découverte D3 (2.8) du 2026-09-17).
//
// # Le défaut que ces ratchets ferment
//
// Trois commandes collaient à la main « (serveur en ecriture ? reessayer) » à leur message
// d'ouverture — `cmd_backfill_replay.go`, `cmd_backfill_replay_repair.go`,
// `cmd_replay_facts_export.go` — et `cmd/replay-corpus-gate` armait son réessai en cherchant ce
// littéral dans le `stderr` du sous-processus. Deux dégâts :
//
//	EN SILENCE   un reformulage de l'un des trois messages désarmait le réessai du gate, qui
//	             reperdait des témoins sur un aléa de quelques secondes (D2 (clôture M1)) ;
//	A TORT       l'indication était collée à TOUTE erreur d'ouverture — un fichier absent
//	             s'annonçait « serveur en ecriture ? » et se faisait réessayer trois fois.
//
// La correction : `duckdb.ErrBaseTenueEnEcriture`, posée par le seul point d'ouverture
// (`openCachedDB`) quand DuckDB signale un verrou d'un AUTRE processus. En processus, on écrit
// `errors.Is`. Le gate, lui, vit dans un autre processus et lit du texte : il MIROITE le
// marqueur, et les deux tests ci-dessous tiennent le miroir.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// litteralBaseTenue : le fragment interdit hors du paquet `duckdb`.
const litteralBaseTenue = "serveur en ecriture"

// porteursDuLitteralBaseTenue : les DEUX endroits qui ont le droit de porter ce texte, chemins
// relatifs à `apps/go-api`, en slash.
//
//   - internal/platform/duckdb/db_recovery.go : la SENTINELLE elle-même, source unique ;
//   - cmd/replay-corpus-gate/facts.go : le MIROIR, tenu égal par
//     TestMarqueurDuGateEgaleLaSentinelleBaseTenue. Le gate exécute `levelup` et lit son
//     `stderr` : `errors.Is` n'y a aucun sens, et importer `internal/platform/duckdb`
//     embarquerait le pilote DuckDB (CGO) dans un binaire de gate qui n'ouvre aucune base.
var porteursDuLitteralBaseTenue = map[string]bool{
	"internal/platform/duckdb/db_recovery.go": true,
	"cmd/replay-corpus-gate/facts.go":         true,
}

// TestLitteralBaseTenueHorsPaquetDuckdbInterdit — personne ne recopie le texte de la sentinelle.
//
// Le balayage ignore les COMMENTAIRES : plusieurs fichiers racontent légitimement l'histoire de
// ce défaut en prose (`cmd_backfill_replay.go`, `cmd_backfill_replay_child.go`,
// `cmd_archive_films.go`). C'est le CODE qui ne doit plus décider sur une chaîne.
//
// Mutation qui doit le faire rougir : remettre `(serveur en ecriture ? reessayer)` dans le
// `fmt.Errorf` de `cmd_backfill_replay.go` (jouée le 2026-09-17).
func TestLitteralBaseTenueHorsPaquetDuckdbInterdit(t *testing.T) {
	goAPIRoot := racineGoAPI(t)
	moi := cheminRelatifDeCeFichier(t, goAPIRoot)
	var fautifs []string
	balayerSourcesGo(t, goAPIRoot, func(rel, texte string) {
		if rel == moi || porteursDuLitteralBaseTenue[rel] {
			return
		}
		if strings.Contains(stripComments(texte), litteralBaseTenue) {
			fautifs = append(fautifs, rel)
		}
	})
	if len(fautifs) > 0 {
		t.Errorf("%d fichier(s) portent le littéral %q hors du paquet duckdb :\n  %s\n"+
			"La base tenue se reconnaît par errors.Is(err, duckdb.ErrBaseTenueEnEcriture) ; "+
			"un message recopié se désarme en silence au premier reformulage.",
			len(fautifs), litteralBaseTenue, strings.Join(fautifs, "\n  "))
	}
}

// TestMarqueurDuGateEgaleLaSentinelleBaseTenue — le miroir du gate est bien un morceau du texte
// de la sentinelle. C'est CE test qui remplace `TestMarqueurBaseTenueExisteChezLevelup` du lot
// 2.8 : même intention (le réessai ne se désarme pas en silence), mais pointé sur une SOURCE
// UNIQUE dans un vrai paquet, au lieu d'un littéral cherché dans un `package main` voisin.
//
// Mutation qui doit le faire rougir : changer le texte de `ErrBaseTenueEnEcriture` sans changer
// `marqueurBaseTenue` (jouée le 2026-09-17).
func TestMarqueurDuGateEgaleLaSentinelleBaseTenue(t *testing.T) {
	goAPIRoot := racineGoAPI(t)
	sentinelle := valeurLitteraleDe(t,
		filepath.Join(goAPIRoot, "internal", "platform", "duckdb", "db_recovery.go"),
		"ErrBaseTenueEnEcriture")
	miroir := valeurLitteraleDe(t,
		filepath.Join(goAPIRoot, "cmd", "replay-corpus-gate", "facts.go"),
		"marqueurBaseTenue")
	if miroir == "" || !strings.Contains(sentinelle, miroir) {
		t.Fatalf("le marqueur du gate (%q) n'est plus un morceau du texte de la sentinelle (%q) "+
			"— le réessai borné de l'export est désarmé : aligner marqueurBaseTenue sur "+
			"duckdb.ErrBaseTenueEnEcriture", miroir, sentinelle)
	}
	if !strings.Contains(sentinelle, litteralBaseTenue) {
		t.Errorf("la sentinelle (%q) ne porte plus %q, que ce fichier-ci interdit ailleurs — "+
			"mettre à jour litteralBaseTenue dans le même commit", sentinelle, litteralBaseTenue)
	}
}

// valeurLitteraleDe rend la chaîne littérale affectée au symbole `nom` (const ou var) dans le
// fichier Go `chemin` — que ce soit `= "x"` ou `= errors.New("x")`.
func valeurLitteraleDe(t *testing.T, chemin, nom string) string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatalf("lecture de %s : %v", chemin, err)
	}
	var valeur string
	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, ident := range spec.Names {
			if ident.Name != nom || i >= len(spec.Values) {
				continue
			}
			valeur = concatLitterales(spec.Values[i])
		}
		return true
	})
	if valeur == "" {
		t.Fatalf("symbole %s introuvable (ou valeur non littérale) dans %s", nom, chemin)
	}
	return valeur
}

// concatLitterales rend la concaténation des chaînes littérales d'une expression — elle traverse
// `errors.New(...)` et les `"a" + "b"` écrits sur plusieurs lignes.
func concatLitterales(e ast.Expr) string {
	var b strings.Builder
	ast.Inspect(e, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if s, err := strconv.Unquote(lit.Value); err == nil {
			b.WriteString(s)
		}
		return true
	})
	return b.String()
}

// cheminRelatifDeCeFichier rend le chemin de CE fichier de test relatif à `apps/go-api` : il
// cite le littéral qu'il interdit, il doit donc s'exclure lui-même.
func cheminRelatifDeCeFichier(t *testing.T, goAPIRoot string) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	rel, _ := filepath.Rel(goAPIRoot, ici)
	return filepath.ToSlash(rel)
}

// balayerSourcesGo appelle `visiter` pour chaque `.go` de `cmd/` et `internal/`.
func balayerSourcesGo(t *testing.T, goAPIRoot string, visiter func(rel, texte string)) {
	t.Helper()
	for _, sub := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(goAPIRoot, sub), func(chemin string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(chemin, ".go") {
				return nil
			}
			buf, rerr := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
			if rerr != nil {
				return rerr
			}
			rel, _ := filepath.Rel(goAPIRoot, chemin)
			visiter(filepath.ToSlash(rel), string(buf))
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", sub, err)
		}
	}
}
