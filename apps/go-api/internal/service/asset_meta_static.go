package service

import (
	"context"
	"strings"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// Compile-time check : StaticAssetMetaRepo implémente port.AssetMetaRepository.
var _ port.AssetMetaRepository = (*StaticAssetMetaRepo)(nil)

// StaticAssetMetaRepo est une implémentation in-memory de port.AssetMetaRepository.
// Les données sont chargées une fois au démarrage depuis la DB, puis servies depuis la RAM.
// Avantage : aucune connexion DB persistante → pas de lock fichier Windows (Air hot-reload).
type StaticAssetMetaRepo struct {
	maps    []canonical.AssetMeta
	weapons []canonical.AssetMeta
}

// NewStaticAssetMetaRepo crée un repo in-memory à partir de slices pré-chargées.
func NewStaticAssetMetaRepo(maps, weapons []canonical.AssetMeta) *StaticAssetMetaRepo {
	return &StaticAssetMetaRepo{maps: maps, weapons: weapons}
}

// ListMapsByTitle retourne les maps filtrées par search (LIKE case-insensitive sur NameEN).
// titleID est ignoré — les maps sont globales (asset_translations n'a pas de colonne title_id).
func (r *StaticAssetMetaRepo) ListMapsByTitle(_ context.Context, _ string, search string) ([]canonical.AssetMeta, error) {
	return filterAssets(r.maps, search), nil
}

// ListWeaponsByTitle retourne les armes filtrées par search (LIKE case-insensitive sur NameEN).
func (r *StaticAssetMetaRepo) ListWeaponsByTitle(_ context.Context, _ string, search string) ([]canonical.AssetMeta, error) {
	return filterAssets(r.weapons, search), nil
}

func filterAssets(items []canonical.AssetMeta, search string) []canonical.AssetMeta {
	if search == "" {
		return items
	}
	lower := strings.ToLower(search)
	var out []canonical.AssetMeta
	for _, m := range items {
		if strings.Contains(strings.ToLower(m.NameEN), lower) {
			out = append(out, m)
		}
	}
	return out
}
