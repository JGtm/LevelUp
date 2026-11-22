package replaybuild

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"levelup/go-api/internal/replayartifact"
)

// ArtifactKickoff lit, dans l'artefact de rejeu rangé à path, le coup d'envoi que le document
// publie (`t0FilmMs`, en ms depuis le début canonique du match), TEL QUEL : aucun détecteur n'est
// rejoué. Seul ce champ est décodé.
//
// Rend (nil, false, nil) quand aucun artefact n'est rangé, (nil, true, nil) quand le document ne
// publie pas de coup d'envoi (schéma antérieur au champ, ou refus du décodeur), et une erreur
// quand l'artefact existe mais ne se lit pas.
func ArtifactKickoff(path string) (*int64, bool, error) {
	c, err := replayartifact.Read(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, true, fmt.Errorf("lecture artefact: %w", err)
	}
	var head struct {
		T0FilmMs *int64 `json:"t0FilmMs"`
	}
	if err := json.Unmarshal(c.JSON, &head); err != nil {
		return nil, true, fmt.Errorf("parse artefact: %w", err)
	}
	return head.T0FilmMs, true, nil
}
