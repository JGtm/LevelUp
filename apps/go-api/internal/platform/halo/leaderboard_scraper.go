// Package halo — leaderboard_scraper.go : récupération du classement CSR mondial
// depuis les pages publiques Halo Waypoint.
//
//	https://www.halowaypoint.com/halo-infinite/leaderboards/{seasonId}/{playlistId}?page=N
//
// Ces pages sont rendues côté serveur, publiques (aucune authentification) et
// paginées. L'API du client de jeu (skill.svc) n'expose PAS de classement
// mondial : le site web est la seule source. Le job
// cmd/snapshot-world-leaderboard appelle ce scraper puis persiste en append-only
// (cf. internal/migration/steps_world_csr_leaderboard.go).
//
// Parsing robuste : les données sont extraites du bloc JSON Next.js __NEXT_DATA__
// embarqué dans la page (`props.pageProps.leaderboard`), indépendant du CSS.
// Chaque entrée fournit player.xuid, player.gamertag et score (le CSR). Le champ
// `rank` du payload vaut 0 : le rang réel est recalculé via (page-1)*pageSize+i+1.
// Couvert par un test sur fixture (testdata/leaderboard_sample.html).
// NE PAS dépendre du proxy tiers sr-nextjs.vercel.app.
package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

const (
	waypointLeaderboardHost = "https://www.halowaypoint.com"
	defaultLeaderboardPage  = 100 // pageSize observé (fallback si absent du payload)
	// Garde-fou : jamais plus de pages que ça (évite une boucle infinie si la
	// détection de fin échoue). 25 pages × 100 = 2500 joueurs max par playlist.
	leaderboardMaxPages = 25
	scraperUserAgent    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0 Safari/537.36"
)

// LeaderboardScraper récupère le classement CSR mondial depuis Halo Waypoint.
type LeaderboardScraper struct {
	client  *http.Client
	host    string        // overridable pour les tests
	perPage time.Duration // délai poli entre deux pages
}

// NewLeaderboardScraper crée un scraper avec un délai poli entre pages.
func NewLeaderboardScraper(politeDelay time.Duration) *LeaderboardScraper {
	return &LeaderboardScraper{
		client:  &http.Client{Timeout: 20 * time.Second},
		host:    waypointLeaderboardHost,
		perPage: politeDelay,
	}
}

// FetchCSRLeaderboard récupère le classement CSR d'une playlist pour une saison.
// Pagine jusqu'à `limit` entrées (0 = jusqu'à épuisement, borné par leaderboardMaxPages).
// fetchedAt est renseigné sur chaque entrée ; le tier est dérivé du CSR.
func (s *LeaderboardScraper) FetchCSRLeaderboard(
	ctx context.Context, seasonID, playlistID string, limit int,
) ([]domain.LeaderboardEntry, error) {
	if strings.TrimSpace(seasonID) == "" || strings.TrimSpace(playlistID) == "" {
		return nil, fmt.Errorf("FetchCSRLeaderboard: season et playlist requis")
	}
	fetchedAt := time.Now().UTC()
	out := make([]domain.LeaderboardEntry, 0, 200)

	for page := 1; page <= leaderboardMaxPages; page++ {
		body, err := s.fetchPageBytes(ctx, seasonID, playlistID, page)
		if err != nil {
			return nil, fmt.Errorf("FetchCSRLeaderboard page %d: %w", page, err)
		}
		parsed, err := parseLeaderboardPage(body)
		if err != nil {
			return nil, fmt.Errorf("FetchCSRLeaderboard page %d: parse: %w", page, err)
		}
		if len(parsed.Leaderboard) == 0 {
			break // fin de l'échelle
		}
		pageSize := parsed.PageSize
		if pageSize <= 0 {
			pageSize = defaultLeaderboardPage
		}
		for i, e := range parsed.Leaderboard {
			gt := strings.TrimSpace(e.Player.Gamertag)
			if gt == "" {
				continue
			}
			tier, subTier := domain.DeriveCSRTier(e.Score)
			out = append(out, domain.LeaderboardEntry{
				Season:    seasonID,
				Playlist:  playlistID,
				Rank:      (page-1)*pageSize + i + 1, // rank du payload = 0, recalculé
				XUID:      strings.TrimSpace(e.Player.XUID),
				Gamertag:  gt,
				CSR:       e.Score,
				CSRValue:  e.Score,
				Tier:      tier,
				SubTier:   subTier,
				Category:  string(domain.LeaderboardCSRWorld),
				Value:     float64(e.Score),
				FetchedAt: fetchedAt,
			})
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
		if len(parsed.Leaderboard) < pageSize {
			break // dernière page (incomplète)
		}
		if s.perPage > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(s.perPage):
			}
		}
	}
	return out, nil
}

// fetchPageBytes récupère le HTML brut d'une page du classement.
func (s *LeaderboardScraper) fetchPageBytes(ctx context.Context, seasonID, playlistID string, page int) ([]byte, error) {
	url := fmt.Sprintf("%s/halo-infinite/leaderboards/%s/%s?page=%d",
		s.host, seasonID, playlistID, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", scraperUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 12<<20)) // 12 MiB garde-fou
}

// ─── Parsing du bloc __NEXT_DATA__ (point d'ajustement unique) ────────────────

// nextDataRe extrait le contenu du script Next.js __NEXT_DATA__.
var nextDataRe = regexp.MustCompile(
	`(?s)<script[^>]*id="__NEXT_DATA__"[^>]*>(.*?)</script>`)

// waypointPageProps est la portion utile du JSON __NEXT_DATA__.props.pageProps.
type waypointPageProps struct {
	Page        int                   `json:"page"`
	PageSize    int                   `json:"pageSize"`
	Leaderboard []waypointLBEntry     `json:"leaderboard"`
	Seasons     []waypointSeasonRef   `json:"seasons"`
	Playlists   []waypointPlaylistRef `json:"playlists"`
}

type waypointLBEntry struct {
	Player struct {
		XUID     string `json:"xuid"`
		Gamertag string `json:"gamertag"`
	} `json:"player"`
	Rank  int `json:"rank"` // toujours 0 dans le payload — ignoré
	Score int `json:"score"`
}

// waypointSeasonRef / waypointPlaylistRef : listes exposées par la page (utiles
// pour peupler les sélecteurs côté job/UI).
type waypointSeasonRef struct {
	SeasonID    string `json:"seasonId"`
	DisplayName string `json:"displayName"`
}

type waypointPlaylistRef struct {
	PlaylistID  string `json:"playlistId"`
	DisplayName string `json:"displayName"`
}

// parseLeaderboardPage extrait pageProps depuis le HTML d'une page.
func parseLeaderboardPage(body []byte) (waypointPageProps, error) {
	m := nextDataRe.FindSubmatch(body)
	if m == nil {
		return waypointPageProps{}, fmt.Errorf("bloc __NEXT_DATA__ introuvable")
	}
	var env struct {
		Props struct {
			PageProps waypointPageProps `json:"pageProps"`
		} `json:"props"`
	}
	if err := json.Unmarshal(m[1], &env); err != nil {
		return waypointPageProps{}, fmt.Errorf("décodage __NEXT_DATA__: %w", err)
	}
	return env.Props.PageProps, nil
}
