// Package service — match_view_friends_extras.go : extras per-friend du scoreboard (panneau
// d'expander) et expected K/D local des amis trackés. Sorti de match_view_data_loaders.go.
//
// Ces lectures tournent APRÈS le chargement parallèle, donc hors de l'errgroup : leur échec est
// dit par les mêmes codes que celui d'un chargeur (`failSection` : journal ERREUR puis
// `section_unavailable`), jamais par une section vide silencieuse.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// loadFriendsExtras charge les extras per-friend depuis la player DB de chaque ami du scoreboard.
// Sans loader injecté ou sans ami à lire : map nil, rien à dire. Une panne de lecture d'une DB
// amie rend les extras des autres amis ET est dite (`section_unavailable`).
func (s *MatchViewService) loadFriendsExtras(
	ctx context.Context, matchID string, meta *domain.MatchMetaRaw, d matchViewData,
) map[string]port.FriendMatchExtras {
	if s.friendsExtras == nil {
		return nil
	}
	xuids := make([]string, 0, len(d.scoreboard))
	for _, sb := range d.scoreboard {
		if sb.XUID != "" && sb.XUID != s.xuid {
			xuids = append(xuids, sb.XUID)
		}
	}
	if len(xuids) == 0 {
		return nil
	}
	gvn := ""
	if meta != nil && meta.GameVariantName != nil {
		gvn = *meta.GameVariantName
	}
	extras, err := s.friendsExtras(ctx, matchID, gvn, xuids)
	d.failures.failSection(ctx, matchID, "friends_extras", err)
	return extras
}

// applyFriendsLocalExpected pose l'expected K/D LOCAL (modèle count∝durée) sur la ligne des amis
// TRACKÉS du scoreboard : leur historique complet est dans shared (joueur synchronisé), chargeable
// par xuid, d'où un drawer attendu-vs-réel pour eux aussi (cohérent avec is_me). Limité aux xuids
// présents dans friendsExtras (l'historique d'un non-tracké ne contiendrait que les matchs communs
// avec l'escouade : échantillon biaisé). Skip si l'API a déjà fourni les K/D (Infinite).
//
// Tous les xuids éligibles sont chargés en UN seul appel (GetHistoryForAvgBulk). Sa panne est dite
// (`section_unavailable`) avant de dégrader sans expected K/D pour les amis.
func (s *MatchViewService) applyFriendsLocalExpected(
	ctx context.Context, matchID string, meta *domain.MatchMetaRaw, d matchViewData,
	friendsExtras map[string]port.FriendMatchExtras, curDurSec int,
) {
	if curDurSec <= 60 || len(friendsExtras) == 0 {
		return
	}
	eligible := func(i int) bool {
		xuid := d.scoreboard[i].XUID
		if xuid == s.xuid || d.scoreboard[i].KillsExpected != nil {
			return false
		}
		_, tracked := friendsExtras[xuid]
		return tracked
	}
	var needXUIDs []string
	seen := make(map[string]bool)
	for i := range d.scoreboard {
		if !eligible(i) {
			continue
		}
		if xuid := d.scoreboard[i].XUID; !seen[xuid] {
			seen[xuid] = true
			needXUIDs = append(needXUIDs, xuid)
		}
	}
	if len(needXUIDs) == 0 {
		return
	}
	histByXUID, err := s.repo.GetHistoryForAvgBulk(ctx, needXUIDs)
	if err != nil {
		d.failures.failSection(ctx, matchID, "friends_hist_avg", err)
		return
	}
	for i := range d.scoreboard {
		if !eligible(i) {
			continue
		}
		fh := histByXUID[d.scoreboard[i].XUID]
		if len(fh) == 0 {
			continue
		}
		if ek, ed, ok := localExpectedKD(fh, meta, curDurSec); ok {
			d.scoreboard[i].KillsExpected = ek
			d.scoreboard[i].DeathsExpected = ed
			d.scoreboard[i].LocallyEstimated = true
		}
	}
}
