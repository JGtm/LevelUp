package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// xboxProfileBaseURL — endpoint Xbox Live "profile settings". Contrairement à
// PeopleHub (social.xboxlive.com, borné au GRAPHE SOCIAL du compte token, max 25),
// cet endpoint résout N'IMPORTE QUEL gamertag public en xuid — y compris les
// adversaires de matchmaking aléatoires hors du graphe social. C'est le résolveur
// UNIVERSEL qui comble la perte de roster Halo 5 (fix #10 : ~84% des matchs 4v4
// avaient < 8 joueurs persistés faute de résolution PeopleHub).
//
// Auth : MÊME token que PeopleHub — un XSTS audience http://xboxlive.com
// (header "XBL3.0 x=<uhs>;<xsts>"). Le projet sait DÉJÀ le produire (cf. xsts.go
// AcquireXSTSForRTA + CachedHeaderProvider câblé dans worldenrich.BuildMultiResolver) ;
// ce résolveur RÉUTILISE le même headerFn, aucune nouvelle chaîne de token requise.
const xboxProfileBaseURL = "https://profile.xboxlive.com/users"

// XboxProfileResolver résout gamertag -> xuid via l'endpoint profil Xbox Live.
// Title-agnostic (xuid Xbox global) : le même résolveur sert tous les titres.
type XboxProfileResolver struct {
	client   *http.Client
	headerFn func(ctx context.Context) (string, error)
	baseURL  string // overridable pour les tests
}

// NewXboxProfileResolver construit le résolveur. headerFn fournit un header
// "XBL3.0 x=<uhs>;<xsts>" valide (XSTS, audience http://xboxlive.com) — le MÊME
// que NewPeopleHubResolver (réutilisable tel quel). Le caller mémoïse/rafraîchit
// le token (cf. CachedHeaderProvider). client nil -> http.DefaultClient.
func NewXboxProfileResolver(client *http.Client, headerFn func(ctx context.Context) (string, error)) *XboxProfileResolver {
	if client == nil {
		client = http.DefaultClient
	}
	return &XboxProfileResolver{client: client, headerFn: headerFn, baseURL: xboxProfileBaseURL}
}

// ResolveXUID retourne l'xuid numérique du gamertag (sans wrapper "xuid()"), ou
// une erreur si la requête échoue, le profil est introuvable, ou l'id est vide.
// Le gamertag est encodé dans le segment de chemin gt(<Gamertag>) — pas un query
// param (l'API exige la forme gt(...)).
func (r *XboxProfileResolver) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	gt := strings.TrimSpace(gamertag)
	if gt == "" {
		return "", fmt.Errorf("xbox profile: gamertag vide")
	}
	header, err := r.headerFn(ctx)
	if err != nil {
		return "", fmt.Errorf("header XSTS: %w", err)
	}
	// gt(<Gamertag>) dans le PATH (pas un query). url.PathEscape sur le gamertag
	// seul (les parenthèses gt(...) sont littérales attendues par l'API).
	endpoint := r.baseURL + "/gt(" + url.PathEscape(gt) + ")/profile/settings?settings=Gamertag"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-xbl-contract-version", "2")
	req.Header.Set("Authorization", header)
	req.Header.Set("Accept-Language", "en-us")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("xbox profile HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data struct {
		ProfileUsers []struct {
			ID string `json:"id"` // xuid numérique
		} `json:"profileUsers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data.ProfileUsers) == 0 || strings.TrimSpace(data.ProfileUsers[0].ID) == "" {
		return "", fmt.Errorf("xbox profile: aucun profil pour gamertag %q", gt)
	}
	return strings.TrimSpace(data.ProfileUsers[0].ID), nil
}
