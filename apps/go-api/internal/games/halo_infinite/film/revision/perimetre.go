package revision

// perimetre.go — LE PERIMETRE D UNE COUCHE EST LA FERMETURE DE SES IMPORTS (lot J3.2 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DU-2 (b), constat SRC-1 de
// `.ai/AUDIT_DECODEUR_FILM_2026-09-24.md`).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// Jusqu au lot J3.2 chaque couche DECLARAIT a la main les racines qu elle hachait. La liste
// oubliait ce que la couche IMPORTE hors d elle-meme : `film/types` (les formes lues),
// `film/damagetag` (ses donnees embarquees et son seuil `Strong`), `games/weapons/filmshell`
// (le catalogue d armes du balayage) — autant de paquets qui DECIDENT la sortie sans qu aucune
// revision ne bouge quand ils changent. Des faits jugés frais sur un decodage perime, des lignes
// de kill hors backlog.
//
// # LA REGLE, EN TROIS LIGNES
//
//	partir     de tous les paquets de l ARBRE de la couche (sa racine et ses sous-dossiers) ;
//	suivre     leurs imports de PRODUCTION (`_test.go` exclus), bornes au MODULE — la
//	           bibliotheque standard et les dependances tierces ne sont pas des sources du depot ;
//	s arreter  a une AUTRE couche revisee : elle entre par sa VALEUR (sa revision), jamais par
//	           ses octets (ADR 0034 D-1, decision V15 (12)). Tout autre paquet rencontre entre par
//	           ses octets, et ses imports sont suivis a leur tour.
//
// La liste des VALEURS AMONT n est donc plus une declaration : c est ce que la fermeture
// rencontre. Une valeur fournie pour une couche que la fermeture ne rencontre pas est une
// ERREUR, comme une couche rencontree sans valeur — une declaration qui diverge du code doit
// echouer bruyamment, pas hacher une dependance qui n existe pas.
//
// Le perimetre calcule est FIGE par un golden par couche (`testdata/<couche>_perimetre.golden`,
// `TestPerimetreDeChaqueCoucheEgaleSonGolden`) : un import ajoute dans une couche le fait rougir,
// et la question « ce paquet decide-t-il la sortie de la couche ? » se pose a la relecture.

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Couche : une couche REVISEE, telle que la fermeture la reconnait — son nom et la racine de son
// arbre, relative au module et en slash.
type Couche struct {
	// Nom : le nom court de la couche (`grammar`), qui nomme aussi sa valeur amont.
	Nom string
	// Racine : le dossier racine de l arbre de la couche, relatif au module
	// (`internal/games/halo_infinite/film/internal/grammar`). Tout paquet sous cette racine est
	// de la couche.
	Racine string
}

// ErrPerimetre : la declaration d une couche et sa fermeture ne s accordent pas (couche inconnue,
// valeur amont manquante ou superflue). C est une faute de DECLARATION, jamais un resultat.
var ErrPerimetre = errors.New("perimetre de couche incoherent")

// Module : le module Go qui borne la fermeture — sa racine sur le disque et son chemin
// d import, lu dans `go.mod`.
type Module struct {
	// Racine : le dossier qui porte `go.mod`.
	Racine string
	// Chemin : le chemin du module (`levelup/go-api`).
	Chemin string
}

// Perimetre : ce que la fermeture d une couche rencontre.
type Perimetre struct {
	// Paquets : les dossiers haches par leurs OCTETS, relatifs au module, en slash, tries —
	// ceux de l arbre de la couche comme ceux qu elle importe.
	Paquets []string
	// Amonts : les couches revisees rencontrees, qui entrent par leur VALEUR, dans l ORDRE de la
	// liste des couches (l ordre fait partie du contrat de l empreinte).
	Amonts []string
}

// ModuleDe remonte depuis `dossier` jusqu au premier `go.mod` et rend le module qu il declare.
func ModuleDe(dossier string) (Module, error) {
	dir, err := filepath.Abs(dossier)
	if err != nil {
		return Module{}, err
	}
	for {
		blob, errLire := os.ReadFile(filepath.Join(dir, "go.mod")) //nolint:gosec // remontee depuis un dossier fourni par l appelant
		if errLire == nil {
			chemin := cheminDuModule(string(blob))
			if chemin == "" {
				return Module{}, fmt.Errorf("%w : %s/go.mod sans directive `module`", ErrPerimetre, dir)
			}
			return Module{Racine: dir, Chemin: chemin}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return Module{}, fmt.Errorf("%w : aucun go.mod au-dessus de %s", ErrPerimetre, dossier)
		}
		dir = parent
	}
}

// cheminDuModule lit la directive `module` d un `go.mod`.
func cheminDuModule(gomod string) string {
	for _, ligne := range strings.Split(strings.ReplaceAll(gomod, "\r\n", "\n"), "\n") {
		if reste, ok := strings.CutPrefix(strings.TrimSpace(ligne), "module "); ok {
			return strings.Trim(strings.TrimSpace(reste), `"`)
		}
	}
	return ""
}

// Fermeture calcule le perimetre de la couche `nom` parmi `couches`.
func (m Module) Fermeture(nom string, couches []Couche) (Perimetre, error) {
	propre, ok := coucheNommee(nom, couches)
	if !ok {
		return Perimetre{}, fmt.Errorf("%w : couche %q absente de la liste des couches", ErrPerimetre, nom)
	}
	file, err := m.paquetsDeLArbre(propre.Racine)
	if err != nil {
		return Perimetre{}, err
	}
	if len(file) == 0 {
		return Perimetre{}, fmt.Errorf("%w : %s", ErrRacineSansSource, propre.Racine)
	}
	vus := map[string]bool{}
	for _, p := range file {
		vus[p] = true
	}
	amonts := map[string]bool{}
	for i := 0; i < len(file); i++ {
		imports, err := m.importsDuPaquet(file[i])
		if err != nil {
			return Perimetre{}, err
		}
		for _, rel := range imports {
			if autre := coucheDuPaquet(rel, couches); autre != "" {
				if autre != nom {
					amonts[autre] = true
				}
				continue
			}
			if !vus[rel] {
				vus[rel] = true
				file = append(file, rel)
			}
		}
	}
	p := Perimetre{Paquets: file}
	slices.Sort(p.Paquets)
	for _, c := range couches {
		if amonts[c.Nom] {
			p.Amonts = append(p.Amonts, c.Nom)
		}
	}
	return p, nil
}

// coucheNommee rend la couche de ce nom.
func coucheNommee(nom string, couches []Couche) (Couche, bool) {
	for _, c := range couches {
		if c.Nom == nom {
			return c, true
		}
	}
	return Couche{}, false
}

// coucheDuPaquet rend le nom de la couche dont l arbre contient `rel`, ou "".
func coucheDuPaquet(rel string, couches []Couche) string {
	for _, c := range couches {
		if rel == c.Racine || strings.HasPrefix(rel, c.Racine+"/") {
			return c.Nom
		}
	}
	return ""
}

// paquetsDeLArbre rend les dossiers de l arbre `racine` qui portent au moins une source `.go`
// de production — les paquets de la couche. `testdata/` est ecarte, comme au hachage.
func (m Module) paquetsDeLArbre(racine string) ([]string, error) {
	var out []string
	base := filepath.Join(m.Racine, filepath.FromSlash(racine))
	err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, errMarche error) error {
		if errMarche != nil {
			return errMarche
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "testdata" {
			return filepath.SkipDir
		}
		sources, err := sourcesDuDossier(chemin)
		if err != nil {
			return err
		}
		if len(sources) > 0 {
			rel, errRel := filepath.Rel(m.Racine, chemin)
			if errRel != nil {
				return errRel
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	return out, err
}

// sourcesDuDossier rend les noms des sources `.go` de production d UN dossier, sans descendre.
func sourcesDuDossier(dossier string) ([]string, error) {
	entrees, err := os.ReadDir(dossier)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entrees {
		if nom := e.Name(); !e.IsDir() && estSourceDeProduction(nom) {
			out = append(out, nom)
		}
	}
	return out, nil
}

// estSourceDeProduction : un fichier `.go` qui n est pas un test.
func estSourceDeProduction(nom string) bool {
	return strings.HasSuffix(nom, ".go") && !strings.HasSuffix(nom, "_test.go")
}

// importsDuPaquet rend les imports DU MODULE du paquet `rel`, relatifs au module, sans doublon.
//
// TOUTES LES SOURCES DE PRODUCTION SONT LUES, contraintes de construction comprises : un fichier
// `//go:build research` qui importe un paquet le fait entrer dans le perimetre. C est le choix
// prudent — l empreinte hache deja ces fichiers, quel que soit leur tag.
func (m Module) importsDuPaquet(rel string) ([]string, error) {
	dossier := filepath.Join(m.Racine, filepath.FromSlash(rel))
	noms, err := sourcesDuDossier(dossier)
	if err != nil {
		return nil, err
	}
	prefixe := m.Chemin + "/"
	vus := map[string]bool{}
	var out []string
	for _, nom := range noms {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dossier, nom), nil, parser.ImportsOnly)
		if err != nil {
			return nil, fmt.Errorf("imports de %s/%s : %w", rel, nom, err)
		}
		for _, imp := range f.Imports {
			chemin, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return nil, fmt.Errorf("imports de %s/%s : %w", rel, nom, err)
			}
			if cible, ok := strings.CutPrefix(chemin, prefixe); ok && !vus[cible] {
				vus[cible] = true
				out = append(out, cible)
			}
		}
	}
	return out, nil
}

// CalculerCouche rend l empreinte de la couche `nom` sous son perimetre, et ce perimetre.
//
// # CE QUI ENTRE DANS LE HACHAGE, DANS L ORDRE
//
//  1. Les VALEURS des couches amont rencontrees, dans l ordre de `couches` ([empreinteur], meme
//     cadre `amont:<longueur>`). `valeurs` doit porter EXACTEMENT ces couches-la.
//  2. L ARBRE de la couche, chemins relatifs a sa racine, `exclure` applique — le cadre des
//     lots precedents, qui survit a un `git mv` de la couche entiere.
//  3. Chaque paquet IMPORTE qui n est pas une couche, dans l ordre de son chemin, SANS descendre
//     dans ses sous-dossiers (un sous-paquet n entre que s il est importe lui-meme), chemins
//     relatifs au MODULE : le paquet nomme ce qu il apporte.
func (m Module) CalculerCouche(nom string, couches []Couche, exclure func(rel string) bool,
	valeurs map[string]string,
) (Resultat, Perimetre, error) {
	p, err := m.Fermeture(nom, couches)
	if err != nil {
		return Resultat{}, Perimetre{}, err
	}
	amonts, err := valeursDesAmonts(nom, p.Amonts, valeurs)
	if err != nil {
		return Resultat{}, Perimetre{}, err
	}
	propre, _ := coucheNommee(nom, couches)
	e := nouvelEmpreinteur(amonts)
	if err := e.arbre(filepath.Join(m.Racine, filepath.FromSlash(propre.Racine)), exclure); err != nil {
		return Resultat{}, Perimetre{}, err
	}
	for _, paquet := range p.Paquets {
		if coucheDuPaquet(paquet, couches) == nom {
			continue
		}
		if err := e.paquet(m.Racine, paquet); err != nil {
			return Resultat{}, Perimetre{}, err
		}
	}
	return e.resultat(), p, nil
}

// valeursDesAmonts rend les valeurs des couches rencontrees, dans l ordre du perimetre, ou
// l ecart entre ce que la couche fournit et ce que sa fermeture rencontre.
func valeursDesAmonts(nom string, rencontrees []string, valeurs map[string]string) ([]string, error) {
	out := make([]string, 0, len(rencontrees))
	for _, a := range rencontrees {
		v, ok := valeurs[a]
		if !ok || v == "" {
			return nil, fmt.Errorf("%w : la couche %s importe la couche %s, dont la valeur n est pas "+
				"fournie", ErrPerimetre, nom, a)
		}
		out = append(out, v)
	}
	for a := range valeurs {
		if !slices.Contains(rencontrees, a) {
			return nil, fmt.Errorf("%w : valeur amont fournie pour %s, que la fermeture de %s ne "+
				"rencontre pas (rencontrees : %v)", ErrPerimetre, a, nom, rencontrees)
		}
	}
	return out, nil
}
