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

	"levelup/go-api/internal/analysis/temporal"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// SquadServiceV2 orchestre la page Squad V2. Son loader (interface SquadV2Loader)
// vit désormais dans le package feuille squadagg (K3b, ré-exporté ici en alias).
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
	experienceTypes []string,
	playlists []string,
	maps []string,
	modes []string,
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

	// Appliquer le filtre cascade (experience_types, playlists, maps, modes) sur les rows de
	// chaque joueur avant l'intersection : seuls les matchs satisfaisant tous les critères
	// sont conservés.
	if len(experienceTypes) > 0 || len(playlists) > 0 || len(maps) > 0 || len(modes) > 0 {
		for gt, rows := range perPlayer {
			perPlayer[gt] = filterRowsByCascade(rows, experienceTypes, playlists, maps, modes)
		}
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

	// gtToXUID est utilise par buildSquadHeader (KPIsByXUID drill-down) ET
	// par les charts/tables ci-dessous (squadXUIDs). Calcul fait une seule fois.
	squadOrder := buildSquadOrder(mainGT, teammateGTs)
	squadXUIDs := extractSquadXUIDs(squadOrder, perPlayer)

	resp.Header = buildSquadHeader(ctx, mainGT, squadXUIDs, resp.SharedMatches)

	// Si pas de matchs partages, retourner sans charger les sections lourdes.
	if len(resp.SharedMatches) == 0 {
		return resp, nil
	}

	// Composition des charts + tableaux. Charge les sources externes
	// (events, weapons, medals) en parallele puis appelle les 16 builders.
	rowsBySharedPlayer := projectSharedRows(resp.SharedMatches)

	squadHistorical := s.loadSquadHistorical(ctx, slug, mainGT, squadXUIDs)
	events, eventsCapGap := s.loadSharedEvents(ctx, slug, resp.SharedMatches, squadXUIDs)
	weapons, weaponsCapGap := s.loadWeapons(ctx, slug, resp.SharedMatches, squadXUIDs)
	medals, medalsCapGap := s.loadMedals(ctx, slug, resp.SharedMatches, squadXUIDs)

	resp.Charts = buildSquadCharts(buildSquadChartsInput{
		mainGT:          mainGT,
		squadOrder:      squadOrder,
		squadXUIDs:      squadXUIDs,
		rowsByPlayer:    rowsBySharedPlayer,
		squadHistorical: squadHistorical,
		events:          events,
		sharedMatches:   resp.SharedMatches,
		provideSpree:    games.ProvidesMaxKillingSpree(slug),
	})
	resp.Tables = buildSquadTables(buildSquadTablesInput{
		sharedMatches: resp.SharedMatches,
		rowsByPlayer:  rowsBySharedPlayer,
		squadOrder:    squadOrder,
		squadXUIDs:    squadXUIDs,
		weapons:       weapons,
		medals:        medals,
	})

	for _, gap := range []*canonical.CapabilityGap{eventsCapGap, weaponsCapGap, medalsCapGap} {
		if gap != nil {
			resp.Capabilities = append(resp.Capabilities, *gap)
		}
	}
	return resp, nil
}

// loadSquadHistorical charge les stats par carte du main AVEC l'escouade
// strict (winrate + perf moyenne sur tous les matchs où le squad complet
// est present). Aucun filtre temporel. Sert aux charts BulletWinrate /
// PerfVsHistorical (S3). Capability absente / squad vide -> nil.
func (s *SquadServiceV2) loadSquadHistorical(
	ctx context.Context,
	slug, mainGT string,
	squadXUIDs map[string]string,
) map[string]domain.MapSquadStats {
	if len(squadXUIDs) == 0 {
		return nil
	}
	xuids := make([]string, 0, len(squadXUIDs))
	for gt, x := range squadXUIDs {
		if gt == mainGT || x == "" {
			continue
		}
		xuids = append(xuids, x)
	}
	if len(xuids) == 0 {
		return nil
	}
	stats, err := s.loader.LoadMapStatsForSquad(ctx, slug, mainGT, xuids)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil
		}
		slog.WarnContext(ctx, "squad: loadSquadHistorical echec",
			"err", err, "player", mainGT, "squad_size", len(xuids))
		return nil
	}
	return stats
}

// sharedMatchTimelines indexe une MatchTimeline par match_id depuis les
// PlayerMatchRow des matchs partagés (durée identique pour tous les joueurs
// d'un match — on prend la première row disponible).
func sharedMatchTimelines(shared []domain.SquadSharedMatch) map[string]domain.MatchTimeline {
	rows := make([]canonical.PlayerMatchRow, 0, len(shared))
	for _, m := range shared {
		for _, pmr := range m.Players {
			rows = append(rows, pmr)
			break
		}
	}
	return timeline.BuildTimelinesFromPlayerMatches(rows)
}

// loadSharedEvents charge les events filmes des matchs partages (squad XUIDs).
// Capability absente -> retourne nil + CapabilityGap pour signaler S5/S6 omis.
func (s *SquadServiceV2) loadSharedEvents(
	ctx context.Context,
	slug string,
	shared []domain.SquadSharedMatch,
	squadXUIDs map[string]string,
) ([]canonical.HighlightEvent, *canonical.CapabilityGap) {
	if len(shared) == 0 || len(squadXUIDs) == 0 {
		return nil, nil
	}
	matchIDs := matchIDsOf(shared)
	xuids := xuidsOf(squadXUIDs)
	// Pour valider les filtres : MatchIDs requis (pas besoin de PlayerXUID
	// dans ce cas, le repo filtrera client-side via squadXUIDs).
	filters := port.HighlightEventFilters{MatchIDs: matchIDs}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "squad: HighlightEventFilters invalides", "err", err)
		return nil, nil
	}
	events, err := s.loader.LoadHighlightEvents(ctx, slug, filters)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, &canonical.CapabilityGap{
				CapabilityKey: "match.detail.events",
				ReasonCode:    "events_unsupported",
				Severity:      "info",
				Message:       "Events filmes non disponibles : Impact + Cadence + Intensite omis.",
			}
		}
		slog.ErrorContext(ctx, "squad: LoadHighlightEvents echec", "err", err)
		return nil, nil
	}
	// Correction chronologie T0 (Phase 1 : T0=0, identite). Point unique amont :
	// les builders cadence / intensity / impact en aval restent agnostiques.
	events = timeline.CorrectEvents(events, sharedMatchTimelines(shared))
	// Filtrer client-side aux squad xuids (repo retourne tous events des matchs).
	xuidSet := make(map[string]bool, len(xuids))
	for _, x := range xuids {
		xuidSet[x] = true
	}
	filtered := events[:0]
	for _, ev := range events {
		if isEventInSquad(ev, xuidSet) {
			filtered = append(filtered, ev)
		}
	}
	return filtered, nil
}

func isEventInSquad(ev canonical.HighlightEvent, xuidSet map[string]bool) bool {
	if ev.KillerXUID != nil && xuidSet[*ev.KillerXUID] {
		return true
	}
	if ev.VictimXUID != nil && xuidSet[*ev.VictimXUID] {
		return true
	}
	if ev.PlayerXUID != nil && xuidSet[*ev.PlayerXUID] {
		return true
	}
	return false
}

// loadWeapons charge les kills aggregees par arme pour les matchs partages.
func (s *SquadServiceV2) loadWeapons(
	ctx context.Context,
	slug string,
	shared []domain.SquadSharedMatch,
	squadXUIDs map[string]string,
) ([]port.WeaponKillRow, *canonical.CapabilityGap) {
	if len(shared) == 0 || len(squadXUIDs) == 0 {
		return nil, nil
	}
	filters := port.WeaponKillFilters{
		MatchIDs:            matchIDsOf(shared),
		XUIDs:               xuidsOf(squadXUIDs),
		IncludeGrenadeMelee: true,
	}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "squad: WeaponKillFilters invalides", "err", err)
		return nil, nil
	}
	rows, err := s.loader.LoadWeaponKills(ctx, slug, filters)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, &canonical.CapabilityGap{
				CapabilityKey: "match.detail.weapon_kills",
				ReasonCode:    "weapon_kills_unsupported",
				Severity:      "info",
				Message:       "Kills par arme non disponibles : tableau armes omis.",
			}
		}
		slog.ErrorContext(ctx, "squad: LoadWeaponKills echec", "err", err)
		return nil, nil
	}
	return rows, nil
}

// loadMedals charge les medailles par (xuid, match) pour les matchs partages.
func (s *SquadServiceV2) loadMedals(
	ctx context.Context,
	slug string,
	shared []domain.SquadSharedMatch,
	squadXUIDs map[string]string,
) ([]port.MedalRow, *canonical.CapabilityGap) {
	if len(shared) == 0 || len(squadXUIDs) == 0 {
		return nil, nil
	}
	filters := port.MedalsByXUIDFilters{
		MatchIDs: matchIDsOf(shared),
		XUIDs:    xuidsOf(squadXUIDs),
	}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "squad: MedalsByXUIDFilters invalides", "err", err)
		return nil, nil
	}
	rows, err := s.loader.LoadMedals(ctx, slug, filters)
	if err != nil {
		if errors.Is(err, games.ErrCapabilityNotSupported) {
			return nil, &canonical.CapabilityGap{
				CapabilityKey: "match.detail.medals",
				ReasonCode:    "medals_unsupported",
				Severity:      "info",
				Message:       "Medailles non disponibles : galerie medailles omise.",
			}
		}
		slog.ErrorContext(ctx, "squad: LoadMedals echec", "err", err)
		return nil, nil
	}
	return rows, nil
}

func matchIDsOf(shared []domain.SquadSharedMatch) []string {
	out := make([]string, 0, len(shared))
	for _, sm := range shared {
		out = append(out, sm.MatchID)
	}
	return out
}

func xuidsOf(squadXUIDs map[string]string) []string {
	out := make([]string, 0, len(squadXUIDs))
	for _, x := range squadXUIDs {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

// buildSquadHeader construit le SquadHeader (KPIs personnels + score equipe +
// cartes joueurs + KPIs per-xuid pour drill-down SessionBriefing) depuis les
// rows par joueur et l'intersection des matchs partages.
//
// SoloKPIs : agreges depuis les rows du joueur principal sur les matchs
// PARTAGES (intersection escouade). Le briefing en haut de page Escouade
// reflete les stats du joueur sur les matchs joues avec l'escouade definie,
// pas sur tout son historique. Si aucun match partage -> SoloKPIs nil.
// AllTimeKPIs nil pour S2 (a remplir dans un chunk dedie quand on cablera la
// tendance ▲▼).
//
// PlayerCards : 1 carte par joueur sur les matchs PARTAGES (intersection),
// pas sur l'historique solo. C'est aligne avec Python qui calcule le score
// d'equipe sur les matchs en escouade.
//
// KPIsByXUID + TeamAvgKPIs : agreges sur les matchs PARTAGES (meme scope que
// PlayerCards). Alimentent le SessionBriefing (drill-down click + reference
// trends ▲/▼). KPIsByXUID est cle par xuid (pas gamertag) pour matcher avec
// PlayerScoreCard.XUID cote front.
//
// Capability gating : si LoadFor a retourne ErrCapabilityNotSupported pour le
// joueur principal, le caller GetSquadPage a deja court-circuite. Pas besoin
// de gate explicite ici.
