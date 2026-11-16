package mappings

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// loader_spawn_families.go — table MODE DE JEU -> FAMILLE du modele de reapparition
// (config/titles/{slug}/mappings/spawn_families.toml).
//
// POURQUOI EN DONNEE : quelles familles de modes ont leurs propres poids, et quels noms de variante
// les composent, est un savoir DU TITRE (ADR 0011). Un titre sans fichier n'a pas de famille :
// l'appelant retombe sur la famille « tous » des poids. L'absence du fichier remonte telle quelle
// (os.IsNotExist), jamais maquillee en table vide.

// SpawnFamilyAll est le nom reserve de la famille qui regroupe toutes les autres : aucune entree du
// fichier ne peut le porter. C'est la valeur de domain.FamilleToutesModes (ce package n'importe pas
// domain ; l'egalite est tenue par un test).
const SpawnFamilyAll = "tous"

// SpawnFamily est une famille de modes : les jetons qui reconnaissent un match.
type SpawnFamily struct {
	ID string
	// VariantContains : sous-chaines (minuscules) du nom de variante de jeu.
	VariantContains []string
	// CategoryContains : sous-chaines (minuscules) de la categorie de mode.
	CategoryContains []string
	// CountdownForecast : la prevision du compte a rebours du rejeu 2D couvre cette famille (cf. l'en-tete du
	// fichier de donnees).
	CountdownForecast bool
}

// SpawnFamilyExclusions est ce qui sort un match de toute famille.
type SpawnFamilyExclusions struct {
	VariantPrefixes  []string
	Categories       []string
	CategoryContains []string
}

// SpawnFamilySet porte la table d'un titre, familles dans l'ordre du fichier.
type SpawnFamilySet struct {
	titleSlug     string
	schemaVersion int
	exclusions    SpawnFamilyExclusions
	families      []SpawnFamily
}

type spawnFamiliesTOML struct {
	Meta       metaSection `toml:"meta"`
	Exclusions struct {
		VariantPrefixes  []string `toml:"variant_prefixes"`
		Categories       []string `toml:"categories"`
		CategoryContains []string `toml:"category_contains"`
	} `toml:"exclusions"`
	Families []struct {
		ID                string   `toml:"id"`
		CountdownForecast bool     `toml:"countdown_forecast"`
		VariantContains   []string `toml:"variant_contains"`
		CategoryContains  []string `toml:"category_contains"`
	} `toml:"families"`
}

// FamilyOf rend la famille d'un match d'apres sa variante de jeu et sa categorie de mode, ou "" si
// le match est exclu ou ne ressemble a aucune famille. Insensible a la casse et aux espaces de bord.
// Un ensemble nil (titre sans fichier) ne reconnait aucune famille.
func (s *SpawnFamilySet) FamilyOf(variant, category string) string {
	if s == nil {
		return ""
	}
	v, c := strings.ToLower(strings.TrimSpace(variant)), strings.ToLower(strings.TrimSpace(category))
	if s.exclut(v, c) {
		return ""
	}
	for _, f := range s.families {
		if contientUn(v, f.VariantContains) || contientUn(c, f.CategoryContains) {
			return f.ID
		}
	}
	return ""
}

// CountdownForecast dit si la prevision du compte a rebours couvre la famille. Une famille inconnue, ou un
// ensemble nil, n'est pas couverte.
func (s *SpawnFamilySet) CountdownForecast(id string) bool {
	if s == nil {
		return false
	}
	for _, f := range s.families {
		if f.ID == id {
			return f.CountdownForecast
		}
	}
	return false
}

func (s *SpawnFamilySet) exclut(v, c string) bool {
	for _, p := range s.exclusions.VariantPrefixes {
		if strings.HasPrefix(v, p) {
			return true
		}
	}
	for _, x := range s.exclusions.Categories {
		if c == x {
			return true
		}
	}
	return contientUn(c, s.exclusions.CategoryContains)
}

func contientUn(texte string, jetons []string) bool {
	for _, j := range jetons {
		if strings.Contains(texte, j) {
			return true
		}
	}
	return false
}

// Families rend les identifiants des familles dans l'ordre du fichier (copie).
func (s *SpawnFamilySet) Families() []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s.families))
	for i, f := range s.families {
		out[i] = f.ID
	}
	return out
}

// TitleSlug retourne le slug declare.
func (s *SpawnFamilySet) TitleSlug() string {
	if s == nil {
		return ""
	}
	return s.titleSlug
}

// SchemaVersion retourne la version de schema declaree.
func (s *SpawnFamilySet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadSpawnFamiliesFromFile lit et valide un spawn_families.toml a un chemin donne.
func LoadSpawnFamiliesFromFile(path string) (*SpawnFamilySet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadSpawnFamiliesFromBytes(path, raw)
}

// LoadSpawnFamiliesFromBytes parse et valide un payload TOML deja en memoire. Validation STRICTE
// (tout-ou-rien) : une famille sans jeton ne reconnaitrait rien, un identifiant double ou reserve
// rendrait le choix ambigu.
func LoadSpawnFamiliesFromBytes(path string, raw []byte) (*SpawnFamilySet, error) {
	var doc spawnFamiliesTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	if len(doc.Families) == 0 {
		return nil, fmt.Errorf("%s: aucune entrée [[families]] — un titre sans famille n'a pas ce fichier", path)
	}
	set := &SpawnFamilySet{
		titleSlug: doc.Meta.TitleSlug, schemaVersion: doc.Meta.SchemaVersion,
		exclusions: SpawnFamilyExclusions{
			VariantPrefixes:  minuscules(doc.Exclusions.VariantPrefixes),
			Categories:       minuscules(doc.Exclusions.Categories),
			CategoryContains: minuscules(doc.Exclusions.CategoryContains),
		},
	}
	vus := map[string]bool{}
	for i, e := range doc.Families {
		id := strings.TrimSpace(e.ID)
		switch {
		case id == "":
			return nil, fmt.Errorf("%s: [[families]] #%d : id vide", path, i)
		case id == SpawnFamilyAll:
			return nil, fmt.Errorf("%s: [[families]] #%d : %q est le nom réservé de la famille qui les regroupe toutes", path, i, id)
		case vus[id]:
			return nil, fmt.Errorf("%s: [[families]] #%d : id %q déjà déclaré", path, i, id)
		}
		f := SpawnFamily{
			ID: id, VariantContains: minuscules(e.VariantContains), CategoryContains: minuscules(e.CategoryContains),
			CountdownForecast: e.CountdownForecast,
		}
		if len(f.VariantContains)+len(f.CategoryContains) == 0 {
			return nil, fmt.Errorf("%s: [[families]] #%d (%s) : aucun jeton (la famille ne reconnaîtrait aucun match)", path, i, id)
		}
		vus[id] = true
		set.families = append(set.families, f)
	}
	return set, nil
}

// minuscules normalise des jetons : espaces de bord retires, minuscules, jetons vides ecartes.
func minuscules(jetons []string) []string {
	var out []string
	for _, j := range jetons {
		if j = strings.ToLower(strings.TrimSpace(j)); j != "" {
			out = append(out, j)
		}
	}
	return out
}
