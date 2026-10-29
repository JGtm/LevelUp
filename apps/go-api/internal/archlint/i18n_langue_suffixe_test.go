// Package archlint — i18n_langue_suffixe_test.go : garde-rail i18n (ADR 0040, plan i18n étape 5).
//
// Aucun champ de struct, aucune fonction ni méthode ne porte un suffixe de langue (`LabelFR`,
// `nameEN`, `XxxFr`, `chooseEn`...) en dehors des tests : un libellé écrit dans plusieurs langues
// est un `i18n.Text`, choisi par `Pick(locale)`, et une fonction qui dépend de la langue la reçoit
// en paramètre. Un nom suffixé fige la liste des langues dans le code ; une 3e langue ne s'y
// ajoute pas.
//
// Trois régimes, dans cet ordre :
//
//   - les champs d'une struct SÉRIALISÉE (tag `json:"nom"`, autre que `-`) sont la classe (b) :
//     le contrat JSON de l'API ne porte plus de `*_fr` / `*_en`, ni en réponse ni en requête (les
//     corps d'écriture de traduction de l'administration sont des saisies `{ langue: texte }`). Ils
//     sont COMPTÉS dans un plafond (suffixeContratPlafond, à 0), avec les fonctions de pont JSON
//     marquées `i18n-contrat` dans leur commentaire de doc : toute réintroduction fait échouer le test ;
//   - les vrais faux positifs (un mot français qui finit par « en » : `suitLOrdreEn`) et les
//     identifiants qui nomment une colonne ou un fichier historique figurent dans suffixeAllowlist,
//     chacun daté et justifié ;
//   - tout le reste est une violation.
//
// Méthode : parcours AST (go/ast) des fichiers non-test sous cmd, internal, pkg, tools, scripts ;
// jamais une regex sur le texte, qui verrait aussi les commentaires et les chaînes.
package archlint

import (
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// suffixeContratPlafond est le nombre de champs sérialisés (classe b) et de fonctions de pont JSON
// tolérés : 0. Le contrat JSON de l'API ne porte aucun champ à suffixe de langue, ni en réponse
// (libellés résolus `label`, `name`...) ni en requête (les corps d'écriture de traduction de
// l'administration sont des saisies `{ langue: texte }`, item 8.6 du plan). La constante reste
// pour que le garde-rail signale une réintroduction avec son compte, sans allowlist.
const suffixeContratPlafond = 0

// marqueContrat, dans le commentaire de doc d'une fonction, la range dans le compteur de classe (b).
const marqueContrat = "i18n-contrat"

// suffixeLangueRE reconnaît un identifiant dont le dernier mot est un code de langue : `FR`, `EN`,
// `Fr`, `En`, précédé d'un caractère qui n'est pas une majuscule de mot (frontière camelCase :
// `LabelFR`, `nameEn`, `chooseFR`) ou formant tout l'identifiant (`En`, `FR`). `Screen`, `Open`,
// `Often` ne correspondent pas (la majuscule est requise).
var suffixeLangueRE = regexp.MustCompile(`(^|[a-z0-9_])(FR|EN|Fr|En)$`)

// suffixeRacines : racines balayées (chemins de garde-rail, [surDisque]) : go-api et la brique du
// décodeur de film, qui y vivait.
var suffixeRacines = []string{"cmd", "internal", "pkg", "tools", "scripts", racineBriqueFilm}

// suffixeFautif est une déclaration dont le nom porte un suffixe de langue.
type suffixeFautif struct {
	Fichier string
	Nom     string
	Genre   string // "champ", "champ-json", "fonction", "fonction-pont"
	Ligne   int
}

func (o suffixeFautif) String() string {
	return fmt.Sprintf("%s:%d %s %s", o.Fichier, o.Ligne, o.Genre, o.Nom)
}

// suffixeAutorise est une entrée de l'allowlist : une déclaration précise, datée et justifiée.
type suffixeAutorise struct {
	Fichier string
	Nom     string
	Date    string
	Raison  string
}

// suffixeAllowlist : exceptions. Ajouter une entrée exige une raison écrite et une date ; la liste
// ne sert jamais à repousser une conversion en i18n.Text.
var suffixeAllowlist = []suffixeAutorise{
	{
		Fichier: "internal/replayverite/violations_vies.go", Nom: "vivantEn", Date: "2026-10-26",
		Raison: "mot français : « vivant en » (état d'un acteur « vivant en <position> »), pas une langue",
	},
	{
		Fichier: "libs/film/internal/grammar/debut_de_liste.go", Nom: "enteteNeufEn",
		Date:   "2026-10-26",
		Raison: "mot français : « en-tête neuf en <position> » (lecture à une position), pas une langue",
	},
	{
		Fichier: "libs/film/internal/grammar/debut_de_liste.go", Nom: "suitLOrdreEn",
		Date:   "2026-10-26",
		Raison: "mot français : « suit l'ordre en <position> », pas une langue",
	},
	{
		Fichier: "internal/migration/steps_metadata_purge_weapons_name_fr.go", Nom: "applyPurgeWeaponsNameFR",
		Date: "2026-10-26",
		Raison: "migration historique : le nom est celui de la colonne `weapons.name_fr` qu'elle retire (le " +
			"fichier porte le même nom) ; elle disparaît avec la colonne, étape 8 du plan i18n",
	},
}

func estAutorise(o suffixeFautif) bool {
	for _, a := range suffixeAllowlist {
		if a.Fichier == o.Fichier && a.Nom == o.Nom {
			return true
		}
	}
	return false
}

// aTagJSON : le champ porte un tag `json:"nom"` autre que `-` (donc il est sérialisé).
func aTagJSON(f *ast.Field) bool {
	if f.Tag == nil {
		return false
	}
	v, ok := reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Lookup("json")
	return ok && v != "-" && !strings.HasPrefix(v, "-,")
}

// inventaireSuffixes liste les champs de struct, fonctions, méthodes et méthodes d'interface dont
// le nom porte un suffixe de langue, triés par fichier puis ligne.
func inventaireSuffixes(srcs []goSource) []suffixeFautif {
	var out []suffixeFautif
	for _, s := range srcs {
		add := func(id *ast.Ident, genre string) {
			if suffixeLangueRE.MatchString(id.Name) {
				out = append(out, suffixeFautif{s.Rel, id.Name, genre, s.Fset.Position(id.Pos()).Line})
			}
		}
		ast.Inspect(s.File, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.StructType:
				for _, f := range x.Fields.List {
					genre := "champ"
					if aTagJSON(f) {
						genre = "champ-json"
					}
					for _, id := range f.Names {
						add(id, genre)
					}
				}
			case *ast.FuncDecl:
				genre := "fonction"
				if x.Doc != nil && strings.Contains(x.Doc.Text(), marqueContrat) {
					genre = "fonction-pont"
				}
				add(x.Name, genre)
			case *ast.InterfaceType:
				for _, m := range x.Methods.List {
					for _, id := range m.Names {
						add(id, "fonction")
					}
				}
			}
			return true
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Fichier != out[j].Fichier {
			return out[i].Fichier < out[j].Fichier
		}
		return out[i].Ligne < out[j].Ligne
	})
	return out
}

// classerSuffixes sépare l'inventaire en violations (rien ne les couvre) et en déclarations de
// contrat comptées (classe b), l'allowlist étant déjà retirée.
func classerSuffixes(occ []suffixeFautif) (violations, contrat []suffixeFautif) {
	for _, o := range occ {
		switch {
		case estAutorise(o):
		case o.Genre == "champ-json" || o.Genre == "fonction-pont":
			contrat = append(contrat, o)
		default:
			violations = append(violations, o)
		}
	}
	return violations, contrat
}

func listeSuffixes(occ []suffixeFautif) string {
	var b strings.Builder
	for _, o := range occ {
		b.WriteString("\n  " + o.String())
	}
	return b.String()
}

func TestAucunSuffixeDeLangueHorsContrat(t *testing.T) {
	occ := inventaireSuffixes(goSourcesUnder(t, suffixeRacines...))
	violations, contrat := classerSuffixes(occ)
	if len(violations) > 0 {
		t.Errorf("%d déclaration(s) à suffixe de langue (champ, fonction ou méthode) : un libellé "+
			"multilingue est un i18n.Text (Pick(locale)), une fonction qui dépend de la langue reçoit "+
			"une i18n.Locale. Un vrai faux positif se justifie dans suffixeAllowlist (daté).%s",
			len(violations), listeSuffixes(violations))
	}
	if len(contrat) > suffixeContratPlafond {
		t.Errorf("%d champs sérialisés / ponts JSON à suffixe de langue (plafond gelé : %d, ADR 0040, "+
			"cible tenue à 0) : le contrat d'API ne peut pas gagner de champ `_fr` / `_en`.%s",
			len(contrat), suffixeContratPlafond, listeSuffixes(contrat))
	}
}

// Le plafond gelé est exactement la mesure : une baisse sans abaissement de la constante laisserait
// la marge se reconstituer en silence.
func TestSuffixeContratPlafondEstLaMesure(t *testing.T) {
	_, contrat := classerSuffixes(inventaireSuffixes(goSourcesUnder(t, suffixeRacines...)))
	if len(contrat) < suffixeContratPlafond {
		t.Errorf("%d champs sérialisés mesurés pour un plafond de %d : abaisser suffixeContratPlafond "+
			"à %d (le plafond ne doit jamais laisser de marge)", len(contrat), suffixeContratPlafond, len(contrat))
	}
}

// Chaque entrée de l'allowlist désigne encore une déclaration existante : une entrée orpheline
// (fichier renommé, nom retiré) autoriserait la prochaine déclaration de ce nom.
func TestSuffixeAllowlistSansEntreeOrpheline(t *testing.T) {
	vues := map[string]bool{}
	for _, o := range inventaireSuffixes(goSourcesUnder(t, suffixeRacines...)) {
		vues[o.Fichier+"|"+o.Nom] = true
	}
	for _, a := range suffixeAllowlist {
		if a.Date == "" || a.Raison == "" {
			t.Errorf("entrée %s %s : date et raison sont obligatoires", a.Fichier, a.Nom)
		}
		if !vues[a.Fichier+"|"+a.Nom] {
			t.Errorf("entrée %s %s : aucune déclaration de ce nom dans ce fichier, la retirer", a.Fichier, a.Nom)
		}
	}
}

// Auto-preuve : le parcours voit les formes visées (champ, champ sérialisé, fonction, méthode,
// méthode d'interface, pont marqué) et ignore les noms qui ne portent pas de langue.
func TestSuffixeLangueAutoPreuve(t *testing.T) {
	src := parseGoSnippet(t, "internal/x/x.go", `package x

type Row struct {
	LabelFR string
	NameEn  string `+"`json:\"name_en\"`"+`
	HiddenFR string `+"`json:\"-\"`"+`
	En, Fr  string
	Screen  string
	Often   int
	Label   string
}

type Port interface {
	LoadTranslationsFR() error
	Load() error
}

func chooseFR() {}

func (r Row) NameEN() string { return "" }

// pont : i18n-contrat
func pontFr() {}

func Open() {}
`)
	want := map[string]string{
		"LabelFR": "champ", "NameEn": "champ-json", "HiddenFR": "champ", "En": "champ", "Fr": "champ",
		"LoadTranslationsFR": "fonction", "chooseFR": "fonction", "NameEN": "fonction", "pontFr": "fonction-pont",
	}
	got := map[string]string{}
	for _, o := range inventaireSuffixes([]goSource{src}) {
		got[o.Nom] = o.Genre
	}
	for nom, genre := range want {
		if got[nom] != genre {
			t.Errorf("%s : genre %q, attendu %q", nom, got[nom], genre)
		}
	}
	for nom := range got {
		if _, ok := want[nom]; !ok {
			t.Errorf("%s ne devrait pas être vu", nom)
		}
	}
	violations, contrat := classerSuffixes(inventaireSuffixes([]goSource{src}))
	if len(contrat) != 2 || len(violations) != len(want)-2 {
		t.Errorf("classement : %d de contrat (attendu 2), %d violations (attendu %d)",
			len(contrat), len(violations), len(want)-2)
	}
}
