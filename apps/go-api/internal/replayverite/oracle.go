package replayverite

// oracle.go — LA LECTURE DU FICHIER D'ORACLE (`<short8>.oracle.json`).
//
// Le fichier est ecrit par `levelup replay-facts-export --oracle` (forme : `domain.MatchOracle`).
// Un seul decodeur, ici, pour ses deux lecteurs : l'outil `cmd/replay-verite` et le gate de corpus.

import (
	"encoding/json"
	"errors"
	"fmt"

	"levelup/go-api/internal/domain"
)

// ErrOracleVide : un oracle sans joueur ne juge rien — le lire comme « aucun ecart » serait faux.
var ErrOracleVide = errors.New("replayverite : oracle sans joueur")

// LireOracle desserialise un fichier d'oracle.
func LireOracle(blob []byte) (*domain.MatchOracle, error) {
	var o domain.MatchOracle
	if err := json.Unmarshal(blob, &o); err != nil {
		return nil, fmt.Errorf("replayverite : oracle illisible : %w", err)
	}
	if o.Empty() {
		return nil, ErrOracleVide
	}
	return &o, nil
}
