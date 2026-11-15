package archlint

// film_verrou_decodage_parite_test.go — LE VERROU « UN DECODAGE A LA FOIS » PARLE LE MEME PROTOCOLE
// DES DEUX COTES DE LA FRONTIERE DE MODULE.
//
// Le verrou de production vit chez l'hote (`internal/filmrun/solo.go`) ; les tests et instruments de
// la brique qui decodent des films du cache prennent le meme verrou par leur propre copie
// (`libs/film/internal/decodelock/decodelock.go`), parce que la brique n'importe pas go-api
// (ADR 0041). Les deux ne s'excluent que s'ils verrouillent le MEME fichier et decrivent leur
// detenteur sous la MEME forme : ce test exige que les declarations qui fixent ce protocole
// (`soloFileName`, `soloHolderFileName`, `soloHolder`) soient identiques, texte pour texte, dans
// les deux fichiers.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Changer `"film_decode.lock"` ou une balise JSON de `soloHolder` dans l'un des deux fichiers.

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"testing"
)

// coteDuVerrou : un fichier qui porte le protocole du verrou de decodage, relatif a la racine du depot.
var cotesDuVerrou = []string{
	"apps/go-api/internal/filmrun/solo.go",
	racineBriqueFilm + "/internal/decodelock/decodelock.go",
}

// nomsDuProtocoleDuVerrou : les declarations qui fixent le protocole.
var nomsDuProtocoleDuVerrou = []string{"soloFileName", "soloHolderFileName", "soloHolder"}

func TestLeVerrouDeDecodageParleLeMemeProtocoleDesDeuxCotes(t *testing.T) {
	racine := workspaceRoot(t)
	ref := declarationsDuProtocole(t, cheminDuDepotOuFatal(t, racine, cotesDuVerrou[0]))
	autre := declarationsDuProtocole(t, cheminDuDepotOuFatal(t, racine, cotesDuVerrou[1]))
	for _, nom := range nomsDuProtocoleDuVerrou {
		a, b := ref[nom], autre[nom]
		if a == "" || b == "" {
			t.Errorf("%s absent de %s ou de %s : le protocole ne se compare plus", nom, cotesDuVerrou[0], cotesDuVerrou[1])
			continue
		}
		if a != b {
			t.Errorf("%s differe entre %s et %s :\n  %s\n  %s", nom, cotesDuVerrou[0], cotesDuVerrou[1], a, b)
		}
	}
}

// declarationsDuProtocole rend le texte imprime (sans commentaire) de chaque declaration de
// constante ou de type de premier niveau du fichier, par nom.
func declarationsDuProtocole(t *testing.T, chemin string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatalf("%s : %v", chemin, err)
	}
	out := map[string]string{}
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || (g.Tok != token.CONST && g.Tok != token.TYPE) {
			continue
		}
		for _, s := range g.Specs {
			var buf bytes.Buffer
			switch sp := s.(type) {
			case *ast.ValueSpec:
				if err := printer.Fprint(&buf, fset, sp); err != nil {
					t.Fatal(err)
				}
				for _, n := range sp.Names {
					out[n.Name] = buf.String()
				}
			case *ast.TypeSpec:
				if err := printer.Fprint(&buf, fset, sp); err != nil {
					t.Fatal(err)
				}
				out[sp.Name.Name] = buf.String()
			}
		}
	}
	return out
}
