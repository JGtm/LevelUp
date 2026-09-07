package main

// manifest.go — LE MANIFESTE VERSIONNE `config/replay_corpus.toml`.
//
// Un temoin par famille de mode (ports de drapeau/crane/bombe, vies de vehicule...), choisi
// dans le parc local, JAMAIS un echantillon aleatoire — cf. l'en-tete du fichier TOML pour la
// raison de chaque entree. Ce fichier ne fait QUE le lire et le valider ; la selection elle-
// meme est une decision produit figee dans la donnee, pas dans le code.

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Manifest est la forme de `config/replay_corpus.toml`.
type Manifest struct {
	Meta struct {
		TitleSlug     string `toml:"title_slug"`
		SchemaVersion int    `toml:"schema_version"`
	} `toml:"meta"`
	Temoins []Temoin `toml:"temoin"`
}

// Temoin est UNE entree du corpus : un match choisi pour la famille de mode qu'il porte.
type Temoin struct {
	ID      string `toml:"id"`
	Famille string `toml:"famille"`
	Mode    string `toml:"mode"`
	Carte   string `toml:"carte"`
	Raison  string `toml:"raison"`
}

// LoadManifest lit et valide le manifeste. Un manifeste sans `title_slug` ou sans temoin est
// une erreur de configuration — le gate n'a alors rien a comparer et ne doit pas le taire.
func LoadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path) //nolint:gosec // chemin fourni par l'operateur du CLI
	if err != nil {
		return Manifest{}, fmt.Errorf("manifeste illisible (%s) : %w", path, err)
	}
	var m Manifest
	if err := toml.Unmarshal(raw, &m); err != nil {
		return Manifest{}, fmt.Errorf("manifeste invalide (%s) : %w", path, err)
	}
	if m.Meta.TitleSlug == "" {
		return Manifest{}, fmt.Errorf("manifeste %s : [meta].title_slug absent", path)
	}
	if len(m.Temoins) == 0 {
		return Manifest{}, fmt.Errorf("manifeste %s : aucun [[temoin]]", path)
	}
	for i, t := range m.Temoins {
		if t.ID == "" {
			return Manifest{}, fmt.Errorf("manifeste %s : temoin #%d sans id", path, i)
		}
		if t.Famille == "" {
			return Manifest{}, fmt.Errorf("manifeste %s : temoin %s sans famille", path, t.ID)
		}
	}
	return m, nil
}
