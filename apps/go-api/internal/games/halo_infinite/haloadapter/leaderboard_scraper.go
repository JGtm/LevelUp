package haloadapter

// leaderboard_scraper.go — le classement CSR mondial lu dans les pages publiques Halo Waypoint
// (`/halo-infinite/leaderboards/{saison}/{playlist}?page=N`), rendues côté serveur, sans
// authentification, paginées : l'API du client de jeu n'expose PAS de classement mondial. Le job
// cmd/snapshot-world-leaderboard l'appelle puis persiste en append-only.
//
// La brique (`infinite.Classement`) lit les pages et leur JSON `__NEXT_DATA__` ; ici : la
// pagination polie, le rang recalculé (le `rank` du JSON vaut 0), le palier dérivé du CSR, les
// noms par langue, la graine de découverte et le compteur de page vide. NE PAS dépendre du proxy
// tiers sr-nextjs.vercel.app.

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/infinite"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/observability"
)

// logModule est l'attribut de routage des journaux du classement vers logs/leaderboard.log
// (internal/observability/logging.ModuleLeaderboard ; valeur littérale pour ne pas coupler ce
// paquet au paquet de journalisation).
const logModule = "leaderboard"

const (
	defaultScraperTimeout  = 20 * time.Second
	defaultLeaderboardPage = 100 // taille de page observée (repli si absente du JSON)
	// leaderboardMaxPages : garde-fou si la détection de fin échoue (25 × 100 = 2 500 joueurs au
	// plus par playlist).
	leaderboardMaxPages = 25
	// compteurPage1Vide : canari de changement de markup (page 1 sans entrée).
	compteurPage1Vide = "leaderboard_empty_page1"
)

// seedSeasonID est la graine de découverte par DÉFAUT, utilisée tant que le cron n'a pas injecté
// de dernière saison persistée (base vide au premier démarrage). Sa seule fonction est de
// construire une URL de classement qui rend la page : la valeur rendue est TOUJOURS seasons[0].
//
// ATTENTION — les saisons passées NE restent PAS accessibles indéfiniment sur le site : une
// graine périmée (cette constante comme la dernière saison persistée injectée par SetSeedSeason)
// est un POINT FIXE MORT. Le repli qui ne dépend d'aucune graine est FetchActiveSeasonByRedirect.
const seedSeasonID = "csrseason13-2"

// ErrLeaderboardPageNotFound : la page d'un couple (saison, playlist) rend 404 — la playlist n'était
// pas classée cette saison-là. Cas nominal d'un parcours multi-saisons, à sauter (errors.Is).
var ErrLeaderboardPageNotFound = infinite.ErrLeaderboardPageNotFound

// WaypointRef : une entrée du menu saison/playlist de la page (DisplayName = nom de la page,
// demandée dans la langue de repli ; Translations : nom par étiquette Halo, saisons seulement).
type WaypointRef = infinite.WaypointRef

// LeaderboardScraper récupère le classement CSR mondial depuis Halo Waypoint.
type LeaderboardScraper struct {
	client     *http.Client
	host       string        // surchargeable par les tests (vide : l'adresse de la brique)
	perPage    time.Duration // délai poli entre deux pages
	seedSeason string        // graine de découverte (défaut seedSeasonID ; MAJ via SetSeedSeason)
}

// NewLeaderboardScraper crée un lecteur avec un délai poli entre pages ; la graine de découverte
// vaut seedSeasonID jusqu'à ce que le cron injecte la dernière saison persistée.
func NewLeaderboardScraper(politeDelay time.Duration) *LeaderboardScraper {
	return &LeaderboardScraper{
		client:     &http.Client{Timeout: defaultScraperTimeout},
		host:       infinite.DefaultEndpoints().Hote(infinite.CleWaypointWeb),
		perPage:    politeDelay,
		seedSeason: seedSeasonID,
	}
}

// SetSeedSeason remplace la graine de découverte ; une graine vide est ignorée.
func (s *LeaderboardScraper) SetSeedSeason(seasonID string) {
	if seasonID != "" {
		s.seedSeason = seasonID
	}
}

// classement rend le lecteur de la brique : pages par le client du lecteur, sonde de saison active
// par un jumeau qui NE SUIT PAS les redirections (l'en-tête Location EST la réponse cherchée) ;
// chaque sortie gardée par netguard sous sa surface.
func (s *LeaderboardScraper) classement() *infinite.Classement {
	return infinite.NewClassement(
		netguardDoer{surface: surfaceClassementPage, next: s.client},
		netguardDoer{surface: surfaceClassementSaison, next: s.noRedirectClient()},
	).WithEndpoints(infinite.DefaultEndpoints().Avec(infinite.CleWaypointWeb, s.host))
}

// noRedirectClient : jumeau du client principal (même délai, transport par défaut) qui ne suit
// pas les redirections.
func (s *LeaderboardScraper) noRedirectClient() *http.Client {
	timeout := defaultScraperTimeout
	if s.client != nil && s.client.Timeout > 0 {
		timeout = s.client.Timeout
	}
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// langueDesPages : l'Accept-Language des pages, construit depuis la langue de repli de
// `config/locales.toml` (étiquette Halo, puis code court à q=0,9). DisplayName est donc le nom
// dans cette langue ; les autres langues viennent de Translations.
func langueDesPages() string {
	return i18n.HaloTags(i18n.Fallback)[0] + "," + string(i18n.Fallback) + ";q=0.9"
}

// FetchCSRLeaderboard récupère le classement CSR d'une playlist pour une saison, page après page,
// jusqu'à `limit` entrées (0 = jusqu'à épuisement, borné par leaderboardMaxPages). Rang recalculé
// ((page-1)·taille + i + 1), palier dérivé du CSR, horodatage commun.
func (s *LeaderboardScraper) FetchCSRLeaderboard(ctx context.Context, seasonID, playlistID string, limit int) ([]domain.LeaderboardEntry, error) {
	if strings.TrimSpace(seasonID) == "" || strings.TrimSpace(playlistID) == "" {
		return nil, fmt.Errorf("FetchCSRLeaderboard: season et playlist requis")
	}
	fetchedAt := time.Now().UTC()
	out := make([]domain.LeaderboardEntry, 0, 200)
	lecteur := s.classement()
	for page := 1; page <= leaderboardMaxPages; page++ {
		parsed, err := lecteur.FetchPage(ctx, seasonID, playlistID, page, langueDesPages())
		if err != nil {
			return nil, fmt.Errorf("FetchCSRLeaderboard page %d: %w", page, err)
		}
		slog.DebugContext(ctx, "leaderboard page parsée", "module", logModule,
			"season", seasonID, "playlist", playlistID, "page", page, "entries", len(parsed.Leaderboard))
		if len(parsed.Leaderboard) == 0 {
			signalerPageVide(ctx, page, seasonID, playlistID)
			break // fin de l'échelle
		}
		pageSize := parsed.PageSize
		if pageSize <= 0 {
			pageSize = defaultLeaderboardPage
		}
		var complet bool
		out, complet = ajouterEntrees(out, parsed.Leaderboard, entreesDePage{season: seasonID, playlist: playlistID, page: page, pageSize: pageSize, fetchedAt: fetchedAt}, limit)
		if complet {
			return out, nil
		}
		if len(parsed.Leaderboard) < pageSize {
			break // dernière page (incomplète)
		}
		if !s.attendrePoliment(ctx) {
			return nil, ctx.Err()
		}
	}
	return out, nil
}

// signalerPageVide : une page 1 vide signale une saison/playlist inexistante ou un markup qui a
// évolué ; compteur expvar (/debug/vars → levelup.leaderboard_empty_page1) pour la surveiller.
func signalerPageVide(ctx context.Context, page int, seasonID, playlistID string) {
	if page != 1 {
		return
	}
	observability.IncCounter(compteurPage1Vide)
	slog.WarnContext(ctx, "leaderboard vide en page 1 (saison/playlist inexistante ou markup changé ?)",
		"module", logModule, "season", seasonID, "playlist", playlistID)
}

// entreesDePage : le contexte de conversion des entrées d'une page.
type entreesDePage struct {
	season, playlist string
	page, pageSize   int
	fetchedAt        time.Time
}

// ajouterEntrees convertit les entrées d'une page (gamertag vide ignoré) et rend true quand
// `limit` (> 0) est atteint.
func ajouterEntrees(out []domain.LeaderboardEntry, entrees []infinite.LeaderboardEntryDTO, p entreesDePage, limit int) ([]domain.LeaderboardEntry, bool) {
	for i, e := range entrees {
		gt := strings.TrimSpace(e.Player.Gamertag)
		if gt == "" {
			continue
		}
		tier, subTier := domain.DeriveCSRTier(e.Score)
		out = append(out, domain.LeaderboardEntry{
			Season:    p.season,
			Playlist:  p.playlist,
			Rank:      (p.page-1)*p.pageSize + i + 1,
			XUID:      strings.TrimSpace(e.Player.XUID),
			Gamertag:  gt,
			CSR:       e.Score,
			CSRValue:  e.Score,
			Tier:      tier,
			SubTier:   subTier,
			Category:  string(domain.LeaderboardCSRWorld),
			Value:     float64(e.Score),
			FetchedAt: p.fetchedAt,
		})
		if limit > 0 && len(out) >= limit {
			return out, true
		}
	}
	return out, false
}

// attendrePoliment attend le délai poli entre deux pages ; false si le contexte se termine.
func (s *LeaderboardScraper) attendrePoliment(ctx context.Context) bool {
	if s.perPage <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(s.perPage):
		return true
	}
}

// FetchActiveSeasonByRedirect découvre la saison CSR active SANS page graine : la racine des
// classements répond par une redirection dont l'en-tête Location porte la saison active. C'est le
// repli de FetchActiveSeason, dont la graine finit par être retirée du site.
func (s *LeaderboardScraper) FetchActiveSeasonByRedirect(ctx context.Context) (string, error) {
	return s.classement().FetchActiveSeasonByRedirect(ctx, langueDesPages())
}

// FetchActiveSeason découvre la saison CSR active dans le menu de la page (seasons[0], liste du
// plus récent au plus ancien). `refPlaylistID` : une playlist classée valide quelconque.
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

// FetchCatalog récupère les menus saisons et playlists de la page, par une requête « graine »
// (saison graine, `refPlaylistID`).
func (s *LeaderboardScraper) FetchCatalog(ctx context.Context, refPlaylistID string) (seasons, playlists []WaypointRef, err error) {
	return s.classement().FetchCatalog(ctx, s.seedSeason, refPlaylistID, langueDesPages())
}

// FetchActivePlaylists rend les playlists CLASSÉES ACTIVES du menu de la page (source directe
// autoritative : le manifeste de build rend une liste vide), asset id et nom affiché.
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

// FetchSeasons rend les saisons CSR du menu de la page, dans son ordre (récentes d'abord), avec leur
// nom par langue ([nomsDeSaison]).
// Source autoritative de season_catalog.
func (s *LeaderboardScraper) FetchSeasons(ctx context.Context, refPlaylistID string) ([]domain.WorldSeasonRef, error) {
	seasons, _, err := s.FetchCatalog(ctx, refPlaylistID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.WorldSeasonRef, 0, len(seasons))
	for _, se := range seasons {
		if id := strings.TrimSpace(se.ID); id != "" {
			out = append(out, domain.WorldSeasonRef{SeasonID: id, Name: nomsDeSaison(se)})
		}
	}
	return out, nil
}

// nomsDeSaison rend le nom d'une saison dans chaque langue de `config/locales.toml` : la langue de
// repli (celle dans laquelle la page est demandée, [langueDesPages]) prend DisplayName ; une autre
// langue prend la traduction que la page porte sous l'une de ses étiquettes Halo (insensible à la
// casse), sinon DisplayName. Une valeur vide n'est pas inscrite (Pick replie sur Fallback).
func nomsDeSaison(r WaypointRef) i18n.Text {
	t := i18n.Text{}
	for _, l := range i18n.Locales() {
		nom := r.DisplayName
		if l != i18n.Fallback {
			if tr := traductionDeSaison(r, i18n.HaloTags(l)); tr != "" {
				nom = tr
			}
		}
		if nom != "" {
			t[l] = nom
		}
	}
	return t
}

// traductionDeSaison rend la traduction non vide de la saison sous l'une des étiquettes Halo
// `etiquettes` ; "" sinon.
func traductionDeSaison(r WaypointRef, etiquettes []string) string {
	for k, v := range r.Translations {
		if !slices.ContainsFunc(etiquettes, func(e string) bool { return strings.EqualFold(strings.TrimSpace(k), e) }) {
			continue
		}
		if tr := strings.TrimSpace(v); tr != "" {
			return tr
		}
	}
	return ""
}
