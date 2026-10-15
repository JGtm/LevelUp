package service

// tactical_sequence_store.go — LE LECTEUR DES PISTES D'UN ARTEFACT DE REJEU, pour la sequence d'une
// mort (vue D). Il lit le fichier `{short8}.json` de l'artefact et n'en DECODE QUE ce dont la
// sequence a besoin (echelle de temps, origine du film, pistes : xuid, bot, points) ; le fond de
// carte, la structure, les objets et les autres calques sont parcourus sans etre materialises.
//
// MEME FICHIER, MEME RESOLVEUR DE CHEMIN que le service de rejeu (`PathResolver.ReplayArtifactPath`,
// racine d'execution). Il n'est monte que derriere la porte `film.replay_artifact` du titre et, au
// transport, derriere les gardes du rejeu (`RequireCapability(CapReplay)` + `LocalOnlyReplay`).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

type tacticalSequenceStore struct {
	runtimeRoot string
	titleSlug   string
}

// NewTacticalSequenceStore construit le lecteur de pistes d'un titre. `runtimeRoot` est la racine
// des artefacts d'execution (celle du service de rejeu).
func NewTacticalSequenceStore(runtimeRoot, titleSlug string) port.TacticalSequenceStore {
	return &tacticalSequenceStore{runtimeRoot: runtimeRoot, titleSlug: titleSlug}
}

// Pistes lit les pistes de l'artefact d'un match, ou (nil, nil) s'il n'y en a pas.
//
// LE FICHIER ENTIER EST LU ET DECODE A CHAQUE SEQUENCE, SANS CACHE : decision 9 du plan
// `.ai/V7.5/PLAN_TACTIQUE_COEQUIPIERS_2026-10-14.md` (une sequence est un geste de l'utilisateur, pas une
// page). Mesure du lot L2.9 : 123 a 139 ms sur le plus gros artefact du parc (13,4 Mo), sous le seuil de
// 500 ms ; un cache n'est justifie que si cette mesure change.
func (s *tacticalSequenceStore) Pistes(ctx context.Context, matchID string) (*domain.TacticalPistes, error) {
	path := title.NewPathResolver(s.runtimeRoot).ReplayArtifactPath(s.titleSlug, matchID)
	stop := timing.FromContext(ctx).Section("tactical_sequence_artefact")
	defer stop()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lecture de l'artefact de rejeu %s: %w", matchID, err)
	}
	var p domain.TacticalPistes
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, fmt.Errorf("deserialisation des pistes de l'artefact %s: %w", matchID, err)
	}
	return &p, nil
}
