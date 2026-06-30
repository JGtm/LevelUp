package service

import (
	"context"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// Compile-time check : StaticAssetMetaRepo implémente port.AssetMetaRepository.
var _ port.AssetMetaRepository = (*StaticAssetMetaRepo)(nil)

// assetSet regroupe les maps + armes + médailles d'un titre (snapshot in-memory).
type assetSet struct {
	maps    []canonical.AssetMeta
	weapons []canonical.AssetMeta
	medals  []canonical.AssetMeta
}

// StaticAssetMetaRepo est une implémentation in-memory de port.AssetMetaRepository.
// Les données sont chargées une fois au démarrage depuis la DB, puis servies depuis la RAM.
// Avantage : aucune connexion DB persistante → pas de lock fichier Windows (Air hot-reload).
//
// Title-aware : `byTitle` porte les overrides par titre (ex. halo_5, dont les
// maps/armes + URLs viennent de l'API Metadata officielle) ; `fallback` sert les
// titres sans override (rétro-compatible — comportement historique mono-titre).
type StaticAssetMetaRepo struct {
	fallback assetSet
	byTitle  map[string]assetSet
}

// NewStaticAssetMetaRepo crée un repo in-memory à partir de slices pré-chargées
// (titre par défaut / fallback). Ajouter des titres via WithTitle.
func NewStaticAssetMetaRepo(maps, weapons []canonical.AssetMeta) *StaticAssetMetaRepo {
	return &StaticAssetMetaRepo{
		fallback: assetSet{maps: maps, weapons: weapons},
		byTitle:  map[string]assetSet{},
	}
}

// WithTitle ajoute (ou remplace) le jeu d'assets d'un titre spécifique. Chainable.
func (r *StaticAssetMetaRepo) WithTitle(titleID string, maps, weapons, medals []canonical.AssetMeta) *StaticAssetMetaRepo {
	if r.byTitle == nil {
		r.byTitle = map[string]assetSet{}
	}
	r.byTitle[titleID] = assetSet{maps: maps, weapons: weapons, medals: medals}
	return r
}

// WithFallbackMedals câble les médailles du titre PAR DÉFAUT (Halo Infinite), servies
// via le fallback (le constructeur ne prend que maps+weapons, hérité du mono-titre).
// Sans ça, le tab « Médailles » de l'Asset Drawer est VIDE pour Infinite (les médailles
// ne sont jamais chargées au boot — seuls les titres additionnels via WithTitle en ont).
// Chainable.
func (r *StaticAssetMetaRepo) WithFallbackMedals(medals []canonical.AssetMeta) *StaticAssetMetaRepo {
	r.fallback.medals = medals
	return r
}

// setFor retourne le jeu d'assets d'un titre (override si présent, sinon fallback).
func (r *StaticAssetMetaRepo) setFor(titleID string) assetSet {
	if s, ok := r.byTitle[titleID]; ok {
		return s
	}
	return r.fallback
}

// ListMapsByTitle retourne les maps du titre, filtrées par search (LIKE case-insensitive).
func (r *StaticAssetMetaRepo) ListMapsByTitle(_ context.Context, titleID string, search string) ([]canonical.AssetMeta, error) {
	return filterAssets(r.setFor(titleID).maps, search), nil
}

// ListWeaponsByTitle retourne les armes du titre, filtrées par search (LIKE case-insensitive).
func (r *StaticAssetMetaRepo) ListWeaponsByTitle(_ context.Context, titleID string, search string) ([]canonical.AssetMeta, error) {
	return filterAssets(r.setFor(titleID).weapons, search), nil
}

// ListMedalsByTitle retourne les médailles du titre, filtrées par search (LIKE case-insensitive).
func (r *StaticAssetMetaRepo) ListMedalsByTitle(_ context.Context, titleID string, search string) ([]canonical.AssetMeta, error) {
	return filterAssets(r.setFor(titleID).medals, search), nil
}

func filterAssets(items []canonical.AssetMeta, search string) []canonical.AssetMeta {
	if search == "" {
		return items
	}
	lower := strings.ToLower(search)
	var out []canonical.AssetMeta
	for _, m := range items {
		if strings.Contains(strings.ToLower(m.NameEN), lower) ||
			strings.Contains(strings.ToLower(m.NameFR), lower) {
			out = append(out, m)
		}
	}
	return out
}
