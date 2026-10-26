package title

import "fmt"

// ModeSource dit où le registre d'un titre porte le MODE d'un match
// (`canonical.MatchSummary.PairMode`). Déclaré par `[title].mode_source` de
// config/titles/{slug}/title.toml, chargé dans TitleDescriptor.ModeSource ; la source
// de lecture des matchs l'applique UNE fois, les pages lisent PairMode sans repli.
type ModeSource string

const (
	// ModeSourcePair : le mode est la partie (pair_*) du registre ; une partie absente = mode inconnu.
	// Valeur par défaut (clé absente, valeur zéro).
	ModeSourcePair ModeSource = "pair"
	// ModeSourceGameVariant : le mode est la partie quand le registre en porte une, sinon le
	// type de jeu (game_variant_*).
	ModeSourceGameVariant ModeSource = "game_variant"
)

// ParseModeSource valide la valeur de `[title].mode_source` : "" (clé absente) = ModeSourcePair ;
// toute autre valeur que "pair" ou "game_variant" est une erreur (jamais un défaut silencieux).
func ParseModeSource(raw string) (ModeSource, error) {
	switch ModeSource(raw) {
	case "", ModeSourcePair:
		return ModeSourcePair, nil
	case ModeSourceGameVariant:
		return ModeSourceGameVariant, nil
	}
	return "", fmt.Errorf("[title].mode_source invalide: %q (attendu pair|game_variant)", raw)
}

// ModeSourceFor rend la source du mode du titre ; ModeSourcePair si le titre est inconnu.
func (r *Registry) ModeSourceFor(slug string) ModeSource {
	if d := r.Get(slug); d != nil && d.ModeSource != "" {
		return d.ModeSource
	}
	return ModeSourcePair
}
