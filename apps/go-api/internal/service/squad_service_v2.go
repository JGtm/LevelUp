// Package service — squad_service_v2.go : nouvelle version de la page Squad
// construite sur les fondations Phase 0 (PLAN_META_FOUNDATIONS_GO).
//
// Vit en parallèle de squad_service.go (legacy, mono-coéquipier) jusqu'à
// migration des consommateurs frontend (cf. PLAN_SQUAD_GO_PORTAGE).
//
// Phase 1 chunk S1 : ce fichier livre uniquement le squelette du service avec
// l'intersection des matchs de N coéquipiers (1..3) sur match_id. Les sections
// riches (KPI, score d'équipe, charts synergies, impact 8 rôles, radar...)
// seront greffées par les chunks S2-S11 sans toucher cette base.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"golang.org/x/sync/errgroup"

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// SquadV2Loader est l'interface minimale consommée par SquadServiceV2 pour
// charger les matchs d'un joueur. Permet d'injecter un mock en test sans
// dépendre de PlayerDB / pool concret.
//
// L'implémentation production résout (slug, gamertag) -> PlayerDB via
// pool.GetOrOpen + duckdb.NewPlayerMatchesRepo (adapter à fournir au handler
// dans un chunk ultérieur).
type SquadV2Loader interface {
	LoadFor(
		ctx context.Context,
		slug string,
		gamertag string,
		filters port.PlayerMatchFilters,
	) ([]canonical.PlayerMatchRow, error)
}

// SquadServiceV2 orchestre la page Squad V2.
type SquadServiceV2 struct {
	loader SquadV2Loader
}

// NewSquadServiceV2 construit le service avec un loader injecté.
func NewSquadServiceV2(loader SquadV2Loader) *SquadServiceV2 {
	return &SquadServiceV2{loader: loader}
}

// MaxTeammates est la borne haute du nombre de coéquipiers acceptés (cohérent
// avec la version Python : sélection 1..3).
const MaxTeammates = 3

// GetSquadPage charge les matchs du joueur principal + chacun des coéquipiers
// (parallèle), calcule l'intersection sur match_id, et retourne le DTO V2.
//
// Capability gating : si un joueur retourne games.ErrCapabilityNotSupported,
// il est exclu de l'intersection (le DTO ne le mentionne pas dans Players)
// et un CapabilityGap est ajouté à Capabilities. Si le joueur principal lui-même
// a la capability absente, la page est vide (SharedMatches=nil) avec un gap
// "fatal".
//
// Erreurs autres que ErrCapabilityNotSupported propagées comme une erreur
// 500 par le handler.
func (s *SquadServiceV2) GetSquadPage(
	ctx context.Context,
	slug string,
	mainGT string,
	teammateGTs []string,
	period temporal.Period,
) (*domain.SquadPageV2Response, error) {
	if mainGT == "" {
		return nil, errors.New("SquadServiceV2.GetSquadPage: mainGT requis")
	}
	if len(teammateGTs) > MaxTeammates {
		return nil, fmt.Errorf("SquadServiceV2.GetSquadPage: max %d coéquipiers, %d fournis",
			MaxTeammates, len(teammateGTs))
	}

	filters := port.PlayerMatchFilters{}
	if period != "" {
		filters.Period = &period
	}
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("SquadServiceV2.GetSquadPage: filters: %w", err)
	}

	perPlayer, capGaps, err := s.loadAllPlayers(ctx, slug, mainGT, teammateGTs, filters)
	if err != nil {
		return nil, err
	}

	resp := &domain.SquadPageV2Response{
		MainPlayer:   mainGT,
		Teammates:    teammateGTs,
		Period:       string(period),
		Capabilities: capGaps,
	}

	if _, hasMain := perPlayer[mainGT]; !hasMain {
		// Joueur principal indisponible : page vide mais capability gap signalé.
		slog.WarnContext(ctx, "squad: capability absente pour le joueur principal",
			"player", mainGT, "title_slug", slug)
		return resp, nil
	}

	resp.SharedMatches = intersectByMatchID(perPlayer)
	resp.SharedMatchesCount = len(resp.SharedMatches)
	return resp, nil
}

// loadAllPlayers charge les matchs du joueur principal + des coéquipiers en
// parallèle. Capability absente -> exclu de perPlayer + ajouté à capGaps.
func (s *SquadServiceV2) loadAllPlayers(
	ctx context.Context,
	slug, mainGT string,
	teammateGTs []string,
	filters port.PlayerMatchFilters,
) (map[string][]canonical.PlayerMatchRow, []canonical.CapabilityGap, error) {
	allGTs := append([]string{mainGT}, teammateGTs...)

	g, gctx := errgroup.WithContext(ctx)
	var mu sync.Mutex
	perPlayer := make(map[string][]canonical.PlayerMatchRow, len(allGTs))
	var capGaps []canonical.CapabilityGap

	for _, gt := range allGTs {
		gt := gt
		g.Go(func() error {
			rows, err := s.loader.LoadFor(gctx, slug, gt, filters)
			if err != nil {
				if errors.Is(err, games.ErrCapabilityNotSupported) {
					mu.Lock()
					capGaps = append(capGaps, canonical.CapabilityGap{
						CapabilityKey: string(games.CapMatchHistory),
						ReasonCode:    "match_history_unsupported",
						Severity:      "warning",
						Message:       fmt.Sprintf("match.history non supporté pour %s", gt),
					})
					mu.Unlock()
					return nil
				}
				return fmt.Errorf("LoadFor(%s): %w", gt, err)
			}
			mu.Lock()
			perPlayer[gt] = rows
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return perPlayer, capGaps, nil
}

// intersectByMatchID retourne les matchs présents chez TOUS les joueurs de
// perPlayer. Trié par StartedAt DESC (match le plus récent en premier).
//
// Si perPlayer est vide ou contient des slices vides, retourne nil.
func intersectByMatchID(perPlayer map[string][]canonical.PlayerMatchRow) []domain.SquadSharedMatch {
	if len(perPlayer) == 0 {
		return nil
	}

	indexed := make(map[string]map[string]canonical.PlayerMatchRow, len(perPlayer))
	for gt, rows := range perPlayer {
		idx := make(map[string]canonical.PlayerMatchRow, len(rows))
		for _, r := range rows {
			idx[r.Summary.MatchID] = r
		}
		indexed[gt] = idx
	}

	// Trouver l'intersection : un match_id présent chez tous.
	sharedIDs := matchIDsPresentInAll(indexed)
	if len(sharedIDs) == 0 {
		return nil
	}

	out := make([]domain.SquadSharedMatch, 0, len(sharedIDs))
	for _, id := range sharedIDs {
		sm := buildSharedMatch(id, indexed)
		out = append(out, sm)
	}

	// Tri par StartedAt DESC, fallback alphabétique sur MatchID si égalité.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].StartedAt.After(out[j].StartedAt)
		}
		return out[i].MatchID < out[j].MatchID
	})
	return out
}

// matchIDsPresentInAll retourne la liste des match_id présents dans toutes les
// maps (intersection ensembliste sur les clés).
func matchIDsPresentInAll(indexed map[string]map[string]canonical.PlayerMatchRow) []string {
	if len(indexed) == 0 {
		return nil
	}
	// Choisir la map la plus petite comme base pour minimiser le travail.
	var smallestGT string
	smallestSize := -1
	for gt, m := range indexed {
		if smallestSize == -1 || len(m) < smallestSize {
			smallestSize = len(m)
			smallestGT = gt
		}
	}

	var out []string
	for id := range indexed[smallestGT] {
		present := true
		for gt, m := range indexed {
			if gt == smallestGT {
				continue
			}
			if _, ok := m[id]; !ok {
				present = false
				break
			}
		}
		if present {
			out = append(out, id)
		}
	}
	return out
}

// buildSharedMatch hydrate un SquadSharedMatch depuis les rows de chaque joueur.
// Les champs niveau-match (Map, Mode, Outcome, StartedAt) sont pris du joueur
// principal sortable (premier dans l'ordre alphabétique des gamertags pour
// reproductibilité).
func buildSharedMatch(matchID string, indexed map[string]map[string]canonical.PlayerMatchRow) domain.SquadSharedMatch {
	sm := domain.SquadSharedMatch{
		MatchID: matchID,
		Players: make(map[string]canonical.PlayerMatchRow, len(indexed)),
	}
	gts := make([]string, 0, len(indexed))
	for gt := range indexed {
		gts = append(gts, gt)
	}
	sort.Strings(gts)
	for _, gt := range gts {
		row := indexed[gt][matchID]
		sm.Players[gt] = row
		if sm.StartedAt.IsZero() {
			sm.StartedAt = row.Summary.StartedAtUTC
			sm.Map = row.Summary.Map
			sm.Mode = row.Summary.GameVariant
			sm.Playlist = row.Summary.Playlist
			sm.Outcome = row.Summary.Outcome
		}
	}
	return sm
}
