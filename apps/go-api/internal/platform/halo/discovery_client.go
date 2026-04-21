// Package halo — discovery_client.go : client API Discovery UGC (assets multilingues).
//
// Sprint 54 : peuplement asset_translations depuis Discovery UGC.
// API endpoint : https://gamecms-hacs.svc.halowaypoint.com/hi/multiplayer/file/...
package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	discoveryUGCHost = "https://gamecms-hacs.svc.halowaypoint.com"
)

// FetchAsset récupère un asset depuis l'API Discovery UGC avec Accept-Language header.
//
// Pattern URL : /hi/multiplayer/file/{asset_type}/{title_id}/{asset_id}/{version_id}
// Exemple : /hi/multiplayer/file/map-variants/halo_infinite/{uuid}/{version_id}
//
// Note : Cette API ne nécessite PAS d'authentification Spartan (publique).
func (p *HaloProvider) FetchAsset(
	ctx context.Context,
	assetType AssetType,
	titleID string,
	assetID string,
	versionID string,
	lang string,
) (*DiscoveryAsset, error) {
	endpoint, ok := AssetTypeToEndpoint[assetType]
	if !ok {
		return nil, fmt.Errorf("asset type invalide : %s", assetType)
	}

	url := fmt.Sprintf(
		"%s/hi/multiplayer/file/%s/%s/%s/%s",
		discoveryUGCHost, endpoint, titleID, assetID, versionID,
	)

	body, err := p.doGetWithLang(ctx, url, lang)
	if err != nil {
		slog.Debug("FetchAsset: request failed", "asset_type", assetType, "asset_id", assetID, "lang", lang, "err", err)
		return nil, fmt.Errorf("fetch asset %s/%s [%s]: %w", assetType, assetID, lang, err)
	}

	var asset DiscoveryAsset
	if err := json.Unmarshal(body, &asset); err != nil {
		return nil, fmt.Errorf("decode asset %s/%s: %w", assetType, assetID, err)
	}

	return &asset, nil
}

// FetchMatchStats récupère les statistiques complètes d'un match.
// Retourne le JSON brut pour extraction flexible des version_ids.
//
// Pattern URL : /hi/players/xuid({xuid})/matches/{match_id}/stats
// Note : Nécessite authentification Spartan (tokens requis).
func (p *HaloProvider) FetchMatchStats(
	ctx context.Context,
	matchID string,
	tokens *domain.HaloTokens,
) (map[string]interface{}, error) {
	// Sprint 54 : Pour populate-assets, on utilise un xuid arbitraire
	// car l'endpoint match stats retourne la structure MatchInfo complète
	// indépendamment du joueur interrogé (MatchInfo est global au match).
	const arbitraryXUID = "0"

	url := fmt.Sprintf(
		"https://halostats.svc.halowaypoint.com/hi/players/xuid(%s)/matches/%s/stats",
		arbitraryXUID, matchID,
	)

	body, err := p.doGet(ctx, url, tokens)
	if err != nil {
		slog.Debug("FetchMatchStats: request failed", "match_id", matchID, "err", err)
		return nil, fmt.Errorf("fetch match stats %s: %w", matchID, err)
	}

	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		return nil, fmt.Errorf("decode match stats %s: %w", matchID, err)
	}

	return stats, nil
}

// doGetWithLang effectue un GET avec Accept-Language header.
// Pas d'authentification (API publique Discovery UGC).
func (p *HaloProvider) doGetWithLang(ctx context.Context, rawURL string, lang string) ([]byte, error) {
	if err := p.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var lastErr error
	backoff := providerRetryBase

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				backoff *= 2
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("new request: %w", err)
		}

		req.Header.Set("Accept-Language", lang)
		req.Header.Set("Accept", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http do: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}

		// 404 : asset inexistant ou supprimé par 343 → erreur non retriable
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("asset not found (404)")
		}

		// 5xx : erreur serveur → retry
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error %d", resp.StatusCode)
			continue
		}

		// Autre erreur (4xx) : non retriable
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
