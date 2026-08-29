// Package api — registry_pages_home.go : factories ServiceRegistry des pages Home /
// MatchHistory / Squad. Extrait de registry_pages.go (K3f god-file split, 2026-07-06), meme package.
package wire

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
	"levelup/go-api/internal/service/teammates"
	sync_pkg "levelup/go-api/internal/sync"
)

// HomeCtx retourne un HomeService + identifiants joueur.
func (r *ServiceRegistry) HomeCtx(ctx context.Context, slug string) (port.HomeService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	homeRepo := r.newHomeRepo(pdb)
	svc := service.NewHomeService(homeRepo).
		WithSocial(duckdb.NewSocialRepo(pdb), slug).
		WithSemanticAdapter(r.semanticFor(pdb.TitleSlug)).
		WithDataAdapter(r.dataAdapterForPDB(pdb)).
		WithMatchesCache(r.homeMatchesCache, pdb.XUID).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithSquadSessionTeammates(duckdb.NewSquadRepo(pdb), r.friendGamertagsResolver()).
		WithCareerLive(r.newCareerLiveService(pdb, homeRepo)).
		WithSkillBadgeResolver(skillBadgeResolverFor(pdb.TitleSlug)).
		// Rejeu 2D des tuiles de match (lien à côté de la playlist) : MÊME service que
		// l'endpoint /replay et l'Explorer — une seule résolution de chemin dans le
		// dépôt. Seul AvailableSet est appelé : un listing de dossier par requête.
		WithReplay(r.replayServiceFor(pdb)).
		WithDemoMode(r.cfg.DemoMode)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// newHomeRepo construit un HomeRepo avec l'AssetURLAdapter du titre câblé
// quand disponible (fallback static FS pour les images map non encore
// peuplées dans map_images_registry — évite la dépendance à
// cmd/migrate-static-maps à chaque ajout de fichier static). Dégradation
// gracieuse : sans titleResolver ou sans adapter, le HomeRepo reste nu et
// ne fait que le lookup registry.
func (r *ServiceRegistry) newHomeRepo(pdb *duckdb.PlayerDB) *duckdb.HomeRepo {
	repo := duckdb.NewHomeRepo(pdb).
		WithPlaylistDisplay(r.playlistLabelConfigFor(pdb))
	// Phase 6 du plan CSR : injection du repo thresholds + saison courante.
	// Sans cette injection, le seuil par défaut (5) est utilisé partout, ce qui
	// est juste pour les saisons S3+ mais incorrect pour les historiques S1-S2.
	if pdb.Metadata != nil {
		csrSeasonID := ""
		if r.cfg != nil {
			// newHomeRepo n'a pas de ctx en paramètre ; le titre vient du pdb.
			csrSeasonID = r.cfg.CSRSeasonIDForTitle(context.Background(), pdb.TitleSlug, nil)
		}
		repo = repo.WithCSRThresholds(duckdb.NewCSRThresholdsRepo(pdb.Metadata), csrSeasonID)
	}
	if r.titleResolver == nil {
		return repo
	}
	adapter, err := r.titleResolver.AssetURL(pdb.TitleSlug)
	if err != nil || adapter == nil {
		return repo
	}
	return repo.WithAssetURL(adapter)
}

// MatchHistoryCtx retourne un MatchHistoryService + identifiants joueur.
func (r *ServiceRegistry) MatchHistoryCtx(ctx context.Context, slug string) (port.MatchHistoryService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewMatchHistoryService(duckdb.NewMatchHistoryRepo(pdb), pdb.Gamertag).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithPlaylistDisplay(r.playlistLabelConfigFor(pdb)).
		// Flag « Prolongation » des lignes Explorer/historique : table
		// réglementaire du titre (regulation.toml). Titre sans table → nil.
		WithRegulation(r.regulationFor(pdb)).
		// Score en MANCHES sur les lignes Explorer/historique : même fichier de config,
		// autre table. Titre qui n'en déclare aucune → nil → tout reste en points.
		WithRoundsDecide(r.roundsDecideFor(pdb)).
		// Image du badge de palier des lignes Explorer/historique : MÊME résolveur
		// title-aware que la home (skill_rank_image_url de RecentMatchItem).
		WithSkillBadgeResolver(skillBadgeResolverFor(pdb.TitleSlug)).
		// Rejeu 2D des lignes (colonne « Rejeu » + filtre replay_scope) : MÊME service
		// que l'endpoint /replay et la Match View — une seule résolution de chemin dans
		// le dépôt. Seul AvailableSet est appelé : un listing de dossier par requête.
		WithReplay(r.replayServiceFor(pdb))
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	// Lien page publique du match (Waypoint pour Infinite) via l'adapter du titre
	// (F3) : nil pour un titre sans page publique (H5) → pas de lien mort.
	if au := r.assetURLFor(pdb.TitleSlug); au != nil {
		svc = svc.WithAssetURL(au)
	}
	// Libellés d'outcome via outcomes.toml du titre (F4) : source de vérité au lieu
	// des littéraux FR ; fallback FR canonique si l'adapter est absent.
	if sem := r.semanticFor(pdb.TitleSlug); sem != nil {
		svc = svc.WithSemantic(sem)
	}
	if pdb.Metadata != nil {
		// PR2 placement X/Y : résolveur season_id → threshold (5 ou 10 selon
		// saison CSR). Fallback à 5 si absent. Cf. csr_thresholds_repo.go.
		svc = svc.WithCSRThresholds(duckdb.NewCSRThresholdsRepo(pdb.Metadata).Get)
	}
	// Gate du module classé du briefing Explorer (DEC-7) : capability
	// match.skill.snapshot, jamais le slug. Inoffensif pour la page Historique
	// (elle ne pose pas include_briefing → briefing étendu non construit).
	svc = svc.WithRankedCapable(r.titleSupportsLiveCSR(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// SquadCtx retourne un SquadService + identifiants joueur.
func (r *ServiceRegistry) SquadCtx(ctx context.Context, slug string) (port.SquadService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewSquadService(duckdb.NewSquadRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag)
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// SquadV2Ctx retourne un SquadV2Service + identifiants joueur. Le service
// utilise un loader production qui résout (titleSlug, gamertag) → PlayerDB
// via le pool global, ce qui permet de charger les matchs des coéquipiers
// sans avoir leur player_slug — ils sont sélectionnés par gamertag dans l'UI.
func (r *ServiceRegistry) SquadV2Ctx(ctx context.Context, slug string) (port.SquadV2Service, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	loader := duckdb.NewSquadV2LoaderAdapter(r.resolveByGT)
	// Le loader resout les DBs `shared` (events / weapons / medals) via le main
	// player ; on lui propage le gamertag de la session courante (chunk S11).
	loader.SetDefaultGamertag(pdb.Gamertag)
	svc := service.NewSquadServiceV2(loader)
	// KPI objectifs par xuid (CTF/Zones/Oddball) : gated par la capability
	// match.objective.stats (Infinite ; absente pour Halo 5). Jamais slug==.
	if r.capabilitiesForPDB(pdb).Has(games.CapMatchObjectiveStats) {
		svc = svc.WithObjectiveStatsRepo(duckdb.NewObjectiveStatsRepo(pdb))
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// MatchExclusion retourne un MatchExclusionService pour le joueur, câblé sur
// un MatchRecomputer (recompute global perf_score + LUSR après chaque
// (dé)exclusion). metadataDBPath peut être vide → le recompute fonctionne
// sans bonus médailles.
func (r *ServiceRegistry) MatchExclusion(ctx context.Context, slug string) (port.MatchExclusionService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	metadataPath := ""
	if pdb.Metadata != nil {
		metadataPath = pdb.Metadata.Path()
	}
	// Sprint B1 commit 13b : pdb.SharedDBPath() survit en mode B-swap où
	// pdb.Shared peut être nil. Provider passé pour coordonner avec readers.
	var provider sharedprovider.Provider
	if r.cfg != nil {
		provider = r.cfg.SharedProvider
	}
	recomputer := sync_pkg.NewMatchRecomputer(
		pdb.Player.Path(),
		pdb.SharedDBPath(),
		metadataPath,
		pdb.XUID,
		pdb.Gamertag,
		provider,
	)
	return service.NewMatchExclusionService(
		duckdb.NewMatchExclusionRepo(pdb),
		recomputer,
	), nil
}

// TeammatesCtx retourne un TeammatesService + identifiants joueur.
//
// Le resolver friend_gamertags est branché sur r.settingsStore quand le
// store est attaché (cf. WithSettingsStore). Sans store → comportement
// legacy : top dropdown brut sans filtre amis.
func (r *ServiceRegistry) TeammatesCtx(ctx context.Context, slug string) (port.TeammatesService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	// SessionBriefing mode squad : besoin d'un loader per-gamertag (le
	// playerMatchesAdapterFor est bound au main, ne sait pas charger les
	// canonical rows d'un coequipier different). On reutilise le SquadV2Loader.
	briefingLoader := duckdb.NewSquadV2LoaderAdapter(r.resolveByGT)
	briefingLoader.SetDefaultGamertag(pdb.Gamertag)
	svc := teammates.NewTeammatesService(duckdb.NewSquadRepo(pdb), r.friendGamertagsResolver()).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithSquadLoader(briefingLoader).
		WithKillSourceRepo(r.killSourceClassRepoFor(pdb)).
		WithMedalDefs(duckdb.NewMedalDefinitionsRepo(pdb)).
		// Précision native par arme (Halo 5) : table weapon_accuracy SHARED par titre →
		// le repo lié au PlayerDB du main charge la précision de tous les xuids de
		// l'escouade. Miroir du câblage Synthesis/Sessions ; nil-safe hors h5.
		WithWeaponAccuracyRepo(duckdb.NewWeaponAccuracyRepo(pdb)).
		// Rejeu 2D du tableau historique de l'escouade : MÊME service que l'endpoint
		// /replay et la Match View (une seule résolution de chemin dans le dépôt).
		// Seul AvailableSet est appelé : un listing de dossier par requête.
		WithReplay(r.replayServiceFor(pdb))
	// Axe « Objectifs » par opportunité du radar synergie : gated par la capability
	// match.objective.stats (Infinite ; absente pour Halo 5 → axe retiré de toutes
	// les séries). Source SHARED → couvre aussi les coéquipiers non suivis.
	if r.capabilitiesForPDB(pdb).Has(games.CapMatchObjectiveStats) {
		svc = svc.WithObjectiveIndexRepo(duckdb.NewObjectiveStatsRepo(pdb))
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// friendGamertagsResolver construit un resolver lisant app_settings.friend_gamertags
// à chaque appel. Retourne nil si aucun settings store n'est attaché — le
// service tourne alors en mode legacy.
func (r *ServiceRegistry) friendGamertagsResolver() teammates.FriendGamertagsResolver {
	if r.settingsStore == nil {
		return nil
	}
	return func(ctx context.Context) []string {
		s, err := r.settingsStore.Load()
		if err != nil {
			slog.WarnContext(ctx, "friend_gamertags_load_failed", "err", err)
			return nil
		}
		return s.FriendGamertags
	}
}

// ─── Sprint 54 : Compare + Leaderboard ───────────────────────────────────────

// Compare retourne un CompareService pour le joueur (slug = joueur A).
// Le PlayerStatsProvider (Waypoint) est injecté via DefaultHaloProvider.
// Le contexte est enrichi avec les HaloTokens pour que FetchRemoteStats puisse
// s'authentifier auprès de Waypoint même si la session HTTP ne porte pas de tokens.
func (r *ServiceRegistry) Compare(ctx context.Context, slug string) (port.CompareService, context.Context, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, ctx, "", "", err
	}
	// r.remoteStats (CachedStatsProvider) plutôt que haloProvider direct : Compare
	// bénéficie du même cache TTL 5 min + singleflight que l'Explorer (dédup +
	// latence réduite sur les cibles consultées en parallèle).
	svc := service.NewCompareService(
		duckdb.NewCompareRepo(pdb),
		r.remoteStats,
		pdb.XUID,
		pdb.TitleSlug,
	).WithLiveIdentity(r.newCareerLiveService(pdb, r.newHomeRepo(pdb))).
		WithLiveGamertagResolver(r.liveGamertagResolver) // nil-safe (no-op en démo)
	csrSeasonID := ""
	if r.cfg != nil {
		csrSeasonID = r.cfg.CSRSeasonIDForTitle(ctx, pdb.TitleSlug, nil)
	}
	// CSR live (Compare) : provider spécifique Infinite → gate capability (F2).
	if r.titleSupportsLiveCSR(pdb) {
		svc = svc.WithCSR(r.newExplorerCSRProvider(), csrSeasonID)
	}
	// Catalogue de rangs carrière (même source que le profil de combat) pour
	// afficher le rang en titre ("Général Platine VI") plutôt qu'en numéro.
	if sem := r.semanticFor(pdb.TitleSlug); sem != nil {
		svc = svc.WithRanks(sem.Ranks())
	}
	enriched := r.enrichWithHaloTokens(ctx, pdb)
	return svc, enriched, pdb.XUID, pdb.Gamertag, nil
}

// Leaderboard retourne un LeaderboardService pour le joueur.
func (r *ServiceRegistry) Leaderboard(ctx context.Context, slug string) (port.LeaderboardService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewLeaderboardService(duckdb.NewLeaderboardRepo(pdb))
	return svc, pdb.XUID, pdb.Gamertag, nil
}

// ─── Sprint 55 : Synthesis (extrait de Squad) ────────────────────────────────

// SynthesisCtx retourne un SynthesisService + identifiants joueur.
// Sprint 55 D1 : séparé de SquadCtx pour refléter la frontière produit.
func (r *ServiceRegistry) SynthesisCtx(ctx context.Context, slug string) (port.SynthesisService, string, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", "", err
	}
	svc := service.NewSynthesisService(duckdb.NewSynthesisRepo(pdb)).
		WithPlayerMatchesRepo(r.playerMatchesAdapterFor(pdb), pdb.TitleSlug, pdb.Gamertag).
		WithPersonalScoreAwardsRepo(duckdb.NewPersonalScoreAwardsRepo(pdb), pdb.XUID).
		WithWeaponKillsRepo(duckdb.NewWeaponKillsRepo(pdb)).
		WithKillSourceRepo(r.killSourceClassRepoFor(pdb)).
		WithWeaponAccuracyRepo(duckdb.NewWeaponAccuracyRepo(pdb))
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	// Titres à commendations NATIVES (Halo 5) : véhicules détruits (match_commendations)
	// et vol à la tire (médailles Hijack/Skyjack, medals_earned) — personal_score_awards
	// y est vide. Capability-gated, jamais slug== (parité titleHasNativeKillMechanics —
	// cf. capabilitiesForPDB).
	if r.capabilitiesForPDB(pdb).Has(games.CapCommendationsNative) {
		svc = svc.WithVehicleDestructionStatsRepo(duckdb.NewVehicleCommendationStatsRepo(pdb))
	}
	// KPI objectifs (CTF/Zones/Oddball) : gated par la capability match.objective.stats
	// (Infinite ; absente pour Halo 5 → bloc objective_stats omis). Jamais slug==.
	if r.capabilitiesForPDB(pdb).Has(games.CapMatchObjectiveStats) {
		svc = svc.WithObjectiveStatsRepo(duckdb.NewObjectiveStatsRepo(pdb))
	}
	return svc, pdb.XUID, pdb.Gamertag, nil
}
