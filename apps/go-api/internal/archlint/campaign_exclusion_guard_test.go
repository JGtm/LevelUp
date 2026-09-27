// Package archlint — campaign_exclusion_guard_test.go : tout lecteur d'affichage qui
// agrège les matchs d'un joueur exclut la Campagne (backlog 2026-09-26, lot B4, D-5 ;
// règle produit du 2026-07-18, item H1).
//
// Remplace le garde structurel de internal/platform/duckdb (TestCampaignExclusion-
// StructuralCoverage), qui ne voyait que les const/var de premier niveau nommées Q…, dans
// un seul répertoire, avec la sous-chaîne exacte « xuid = ? ». Ses six dispenses sont
// reprises à l'identique (campaign_exclusion_guard_dispenses_test.go).
//
// Balayage : chaque déclaration (FuncDecl, et chaque nom d'une const/var de premier niveau)
// des fichiers non-test des cinq racines de D-5. Le texte d'une déclaration est l'agrégat
// de ses littéraux chaîne, plus celui des constantes et variables chaîne du paquet qu'elle
// nomme (concaténation, fmt.Sprintf, WriteString, variable intermédiaire). Une constante
// nommée qui est elle-même un lecteur garde son verdict : son texte n'est pas recopié chez
// ses utilisateurs (un seul signalement, au lieu de la déclarer deux fois).
//
// Lecteur = \b(match_participants|mv_player_matches)\b ET \bxuid\s*(=\s*\?|IN\s*\(), la
// forme de projection « (… xuid = ?) AS … » étant retirée avant le test.
// Exclusion reconnue = le jeton (campaignExclusionToken / analysis.CampaignExclusionToken)
// ou l'appel d'un résolveur listé dans resolveursCampagne, dans la déclaration ou dans une
// constante-fragment qu'elle compose.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// racinesExclusionCampagne : les cinq racines de D-5 (sous-répertoires compris).
var racinesExclusionCampagne = []string{
	"internal/platform/duckdb",
	"internal/progression",
	"internal/api/wire",
	"internal/service",
	"internal/analysis",
}

// racinesSansSousRepertoires : racine balayée sans descendre (D-5 : `internal/api/wire`).
var racinesSansSousRepertoires = map[string]bool{"internal/api/wire": true}

// resolveursCampagne : fonctions qui posent l'exclusion (appel reconnu quel que soit le
// qualificatif). Les alias de platform/duckdb délèguent à la source unique analysis ;
// campaignExcl est la méthode de progression/profile (même délégation).
var resolveursCampagne = map[string]bool{
	"SQLResolveCampaignExclusion":       true,
	"SQLExcludeCampaignVariants":        true,
	"SQLExcludeCampaignByMatchID":       true,
	"SQLExcludeAllCampaignByMatchID":    true,
	"resolveCampaignExclusion":          true,
	"resolveCampaignExclusionByMatchID": true,
	"excludeCampaignClause":             true,
	"excludeCampaignByMatchID":          true,
	"excludeAllCampaignByMatchID":       true,
	"campaignExcl":                      true,
}

// jetonsCampagne : identifiants du jeton d'exclusion (résolu au call site).
var jetonsCampagne = map[string]bool{"campaignExclusionToken": true, "CampaignExclusionToken": true}

var (
	sourceMatchsJoueur = regexp.MustCompile(`\b(match_participants|mv_player_matches)\b`)
	filtreXUID         = regexp.MustCompile(`(?i)\bxuid\s*(=\s*\?|IN\s*\()`)
	projectionXUID     = regexp.MustCompile(`(?i)\([^()]*\bxuid\s*=\s*\?[^()]*\)\s*AS\b`)
)

// estLecteurCampagne applique le critère de D-5 à un texte SQL agrégé.
func estLecteurCampagne(texte string) bool {
	if !sourceMatchsJoueur.MatchString(texte) {
		return false
	}
	return filtreXUID.MatchString(projectionXUID.ReplaceAllString(texte, " "))
}

// declCampagne : une déclaration balayée.
type declCampagne struct {
	cle   string // chemin.go:Recv.Func ou chemin.go:Nom
	ligne int
	noeud ast.Node
}

// paquetCampagne : un répertoire (un paquet non-test) et sa table de constantes chaîne.
type paquetCampagne struct {
	valeurs map[string]ast.Expr // const/var de premier niveau → expression
	decls   []declCampagne
	texte   map[string]string // mémo : texte complet d'une constante
	marque  map[string]bool   // mémo : la constante porte une exclusion
	encours map[string]bool
}

type bilanCampagne struct {
	decls, lecteurs, exclus int
}

func TestCampaignExclusionGuard(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	vues := map[string]bool{}
	total := bilanCampagne{}
	for _, racine := range racinesExclusionCampagne {
		paquets, err := chargerPaquetsCampagne(goAPIRoot, racine)
		if err != nil {
			t.Fatalf("%s : %v", racine, err)
		}
		bilan := bilanCampagne{}
		for _, p := range paquets {
			for _, d := range p.decls {
				bilan.decls++
				texte, marque := p.texteDecl(d.noeud)
				if !estLecteurCampagne(texte) {
					continue
				}
				bilan.lecteurs++
				if marque {
					bilan.exclus++
					continue
				}
				if _, dispense := dispensesExclusionCampagne[d.cle]; dispense {
					vues[d.cle] = true
					continue
				}
				violations = append(violations, d.cle+" (ligne "+strconv.Itoa(d.ligne)+")")
			}
		}
		if bilan.decls == 0 {
			t.Errorf("%s : aucune déclaration balayée, le garde ne couvre plus cette racine", racine)
		}
		t.Logf("%s : %d déclarations, %d lecteurs, %d exclus", racine, bilan.decls, bilan.lecteurs, bilan.exclus)
		total.decls += bilan.decls
		total.lecteurs += bilan.lecteurs
		total.exclus += bilan.exclus
	}
	if total.lecteurs == 0 || total.exclus == 0 {
		t.Fatalf("aucun lecteur (%d) ou aucun lecteur exclu (%d) : le critère ne voit plus rien", total.lecteurs, total.exclus)
	}
	verifierDispensesCampagne(t, vues)
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("lecteur des matchs d'un joueur SANS exclusion de la Campagne (D-5) : poser le jeton "+
			"campaignExclusionToken résolu au call site, ou un résolveur (excludeCampaignByMatchID…), "+
			"ou une dispense datée et justifiée dans dispensesExclusionCampagne :\n  %s", strings.Join(violations, "\n  "))
	}
}

// verifierDispensesCampagne : chaque dispense est complète et désigne encore un lecteur.
func verifierDispensesCampagne(t *testing.T, vues map[string]bool) {
	t.Helper()
	for cle, d := range dispensesExclusionCampagne {
		if !categoriesDispenseCampagne[d.categorie] || d.date == "" || d.raison == "" {
			t.Errorf("dispense %s incomplète : catégorie connue, date et raison sont obligatoires", cle)
		}
		if !vues[cle] {
			t.Errorf("dispense périmée : %s ne désigne plus un lecteur non exclu, la retirer", cle)
		}
	}
}

// chargerPaquetsCampagne parse les fichiers non-test d'une racine, un paquet par répertoire.
func chargerPaquetsCampagne(goAPIRoot, racine string) ([]*paquetCampagne, error) {
	parRepertoire := map[string]*paquetCampagne{}
	base := filepath.Join(goAPIRoot, filepath.FromSlash(racine))
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != base && (d.Name() == "testdata" || racinesSansSousRepertoires[racine]) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(goAPIRoot, path)
		rel = filepath.ToSlash(rel)
		fset := token.NewFileSet()
		f, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		dir := filepath.Dir(path)
		p := parRepertoire[dir]
		if p == nil {
			p = &paquetCampagne{valeurs: map[string]ast.Expr{}, texte: map[string]string{},
				marque: map[string]bool{}, encours: map[string]bool{}}
			parRepertoire[dir] = p
		}
		p.ajouterFichier(fset, f, rel)
		return nil
	})
	out := make([]*paquetCampagne, 0, len(parRepertoire))
	for _, p := range parRepertoire {
		out = append(out, p)
	}
	return out, err
}

// ajouterFichier enregistre les déclarations d'un fichier et ses constantes chaîne.
func (p *paquetCampagne) ajouterFichier(fset *token.FileSet, f *ast.File, rel string) {
	for _, decl := range f.Decls {
		switch x := decl.(type) {
		case *ast.FuncDecl:
			p.decls = append(p.decls, declCampagne{cle: rel + ":" + nomFonctionCampagne(x),
				ligne: fset.Position(x.Pos()).Line, noeud: x})
		case *ast.GenDecl:
			if x.Tok != token.CONST && x.Tok != token.VAR {
				continue
			}
			for _, spec := range x.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, nom := range vs.Names {
					if i >= len(vs.Values) || nom.Name == "_" {
						continue
					}
					p.valeurs[nom.Name] = vs.Values[i]
					p.decls = append(p.decls, declCampagne{cle: rel + ":" + nom.Name,
						ligne: fset.Position(nom.Pos()).Line, noeud: vs.Values[i]})
				}
			}
		}
	}
}

// nomFonctionCampagne rend « Recv.Func » (sans étoile) ou « Func ».
func nomFonctionCampagne(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	typ := fn.Recv.List[0].Type
	if st, ok := typ.(*ast.StarExpr); ok {
		typ = st.X
	}
	if ix, ok := typ.(*ast.IndexExpr); ok {
		typ = ix.X
	}
	if id, ok := typ.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// texteDecl rend le texte agrégé d'une déclaration et la présence d'une exclusion. Une
// constante référencée qui est elle-même un lecteur n'y est pas recopiée (son verdict est
// le sien).
func (p *paquetCampagne) texteDecl(n ast.Node) (string, bool) {
	return p.parcourir(n, false)
}

// texteComplet : texte d'une constante nommée, constantes référencées toutes incluses.
func (p *paquetCampagne) texteComplet(nom string) (string, bool) {
	if t, ok := p.texte[nom]; ok {
		return t, p.marque[nom]
	}
	if p.encours[nom] {
		return "", false
	}
	p.encours[nom] = true
	t, m := p.parcourir(p.valeurs[nom], true)
	delete(p.encours, nom)
	p.texte[nom], p.marque[nom] = t, m
	return t, m
}

// parcourir agrège littéraux, constantes référencées, jeton et appels de résolveur.
func (p *paquetCampagne) parcourir(n ast.Node, complet bool) (string, bool) {
	var morceaux []string
	marque := false
	selecteurs := map[*ast.Ident]bool{}
	ast.Inspect(n, func(x ast.Node) bool {
		switch v := x.(type) {
		case *ast.SelectorExpr:
			selecteurs[v.Sel] = true
			if jetonsCampagne[v.Sel.Name] {
				marque = true
			}
		case *ast.CallExpr:
			if resolveursCampagne[nomAppele(v.Fun)] {
				marque = true
			}
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				if s, err := strconv.Unquote(v.Value); err == nil {
					morceaux = append(morceaux, s)
				}
			}
		case *ast.Ident:
			if selecteurs[v] {
				return true
			}
			if jetonsCampagne[v.Name] {
				marque = true
			}
			if _, ok := p.valeurs[v.Name]; !ok {
				return true
			}
			t, m := p.texteComplet(v.Name)
			if !complet && estLecteurCampagne(t) {
				return true
			}
			morceaux = append(morceaux, t)
			marque = marque || m
		}
		return true
	})
	return strings.Join(morceaux, " "), marque
}
