package archlint

// effective_hp_copies_test.go — garde-rail : la vie effective d'un Spartan (points de vie pour tuer,
// baseline du Rendement et de la Résistance) n'est écrite qu'à UN endroit, la valeur du titre dans
// `config/titles/{slug}/constants.toml` [damage_model] ; le web la lit de /bootstrap, le Go du
// resolver du titre. Le test interdit tout littéral numérique 225 ou 229 dans le code non testé (Go,
// TypeScript) : une copie de plus re-diverge. Les chaînes de caractères et les commentaires sont hors
// champ (exemples chiffrés des textes d'aide, prose).

import (
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// effectiveHpLiteral : une vie effective écrite en nombre (225 ou 229, entier ou décimal).
var effectiveHpLiteral = regexp.MustCompile(`^22[59](\.0+)?$`)

// effectiveHpTSLiteral : le même nombre dans une ligne de code TypeScript.
var effectiveHpTSLiteral = regexp.MustCompile(`\b22[59](\.0+)?\b`)

// effectiveHpAllowlist : fichiers (chemins de garde-rail du module Go) autorisés à porter le
// littéral, chacun justifié.
var effectiveHpAllowlist = map[string]string{
	// Repli du resolver quand il n'est pas câblé (commandes hors serveur, tests des autres paquets) ;
	// égalé au TOML d'Infinite par games.TestDefaultEffectiveHpToKill_EgaleLeTOMLDInfinite.
	"internal/games/damage_model.go": "repli du resolver, égalé au TOML par un test",
}

var (
	reTSString      = regexp.MustCompile(`"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|` + "`[^`]*`")
	reTSLineComment = regexp.MustCompile(`//.*$`)
)

// effectiveHpGoLiterals rend les lignes d'un source Go dont le CODE porte le littéral (le scanner
// ignore chaînes et commentaires).
func effectiveHpGoLiterals(src []byte) []int {
	fset := token.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(src))
	var s scanner.Scanner
	s.Init(file, src, nil, 0)
	var out []int
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			return out
		}
		if (tok == token.INT || tok == token.FLOAT) && effectiveHpLiteral.MatchString(lit) {
			out = append(out, file.Position(pos).Line)
		}
	}
}

// effectiveHpTSLiterals : même exercice pour TypeScript, par lignes. Un commentaire de bloc est
// reconnu à ses lignes ouvrantes (`/*`, ` *`).
func effectiveHpTSLiterals(src string) []int {
	var out []int
	for i, line := range strings.Split(src, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") || strings.HasPrefix(trimmed, "/*") {
			continue
		}
		code := reTSLineComment.ReplaceAllString(reTSString.ReplaceAllString(line, `""`), "")
		if effectiveHpTSLiteral.MatchString(code) {
			out = append(out, i+1)
		}
	}
	return out
}

func TestEffectiveHpToKill_AucunLitteralHorsDuTOMLGo(t *testing.T) {
	for _, src := range goSourcesUnder(t, "cmd", "internal", "pkg") {
		if _, allowed := effectiveHpAllowlist[src.Rel]; allowed {
			continue
		}
		for _, line := range effectiveHpGoLiterals(src.Src) {
			t.Errorf("%s:%d : littéral 225/229 dans le code : la vie effective vient de config/titles/{slug}/constants.toml [damage_model] (resolver games.EffectiveHpToKill)", src.Rel, line)
		}
	}
}

func TestEffectiveHpToKill_AucunLitteralHorsDuTOMLWeb(t *testing.T) {
	webSrc := filepath.Join(apiModuleRoot(t), "..", "web", "src")
	if err := exigerDossier(webSrc); err != nil {
		t.Fatalf("racine déclarée apps/web/src : %v", err)
	}
	err := filepath.WalkDir(webSrc, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if d.Name() == "node_modules" || d.Name() == "generated" {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		ext := filepath.Ext(name)
		if (ext != ".ts" && ext != ".tsx") || strings.Contains(name, ".test.") || strings.Contains(name, ".fixtures.") ||
			strings.HasSuffix(name, ".gen.ts") || strings.HasSuffix(name, "generated.ts") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(webSrc, path)
		for _, line := range effectiveHpTSLiterals(string(raw)) {
			t.Errorf("apps/web/src/%s:%d : littéral 225/229 dans le code : la vie effective se lit de /bootstrap (useEffectiveHpToKill)", filepath.ToSlash(rel), line)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de apps/web/src : %v", err)
	}
}

// Auto-preuve : les détecteurs voient le code, ignorent les commentaires et les chaînes.
func TestEffectiveHpToKill_LesDetecteursDistinguentCodeEtProse(t *testing.T) {
	goSrc := "package p\nconst A = 229\nvar x = 225.0 * k\n// 229 commentaire\nvar s = \"229 x\"\nvar z = 2290\n"
	if got := effectiveHpGoLiterals([]byte(goSrc)); len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("Go : lignes détectées %v, attendu [2 3]", got)
	}
	tsSrc := strings.Join([]string{
		"const A = 229",
		"const x = 225.0 * k",
		"// 229 commentaire",
		" * 225 commentaire de bloc",
		`const l = "229 × frags"`,
		"const y = 3 // 229 en fin de ligne",
		"const z = 2290",
	}, "\n")
	if got := effectiveHpTSLiterals(tsSrc); len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Errorf("TypeScript : lignes détectées %v, attendu [1 2]", got)
	}
}

// L'allowlist reste justifiée : un fichier listé existe et porte encore le littéral.
func TestEffectiveHpToKill_AllowlistJustifieeEtVivante(t *testing.T) {
	for rel, why := range effectiveHpAllowlist {
		if strings.TrimSpace(why) == "" {
			t.Errorf("%s : justification vide", rel)
		}
		raw, err := os.ReadFile(filepath.Join(apiModuleRoot(t), filepath.FromSlash(rel)))
		if err != nil {
			t.Errorf("%s : %v", rel, err)
			continue
		}
		if len(effectiveHpGoLiterals(raw)) == 0 {
			t.Errorf("%s : ne porte plus le littéral, retirer l'entrée de l'allowlist", rel)
		}
	}
}
