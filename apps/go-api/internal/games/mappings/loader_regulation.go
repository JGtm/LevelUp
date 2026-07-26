package mappings

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// RegulationSet porte le temps RÉGLEMENTAIRE par variante de jeu d'un titre
// (data-driven, config/titles/{slug}/mappings/regulation.toml). C'est la source
// unique du seuil qui sépare un match terminé dans le temps d'un match parti en
// PROLONGATION (cf. analysis.ComputeOvertime).
//
// La clé est `match_registry.game_variant_name` = un NOM D'ASSET UGC, PAS un
// identifiant stable : il bouge au fil des saisons du titre. Une variante
// inconnue n'a donc pas de temps réglementaire → aucun flag (dégradation sûre,
// jamais de faux positif). Fichier absent = table vide, jamais une erreur.
type RegulationSet struct {
	titleSlug     string
	schemaVersion int
	// seconds : game_variant_name → temps réglementaire en secondes (> 0).
	seconds map[string]int
}

// regulationTOML — projection brute de regulation.toml.
type regulationTOML struct {
	Meta    metaSection    `toml:"meta"`
	Seconds map[string]int `toml:"regulation_seconds"`
}

// Seconds retourne le temps réglementaire de la variante et true s'il est connu.
// nil-safe et variante inconnue → (0, false) : l'appelant ne flague pas.
func (s *RegulationSet) Seconds(gameVariantName string) (int, bool) {
	if s == nil {
		return 0, false
	}
	v, ok := s.seconds[strings.TrimSpace(gameVariantName)]
	return v, ok
}

// SecondsMap retourne une copie de la table variante → secondes. nil-safe (map
// vide). Utilisé par le wiring pour injecter la table dans les services sans
// exposer le type interne.
func (s *RegulationSet) SecondsMap() map[string]int {
	out := make(map[string]int)
	if s == nil {
		return out
	}
	for k, v := range s.seconds {
		out[k] = v
	}
	return out
}

// TitleSlug retourne le slug déclaré.
func (s *RegulationSet) TitleSlug() string {
	if s == nil {
		return ""
	}
	return s.titleSlug
}

// SchemaVersion retourne la version de schéma déclarée.
func (s *RegulationSet) SchemaVersion() int {
	if s == nil {
		return 0
	}
	return s.schemaVersion
}

// LoadRegulationFromFile lit et valide un regulation.toml à un chemin donné.
func LoadRegulationFromFile(path string) (*RegulationSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadRegulationFromBytes(path, raw)
}

// LoadRegulationFromBytes parse et valide un payload TOML déjà en mémoire.
// Validation stricte : title_slug + schema_version obligatoires ; chaque entrée
// a une clé non vide et une valeur > 0. Une table VIDE est VALIDE (titre sans
// temps réglementaire mesuré, ex. Halo 5 → aucun flag).
func LoadRegulationFromBytes(path string, raw []byte) (*RegulationSet, error) {
	var doc regulationTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.Meta.TitleSlug == "" {
		return nil, fmt.Errorf("%s: [meta].title_slug manquant", path)
	}
	if doc.Meta.SchemaVersion <= 0 {
		return nil, fmt.Errorf("%s: [meta].schema_version doit être > 0 (reçu %d)", path, doc.Meta.SchemaVersion)
	}
	seconds := make(map[string]int, len(doc.Seconds))
	for rawName, secs := range doc.Seconds {
		key := strings.TrimSpace(rawName)
		if key == "" {
			return nil, fmt.Errorf("%s: game_variant_name vide", path)
		}
		if secs <= 0 {
			return nil, fmt.Errorf("%s: variante %q : temps réglementaire doit être > 0 (reçu %d)", path, key, secs)
		}
		seconds[key] = secs
	}
	return &RegulationSet{
		titleSlug:     doc.Meta.TitleSlug,
		schemaVersion: doc.Meta.SchemaVersion,
		seconds:       seconds,
	}, nil
}
