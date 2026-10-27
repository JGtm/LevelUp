// Package service — asset_service.go : Asset Drawer (maps, armes).
package service

import (
	"context"
	"fmt"

	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// AssetService construit les métadonnées d'assets pour l'Asset Drawer.
type AssetService struct {
	repo        port.AssetMetaRepository
	mapImageURL func(titleID, nameEN string) string // nil = UUID-based URL (fallback)
	// weaponImageURL rend l'URL d'icône ET son mode de rendu (masque à teindre vs
	// image finie) : les deux sont indissociables, le front ne peut pas deviner le
	// second. nil = pas d'image.
	weaponImageURL func(titleID string, weaponID int64) (url string, tinted bool)
}

// NewAssetService crée un AssetService.
func NewAssetService(repo port.AssetMetaRepository) *AssetService {
	return &AssetService{repo: repo}
}

// WithMapImageURL configure une fonction de construction d'URL d'image de map.
// Prioritaire sur l'URL UUID par défaut — utiliser l'adapter TitleAssetURLAdapter.
func (s *AssetService) WithMapImageURL(fn func(titleID, nameEN string) string) *AssetService {
	s.mapImageURL = fn
	return s
}

// WithWeaponImageURL configure une fonction de construction d'URL d'image d'arme,
// keyée par weapon_id (l'ID natif du titre — cf. games.TitleAssetURLAdapter).
func (s *AssetService) WithWeaponImageURL(fn func(titleID string, weaponID int64) (string, bool)) *AssetService {
	s.weaponImageURL = fn
	return s
}

// ListMaps retourne les maps d'un titre avec image_url, UNE entrée par visuel (cf.
// oneCardPerImage) triée par nom canonique, avec le nom dans la langue de la requête
// (`i18n.FromContext`, repli `en`).
func (s *AssetService) ListMaps(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error) {
	items, err := s.repo.ListMapsByTitle(ctx, titleID, search)
	if err != nil {
		return nil, fmt.Errorf("AssetService.ListMaps: %w", err)
	}
	// Init [] plutôt que nil : la valeur sera marshallée à la racine d'une
	// réponse HTTP — un slice nil donne `null` et crashe le front. Cf.
	// testutil.RequireNoNilSlicesWithoutOmitempty.
	out := make([]canonical.AssetMeta, 0, len(items))
	for i := range items {
		switch {
		case items[i].ImageURL != "":
			// Déjà résolue depuis la DB (ex. h5 : image_url officielle) → conserver.
		case s.mapImageURL != nil:
			items[i].ImageURL = s.mapImageURL(titleID, items[i].CanonicalName())
			if items[i].ImageURL == "" {
				continue // variante mode+map sans image : exclure du drawer
			}
		default:
			items[i].ImageURL = fmt.Sprintf("/api/v1/assets/maps/%s/%s/image", titleID, items[i].ID)
		}
		out = append(out, items[i])
	}
	// Une carte par VISUEL (URL d'image résolue), pas par asset : cf. oneCardPerImage.
	loc := i18n.FromContext(ctx)
	return localizeAssets(oneCardPerImage(out, loc), loc), nil
}

// ListWeapons retourne les armes avec image_url, nom dans la langue de la requête.
func (s *AssetService) ListWeapons(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error) {
	items, err := s.repo.ListWeaponsByTitle(ctx, titleID, search)
	if err != nil {
		return nil, fmt.Errorf("AssetService.ListWeapons: %w", err)
	}
	for i := range items {
		// Conserver l'ImageURL déjà résolue depuis la DB (ex. h5 : icon_url officielle).
		if items[i].ImageURL != "" || s.weaponImageURL == nil {
			continue
		}
		// Un ID d'arme non numérique n'a pas d'icône résolvable : on laisse "" plutôt
		// que d'appeler le builder avec un identifiant inventé.
		if id, ok := items[i].WeaponID(); ok {
			items[i].ImageURL, items[i].ImageTinted = s.weaponImageURL(titleID, id)
		}
	}
	// Init [] plutôt que nil : même raison que ListMaps.
	if items == nil {
		items = []canonical.AssetMeta{}
	}
	return localizeAssets(items, i18n.FromContext(ctx)), nil
}

// ListMedals retourne les médailles d'un titre. L'icône est un SPRITE (feuille +
// offset, champs Sprite* déjà résolus en amont depuis medal_definitions) — pas de
// mapImageURL/weaponImageURL. ImageURL reste vide (le front rend le sprite). Nom et description
// sont ceux de la langue de la requête.
func (s *AssetService) ListMedals(ctx context.Context, titleID, search string) ([]canonical.AssetMeta, error) {
	items, err := s.repo.ListMedalsByTitle(ctx, titleID, search)
	if err != nil {
		return nil, fmt.Errorf("AssetService.ListMedals: %w", err)
	}
	if items == nil {
		items = []canonical.AssetMeta{}
	}
	return localizeAssets(items, i18n.FromContext(ctx)), nil
}

// localizeAssets rend une tranche neuve dont chaque entrée porte `Name` et `Description` de la langue
// `loc`. La tranche reçue (peut être celle d'un dépôt en mémoire partagé entre requêtes) n'est jamais
// modifiée : deux requêtes de langues différentes ne se marchent pas dessus.
func localizeAssets(items []canonical.AssetMeta, loc i18n.Locale) []canonical.AssetMeta {
	out := make([]canonical.AssetMeta, len(items))
	for i := range items {
		out[i] = items[i].Localized(loc)
	}
	return out
}
