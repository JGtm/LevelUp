// Package halo — discovery_client.go : client API Discovery UGC (assets multilingues).
//
// Sprint 54 : peuplement asset_translations depuis Discovery UGC.
// Hôte réel : https://discovery-infiniteugc.svc.halowaypoint.com/hi/{segment}/{id}/versions/{ver}
// — PAS gamecms-hacs (cet hôte-là sert médailles/défis/BP, cf. internal/assets/
// fetcher_gamecms.go). Les deux exigent un token Spartan (mesure 2026-07-25, cf.
// FetchAsset).
package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

const (
	discoveryUGCHost = "https://discovery-infiniteugc.svc.halowaypoint.com"
)

// FetchAsset récupère un asset UGC (playlist/map/pair/gameVariant) depuis
// discovery-infiniteugc avec Accept-Language, et retourne son PublicName localisé.
//
// Pattern URL (SPNKr/Grunt + sync.GetPlaylistConfig, validé contre l'API 2026-06-12) :
//
//	GET https://discovery-infiniteugc.svc.halowaypoint.com/hi/{segment}/{assetId}/versions/{versionId}
//
// AUTH REQUISE : Spartan token + 343-clearance, réellement posés par
// attemptHaloGet (en-têtes x-343-authorization-spartan / 343-clearance) dès que
// p.staticTokens est non-nil — c'est toujours le cas en production, le token vient
// du pool unifié (cf. NewAssetNameFetcher). Sonde ANONYME du 2026-07-25 : GET
// /hi/Maps/{assetId} sans en-tête → HTTP 401 (corps vide, Server: Kestrel) ;
// gamecms-hacs répond 401 + en-tête `WWW-Authenticate: SpartanToken`. Aucun de ces
// hôtes n'est public. version_id est REQUIS (l'API 404 sans, et "latest" → 400).
// titleID n'est PAS un segment ici (le préfixe jeu "hi" est fixe, comme le client
// prouvé) — conservé pour la signature mais ignoré.
//
// Voie SANS jeton pour les assets UGC (éprouvée 2026-07-25 : 120/123 cartes
// récupérées), non câblée ici mais à connaître avant de conclure « impossible sans
// auth » : la page publique www.halowaypoint.com/halo-infinite/ugc/maps/{assetId}
// sert le JSON d'asset COMPLET (Files.Prefix, FileRelativePaths, VersionId) dans
// son <script id="__NEXT_DATA__">, et blobs-infiniteugc.svc.halowaypoint.com est en
// lecture anonyme (un chemin invalide → 404 Azure BlobNotFound, pas 401).
func (p *HaloProvider) FetchAsset(
	ctx context.Context,
	assetType AssetType,
	titleID string,
	assetID string,
	versionID string,
	lang string,
) (*DiscoveryAsset, error) {
	segment, ok := AssetTypeToEndpoint[assetType]
	if !ok {
		return nil, fmt.Errorf("asset type invalide : %s", assetType)
	}
	if versionID == "" {
		return nil, fmt.Errorf("fetch asset %s/%s: version_id requis (discovery-infiniteugc 404 sans)", assetType, assetID)
	}

	rawURL := fmt.Sprintf("%s/%s/%s/%s/versions/%s",
		p.hostFor(ctx, games.EndpointDiscoveryUGC, "", discoveryUGCHost), p.gamePrefix(ctx), segment, neturl.PathEscape(assetID), neturl.PathEscape(versionID))

	body, err := p.doGetWithLang(ctx, rawURL, lang)
	if err != nil {
		slog.Debug("FetchAsset: request failed", "asset_type", assetType, "asset_id", assetID, "lang", lang, "err", err)
		return nil, fmt.Errorf("fetch asset %s/%s [%s]: %w", assetType, assetID, lang, err)
	}

	// PublicName/Description peuvent être une string brute OU un objet {value:"..."}
	// (forme localisée) — cf. sync.decodeLocalizedName, validé contre l'API.
	// Files (Prefix + FileRelativePaths) porte les blobs CDN de l'asset (images
	// incluses) — cf. structure SPNKr/Grunt.
	var raw struct {
		AssetID     string          `json:"AssetId"`
		VersionID   string          `json:"VersionId"`
		PublicName  json.RawMessage `json:"PublicName"`
		Description json.RawMessage `json:"Description"`
		Files       struct {
			Prefix            string   `json:"Prefix"`
			FileRelativePaths []string `json:"FileRelativePaths"`
		} `json:"Files"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode asset %s/%s: %w", assetType, assetID, err)
	}

	return &DiscoveryAsset{
		AssetID:     raw.AssetID,
		VersionID:   raw.VersionID,
		PublicName:  decodeLocalizedAssetName(raw.PublicName),
		Description: decodeLocalizedAssetName(raw.Description),
		ImageURL:    buildAssetImageURL(raw.Files.Prefix, raw.Files.FileRelativePaths),
	}, nil
}

// decodeLocalizedAssetName tolère un PublicName/Description string brut OU un objet
// {value:"..."} (forme localisée de discovery-infiniteugc). Miroir local de
// sync.decodeLocalizedName (évite une dépendance halo → sync).
func decodeLocalizedAssetName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Value string `json:"value"`
	}
	if json.Unmarshal(raw, &obj) == nil {
		return obj.Value
	}
	return ""
}

// buildAssetImageURL construit l'URL CDN d'une image d'asset à partir du bloc Files
// (Prefix + FileRelativePaths) : préfère un chemin "thumbnail", sinon
// "hero"/"screenshot", sinon la 1re image (.jpg/.png/.webp). Retourne "" si rien
// d'exploitable. Pur — les chemins viennent de la VRAIE liste de l'asset (pas un
// chemin deviné en dur).
func buildAssetImageURL(prefix string, relPaths []string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || len(relPaths) == 0 {
		return ""
	}
	pick := func(pred func(string) bool) string {
		for _, p := range relPaths {
			if pred(strings.ToLower(strings.TrimSpace(p))) {
				return strings.TrimSpace(p)
			}
		}
		return ""
	}
	isImage := func(p string) bool {
		return strings.HasSuffix(p, ".jpg") || strings.HasSuffix(p, ".jpeg") ||
			strings.HasSuffix(p, ".png") || strings.HasSuffix(p, ".webp")
	}
	img := pick(func(p string) bool { return strings.Contains(p, "thumbnail") && isImage(p) })
	if img == "" {
		img = pick(func(p string) bool {
			return (strings.Contains(p, "hero") || strings.Contains(p, "screenshot")) && isImage(p)
		})
	}
	if img == "" {
		img = pick(isImage)
	}
	if img == "" {
		return ""
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + img
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
		"https://halostats.svc.halowaypoint.com/%s/players/xuid(%s)/matches/%s/stats",
		p.gamePrefix(ctx), arbitraryXUID, matchID,
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
// Ajoute le Spartan token si p.staticTokens est configuré (APIs gamecms-hacs requérant auth).
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

		body, retriable, err := p.attemptHaloGet(ctx, rawURL, lang)
		if err == nil {
			return body, nil
		}
		if !retriable {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

// attemptHaloGet effectue une tentative GET unique. Retourne (body, retriable, err).
// retriable=true → l'appelant doit retry avec backoff ; retriable=false → erreur finale.
func (p *HaloProvider) attemptHaloGet(ctx context.Context, rawURL, lang string) ([]byte, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("new request: %w", err)
	}

	req.Header.Set("Accept-Language", lang)
	req.Header.Set("Accept", "application/json")
	if p.staticTokens != nil {
		req.Header.Set("x-343-authorization-spartan", p.staticTokens.SpartanToken)
		if p.staticTokens.ClearanceToken != "" {
			req.Header.Set("343-clearance", p.staticTokens.ClearanceToken)
		}
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("http do: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return nil, true, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		return body, false, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, false, fmt.Errorf("asset not found (404)")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, false, fmt.Errorf("http %d: auth requise — %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("server error %d", resp.StatusCode)
	}
	return nil, false, fmt.Errorf("http %d: %s", resp.StatusCode, string(body))
}
