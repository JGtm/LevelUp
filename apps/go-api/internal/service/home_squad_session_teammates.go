// Package service — home_squad_session_teammates.go : dérivation des coéquipiers
// d'une session escouade pour l'accueil. Alimente le deep-link card escouade →
// /squad (pré-sélection de la composition exacte).
package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/service/teammates"
)

// maxHomeSessionTeammates borne le nombre de coéquipiers attachés à une session
// escouade de l'accueil = MAX_SELECTION de la page Escouade (composition max 3).
const maxHomeSessionTeammates = 3

// mainTeamParticipantsLoader (optionnel) : charge les participants de l'équipe
// alliée du joueur principal sur une liste de matchs. Implémenté par
// *duckdb.SquadRepo. Permet de dériver les coéquipiers d'une session escouade.
type mainTeamParticipantsLoader interface {
	LoadMainTeamParticipants(ctx context.Context, mainXUID string, matchIDs []string) ([]domain.AllyParticipant, error)
}

// WithSquadSessionTeammates injecte de quoi renseigner SessionSummaryItem.Teammates
// sur les sessions escouade : un loader de participants alliés + un résolveur d'amis
// configurés (optionnel — restreint la composition aux amis déclarés). Chaînable.
func (s *HomeService) WithSquadSessionTeammates(loader mainTeamParticipantsLoader, friends teammates.FriendGamertagsResolver) *HomeService {
	s.sessionTeammatesLoader = loader
	s.sessionFriendsResolver = friends
	return s
}

// enrichSquadSessionsTeammates renseigne, best-effort, SessionSummaryItem.Teammates
// (gamertags des coéquipiers les plus présents, restreints aux amis configurés,
// top maxHomeSessionTeammates) sur les sessions escouade. No-op silencieux si le
// loader n'est pas câblé, si l'xuid est inconnu, ou en cas d'erreur DB — les cards
// dégradent alors en deep-link sans pré-sélection de composition.
func (s *HomeService) enrichSquadSessionsTeammates(
	ctx context.Context,
	sessions []domain.SessionSummaryItem,
	rows []canonical.PlayerMatchRow,
) {
	if s.sessionTeammatesLoader == nil || s.xuid == "" || len(sessions) == 0 {
		return
	}

	matchIDsByLabel, allIDs := squadSessionMatchIDs(sessions, rows)
	if len(allIDs) == 0 {
		return
	}

	allies, err := s.sessionTeammatesLoader.LoadMainTeamParticipants(ctx, s.xuid, allIDs)
	if err != nil {
		slog.WarnContext(ctx, "home: coéquipiers de session escouade indisponibles", "err", err)
		return
	}
	// match -> gamertags alliés (hors joueur principal).
	alliesByMatch := make(map[string][]string, len(allIDs))
	for _, a := range allies {
		if a.XUID == s.xuid || a.Gamertag == "" {
			continue
		}
		alliesByMatch[a.MatchID] = append(alliesByMatch[a.MatchID], a.Gamertag)
	}

	friendSet := s.configuredFriendSet(ctx)
	enriched := 0
	for i := range sessions {
		ids := matchIDsByLabel[sessions[i].SessionLabel]
		if len(ids) == 0 {
			continue
		}
		if tm := sessionCoreTeammates(ids, alliesByMatch, friendSet); len(tm) > 0 {
			sessions[i].Teammates = tm
			enriched++
		}
	}
	slog.DebugContext(ctx, "home: squad session teammates resolved",
		"player", s.gamertag,
		"squad_sessions", len(sessions),
		"sessions_with_teammates", enriched,
		"matches_scanned", len(allIDs),
	)
}

// squadSessionMatchIDs mappe chaque session escouade ciblée vers ses match IDs
// (depuis les rows canoniques is_with_friends) et retourne la liste plate des IDs.
func squadSessionMatchIDs(
	sessions []domain.SessionSummaryItem,
	rows []canonical.PlayerMatchRow,
) (map[string][]string, []string) {
	wanted := make(map[string]struct{}, len(sessions))
	for i := range sessions {
		wanted[sessions[i].SessionLabel] = struct{}{}
	}
	byLabel := make(map[string][]string, len(sessions))
	allIDs := make([]string, 0, len(rows))
	for i := range rows {
		e := rows[i].Enrichment
		if !e.IsWithFriends || e.SessionLabel == nil {
			continue
		}
		lbl := *e.SessionLabel
		if _, ok := wanted[lbl]; !ok {
			continue
		}
		mid := rows[i].Summary.MatchID
		if mid == "" {
			continue
		}
		byLabel[lbl] = append(byLabel[lbl], mid)
		allIDs = append(allIDs, mid)
	}
	return byLabel, allIDs
}

// configuredFriendSet retourne l'ensemble (clé = gamertag minuscule) des amis
// configurés via le résolveur, ou nil si aucun résolveur / aucun ami (= aucune
// restriction, on garde tous les coéquipiers alliés).
func (s *HomeService) configuredFriendSet(ctx context.Context) map[string]struct{} {
	if s.sessionFriendsResolver == nil {
		return nil
	}
	friends := s.sessionFriendsResolver(ctx)
	if len(friends) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(friends))
	for _, g := range friends {
		if g = strings.TrimSpace(g); g != "" {
			set[strings.ToLower(g)] = struct{}{}
		}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

// sessionCoreTeammates retourne le « cœur » de coéquipiers d'une session : ceux
// présents dans TOUS ses matchs (intersection), restreints aux amis configurés si
// friendSet est non nil, cappés à maxHomeSessionTeammates (ordre alphabétique
// déterministe). L'intersection est volontaire : elle garantit que la session reste
// dans la composition EXACTE côté /squad (decideCompositionReanchor sinon la
// remplacerait par la dernière session de la composition). Si aucun ami n'est commun
// à tous les matchs (amis tournants), retourne nil → /squad ouvre la session sans
// composition (chemin hasTeammates=false, qui conserve aussi la session).
func sessionCoreTeammates(
	matchIDs []string,
	alliesByMatch map[string][]string,
	friendSet map[string]struct{},
) []string {
	if len(matchIDs) == 0 {
		return nil
	}
	// Nombre de matchs où chaque coéquipier (clé minuscule) est présent.
	present := make(map[string]int)
	display := make(map[string]string)
	for _, mid := range matchIDs {
		seen := make(map[string]struct{}) // dédup intra-match (défensif)
		for _, gt := range alliesByMatch[mid] {
			key := strings.ToLower(gt)
			if friendSet != nil {
				if _, ok := friendSet[key]; !ok {
					continue
				}
			}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			present[key]++
			if _, ok := display[key]; !ok {
				display[key] = gt
			}
		}
	}
	// Cœur = présents dans TOUS les matchs de la session.
	n := len(matchIDs)
	core := make([]string, 0, len(present))
	for key, cnt := range present {
		if cnt == n {
			core = append(core, key)
		}
	}
	if len(core) == 0 {
		return nil
	}
	sort.Strings(core)
	limit := maxHomeSessionTeammates
	if len(core) < limit {
		limit = len(core)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, display[core[i]])
	}
	return out
}
