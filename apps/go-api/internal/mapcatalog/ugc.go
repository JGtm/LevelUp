// Package mapcatalog — LE CATALOGUE DES CARTES : récupérer une variante `.mvar`, en tirer une
// entrée de catalogue, et n'écrire ce catalogue que de façon sûre. Les CLI de cuisson et le
// rattrapage au fetch de films (RUNTIME) en partagent le code : une seule copie.
//
// ugc.go : la façade LevelUp du client UGC des cartes. Le protocole (URL, en-têtes, lecture
// bornée) est dans `libs/haloapi/infinite` (`FetchCarteUGC`, `TelechargerFichierUGC`) ; ici
// restent l'identité présentée, le client HTTP et son délai, la garde `netguard`, la validation du
// DTO, les journaux et la forme des erreurs. Deux étapes : le Discovery UGC (AUTHENTIFIÉ, jeton
// Spartan, ADR 0023) résout assetId → Files.Prefix ; le stockage blob télécharge chaque *.mvar.
package mapcatalog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/netguard"

	"github.com/JGtm/LevelUp/libs/haloapi/infinite"
)

const (
	requestTimeout  = 30 * time.Second
	politeUserAgent = "LevelUp/1.0 (dashboard stats Halo, usage personnel)"
	// originWaypoint : l'`Origin` présentée au Discovery (le seul hôte Halo écrit dans ce fichier).
	originWaypoint = "https://www.halowaypoint.com"
	// surfaceUGC : le point de sortie dans le journal de `netguard` (coupe-circuit démo).
	surfaceUGC = "mapcatalog_ugc.get"
)

// Client est le client UGC des cartes : la brique, les jetons passés à chaque appel et l'identité.
type Client struct {
	api  *infinite.Client
	auth infinite.Auth
	id   infinite.Identite
}

// NewClient construit un client aux jetons `tokens` (nil admis : le Discovery est alors refusé
// avant toute requête, le stockage blob se lit sans jeton).
func NewClient(tokens *domain.HaloTokens) *Client {
	c := &Client{
		api: nouvelleAPI(&http.Client{Timeout: requestTimeout}),
		id:  infinite.Identite{UserAgent: politeUserAgent, Origin: originWaypoint},
	}
	if tokens != nil {
		c.auth = infinite.Auth{Spartan: tokens.SpartanToken, Clearance: tokens.ClearanceToken}
	}
	return c
}

// nouvelleAPI rend le client de la brique dont toute sortie passe par `next`, gardée par netguard.
func nouvelleAPI(next *http.Client) *infinite.Client {
	return infinite.New(netguard.Doer{Surface: surfaceUGC, Next: next}, infinite.RetryAucune())
}

// WithHTTPClient remplace le *http.Client des requêtes (tests via RoundTripper). nil ignore.
// Chainable.
func (c *Client) WithHTTPClient(h *http.Client) *Client {
	if h != nil {
		c.api = nouvelleAPI(h)
	}
	return c
}

// FetchAsset résout un assetId de carte via Discovery UGC.
// versionID vide → dernière version publiée.
func (c *Client) FetchAsset(ctx context.Context, assetID, versionID string) (*infinite.CarteUGC, error) {
	asset, err := c.api.FetchCarteUGC(ctx, c.auth, assetID, versionID, c.id)
	if err != nil {
		return nil, formerErreur(err)
	}
	if asset.Files.Prefix == "" || len(asset.Files.FileRelativePaths) == 0 {
		return nil, fmt.Errorf("asset %s: aucun fichier exposé (Files vide)", assetID)
	}
	slog.InfoContext(ctx, "mapobj: asset résolu",
		"asset_id", assetID, "version_id", asset.VersionID, "name", asset.PublicName,
		"files", len(asset.Files.FileRelativePaths),
		"num_objects_custom_data", asset.CustomData.NumOfObjectsOnMap)
	return asset, nil
}

// MvarPaths retourne les chemins relatifs *.mvar de l'asset.
func MvarPaths(a *infinite.CarteUGC) []string {
	var out []string
	for _, p := range a.Files.FileRelativePaths {
		if strings.HasSuffix(strings.ToLower(p), ".mvar") {
			out = append(out, p)
		}
	}
	return out
}

// FetchMvar télécharge un fichier de variante depuis le stockage blob.
func (c *Client) FetchMvar(ctx context.Context, asset *infinite.CarteUGC, relPath string) ([]byte, error) {
	body, err := c.FetchFileAt(ctx, asset.Files.Prefix, relPath)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "mapobj: mvar téléchargé",
		"asset_id", asset.AssetID, "file", relPath, "bytes", len(body))
	return body, nil
}

// FetchFileAt télécharge un fichier du STOCKAGE BLOB à partir d'un préfixe déjà connu.
//
// Le préfixe blob d'une carte n'a pas à être re-résolu quand on l'a déjà (l'inventaire UGC versionné
// de `cmd/mapcallouts-build/donnees/` le porte) ; le stockage blob répond SANS jeton, un `Client` à
// jetons nil suffit. Soumis à `netguard` comme toute émission sortante d'`internal/`.
func (c *Client) FetchFileAt(ctx context.Context, prefix, relPath string) ([]byte, error) {
	if prefix == "" || relPath == "" {
		return nil, fmt.Errorf("préfixe blob ou chemin relatif vide (%q, %q)", prefix, relPath)
	}
	body, err := c.api.TelechargerFichierUGC(ctx, prefix, relPath, c.id)
	if err != nil {
		return nil, formerErreur(err)
	}
	return body, nil
}

// formerErreur redonne aux erreurs de la brique la forme que lisent les journaux des appelants :
// `GET <url>: HTTP <statut> (<début du corps>)` (non-200), `GET <url>: <cause>` (échec d'émission) ;
// les autres (lecture, garde, jeton absent) passent telles quelles.
func formerErreur(err error) error {
	var statut *infinite.HTTPError
	var transport *infinite.ErreurTransport
	switch {
	case errors.As(err, &statut):
		return fmt.Errorf("GET %s: HTTP %d (%s)", statut.URL, statut.StatusCode, strings.TrimSpace(string(statut.Body)))
	case errors.As(err, &transport):
		return fmt.Errorf("GET %s: %w", transport.URL, transport.Err)
	}
	return err
}
