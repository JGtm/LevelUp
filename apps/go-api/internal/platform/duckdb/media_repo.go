// Package duckdb - media_repo.go : acces DB pour la galerie medias.
//
// Le code est decoupe en fichiers thematiques pour respecter la limite des
// 500 lignes par fichier (CLAUDE.md). Ce fichier contient le type repo, le
// constructeur, l'adapter d'asset URL, socialDB, LoadMediaFiles +
// enrichMediaModeCategories + CountMediaFiles. Les autres responsabilites
// vivent dans :
//
//   - media_repo_filters.go      : LoadMediaFilterOptions +
//     translatePlaylistFilterOptions +
//     LoadMatchCandidatesForMedia +
//     loadMatchLobbies + normalizeModeLabel
//   - media_repo_writes.go       : SetMediaMatchAssociation + MediaExists +
//     SetMediaLikeAtomic + ToggleSharedLike +
//     GetMediaLikers + queryConfig + joinStrings
//   - media_repo_translations.go : enrichMediaMapTranslations +
//     loadMapCatalogNames +
//     translateMap/ModeFilterOptions +
//     loadAssetTranslationNames +
//     loadMapImageURLsByID +
//     loadModeNameTranslations
package duckdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// MediaRepo implémente port.MediaRepository.
//
// Comme HomeRepo / MatchViewService, un `TitleAssetURLAdapter` optionnel peut
// être injecté via `WithAssetURL` pour résoudre l'URL d'image map quand le
// `map_images_registry` (DB cache) n'a pas d'entrée pour un map_id donné.
// Dans ce cas, l'adapter scanne `static/maps/...` au boot et résout l'URL
// à partir du nom EN (asset_translations en-US). Évite la dépendance à la CLI
// `migrate-static-maps` à chaque nouveau fichier static.
type MediaRepo struct {
	pdb      *PlayerDB
	assetURL mediaAssetURLAdapter
	// modeTax : classification des modes du titre, injectée au wiring pour éviter
	// le couplage platform/duckdb → games/halo_infinite (F1). Zéro-value = pas de
	// classification (dégradation gracieuse).
	modeTax analysis.ModeTaxonomy
	// viewerSlug : joueur qui REGARDE la galerie (session), injecté par requête au
	// wiring (WithViewer). Détermine l'état du cœur servi au front : `liked` = « ce
	// média est liké PAR CE VIEWER », lu dans media_likes_latest. Distinct de
	// pdb.Gamertag, qui est le PROPRIÉTAIRE de la galerie consultée — les deux
	// diffèrent dès qu'on consulte les médias d'un coéquipier.
	// Vide → repli sur le propriétaire, cf. viewer().
	viewerSlug string
}

// viewer retourne le liker dont l'état de like doit être servi.
//
// Repli sur le propriétaire de la galerie quand aucun viewer n'est injecté :
// une instance MONO-UTILISATEUR (auth_mode=none, pas de login) n'a pas de joueur
// courant en session, et la page qu'on y consulte est celle du joueur local.
// Sans ce repli, tous ses cœurs seraient vides — la régression exacte que le
// passage au like par-viewer devait corriger, retournée contre son cas d'usage
// principal.
//
// Le pendant côté ÉCRITURE est le repli du liker sur le player_slug de la route
// (handlers.resolveLikerIdentity) : lire et écrire doivent désigner le même
// viewer, sinon on like pour un autre que celui dont on affiche le cœur.
func (r *MediaRepo) viewer() string {
	if r.viewerSlug != "" {
		return r.viewerSlug
	}
	if r.pdb == nil {
		return ""
	}
	return r.pdb.Gamertag
}

// mediaAssetURLAdapter est l'interface duck-typed que le caller (registry.go)
// satisfait via halo_infinite.AssetURLAdapter. Mêmes raisons que homeRepo :
// évite l'import de games depuis platform/duckdb et garde le repo minimal.
type mediaAssetURLAdapter interface {
	MapImageURL(mapName string) string
}

// NewMediaRepo crée un MediaRepo pour un joueur.
func NewMediaRepo(pdb *PlayerDB) *MediaRepo {
	return &MediaRepo{pdb: pdb}
}

// WithAssetURL injecte l'AssetURLAdapter pour le fallback name-based d'image
// map dans le picker de réassociation (cf. doc MediaRepo). Optionnel : sans
// adapter câblé, la résolution reste exclusivement via map_images_registry.
func (r *MediaRepo) WithAssetURL(a mediaAssetURLAdapter) *MediaRepo {
	r.assetURL = a
	return r
}

// WithViewer injecte le slug du joueur qui consulte la galerie (session HTTP).
// C'est lui — et non le propriétaire de la page — qui détermine l'état `liked`
// servi au front. Vide ou non appelé : repli documenté dans viewer().
func (r *MediaRepo) WithViewer(slug string) *MediaRepo {
	r.viewerSlug = slug
	return r
}

// WithModeTaxonomy injecte la classification des modes du titre (inférence de
// catégorie + préfixes pair_name). Sans injection, les modes ne sont pas classés
// (dégradation gracieuse). Câblé au wiring depuis games/halo_infinite (F1).
func (r *MediaRepo) WithModeTaxonomy(t analysis.ModeTaxonomy) *MediaRepo {
	r.modeTax = t
	return r
}

// socialDB retourne SharedSocial si disponible, sinon Player (fallback de transition).
func (r *MediaRepo) socialDB() *DB {
	if r.pdb.SharedSocial != nil {
		return r.pdb.SharedSocial
	}
	return r.pdb.Player
}

// LoadMediaFiles charge une page de fichiers médias avec filtres dynamiques (Q37).
//
// Pipeline en 3 phases (refactor P1 / ADR 0016, plus aucune query cross-DB SQL) :
//   - Phase A : SELECT media_files + media_match_associations sur SharedSocial
//     avec filtres LOCAUX (kind, liked, date, player_slug).
//   - Phase B : SELECT match_registry sur SharedReader pour les match_ids
//     résultants (sans préfixe `shared.` — la conn pointe sur shared).
//   - Phase C (Go) : enrich + filtres cross-DB (map/mode/playlist) + dédup
//     mf.file_path (équivalent QUALIFY ROW_NUMBER) + tri + pagination.
func (r *MediaRepo) LoadMediaFiles(ctx context.Context, filters domain.MediaFilters, limit, offset int) ([]domain.MediaFileRow, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	enriched, err := r.runMediaPipeline(ctx, filters, mediaWhereConfig{
		includeMapFilter:      true,
		includeModeFilter:     true,
		includePlaylistFilter: true,
	}, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("LoadMediaFiles: %w", err)
	}

	result := toMediaFileRows(enriched)
	r.enrichMediaMapTranslations(ctx, result)
	r.enrichMediaModeCategories(result)
	return result, nil
}

// enrichMediaModeCategories remplace ModeName par la catégorie custom inférée
// depuis pair_name brut (Assassin/Fiesta/BTB/Ranked/Firefight/Other). La logique
// de classification est injectée via WithModeTaxonomy (F1) — sans taxonomie, les
// modes ne sont pas classés.
func (r *MediaRepo) enrichMediaModeCategories(rows []domain.MediaFileRow) {
	for i := range rows {
		if rows[i].PairNameRaw == nil || strings.TrimSpace(*rows[i].PairNameRaw) == "" {
			// Pas de pair : garder ModeName tel quel. Infinite : un mode sans pair
			// n'existe pas (computedModeLabel vide → ModeName déjà nil, équivalent
			// à l'ancien nil explicite). Halo 5 : ModeName vient du fallback
			// game_variant (DEC-7) et ne doit pas être effacé par la classification.
			continue
		}
		cat := r.modeTax.Classify(*rows[i].PairNameRaw)
		if cat != "" {
			c := cat
			rows[i].ModeName = &c
		}
	}
}

// CountMediaFiles retourne le nombre total de fichiers médias actifs selon les filtres.
//
// Même pipeline que LoadMediaFiles (sans limit/offset) — retourne juste le
// nombre de rows distinctes par file_path après filtres + dédup.
func (r *MediaRepo) CountMediaFiles(ctx context.Context, filters domain.MediaFilters) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	enriched, err := r.runMediaPipeline(ctx, filters, mediaWhereConfig{
		includeMapFilter:      true,
		includeModeFilter:     true,
		includePlaylistFilter: true,
	}, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("CountMediaFiles: %w", err)
	}
	return len(enriched), nil
}

// LoadMediaFilterOptions retourne les valeurs distinctes des filtres carte/mode,
// avec libellés enrichis en FR (asset_translations + mode_name_tr de metadata.duckdb)
// et déduplication par libellé FR. Pour les modes, plusieurs raw EN qui se
// normalisent vers le même FR (ex: "Capture the Flag", "CTF - Ranked", "CTF on
// Bazaar" → "Capture du drapeau") sont fusionnés en une seule entrée.
//
// Pipeline P1 (cf. ADR 0016) : chaque option (maps/modes/playlists) lance le
// pipeline media avec un whereCfg qui EXCLUT son propre filtre (pour montrer
// les alternatives disponibles), puis extrait les pairs distinctes (id, label)
// des rows enrichies, puis enrichit en FR via translate*FilterOptions.
