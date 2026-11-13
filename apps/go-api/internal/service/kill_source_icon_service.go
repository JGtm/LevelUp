package service

// kill_source_icon_service.go — LE RÉFÉRENTIEL DES PICTOGRAMMES D'ARME D'UN TITRE.
//
// POURQUOI. Le pictogramme d'une arme n'existait que sur les éliminations (Match View), liées à un
// match. Une page qui ne connaît qu'une weapon_key (la cause d'une destruction de véhicule du
// rejeu) n'avait aucun moyen de retrouver le dessin d'une arme qui n'a éliminé personne dans le
// match. Ce service sert la table entière, par titre.
//
// SOURCE UNIQUE : l'adapter d'assets du titre (`port.KillSourceIconCatalog`), la même résolution
// que `KillSourceIcon`. Le nom vient de `weapon_names.toml` du titre (toutes les langues gardées en
// mémoire, la langue est choisie APRÈS le cache : ADR 0040), avec repli sur le nom propre de la source.
//
// TITLE-AGNOSTIC : aucune comparaison de slug. Un titre non résolu, ou dont l'adapter n'énumère rien,
// rend une liste vide.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/port"
)

// KillSourceIconService implémente port.KillSourceIconService.
type KillSourceIconService struct {
	assetURL func(titleSlug string) (games.TitleAssetURLAdapter, error)
	paths    *title.PathResolver

	mu    sync.Mutex
	names map[string]map[string]i18n.Text // titre -> weapon_key -> nom par langue
}

// NewKillSourceIconService crée le service. `assetURL` résout l'adapter d'assets d'un titre
// (`games.StaticResolver.AssetURL`), `paths` situe les mappings versionnés du titre.
func NewKillSourceIconService(
	assetURL func(titleSlug string) (games.TitleAssetURLAdapter, error),
	paths *title.PathResolver,
) *KillSourceIconService {
	return &KillSourceIconService{assetURL: assetURL, paths: paths, names: map[string]map[string]i18n.Text{}}
}

// ListKillSourceIcons rend une entrée par weapon_key du titre, triée par clé.
func (s *KillSourceIconService) ListKillSourceIcons(ctx context.Context, titleID string) ([]domain.KillSourceIconEntry, error) {
	out := []domain.KillSourceIconEntry{}
	adapter, err := s.assetURL(titleID)
	if err != nil {
		if errors.Is(err, games.ErrTitleNotResolved) {
			slog.DebugContext(ctx, "pictogrammes d'arme : titre non résolu, référentiel vide", "titleSlug", titleID)
			return out, nil
		}
		return nil, fmt.Errorf("KillSourceIconService.ListKillSourceIcons: %w", err)
	}
	catalog, ok := adapter.(port.KillSourceIconCatalog)
	if !ok {
		return out, nil
	}
	icons := catalog.KillSourceIconsByKey()
	if len(icons) == 0 {
		return out, nil
	}
	names := s.weaponNames(ctx, titleID)
	loc := i18n.FromContext(ctx)
	for _, ic := range icons {
		label := names[ic.WeaponKey].Pick(loc)
		if label == "" {
			label = ic.Label
		}
		out = append(out, domain.KillSourceIconEntry{
			Key: ic.WeaponKey, Label: label, ImageURL: ic.ImageURL, Tinted: ic.Tinted,
		})
	}
	return out, nil
}

// weaponNames rend les noms d'armes du titre (toutes les langues), lus une fois par processus. Un
// fichier absent ou illisible n'est pas mémorisé et n'est pas une erreur de requête : les entrées
// gardent le nom propre de leur source, et le journal le dit.
func (s *KillSourceIconService) weaponNames(ctx context.Context, titleID string) map[string]i18n.Text {
	s.mu.Lock()
	defer s.mu.Unlock()
	if names, ok := s.names[titleID]; ok {
		return names
	}
	set, err := mappings.LoadWeaponNamesInDir(s.paths.TitleMappingsDir(titleID))
	if err != nil {
		slog.WarnContext(ctx, "pictogrammes d'arme : noms d'armes du titre illisibles, noms propres des sources servis",
			"err", err, "titleSlug", titleID)
		return nil
	}
	names := set.Names()
	s.names[titleID] = names
	return names
}
