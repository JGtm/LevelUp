package classification

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// rankedHoppersTOML — projection brute de config/titles/<slug>/catalog/ranked_hoppers.toml.
type rankedHoppersTOML struct {
	SchemaVersion   int      `toml:"schema_version"`
	RankedHopperIDs []string `toml:"ranked_hopper_ids"`
	PvEHopperIDs    []string `toml:"pve_hopper_ids"`
}

// rankedHoppersSchemaVersion est la seule version de schéma supportée.
const rankedHoppersSchemaVersion = 1

// LoadSetClassifier charge un SetClassifier depuis un TOML d'ids autoritatifs.
//
// Fichier ABSENT → classifier VIDE (verdicts nil), PAS une erreur : la config est
// OPTIONNELLE tant que la liste autoritative n'est pas publiée (un titre sans ce
// fichier ne classe simplement pas → dégradation conservatrice). Erreur de
// lecture (≠ absent), de parse, ou schema_version non supportée → erreur (config
// malformée = bug à corriger, pas à avaler silencieusement).
func LoadSetClassifier(path string) (*SetClassifier, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewSetClassifier(nil, nil), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc rankedHoppersTOML
	if err := toml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if doc.SchemaVersion != rankedHoppersSchemaVersion {
		return nil, fmt.Errorf("ranked_hoppers %s: schema_version=%d non supportée (attendu %d)",
			path, doc.SchemaVersion, rankedHoppersSchemaVersion)
	}
	return NewSetClassifier(doc.RankedHopperIDs, doc.PvEHopperIDs), nil
}
