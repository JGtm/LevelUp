// cmd/variant-probe — fetch.go : client UGC pour les assets de mode.
//
// Deux étapes, exactement comme cmd/mapobj-build/fetch.go pour les cartes :
//  1. Discovery UGC (AUTHENTIFIÉ, jeton Spartan) : assetId → document d'asset
//     (VersionId, Files.Prefix, FileRelativePaths, et tout le reste — on garde
//     le JSON BRUT, la sonde cherche des champs qu'on ne connaît pas encore) ;
//  2. Stockage blob (ANONYME) : télécharge les fichiers depuis Files.Prefix.
//
// Segments : `ugcGameVariants` (internal/platform/halo/discovery_types.go) et
// `engineGameVariants`, découvert par l'EngineGameVariantLink que sert le premier.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	discoveryHost   = "https://discovery-infiniteugc.svc.halowaypoint.com"
	variantSegment  = "ugcGameVariants"
	engineSegment   = "engineGameVariants"
	requestTimeout  = 30 * time.Second
	maxPayloadBytes = 64 << 20
	politeUserAgent = "LevelUp/1.0 (dashboard stats Halo, usage personnel)"
)

type ugcClient struct {
	http   *http.Client
	tokens *domain.HaloTokens
}

func newUGCClient(tokens *domain.HaloTokens) *ugcClient {
	return &ugcClient{
		http:   &http.Client{Timeout: requestTimeout},
		tokens: tokens,
	}
}

// fetchAssetRaw récupère le document d'un asset de mode, BRUT.
// versionID vide → endpoint sans /versions/ (dernière version publiée) : c'est le
// cas de 1544 lignes du registre sur 1819, qui n'ont pas de game_variant_version_id.
func (c *ugcClient) fetchAssetRaw(ctx context.Context, segment, assetID, versionID string) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/hi/%s/%s", discoveryHost, segment, url.PathEscape(assetID))
	if versionID != "" {
		endpoint += "/versions/" + url.PathEscape(versionID)
	}
	return c.get(ctx, endpoint, true)
}

// fetchBlob télécharge un fichier référencé par l'asset depuis le stockage blob
// (lecture anonyme : les en-têtes Spartan y seraient inattendus).
func (c *ugcClient) fetchBlob(ctx context.Context, prefix, relPath string) ([]byte, error) {
	endpoint := strings.TrimSuffix(prefix, "/") + "/" + relPath
	body, err := c.get(ctx, endpoint, false)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "variant-probe: blob téléchargé", "file", relPath, "bytes", len(body))
	return body, nil
}

// get exécute un GET. withAuth pose les en-têtes Spartan/Clearance.
func (c *ugcClient) get(ctx context.Context, endpoint string, withAuth bool) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", politeUserAgent)
	req.Header.Set("Accept", "application/json")
	if withAuth {
		if c.tokens == nil || c.tokens.SpartanToken == "" {
			return nil, fmt.Errorf("aucun jeton Spartan disponible pour %s", endpoint)
		}
		req.Header.Set("x-343-authorization-spartan", c.tokens.SpartanToken)
		if c.tokens.ClearanceToken != "" {
			req.Header.Set("343-clearance", c.tokens.ClearanceToken)
		}
		req.Header.Set("Origin", "https://www.halowaypoint.com")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.WarnContext(ctx, "variant-probe: fermeture du corps de réponse", "err", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GET %s: HTTP %d (%s)", endpoint, resp.StatusCode,
			strings.TrimSpace(string(snippet)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPayloadBytes))
	if err != nil {
		return nil, fmt.Errorf("lecture %s: %w", endpoint, err)
	}
	return body, nil
}
