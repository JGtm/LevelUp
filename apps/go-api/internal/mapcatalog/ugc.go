// Package mapcatalog — LE CATALOGUE DES CARTES : récupérer une variante `.mvar`, en tirer une
// entrée de catalogue, et n'écrire ce catalogue que de façon sûre.
//
// POURQUOI CE PAQUET EXISTE. Ces trois gestes vivaient dans le `package mapcatalog` de DEUX CLI
// (`mapobj-build` pour le fetch, `mapopads-build` pour l'entrée), donc importables par
// personne. Le rattrapage au fetch de films en a besoin À L'EXÉCUTION : sans extraction, il
// aurait fallu une troisième copie — ce que la règle du dépôt interdit (≤ 2 copies, et un
// garde-rail à la 3ᵉ).
//
// ugc.go : client UGC pour récupérer la variante de carte (.mvar).
//
// Deux étapes :
//  1. Discovery UGC (AUTHENTIFIÉ, jeton Spartan) : résout assetId → métadonnées
//     (VersionId, Files.Prefix, FileRelativePaths).
//  2. Stockage blob : télécharge chaque *.mvar depuis Files.Prefix.
//
// TÉMOIN d'authentification (mesuré) : un GET anonyme sur discovery-infiniteugc
// répond HTTP 401. Le jeton vient de l'auth existante du projet (ADR 0023,
// auth.RefreshHaloTokensViaStoreFirst) — aucune re-capture de jeton.
package mapcatalog

import (
	"context"
	"encoding/json"
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
	requestTimeout  = 30 * time.Second
	maxMvarBytes    = 64 << 20 // garde-fou : un .mvar réaliste pèse ~100 Ko
	politeUserAgent = "LevelUp/1.0 (dashboard stats Halo, usage personnel)"
)

// Asset est la forme du DTO Discovery UGC réellement servie (champs utilisés
// uniquement — le reste du document est ignoré).
type Asset struct {
	AssetID    string `json:"AssetId"`
	VersionID  string `json:"VersionId"`
	PublicName string `json:"PublicName"`
	Files      struct {
		Prefix            string   `json:"Prefix"`
		FileRelativePaths []string `json:"FileRelativePaths"`
	} `json:"Files"`
	CustomData struct {
		NumOfObjectsOnMap int   `json:"NumOfObjectsOnMap"`
		TagLevelID        int64 `json:"TagLevelId"`
		IsBaked           bool  `json:"IsBaked"`
	} `json:"CustomData"`
}

type Client struct {
	http   *http.Client
	tokens *domain.HaloTokens
}

func NewClient(tokens *domain.HaloTokens) *Client {
	return &Client{
		http:   &http.Client{Timeout: requestTimeout},
		tokens: tokens,
	}
}

// FetchAsset résout un assetId de carte via Discovery UGC.
// versionID vide → dernière version publiée.
func (c *Client) FetchAsset(ctx context.Context, assetID, versionID string) (*Asset, error) {
	endpoint := fmt.Sprintf("%s/hi/Maps/%s", discoveryHost, url.PathEscape(assetID))
	if versionID != "" {
		endpoint += "/versions/" + url.PathEscape(versionID)
	}
	body, err := c.get(ctx, endpoint, true)
	if err != nil {
		return nil, err
	}
	var asset Asset
	if err := json.Unmarshal(body, &asset); err != nil {
		return nil, fmt.Errorf("décoder l'asset %s: %w", assetID, err)
	}
	if asset.Files.Prefix == "" || len(asset.Files.FileRelativePaths) == 0 {
		return nil, fmt.Errorf("asset %s: aucun fichier exposé (Files vide)", assetID)
	}
	slog.InfoContext(ctx, "mapobj: asset résolu",
		"asset_id", assetID, "version_id", asset.VersionID, "name", asset.PublicName,
		"files", len(asset.Files.FileRelativePaths),
		"num_objects_custom_data", asset.CustomData.NumOfObjectsOnMap)
	return &asset, nil
}

// MvarPaths retourne les chemins relatifs *.mvar de l'asset.
func (a *Asset) MvarPaths() []string {
	var out []string
	for _, p := range a.Files.FileRelativePaths {
		if strings.HasSuffix(strings.ToLower(p), ".mvar") {
			out = append(out, p)
		}
	}
	return out
}

// FetchMvar télécharge un fichier de variante depuis le stockage blob.
func (c *Client) FetchMvar(ctx context.Context, asset *Asset, relPath string) ([]byte, error) {
	endpoint := strings.TrimSuffix(asset.Files.Prefix, "/") + "/" + relPath
	body, err := c.get(ctx, endpoint, false)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "mapobj: mvar téléchargé",
		"asset_id", asset.AssetID, "file", relPath, "bytes", len(body))
	return body, nil
}

// get exécute un GET. withAuth pose les en-têtes Spartan/Clearance ; le stockage
// blob n'en a pas besoin (et les refuserait comme en-têtes inattendus).
func (c *Client) get(ctx context.Context, endpoint string, withAuth bool) ([]byte, error) {
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
			slog.WarnContext(ctx, "mapobj: fermeture du corps de réponse", "err", cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GET %s: HTTP %d (%s)", endpoint, resp.StatusCode,
			strings.TrimSpace(string(snippet)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxMvarBytes))
	if err != nil {
		return nil, fmt.Errorf("lecture %s: %w", endpoint, err)
	}
	return body, nil
}
