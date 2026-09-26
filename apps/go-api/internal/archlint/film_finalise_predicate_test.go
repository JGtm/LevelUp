package archlint

// film_finalise_predicate_test.go — UN SEUL PREDICAT « FILM FINALISE », ET PERSONNE NE COMPARE
// AU TYPE DES TEMPS FORTS EN DEHORS DE LUI (lot L3 de PLAN_RETOURS_REJEU_2026-09-23, 2026-09-23).
//
// # LE DEFAUT QUE CE RATCHET FERME
//
// Un film est FINALISE quand son manifeste porte le morceau des temps forts (`chunk_type 3`).
// Avant ce lot, cinq endroits le savaient chacun a leur facon — « le dernier numero » dans
// `replay.ScanDeaths`, une comparaison au type 3 dans `haloclient`, une autre dans `objectives`,
// une troisieme dans une commande, une quatrieme dans une fixture — et AUCUN ne l appliquait au
// seul moment qui comptait : l archivage. `ab526724` a ete archive 50 s apres la fin de son match,
// sur 34 morceaux au lieu de 37, et toute sa chaine d identite est tombee. La regle vit
// desormais dans `film/filmcache/finalise.go` ([filmcache.Finalise] pour juger un film,
// [filmcache.EstTempsForts] pour SELECTIONNER le morceau), et CLAUDE.md regle 6 veut le
// garde-rail avec la centralisation : une factorisation sans garde-rail re-diverge.
//
// # CE QU IL CHERCHE, DANS LES SOURCES DE PRODUCTION DE `cmd/` ET `internal/`
//
//	une comparaison (`==`, `!=`, `<`, `>`, `<=`, `>=`) ou un `case` dont un operande designe le type
//	des temps forts :
//	  - `FilmChunkTypeHighlightEvents` ou `ChunkTypeTempsForts`, qualifies ou non ;
//	  - une constante LOCALE de valeur 3 dont le nom contient « chunk » et « type » (le patron de
//	    `objectives.chunkTypePied`) ;
//	  - le litteral `3` quand l autre operande (ou l etiquette du `switch`) nomme un type de
//	    morceau (« chunktype », casse et soulignes ignores).
//
// Les commentaires ne sont pas du code : le balayage passe par l AST.
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
//   - ecrire `if c.ChunkType == 3 {` dans un fichier de production hors allowlist (jouee le
//     2026-09-23 dans `sync/killcollector/cache_films.go`, et figee par
//     [TestDetecteurDeComparaisonAuTypeTempsForts]) ;
//   - remettre `chunk.ChunkType != FilmChunkTypeHighlightEvents` dans `haloclient` ;
//   - ecrire `if m.ChunkType >= 3 {` a la place de `filmcache.EstTempsForts` (mutation R-b de la
//     revue adverse du lot, invisible avant que l ordre entre dans le detecteur) ;
//   - ajouter une SECONDE comparaison dans un fichier tolere (compte gele, constat L3-R7) ;
//   - migrer un site allowliste sans retirer son entree : « entree perimee, la retirer ».
//
// # CE QU IL NE GARDE PAS, ET QUI EST CONSIGNE
//
// L heuristique « le DERNIER numero porte les temps forts » (`nums[:len(nums)-1]`) ne se detecte
// pas par un motif sur le type : elle survit hors du perimetre du lot dans
// `film/replay/player_index.go` et `film/internal/facts/killsource/index_motif.go`, qui excluent le
// dernier numero en le croyant celui des temps forts. Consignee en decouverte (lot L3, 2026-09-23),
// migration future : exclure le morceau par son TYPE. Et un repli nomme la porte encore, pour un
// film charge sans manifeste (`repli_temps_forts_dernier_numero`, registre `facts/fallback`).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// fichierDuPredicatFinalise : le SEUL fichier qui compare au type des temps forts.
const fichierDuPredicatFinalise = "internal/games/halo_infinite/film/filmcache/finalise.go"

// operateursDeComparaison : egalite ET ordre. L ORDRE EN FAIT PARTIE depuis la reprise du lot
// (constat L3-R7 de la revue adverse, mutation R-b : `m.ChunkType >= 3` a la place de
// `filmcache.EstTempsForts(m.ChunkType)` restait vert) — « au moins le type des temps forts »
// est une regle recopiee au meme titre que « egal au type des temps forts ».
var operateursDeComparaison = map[token.Token]bool{
	token.EQL: true, token.NEQ: true, token.LSS: true, token.GTR: true, token.LEQ: true, token.GEQ: true,
}

// nomsDuTypeTempsForts : les constantes nommees du type des temps forts.
var nomsDuTypeTempsForts = map[string]bool{
	"FilmChunkTypeHighlightEvents": true,
	"ChunkTypeTempsForts":          true,
}

// comparaisonsDuPredicat : le nombre de comparaisons au type des temps forts que porte le
// predicat lui-meme — UNE, celle d `EstTempsForts`. Gele comme les tolerances : une seconde
// comparaison dans `finalise.go` serait une regle de plus a cote de la regle.
const comparaisonsDuPredicat = 1

// comparaisonAuTypeTempsFortsToleree : un site ANTERIEUR au lot, hors de son perimetre.
//
// `sites` GELE LE COMPTE DU FICHIER (constat L3-R7 de la revue adverse) : une allowlist par
// fichier laissait un fichier tolere recevoir de NOUVELLES comparaisons sans rougir. Un compte
// qui monte est une violation ; un compte qui baisse est une entree a reduire.
type comparaisonAuTypeTempsFortsToleree struct {
	fichier string
	sites   int
	pose    string
	retrait string
}

// comparaisonsAuTypeTempsFortsTolerees — LES TROIS SITES MESURES LE 2026-09-23, hors du
// perimetre ferme du lot L3 (qui a migre les siens : `haloclient`, `replay.ScanDeaths`). Tous
// trois SELECTIONNENT le morceau des temps forts ; aucun ne juge la finalisation. Critere de
// retrait commun : le site passe par `filmcache.EstTempsForts`, et son entree part dans le meme
// commit.
var comparaisonsAuTypeTempsFortsTolerees = []comparaisonAuTypeTempsFortsToleree{
	{
		fichier: "cmd/levelup/cmd_backfill_medailles_feed.go", sites: 1, pose: "2026-09-23",
		retrait: "remplacer `chunk.ChunkType != haloclient.FilmChunkTypeHighlightEvents` par " +
			"`!filmcache.EstTempsForts(chunk.ChunkType)`",
	},
	{
		fichier: "internal/games/halo_infinite/film/internal/facts/objectives/extract.go",
		sites:   1,
		pose:    "2026-09-23",
		retrait: "remplacer `chunkTypePied` par `filmcache.EstTempsForts` — geste de la couche " +
			"`facts` : l empreinte de `objectives.Rev` bouge, a recopier a revision constante avec une " +
			"note ecrite (sortie identique)",
	},
	{
		fichier: "internal/testfixtures/jgtm_full_match.go", sites: 1, pose: "2026-09-23",
		retrait: "remplacer `c.ChunkType == 3` par `filmcache.EstTempsForts(c.ChunkType)`",
	},
}

// TestAucuneComparaisonAuTypeTempsFortsHorsDuPredicat : LE RATCHET. Hors du predicat et des
// tolerances, toute comparaison est une violation ; DANS un fichier tolere (ou dans le predicat),
// toute comparaison AU-DELA du compte gele en est une aussi.
func TestAucuneComparaisonAuTypeTempsFortsHorsDuPredicat(t *testing.T) {
	gele := map[string]int{fichierDuPredicatFinalise: comparaisonsDuPredicat}
	for _, c := range comparaisonsAuTypeTempsFortsTolerees {
		gele[c.fichier] = c.sites
	}
	var violations []string
	for rel, lignes := range sitesDeComparaisonAuTypeTempsForts(t) {
		if plafond, ok := gele[rel]; ok {
			if len(lignes) > plafond {
				violations = append(violations, rel+" : "+strconv.Itoa(len(lignes))+
					" comparaisons pour un compte gele a "+strconv.Itoa(plafond)+" (lignes "+
					joindreLignes(lignes)+")")
			}
			continue
		}
		for _, l := range lignes {
			violations = append(violations, rel+":"+strconv.Itoa(l))
		}
	}
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Errorf("%d comparaison(s) au type des temps forts hors du predicat unique :\n  %s\n"+
		"Juger un film : filmcache.Finalise(chunks, typeDe). Selectionner son morceau des temps "+
		"forts : filmcache.EstTempsForts(chunkType). Une regle recopiee re-diverge — c est ainsi "+
		"qu `ab526724` a ete archive avant sa finalisation.",
		len(violations), strings.Join(violations, "\n  "))
}

// TestAllowlistDuTypeTempsFortsNEstPasPerimee : une entree qui ne compare plus se RETIRE, et une
// entree dont le compte a BAISSE se reduit — le ratchet ne tolere jamais plus que ce qui existe.
func TestAllowlistDuTypeTempsFortsNEstPasPerimee(t *testing.T) {
	sites := sitesDeComparaisonAuTypeTempsForts(t)
	if n := len(sites[fichierDuPredicatFinalise]); n != comparaisonsDuPredicat {
		t.Errorf("%s porte %d comparaison(s) au type des temps forts, attendu %d (fichier deplace ? "+
			"mettre a jour fichierDuPredicatFinalise)", fichierDuPredicatFinalise, n, comparaisonsDuPredicat)
	}
	for _, c := range comparaisonsAuTypeTempsFortsTolerees {
		if strings.TrimSpace(c.retrait) == "" || c.pose == "" || c.sites <= 0 {
			t.Errorf("%s : tolerance sans date, sans critere de retrait ou sans compte", c.fichier)
		}
		switch n := len(sites[c.fichier]); {
		case n == 0:
			t.Errorf("%s (pose %s) ne compare plus au type des temps forts : entree perimee, la "+
				"retirer de comparaisonsAuTypeTempsFortsTolerees", c.fichier, c.pose)
		case n < c.sites:
			t.Errorf("%s : %d comparaison(s) pour un compte gele a %d — reduire le compte", c.fichier,
				n, c.sites)
		}
	}
}

// joindreLignes rend « 12, 40 ».
func joindreLignes(lignes []int) string {
	out := make([]string, 0, len(lignes))
	for _, l := range lignes {
		out = append(out, strconv.Itoa(l))
	}
	return strings.Join(out, ", ")
}

// TestDetecteurDeComparaisonAuTypeTempsForts : le detecteur, sur des sources construites — la
// mutation du ratchet, figee pour qu un detecteur devenu aveugle rougisse aussi.
func TestDetecteurDeComparaisonAuTypeTempsForts(t *testing.T) {
	cas := []struct {
		nom, src string
		attendu  int
	}{
		{"litteral", `package p; func f(c struct{ChunkType int}) bool { return c.ChunkType == 3 }`, 1},
		{"litteral a gauche", `package p; func f(chunkType int) bool { return 3 != chunkType }`, 1},
		{"constante nommee", `package p; import "x"; func f(t int) bool { return t == x.FilmChunkTypeHighlightEvents }`, 1},
		{"constante locale", `package p; const chunkTypePied = 3; func f(t int) bool { return t == chunkTypePied }`, 1},
		{"case", `package p; func f(c struct{ChunkType int}) { switch c.ChunkType { case 3: } }`, 1},
		{"ordre", `package p; func f(c struct{ChunkType int}) bool { return c.ChunkType >= 3 }`, 1},
		{"ordre nommee", `package p; import "x"; func f(t int) bool { return t < x.ChunkTypeTempsForts }`, 1},
		{"autre type", `package p; func f(c struct{ChunkType int}) bool { return c.ChunkType == 2 }`, 0},
		{"autre ordre", `package p; func f(n int) bool { return n > 3 }`, 0},
		{"autre trois", `package p; func f(n int) bool { return n == 3 }`, 0},
		{"accesseur", `package p; func f(c struct{ChunkType int}) int { return c.ChunkType }`, 0},
	}
	for _, c := range cas {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "x.go", c.src, 0)
		if err != nil {
			t.Fatalf("%s : %v", c.nom, err)
		}
		got := len(comparaisonsAuTypeTempsForts(fset, f, constantesLocalesDeTypeTempsForts(f)))
		if got != c.attendu {
			t.Errorf("%s : %d comparaison(s) detectee(s), attendu %d", c.nom, got, c.attendu)
		}
	}
}

// sitesDeComparaisonAuTypeTempsForts rend, par fichier de production, les lignes qui comparent au
// type des temps forts. Parcours propre (aucune exception de repertoire heritee d un autre
// ratchet) : `cmd/` et `internal/`, hors `_test.go` et `testdata/`. Les constantes locales se
// cherchent PAR PAQUET (repertoire) : `objectives` declare `chunkTypePied` dans `film.go` et le
// compare dans `extract.go`.
func sitesDeComparaisonAuTypeTempsForts(t *testing.T) map[string][]int {
	t.Helper()
	racine := racineGoAPI(t)
	parDossier := map[string][]*ast.File{}
	noms := map[*ast.File]string{}
	fset := token.NewFileSet()
	for _, sous := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(racine, sous), func(chemin string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fset, chemin, nil, 0)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(racine, chemin)
			noms[f] = filepath.ToSlash(rel)
			parDossier[filepath.Dir(chemin)] = append(parDossier[filepath.Dir(chemin)], f)
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", sous, err)
		}
	}
	if len(noms) < 1000 {
		t.Fatalf("%d fichiers de production balayes : l arborescence a bouge, le ratchet ne garde "+
			"plus rien", len(noms))
	}
	out := map[string][]int{}
	for _, fichiers := range parDossier {
		locales := map[string]bool{}
		for _, f := range fichiers {
			for nom := range constantesLocalesDeTypeTempsForts(f) {
				locales[nom] = true
			}
		}
		for _, f := range fichiers {
			if lignes := comparaisonsAuTypeTempsForts(fset, f, locales); len(lignes) > 0 {
				out[noms[f]] = lignes
			}
		}
	}
	return out
}

// comparaisonsAuTypeTempsForts rend les lignes d un fichier qui comparent au type des temps forts.
// `locales` : les constantes du PAQUET de valeur 3 qui nomment un type de morceau.
func comparaisonsAuTypeTempsForts(fset *token.FileSet, f *ast.File, locales map[string]bool) []int {
	designe := func(e ast.Expr) bool {
		nom := nomTerminal(e)
		return nomsDuTypeTempsForts[nom] || locales[nom]
	}
	var lignes []int
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BinaryExpr:
			if !operateursDeComparaison[x.Op] {
				return true
			}
			if designe(x.X) || designe(x.Y) ||
				(estTrois(x.X) && nommeUnTypeDeMorceau(x.Y)) || (estTrois(x.Y) && nommeUnTypeDeMorceau(x.X)) {
				lignes = append(lignes, fset.Position(x.Pos()).Line)
			}
		case *ast.SwitchStmt:
			for _, s := range x.Body.List {
				cc, ok := s.(*ast.CaseClause)
				if !ok {
					continue
				}
				for _, e := range cc.List {
					if designe(e) || (estTrois(e) && x.Tag != nil && nommeUnTypeDeMorceau(x.Tag)) {
						lignes = append(lignes, fset.Position(e.Pos()).Line)
					}
				}
			}
		}
		return true
	})
	return lignes
}

// constantesLocalesDeTypeTempsForts : les constantes du fichier de valeur 3 dont le nom nomme un
// type de morceau (`chunkTypePied = 3`).
func constantesLocalesDeTypeTempsForts(f *ast.File) map[string]bool {
	out := map[string]bool{}
	for _, decl := range f.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, spec := range g.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, nom := range vs.Names {
				if i < len(vs.Values) && estTrois(vs.Values[i]) && nommeUnTypeDeMorceauNom(nom.Name) {
					out[nom.Name] = true
				}
			}
		}
	}
	return out
}

// nomTerminal : le nom d un identifiant ou le selecteur final d une expression `a.b.C`.
func nomTerminal(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return x.Sel.Name
	case *ast.ParenExpr:
		return nomTerminal(x.X)
	}
	return ""
}

func estTrois(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "3"
}

func nommeUnTypeDeMorceau(e ast.Expr) bool { return nommeUnTypeDeMorceauNom(nomTerminal(e)) }

func nommeUnTypeDeMorceauNom(nom string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(nom, "_", "")), "chunktype")
}
