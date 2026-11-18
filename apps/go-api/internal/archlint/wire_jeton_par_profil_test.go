// Package archlint — wire_jeton_par_profil_test.go : garde-rail « un jeton ne se prend jamais en
// parcourant les profils suivis ».
//
// POURQUOI. Un jeton n'entre dans la mise en commun que sur l'accord de son propriétaire (ADR 0038 :
// aucune décision = refus). Une fonction qui énumère les profils (`LoadPlayers`) et rafraîchit le jeton
// du premier qui répond (`RefreshHaloTokensViaStoreFirst`) sert le jeton d'un joueur à une opération qui
// n'est pas la sienne, sans passer par la mise en commun. Un téléchargement dont le contenu ne dépend
// d'aucun joueur prend son jeton à la source de lecture publique de la mise en commun ; un
// téléchargement pour un joueur donné prend le jeton de ce joueur seul.
//
// INTERDIT dans les fichiers non-test de internal/api/wire : un même corps de fonction (déclaration ou
// littéral de fonction, dont les littéraux imbriqués ne comptent pas) qui appelle à la fois
// `RefreshHaloTokensViaStoreFirst` et `LoadPlayers`. Un rafraîchissement dans un littéral imbriqué
// échappe à la règle : limite assumée, la forme visée est la boucle sur les profils dans une même
// fonction.
//
// La liste d'exemptions est VIDE. Une entrée porte « fichier:fonction », une raison datée, et échoue
// quand elle ne couvre plus aucune faute.
package archlint

import (
	"go/ast"
	"sort"
	"strings"
	"testing"
)

const (
	racineWireJetons      = "internal/api/wire"
	appelRafraichirJeton  = "RefreshHaloTokensViaStoreFirst"
	appelParcoursProfils  = "LoadPlayers"
	prefixeDateExemptions = "2026-"
)

// exemptionsJetonParProfil : « fichier:fonction » -> raison datée. Vide.
var exemptionsJetonParProfil = map[string]string{}

// nomAppeleParCall rend le nom de la fonction ou de la méthode qu'appelle call (`f(...)`, `x.f(...)`).
func nomAppeleParCall(call *ast.CallExpr) string {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	}
	return ""
}

// appelsDirectsDuCorps rend l'ensemble des noms appelés directement par un corps de fonction, sans descendre
// dans les littéraux de fonction qu'il contient.
func appelsDirectsDuCorps(body *ast.BlockStmt) map[string]bool {
	noms := map[string]bool{}
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.CallExpr:
			noms[nomAppeleParCall(x)] = true
		}
		return true
	})
	return noms
}

// corpsDeFonctionJeton est un corps de fonction nommé pour le rapport.
type corpsDeFonctionJeton struct {
	nom  string
	body *ast.BlockStmt
}

// corpsDeFonctionsDuFichier liste les déclarations de fonction du fichier et, séparément, chacun de leurs
// littéraux de fonction.
func corpsDeFonctionsDuFichier(src goSource) []corpsDeFonctionJeton {
	var out []corpsDeFonctionJeton
	for _, decl := range src.File.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		out = append(out, corpsDeFonctionJeton{fd.Name.Name, fd.Body})
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if lit, ok := n.(*ast.FuncLit); ok {
				out = append(out, corpsDeFonctionJeton{fd.Name.Name + ".func", lit.Body})
			}
			return true
		})
	}
	return out
}

// fautesJetonParProfil rend « fichier:fonction » de chaque corps qui rafraîchit un jeton et parcourt
// les profils, hors exemptions ; et les exemptions devenues inutiles.
func fautesJetonParProfil(sources []goSource, exemptions map[string]string) []string {
	var out []string
	vues := map[string]bool{}
	for _, src := range sources {
		for _, c := range corpsDeFonctionsDuFichier(src) {
			appels := appelsDirectsDuCorps(c.body)
			if !appels[appelRafraichirJeton] || !appels[appelParcoursProfils] {
				continue
			}
			cle := src.Rel + ":" + c.nom
			if _, ok := exemptions[cle]; ok {
				vues[cle] = true
				continue
			}
			out = append(out, cle+" : "+appelRafraichirJeton+" et "+appelParcoursProfils+" dans le même corps")
		}
	}
	for cle := range exemptions {
		if !vues[cle] {
			out = append(out, cle+" : exemption sans faute à couvrir, la retirer")
		}
	}
	sort.Strings(out)
	return out
}

func TestWireSansJetonParcourantLesProfils(t *testing.T) {
	fautes := fautesJetonParProfil(goSourcesUnder(t, racineWireJetons), exemptionsJetonParProfil)
	if len(fautes) > 0 {
		t.Errorf("jeton pris en parcourant les profils (ADR 0038 : le prendre à la source de lecture publique "+
			"de la mise en commun, ou au joueur concerné seul) :\n  %s", strings.Join(fautes, "\n  "))
	}
}

func TestExemptionsJetonParProfilDatees(t *testing.T) {
	for cle, raison := range exemptionsJetonParProfil {
		if !strings.HasPrefix(raison, prefixeDateExemptions) {
			t.Errorf("%s : l'exemption commence par une date (« 2026-MM-JJ : raison »), obtenu %q", cle, raison)
		}
	}
}

func TestFautesJetonParProfilVoitLaFaute(t *testing.T) {
	const ancienCorps = `package wire
func (r *R) haloTokensForDrain() {
	players, _ := r.cfg.LoadPlayers("x")
	for _, p := range players {
		auth.RefreshHaloTokensViaStoreFirst(ctx, store, provider, p.XUID, p.Gamertag)
	}
}`
	cases := []struct {
		nom, src string
		want     int
	}{
		{"boucle sur les profils", ancienCorps, 1},
		{"dans un littéral de fonction", `package wire
func f() {
	g := func() {
		cfg.LoadPlayers("x")
		auth.RefreshHaloTokensViaStoreFirst()
	}
	g()
}`, 1},
		{"le parcours et le rafraîchissement dans deux corps", `package wire
func f() {
	d := Deps{
		Load: func() { cfg.LoadPlayers("x") },
		Fetch: func() { auth.RefreshHaloTokensViaStoreFirst() },
	}
	_ = d
}`, 0},
		{"rafraîchissement d'un joueur seul", `package wire
func f() { auth.RefreshHaloTokensViaStoreFirst() }`, 0},
		{"parcours seul", `package wire
func f() { cfg.LoadPlayers("x") }`, 0},
		{"mention en commentaire", "package wire\n// LoadPlayers RefreshHaloTokensViaStoreFirst\nfunc f() {}", 0},
	}
	for _, tc := range cases {
		t.Run(tc.nom, func(t *testing.T) {
			got := fautesJetonParProfil([]goSource{parseGoSnippet(t, "internal/api/wire/x.go", tc.src)}, nil)
			if len(got) != tc.want {
				t.Fatalf("fautes = %v, attendu %d", got, tc.want)
			}
		})
	}
	exemptee := fautesJetonParProfil([]goSource{parseGoSnippet(t, "internal/api/wire/x.go", ancienCorps)},
		map[string]string{"internal/api/wire/x.go:haloTokensForDrain": "2026-11-18 : test"})
	if len(exemptee) != 0 {
		t.Errorf("faute exemptée signalée : %v", exemptee)
	}
	inutile := fautesJetonParProfil(nil, map[string]string{"internal/api/wire/x.go:f": "2026-11-18 : test"})
	if len(inutile) != 1 {
		t.Errorf("exemption inutile non signalée : %v", inutile)
	}
}
