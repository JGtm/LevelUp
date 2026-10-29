// briques_internes_garde_test.go — auto-preuves de la table des briques INTERNES
// (`briques_internes_test.go`, plan D3 lot 1) : chaque règle de confrontation rougit sur un extrait
// synthétique, présenté sous le chemin où le fichier vivra.
//
// Mutation qui doit rougir `TestBriquesInternes_Imports` : abaisser une tolérance de 1.
package archlint

import (
	"strings"
	"testing"
)

// fichierImportsDeSnippet analyse un extrait de source présenté comme le fichier `rel` (relatif à
// la racine du dépôt) et en rend les imports, comme le parcours du disque les livre.
func fichierImportsDeSnippet(t *testing.T, rel, src string) fichierImports {
	t.Helper()
	gs := parseGoSnippet(t, rel, src)
	fi := fichierImports{rel: rel}
	for _, imp := range gs.File.Imports {
		fi.imports = append(fi.imports, strings.Trim(imp.Path.Value, `"`))
	}
	return fi
}

// briqueHaloapiACloture : la ligne `haloapi` de la table, sans tolérance — l'état de D3 une fois
// les coupes faites.
func briqueHaloapiACloture(t *testing.T) briqueInterne {
	t.Helper()
	for _, b := range briquesInternes {
		if b.nom == "haloapi" {
			b.tolerances = nil
			return b
		}
	}
	t.Fatal("la ligne `haloapi` a disparu de la table des briques internes")
	return briqueInterne{}
}

// TestBriquesInternes_AutoPreuve_Franchissements : à la clôture (sans tolérance), un import de
// `domain` depuis un fichier présenté comme `internal/haloapi/x.go` est un franchissement ; la
// bibliothèque standard, `haloapi/...` et les deux tiers permis ne le sont pas ; un tiers hors
// liste l'est ; le paquet propre d'un fichier de `platform/auth` n'en est pas un, mais le paquet
// d'un autre dossier hors périmètre si.
func TestBriquesInternes_AutoPreuve_Franchissements(t *testing.T) {
	b := briqueHaloapiACloture(t)
	fichiers := []fichierImports{
		fichierImportsDeSnippet(t, "apps/go-api/internal/haloapi/x.go", `package haloapi
import (
	"fmt"
	"net/http"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
	"levelup/go-api/internal/haloapi/xblauth"
	"levelup/go-api/internal/domain"
)
`),
		fichierImportsDeSnippet(t, "apps/go-api/internal/haloapi/infinite/y.go", `package infinite
import (
	"levelup/go-api/internal/haloapi"
	"golang.org/x/sync/singleflight"
	"github.com/JGtm/LevelUp/libs/film/decfilm"
)
`),
		fichierImportsDeSnippet(t, "apps/go-api/internal/platform/auth/xsts.go", `package auth
import (
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/halo"
	"levelup/go-api/internal/platform/halo/duration"
	"levelup/go-api/internal/observability/logging"
)
`),
		fichierImportsDeSnippet(t, "apps/go-api/internal/platform/halo/z.go", `package halo
import "levelup/go-api/internal/platform/auth"
`),
	}
	got := comptesDeFranchissements(b, fichiers)
	attendu := map[string]int{
		"domain":                         1,
		"golang.org/x/sync/singleflight": 1,
		"libs/film/decfilm":              1,
		"observability/logging":          1,
		"platform/auth":                  1, // `platform/halo` n'a pas droit au paquet d'exploitation des jetons
	}
	if len(got) != len(attendu) {
		t.Fatalf("franchissements = %v, attendu %v", got, attendu)
	}
	for imp, n := range attendu {
		if got[imp] != n {
			t.Errorf("franchissements[%q] = %d, attendu %d (tous : %v)", imp, got[imp], n, got)
		}
	}
	// Sans tolérance, le verdict est rouge et nomme l'import.
	ecarts := confronterTolerances(got, nil)
	if len(ecarts) != 5 || !strings.Contains(strings.Join(ecarts, "\n"), "domain : 1 fichier(s), import absent") {
		t.Fatalf("écarts = %v, attendu 5 imports absents de la table dont domain", ecarts)
	}
}

// TestBriquesInternes_AutoPreuve_Tolerances : les règles de confrontation (D2), une par cas, et le
// refus d'une table mal formée.
func TestBriquesInternes_AutoPreuve_Tolerances(t *testing.T) {
	toleres := map[string]toleranceImport{
		"domain":  {3, "2026-10-29", "lot 4"},
		"ctxkeys": {2, "2026-10-29", "lot 4"},
		"assets":  {0, "2026-10-29", "lot 4"},
		"i18n":    {1, "2026-10-29", "lot 4"},
	}
	cas := map[string]struct {
		comptes map[string]int
		attendu string // sous-chaîne attendue dans les écarts
	}{
		"tolérance 0":               {map[string]int{"domain": 3, "ctxkeys": 2, "i18n": 1}, "assets : tolérance 0"},
		"au-dessus de la tolérance": {map[string]int{"domain": 4, "ctxkeys": 2, "i18n": 1}, "domain : 4 fichier(s) > tolérance 3"},
		"en dessous (baisse)":       {map[string]int{"domain": 2, "ctxkeys": 2, "i18n": 1}, "domain : baisse à 2"},
		"entrée morte":              {map[string]int{"domain": 3, "ctxkeys": 2}, "i18n : baisse à 0"},
		"import absent de la table": {map[string]int{"domain": 3, "ctxkeys": 2, "i18n": 1, "metadata": 1}, "metadata : 1 fichier(s), import absent"},
	}
	for nom, c := range cas {
		ecarts := strings.Join(confronterTolerances(c.comptes, toleres), "\n")
		if !strings.Contains(ecarts, c.attendu) {
			t.Errorf("%s : écarts %q, attendu une ligne contenant %q", nom, ecarts, c.attendu)
		}
	}
	// Table égale aux comptes : aucun écart.
	delete(toleres, "assets")
	if ecarts := confronterTolerances(map[string]int{"domain": 3, "ctxkeys": 2, "i18n": 1}, toleres); len(ecarts) != 0 {
		t.Errorf("table égale aux comptes : aucun écart attendu, obtenu %v", ecarts)
	}
	mauvaise := map[string]toleranceImport{
		"a": {1, "29/10/2026", "lot 4"},
		"b": {1, "2026-10-29", " "},
		"c": {-1, "2026-10-29", "lot 4"},
	}
	if got := anomaliesDeTable(mauvaise); len(got) != 3 {
		t.Errorf("anomaliesDeTable = %v, attendu 3 défauts (date, lot, négatif)", got)
	}
}

// TestBriquesInternes_AutoPreuve_DossierAbsent : un dossier déclaré qui n'existe pas, un fichier
// nommé qui a disparu et un dossier à liste absent sont refusés (une table qui ne voit rien ne
// garde rien) ; le dossier cible, lui, existe.
func TestBriquesInternes_AutoPreuve_DossierAbsent(t *testing.T) {
	racine := workspaceRoot(t)
	for nom, b := range map[string]briqueInterne{
		"dossier absent":         {nom: "x", dossiers: []string{"apps/go-api/internal/ce_dossier_n_existe_pas"}},
		"fichier disparu":        {nom: "x", fichiers: map[string][]string{dossierPlatAuth: {"fichier_disparu.go"}}},
		"dossier à liste absent": {nom: "x", fichiers: map[string][]string{"apps/go-api/internal/absent": {"a.go"}}},
	} {
		if _, err := fichiersDeBriqueInterne(racine, b); err == nil {
			t.Errorf("%s : une description d'arborescence disparue doit être refusée", nom)
		}
	}
	if _, err := fichiersDeBriqueInterne(racine, briqueInterne{nom: "ok", dossiers: []string{dossierHaloapi}}); err != nil {
		t.Errorf("le dossier cible %s doit exister (doc.go du paquet parent) : %v", dossierHaloapi, err)
	}
}

// TestCleDImport : la clé suit la forme des trois sortes d'imports.
func TestCleDImport(t *testing.T) {
	for imp, attendu := range map[string]string{
		"levelup/go-api/internal/platform/netguard": "platform/netguard",
		"levelup/go-api/pkg/x":                      "pkg/x",
		"github.com/JGtm/LevelUp/libs/film/decfilm": "libs/film/decfilm",
		"golang.org/x/sync/singleflight":            "golang.org/x/sync/singleflight",
	} {
		if got := cleDImport(imp); got != attendu {
			t.Errorf("cleDImport(%q) = %q, attendu %q", imp, got, attendu)
		}
	}
	if estBibliothequeStandard("levelup/go-api/internal/x") || !estBibliothequeStandard("net/http") ||
		estBibliothequeStandard("golang.org/x/time/rate") {
		t.Error("estBibliothequeStandard : stdlib = premier élément sans point et hors module")
	}
}
