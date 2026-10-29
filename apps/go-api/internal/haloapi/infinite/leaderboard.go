package infinite

// leaderboard.go — classement CSR mondial lu dans les pages publiques du site ([CleWaypointWeb],
// `/halo-infinite/leaderboards/{saison}/{playlist}?page=N`) : GET des pages, extraction du JSON Next.js
// `__NEXT_DATA__` par expression régulière, DTO de page, menu saisons/playlists, découverte de la
// saison active par redirection. Les entrées de domaine (rang recalculé, palier), les noms par langue,
// la graine de découverte, la pagination polie et le compteur de page vide sont à l'appelant. Aucun
// jeton n'est envoyé (site public).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path"
	"regexp"
	"strings"

	"levelup/go-api/internal/haloapi"
)

const (
	// leaderboardsBasePath : racine des pages ; sans saison ni playlist, le site redirige vers la
	// saison ACTIVE.
	leaderboardsBasePath = "/halo-infinite/leaderboards"
	// seasonIDPrefix : préfixe des identifiants de saison CSR (« csrseason13-3 »).
	seasonIDPrefix = "csrseason"
	// pageMaxOctets : garde-fou de lecture d'une page.
	pageMaxOctets = 12 << 20
	// ScraperUserAgent : l'agent annoncé au site (navigateur de bureau).
	ScraperUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0 Safari/537.36"
	acceptHTML = "text/html,application/xhtml+xml"
)

// ErrLeaderboardPageNotFound : la page d'un couple (saison, playlist) rend 404 — la playlist n'était
// pas classée cette saison-là. Cas nominal d'un parcours multi-saisons, à sauter (errors.Is).
var ErrLeaderboardPageNotFound = errors.New("classement absent pour cette (saison, playlist)")

// Classement : le lecteur des pages de classement. Il ne porte aucune reprise ni limiteur : le
// rythme entre pages est celui de l'appelant.
type Classement struct {
	pages           haloapi.Doer
	sansRedirection haloapi.Doer
	endpoints       Endpoints
}

// NewClassement construit le lecteur : `pages` sert les pages (redirections suivies),
// `sansRedirection` la sonde de saison active (il ne doit PAS suivre les redirections : l'en-tête
// Location est la réponse cherchée). Adresses par défaut ([DefaultEndpoints]).
func NewClassement(pages, sansRedirection haloapi.Doer) *Classement {
	return &Classement{pages: pages, sansRedirection: sansRedirection, endpoints: DefaultEndpoints()}
}

// WithEndpoints rend une copie du lecteur sur les adresses `e`.
func (s *Classement) WithEndpoints(e Endpoints) *Classement {
	clone := *s
	clone.endpoints = e
	return &clone
}

// requetePage construit un GET de page HTML ; `langue` = valeur d'Accept-Language de l'appelant.
func requetePage(ctx context.Context, u, langue string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ScraperUserAgent)
	req.Header.Set(enteteAccept, acceptHTML)
	if langue != "" {
		req.Header.Set(enteteLangue, langue)
	}
	return req, nil
}

// FetchPageBytes récupère le HTML brut d'une page du classement. 404 : [ErrLeaderboardPageNotFound].
func (s *Classement) FetchPageBytes(ctx context.Context, seasonID, playlistID string, page int, langue string) ([]byte, error) {
	u := fmt.Sprintf("%s%s/%s/%s?page=%d", s.endpoints.Hote(CleWaypointWeb), leaderboardsBasePath, seasonID, playlistID, page)
	req, err := requetePage(ctx, u, langue)
	if err != nil {
		return nil, err
	}
	resp, err := s.pages.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }() // la fermeture ne change pas l'issue
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("statut HTTP 404: %w", ErrLeaderboardPageNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, pageMaxOctets))
}

// FetchPage récupère et décode une page du classement.
func (s *Classement) FetchPage(ctx context.Context, seasonID, playlistID string, page int, langue string) (*LeaderboardPageDTO, error) {
	body, err := s.FetchPageBytes(ctx, seasonID, playlistID, page, langue)
	if err != nil {
		return nil, err
	}
	parsed, err := ParseLeaderboardPage(body)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return parsed, nil
}

// WaypointRef : une entrée du menu saison/playlist de la page.
type WaypointRef struct {
	ID          string
	DisplayName string
	// Translations : nom par étiquette de langue, tel que la page l'expose (saisons seulement).
	Translations map[string]string
}

// FetchCatalog lit les menus saisons et playlists d'une page « graine » (`seedSeasonID`,
// `refPlaylistID` : un couple qui rend une page). Les identifiants vides sont omis.
func (s *Classement) FetchCatalog(ctx context.Context, seedSeasonID, refPlaylistID, langue string) (seasons, playlists []WaypointRef, err error) {
	if strings.TrimSpace(refPlaylistID) == "" {
		return nil, nil, fmt.Errorf("FetchCatalog: refPlaylistID requis")
	}
	parsed, err := s.FetchPage(ctx, seedSeasonID, refPlaylistID, 1, langue)
	if err != nil {
		return nil, nil, fmt.Errorf("FetchCatalog: %w", err)
	}
	for _, se := range parsed.Seasons {
		if id := strings.TrimSpace(se.SeasonID); id != "" {
			seasons = append(seasons, WaypointRef{ID: id, DisplayName: se.DisplayName, Translations: se.Translations})
		}
	}
	for _, pl := range parsed.Playlists {
		if id := strings.TrimSpace(pl.PlaylistID); id != "" {
			playlists = append(playlists, WaypointRef{ID: id, DisplayName: pl.DisplayName})
		}
	}
	return seasons, playlists, nil
}

// FetchActiveSeasonByRedirect découvre la saison CSR active SANS page graine : la racine des
// classements répond par une redirection dont l'en-tête Location porte la saison active. Aucun HTML
// n'est lu.
func (s *Classement) FetchActiveSeasonByRedirect(ctx context.Context, langue string) (string, error) {
	req, err := requetePage(ctx, s.endpoints.Hote(CleWaypointWeb)+leaderboardsBasePath, langue)
	if err != nil {
		return "", fmt.Errorf("FetchActiveSeasonByRedirect: requête: %w", err)
	}
	resp, err := s.sansRedirection.Do(req)
	if err != nil {
		return "", fmt.Errorf("FetchActiveSeasonByRedirect: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // la fermeture ne change pas l'issue
	if resp.StatusCode < http.StatusMultipleChoices || resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("FetchActiveSeasonByRedirect: statut HTTP %d, redirection attendue vers la saison active", resp.StatusCode)
	}
	loc, err := resp.Location()
	if err != nil {
		return "", fmt.Errorf("FetchActiveSeasonByRedirect: en-tête Location exploitable absent (statut %d): %w", resp.StatusCode, err)
	}
	season := path.Base(strings.TrimRight(strings.TrimSpace(loc.Path), "/"))
	if !strings.HasPrefix(season, seasonIDPrefix) {
		return "", fmt.Errorf("FetchActiveSeasonByRedirect: Location %q ne désigne pas une saison (préfixe %q attendu)", loc.Path, seasonIDPrefix)
	}
	slog.InfoContext(ctx, "saison active découverte par redirection (sans page-graine)", "module", "leaderboard",
		"season", season, "status", resp.StatusCode)
	return season, nil
}

// nextDataRe extrait le contenu du script Next.js `__NEXT_DATA__`.
var nextDataRe = regexp.MustCompile(`(?s)<script[^>]*id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

// LeaderboardPageDTO : la portion utile de `__NEXT_DATA__.props.pageProps`.
type LeaderboardPageDTO struct {
	Page        int                   `json:"page"`
	PageSize    int                   `json:"pageSize"`
	Leaderboard []LeaderboardEntryDTO `json:"leaderboard"`
	Seasons     []WaypointSeasonDTO   `json:"seasons"`
	Playlists   []WaypointPlaylistDTO `json:"playlists"`
}

// LeaderboardEntryDTO : une entrée du classement ; `rank` vaut toujours 0 dans la page.
type LeaderboardEntryDTO struct {
	Player struct {
		XUID     string `json:"xuid"`
		Gamertag string `json:"gamertag"`
	} `json:"player"`
	Rank  int `json:"rank"`
	Score int `json:"score"`
}

// WaypointSeasonDTO : une saison du menu.
type WaypointSeasonDTO struct {
	SeasonID     string            `json:"seasonId"`
	DisplayName  string            `json:"displayName"`
	Translations map[string]string `json:"translations"`
}

// WaypointPlaylistDTO : une playlist du menu.
type WaypointPlaylistDTO struct {
	PlaylistID  string `json:"playlistId"`
	DisplayName string `json:"displayName"`
}

// ParseLeaderboardPage extrait `pageProps` du HTML d'une page. Fonction pure.
func ParseLeaderboardPage(body []byte) (*LeaderboardPageDTO, error) {
	m := nextDataRe.FindSubmatch(body)
	if m == nil {
		return nil, fmt.Errorf("bloc __NEXT_DATA__ introuvable")
	}
	var env struct {
		Props struct {
			PageProps LeaderboardPageDTO `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &env); err != nil {
		return nil, fmt.Errorf("décodage __NEXT_DATA__: %w", err)
	}
	return &env.Props.PageProps, nil
}
