// Package halo_5 — adapter_data_loaders.go : methodes DataAdapter carriere + chargement de
// matchs (LoadCareerSnapshot / LoadMatchSummaries / LoadMatchDetail + helpers). Extrait de
// adapter_data.go (K3f god-file split, 2026-07-06), meme package.
package halo_5

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
)

// LoadCareerSnapshot projette la carrière Halo 5 (palier CSR + rang XP « SR ») vers
// CareerSnapshot. `xuid` = GAMERTAG. Stratégie LIVE-FIRST → FALLBACK LOCAL : on tente
// d'abord le live (rang FRAIS, token du joueur) ; s'il est indisponible — token du
// joueur mort (RT révoqué), démo (aucun token), ou panne gracieuse — on sert le rang
// PERSISTÉ depuis le DuckDB synchronisé (title-agnostic, sans dépendre du token du
// joueur consulté). Ainsi la carrière d'un joueur SUIVI par l'app ne disparaît jamais
// parce que SON refresh_token est mort.
func (a *DataAdapter) LoadCareerSnapshot(ctx context.Context, xuid string, opts canonical.CareerOptions) (*canonical.CareerSnapshot, error) {
	gamertag := xuid
	if snap, ok := a.liveCareerSnapshot(ctx, gamertag); ok {
		a.attachH5XPHistory(ctx, snap, opts)
		return snap, nil
	}
	if a.careerLocal != nil {
		if snap := a.localCareerSnapshot(ctx, gamertag); snap != nil {
			a.attachH5XPHistory(ctx, snap, opts)
			return snap, nil
		}
	}
	// Aucune source exploitable : si même la voie live n'est pas câblée (builder
	// global live-only, newSource nil) ET pas de local → capability indisponible
	// (le caller masque). Sinon snapshot d'identité minimal (live câblé mais vide).
	if a.newSource == nil && a.careerLocal == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	return &canonical.CareerSnapshot{Player: h5Identity(gamertag)}, nil
}

// attachH5XPHistory peuple snap.History depuis career_progression quand
// opts.IncludeHistory est demandé et qu'une source locale est câblée. Parité avec
// Halo Infinite (adapter_data.projectCareerHistory) : le rang COURANT vient de la voie
// live/local, l'historique XP vient TOUJOURS du substrat DuckDB synchronisé (le carnage
// h5 ne porte pas l'historique). Best-effort : une lecture échouée laisse l'historique
// vide (le graphe « Historique XP » se masque) sans casser la page Carrière.
func (a *DataAdapter) attachH5XPHistory(ctx context.Context, snap *canonical.CareerSnapshot, opts canonical.CareerOptions) {
	if snap == nil || !opts.IncludeHistory || a.careerLocal == nil {
		return
	}
	hist, err := a.careerLocal.GetXPHistory(ctx)
	if err != nil {
		a.logger.DebugContext(ctx, "h5 career: GetXPHistory échouée (historique XP vide)", "err", err)
		return
	}
	snap.History = projectH5CareerHistory(hist)
}

// projectH5CareerHistory projette l'historique XP domaine → canonique (parité stricte
// avec halo_infinite.projectCareerHistory). nil si vide (le service ré-initialise
// nil→[] après projection).
func projectH5CareerHistory(rows []domain.XPHistoryPoint) []canonical.CareerHistoryEntry {
	if len(rows) == 0 {
		return nil
	}
	out := make([]canonical.CareerHistoryEntry, 0, len(rows))
	for _, p := range rows {
		cur, tot := p.CurrentXP, p.XPTotal
		out = append(out, canonical.CareerHistoryEntry{
			RecordedAt: p.RecordedAt,
			RankNumber: p.Rank,
			CurrentXP:  &cur,
			XPTotal:    &tot,
		})
	}
	return out
}

// liveCareerSnapshot tente la voie LIVE (CSR service record + SR carnage du dernier
// match). (snap, true) si une source live a répondu et produit un snapshot ;
// (nil, false) si la source/token est absente ou l'appel échoue — gracieux
// (401/403/404, signal re-auth) OU panne dure (réseau/5xx/décodage), tous deux logués
// et NON propagés : le fallback local peut couvrir. Le caller bascule alors dessus.
func (a *DataAdapter) liveCareerSnapshot(ctx context.Context, gamertag string) (*canonical.CareerSnapshot, bool) {
	src, err := a.resolveSource(ctx)
	if err != nil {
		return nil, false
	}
	rctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()
	resp, err := src.GetServiceRecords(rctx, gamertag, h5RecordModeArena)
	if err != nil {
		if !a.degradeUnavailable(rctx, err, gamertag, "LoadCareerSnapshot") {
			a.logger.WarnContext(rctx, "h5 career live: échec dur (fallback local)", "player", gamertag, "err", err)
		}
		return nil, false
	}
	snap := mapCareerSnapshot(resp, gamertag, a.placementTotal)
	if snap == nil {
		return nil, false
	}
	a.enrichSpartanRank(rctx, src, gamertag, snap)
	return snap, true
}

// enrichSpartanRank ajoute le rang XP (SR) au CareerSnapshot, lu dans la carnage du
// DERNIER match (XpInfo.SpartanRank/TotalXP — seule source du SR : ni la liste de
// matchs ni le service record ne le portent). BEST-EFFORT : toute indisponibilité
// (token/404/absence/match introuvable) laisse le snapshot CSR intact, sans erreur.
func (a *DataAdapter) enrichSpartanRank(ctx context.Context, src h5Source, gamertag string, snap *canonical.CareerSnapshot) {
	matches, err := src.GetPlayerMatches(ctx, gamertag, 0, 1)
	if err != nil || matches == nil || len(matches.Results) == 0 {
		return
	}
	m := matches.Results[0]
	if m.Id.MatchId == "" {
		return
	}
	carnage, err := src.GetMatchCarnage(ctx, m.Id.MatchId, h5GameModeSegment(m.Id.GameMode))
	if err != nil || carnage == nil {
		return
	}
	for i := range carnage.PlayerStats {
		p := &carnage.PlayerStats[i]
		if p.XpInfo != nil && strings.EqualFold(p.Player.Gamertag, gamertag) {
			applySpartanRank(snap, p.XpInfo.SpartanRank, p.XpInfo.TotalXP)
			return
		}
	}
}

// degradeUnavailable retourne true (et logue) si l'erreur est une indisponibilite
// gracieuse : 404/410 (le joueur n'a pas de record sur ce mode) OU 401/403 (token
// expire/insuffisant -> signal de re-auth, pas une panne data ; un endpoint
// read-only de profil ne doit pas casser la page). Les autres erreurs (reseau,
// 5xx, decode) sont des pannes a propager.
func (a *DataAdapter) degradeUnavailable(ctx context.Context, err error, gamertag, op string) bool {
	var he *HTTPError
	if !errors.As(err, &he) {
		return false
	}
	switch he.StatusCode {
	case http.StatusNotFound, http.StatusGone:
		a.logger.DebugContext(ctx, "h5 record absent", "op", op, "player", gamertag, "status", he.StatusCode)
		return true
	case http.StatusUnauthorized, http.StatusForbidden:
		a.logger.WarnContext(ctx, "h5 token expire/insuffisant (re-auth requise)", "op", op, "player", gamertag, "status", he.StatusCode)
		return true
	}
	return false
}

// LoadMatchSummaries projette l'historique de matchs Halo 5 vers []MatchSummary
// depuis le substrat LOCAL synchronisé (shared_matches_v2.duckdb du titre :
// match_registry ⨝ match_participants), comme le MatchHistoryRepo d'Infinite —
// PAS de fetch live (la donnée y est déjà écrite par le livesync).
//
//   - matchIDs nil/vide → les N derniers matchs du joueur (ORDER BY start_time DESC) ;
//   - matchIDs non vide → filtre sur ces IDs, ordre d'entrée préservé.
//
// Dégradation propre : source non câblée (nil) → ErrCapabilityNotSupported (le
// caller dégrade). C'est aussi le cas sous le builder global live-only (pas de
// PlayerDB) : seul le builder player-scoped injecte la source.
func (a *DataAdapter) LoadMatchSummaries(ctx context.Context, matchIDs []string) ([]canonical.MatchSummary, error) {
	if a.matchHistory == nil {
		return nil, games.ErrCapabilityNotSupported
	}
	summaries, err := a.matchHistory.GetMatchSummaries(ctx, matchIDs)
	if err != nil {
		return nil, fmt.Errorf("h5 LoadMatchSummaries: %w", err)
	}
	return summaries, nil
}

// h5MatchDetailMaxPages borne la recherche du matchID dans l'historique du viewer
// (GetPlayerMatches est player+page-based, pas ID-based). 8 pages × 25 = 200 matchs
// récents couverts ; au-delà, on dégrade (un match très ancien n'est de toute façon
// pas une cible Match View live réaliste).
const (
	h5MatchDetailMaxPages = 8
	h5MatchDetailPageSize = 25
)

// LoadMatchDetail assemble un canonical.MatchDetail d'un match Halo 5 depuis la
// donnée LIVE (best-effort). Flux :
//  1. résoudre le GAMERTAG du viewer depuis le contexte (ctxkeys.ViewerGamertag) —
//     Halo 5 est gamertag-keyé et la carnage exige le MODE + les refs header issus
//     de l'entrée d'historique du joueur (GetPlayerMatches) ;
//  2. paginer GetPlayerMatches(gamertag) pour retrouver l'entrée du matchID
//     (→ mode du carnage + refs header via le mapper summary réutilisé) ;
//  3. fetch GetMatchCarnage(matchID, mode) (roster complet) ;
//  4. projeter carnage + header → MatchDetail (mappers parallèles à l'ingestion).
//
// DEGRADATION (→ nil, ErrCapabilityNotSupported, le service Part B retombe sur le
// repo) : viewer gamertag absent du contexte, source/token indisponible, matchID
// introuvable dans l'historique récent, carnage 404/vide. Toute indisponibilité
// gracieuse (token expiré, 404) est loguée, pas propagée en erreur dure.
func (a *DataAdapter) LoadMatchDetail(ctx context.Context, matchID string) (*canonical.MatchDetail, error) {
	matchID = strings.TrimSpace(matchID)
	if matchID == "" {
		return nil, games.ErrCapabilityNotSupported
	}
	gamertag := strings.TrimSpace(ctxkeys.ViewerGamertag(ctx))
	if gamertag == "" {
		// Sans viewer, ni le MODE de la carnage ni les refs header ne sont
		// résolvables (Player.Xuid null, pas d'historique). Dégradation propre.
		a.logger.DebugContext(ctx, "h5 LoadMatchDetail: viewer gamertag absent du contexte (dégradation)",
			"match_id", matchID)
		return nil, games.ErrCapabilityNotSupported
	}
	src, err := a.resolveSource(ctx)
	if err != nil {
		return nil, games.ErrCapabilityNotSupported
	}
	ctx, cancel := context.WithTimeout(ctx, h5RequestTimeout)
	defer cancel()

	header, gameMode, found := a.findMatchInHistory(ctx, src, gamertag, matchID)
	if !found {
		a.logger.DebugContext(ctx, "h5 LoadMatchDetail: match introuvable dans l'historique récent (dégradation)",
			"player", gamertag, "match_id", matchID)
		return nil, games.ErrCapabilityNotSupported
	}

	carnage, err := src.GetMatchCarnage(ctx, matchID, h5GameModeSegment(gameMode))
	if err != nil {
		if a.degradeUnavailable(ctx, err, gamertag, "LoadMatchDetail") {
			return nil, games.ErrCapabilityNotSupported
		}
		return nil, fmt.Errorf("h5 LoadMatchDetail(%s): %w", matchID, err)
	}
	detail := mapCarnageToCanonicalDetail(matchID, gamertag, header, carnage)
	if detail == nil {
		a.logger.DebugContext(ctx, "h5 LoadMatchDetail: carnage vide (dégradation)",
			"player", gamertag, "match_id", matchID)
		return nil, games.ErrCapabilityNotSupported
	}
	a.enrichCommendations(ctx, detail) // best-effort : noms + icônes natifs (AXE B)
	return detail, nil
}

// findMatchInHistory pagine l'historique du viewer pour retrouver l'entrée du
// matchID. Retourne le header canonique (refs map/playlist/variant + dates +
// isRanked, via le mapper summary réutilisé) + le GameMode (carnage) + found.
// Best-effort : toute erreur de page (token/404/réseau) arrête la pagination et
// renvoie found=false (le caller dégrade).
func (a *DataAdapter) findMatchInHistory(ctx context.Context, src h5Source, gamertag, matchID string) (*canonical.MatchSummary, int, bool) {
	seen := make(map[string]struct{}) // garde anti-boucle si l'API n'honore pas `start`
	for page := 0; page < h5MatchDetailMaxPages; page++ {
		resp, err := src.GetPlayerMatches(ctx, gamertag, page*h5MatchDetailPageSize, h5MatchDetailPageSize)
		if err != nil {
			a.degradeUnavailable(ctx, err, gamertag, "LoadMatchDetail.history")
			return nil, 0, false
		}
		if resp == nil || len(resp.Results) == 0 {
			return nil, 0, false // fin de l'historique
		}
		for i := range resp.Results {
			r := &resp.Results[i]
			if _, dup := seen[r.Id.MatchId]; dup {
				return nil, 0, false // pagination non avançante -> stop défensif
			}
			seen[r.Id.MatchId] = struct{}{}
			if h5HeaderMatchID(r.Id.MatchId, matchID) {
				summary := mapOneMatchSummary(r, gamertag, a.classifier)
				return &summary, r.Id.GameMode, true
			}
		}
	}
	return nil, 0, false
}
