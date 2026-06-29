// Package ops — seed_demo_manifest.go : manifeste démo FIGÉ.
//
// Problème résolu : `seed-demo` reconstruisait le corpus depuis les DERNIERS matchs
// live à chaque regen → la sélection curée (matchs avec médias associés, panel large
// solo + escouade, sessions multi-matchs) dérivait, médias désassociés, sessions non
// représentatives. Le manifeste gèle la sélection : quand il est présent, `seed-demo`
// lit ses 4 sous-listes de match_ids au lieu des requêtes dynamiques « N plus récents ».
//
// Pourquoi geler le CORPUS suffit : le roster anonymisé (buildDemoRoster classe les
// coéquipiers sur le corpus escouade, ordre stable `n DESC, xuid`) et l'association des
// médias (ancrage sur un match du corpus) sont des FONCTIONS du corpus + des données
// source IMMUABLES (un match ne change pas). Corpus figé ⇒ roster + médias déterministes,
// sans avoir à les lister explicitement. Dériver le roster du corpus garantit en outre
// que TOUT xuid réel du corpus est anonymisé (aucune fuite).
package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const demoManifestVersion = "1"

// DemoManifest gèle la sélection de matchs d'une démo pour un couple (gamertag, titre).
// Committé sous config/demo/<gamertag>/<slug>.json.
type DemoManifest struct {
	Version        string             `json:"version"`
	TitleSlug      string             `json:"title_slug"`
	SourceGamertag string             `json:"source_gamertag,omitempty"`
	SourceXUID     string             `json:"source_xuid,omitempty"`
	GeneratedAt    string             `json:"generated_at,omitempty"`
	GeneratedBy    string             `json:"generated_by,omitempty"`
	Notes          string             `json:"notes,omitempty"`
	Corpus         DemoManifestCorpus `json:"corpus"`
}

// DemoManifestCorpus garde les 4 sous-listes SÉPARÉES (et non un seul tableau fusionné) :
//   - préserve l'ordre/dedup de unionMatchIDs (solo→squad→ranked→media) ;
//   - `squad_match_ids` sert SEUL à classer les coéquipiers principaux (buildDemoRoster),
//     distinct du corpus complet.
type DemoManifestCorpus struct {
	SoloMatchIDs   []string `json:"solo_match_ids"`
	SquadMatchIDs  []string `json:"squad_match_ids"`
	RankedMatchIDs []string `json:"ranked_match_ids"`
	MediaMatchIDs  []string `json:"media_match_ids"`
}

// CorpusMatchIDs rejoue unionMatchIDs sur les 4 sous-listes (même ordre/dedup que la
// sélection dynamique).
func (m *DemoManifest) CorpusMatchIDs() []string {
	return unionMatchIDs(m.Corpus.SoloMatchIDs, m.Corpus.SquadMatchIDs, m.Corpus.RankedMatchIDs, m.Corpus.MediaMatchIDs)
}

// LoadDemoManifest lit le manifeste à path. found=false (err nil) si le fichier est
// ABSENT → le caller retombe sur la sélection dynamique. Un fichier PRÉSENT mais
// invalide est une erreur DURE : un manifeste committé corrompu doit casser le seed,
// pas revenir silencieusement au dynamique (sinon la démo « dégèle » sans qu'on le voie).
func LoadDemoManifest(path string) (*DemoManifest, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("lecture manifeste démo %s: %w", path, err)
	}
	var m DemoManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false, fmt.Errorf("parse manifeste démo %s: %w", path, err)
	}
	if err := validateDemoManifest(&m); err != nil {
		return nil, false, fmt.Errorf("manifeste démo %s invalide: %w", path, err)
	}
	return &m, true, nil
}

// validateDemoManifest vérifie la version supportée et un corpus non vide.
func validateDemoManifest(m *DemoManifest) error {
	if m.Version == "" {
		return fmt.Errorf("champ version manquant")
	}
	if m.Version != demoManifestVersion {
		return fmt.Errorf("version %q non supportée (attendu %q)", m.Version, demoManifestVersion)
	}
	if len(m.CorpusMatchIDs()) == 0 {
		return fmt.Errorf("corpus vide (aucun match_id)")
	}
	return nil
}

// writeDemoManifest sérialise m vers path (création des dossiers parents).
func writeDemoManifest(path string, m *DemoManifest) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir manifeste démo: %w", err)
	}
	return writeJSONFile(path, m)
}
