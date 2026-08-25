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
	"strings"

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

// AvailableSet liste les matchs du titre qui ont un artefact, par UN SEUL listing du
// dossier d'artefacts — jamais un os.Stat par match. C'est la forme qu'interrogent les
// TABLEAUX de matchs (Explorer, escouade) : ils affichent des centaines de lignes, et un
// appel disque par ligne coûterait la page entière pour une icône.
//
// Le dossier absent est nominal (titre sans rejeu construit) : ensemble vide, pas
// d'erreur. Une lecture qui échoue pour une autre raison est journalisée ET remontée —
// l'appelant dégrade sur l'ensemble vide (aucune icône), jamais sur un 500.
func (s *replayService) AvailableSet(ctx context.Context) (port.ReplayAvailability, error) {
	dir := title.NewPathResolver(s.repoRoot).ReplayArtifactsDir(s.titleSlug)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return port.ReplayAvailability{}, nil
	}
	if err != nil {
		slog.ErrorContext(ctx, "rejeu 2D : dossier d'artefacts illisible",
			"err", err, "dir", dir, "titleSlug", s.titleSlug)
		return port.ReplayAvailability{}, fmt.Errorf("listing artefacts rejeu %s: %w", s.titleSlug, err)
	}
	set := make(port.ReplayAvailability, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, replayArtifactExt) {
			continue // on ne compte QUE des artefacts {short8}.json
		}
		set[strings.TrimSuffix(name, replayArtifactExt)] = struct{}{}
	}
	return set, nil
}

// replayArtifactExt est l'extension des artefacts de rejeu ({short8}.json) — la même
// convention que PathResolver.ReplayArtifactPath et que la purge récurrente.
const replayArtifactExt = ".json"

// GetReplay lit et désérialise l'artefact du match. Retourne port.ErrReplayNotAvailable
// si aucun artefact n'existe (404 côté handler), une erreur enveloppée sinon.
//
// TROIS RÉSOLUTIONS SE POSENT ICI, à la requête, et pour la même raison : elles viennent
// d'un catalogue du TITRE, que l'artefact (décodé des seuls chunks du film) ne connaît pas.
//   - le calque d'objectifs statiques (MapObjectives), qui dépend de la carte et du mode ;
//   - les EMPLACEMENTS DE SOCLE de la carte (MapWeaponPads), croisés avec les socles du
//     match : le fichier de carte les pose, seul le film dit lesquels sont allumés (cf.
//     replay_map_weapon_pads.go) ;
//   - ce que le TITRE sait de chaque arme (WeaponLabel.Key et .Tint) : la clé qui ouvre la
//     banque de sons du client, et la nature de la décharge qui teinte son éclair de
//     bouche (cf. replay_weapon_labels.go).
//
// L'absence de l'une ou de l'autre n'est jamais une erreur — le rejeu se sert entier sans.
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
	// LES CLÉS DE CARTE, UNE SEULE FOIS : deux calques statiques les partagent, et la base
	// ne doit pas être interrogée deux fois pour la même réponse.
	keys := s.matchMapKeys(ctx, matchID)
	doc.MapObjectives = s.mapObjectivesForKeys(ctx, matchID, keys)
	doc.MapWeaponPads = s.mapWeaponPadsForKeys(ctx, matchID, keys, doc.WeaponPads)
	// La CLÉ CANONIQUE et la TEINTE de chaque arme, même règle et même raison : elles se
	// résolvent d'un catalogue du titre, donc ici et pas dans l'artefact (cf.
	// replay_weapon_labels.go).
	s.resolveWeaponLabels(ctx, &doc)
	return doc, nil
}
