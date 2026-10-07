package archlint

// film_faits_sans_octets_test.go — LA COUCHE DES FAITS NE LIT AUCUN OCTET DE FILM (lot 2.6 de la
// representation intermediaire, ADR 0037 : D-2 amende).
//
// # LA REGLE
//
// ADR 0034 D-2 : seule la couche `source` touche les octets d un film, et les ratchets
// `no_raw_film_bytes_*` le tiennent. Ils ne voient pas ce que l ADR 0037 ajoute : un appel a
// `source.BitsTronques` PASSE PAR LA PORTE, il est donc legal pour eux. L amendement dit que
// personne hors de la grammaire ne parcourt les paquets d un film et que LES FAITS LISENT LA
// STRUCTURE. Ce ratchet le tient pour la couche des faits (`film/internal/facts/`) : un fichier
// de production n y nomme de la couche `source` que le type du film charge (`source.Film`, qu il
// passe a la grammaire), n y appelle ni `Chunk` ni `Packets` (les octets d un chunk, les paquets
// d un chunk) et n y lit pas `Payload` (les octets d un paquet).
//
// Les FICHIERS DE TEST n y sont pas soumis, pour la raison que donne
// `no_raw_film_bytes_outside_source_test.go` : les instruments de mesure lisent des octets par
// construction.
//
// # LES EXCEPTIONS, DATEES
//
// Aucune : la liste [exceptionsDesFaits] est VIDE, et c est un cliquet. La seule qu elle ait portee,
// `facts/killsource` (sa propre marche du film), est partie au lot 2.7.c du plan de l etape 2
// (`.ai/PLAN_REPRESENTATION_INTERMEDIAIRE_ETAPE2_2026-10-03.md`) : killsource prend ses morts et
// ses messages de kill dans la marche unique. Une exception ajoutee porte sa date, sa raison et son
// critere de retrait ; celle qui ne couvre plus aucune lecture ROUGIT
// (`TestExceptionsDesFaitsSontDateesEtVivantes`).
//
// # MUTATIONS JOUEES (2026-10-03), ROUGES, PUIS RETIREES
//
//   - un appel `source.BitsTronques(nil, 0, 1)` dans `facts/objectives/statborg.go` ;
//   - un appel `film.Chunk(0)` dans le meme fichier ;
//   - l exception de `killsource` deplacee sur `facts/fallback`, qui ne lit rien : perimee.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const (
	// racineDesFaits : la couche des faits, relative a `apps/go-api`.
	racineDesFaits = "internal/games/halo_infinite/film/internal/facts"
	// importDeLaSource : la couche `source`, telle qu un fichier l importe.
	importDeLaSource = "levelup/go-api/internal/games/halo_infinite/film/internal/source"
	// plancherFichiersDesFaits : LE PLANCHER CONTRE UN BALAYAGE MUET. 81 fichiers de production
	// mesures le 2026-10-03 (killsource 38, objectives 26, fallback 17).
	plancherFichiersDesFaits = 60
)

// nomsPermisDeLaSource : ce qu un fichier de faits peut nommer de la couche `source`.
var nomsPermisDeLaSource = map[string]bool{"Film": true}

// accesAuxOctets : les methodes du film charge qui rendent des octets ou des paquets, et le champ
// qui porte les octets d un paquet.
var accesAuxOctets = map[string]bool{"Chunk": true, "Packets": true, "Payload": true}

// exceptionDesFaits : UN paquet de la couche des faits qui lit encore des octets, tolere par
// DECISION ecrite, avec son critere de retrait.
type exceptionDesFaits struct {
	paquet, date, raison, retrait string
}

// exceptionsDesFaits : vide (cliquet), cf. l en-tete.
var exceptionsDesFaits []exceptionDesFaits

// lectureDesFaits : une lecture d octets dans un fichier de production de la couche des faits.
type lectureDesFaits struct{ fichier, detail string }

// lecturesDesFaits rend, par paquet (dossier relatif a `apps/go-api`), les lectures d octets de
// la couche des faits, et le nombre de fichiers de production parcourus.
func lecturesDesFaits(t *testing.T) (map[string][]lectureDesFaits, int) {
	t.Helper()
	racine := racineGoAPI(t)
	out := map[string][]lectureDesFaits{}
	fichiers := 0
	fset := token.NewFileSet()
	err := filepath.WalkDir(filepath.Join(racine, filepath.FromSlash(racineDesFaits)),
		func(chemin string, d os.DirEntry, errMarche error) error {
			if errMarche != nil {
				return errMarche
			}
			if d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fset, chemin, nil, 0)
			if err != nil {
				t.Errorf("parse de %s : %v", chemin, err)
				return nil
			}
			fichiers++
			rel, _ := filepath.Rel(racine, chemin)
			rel = filepath.ToSlash(rel)
			for _, detail := range lecturesDuFichier(f) {
				out[filepath.ToSlash(filepath.Dir(rel))] = append(out[filepath.ToSlash(filepath.Dir(rel))],
					lectureDesFaits{fichier: rel, detail: detail})
			}
			return nil
		})
	if err != nil {
		t.Fatalf("parcours de %s : %v", racineDesFaits, err)
	}
	return out, fichiers
}

// lecturesDuFichier rend les lectures d octets d un fichier : un nom de la couche `source` autre
// que le type du film, un acces aux octets ou aux paquets.
func lecturesDuFichier(f *ast.File) []string {
	local := ""
	for _, imp := range f.Imports {
		if chemin, _ := strconv.Unquote(imp.Path.Value); chemin == importDeLaSource {
			local = "source"
			if imp.Name != nil {
				local = imp.Name.Name
			}
		}
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && local != "" && id.Name == local {
			if !nomsPermisDeLaSource[sel.Sel.Name] {
				out = append(out, local+"."+sel.Sel.Name)
			}
			return true
		}
		if accesAuxOctets[sel.Sel.Name] {
			out = append(out, "."+sel.Sel.Name)
		}
		return true
	})
	return out
}

// exceptionDuPaquet rend l exception qui couvre un paquet, nil sinon.
func exceptionDuPaquet(paquet string) *exceptionDesFaits {
	for i := range exceptionsDesFaits {
		if exceptionsDesFaits[i].paquet == paquet {
			return &exceptionsDesFaits[i]
		}
	}
	return nil
}

// TestLaCoucheDesFaitsNeLitAucunOctet : hors des exceptions datees, aucun fichier de production de
// la couche des faits ne lit d octet de film.
func TestLaCoucheDesFaitsNeLitAucunOctet(t *testing.T) {
	lectures, fichiers := lecturesDesFaits(t)
	if fichiers < plancherFichiersDesFaits {
		t.Fatalf("balayage muet : %d fichiers de production parcourus, plancher %d — la racine a "+
			"bouge ou le filtre est casse, et ce ratchet ne garde plus rien.", fichiers, plancherFichiersDesFaits)
	}
	var lignes []string
	for paquet, ls := range lectures {
		if exceptionDuPaquet(paquet) != nil {
			continue
		}
		for _, l := range ls {
			lignes = append(lignes, l.fichier+" : "+l.detail)
		}
	}
	if len(lignes) == 0 {
		return
	}
	sort.Strings(lignes)
	t.Errorf("%d lecture(s) d octets dans la couche des faits (ADR 0037, D-2 amende) :\n  %s\n"+
		"QUOI FAIRE : la lecture descend dans la grammaire (`film/internal/grammar`, une feuille comme "+
		"`grammar/signaux` si des tests de `grammar` importent le paquet), et le fait la consomme. "+
		"Le fait ne recoit du film que son type, pour le passer a la grammaire.",
		len(lignes), strings.Join(lignes, "\n  "))
}

// TestExceptionsDesFaitsSontDateesEtVivantes : chaque exception porte sa date, sa raison et son
// critere de retrait, et couvre une lecture REELLE — une exception perimee rougit.
func TestExceptionsDesFaitsSontDateesEtVivantes(t *testing.T) {
	lectures, _ := lecturesDesFaits(t)
	for _, e := range exceptionsDesFaits {
		if strings.TrimSpace(e.date) == "" || strings.TrimSpace(e.raison) == "" ||
			strings.TrimSpace(e.retrait) == "" {
			t.Errorf("exception %s : date, raison ET critere de retrait sont obligatoires", e.paquet)
		}
		if len(lectures[e.paquet]) == 0 {
			t.Errorf("exception PERIMEE %s : le paquet ne lit plus d octet. Critere de retrait atteint "+
				"— retirer la ligne.", e.paquet)
		}
	}
}
