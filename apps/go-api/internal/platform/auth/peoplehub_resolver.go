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

// peopleHubSearchURL — endpoint de recherche Xbox Live PeopleHub.
const peopleHubSearchURL = "https://peoplehub.xboxlive.com/users/me/people/search/decoration/detail,preferredColor"

// PeopleHubResolver résout gamertag -> xuid via PeopleHub (recherche, filtrée sur
// correspondance exacte case-insensitive). Utilisé par l'agrégateur du classement
// mondial pour résoudre les xuid des joueurs scrapés (single-token RTA, bas volume).
type PeopleHubResolver struct {
	client   *http.Client
	headerFn func(ctx context.Context) (string, error)
	baseURL  string // overridable pour les tests
}

// NewPeopleHubResolver construit un résolveur. headerFn fournit un header
// "XBL3.0 x=<hash>;<token>" valide (RTA XSTS, audience http://xboxlive.com) —
// invoqué à chaque résolution ; charge au fournisseur de mémoïser/rafraîchir le
// token avant expiration (~quelques heures).
func NewPeopleHubResolver(client *http.Client, headerFn func(ctx context.Context) (string, error)) *PeopleHubResolver {
	if client == nil {
		client = http.DefaultClient
	}
	return &PeopleHubResolver{client: client, headerFn: headerFn, baseURL: peopleHubSearchURL}
}

// ResolveXUID retourne l'xuid numérique du gamertag (sans wrapper "xuid()"), ou
// une erreur si la recherche échoue ou ne renvoie aucune correspondance exacte.
func (r *PeopleHubResolver) ResolveXUID(ctx context.Context, gamertag string) (string, error) {
	header, err := r.headerFn(ctx)
	if err != nil {
		return "", fmt.Errorf("header RTA: %w", err)
	}
	endpoint := r.baseURL + "?q=" + url.QueryEscape(gamertag) + "&maxItems=25"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("x-xbl-contract-version", "3")
	req.Header.Set("Authorization", header)
	req.Header.Set("Accept-Language", "en-us")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("peoplehub HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var data struct {
		People []struct {
			Gamertag string `json:"gamertag"`
			XUID     string `json:"xuid"`
		} `json:"people"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	for _, p := range data.People {
		if strings.EqualFold(strings.TrimSpace(p.Gamertag), strings.TrimSpace(gamertag)) {
			return p.XUID, nil
		}
	}
	return "", fmt.Errorf("gamertag %q absent des %d résultats peoplehub", gamertag, len(data.People))
}
