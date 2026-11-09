package ops

import (
	_ "embed"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/pelletier/go-toml/v2"

	"levelup/go-api/internal/i18n"
)

// citationTextsTOML : noms et descriptions des citations, une table par langue et par entrée
// (citation_texts.toml). Embarqué dans le paquet : le seed tourne aussi sans arborescence de
// configuration (seed de démonstration synthétique).
//
//go:embed citation_texts.toml
var citationTextsTOML []byte

// citationLabels est le nom et la description d'une citation, une valeur par langue.
type citationLabels struct {
	Name        i18n.Text `toml:"name"`
	Description i18n.Text `toml:"description"`
}

type citationTextsDoc struct {
	Meta struct {
		TitleSlug     string `toml:"title_slug"`
		SchemaVersion int    `toml:"schema_version"`
	} `toml:"meta"`
	Citations map[string]citationLabels `toml:"citations"`
}

// parseCitationTexts lit et valide un citation_texts.toml : chaque entrée porte son nom ET sa
// description dans toutes les langues `required` de config/locales.toml, sans langue hors liste
// (i18n.Text.Validate). Toutes les entrées fautives sont signalées d'un coup.
func parseCitationTexts(raw []byte) (map[string]citationLabels, error) {
	var doc citationTextsDoc
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("citation_texts.toml : %w", err)
	}
	var errs []error
	if doc.Meta.SchemaVersion <= 0 {
		errs = append(errs, fmt.Errorf("[meta].schema_version doit être > 0 (reçu %d)", doc.Meta.SchemaVersion))
	}
	norms := make([]string, 0, len(doc.Citations))
	for norm := range doc.Citations {
		norms = append(norms, norm)
	}
	sort.Strings(norms)
	out := make(map[string]citationLabels, len(norms))
	for _, norm := range norms {
		e := doc.Citations[norm]
		e.Name, e.Description = e.Name.Trimmed(), e.Description.Trimmed()
		if err := e.Name.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[citations.%s] name : %w", norm, err))
		}
		if err := e.Description.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[citations.%s] description : %w", norm, err))
		}
		out[norm] = e
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return out, nil
}

// loadedCitationTexts charge et valide le fichier embarqué une seule fois.
var loadedCitationTexts = sync.OnceValues(func() (map[string]citationLabels, error) {
	return parseCitationTexts(citationTextsTOML)
})

// citationTextsFor rend le nom et la description de la citation `norm`, chacun dans toutes les
// langues ; une citation sans entrée est une erreur (le seed ne pose jamais un texte inventé).
func citationTextsFor(norm string) (names, descriptions i18n.Text, err error) {
	all, err := loadedCitationTexts()
	if err != nil {
		return nil, nil, err
	}
	e, ok := all[norm]
	if !ok {
		return nil, nil, fmt.Errorf("citation %q sans entrée dans citation_texts.toml", norm)
	}
	return e.Name.Clone(), e.Description.Clone(), nil
}
