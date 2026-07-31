// Package service — replay_service.go : sert l'artefact de rejeu 2D pré-construit d'un
// match (data/cache/replays/{title}/{matchId}.json). Aucune logique de décodage ici —
// l'assemblage lourd est fait hors ligne par cmd/replay-build ; ce service lit l'artefact.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// replayService lit l'artefact de rejeu d'un titre donné via le PathResolver.
type replayService struct {
	titleSlug string
	repoRoot  string
}

// NewReplayService construit le service de rejeu pour un titre (résolu depuis le joueur).
func NewReplayService(titleSlug, repoRoot string) port.ReplayService {
	return &replayService{titleSlug: titleSlug, repoRoot: repoRoot}
}

// GetReplay lit et désérialise l'artefact du match. Retourne port.ErrReplayNotAvailable
// si aucun artefact n'existe (404 côté handler), une erreur enveloppée sinon.
func (s *replayService) GetReplay(_ context.Context, matchID string) (replay.ReplayDocument, error) {
	path := title.NewPathResolver(s.repoRoot).ReplayArtifactPath(s.titleSlug, matchID)
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return replay.ReplayDocument{}, port.ErrReplayNotAvailable
	}
	if err != nil {
		return replay.ReplayDocument{}, fmt.Errorf("lecture artefact rejeu %s: %w", matchID, err)
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return replay.ReplayDocument{}, fmt.Errorf("désérialisation artefact rejeu %s: %w", matchID, err)
	}
	return doc, nil
}
