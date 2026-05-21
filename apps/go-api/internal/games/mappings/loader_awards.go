package mappings

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// loader_awards.go — parser et validateur pour awards.toml (V2 §2).
//
// Source de vérité du mapping `personal_score_awards.award_name →
// ParticipationAxis` consommé par profile.computeRadarAxes pour produire un
// axe Objective fiable et un axe Impact enrichi.
//
// Pattern aligné sur loader_outcomes.go / loader_assets.go.

// awardsTOML est la projection brute du fichier awards.toml avant validation.
type awardsTOML struct {
	Meta   metaSection               `toml:"meta"`
	Awards map[string]awardEntryTOML `toml:"awards"`
}

type awardEntryTOML struct {
	Axes   []string `toml:"axes"`
	Weight float64  `toml:"weight"`
}

// AwardMapping est l'entrée résolue : 1 award → 1+ axes avec multiplicateur.
type AwardMapping struct {
	AwardName string
	Axes      []string
	Weight    float64
}

// AwardMappingSet contient la totalité des awards mappés pour un titre,
// indexés par award_name pour lookup O(1).
type AwardMappingSet struct {
	titleSlug     string
	schemaVersion int
	byName        map[string]AwardMapping
}

// NewAwardMappingSet construit un set à partir d'une map déjà validée.
// Exporté pour les tests qui veulent monter un set in-memory sans TOML.
func NewAwardMappingSet(titleSlug string, schemaVersion int, byName map[string]AwardMapping) *AwardMappingSet {
	return &AwardMappingSet{
		titleSlug:     titleSlug,
		schemaVersion: schemaVersion,
		byName:        byName,
	}
}

// TitleSlug retourne le slug du titre auquel ce set est rattaché.
func (s *AwardMappingSet) TitleSlug() string { return s.titleSlug }

// SchemaVersion retourne la version de schéma déclarée.
func (s *AwardMappingSet) SchemaVersion() int { return s.schemaVersion }

// Get retourne le mapping d'un award_name, ou empty + false si absent.
// Les awards absents = pas de contribution à aucun axe (V1 fallback).
func (s *AwardMappingSet) Get(awardName string) (AwardMapping, bool) {
	if s == nil {
		return AwardMapping{}, false
	}
	m, ok := s.byName[awardName]
	return m, ok
}

// All retourne tous les mappings (ordre non garanti). Utilisé par les tests.
func (s *AwardMappingSet) All() map[string]AwardMapping {
	if s == nil {
		return nil
	}
	out := make(map[string]AwardMapping, len(s.byName))
	for k, v := range s.byName {
		out[k] = v
	}
	return out
}

// axisObjective est l'axe de participation "objective" (porteur du drapeau, etc.).
// Aligné sur narrative.AxisObjective. Centralisé ici pour réduire les littéraux
// (référencé dans allowedAxes + message d'erreur de validation).
const axisObjective = "objective"

// allowedAxes liste les axes ParticipationAxis canoniques.
// Aligné sur narrative.ParticipationAxis. Toute autre valeur est rejetée.
var allowedAxes = map[string]struct{}{
	"combat":      {},
	"survival":    {},
	"support":     {},
	"score":       {},
	axisObjective: {},
	"impact":      {},
}

// LoadAwardsFromFile lit et valide un awards.toml à un chemin donné.
func LoadAwardsFromFile(path string) (*AwardMappingSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadAwardsFromBytes(path, raw)
}

// LoadAwardsFromBytes parse et valide un payload TOML déjà chargé en mémoire.
func LoadAwardsFromBytes(path string, raw []byte) (*AwardMappingSet, error) {
	var doc awardsTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var errs []error
	if doc.Meta.TitleSlug == "" {
		errs = append(errs, fmt.Errorf("[meta].title_slug manquant"))
	}
	if doc.Meta.SchemaVersion <= 0 {
		errs = append(errs, fmt.Errorf("[meta].schema_version doit être > 0 (reçu %d)", doc.Meta.SchemaVersion))
	}

	byName := make(map[string]AwardMapping, len(doc.Awards))
	for name, entry := range doc.Awards {
		entryErrs := validateAward(name, entry)
		if len(entryErrs) > 0 {
			for _, e := range entryErrs {
				errs = append(errs, fmt.Errorf("[awards.%s]: %w", name, e))
			}
			continue
		}
		// Copie défensive du slice axes.
		axes := make([]string, len(entry.Axes))
		copy(axes, entry.Axes)
		byName[name] = AwardMapping{
			AwardName: name,
			Axes:      axes,
			Weight:    entry.Weight,
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}
	return NewAwardMappingSet(doc.Meta.TitleSlug, doc.Meta.SchemaVersion, byName), nil
}

func validateAward(name string, e awardEntryTOML) []error {
	var errs []error
	if strings.TrimSpace(name) == "" {
		errs = append(errs, fmt.Errorf("award_name vide"))
	}
	if len(e.Axes) == 0 {
		errs = append(errs, fmt.Errorf("axes vide — un award doit cibler au moins 1 axe"))
	}
	for _, axis := range e.Axes {
		if _, ok := allowedAxes[axis]; !ok {
			errs = append(errs, fmt.Errorf("axe inconnu %q (admis: combat, survival, support, score, objective, impact)", axis))
		}
	}
	if e.Weight <= 0 {
		errs = append(errs, fmt.Errorf("weight doit être > 0 (reçu %.2f)", e.Weight))
	}
	return errs
}
