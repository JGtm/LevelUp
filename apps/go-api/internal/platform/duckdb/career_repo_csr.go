// Package duckdb — career_repo_csr.go : GetCSRSnapshots (classements CSR par
// playlist ranked, merge snapshots joueur + catalogue) pour la page Carrière.
// Découpé de career_repo.go (god-file split, refactor 2026-05-27).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/domain"
)

// GetCSRSnapshots retourne les classements CSR du joueur depuis player_csr_snapshots,
// mergés avec le catalogue des playlists ranked actives (metadata.duckdb).
//
// Algo catalogue-first : le catalogue est la source de vérité pour QUELLES
// playlists afficher. Pour chaque entrée catalogue on overlay le snapshot du
// joueur si disponible, sinon "Non classé" synthétique. Les snapshots hors
// catalogue (playlists inactives jouées par le joueur) sont ajoutés en fin de
// liste. Dégradation : si le catalogue est vide/indisponible → snapshots seuls.
//
// Retourne slice vide (pas d'erreur) si ni snapshots ni catalogue ne sont disponibles.
func (r *CareerRepo) GetCSRSnapshots(ctx context.Context) ([]domain.CareerPlaylistCSR, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	snapshots, err := r.loadCSRSnapshotRows(ctx)
	if err != nil {
		return nil, err
	}

	catalog := r.loadRankedPlaylistsCatalog(ctx)
	if len(catalog) == 0 {
		slog.DebugContext(ctx, "GetCSRSnapshots: catalogue indisponible, retour snapshots seuls", "snapshots", len(snapshots))
		return snapshots, nil
	}

	slog.DebugContext(ctx, "GetCSRSnapshots: catalogue-first", "catalog", len(catalog), "snapshots", len(snapshots))

	// Index des snapshots par playlist_id pour lookup O(1).
	snapshotIdx := make(map[string]domain.CareerPlaylistCSR, len(snapshots))
	for _, s := range snapshots {
		snapshotIdx[s.PlaylistID] = s
	}

	// Catalogue en premier : TOUTES les playlists ranked apparaissent.
	catalogSeen := make(map[string]struct{}, len(catalog))
	out := make([]domain.CareerPlaylistCSR, 0, len(catalog)+len(snapshots))
	for _, c := range catalog {
		catalogSeen[c.playlistID] = struct{}{}
		if snap, ok := snapshotIdx[c.playlistID]; ok {
			out = append(out, snap)
		} else {
			threshold := r.csrThreshold(r.currentCSRSID)
			out = append(out, newPlacementPlaylistCSR(c.playlistID, c.name, threshold))
		}
	}

	// Snapshots hors catalogue (playlists inactives déjà jouées par le joueur).
	for _, s := range snapshots {
		if _, ok := catalogSeen[s.PlaylistID]; !ok {
			out = append(out, s)
		}
	}

	r.enrichCSRPlaylistNames(ctx, out)
	return out, nil
}

// enrichCSRPlaylistNames résout les noms de playlists FR via asset_translations.
// Même pattern que enrichLUSRPlaylistNames. Best-effort : silencieux si indisponible.
func (r *CareerRepo) enrichCSRPlaylistNames(ctx context.Context, playlists []domain.CareerPlaylistCSR) {
	if r.pdb == nil || r.pdb.Metadata == nil || len(playlists) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(playlists))
	var ids []string
	for _, p := range playlists {
		id := strings.TrimSpace(p.PlaylistID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}
	names, err := NewMetadataRepoFromDB(r.pdb.Metadata).ResolveAssetNamesBulk(ctx, "playlist", ids, PreferredLangsForLocale("fr"))
	if err != nil || len(names) == 0 {
		return
	}
	for i := range playlists {
		id := strings.TrimSpace(playlists[i].PlaylistID)
		if name := strings.TrimSpace(names[id]); name != "" {
			playlists[i].PlaylistName = name
		}
	}
}

// loadCSRSnapshotRows lit player_csr_snapshots (logique historique). Retourne
// nil sans erreur si la table n'existe pas (joueur jamais syncé pour CSR).
func (r *CareerRepo) loadCSRSnapshotRows(ctx context.Context) ([]domain.CareerPlaylistCSR, error) {
	rows, err := r.pdb.ReadDB().Query(ctx, Q26csrSnapshots)
	if err != nil {
		if isTableNotFoundErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("CareerRepo.GetCSRSnapshots: %w", err)
	}
	defer rows.Close()

	var out []domain.CareerPlaylistCSR
	for rows.Next() {
		var p domain.CareerPlaylistCSR
		var seasonID string // col 5 — exposé via PlacementTotal lookup ci-dessous
		if err := rows.Scan(
			&p.PlaylistID, &p.PlaylistName, &p.Queue, &p.Input,
			&seasonID,
			&p.Current.Value, &p.Current.Tier, &p.Current.SubTier, &p.Current.MeasurementMatchesRemaining,
			&p.Season.Value, &p.Season.Tier, &p.Season.SubTier,
			&p.AllTime.Value, &p.AllTime.Tier, &p.AllTime.SubTier,
		); err != nil {
			return nil, fmt.Errorf("CareerRepo.GetCSRSnapshots scan: %w", err)
		}
		// Phase 6 : lookup threshold par saison du snapshot. Renseigne
		// PlacementTotal sur les 3 niveaux (Current/Season/AllTime) pour que le
		// front puisse afficher "(X/N)" avec le bon N selon l'historique.
		threshold := r.csrThreshold(seasonID)
		p.Current.PlacementTotal = threshold
		p.Season.PlacementTotal = threshold
		p.AllTime.PlacementTotal = threshold
		p.Current.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(
			p.Current.Tier, "", p.Current.SubTier, homeStaticTitleSlug,
			p.Current.MeasurementMatchesRemaining, threshold,
		)
		p.Season.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(
			p.Season.Tier, "", p.Season.SubTier, homeStaticTitleSlug, 0, threshold,
		)
		p.AllTime.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold(
			p.AllTime.Tier, "", p.AllTime.SubTier, homeStaticTitleSlug, 0, threshold,
		)
		out = append(out, p)
	}
	return out, rows.Err()
}

// rankedCatalogEntry : playlist ranked active du catalogue partagé (metadata).
type rankedCatalogEntry struct {
	playlistID string
	name       string
}

// loadRankedPlaylistsCatalog lit playlists_catalog (metadata.duckdb) et retourne
// les playlists ranked actives du titre du joueur. Retourne nil silencieusement
// si la table ou la connexion metadata est indisponible (dégradation legacy).
func (r *CareerRepo) loadRankedPlaylistsCatalog(ctx context.Context) []rankedCatalogEntry {
	if r.pdb == nil || r.pdb.Metadata == nil {
		return nil
	}
	titleSlug := strings.TrimSpace(r.pdb.TitleSlug)
	if titleSlug == "" {
		titleSlug = homeStaticTitleSlug
	}
	rows, err := r.pdb.Metadata.Query(ctx, QPlaylistsCatalogRanked, titleSlug)
	if err != nil {
		slog.WarnContext(ctx, "loadRankedPlaylistsCatalog: query failed (dégradation)", "err", err, "titleSlug", titleSlug)
		return nil
	}
	defer rows.Close()
	var out []rankedCatalogEntry
	for rows.Next() {
		var e rankedCatalogEntry
		if err := rows.Scan(&e.playlistID, &e.name); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out
}

// newPlacementPlaylistCSR construit une ligne synthétique pour une playlist
// ranked du catalogue jamais jouée (0 match de placement effectué). threshold
// est le seuil placement de la saison courante (5 depuis S3, 10 historique).
func newPlacementPlaylistCSR(playlistID, name string, threshold int) domain.CareerPlaylistCSR {
	if threshold <= 0 {
		threshold = CSRPlacementThresholdDefault
	}
	p := domain.CareerPlaylistCSR{
		PlaylistID:   playlistID,
		PlaylistName: name,
	}
	p.Current.MeasurementMatchesRemaining = threshold
	p.Current.PlacementTotal = threshold
	p.Season.PlacementTotal = threshold
	p.AllTime.PlacementTotal = threshold
	p.Current.BadgeImageURL = buildHomeSkillPeakBadgeURLForThreshold("", "", 0, homeStaticTitleSlug, threshold, threshold)
	return p
}
