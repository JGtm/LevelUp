package archlint

// no_unregistered_fallback_sites_test.go — LES DIRECTIONS (C) ET (D) DU REGISTRE DES REPLIS.
//
// SCINDÉ DE `no_unregistered_fallback_test.go` À LA REVUE DE JALON M1 (2026-09-15) : le fichier
// passait 500 lignes, seuil du dépôt. La coupure suit une responsabilité, pas une arithmétique —
// là-bas les deux directions qui confrontent le CODE au registre par la convention de nommage et
// par l'ancre ; ici celles qui confrontent les DÉCLENCHEMENTS et les EXEMPTIONS aux sites qu'ils
// prétendent avoir. L'en-tête de l'autre fichier porte la doctrine des quatre directions.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/facts/fallback"
)

// TestToutDeclenchementEstAUnSiteDuRegistre — DIRECTION (C) : déclenchement -> site.
//
// CE QUE (B) NE VOIT PAS. (B) relit les ancres que l'entrée cite ; elle ne regarde jamais les
// appels que l'entrée NE cite PAS. Un repli déclenché depuis un second endroit — une autre
// condition d'ouverture, un autre geste de retrait — reste donc invisible, et c'est exactement ce
// qui s'était produit pour `repli_vie_coupee_au_trou_de_replication` entre le lot 1.9.13 et la
// revue de jalon M1 (2026-09-15).
func TestToutDeclenchementEstAUnSiteDuRegistre(t *testing.T) {
	racine := racineGoAPI(t)
	valeurDeLaConstante := nomsDuPaquetFallback(t, racine)
	fichiersParNom := fichiersDesSites()
	appelsVus := 0
	for _, sous := range perimetreReplis {
		parcourirGoProduction(t, filepath.Join(racine, sous), func(rel string, f *ast.File) {
			for _, a := range declenchementsDuFichier(f) {
				nom, connu := valeurDeLaConstante[a]
				if !connu {
					t.Errorf("%s : `Declenche`/`DeclencheN` appelé avec %q, qui n'est pas une "+
						"constante de `fallback/noms.go` — un nom littéral ne se relie à aucune "+
						"entrée à la compilation", rel, a)
					continue
				}
				appelsVus++
				if fichiersParNom[nom][rel] {
					continue
				}
				t.Errorf("%s : le repli %q y est déclenché, mais aucun [fallback.Site] de son "+
					"entrée ne cite ce fichier (sites : %v).\n"+
					"  Un repli déclenché depuis un endroit que le registre ne décrit pas porte une\n"+
					"  condition d'ouverture non déclarée (D14 b). Ajouter le site à l'entrée, avec\n"+
					"  son ancre et — s'il diffère — son propre `Condition`.",
					rel, nom, triees(fichiersParNom[nom]))
			}
		}, racine)
	}
	if appelsVus < plancherDeclenchements {
		t.Fatalf("le parcours n'a vu que %d declenchements (plancher %d) : un ratchet qui ne "+
			"trouve plus les appels passe en silence", appelsVus, plancherDeclenchements)
	}
}

// plancherDeclenchements : le nombre minimal d'appels `Declenche`/`DeclencheN` que le parcours
// doit voir. Mesuré le 2026-09-15 : 13. Plancher à 8 — assez serré pour qu'un parcours cassé
// échoue, assez lâche pour qu'une conversion qui retire un repli ne le fasse pas rougir.
const plancherDeclenchements = 8

// TestChaqueNomConstantEstAuRegistre : une constante de `fallback/noms.go` sans entrée au
// registre est une constante ORPHELINE — le lot de conversion a retiré l'entrée sans retirer le
// nom que le code cite.
//
// L'EN-TÊTE DE `noms.go` ANNONCE CE TEST DEPUIS LE LOT 1.9.0 ; il n'existait pas (revue de jalon
// M1, 2026-09-15). Une documentation qui décrit un garde-rail absent est pire qu'aucune : elle
// fait croire la direction tenue.
func TestChaqueNomConstantEstAuRegistre(t *testing.T) {
	noms := nomsDuPaquetFallback(t, racineGoAPI(t))
	if len(noms) < plancherNomsConstants {
		t.Fatalf("`fallback/noms.go` n'a rendu que %d constantes (plancher %d) : le parcours "+
			"est casse", len(noms), plancherNomsConstants)
	}
	for id, nom := range noms {
		if _, ok := fallback.Lire(nom); !ok {
			t.Errorf("la constante %s vaut %q, qui n'est au registre d'AUCUNE entree — "+
				"retirer la constante avec l'entree (D14 d)", id, nom)
		}
	}
}

// plancherNomsConstants : mesuré le 2026-09-15 — 11 constantes. Plancher à 5.
const plancherNomsConstants = 5

// TestChaqueExemptionDeLEcrivainATouJoursSonSite — CONTRÔLE INVERSE DE L'ALLOWLIST (constat arch
// C2 de la revue de jalon M1, 2026-09-15).
//
// Une exemption est une PORTE OUVERTE sur la direction (A) : tant qu'elle vit, l'identifiant
// qu'elle nomme échappe au registre. Quand le code qui la justifiait disparaît, la porte reste —
// et le jour où un identifiant du même nom réapparaît ailleurs, c'est un repli de LevelUp qui
// passe. Le ratchet frère `no_title_package_in_analysis_test.go` tient la même règle sur ses
// exemptions ; celle-ci ne l'avait pas, et trois de ses sept lignes avaient déjà dérivé.
func TestChaqueExemptionDeLEcrivainATouJoursSonSite(t *testing.T) {
	racine := racineGoAPI(t)
	fset := token.NewFileSet()
	for id, ex := range replisDeLEcrivainDuJeu {
		chemin := filepath.Join(racine, filepath.FromSlash(ex.Fichier))
		f, err := parser.ParseFile(fset, chemin, nil, 0)
		if err != nil {
			t.Errorf("exemption %s : %s est illisible (%v) — l'identifiant a-t-il disparu ? "+
				"Dans ce cas, RETIRER la ligne de l'allowlist", id, ex.Fichier, err)
			continue
		}
		declare := false
		for _, nom := range identifiantsDeclares(f) {
			if nom == id {
				declare = true
				break
			}
		}
		if !declare {
			t.Errorf("exemption %s SANS SITE : %s ne declare plus cet identifiant.\n"+
				"  Une exemption qui survit a son site finit par en couvrir un autre : la RETIRER,\n"+
				"  ou corriger le fichier cite si le code a seulement demenage.", id, ex.Fichier)
		}
		if !dateExemptionConforme(ex.Date) || strings.TrimSpace(ex.Raison) == "" {
			t.Errorf("exemption %s : date %q ou raison vide — chaque ligne porte sa date et sa "+
				"raison", id, ex.Date)
		}
	}
}

// dateExemptionConforme : `AAAA-MM-JJ`, syntaxique. Une date d'exemption est une reference.
func dateExemptionConforme(d string) bool {
	if len(d) != 10 || d[4] != '-' || d[7] != '-' {
		return false
	}
	for i, c := range d {
		if i != 4 && i != 7 && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// nomsDuPaquetFallback rend `identifiant de constante -> valeur` en PARSANT `fallback/noms.go`.
//
// PAR L'AST, ET NON PAR REFLEXION : un test d'`archlint` ne peut pas enumerer les constantes d'un
// paquet importe. Le fichier est une simple liste de `NomXxx Nom = "repli_..."` ; le parser la
// relit sans que personne n'ait a la recopier.
func nomsDuPaquetFallback(t *testing.T, racine string) map[string]fallback.Nom {
	t.Helper()
	chemin := filepath.Join(racine, filepath.FromSlash(cheminNomsFallback))
	f, err := parser.ParseFile(token.NewFileSet(), chemin, nil, 0)
	if err != nil {
		t.Fatalf("parse de %s : %v", cheminNomsFallback, err)
	}
	out := map[string]fallback.Nom{}
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
			return true
		}
		lit, ok := vs.Values[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v, errU := strconv.Unquote(lit.Value)
		if errU != nil {
			return true
		}
		out[vs.Names[0].Name] = fallback.Nom(v)
		return true
	})
	return out
}

// cheminNomsFallback : le fichier des constantes de noms, relatif a `apps/go-api/`.
const cheminNomsFallback = "internal/games/halo_infinite/film/facts/fallback/noms.go"

// fichiersDesSites : `nom de repli -> ensemble des fichiers que ses sites citent`.
func fichiersDesSites() map[fallback.Nom]map[string]bool {
	out := map[fallback.Nom]map[string]bool{}
	for _, r := range fallback.Table() {
		if out[r.Nom] == nil {
			out[r.Nom] = map[string]bool{}
		}
		for _, s := range r.Sites {
			out[r.Nom][s.Fichier] = true
		}
	}
	return out
}

// declenchementsDuFichier rend, pour chaque appel `X.Declenche(a)` / `X.DeclencheN(a, …)` du
// fichier, l'IDENTIFIANT de constante passe en premier argument (`fallback.NomXxx` -> `NomXxx`),
// ou le litteral brut quand l'argument n'est pas une constante nommee.
func declenchementsDuFichier(f *ast.File) []string {
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		appel, ok := n.(*ast.CallExpr)
		if !ok || len(appel.Args) == 0 {
			return true
		}
		sel, ok := appel.Fun.(*ast.SelectorExpr)
		if !ok || (sel.Sel.Name != "Declenche" && sel.Sel.Name != "DeclencheN") {
			return true
		}
		out = append(out, nomDeLArgument(appel.Args[0]))
		return true
	})
	return out
}

// nomDeLArgument rend `NomXxx` pour `fallback.NomXxx` ou `NomXxx`, et la forme source sinon —
// ce qui fait rougir la direction (C) sur un nom litteral, comme voulu.
func nomDeLArgument(a ast.Expr) string {
	switch v := a.(type) {
	case *ast.SelectorExpr:
		return v.Sel.Name
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		return v.Value
	}
	return "<expression>"
}

// triees rend les cles d'un ensemble, triees — pour un message d'erreur stable.
func triees(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
