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
	"strings"

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

// FiltrerTemoins restreint le manifeste aux temoins NOMMES, dans l'ordre ou l'operateur les a
// nommes. `noms` vide rend le manifeste tel quel.
//
// # POURQUOI CE FILTRE EXISTE (D2 (cloture M1), 2026-09-17)
//
// Quand l'export des faits tombe sur la base tenue en ecriture, un ou deux temoins sortent
// ABSENT sur un alea de quelques secondes. Le pilote de la cloture M1 les a rejoues en ecrivant
// un manifeste REDUIT a la main — un fichier qui ressemble au corpus sans en etre un, et dont
// personne ne sait plus, deux jours plus tard, qu'il n'etait qu'une reprise. `--temoins a,b`
// dit la meme chose sans fabriquer ce faux corpus : le manifeste versionne reste le seul.
//
// UN NOM INCONNU EST UNE ERREUR, jamais un silence : une faute de frappe rendrait sinon un
// manifeste vide (ou ampute) et un gate qui compare moins que ce qu'on croit — exactement le
// silence que `verifierCouverture` existe pour interdire.
func FiltrerTemoins(m Manifest, noms []string) (Manifest, error) {
	if len(noms) == 0 {
		return m, nil
	}
	parID := make(map[string]Temoin, len(m.Temoins))
	for _, t := range m.Temoins {
		parID[t.ID] = t
	}
	garde := make([]Temoin, 0, len(noms))
	var inconnus []string
	vus := map[string]bool{}
	for _, nom := range noms {
		t, ok := parID[nom]
		if !ok {
			inconnus = append(inconnus, nom)
			continue
		}
		if vus[nom] {
			continue
		}
		vus[nom] = true
		garde = append(garde, t)
	}
	if len(inconnus) > 0 {
		return Manifest{}, fmt.Errorf(
			"--temoins : %d nom(s) absent(s) du manifeste (%s) : %s — les ids connus sont : %s",
			len(inconnus), m.Meta.TitleSlug, strings.Join(inconnus, ", "), strings.Join(idsDuManifeste(m), ", "))
	}
	m.Temoins = garde
	return m, nil
}

// NomsDemandes decoupe la valeur de `--temoins` : ids separes par des virgules, espaces
// toleres, entrees vides ignorees.
func NomsDemandes(liste string) []string {
	var out []string
	for _, part := range strings.Split(liste, ",") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}
