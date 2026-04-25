// Package presence — steam_poller.go : détection de présence via l'API Steam.
//
// Fallback pour les joueurs Halo Infinite via Steam, dont la présence Xbox RTA
// est inconsistante. Poll l'API Steam toutes les 60s.
//
// Endpoint : GET https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/
// Paramètres : ?key=<STEAM_API_KEY>&steamids=<steam_id>
// Réponse : players[0].gameid = "1336960" → Halo Infinite via Steam
package presence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	steamAPIURL          = "https://api.steampowered.com/ISteamUser/GetPlayerSummaries/v2/"
	defaultSteamInterval = 60 * time.Second
)

// SteamCallback est appelé quand le joueur est actif sur un titre.
type SteamActiveCallback func(gameID, gameName string)

// SteamInactiveCallback est appelé quand le joueur n'est plus en jeu.
type SteamInactiveCallback func()

// SteamPoller poll l'API Steam pour détecter l'activité d'un joueur.
type SteamPoller struct {
	steamID    string
	apiKey     string
	interval   time.Duration
	onActive   SteamActiveCallback
	onInactive SteamInactiveCallback
	client     *http.Client
	wasActive  bool // état précédent pour ne logger les transitions qu'une fois
}

// NewSteamPoller crée un poller Steam.
func NewSteamPoller(steamID, apiKey string, onActive SteamActiveCallback, onInactive SteamInactiveCallback) *SteamPoller {
	return &SteamPoller{
		steamID:    steamID,
		apiKey:     apiKey,
		interval:   defaultSteamInterval,
		onActive:   onActive,
		onInactive: onInactive,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Run démarre le polling. Bloquant — à lancer dans une goroutine.
func (p *SteamPoller) Run(ctx context.Context) {
	slog.InfoContext(ctx, "steam_poller: démarré",
		"steam_id", p.steamID,
		"interval", p.interval,
	)

	// Poll immédiat
	p.poll(ctx)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "steam_poller: arrêté", "steam_id", p.steamID)
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// steamPlayerSummary est la réponse partielle de l'API Steam.
type steamPlayerSummary struct {
	SteamID       string `json:"steamid"`
	GameID        string `json:"gameid"`        // présent uniquement si en jeu
	GameExtraInfo string `json:"gameextrainfo"` // nom du jeu
	PersonaState  int    `json:"personastate"`
	PersonaName   string `json:"personaname"`
}

type steamAPIResponse struct {
	Response struct {
		Players []steamPlayerSummary `json:"players"`
	} `json:"response"`
}

// poll effectue un appel à l'API Steam.
func (p *SteamPoller) poll(ctx context.Context) {
	url := fmt.Sprintf("%s?key=%s&steamids=%s", steamAPIURL, p.apiKey, p.steamID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.WarnContext(ctx, "steam_poller: erreur construction requête", "err", err)
		return
	}

	resp, err := p.client.Do(req)
	if err != nil {
		slog.WarnContext(ctx, "steam_poller: erreur appel API", "steam_id", p.steamID, "err", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "steam_poller: API HTTP erreur",
			"steam_id", p.steamID,
			"status", resp.StatusCode,
		)
		return
	}

	var result steamAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.WarnContext(ctx, "steam_poller: erreur décodage", "err", err)
		return
	}

	if len(result.Response.Players) == 0 {
		slog.WarnContext(ctx, "steam_poller: aucun joueur trouvé (profil privé ?)",
			"steam_id", p.steamID,
		)
		return
	}

	player := result.Response.Players[0]
	if player.GameID != "" {
		if !p.wasActive {
			slog.InfoContext(ctx, "steam_poller: joueur en jeu",
				"steam_id", p.steamID,
				"game_id", player.GameID,
				"game_name", player.GameExtraInfo,
			)
		}
		p.wasActive = true
		if p.onActive != nil {
			p.onActive(player.GameID, player.GameExtraInfo)
		}
	} else {
		if p.wasActive {
			slog.InfoContext(ctx, "steam_poller: joueur plus en jeu",
				"steam_id", p.steamID,
			)
		}
		p.wasActive = false
		if p.onInactive != nil {
			p.onInactive()
		}
	}
}
