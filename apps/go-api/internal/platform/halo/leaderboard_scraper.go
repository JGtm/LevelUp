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
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// logModule est l'attribut de routage des logs du scraper vers logs/leaderboard.log
// (cf. internal/observability/logging.ModuleLeaderboard). Valeur littérale pour
// éviter de coupler le package halo au package logging.
const logModule = "leaderboard"

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
		slog.DebugContext(ctx, "leaderboard page parsée", "module", logModule,
			"season", seasonID, "playlist", playlistID, "page", page, "entries", len(parsed.Leaderboard))
		if len(parsed.Leaderboard) == 0 {
			if page == 1 {
				// Canari de changement de markup Halo Waypoint : une page 1 vide
				// signale soit une saison/playlist inexistante, soit un markup qui
				// a évolué (bloc __NEXT_DATA__ déplacé/renommé). Compteur expvar
				// (/debug/vars → levelup.leaderboard_empty_page1) pour le surveiller
				// sans grep de logs ; au-delà de ~0 en régime nominal, rafraîchir la
				// fixture testdata/leaderboard_sample.html et le parsing.
				observability.IncCounter("leaderboard_empty_page1")
				slog.WarnContext(ctx, "leaderboard vide en page 1 (saison/playlist inexistante ou markup changé ?)",
					"module", logModule, "season", seasonID, "playlist", playlistID)
			}
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

// WaypointRef est une entrée du menu déroulant saison/playlist exposé par la page
// Halo Waypoint (props.pageProps.{seasons,playlists}). Sert à peupler les
// sélecteurs côté UI et à découvrir la saison active côté cron.
type WaypointRef struct {
	ID          string
	DisplayName string
	// Translations : nom localisé par locale BCP-47 (ex "fr-FR" → "Ombres"),
	// tel qu'exposé par le payload Waypoint. Renseigné pour les SAISONS
	// uniquement (les playlists n'en exposent pas) ; nil sinon. DisplayName
	// reste le nom EN canonique (la page est requêtée en en-US).
	Translations map[string]string
}

// seedSeasonID est la saison « graine » utilisée pour bootstrapper la requête de
// découverte du catalogue (FetchCatalog/FetchActiveSeason). Sa seule fonction est
// de construire une URL leaderboard qui rend la page : la valeur retournée est
// TOUJOURS seasons[0] (la saison active du jour), qui se corrige d'elle-même même
// si cette graine est périmée. Les saisons passées restent accessibles
// indéfiniment sur Halo Waypoint (cf. fixture : csrseason3-1 → 13-2 toutes
// sélectionnables), donc l'URL graine ne 404 jamais une fois la saison créée.
const seedSeasonID = "csrseason13-2"

// FetchActiveSeason découvre la saison CSR active en lisant le menu déroulant de
// la page Halo Waypoint (seasons[0], la liste étant ordonnée du plus récent au
// plus ancien). Mode autonome : aucune saison codée en dur côté cron — quand Halo
// passe à la saison suivante, la découverte suit automatiquement.
//
// refPlaylistID : une playlist classée valide quelconque (n'importe laquelle rend
// le même menu de saisons). Retourne une erreur si la page n'expose aucune saison
// (markup changé ou playlist invalide).
func (s *LeaderboardScraper) FetchActiveSeason(ctx context.Context, refPlaylistID string) (string, error) {
	seasons, _, err := s.FetchCatalog(ctx, refPlaylistID)
	if err != nil {
		return "", err
	}
	if len(seasons) == 0 {
		return "", fmt.Errorf("FetchActiveSeason: aucune saison exposée par la page (markup changé ?)")
	}
	slog.DebugContext(ctx, "saison active découverte", "module", logModule,
		"season", seasons[0].ID, "display", seasons[0].DisplayName, "total_seasons", len(seasons))
	return seasons[0].ID, nil
}

// FetchCatalog récupère les listes saisons + playlists exposées par la page
// (props.pageProps.{seasons,playlists}), via une unique requête « graine ». Utilisé
// par le cron (saison active = seasons[0]) et par l'API des sélecteurs dynamiques.
func (s *LeaderboardScraper) FetchCatalog(
	ctx context.Context, refPlaylistID string,
) (seasons, playlists []WaypointRef, err error) {
	if strings.TrimSpace(refPlaylistID) == "" {
		return nil, nil, fmt.Errorf("FetchCatalog: refPlaylistID requis")
	}
	body, err := s.fetchPageBytes(ctx, seedSeasonID, refPlaylistID, 1)
	if err != nil {
		return nil, nil, fmt.Errorf("FetchCatalog: %w", err)
	}
	parsed, err := parseLeaderboardPage(body)
	if err != nil {
		return nil, nil, fmt.Errorf("FetchCatalog: parse: %w", err)
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

// FetchActivePlaylists retourne les playlists CLASSÉES ACTIVES exposées par le menu
// déroulant de la page classement Waypoint (portion `playlists` de FetchCatalog) :
// c'est la source directe autoritative des playlists actives (le manifest de build
// renvoie un PlaylistLinks vide). `refPlaylistID` = une playlist connue servant de
// graine pour charger la page. Renvoie asset id + nom affiché de chaque playlist.
func (s *LeaderboardScraper) FetchActivePlaylists(ctx context.Context, refPlaylistID string) ([]domain.WorldPlaylistRef, error) {
	_, playlists, err := s.FetchCatalog(ctx, refPlaylistID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.WorldPlaylistRef, 0, len(playlists))
	for _, pl := range playlists {
		if id := strings.TrimSpace(pl.ID); id != "" {
			out = append(out, domain.WorldPlaylistRef{AssetID: id, DisplayName: pl.DisplayName})
		}
	}
	return out, nil
}

// FetchSeasons retourne la liste des saisons CSR exposées par le menu déroulant de
// la page classement Waypoint (portion `seasons` de FetchCatalog), dans l'ordre de
// la page (récentes d'abord). Chaque entrée porte le nom EN (DisplayName) et sa
// traduction FR résolue (fr-FR, fallback EN). Source autoritative pour season_catalog
// (C2). `refPlaylistID` = une playlist classée quelconque servant de graine.
func (s *LeaderboardScraper) FetchSeasons(ctx context.Context, refPlaylistID string) ([]domain.WorldSeasonRef, error) {
	seasons, _, err := s.FetchCatalog(ctx, refPlaylistID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.WorldSeasonRef, 0, len(seasons))
	for _, se := range seasons {
		if id := strings.TrimSpace(se.ID); id != "" {
			out = append(out, domain.WorldSeasonRef{SeasonID: id, DisplayName: se.DisplayName, NameFR: se.FrenchName()})
		}
	}
	return out, nil
}

// FrenchName retourne la traduction fr-FR de la saison (insensible à la casse de la
// clé locale), sinon le DisplayName EN en secours.
func (r WaypointRef) FrenchName() string {
	for k, v := range r.Translations {
		if strings.EqualFold(strings.TrimSpace(k), "fr-FR") {
			if fr := strings.TrimSpace(v); fr != "" {
				return fr
			}
		}
	}
	return r.DisplayName
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
	SeasonID     string            `json:"seasonId"`
	DisplayName  string            `json:"displayName"`
	Translations map[string]string `json:"translations"`
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
