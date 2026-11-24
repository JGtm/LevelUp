package archlint

// settings_store_archive_default_test.go — tout magasin de réglages qui peut ÉCRIRE reçoit le défaut d'archive du
// profil (`WithArchiveFilmsDefault`).
//
// `settings.Store.Save` écrit TOUTES les clés connues, dont `archive_films`. Un magasin construit sans le défaut du
// profil charge `archive_films = true` (défaut historique) et le fige dans le fichier à la première écriture : sur le
// site hébergé, où le défaut est « ne pas archiver », le correctif de l'archive serait annulé en silence. Les sites
// qui ne font que LIRE un réglage étranger à l'archive sont listés ci-dessous, avec la raison.

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

const settingsPkgPath = "levelup/go-api/internal/platform/settings"

// storesEnLectureSeule : fichier -> raison. Datée du 2026-11-24 ; une entrée qui se met à écrire doit en sortir.
var storesEnLectureSeule = map[string]string{
	"cmd/server/boot_memory.go":  "2026-11-24 : lit memory_limit_mb au démarrage, jamais de Save",
	"cmd/server/refresh_only.go": "2026-11-24 : fournisseur de jetons du mode refresh-only, lecture seule",
}

// alias rend le nom sous lequel le fichier importe le paquet des réglages ("" s'il ne l'importe pas).
func aliasReglages(f *ast.File) string {
	for _, imp := range f.Imports {
		chemin, err := strconv.Unquote(imp.Path.Value)
		if err != nil || chemin != settingsPkgPath {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "settings"
	}
	return ""
}

// storesSansDefautDArchive rend la position de chaque `settings.NewStore(...)` du fichier dont la chaîne d'appels
// ne passe pas par `.WithArchiveFilmsDefault(...)`.
func storesSansDefautDArchive(src goSource) []token.Pos {
	alias := aliasReglages(src.File)
	if alias == "" {
		return nil
	}
	estNewStore := func(c *ast.CallExpr) bool {
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "NewStore" {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		return ok && id.Name == alias
	}
	avecDefaut := map[token.Pos]bool{}
	ast.Inspect(src.File, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := c.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "WithArchiveFilmsDefault" {
			return true
		}
		// Descend la chaîne de gauche à droite jusqu'au constructeur.
		for x := sel.X; x != nil; {
			switch v := x.(type) {
			case *ast.CallExpr:
				if estNewStore(v) {
					avecDefaut[v.Pos()] = true
					x = nil
				} else {
					x = v.Fun
				}
			case *ast.SelectorExpr:
				x = v.X
			default:
				x = nil
			}
		}
		return true
	})
	var manquants []token.Pos
	ast.Inspect(src.File, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok && estNewStore(c) && !avecDefaut[c.Pos()] {
			manquants = append(manquants, c.Pos())
		}
		return true
	})
	return manquants
}

func TestToutMagasinDeReglagesEcrivainPorteLeDefautDArchive(t *testing.T) {
	for _, src := range goSourcesUnder(t, "cmd", "internal") {
		if strings.HasPrefix(src.Rel, "internal/platform/settings/") {
			continue // le paquet lui-même (SessionOptionsForConfig : lecture)
		}
		manquants := storesSansDefautDArchive(src)
		if len(manquants) == 0 {
			continue
		}
		if raison, ok := storesEnLectureSeule[src.Rel]; ok && raison != "" {
			continue
		}
		for _, p := range manquants {
			t.Errorf("%s : settings.NewStore sans .WithArchiveFilmsDefault(cfg.Capabilities.ArchiveFilmsByDefault) — "+
				"son Save figerait archive_films=true dans le fichier. Ajouter l'option, ou lister le fichier "+
				"(daté, raison) s'il ne fait que lire", src.Fset.Position(p))
		}
	}
}

// Auto-preuve : le détecteur voit un magasin sans l'option, et accepte la chaîne qui la porte.
func TestDetecteurDefautDArchive(t *testing.T) {
	sans := parseGoSnippet(t, "x/sans.go", `package x
import "levelup/go-api/internal/platform/settings"
func f(p string) { s := settings.NewStore(p).WithEnforcedDefaults(true); _ = s }`)
	if n := len(storesSansDefautDArchive(sans)); n != 1 {
		t.Errorf("magasin sans option : %d détecté(s), attendu 1", n)
	}
	avec := parseGoSnippet(t, "x/avec.go", `package x
import settings_platform "levelup/go-api/internal/platform/settings"
func f(p string) {
	s := settings_platform.NewStore(p).
		WithEnforcedDefaults(true).
		WithArchiveFilmsDefault(false)
	_ = s
}`)
	if n := len(storesSansDefautDArchive(avec)); n != 0 {
		t.Errorf("magasin avec option : %d détecté(s), attendu 0", n)
	}
}
