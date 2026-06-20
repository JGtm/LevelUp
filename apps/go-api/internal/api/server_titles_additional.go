package api

import (
	"log/slog"

	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	halo5 "levelup/go-api/internal/games/halo_5"
	"levelup/go-api/internal/games/mappings"
	platform_duckdb "levelup/go-api/internal/platform/duckdb"
)

// additionalTitleRegistrar câble les adapters (semantic + data global + builder
// player-scoped) d'UN titre additionnel dans le resolver et le ServiceRegistry.
// Un titre = un registrar, dispatché par slug (clé de map, pas de comparaison
// littérale — cf. archlint no_slug_comparison).
type additionalTitleRegistrar func(
	td *titlePkg.TitleDescriptor,
	resolver *games.StaticResolver,
	reg *ServiceRegistry,
	fm *mappings.Registry,
)

// additionalTitleRegistrars mappe le slug d'un titre additionnel vers sa fonction
// d'enregistrement. La CLÉ est l'identité du titre (const de package), jamais un
// gating `slug == ...`. Un 3e titre s'ajoute ici par une entrée + son registrar.
var additionalTitleRegistrars = map[string]additionalTitleRegistrar{
	halo5.TitleSlug: registerHalo5Adapters,
}

// registerAdditionalTitles enregistre les adapters des titres ACTIFS ≠ défaut.
//
// Halo Infinite (le titre par défaut) est câblé directement plus haut dans
// NewRouter et reste BYTE-IDENTIQUE : la boucle le saute via td.IsDefault. Tant
// qu'aucun 2e titre n'est ACTIF (Active() exclut coming_soon/archived), la
// fonction est un NO-OP total — l'unique chemin reste Halo Infinite. À
// l'activation d'un titre (status=active dans son title.toml), Active() le
// retourne et son registrar le câble. Gating registry-driven, jamais par slug.
func registerAdditionalTitles(
	titleRegistry *titlePkg.Registry,
	resolver *games.StaticResolver,
	reg *ServiceRegistry,
	fm *mappings.Registry,
) {
	for _, td := range titleRegistry.Active() {
		if td.IsDefault {
			continue // titre par défaut câblé directement (chemin byte-identique)
		}
		registrar, ok := additionalTitleRegistrars[td.Slug]
		if !ok {
			slog.Warn("additional_title_no_adapter_registrar",
				"title_slug", td.Slug,
				"note", "titre actif sans registrar d'adapter — non servi")
			continue
		}
		registrar(td, resolver, reg, fm)
	}
}

// registerHalo5Adapters câble Halo 5 (titre live-only Phase 1a). Source de
// données = API cryptum live : la SourceFactory NewSpartanTokenSource résout le
// SpartanToken depuis le contexte de requête à chaque appel. Aucune player DB
// n'est nécessaire (identité par gamertag), donc le builder player-scoped ignore
// la *PlayerDB et reconstruit l'adapter live.
func registerHalo5Adapters(
	td *titlePkg.TitleDescriptor,
	resolver *games.StaticResolver,
	reg *ServiceRegistry,
	fm *mappings.Registry,
) {
	fields, ok := fm.Get(td.Slug)
	if !ok {
		slog.Error("adapter_load_failed",
			"title_slug", td.Slug, "kind", "semantic", "reason", "fields_mapping_set_missing")
		return
	}
	assets, _ := fm.GetAssets(td.Slug)
	outcomes, _ := fm.GetOutcomes(td.Slug)

	// Semantic : adapter générique partagé. ranks nil → RankCatalog vide (Halo 5
	// expose un CSR natif, pas de career_rank_translations en metadata).
	if sem := games.NewGenericSemanticAdapter(td.Slug, fields, nil, assets, outcomes); sem != nil {
		resolver.RegisterSemantic(sem)
		slog.Info("adapter_loaded",
			"title_slug", sem.TitleSlug(),
			"kind", "semantic",
			"schema_version", sem.SchemaVersion(),
			"assets_loaded", assets != nil,
			"outcomes_loaded", outcomes != nil,
		)
	} else {
		slog.Error("adapter_load_failed",
			"title_slug", td.Slug, "kind", "semantic", "reason", "fields_mapping_set_nil")
		return
	}

	// Capabilities honnêtes depuis capabilities.toml (seules les surfaces réellement
	// câblées sont supported ; cf. capabilities_parity).
	var caps games.CapabilityMap
	if cset, okCaps := fm.GetCapabilities(td.Slug); okCaps {
		if c, err := games.CapabilityMapFromMappings(cset); err == nil {
			caps = c
		} else {
			slog.Warn("capabilities_convert_failed", "title_slug", td.Slug, "err", err.Error())
		}
	}

	// buildLiveData reconstruit un DataAdapter live (placement title-aware +
	// capabilities). Réutilisé pour le data adapter global ET le builder
	// player-scoped (Halo 5 ignore la player DB).
	buildLiveData := func() games.TitleDataAdapter {
		a := halo5.NewDataAdapter(halo5.NewSpartanTokenSource, slog.Default()).
			WithPlacementTotal(td.PlacementMatches)
		if caps != nil {
			a = a.WithCapabilities(caps)
		}
		return a
	}

	resolver.RegisterData(buildLiveData())
	slog.Info("adapter_loaded",
		"title_slug", td.Slug,
		"kind", "data",
		"source", "live",
		"placement_total", td.PlacementMatches,
	)

	reg.RegisterPlayerDataBuilder(td.Slug, func(_ *platform_duckdb.PlayerDB) games.TitleDataAdapter {
		return buildLiveData()
	})
}
