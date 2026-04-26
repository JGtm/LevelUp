package mappings

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// outcomesTOML est la projection brute du fichier outcomes.toml avant validation.
type outcomesTOML struct {
	Meta     metaSection                  `toml:"meta"`
	Outcomes map[string]outcomeEntryTOML  `toml:"outcomes"`
}

type outcomeEntryTOML struct {
	Labels     map[string]string `toml:"labels"`
	ColorToken string            `toml:"color_token"`
}

// allowedOutcomeKeys est la liste exhaustive des outcomes canoniques admis.
// Aligné sur canonical/enums.go (Outcome enum). Toute autre clé est rejetée.
var allowedOutcomeKeys = map[string]struct{}{
	"win":  {},
	"loss": {},
	"tie":  {},
	"dnf":  {},
}

// LoadOutcomesFromFile lit et valide un outcomes.toml à un chemin donné.
func LoadOutcomesFromFile(path string) (*OutcomeMappingSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return LoadOutcomesFromBytes(path, raw)
}

// LoadOutcomesFromBytes parse et valide un payload TOML déjà chargé en mémoire.
func LoadOutcomesFromBytes(path string, raw []byte) (*OutcomeMappingSet, error) {
	var doc outcomesTOML
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

	byKey := make(map[string]OutcomeMapping, len(doc.Outcomes))
	for key, entry := range doc.Outcomes {
		if _, ok := allowedOutcomeKeys[key]; !ok {
			errs = append(errs, fmt.Errorf("[outcomes.%s] clé inconnue (admises : win, loss, tie, dnf)", key))
			continue
		}
		entryErrs := validateOutcome(key, entry)
		if len(entryErrs) > 0 {
			for _, e := range entryErrs {
				errs = append(errs, fmt.Errorf("[outcomes.%s]: %w", key, e))
			}
			continue
		}
		byKey[key] = OutcomeMapping{
			Key:        key,
			Labels:     copyStringMap(entry.Labels),
			ColorToken: entry.ColorToken,
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("validation %s: %w", path, errors.Join(errs...))
	}

	return NewOutcomeMappingSet(doc.Meta.TitleSlug, doc.Meta.SchemaVersion, byKey), nil
}

func validateOutcome(key string, e outcomeEntryTOML) []error {
	var errs []error
	if _, ok := e.Labels[LocaleEN]; !ok || strings.TrimSpace(e.Labels[LocaleEN]) == "" {
		errs = append(errs, fmt.Errorf("label EN manquant"))
	}
	if _, ok := e.Labels[LocaleFR]; !ok || strings.TrimSpace(e.Labels[LocaleFR]) == "" {
		errs = append(errs, fmt.Errorf("label FR manquant"))
	}
	if strings.TrimSpace(e.ColorToken) == "" {
		errs = append(errs, fmt.Errorf("color_token manquant"))
	}
	_ = key
	return errs
}
