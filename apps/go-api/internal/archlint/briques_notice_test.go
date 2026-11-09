// briques_notice_test.go — chaque brique de `libs/` dit d'où viennent ses fichiers de données.
//
// # POURQUOI
//
// Une brique est publiée seule (ADR 0043) : quiconque la télécharge reçoit ses fixtures et ses tables
// embarquées sans rien savoir de leur origine. Une brique dont un fichier non source (XML de disposition,
// image, variante de carte, maillage, page capturée) n'a aucune ligne de provenance publie ce fichier sans
// dire à qui il appartient. `NOTICE.md`, à la racine de la brique, est cette ligne ; ce test la rend
// obligatoire.
//
// # CE QU'IL VÉRIFIE
//
// Chaque dossier de `libs/` (lu sur le disque, pas dans une table) a un `NOTICE.md`, et ce `NOTICE.md`
// cite, par son chemin relatif à la brique (séparateur `/`), tout fichier d'extension `.xml`, `.png`,
// `.mvar`, `.blob`, `.html`, `.jpg` ou `.webp` du dossier, `testdata` compris. Une citation est la présence
// du chemin dans le texte : le nom seul ne suffit pas, il s'en trouve de communs à deux dossiers.
//
// # CE QU'IL NE FAIT PAS
//
// Il ne juge pas le contenu de la notice, seulement qu'elle nomme le fichier. Les autres extensions
// (`.bin`, `.gz`, `.json`, `.tsv`, `.txt`) ne sont pas exigées : leurs dossiers portent leur propre
// `PROVENANCE.txt` ou sont du texte écrit à la main.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Ajouter `libs/haloapi/infinite/testdata/autre.html`, ou retirer `libs/himap/NOTICE.md`.
package archlint

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// extensionsNoticees : les extensions de fichier dont une brique doit citer l'origine.
var extensionsNoticees = map[string]bool{
	".xml": true, ".png": true, ".mvar": true, ".blob": true, ".html": true, ".jpg": true, ".webp": true,
}

// fichiersANoticer rend, triés et relatifs à `racine` (séparateur `/`), les fichiers sous `racine` dont
// l'extension figure dans `extensionsNoticees`.
func fichiersANoticer(t *testing.T, racine string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !extensionsNoticees[strings.ToLower(filepath.Ext(chemin))] {
			return nil
		}
		rel, relErr := filepath.Rel(racine, chemin)
		if relErr != nil {
			return relErr
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("parcours de %s : %v", racine, err)
	}
	sort.Strings(out)
	return out
}

// fichiersNonCites rend, dans l'ordre reçu, les chemins de `fichiers` que le texte de la notice ne contient pas.
func fichiersNonCites(notice string, fichiers []string) []string {
	var manquants []string
	for _, f := range fichiers {
		if !strings.Contains(notice, f) {
			manquants = append(manquants, f)
		}
	}
	return manquants
}

// dossiersDeBriques rend les noms des dossiers de `libs/`, lus sur le disque.
func dossiersDeBriques(t *testing.T) (racineLibs string, noms []string) {
	t.Helper()
	racineLibs = filepath.Join(workspaceRoot(t), "libs")
	entrees, err := os.ReadDir(racineLibs)
	if err != nil {
		t.Fatalf("lecture de libs/ : %v", err)
	}
	for _, e := range entrees {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			noms = append(noms, e.Name())
		}
	}
	return racineLibs, noms
}

func TestChaqueBriqueADesNoticeQuiCiteSesFichiersDeDonnees(t *testing.T) {
	racineLibs, briquesDisque := dossiersDeBriques(t)
	if len(briquesDisque) < 4 {
		t.Fatalf("%d dossiers sous libs/, au moins 4 attendus (film, gamefiles, haloapi, himap) : le parcours ne voit plus l'arborescence", len(briquesDisque))
	}
	for _, nom := range briquesDisque {
		racine := filepath.Join(racineLibs, nom)
		brut, err := os.ReadFile(filepath.Join(racine, "NOTICE.md")) //nolint:gosec // chemin issu du parcours de libs/
		if err != nil {
			t.Errorf("libs/%s : pas de NOTICE.md (%v) — chaque brique dit l'origine de ses fichiers de données", nom, err)
			continue
		}
		if manquants := fichiersNonCites(string(brut), fichiersANoticer(t, racine)); len(manquants) > 0 {
			t.Errorf("libs/%s/NOTICE.md ne cite pas ces fichiers (chemin relatif à la brique) :\n  %s",
				nom, strings.Join(manquants, "\n  "))
		}
	}
}

// TestFichiersNonCites_AutoPreuve : un fichier absent du texte rougit, un chemin cité passe, et un nom
// seul sans son dossier ne vaut pas citation.
func TestFichiersNonCites_AutoPreuve(t *testing.T) {
	notice := "| `himap/sbsp.xml` | layout |\n| `mapvar/testdata/a.mvar` | map |"
	fichiers := []string{"himap/sbsp.xml", "mapvar/testdata/a.mvar", "autre/sbsp.xml", "testdata/b.html"}
	got := fichiersNonCites(notice, fichiers)
	want := []string{"autre/sbsp.xml", "testdata/b.html"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("fichiersNonCites = %v, attendu %v", got, want)
	}
	if rien := fichiersNonCites(notice, fichiers[:2]); len(rien) != 0 {
		t.Fatalf("deux fichiers cités rendus manquants : %v", rien)
	}
}

// TestFichiersANoticer_ExtensionsEtChemins : la liste est relative à la brique, triée, et ne voit que
// les extensions exigées (insensible à la casse).
func TestFichiersANoticer_ExtensionsEtChemins(t *testing.T) {
	racine := t.TempDir()
	for _, rel := range []string{"a/x.xml", "a/testdata/y.PNG", "z.mvar", "a/code.go", "a/note.txt", "b/c.bin"} {
		chemin := filepath.Join(racine, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(chemin), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(chemin, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got := strings.Join(fichiersANoticer(t, racine), ",")
	if want := "a/testdata/y.PNG,a/x.xml,z.mvar"; got != want {
		t.Fatalf("fichiersANoticer = %q, attendu %q", got, want)
	}
}
