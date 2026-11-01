package halo5

// matches.go — les services de l'hôte stats : historique, service records, carnage, événements.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// decoder décode le corps JSON rendu à la méthode `op`.
func decoder[T any](op string, corps []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(corps, &v); err != nil {
		return nil, fmt.Errorf("%s decode: %w", op, err)
	}
	return &v, nil
}

// GetPlayerMatches recupere une page d'historique de matchs (tous modes confondus).
// GAMERTAG BRUT (pas xuid(NNN)). Path confirme sonde : /h5/players/{gt}/matches.
func (c *Client) GetPlayerMatches(ctx context.Context, auth Auth, gamertag string, start, count int) (*H5MatchesResponse, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, errors.New("GetPlayerMatches: gamertag vide")
	}
	// include-times=true : SANS ce param, l'API met le composant heure de
	// MatchCompletedDate à 00:00:00 (fidelity 1, jour seulement) → StartedAtUTC
	// dérivé (fin − durée) tombe à minuit−durée (faux). AVEC, MatchCompletedDate
	// est horodaté à la ms (fidelity 2) → heure de début exacte. Requis pour
	// l'association media par timestamp + la qualité temporelle des matchs.
	endpoint := fmt.Sprintf("%s/h5/players/%s/matches?start=%d&count=%d&include-times=true",
		c.endpoints.hoteStats(), url.PathEscape(gamertag), start, count)
	body, err := c.getJSON(ctx, auth, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetPlayerMatches(%s): %w", gamertag, err)
	}
	return decoder[H5MatchesResponse]("GetPlayerMatches", body)
}

// GetServiceRecords recupere le service record agrege d'un joueur pour un mode.
// recordType : "arena" | "warzone" | "campaign" | "custom". Path confirme sonde :
// /h5/servicerecords/{recordType}?players={gt}.
func (c *Client) GetServiceRecords(ctx context.Context, auth Auth, gamertag, recordType string) (*H5ServiceRecordResponse, error) {
	if strings.TrimSpace(gamertag) == "" {
		return nil, errors.New("GetServiceRecords: gamertag vide")
	}
	if strings.TrimSpace(recordType) == "" {
		return nil, errors.New("GetServiceRecords: recordType vide")
	}
	endpoint := fmt.Sprintf("%s/h5/servicerecords/%s?players=%s",
		c.endpoints.hoteStats(), url.PathEscape(recordType), url.QueryEscape(gamertag))
	body, err := c.getJSON(ctx, auth, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetServiceRecords(%s,%s): %w", gamertag, recordType, err)
	}
	return decoder[H5ServiceRecordResponse]("GetServiceRecords", body)
}

// GetMatchCarnage recupere le carnage report TYPE d'un match (roster complet +
// scoreboard etendu). mode = "arena"|"warzone"|... (cf. Links.StatsMatchDetails).
// Consommee par l'ingestion participants.
func (c *Client) GetMatchCarnage(ctx context.Context, auth Auth, matchID, mode string) (*H5CarnageResponse, error) {
	if strings.TrimSpace(matchID) == "" || strings.TrimSpace(mode) == "" {
		return nil, errors.New("GetMatchCarnage: matchID/mode vide")
	}
	endpoint := fmt.Sprintf("%s/h5/%s/matches/%s",
		c.endpoints.hoteStats(), url.PathEscape(mode), url.PathEscape(matchID))
	body, err := c.getJSON(ctx, auth, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetMatchCarnage(%s): %w", matchID, err)
	}
	return decoder[H5CarnageResponse]("GetMatchCarnage", body)
}

// GetMatchEvents recupere la timeline d'events NATIVE d'un match (kill-feed +
// arme-par-kill + medailles + positions monde, horodates). Path CONFIRME sonde :
// /h5/matches/{id}/events — SANS segment de mode (le /h5/{mode}/matches/{id}/
// events renvoie 404). Cf. HANDOFF_HALO5_EXPERIMENTAL §0-quater.
func (c *Client) GetMatchEvents(ctx context.Context, auth Auth, matchID string) (*H5MatchEventsResponse, error) {
	if strings.TrimSpace(matchID) == "" {
		return nil, errors.New("GetMatchEvents: matchID vide")
	}
	endpoint := fmt.Sprintf("%s/h5/matches/%s/events", c.endpoints.hoteStats(), url.PathEscape(matchID))
	body, err := c.getJSON(ctx, auth, endpoint)
	if err != nil {
		return nil, fmt.Errorf("GetMatchEvents(%s): %w", matchID, err)
	}
	return decoder[H5MatchEventsResponse]("GetMatchEvents", body)
}
