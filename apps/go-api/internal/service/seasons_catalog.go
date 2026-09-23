// Package service — seasons_catalog.go : résolveur unifié des saisons.
//
// V2 saisons : pattern lazy-fetch + persist symétrique au battle pass
// (cf. season_pass_service.go). Combine 3 sources :
//
//  1. TOML statique  (config/titles/{slug}/mappings/assets.toml, kind "season")
//     → libellés FR + display_order + extras (csr_season_id, short_label)
//  2. DB MetadataRepository (table season_calendars)
//     → source de vérité dates (Waypoint corrige parfois rétroactivement)
//  3. SeasonProvider (Waypoint)
//     → fallback live si DB vide (lazy-fetch + persist), token requis dans ctx
//
// Politique de merge :
//   - Index par SeasonID
//   - DB wins pour StartDate/EndDate (dates fraîches)
//   - TOML wins pour Label/ShortLabel/DisplayOrder/CsrSeasonID (i18n + ordre stable)
//   - Saison en DB sans TOML correspondant → conservée avec libellé fallback
//     (Name brut du Waypoint) — l'utilisateur enrichit le TOML quand il veut.
//   - Saison en TOML sans DB correspondante → conservée (ex: démarrage offline)
//
// Le merge garantit qu'une nouvelle Operation Halo apparaît dans la SaisonPill
// dès qu'un user authentifié déclenche le filtre — sans intervention manuelle.
//
// Cache (plan perf 2026-09-23, lot L5b, décision D5b.1) : le catalogue résolu est
// gardé en mémoire PAR TITRE (seasonsCatalogTTL) et l'échec du fetch live est
// mémorisé par titre (seasonsLiveFetchBackoff). Avant, chaque /filters/resolve,
// field-mappings et highlight-matches relisait la base puis retentait le GET Waypoint
// (403, environ 100 ms, 38 échecs sur une matinée — état des lieux C6).
package service

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/observability/timing"
	"levelup/go-api/internal/port"
)

// Durées du cache du catalogue (D5b.1).
const (
	// seasonsCatalogTTL : durée de vie d'un catalogue résolu. Les dates d'une saison
	// bougent à l'échelle de la semaine ; une heure borne l'écart avec la base.
	seasonsCatalogTTL = time.Hour
	// seasonsLiveFetchBackoff : après un échec du fetch live, plus aucun appel réseau
	// pour ce titre pendant cette durée. Le catalogue de repli (TOML + base) est gardé
	// jusqu'au terme de l'attente, puis la requête suivante retente le fetch.
	seasonsLiveFetchBackoff = 30 * time.Minute
)

// SeasonCatalogEntry est une saison résolue (TOML + DB unifiés) prête pour
// le calcul de SeasonCounts et l'exposition via field-mappings.
//
// Volontairement plus riche que SeasonWindow (qui n'a que ID/Start/End) : porte
// aussi le label et les extras pour pouvoir alimenter le DTO field-mappings
// avec une saison DB-only sans entrée TOML.
type SeasonCatalogEntry struct {
	ID string
	// Label = libellé FR si TOML, sinon Name Waypoint brut. Reste le libellé
	// canonique/défaut (consommé par l'Explorer via short_label et les tests).
	Label string
	// LabelEN = libellé EN du TOML (GH3-1). Pour une saison DB-only sans
	// traduction, vaut le même Name brut que Label. Sert la résolution
	// locale-aware du bucket "season" servi à la SaisonPill (projectCatalogToBucket).
	LabelEN      string
	Start        time.Time
	End          *time.Time        // nil = saison ouverte
	DisplayOrder int               // ordre TOML, ou int max(displayOrders)+10 pour DB-only (en fin de liste)
	Extra        map[string]string // copie de TOML.Extra ; vide pour DB-only
	Source       SeasonSource
}

// SeasonSource indique d'où provient l'entrée (utile pour debugging et tests).
type SeasonSource string

const (
	SeasonSourceTOMLOnly SeasonSource = "toml_only"
	SeasonSourceDBOnly   SeasonSource = "db_only"
	SeasonSourceMerged   SeasonSource = "merged"
)

// SeasonsCatalog est le résolveur unifié, concurrence-safe. Un catalogue résolu
// est servi depuis la mémoire tant qu'il est frais ; sinon Load relit la base et,
// si elle est vide, tente un fetch live (best-effort, fallback gracieux). Après un
// échec live, le repli reste caché seasonsLiveFetchBackoff : aucun appel réseau
// pendant cette attente.
type SeasonsCatalog struct {
	repo     port.MetadataRepository // peut être nil → DB skipée
	provider port.SeasonProvider     // peut être nil → pas de lazy-fetch
	static   []SeasonCatalogEntry    // projection TOML, pré-calculée au boot
	logger   *slog.Logger

	now     func() time.Time   // horloge (time.Now hors tests)
	flights singleflight.Group // une seule résolution en vol par titre
	mu      sync.Mutex         // protège byTitle
	byTitle map[string]*seasonsTitleState
}

// seasonsTitleState est l'état du cache pour un titre. Après un échec du fetch
// live, expiresAt porte le terme de l'attente (seasonsLiveFetchBackoff).
type seasonsTitleState struct {
	entries   []SeasonCatalogEntry // catalogue résolu (jamais rendu tel quel : copie à la lecture)
	expiresAt time.Time            // fin de validité du catalogue caché
}

// NewSeasonsCatalog construit un résolveur. `staticAssets` est l'AssetMappingSet
// du titre (kind "season" attendu) ; nil ou vide acceptable. `repo` et
// `provider` peuvent être nil pour des contextes sans DB / sans live (tests
// purs, démo offline) — la résolution dégrade alors vers les sources disponibles.
func NewSeasonsCatalog(
	staticAssets *mappings.AssetMappingSet,
	repo port.MetadataRepository,
	provider port.SeasonProvider,
	logger *slog.Logger,
) *SeasonsCatalog {
	if logger == nil {
		logger = slog.Default()
	}
	return &SeasonsCatalog{
		repo:     repo,
		provider: provider,
		static:   projectTOMLSeasons(staticAssets),
		logger:   logger,
		now:      time.Now,
		byTitle:  make(map[string]*seasonsTitleState),
	}
}

// projectTOMLSeasons est l'équivalent de SeasonsFromAssets mais en SeasonCatalogEntry
// (porte aussi labels/extras pour le merge). Préserve l'ordre TOML.
func projectTOMLSeasons(assets *mappings.AssetMappingSet) []SeasonCatalogEntry {
	if assets == nil {
		return nil
	}
	entries := assets.AllOfKind("season")
	out := make([]SeasonCatalogEntry, 0, len(entries))
	for _, e := range entries {
		if e.StartDate == nil {
			continue
		}
		label, _ := e.Label(mappings.LocaleFR)
		labelEN, _ := e.Label(mappings.LocaleEN)
		extra := make(map[string]string, len(e.Extra))
		for k, v := range e.Extra {
			extra[k] = v
		}
		out = append(out, SeasonCatalogEntry{
			ID:           e.ID,
			Label:        label,
			LabelEN:      labelEN,
			Start:        *e.StartDate,
			End:          e.EndDate,
			DisplayOrder: e.DisplayOrder,
			Extra:        extra,
			Source:       SeasonSourceTOMLOnly,
		})
	}
	return out
}

// seasonsCatalogLoader est le contrat minimal consommé par CareerService et
// FiltersService : charger les entrées saison d'un titre. Interface consumer-side
// (K1i, ARCHI 8) — leurs champs ne sont plus typés sur *SeasonsCatalog concret,
// mockables en test (même package). Implémenté par *SeasonsCatalog.
type seasonsCatalogLoader interface {
	Load(ctx context.Context, titleID string) []SeasonCatalogEntry
}

// Load résout le catalog complet pour un titre. Pipeline :
//
//  0. Catalogue en cache et frais → copie rendue sans aucune I/O (marqueur timing
//     `seasons_catalog_hit`) ; c'est aussi le cas pendant l'attente qui suit un
//     échec live (le repli y est caché). Sinon marqueur `seasons_catalog_miss` puis :
//  1. Lit DB via repo.ListSeasons (si repo != nil)
//  2. Si DB vide ET provider != nil → tente FetchSeasonCalendar + UpsertSeason,
//     puis relit DB. Tokens lus depuis ctx (échec gracieux si absent).
//  3. Merge DB + static TOML par ID, mis en cache pour la durée que la lecture
//     autorise (pas de cache après un échec de lecture de la base).
//
// Retourne toujours une slice (possiblement vide) que l'appelant peut modifier :
// c'est une copie. Aucune erreur fatale — les échecs de I/O sont loggués et
// l'appelant reçoit ce qu'on a pu collecter.
func (c *SeasonsCatalog) Load(ctx context.Context, titleID string) []SeasonCatalogEntry {
	if entries, ok := c.cachedEntries(titleID); ok {
		timing.FromContext(ctx).Section("seasons_catalog_hit")()
		return cloneSeasonCatalog(entries)
	}
	timing.FromContext(ctx).Section("seasons_catalog_miss")()
	// Une seule résolution en vol par titre : les requêtes concurrentes attendent
	// son résultat au lieu de relire la base et de retenter le réseau chacune. Le
	// cache est relu dans le vol : une requête arrivée juste après la fin d'un vol
	// précédent ne relance pas la résolution.
	v, _, _ := c.flights.Do(titleID, func() (any, error) {
		if entries, ok := c.cachedEntries(titleID); ok {
			return entries, nil
		}
		return c.resolve(ctx, titleID), nil
	})
	entries, _ := v.([]SeasonCatalogEntry)
	return cloneSeasonCatalog(entries)
}

// resolve lit les sources, fusionne et met le résultat en cache pour la durée
// que la lecture autorise (0 = pas de cache).
func (c *SeasonsCatalog) resolve(ctx context.Context, titleID string) []SeasonCatalogEntry {
	dbSeasons, cacheFor := c.loadDBWithFallback(ctx, titleID)
	merged := mergeSeasonSources(c.static, dbSeasons)
	if cacheFor > 0 {
		c.mu.Lock()
		c.byTitle[titleID] = &seasonsTitleState{entries: merged, expiresAt: c.now().Add(cacheFor)}
		c.mu.Unlock()
	}
	return merged
}

// cachedEntries rend le catalogue caché du titre s'il est encore frais.
func (c *SeasonsCatalog) cachedEntries(titleID string) ([]SeasonCatalogEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st, ok := c.byTitle[titleID]
	if !ok || !c.now().Before(st.expiresAt) {
		return nil, false
	}
	return st.entries, true
}

// loadDBWithFallback lit la DB, et si vide tente le fetch live + persist. Rend
// aussi la durée pendant laquelle le résultat peut être caché (0 = ne pas cacher).
func (c *SeasonsCatalog) loadDBWithFallback(ctx context.Context, titleID string) ([]domain.SeasonCalendar, time.Duration) {
	if c.repo == nil {
		return nil, seasonsCatalogTTL
	}
	rows, err := c.repo.ListSeasons(ctx, titleID)
	if err != nil {
		// Pas de cache : une base momentanément illisible ne doit pas figer le
		// repli TOML pour une heure — la requête suivante relit.
		c.logger.WarnContext(ctx, "seasons_catalog: ListSeasons échec — fallback static TOML",
			"titleSlug", titleID, "err", err)
		return nil, 0
	}
	if len(rows) > 0 {
		return rows, seasonsCatalogTTL
	}
	if c.provider == nil {
		return nil, seasonsCatalogTTL
	}
	return c.fetchLive(ctx, titleID)
}

// fetchLive tente le fetch Waypoint pour une base vide et rend la durée de cache
// du résultat. Un succès persiste les saisons (catalogue caché seasonsCatalogTTL) ;
// un échec est mémorisé par recordFetchFailure.
func (c *SeasonsCatalog) fetchLive(ctx context.Context, titleID string) ([]domain.SeasonCalendar, time.Duration) {
	c.logger.DebugContext(ctx, "seasons_catalog: DB vide → tentative fetch live",
		"titleSlug", titleID)
	fetched, _, err := c.provider.FetchSeasonCalendar(ctx, titleID)
	if err != nil {
		return nil, c.recordFetchFailure(ctx, titleID, err)
	}
	for _, s := range fetched {
		if uerr := c.repo.UpsertSeason(ctx, s); uerr != nil {
			c.logger.WarnContext(ctx, "seasons_catalog: UpsertSeason échec",
				"titleSlug", titleID, "season_id", s.SeasonID, "err", uerr)
		}
	}
	c.logger.InfoContext(ctx, "seasons_catalog: catalog rafraîchi depuis Waypoint",
		"titleSlug", titleID, "count", len(fetched))
	// Relit la DB pour bénéficier du tri ASC stabilisé par la persistance.
	rows, err := c.repo.ListSeasons(ctx, titleID)
	if err != nil {
		c.logger.WarnContext(ctx, "seasons_catalog: relecture après fetch échec",
			"titleSlug", titleID, "err", err)
		return fetched, seasonsCatalogTTL // best effort : retourne ce qu'on a fetché
	}
	return rows, seasonsCatalogTTL
}

// recordFetchFailure journalise l'échec du fetch live et rend la durée de cache du
// repli (TOML + base vide). La MÉMOIRE DE L'ÉCHEC EST CE CACHE : tant que le repli
// est frais, Load le sert sans relire la base ni rappeler le réseau ; à son terme,
// la requête suivante retente le fetch (et un succès remplace le repli).
//
// Un échec SANS jeton dans le contexte n'a fait aucun appel réseau (le provider
// refuse avant) : il n'est pas mémorisé — sinon une requête anonyme bloquerait
// 30 min le fetch d'une requête authentifiée qui, elle, peut réussir.
func (c *SeasonsCatalog) recordFetchFailure(ctx context.Context, titleID string, err error) time.Duration {
	if ctxkeys.HaloTokens(ctx) == nil {
		c.logger.InfoContext(ctx, "seasons_catalog: fetch live impossible sans jeton — fallback static TOML",
			"titleSlug", titleID, "err", err)
		return 0
	}
	c.logger.ErrorContext(ctx, "seasons_catalog: fetch live échec — fallback static TOML, aucun nouvel essai avant retry_at",
		"titleSlug", titleID, "retry_at", c.now().Add(seasonsLiveFetchBackoff), "err", err)
	return seasonsLiveFetchBackoff
}

// cloneSeasonCatalog copie un catalogue (Extra et End compris) : le catalogue caché
// est partagé entre requêtes, aucun appelant ne doit pouvoir le modifier.
func cloneSeasonCatalog(entries []SeasonCatalogEntry) []SeasonCatalogEntry {
	if entries == nil {
		return nil
	}
	out := make([]SeasonCatalogEntry, len(entries))
	for i, e := range entries {
		if e.End != nil {
			end := *e.End
			e.End = &end
		}
		e.Extra = maps.Clone(e.Extra)
		out[i] = e
	}
	return out
}

// mergeSeasonSources fusionne TOML + DB par ID :
//   - DB wins pour Start/End (dates fraîches)
//   - TOML wins pour Label/DisplayOrder/Extra (i18n + ordre stable)
//   - Saison DB-only conservée avec libellé fallback (Name Waypoint)
//   - Saison TOML-only conservée
//
// L'ordre du résultat est : entrées avec DisplayOrder TOML d'abord (ASC),
// puis DB-only en fin (par StartDate ASC).
func mergeSeasonSources(toml []SeasonCatalogEntry, db []domain.SeasonCalendar) []SeasonCatalogEntry {
	if len(toml) == 0 && len(db) == 0 {
		return nil
	}
	byID := make(map[string]*SeasonCatalogEntry, len(toml)+len(db))
	for i := range toml {
		entry := toml[i]
		byID[entry.ID] = &entry
	}

	// Calcule un displayOrder fallback pour les entrées DB-only : max+10 pour
	// que le tri continue de placer le TOML en tête (ordre marketing) et que
	// les nouvelles saisons inconnues du TOML viennent en queue.
	maxOrder := 0
	for _, e := range toml {
		if e.DisplayOrder > maxOrder {
			maxOrder = e.DisplayOrder
		}
	}

	for i, dbS := range db {
		id := dbS.SeasonID
		if existing, ok := byID[id]; ok {
			// Merge : DB pour dates, TOML pour le reste.
			existing.Start = dbS.StartDate
			existing.End = dbS.EndDate
			existing.Source = SeasonSourceMerged
			continue
		}
		// DB-only : libellé fallback = Name brut Waypoint, sinon ID. Pas de
		// traduction disponible → LabelEN identique (même Name servi FR et EN).
		fallbackLabel := dbS.Name
		if fallbackLabel == "" {
			fallbackLabel = id
		}
		byID[id] = &SeasonCatalogEntry{
			ID:           id,
			Label:        fallbackLabel,
			LabelEN:      fallbackLabel,
			Start:        dbS.StartDate,
			End:          dbS.EndDate,
			DisplayOrder: maxOrder + 10*(i+1),
			Source:       SeasonSourceDBOnly,
		}
	}

	out := make([]SeasonCatalogEntry, 0, len(byID))
	for _, e := range byID {
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].DisplayOrder < out[j].DisplayOrder
	})
	return out
}

// SeasonWindowsFromCatalog projette le catalog vers la liste minimale
// utilisée par BuildSeasonCounts.
func SeasonWindowsFromCatalog(catalog []SeasonCatalogEntry) []SeasonWindow {
	out := make([]SeasonWindow, 0, len(catalog))
	for _, e := range catalog {
		out = append(out, SeasonWindow{ID: e.ID, Start: e.Start, End: e.End})
	}
	return out
}

// errEmpty est conservé pour permettre une distinction future entre erreurs
// I/O et catalog vide (utile en tests si on veut forcer un cas).
var errEmpty = errors.New("seasons_catalog: empty")

var _ = errEmpty
