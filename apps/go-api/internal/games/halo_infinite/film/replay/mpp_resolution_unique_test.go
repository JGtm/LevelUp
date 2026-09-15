package replay

// mpp_resolution_unique_test.go — UNE SEULE PORTE POUR LE DECOUPAGE MPP (constat 2 de la revue
// de jalon M1, 2026-09-15, lentille D13).
//
// # CE QUE CES DEUX TESTS TIENNENT
//
//	LA DECISION   `gwWidthsForFilm` suit la VERSION DE FORMAT du film, et non son nom de build :
//	              un film au format 27 dont le build est hors table prend les largeurs RELUES,
//	              pas les calibrees.
//	LE CHEMIN     aucun fichier de production de `filmdec/` ni de `replay/` ne resout le
//	              decoupage MPP par le profil complet (`BuildProfileFromFilm`). Deux chemins de
//	              resolution, c est deux chemins qui divergent — et ils avaient diverge : les
//	              poses d equipement lisaient les largeurs relues pendant que les socles et les
//	              vehicules prenaient les calibrees, DANS LA MEME CUISSON.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmsource"
	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

// TestGwWidthsForFilmSuitLaVersionDeFormat — LE TEMOIN DU CONSTAT 2, COTE SOCLES.
//
// Le film porte le format 27 (largeurs relues 9/5) et un build hors de la table des sept. Les
// largeurs CALIBREES proposees sont volontairement differentes : si le site consultait encore le
// build, il les rendrait.
func TestGwWidthsForFilmSuitLaVersionDeFormat(t *testing.T) {
	film := filmFormat27BuildInconnu(t)
	calibrees := filmdec.MPPWidths{Lead: 8, Index: 3}
	got := gwWidthsForFilm(filmdec.NewFilmContext(film), calibrees)
	if got.Lead != 9 || got.Index != 5 {
		t.Fatalf("largeurs installees = %d/%d, 9/5 attendues (les relues du format 27) — le site "+
			"consulte-t-il encore le nom de build ?", got.Lead, got.Index)
	}
}

// TestGwWidthsForFilmSeReplieQuandLaLargeurEstIndeterminee — l autre cote de la frontiere.
//
// Format 24 : la largeur MPP n est PAS relue (les deux oracles se contredisent, cf.
// `build_profile.go`). Le repli calibre decide encore, et c est voulu.
func TestGwWidthsForFilmSeReplieQuandLaLargeurEstIndeterminee(t *testing.T) {
	d0 := chunk00Brut(t, "11de8353") // HI_1_9_0, format 24
	film := filmDUnChunk(t, d0)
	calibrees := filmdec.MPPWidths{Lead: 8, Index: 3}
	got := gwWidthsForFilm(filmdec.NewFilmContext(film), calibrees)
	if got != calibrees {
		t.Fatalf("largeurs installees = %d/%d, les calibrees %d/%d attendues : le format 24 n a "+
			"pas de largeur relue", got.Lead, got.Index, calibrees.Lead, calibrees.Index)
	}
}

// TestAucuneResolutionMPPParLeProfilComplet — LE RATCHET DU CHEMIN UNIQUE.
//
// `BuildProfileFromFilm` resout le profil par le NOM DE BUILD (table de sept builds en dur). Il
// n a plus aucun appelant de production : la grammaire du bloc MPP est keyee par la version de
// format, et [filmdec.MPPWidthsForFilm] en est la porte. Un appel neuf ici serait le retour du
// defaut — et il ne se verrait pas, puisqu il ne change rien sur les builds connus.
func TestAucuneResolutionMPPParLeProfilComplet(t *testing.T) {
	fichiersVus := 0
	for _, dir := range []string{".", filepath.Join("..", "filmdec")} {
		parcourirProductionGo(t, dir, func(rel string, appels []string) {
			fichiersVus++
			for _, a := range appels {
				if a != "BuildProfileFromFilm" {
					continue
				}
				t.Errorf("%s appelle `BuildProfileFromFilm` : la resolution du decoupage MPP "+
					"passe par `filmdec.MPPWidthsForFilm` (version de format), jamais par le "+
					"nom de build (constat 2 de la revue M1, 2026-09-15)", rel)
			}
		})
	}
	if fichiersVus < plancherFichiersMPP {
		t.Fatalf("le parcours n a vu que %d fichiers (plancher %d) : un ratchet qui ne scanne "+
			"rien passe en silence", fichiersVus, plancherFichiersMPP)
	}
}

// plancherFichiersMPP : mesure du 2026-09-15 — `replay/` et `filmdec/` portent bien plus de
// 150 fichiers de production. Plancher pose bas, il ne garde qu une chose : que le parcours ait
// effectivement parcouru.
const plancherFichiersMPP = 150

// parcourirProductionGo applique `visite` aux APPELS QUALIFIES de chaque fichier Go NON-test d un
// repertoire, sans descendre dans les sous-repertoires (les deux paquets sont plats).
//
// PAR L AST, ET NON PAR `strings.Contains` : ces deux fichiers CITENT `BuildProfileFromFilm` dans
// leur en-tete, pour dire precisement pourquoi ils ne l appellent plus. Un ratchet qui mordrait
// sur la documentation de sa propre regle ferait retirer l explication — et c est l explication
// qui empeche la regression de revenir.
func parcourirProductionGo(t *testing.T, dir string, visite func(rel string, appels []string)) {
	t.Helper()
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s : %v", dir, err)
	}
	for _, e := range entrees {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		chemin := filepath.Join(dir, nom)
		visite(filepath.ToSlash(chemin), appelsQualifiesDuFichier(t, chemin))
	}
}

// appelsQualifiesDuFichier rend le nom de la fonction appelee de chaque appel du fichier, que
// l appel soit QUALIFIE (`filmdec.Xxx(...)`, depuis `replay/`) ou NU (`Xxx(...)`, depuis
// `filmdec/` lui-meme). Les DEUX formes comptent : le site de `filmdec` appelait
// `BuildProfileFromFilm` sans qualificateur, et un ratchet qui n aurait vu que la forme qualifiee
// aurait laisse passer exactement la moitie du defaut.
//
// Les commentaires ne sont pas parses, donc ils ne peuvent pas declencher le ratchet.
func appelsQualifiesDuFichier(t *testing.T, chemin string) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), chemin, nil, 0)
	if err != nil {
		t.Fatalf("parse de %s : %v", chemin, err)
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		appel, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fn := appel.Fun.(type) {
		case *ast.SelectorExpr:
			out = append(out, fn.Sel.Name)
		case *ast.Ident:
			out = append(out, fn.Name)
		}
		return true
	})
	return out
}

// filmFormat27BuildInconnu : la bobine `bcb6d393` (format 27) dont le nom de build est remplace
// par un build HORS TABLE, de meme longueur — la substitution ne deplace aucun octet.
func filmFormat27BuildInconnu(t *testing.T) *filmsource.Film {
	t.Helper()
	d0 := chunk00Brut(t, "bcb6d393")
	patche := append([]byte(nil), d0...)
	const ancien, neuf = "HI_1_12_0", "HI_9_99_9"
	i := strings.Index(string(patche), ancien)
	if i < 0 {
		t.Fatalf("nom de build %q introuvable dans le chunk_00 du temoin", ancien)
	}
	copy(patche[i:], neuf)
	return filmDUnChunk(t, patche)
}

// chunk00Brut rend le `chunk_00` INFLATE d une bobine versionnee, depuis son fichier — le
// helper voisin `chunk00DeLaBobine` prend un `buildMiniFilm`, dont ce fichier n a pas besoin.
func chunk00Brut(t *testing.T, court string) []byte {
	t.Helper()
	chemin := filepath.Join("testdata", "minifilm_"+court, "chunk_00.bin")
	b, err := os.ReadFile(chemin) //nolint:gosec // bobine versionnee du depot
	if err != nil {
		t.Fatalf("lecture de %s : %v", chemin, err)
	}
	return filmsource.Inflate(b)
}

// filmDUnChunk charge un film d un seul chunk deja inflate (le registre en position 0).
func filmDUnChunk(t *testing.T, chunk []byte) *filmsource.Film {
	t.Helper()
	f, err := filmsource.Load(filmsource.MemoryChunks{chunk}, nil)
	if err != nil {
		t.Fatalf("chargement du temoin : %v", err)
	}
	return f
}
