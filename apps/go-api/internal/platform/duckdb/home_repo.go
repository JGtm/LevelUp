// Package duckdb — home_repo.go : façade du repository DuckDB pour la page
// d'accueil Mission Control (HomeRepo) + helpers package-private partagés
// par les sous-modules (stringPtr, isTableNotFoundErr, safeStringValue,
// optionalNull*).
//
// La logique est répartie dans les sous-modules suivants :
//   - home_repo_matches.go         : LoadHomeMatches, CountPlayerMatches,
//     LoadHomeSessions, LoadRecentMedia
//   - home_repo_identity.go        : Spartan identity (Q26c, Q26d)
//   - home_repo_skill_peak.go      : CSR/LUSR peak + badge URL builder
//   - home_repo_playlist_ranks.go  : 3 dernières playlists (Q26g, 3 phases)
//   - home_repo_translations.go    : enrichissements FR/EN (legacy + canonical)
//   - home_repo_medals_citations.go: médailles (Q26h), citations (Q26i/j),
//     arme favorite (Q26k)
//   - home_repo_cache.go           : BattlePass + Challenges caches
package duckdb

import (
	"database/sql"
	"strings"

	titlepkg "levelup/go-api/internal/domain/title"
)

const homeStaticTitleSlug = "halo_infinite"

// HomeRepo fournit les données de la page d'accueil depuis DuckDB.
//
// Optionnellement, un `TitleAssetURLAdapter` peut être injecté via
// `WithAssetURL` pour permettre la résolution d'URL d'image map quand le
// `map_images_registry` (DB cache peuplée par cmd/migrate-static-maps) est
// vide pour une map donnée. Dans ce cas, l'adapter scanne `static/maps/...`
// au boot et résout l'URL à partir du **nom EN** (résolu via
// asset_translations en-US). Évite la dépendance manuelle à la CLI à chaque
// ajout de fichier static.
type HomeRepo struct {
	pdb      *PlayerDB
	assetURL homeAssetURLAdapter
}

// homeAssetURLAdapter expose l'unique méthode dont HomeRepo a besoin de
// l'AssetURLAdapter — évite l'import du package `internal/games` (cycle
// potentiel platform/duckdb → games qui dépend de domain → ...). Le caller
// (server.go) injecte directement l'instance halo_infinite.AssetURLAdapter
// qui satisfait cette interface implicite (Go duck-typing).
type homeAssetURLAdapter interface {
	MapImageURL(mapName string) string
}

// NewHomeRepo crée un HomeRepo pour un joueur.
func NewHomeRepo(pdb *PlayerDB) *HomeRepo {
	return &HomeRepo{pdb: pdb}
}

// WithAssetURL injecte l'AssetURLAdapter pour le fallback name-based d'image
// map (cf. doc HomeRepo). Optionnel : sans adapter câblé, la résolution
// reste exclusivement via map_images_registry (registry vide → pas d'image,
// dépend de cmd/migrate-static-maps).
func (r *HomeRepo) WithAssetURL(a homeAssetURLAdapter) *HomeRepo {
	r.assetURL = a
	return r
}

func (r *HomeRepo) titleSlug() string {
	if r == nil || r.pdb == nil {
		return titlepkg.DefaultSlug
	}
	trimmed := strings.TrimSpace(r.pdb.TitleSlug)
	if trimmed == "" {
		return titlepkg.DefaultSlug
	}
	return trimmed
}

// ---------------------------------------------------------------------------
// Helpers package-private partagés par tous les sous-modules home_repo_*.
// Utilisés aussi par career_repo, career_live_repo, squad_repo,
// season_pass_repo, match_history_fr_translations (cf. duckdb package).
// ---------------------------------------------------------------------------

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalNullStringValue(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return strings.TrimSpace(value.String)
}

func optionalNullInt16Value(value sql.NullInt16) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int16)
}

func safeStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// isTableNotFoundErr détecte les erreurs "table not found" DuckDB.
func isTableNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Table with name") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "no such table")
}
