package service

// map_module_resolver.go — la clé de module des cartes du registre, pour le regroupement des cartes
// republiées (analysis.GroupMapsByBase). Elle vient de la cascade des zones (mapcatalog.CleDeModule,
// essai 1) sur les catalogues déjà décodés par le processus : aucun catalogue n'est relu ici.

import (
	"context"
	"log/slog"
	"os"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/mapcatalog"
)

// MapModuleResolverFor rend le résolveur de clé de module du titre, et `false` quand le titre n'a
// pas de catalogue des bornes (aucune carte intégrée connue : les cartes ne se réunissent alors que
// par variante). Les catalogues sont lus à l'appel du résolveur, par les caches du processus ; une
// lecture en échec est journalisée et rend aucune clé (chaque carte reste seule).
func MapModuleResolverFor(repoRoot, titleSlug string) (analysis.MapModuleResolver, bool) {
	res := title.NewPathResolver(repoRoot)
	if _, err := os.Stat(res.MapQuantBoundsPath(titleSlug)); err != nil {
		return nil, false
	}
	return func(ctx context.Context, maps []analysis.MapNameCount) map[string]string {
		sources, ok := sourcesDeModules(ctx, res, titleSlug)
		if !ok {
			return nil
		}
		modules := make(map[string]string, len(maps))
		for _, m := range maps {
			if module, ok := sources.CleDeModule(mapcatalog.IdentitesDeCarte{MapID: m.MapID, Noms: []string{m.Name}}); ok {
				modules[m.MapID] = module
			}
		}
		return modules
	}, true
}

// sourcesDeModules assemble les deux catalogues que lit l'essai 1 : le catalogue des bornes
// (carte -> module) et le catalogue versionné des zones (le module est une carte intégrée).
func sourcesDeModules(ctx context.Context, res *title.PathResolver, titleSlug string) (mapcatalog.SourcesDeZones, bool) {
	var s mapcatalog.SourcesDeZones
	versionne, err := catalogueDeCallouts(res.MapCalloutsPath(titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "cartes : catalogue des zones illisible, cartes republiées non réunies",
			"err", err, "titleSlug", titleSlug)
		return s, false
	}
	bornes, err := catalogueDeBornes(res.MapQuantBoundsPath(titleSlug))
	if err != nil {
		slog.WarnContext(ctx, "cartes : catalogue des bornes illisible, cartes republiées non réunies",
			"err", err, "titleSlug", titleSlug)
		return s, false
	}
	s.Versionne, s.Bornes = versionne, bornes
	return s, true
}
