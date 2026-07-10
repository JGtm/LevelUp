// Package service - teammates_squad_charts_medal_digest.go : load impact
// events + medal digest builder + helpers. Decoupe de
// teammates_squad_charts.go (god-file split, refactor 2026-05-27).
package teammates

import (
	"context"
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/assets/static"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// CorrectSquadImpactEvents ramène les TimeMS au référentiel gameplay (T0 /
// countdown pré-match retranché, §4.A-bis) et émet un log d'observabilité
// (combien de matchs du lot portaient un countdown réel) → logs/service.log,
// grep "squad_t0_applied". Centralise le pattern partagé par les 4 consommateurs
// d'events de la page Escouade (.17, .07, .13, squad V1). timelines nil ou
// match absent → identité (T0=0).
func CorrectSquadImpactEvents(
	ctx context.Context,
	chart string,
	events []domain.ImpactEventRow,
	timelines map[string]domain.MatchTimeline,
) []domain.ImpactEventRow {
	corrected := timeline.CorrectImpactEvents(events, timelines)
	matchesWithT0 := 0
	for _, tl := range timelines {
		if tl.T0Ms > 0 {
			matchesWithT0++
		}
	}
	slog.DebugContext(ctx, "squad_t0_applied",
		"chart", chart,
		"n_events", len(corrected),
		"n_matches", len(timelines),
		"n_matches_with_t0", matchesWithT0)
	return corrected
}

// timelines fournit le T0 par match (§4.A-bis) ; les TimeMS sont ramenés au
// référentiel gameplay avant conversion en analysis.ImpactEvent. Une map nil ou
// un match absent → T0=0 (identité).
func (s *TeammatesService) loadImpactEventsByMatch(
	ctx context.Context,
	matchIDs []string,
	timelines map[string]domain.MatchTimeline,
) map[string][]analysis.ImpactEvent {
	out := make(map[string][]analysis.ImpactEvent, len(matchIDs))
	if s.repo == nil || len(matchIDs) == 0 {
		return out
	}
	rows, err := s.repo.LoadImpactEvents(ctx, matchIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates_impact_events_load_failed",
			"err", err, "n_matches", len(matchIDs))
		return out
	}
	rows = CorrectSquadImpactEvents(ctx, "teammates.07", rows, timelines)
	for _, r := range rows {
		// EventType de ImpactEventRow est le BadgeKey original ou un type kill/death.
		// analysis.ComputeMatchImpactFull attend EventType == "kill" ou "death".
		// On dérive depuis le BadgeKey si présent ; sinon on laisse passer tel quel.
		ev := analysis.ImpactEvent{
			TimeMS:    r.TimeMS,
			EventType: r.EventType,
			ActorXUID: r.XUID,
		}
		out[r.MatchID] = append(out[r.MatchID], ev)
	}
	return out
}

// ---------------------------------------------------------------------------
// MedalDigest — résumé narratif médailles par joueur (SquadSynergiesPage)
// ---------------------------------------------------------------------------

// buildMedalDigest agrège les médailles de chaque joueur sur les matchs
// partagés et retourne un []domain.MedalDigestEntry trié par gamertag
// (main player en tête). Retourne nil si squadLoader ou medalDefs sont absents.
func (s *TeammatesService) buildMedalDigest(
	ctx context.Context,
	allSquadRows []domain.SquadMatchRow,
	mainGamertag, mainXUID string,
	teammates []domain.TeammateRow,
	locale string,
) []domain.MedalDigestEntry {
	if s.squadLoader == nil || len(allSquadRows) == 0 || len(teammates) == 0 {
		return nil
	}
	sharedMatches := collectSharedMatchIDsForDigest(allSquadRows)
	if len(sharedMatches) == 0 {
		return nil
	}
	players := collectDigestPlayerXUIDs(mainGamertag, mainXUID, teammates)
	if len(players) == 0 {
		return nil
	}
	xuids := make([]string, len(players))
	for i, p := range players {
		xuids[i] = p.xuid
	}
	rows, err := s.squadLoader.LoadMedals(ctx, s.titleSlug, port.MedalsByXUIDFilters{
		MatchIDs: sharedMatches,
		XUIDs:    xuids,
	})
	if err != nil || len(rows) == 0 {
		if err != nil {
			slog.WarnContext(ctx, "teammates_medal_digest_load_failed", "err", err)
		}
		return nil
	}
	gamertags := make([]string, len(players))
	for i, p := range players {
		gamertags[i] = p.gamertag
	}
	emblems := s.squadLoader.LoadEmblemURLs(ctx, s.titleSlug, gamertags)
	defs := resolveMedalDigestDefs(ctx, s.medalDefs, rows, locale)
	return assembleMedalDigest(rows, players, defs, emblems, s.titleSlug)
}

// collectSharedMatchIDsForDigest retourne les match_id distincts de
// allSquadRows. L'input est déjà l'intersection composition exacte — 1 row
// par match, dédupliquée par intersectSquadRowsByMatchID (commit 851e10ef5).
// L'ancien seuil d'occurrences (n >= minTeammates) datait de l'ère "union
// avec doublons par coéquipier" : sur l'input dédupliqué il rendait le digest
// TOUJOURS vide dès 2 coéquipiers sélectionnés (1 >= 2 faux).
func collectSharedMatchIDsForDigest(allSquadRows []domain.SquadMatchRow) []string {
	seen := make(map[string]struct{}, len(allSquadRows))
	out := make([]string, 0, len(allSquadRows))
	for _, m := range allSquadRows {
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		out = append(out, m.MatchID)
	}
	return out
}

type digestPlayer struct {
	gamertag string
	xuid     string
}

// collectDigestPlayerXUIDs construit la liste ordonnée (main en tête + teammates
// avec xuid résolu) pour le chargement des médailles.
func collectDigestPlayerXUIDs(mainGT, mainXUID string, teammates []domain.TeammateRow) []digestPlayer {
	players := make([]digestPlayer, 0, 1+len(teammates))
	if mainXUID != "" {
		players = append(players, digestPlayer{mainGT, mainXUID})
	}
	for _, tm := range teammates {
		if tm.XUID == nil || *tm.XUID == "" {
			continue
		}
		players = append(players, digestPlayer{tm.Gamertag, *tm.XUID})
	}
	return players
}

// resolveMedalDigestDefs charge les définitions (label + description) pour les
// medal_ids présents dans rows. Tolère un repo nil (retourne map vide).
func resolveMedalDigestDefs(
	ctx context.Context,
	repo port.MedalDefinitionsRepository,
	rows []port.MedalRow,
	locale string,
) map[int64]port.MedalDefinitionRow {
	if repo == nil {
		return nil
	}
	seen := make(map[int64]struct{}, len(rows))
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.MedalID]; !ok {
			seen[r.MedalID] = struct{}{}
			ids = append(ids, r.MedalID)
		}
	}
	defs, err := repo.LookupByIDs(ctx, ids, locale)
	if err != nil {
		slog.WarnContext(ctx, "teammates_medal_digest_defs_failed", "err", err)
		return nil
	}
	return defs
}

// assembleMedalDigest construit les entrées digest par joueur à partir des
// rows de médailles + définitions résolues + emblèmes.
func assembleMedalDigest(
	rows []port.MedalRow,
	players []digestPlayer,
	defs map[int64]port.MedalDefinitionRow,
	emblems map[string]string,
	titleSlug string,
) []domain.MedalDigestEntry {
	type agg struct{ totalCount, matchCount int }
	perXUID := make(map[string]map[int64]*agg, len(players))
	perXUIDMatch := make(map[string]map[string]int, len(players))
	for _, r := range rows {
		if _, ok := perXUID[r.XUID]; !ok {
			perXUID[r.XUID] = make(map[int64]*agg)
			perXUIDMatch[r.XUID] = make(map[string]int)
		}
		a := perXUID[r.XUID]
		if _, ok := a[r.MedalID]; !ok {
			a[r.MedalID] = &agg{}
		}
		a[r.MedalID].totalCount += r.Count
		a[r.MedalID].matchCount++
		perXUIDMatch[r.XUID][r.MatchID] += r.Count
	}

	out := make([]domain.MedalDigestEntry, 0, len(players))
	for _, p := range players {
		byMedal := perXUID[p.xuid]
		if len(byMedal) == 0 {
			continue
		}
		items := make([]domain.MedalDigestItem, 0, len(byMedal))
		total := 0
		for medalID, ma := range byMedal {
			def := defs[medalID]
			item := domain.MedalDigestItem{
				MedalID:       medalID,
				Label:         def.Label,
				Description:   def.Description,
				TotalCount:    ma.totalCount,
				MatchCount:    ma.matchCount,
				Category:      def.MedalType,
				Difficulty:    def.Difficulty,
				PersonalScore: def.PersonalScore,
			}
			if titleSlug != "" {
				png, sp := static.MedalImage(titleSlug, medalID)
				if sp != nil {
					item.SpriteSheet, item.SpriteLeft, item.SpriteTop, item.SpriteWidth, item.SpriteHeight =
						sp.SheetURL, sp.Left, sp.Top, sp.Width, sp.Height
				} else {
					item.ImageURL = png
				}
			}
			items = append(items, item)
			total += ma.totalCount
		}
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].TotalCount != items[j].TotalCount {
				return items[i].TotalCount > items[j].TotalCount
			}
			return items[i].MedalID < items[j].MedalID
		})
		peak := 0
		for _, n := range perXUIDMatch[p.xuid] {
			if n > peak {
				peak = n
			}
		}
		avg := 0.0
		if nm := len(perXUIDMatch[p.xuid]); nm > 0 {
			avg = float64(total) / float64(nm)
		}
		top := items
		if len(top) > 5 {
			top = items[:5]
		}
		emblemURL := ""
		if emblems != nil {
			emblemURL = emblems[p.gamertag]
		}
		out = append(out, domain.MedalDigestEntry{
			Player:        p.gamertag,
			EmblemURL:     emblemURL,
			DistinctTypes: len(byMedal),
			TotalCount:    total,
			AvgPerMatch:   avg,
			PeakInMatch:   peak,
			TopMedals:     top,
			AllMedals:     items,
		})
	}
	return out
}
