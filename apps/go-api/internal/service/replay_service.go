// Package service — replay_service.go : sert l'artefact de rejeu 2D pré-construit d'un
// match (data/cache/replays/{title}/{matchId}.json). Aucune logique de décodage ici —
// l'assemblage lourd est fait hors ligne par cmd/replay-build ; ce service lit l'artefact.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/port"
)

// replayService lit l'artefact de rejeu d'un titre donné via le PathResolver.
type replayService struct {
	titleSlug string
	repoRoot  string
	// maps nomme la carte d'un match (fond de carte). Nil = pas de fond servi, jamais
	// d'erreur : le rejeu reste lisible sur son sol structurel.
	maps port.ReplayMapNameRepo
}

// NewReplayService construit le service de rejeu pour un titre (résolu depuis le joueur).
//
// `maps` est la SEULE dépendance base du service — elle nomme la carte du match, ce que
// l'artefact ne sait pas faire. Nil est un cas servi : pas de fond de carte, le rejeu garde
// son sol structurel. Un paramètre plutôt qu'un `With*` : un service à deux formes de
// construction finit toujours par n'en avoir qu'une de testée.
func NewReplayService(titleSlug, repoRoot string, maps port.ReplayMapNameRepo) port.ReplayService {
	return &replayService{titleSlug: titleSlug, repoRoot: repoRoot, maps: maps}
}

// IsAvailable dit si l'artefact existe, par un os.Stat — JAMAIS par une lecture : la
// Match View interroge cette présence à chaque affichage de match, et charger 2 Mo de
// trajectoires pour répondre « oui » serait payer le rejeu sans le montrer.
//
// Tout échec vaut « pas de rejeu » (répertoire absent, droits, chemin non résolu) :
// l'absence de lien est la dégradation sûre, une erreur 500 sur la page match ne l'est pas.
func (s *replayService) IsAvailable(ctx context.Context, matchID string) bool {
	path := title.NewPathResolver(s.repoRoot).ReplayArtifactPath(s.titleSlug, matchID)
	info, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			// Un artefact présent mais illisible n'est pas un cas normal : il ne doit pas
			// se confondre en silence avec « ce match n'a pas de rejeu ».
			slog.WarnContext(ctx, "rejeu 2D : artefact non consultable",
				"err", err, "match_id", matchID, "titleSlug", s.titleSlug)
		}
		return false
	}
	return !info.IsDir()
}

// GetReplay lit et désérialise l'artefact du match. Retourne port.ErrReplayNotAvailable
// si aucun artefact n'existe (404 côté handler), une erreur enveloppée sinon.
//
// Le calque d'objectifs statiques (MapObjectives) est joint ICI, à la requête : il
// dépend de la carte et du mode, que l'artefact ne connaît pas (décodé des seuls chunks
// du film). Son absence n'est jamais une erreur — le rejeu se sert entier sans lui.
func (s *replayService) GetReplay(ctx context.Context, matchID string) (replay.ReplayDocument, error) {
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
	doc.MapObjectives = s.mapObjectivesForMatch(ctx, matchID)
	return doc, nil
}
