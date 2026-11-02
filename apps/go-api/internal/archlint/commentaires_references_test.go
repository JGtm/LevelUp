package archlint

// commentaires_references_test.go — ce qu'un commentaire du code a le droit de citer.
//
// Le dossier de travail des agents (`.ai` à la racine du dépôt : plans, audits, rapports,
// journaux) tourne, se range et s'archive. Un commentaire qui y renvoie pourrit au premier
// rangement, et un garde-rail qui exigeait l'existence du document cité cassait la CI pour un
// simple déplacement de fichier. Le code porte le contrat ; l'histoire va dans l'ADR ou le journal.
//
//   - TestAucunCommentaireGoNeCiteLeDossierAi : aucun commentaire Go de l'espace de travail
//     (`apps/go-api` et `libs/**`, ADR 0041) ne cite un chemin sous `.ai`. Seule exception : le
//     dépôt local des dumps du jeu (`re_dump`), dossier de DONNÉES ignoré par git, pas un document.
//     Les chaînes ne sont pas lues ici (c'est du comportement : `pas_de_lecture_sous_ai_test.go`).
//   - TestAucunSourceWebNeCiteLeDossierAi : aucun fichier source du web (`apps/web/src`,
//     `apps/web/e2e`) ne contient un chemin sous `.ai`, commentaire ou chaîne.
//   - TestAucunCommentaireNeViseLAncienPaquetFilmdec : le littéral de l'ancien chemin du
//     décodeur n'apparaît plus dans aucun fichier Go.
//
// Mutation qui doit faire rougir le premier : ajouter à `internal/sync/scope.go` un commentaire
// qui cite un plan sous `.ai`.

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// prefixeAi : un chemin sous le dossier de travail, découpé pour que ce fichier ne porte pas
// lui-même le littéral qu'il interdit (ni en commentaire, ni en chaîne).
const prefixeAi = "." + "ai/"

// prefixeReDump : le dépôt local des dumps du jeu, seule mention admise dans un commentaire Go.
const prefixeReDump = prefixeAi + "re_dump"

// citationsAi rend les positions (octets) des chemins sous `.ai` dans un texte. Une occurrence
// collée à une lettre ou un chiffre est un domaine (`claude.ai` suivi d'un chemin), pas le
// dossier du dépôt.
func citationsAi(texte string) []int {
	var out []int
	for i := 0; ; {
		j := strings.Index(texte[i:], prefixeAi)
		if j < 0 {
			return out
		}
		pos := i + j
		if pos == 0 || !estCaractereDeNom(texte[pos-1]) {
			out = append(out, pos)
		}
		i = pos + len(prefixeAi)
	}
}

func estCaractereDeNom(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '-'
}

// TestAucunCommentaireGoNeCiteLeDossierAi — plafond 0 : un commentaire cite un ADR, un fichier de
// code, ou rien.
func TestAucunCommentaireGoNeCiteLeDossierAi(t *testing.T) {
	balayerGo(t, func(rel, chemin string) {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, chemin, nil, parser.ParseComments|parser.SkipObjectResolution)
		if err != nil {
			t.Errorf("%s : analyse impossible : %v", rel, err)
			return
		}
		for _, groupe := range f.Comments {
			for _, c := range groupe.List {
				for _, pos := range citationsAi(c.Text) {
					if strings.HasPrefix(c.Text[pos:], prefixeReDump) {
						continue
					}
					t.Errorf("%s:%d : un commentaire cite le dossier de travail .ai — le code porte "+
						"le contrat : citer un ADR, un fichier de code, ou rien", rel, fset.Position(c.Pos()).Line)
				}
			}
		}
	})
}

// sourcesWeb : les racines du web balayées, relatives à la racine du dépôt.
var sourcesWeb = []string{"apps/web/src", "apps/web/e2e", "apps/web/scripts"}

// extensionsWeb : les fichiers source du web (la documentation `.md` n'est pas du code).
var extensionsWeb = map[string]bool{".ts": true, ".tsx": true, ".js": true, ".mjs": true, ".css": true}

// TestAucunSourceWebNeCiteLeDossierAi — plafond 0, commentaires et chaînes : le web n'a aucune
// raison de nommer le dossier de travail.
func TestAucunSourceWebNeCiteLeDossierAi(t *testing.T) {
	depot := workspaceRoot(t)
	vus := 0
	for _, racine := range sourcesWeb {
		base := filepath.Join(depot, filepath.FromSlash(racine))
		err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}
			if !extensionsWeb[filepath.Ext(chemin)] {
				return nil
			}
			vus++
			buf, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(depot, chemin)
			for _, ligne := range lignesCitantAi(string(buf)) {
				t.Errorf("%s:%d : cite le dossier de travail .ai — citer un ADR, un fichier de code, "+
					"ou rien", filepath.ToSlash(rel), ligne)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", racine, err)
		}
	}
	if vus < 1000 {
		t.Errorf("%d fichier(s) web lus, plancher 1000 : le balayage ne lit plus le web", vus)
	}
}

// lignesCitantAi rend les numéros de ligne (à partir de 1) qui citent un chemin sous `.ai`.
func lignesCitantAi(texte string) []int {
	var out []int
	for i, ligne := range strings.Split(texte, "\n") {
		if len(citationsAi(ligne)) > 0 {
			out = append(out, i+1)
		}
	}
	return out
}

// ancienPaquetFilmdec : l'ancien chemin du décodeur, découpé pour que ce fichier ne le porte
// pas lui-même. Le paquet a éclaté en `film/internal/{source,profile,grammar,facts}`,
// `film/replay` et `film/decfilm` (ADR 0034).
var ancienPaquetFilmdec = "film/" + "filmdec/"

// TestAucunCommentaireNeViseLAncienPaquetFilmdec — aucun fichier Go du module ne cite
// l'ancien chemin du décodeur. Ces citations sont des commandes `go test` d'en-tête : copiées
// telles quelles, elles échouaient (« directory not found »).
func TestAucunCommentaireNeViseLAncienPaquetFilmdec(t *testing.T) {
	balayerGo(t, func(rel, chemin string) {
		buf, err := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
		if err != nil {
			t.Errorf("%s : %v", rel, err)
			return
		}
		n := strings.Count(string(buf), ancienPaquetFilmdec)
		if n == 0 {
			return
		}
		t.Errorf("%s cite %d fois %q, qui n'existe plus — viser le paquet actuel "+
			"(film/internal/grammar/, film/replay/, …)", rel, n, ancienPaquetFilmdec)
	})
}
