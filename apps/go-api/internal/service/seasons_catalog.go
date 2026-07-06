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
package service

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"
)

// SeasonCatalogEntry est une saison résolue (TOML + DB unifiés) prête pour
// le calcul de SeasonCounts et l'exposition via field-mappings.
//
// Volontairement plus riche que SeasonWindow (qui n'a que ID/Start/End) : porte
// aussi le label et les extras pour pouvoir alimenter le DTO field-mappings
// avec une saison DB-only sans entrée TOML.
type SeasonCatalogEntry struct {
	ID           string
	Label        string // FR si TOML, sinon Name Waypoint brut
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

// SeasonsCatalog est le résolveur unifié. Stateless et concurrence-safe :
// chaque appel à Load(ctx, titleID) déclenche éventuellement un fetch live
// si la DB est vide (best-effort, fallback gracieux).
type SeasonsCatalog struct {
	repo     port.MetadataRepository // peut être nil → DB skipée
	provider port.SeasonProvider     // peut être nil → pas de lazy-fetch
	static   []SeasonCatalogEntry    // projection TOML, pré-calculée au boot
	logger   *slog.Logger
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
		extra := make(map[string]string, len(e.Extra))
		for k, v := range e.Extra {
			extra[k] = v
		}
		out = append(out, SeasonCatalogEntry{
			ID:           e.ID,
			Label:        label,
			Start:        *e.StartDate,
			End:          e.EndDate,
			DisplayOrder: e.DisplayOrder,
			Extra:        extra,
			Source:       SeasonSourceTOMLOnly,
		})
	}
	return out
}

// Load résout le catalog complet pour un titre. Pipeline :
//
//  1. Lit DB via repo.ListSeasons (si repo != nil)
//  2. Si DB vide ET provider != nil → tente FetchSeasonCalendar + UpsertSeason,
//     puis relit DB. Tokens lus depuis ctx (échec gracieux si absent).
//  3. Merge DB + static TOML par ID
//
// Retourne toujours une slice (possiblement vide). Aucune erreur fatale —
// les échecs de I/O sont loggués et l'appelant reçoit ce qu'on a pu collecter.
// seasonsCatalogLoader est le contrat minimal consommé par CareerService et
// FiltersService : charger les entrées saison d'un titre. Interface consumer-side
// (K1i, ARCHI 8) — leurs champs ne sont plus typés sur *SeasonsCatalog concret,
// mockables en test (même package). Implémenté par *SeasonsCatalog.
type seasonsCatalogLoader interface {
	Load(ctx context.Context, titleID string) []SeasonCatalogEntry
}

func (c *SeasonsCatalog) Load(ctx context.Context, titleID string) []SeasonCatalogEntry {
	dbSeasons := c.loadDBWithFallback(ctx, titleID)
	return mergeSeasonSources(c.static, dbSeasons)
}

// loadDBWithFallback lit la DB, et si vide tente le fetch live + persist.
func (c *SeasonsCatalog) loadDBWithFallback(ctx context.Context, titleID string) []domain.SeasonCalendar {
	if c.repo == nil {
		return nil
	}
	rows, err := c.repo.ListSeasons(ctx, titleID)
	if err != nil {
		c.logger.WarnContext(ctx, "seasons_catalog: ListSeasons échec — fallback static TOML",
			"titleSlug", titleID, "err", err)
		return nil
	}
	if len(rows) > 0 {
		return rows
	}
	if c.provider == nil {
		return nil
	}
	c.logger.DebugContext(ctx, "seasons_catalog: DB vide → tentative fetch live",
		"titleSlug", titleID)
	fetched, _, err := c.provider.FetchSeasonCalendar(ctx, titleID)
	if err != nil {
		// Pas une erreur fatale : token absent ou Waypoint indispo. Tomber sur TOML.
		c.logger.InfoContext(ctx, "seasons_catalog: fetch live échec — fallback static TOML",
			"titleSlug", titleID, "err", err)
		return nil
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
	rows, err = c.repo.ListSeasons(ctx, titleID)
	if err != nil {
		c.logger.WarnContext(ctx, "seasons_catalog: relecture après fetch échec",
			"titleSlug", titleID, "err", err)
		return fetched // best effort : retourne ce qu'on a fetché
	}
	return rows
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
		// DB-only : libellé fallback = Name brut Waypoint, sinon ID.
		fallbackLabel := dbS.Name
		if fallbackLabel == "" {
			fallbackLabel = id
		}
		byID[id] = &SeasonCatalogEntry{
			ID:           id,
			Label:        fallbackLabel,
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
