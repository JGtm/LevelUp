// Package service - teammates_service_assets.go : asset enrichment +
// collectUniqueIDs + modeLabel + computeMapBreakdown + collectModeENs +
// buildSquadMatchHistory + buildMatchSeries. Decoupe de teammates_service.go
// (god-file split, refactor 2026-05-27).
package teammates

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

func enrichSquadMatchAssets(ctx context.Context, repo port.SquadRepository, rows []domain.SquadMatchRow) {
	mapIDs := collectUniqueIDs(rows, func(r domain.SquadMatchRow) string { return r.MapID })
	playlistIDs := collectUniqueIDs(rows, func(r domain.SquadMatchRow) string { return r.PlaylistID })
	pairIDs := collectUniqueIDs(rows, func(r domain.SquadMatchRow) string { return r.PairID })
	gameVariantIDs := collectUniqueIDs(rows, func(r domain.SquadMatchRow) string { return r.GameVariantID })

	mapFR, err := repo.LoadAssetTranslationsFR(ctx, games.AssetKindMap, mapIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates: LoadAssetTranslationsFR map failed", "err", err)
	}
	playlistFR, err := repo.LoadAssetTranslationsFR(ctx, games.AssetKindPlaylist, playlistIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates: LoadAssetTranslationsFR playlist failed", "err", err)
	}
	pairAssetFR, err := repo.LoadAssetTranslationsFR(ctx, games.AssetKindPair, pairIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates: LoadAssetTranslationsFR pair failed", "err", err)
	}
	// game_variant (FR) : source du mode pour les titres SANS pair_name (Halo 5).
	// Le mode = nom de la variante de jeu résolu depuis game_variant_id. Même
	// résolveur que map/playlist/pair (asset_translations) — read-time, zéro
	// backfill. Title-agnostic : Infinite a pair_name → ce fallback n'est pas
	// consulté (squadModeUI le préfère). Un 3e titre sans pair_name en bénéficie.
	gameVariantFR, err := repo.LoadAssetTranslationsFR(ctx, games.AssetKindGameVariant, gameVariantIDs)
	if err != nil {
		slog.WarnContext(ctx, "teammates: LoadAssetTranslationsFR game_variant failed", "err", err)
	}
	// mode_name_tr (FR) des modes EN normalisés — dérivés du pair_name brut ET
	// des noms d'asset résolus, pour couvrir le cas pair_name=UUID. Même cascade
	// canonique que match_history (applyMatchHistoryFRTranslations).
	modeFR := loadSquadModeFR(ctx, repo, rows, pairAssetFR)

	for i := range rows {
		if fr := strings.TrimSpace(mapFR[rows[i].MapID]); fr != "" {
			rows[i].MapUI = fr
		}
		if fr := strings.TrimSpace(playlistFR[rows[i].PlaylistID]); fr != "" {
			rows[i].PlaylistName = fr
		}
		if fr := strings.TrimSpace(gameVariantFR[rows[i].GameVariantID]); fr != "" {
			rows[i].GameVariantNameFR = fr
		}
		// Résolution canonique du libellé FR du mode (source unique partagée avec
		// home / historique / filtres). Gère pair_name brut, vide ou UUID.
		rows[i].PairNameFR = analysis.ResolvePairNameFR(
			rows[i].PairName, rows[i].PairNameFR, pairAssetFR[rows[i].PairID], modeFR)
	}
}

// loadSquadModeFR charge mode_name_tr (FR) pour les modes EN normalisés issus
// des pair_name bruts ET des noms d'asset résolus (pair_id). Miroir de
// match_history : nécessaire car un pair_name brut peut être un UUID, auquel cas
// le nom EN exploitable vient de asset_translations[pair_id].
func loadSquadModeFR(
	ctx context.Context,
	repo port.SquadRepository,
	rows []domain.SquadMatchRow,
	pairAssetFR map[string]string,
) map[string]string {
	seen := make(map[string]struct{}, 16)
	modeENs := make([]string, 0, 16)
	add := func(raw string) {
		if en := analysis.NormalizeModeLabel(raw); en != "" {
			if _, ok := seen[en]; !ok {
				seen[en] = struct{}{}
				modeENs = append(modeENs, en)
			}
		}
	}
	for _, r := range rows {
		add(r.PairName)
		if r.PairID != "" {
			add(pairAssetFR[r.PairID])
		}
	}
	if len(modeENs) == 0 {
		return nil
	}
	modeFR, err := repo.LoadModeTranslationsFR(ctx, modeENs)
	if err != nil {
		slog.WarnContext(ctx, "teammates: LoadModeTranslationsFR failed", "err", err)
	}
	return modeFR
}

func collectUniqueIDs(rows []domain.SquadMatchRow, idOf func(domain.SquadMatchRow) string) []string {
	seen := make(map[string]struct{}, len(rows))
	result := make([]string, 0, len(rows))
	for _, r := range rows {
		id := idOf(r)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

// squadModeUI calcule le libellé de mode affiché dans la table Escouade.
// Normalise la meilleure source déjà résolue (PairNameFR via la cascade
// canonique, sinon EN brut), puis masque un éventuel UUID résiduel (trou de
// metadata) pour ne JAMAIS afficher d'UUID — même garde que cleanAssetLabel
// côté home. Retourne "" si rien d'exploitable (le front affiche alors "-").
//
// Fallback title-agnostic (presence-driven) : si la source pair_name/pair_name_fr
// est vide (titre SANS pair_name — Halo 5), le mode retombe sur le nom de la
// variante de jeu (GameVariantNameFR, résolu depuis game_variant_id via
// asset_translations). Infinite (qui a pair_name) est inchangé : la source pair
// est préférée. Aucun gating par titre — c'est une résolution de données.
func squadModeUI(m domain.SquadMatchRow) string {
	src := m.PairNameFR
	if strings.TrimSpace(src) == "" {
		src = m.PairName
	}
	if strings.TrimSpace(src) == "" {
		// Titre sans pair_name : le mode = nom de la variante de jeu.
		src = m.GameVariantNameFR
	}
	ui := analysis.NormalizeModeLabel(src, m.MapUI)
	if analysis.IsRawAssetUUID(ui) {
		return ""
	}
	return ui
}

// computeMapBreakdown agrège les stats par carte depuis les matchs escouade.
// PerformanceAvg = moyenne des PerformanceScore non nil ; nil si aucun.
//
// Résultat trié par ordre CHRONOLOGIQUE de première apparition (firstSeen =
// StartTime minimal des matchs de la carte), tie-break MapUI pour un ordre
// déterministe — l'itération d'une map Go ne l'est pas, ce qui produisait un
// ordre aléatoire d'un appel à l'autre avant ce tri (bug latent). Même
// stratégie de tri que buildSquadMapHeatmap
// (teammates_squad_charts_sessions_maps.go), pour une cohérence d'ordre entre
// les charts teammates.02/13 et la heatmap teammates.03.
func computeMapBreakdown(matches []domain.SquadMatchRow) []domain.MapBreakdownRow {
	type stats struct {
		mapUI       string
		count, wins int
		perfSum     float64
		perfCount   int
		firstSeen   time.Time
	}
	m := map[string]*stats{}
	for _, r := range matches {
		// Clé interne = UUID si dispo (language-agnostic), sinon label d'affichage.
		key := r.MapID
		if key == "" {
			key = r.MapUI
		}
		if key == "" {
			key = tsLabelUnknown
		}
		s, ok := m[key]
		if !ok {
			lbl := r.MapUI
			if lbl == "" {
				lbl = tsLabelUnknown
			}
			s = &stats{mapUI: lbl, firstSeen: r.StartTime}
			m[key] = s
		} else if r.StartTime.Before(s.firstSeen) {
			s.firstSeen = r.StartTime
		}
		s.count++
		if r.Outcome == analysis.OutcomeWin {
			s.wins++
		}
		if r.PerformanceScore != nil {
			s.perfSum += *r.PerformanceScore
			s.perfCount++
		}
	}
	result := make([]domain.MapBreakdownRow, 0, len(m))
	for mapKey, s := range m {
		row := domain.MapBreakdownRow{
			MapID:      mapKey,
			MapUI:      s.mapUI,
			MatchCount: s.count,
			WinRate:    round2(float64(s.wins) / float64(s.count)),
		}
		if s.perfCount > 0 {
			avg := round2(s.perfSum / float64(s.perfCount))
			row.PerformanceAvg = &avg
		}
		result = append(result, row)
	}
	sort.Slice(result, func(i, j int) bool {
		fi, fj := m[result[i].MapID].firstSeen, m[result[j].MapID].firstSeen
		if !fi.Equal(fj) {
			return fi.Before(fj)
		}
		return result[i].MapUI < result[j].MapUI
	})
	return result
}

// buildSquadMatchHistory construit la table historique pour teammates.11 :
// une ligne par match unique, triée par StartTime DESC. Pas de cap serveur —
// la pagination (20/page) est gérée côté client (TanStack Table).
//
// mapWR : (wins, total) par MapID sur l'historique complet du joueur
// principal — sert à injecter le taux historique par carte. Si nil ou clé
// absente, WinRateHist reste nil (la cellule front affiche "—").
//
// titleSlug : titre courant. Quand le titre ne fournit pas de MMR d'équipe
// (games.ProvidesTeamMMR=false, ex. Halo 5), TeamMMRAvg reste nil (la colonne MMR
// est masquée côté front) au lieu d'afficher un 0 trompeur.
func buildSquadMatchHistory(matches []domain.SquadMatchRow, mapWR map[string][2]int, titleSlug string) []domain.SquadMatchHistoryRow {
	provideTeamMMR := games.ProvidesTeamMMR(titleSlug)
	seen := make(map[string]struct{}, len(matches))
	rows := make([]domain.SquadMatchHistoryRow, 0, len(matches))
	for _, m := range matches {
		if m.MatchID == "" {
			continue
		}
		if _, dup := seen[m.MatchID]; dup {
			continue
		}
		seen[m.MatchID] = struct{}{}
		// MMR (équipe / adverse / delta) : nil quand le titre ne fournit pas de MMR
		// d'équipe (Halo 5) → la colonne MMR est masquée côté front au lieu d'afficher
		// un 0 trompeur. Les trois champs partent ensemble (delta dérive des deux).
		var teamMMR, deltaMMR, enemyMMR *float64
		if provideTeamMMR {
			tm := m.TeamMMR
			teamMMR = &tm
			enemyMMR = m.EnemyMMR
			if m.EnemyMMR != nil {
				d := m.TeamMMR - *m.EnemyMMR
				deltaMMR = &d
			}
		}
		var scoreLabel string
		if m.MyTeamScore != nil && m.EnemyTeamScore != nil {
			scoreLabel = fmt.Sprintf("%d - %d", *m.MyTeamScore, *m.EnemyTeamScore)
		}
		var winRate *float64
		var winRateTotal *int
		if mapWR != nil {
			key := m.MapID
			if key == "" {
				key = m.MapName
			}
			if key != "" {
				if entry, ok := mapWR[key]; ok && entry[1] > 0 {
					v := round2(float64(entry[0]) / float64(entry[1]))
					winRate = &v
					total := entry[1]
					winRateTotal = &total
				}
			}
		}
		modeUI := squadModeUI(m)
		rows = append(rows, domain.SquadMatchHistoryRow{
			MatchID:      m.MatchID,
			StartTime:    m.StartTime.Format("2006-01-02T15:04:05Z"),
			MapUI:        m.MapUI,
			PlaylistName: m.PlaylistName,
			// PairName (json pair_name) = même libellé résolu que ModeUI : sert de
			// fallback front et ne doit donc jamais être un UUID.
			PairName:                modeUI,
			ModeUI:                  modeUI,
			Outcome:                 m.Outcome,
			Kills:                   m.Kills,
			Deaths:                  m.Deaths,
			Assists:                 m.Assists,
			Accuracy:                m.Accuracy,
			PerformanceScore:        m.PerformanceScore,
			TeamMMRAvg:              teamMMR,
			EnemyMMRAvg:             enemyMMR,
			DeltaMMR:                deltaMMR,
			ScoreLabel:              scoreLabel,
			DurationSeconds:         m.DurationSeconds,
			GameplayDurationSeconds: m.GameplayDurationSeconds,
			WinRateHist:             winRate,
			WinRateHistTotal:        winRateTotal,
			ExpectedWinProb:         m.ExpectedWinProb,
			SessionLabel:            m.SessionLabel,
			DominanceFlag:           m.DominanceFlag,
		})
	}
	slices.SortFunc(rows, func(a, b domain.SquadMatchHistoryRow) int {
		return cmp.Compare(b.StartTime, a.StartTime) // DESC
	})
	return rows
}

// buildMatchSeries construit la sÃƒÂ©rie temporelle des matchs pour un coÃƒÂ©quipier.
func buildMatchSeries(matches []domain.SquadMatchRow) []domain.SquadMatchSeriesPoint {
	series := make([]domain.SquadMatchSeriesPoint, 0, len(matches))
	for _, m := range matches {
		series = append(series, domain.SquadMatchSeriesPoint{
			MatchID:          m.MatchID,
			StartTime:        m.StartTime.Format("2006-01-02T15:04:05Z"),
			Outcome:          m.Outcome,
			PerformanceScore: m.PerformanceScore,
			TeamMMRAvg:       m.TeamMMR,
			SessionLabel:     m.SessionLabel,
		})
	}
	return series
}
