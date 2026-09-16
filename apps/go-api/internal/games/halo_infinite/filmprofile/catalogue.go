// Package filmprofile LIT le catalogue des profils de film d un titre, HORS du decodeur.
//
// # POURQUOI CE PAQUET EST DEHORS
//
// Le profil d un film — ce que le depot sait de sa grammaire — vivait jusqu ici en DUR dans
// `grammar` (table du lot 2.1, `filmdec/profile_table.go`). Le lot 3.1 en fait une DONNEE :
// un fichier de reference versionne, `data/titles/{slug}/reference/film_profiles.json`, resolu
// par `title.PathResolver.FilmProfilesPath` (D12 du plan decodeur). Ce paquet-ci est le seul
// lecteur de ce fichier, et il N IMPORTE AUCUN paquet du film : un consommateur qui n a pas le
// decodeur (un outil, une verification de livraison, un futur titre) doit pouvoir dire ce que
// le depot sait d un build sans embarquer la lecture des bits.
//
// # LA CLE EST CE QUE LE FILM ECRIT, ET IL Y EN A TROIS
//
// Fait tranche par l utilisateur le 2026-09-16 : le film est AUTOPORTANT. Un profil n est donc
// jamais un « reglage par build » exterieur qui refuserait un film, c est une TABLE indexee par
// ce que le film PORTE, et le depot en connait trois clefs (cf. l en-tete de
// `filmdec/profile_table.go`, qui reste la source de verite du CONTENU) :
//
//	format=<n>     version de format, `chunk_00+4` — la grammaire des bits ;
//	build=<id>     chaine en clair de la section 2 — la taille des structures de contenu ;
//	majeure<=|>=|=<n>  version majeure, `chunk_00+0` — l implantation du gamertag ;
//	toutes         la valeur ne depend d aucune clef ecrite (invariant du depot).
//
// # DEUX NATURES D ENTREES DANS LE MEME FICHIER
//
//	DERIVEE  ce qui se deduit des FICHIERS DU JEU. Les bornes de carte ont deja leur catalogue
//	         (`map_quant_bounds.json`, produit par `cmd/mapquant-build`) : le profil ne les
//	         RECOPIE pas, il en porte l EMPREINTE ([Derive]) — de quelle derivation il est
//	         solidaire. `cmd/film-profiles-build` produit ce bloc, rien d autre.
//	SAISIE   ce qui vient de l EXECUTABLE ou d un film temoin. Chaque ligne porte sa
//	         provenance (relue / mesuree / presumee), sa preuve et sa date : une valeur sans
//	         provenance n entre pas au catalogue (D3 du plan, D-3 d ADR 0034).
package filmprofile

import (
	"fmt"
	"strconv"
	"strings"
)

// SchemaVersionCourante : la version de schema que ce paquet sait lire. Une version
// differente est REFUSEE — un catalogue de profil lu a moitie vaut moins que pas de catalogue.
const SchemaVersionCourante = 1

// Provenance dit d ou vient une valeur du catalogue.
//
// Les trois valeurs sont EXACTEMENT celles de la table du lot 2.1
// (`profile.Provenance`) : le catalogue est la meme donnee, deplacee dans un fichier. Le test
// de conformite ([TestCatalogueConformeALaTableDuLot21]) tient les deux vocabulaires egaux tant
// qu ils coexistent ; le lot 3.1.1 supprime la copie en faisant lire ce paquet a `grammar`.
type Provenance string

// Les trois provenances, et rien d autre.
const (
	// ProvenanceRelue : l executable est ouvert et la fonction est citee (Ghidra).
	ProvenanceRelue Provenance = "relue"
	// ProvenanceMesuree : la valeur est mesuree sur un film temoin par un oracle interne.
	ProvenanceMesuree Provenance = "mesuree"
	// ProvenancePresumee : ni relue ni mesuree — le registre de ce qui reste a etablir.
	ProvenancePresumee Provenance = "presumee"
)

// EstConnue dit si la provenance est l une des trois. Une quatrieme n existe pas.
func (p Provenance) EstConnue() bool {
	return p == ProvenanceRelue || p == ProvenanceMesuree || p == ProvenancePresumee
}

// SorteDeCle nomme LAQUELLE des trois clefs ecrites selectionne une entree.
type SorteDeCle string

// Les quatre sortes : les trois clefs du film, plus l invariant.
const (
	SorteFormat  SorteDeCle = "format"
	SorteBuild   SorteDeCle = "build"
	SorteMajeure SorteDeCle = "majeure"
	SorteToutes  SorteDeCle = "toutes"
)

// Comparateur : la forme d une clef de version majeure (`majeure<=38`, `majeure=39,40`).
type Comparateur string

// Les trois formes ecrites dans le catalogue.
const (
	ComparateurEgal   Comparateur = "="
	ComparateurAuPlus Comparateur = "<="
	ComparateurAuMoin Comparateur = ">="
)

// Les prefixes ecrits dans le fichier. Nommes une fois : ils servent a l analyse ET aux
// messages d erreur, et une troisieme copie divergerait (regle 6 du depot).
const (
	prefixeToutes        = "toutes"
	prefixeFormat        = "format="
	prefixeBuild         = "build="
	prefixeMajeureAuPlus = "majeure<="
	prefixeMajeureAuMoin = "majeure>="
	prefixeMajeureEgal   = "majeure="
)

// ClesLues sont les trois clefs telles qu elles ont ete LUES dans un film donne.
//
// `Majeure` a 0 pour « non lue » (film sans registre), comme le fait deja
// `grammar.implantationDuGamertag` : c est un etat, pas une valeur manquante a deviner.
type ClesLues struct {
	Format  int
	Build   string
	Majeure int
}

// Cle est une clef de catalogue ANALYSEE : sa sorte, ce qu elle selectionne, et son ecriture
// d origine (conservee pour les messages et pour le rendu a l identique).
type Cle struct {
	Sorte       SorteDeCle
	Brut        string
	Formats     []int
	Build       string
	Majeures    []int
	Comparateur Comparateur
}

// ParseCle analyse une clef ecrite du catalogue.
//
// Elle REFUSE ce qu elle ne comprend pas : une clef inconnue selectionnerait zero film en
// silence, c est-a-dire une valeur de profil qui ne s applique jamais sans que rien ne le dise.
func ParseCle(brut string) (Cle, error) {
	s := strings.TrimSpace(brut)
	switch {
	case s == prefixeToutes:
		return Cle{Sorte: SorteToutes, Brut: s}, nil
	case strings.HasPrefix(s, prefixeFormat):
		nums, err := listeDEntiers(strings.TrimPrefix(s, prefixeFormat))
		if err != nil {
			return Cle{}, fmt.Errorf("clef %q : %w", brut, err)
		}
		return Cle{Sorte: SorteFormat, Brut: s, Formats: nums}, nil
	case strings.HasPrefix(s, prefixeBuild):
		id := strings.TrimSpace(strings.TrimPrefix(s, prefixeBuild))
		if id == "" {
			return Cle{}, fmt.Errorf("clef %q : build vide", brut)
		}
		return Cle{Sorte: SorteBuild, Brut: s, Build: id}, nil
	case strings.HasPrefix(s, prefixeMajeureAuPlus):
		return cleMajeure(s, prefixeMajeureAuPlus, ComparateurAuPlus, brut)
	case strings.HasPrefix(s, prefixeMajeureAuMoin):
		return cleMajeure(s, prefixeMajeureAuMoin, ComparateurAuMoin, brut)
	case strings.HasPrefix(s, prefixeMajeureEgal):
		return cleMajeure(s, prefixeMajeureEgal, ComparateurEgal, brut)
	}
	return Cle{}, fmt.Errorf("clef %q : ni %q, ni %q…, ni %q…, ni %q… — le catalogue n a que "+
		"les trois clefs que le film ecrit", brut, prefixeToutes, prefixeFormat, prefixeBuild,
		prefixeMajeureEgal)
}

// cleMajeure analyse une clef de version majeure, comparateur deja identifie.
func cleMajeure(s, prefixe string, cmp Comparateur, brut string) (Cle, error) {
	nums, err := listeDEntiers(strings.TrimPrefix(s, prefixe))
	if err != nil {
		return Cle{}, fmt.Errorf("clef %q : %w", brut, err)
	}
	if cmp != ComparateurEgal && len(nums) != 1 {
		return Cle{}, fmt.Errorf("clef %q : un comparateur %q borne UNE valeur, pas %d",
			brut, cmp, len(nums))
	}
	return Cle{Sorte: SorteMajeure, Brut: s, Majeures: nums, Comparateur: cmp}, nil
}

// listeDEntiers analyse `27` ou `20,21,24,25`.
func listeDEntiers(reste string) ([]int, error) {
	champs := strings.Split(reste, ",")
	out := make([]int, 0, len(champs))
	for _, c := range champs {
		n, err := strconv.Atoi(strings.TrimSpace(c))
		if err != nil {
			return nil, fmt.Errorf("%q n est pas un entier", strings.TrimSpace(c))
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("liste d entiers vide")
	}
	return out, nil
}

// Correspond dit si cette clef selectionne un film dont les trois clefs ont ete lues.
func (c Cle) Correspond(lues ClesLues) bool {
	switch c.Sorte {
	case SorteToutes:
		return true
	case SorteFormat:
		return contientEntier(c.Formats, lues.Format)
	case SorteBuild:
		return c.Build == lues.Build
	case SorteMajeure:
		return c.correspondMajeure(lues.Majeure)
	}
	return false
}

func (c Cle) correspondMajeure(majeure int) bool {
	switch c.Comparateur {
	case ComparateurAuPlus:
		return majeure <= c.Majeures[0]
	case ComparateurAuMoin:
		return majeure >= c.Majeures[0]
	case ComparateurEgal:
		return contientEntier(c.Majeures, majeure)
	}
	return false
}

func contientEntier(liste []int, v int) bool {
	for _, n := range liste {
		if n == v {
			return true
		}
	}
	return false
}

// Entree est UNE ligne du catalogue : une valeur, sa clef, sa provenance et sa preuve. Les
// champs sont ceux de `profile.LigneProfil`, un pour un.
type Entree struct {
	// Cle est la clef ECRITE qui selectionne cette entree (`format=27`, `build=HI_1_13_0`,
	// `majeure>=41`, `toutes`).
	Cle string `json:"key"`
	// Champ nomme ce que l entree pose, dans le vocabulaire du profil du decodeur.
	Champ string `json:"field"`
	// Valeur est la valeur posee, ecrite pour etre lue par un humain.
	Valeur string `json:"value"`
	// Source est la provenance.
	Source Provenance `json:"provenance"`
	// Preuve est la fonction Ghidra, le film temoin et son oracle, ou ce qui manque.
	Preuve string `json:"proof"`
	// Date est le jour ou cette provenance a ete etablie, `AAAA-MM-JJ`.
	Date string `json:"date"`
}

// Derive est l EMPREINTE de la part du profil qui se derive des fichiers du jeu.
//
// Elle ne porte pas les bornes : elle dit DE QUELLE derivation ce catalogue est solidaire, pour
// qu une divergence (jeu mis a jour, catalogue de bornes regenere) se voie au lieu de se
// deviner. Le champ `Source` du catalogue de bornes n entre PAS dans l empreinte : il porte le
// chemin d installation de la machine qui l a produit, qui n est pas une donnee du jeu.
type Derive struct {
	// Catalogue est le NOM du fichier derive dont on porte l empreinte.
	Catalogue string `json:"catalogue"`
	// SchemaVersion est la version de schema de ce catalogue derive.
	SchemaVersion int `json:"schemaVersion"`
	// Cartes est le nombre d entrees de carte qu il porte.
	Cartes int `json:"maps"`
	// EmpreinteCartes est le sha256 de ses entrees de carte, hors champs de machine.
	EmpreinteCartes string `json:"mapsSha256"`
	// Outil est la commande qui PRODUIT le fichier derive.
	Outil string `json:"tool"`
}

// Catalogue est le fichier entier.
type Catalogue struct {
	// SchemaVersion doit valoir [SchemaVersionCourante].
	SchemaVersion int `json:"schemaVersion"`
	// TitleSlug est le titre auquel ce catalogue appartient (isolation par titre, ADR 0008).
	TitleSlug string `json:"titleSlug"`
	// APropos dit, dans le fichier lui-meme, ce qu il est et ou se trouve sa procedure.
	APropos string `json:"about"`
	// Derive est la part fabriquee (cf. [Derive]).
	Derive Derive `json:"derived"`
	// Entrees est la part saisie, dans l ordre de la table du lot 2.1.
	Entrees []Entree `json:"entries"`
}

// EntreesPour rend, dans l ordre du fichier, les entrees que les clefs lues selectionnent.
//
// Elle ne tranche RIEN : deux entrees peuvent porter le meme champ pour des clefs de sortes
// differentes, et c est au consommateur de dire laquelle prime. Un champ absent est un champ
// que le depot ne sait pas pour ce film — pas un defaut a inventer.
//
// Elle rend une ERREUR sur une clef illisible plutot que de la sauter : un catalogue dont une
// ligne ne selectionne jamais rien est un profil incomplet en silence. [Catalogue.Valide] a
// deja refuse ce cas a la lecture — cette erreur-ci couvre un catalogue construit en memoire.
func (c *Catalogue) EntreesPour(lues ClesLues) ([]Entree, error) {
	out := make([]Entree, 0, len(c.Entrees))
	for _, e := range c.Entrees {
		cle, err := ParseCle(e.Cle)
		if err != nil {
			return nil, fmt.Errorf("entree %q/%q : %w", e.Cle, e.Champ, err)
		}
		if cle.Correspond(lues) {
			out = append(out, e)
		}
	}
	return out, nil
}
