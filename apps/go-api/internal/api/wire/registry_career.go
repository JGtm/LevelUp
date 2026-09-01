// Package api — registry_career.go : factories services liés à la carrière
// (Career, Achievements, TitleDataAdapter, CareerLive, CSR coverage,
// progression diag) + helpers spécifiques (buildFriendsXPLoader,
// allZeroXPTotal). Découpé de registry.go (god-file split, refactor 2026-05-27).
package wire

import (
	"context"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service"
)

// Career retourne un CareerService pour le joueur identifié par slug.
//
// Phase C+ multi-titres : quand le titre courant est HI, un DataAdapter
// player-scoped est injecté dans le service. GetEncounters passe alors par
// games.TitleDataAdapter.LoadEncounters → projection canonique →
// domain.EncounterDTO, avec parité de payload par construction. Si la
// capability LoadEncounters retourne ErrCapabilityNotSupported, le service
// retombe automatiquement sur s.repo.GetEncounters.
func (r *ServiceRegistry) Career(ctx context.Context, slug string) (port.CareerService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	careerRepo := duckdb.NewCareerRepo(pdb)
	// Phase 6 du plan CSR : injection thresholds repo + saison courante. Permet
	// à GetCSRSnapshots de renseigner PlacementTotal par snapshot.season_id.
	if pdb.Metadata != nil {
		csrSeasonID := ""
		if r.cfg != nil {
			csrSeasonID = r.cfg.CSRSeasonIDForTitle(ctx, pdb.TitleSlug, nil)
		}
		careerRepo = careerRepo.WithCSRThresholds(duckdb.NewCSRThresholdsRepo(pdb.Metadata), csrSeasonID)
	}
	svc := service.NewCareerService(careerRepo).WithTitleSlug(pdb.TitleSlug)
	if a := r.dataAdapterForPDB(pdb); a != nil {
		svc = svc.WithDataAdapter(a)
	}
	if loader := r.buildFriendsXPLoader(pdb); loader != nil {
		svc = svc.WithFriendsXPLoader(loader)
	}
	if r.rankCatalog != nil {
		svc = svc.WithRankCatalog(r.rankCatalog)
	}
	// Title-agnostic (D.2) : injecter la map d'images de rang DU TITRE du joueur.
	// Les images HINF sont keyées par numéro de rang HINF (1..272) ; pour Halo 5
	// le RankNumber est le Spartan Rank (1..152) et le titre n'a aucune image de
	// rang par niveau → sa map est vide/absente, donc on n'injecte rien et le SR
	// s'affiche en chiffre (au lieu d'une image de rang HINF erronée).
	if imgs := r.rankImageURLsByTitle[pdb.TitleSlug]; imgs != nil {
		svc = svc.WithRankImageURLs(imgs)
	}
	// Wiring des amis — utilisé par GetTopEncounters pour le tableau "joueurs
	// les plus croisés (hors amis)". Si le settingsStore n'est pas attaché ou
	// que la liste est vide, GetTopEncounters n'exclut personne (dégradation
	// gracieuse).
	if resolver := r.friendGamertagsResolver(); resolver != nil {
		svc = svc.WithFriendGamertagsResolver(resolver)
	}
	// Résolveur gamertag → xuid : ExplorerRepo.ResolveXUIDByGamertag interroge
	// shared.v_gamertag_lookup (cascade xuid_aliases ∪ match_participants),
	// donc capture les amis qui ne sont pas encore dans xuid_aliases mais
	// déjà apparus en match. Source unique de vérité partagée avec Explorer.
	explorerRepo := duckdb.NewExplorerRepo(pdb, pdb.XUID).WithKillSourceClassifier(r.killSourceClassifierFor(pdb))
	svc = svc.WithFriendXUIDResolver(explorerRepo.ResolveXUIDByGamertag)
	// SeasonsCatalog (TOML + DB + lazy-fetch) — alimente le filtre Saisons
	// + cascade counts dans la section "Matchs marquants". Mêmes seasons
	// que la SaisonPill côté Squad/Explorer.
	if r.seasonsCatalog != nil {
		svc = svc.WithSeasonsCatalog(r.seasonsCatalog)
	}
	if r.cfg != nil {
		if id := r.cfg.CSRSeasonIDForTitle(ctx, pdb.TitleSlug, nil); id != "" {
			svc = svc.WithCSRSeasonID(id)
		}
	}
	return svc, nil
}

// RelationsCtx retourne un RelationsService pour le joueur identifié par slug.
// Page transverse (non gatée) : réutilise CareerRepo (méthode GetRelations,
// lecture seule shared). Aucun adapter / friends nécessaire — le hub affiche
// tous les joueurs récurrents.
func (r *ServiceRegistry) RelationsCtx(ctx context.Context, slug string) (port.RelationsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewCareerRepo(pdb)
	svc := service.NewRelationsService(repo)
	// Phase 2 : segmentation serveur. Le FiltersService résout le sous-ensemble
	// de match_id (expérience/classé, saison/période, playlist/mode, vue
	// solo/escouade) via le MÊME pipeline cascade que /filters/resolve, en
	// réutilisant le FiltersRepo (cross-DB : shared + player DB is_with_friends).
	filtersSvc := service.NewFiltersService(duckdb.NewFiltersRepo(pdb))
	svc = svc.WithFilters(filtersSvc)
	// Phase 3b (additif, best-effort) : badge cross-jeu. Énumère les autres
	// titres actifs via le TitleRegistry et lit leur catalogue shared en RO.
	// nil-safe : si la dépendance n'est pas constructible (config absente,
	// joueur sans xuid), le badge reste inerte sans impacter /relations.
	if cg := r.buildCrossGameCooccurrence(pdb); cg != nil {
		svc = svc.WithCrossGame(cg)
	}
	return svc, nil
}

// buildFriendsXPLoader construit un loader d'historique XP pour tous les amis
// du joueur courant (joueurs référencés dans db_profiles.json, hors joueur
// courant). Retourne nil si cfg est indisponible ou s'il n'y a aucun autre joueur.
func (r *ServiceRegistry) buildFriendsXPLoader(mainPDB *duckdb.PlayerDB) service.FriendsXPLoader {
	if r.cfg == nil {
		return nil
	}
	players, err := r.cfg.LoadPlayers(mainPDB.TitleSlug)
	if err != nil {
		slog.Warn("friends_xp: load players failed (loader disabled)",
			"titleSlug", mainPDB.TitleSlug, "err", err)
		return nil
	}
	// Filtrer pour ne garder que les amis (≠ joueur courant, avec XUID renseigné).
	type friendEntry struct{ gamertag, playerSlug string }
	var friends []friendEntry
	for _, p := range players {
		if p.XUID == "" || p.XUID == mainPDB.XUID {
			continue
		}
		friends = append(friends, friendEntry{gamertag: p.Gamertag, playerSlug: p.PlayerSlug})
	}
	if len(friends) == 0 {
		return nil
	}
	cfg := r.cfg
	titleSlug := mainPDB.TitleSlug
	return func(ctx context.Context, _ string) ([]domain.FriendXPHistory, error) {
		var results []domain.FriendXPHistory
		for _, f := range friends {
			pdb, perr := config.ResolvePlayer(ctx, cfg, f.playerSlug, titleSlug)
			if perr != nil {
				slog.WarnContext(ctx, "friends_xp: resolve failed",
					"gamertag", f.gamertag, "err", perr)
				continue
			}
			history, herr := duckdb.NewCareerRepo(pdb).GetXPHistory(ctx)
			if herr != nil {
				slog.WarnContext(ctx, "friends_xp: get history failed",
					"gamertag", f.gamertag, "err", herr)
				continue
			}
			if len(history) == 0 {
				slog.DebugContext(ctx, "friends_xp: skipped, no_history",
					"gamertag", f.gamertag)
				continue
			}
			// Defense-in-depth : Q7 filtre deja xp_total > 0 cote SQL, mais on
			// re-verifie cote Go pour blinder contre toute regression future
			// d'ecriture (ami avec rows xp_total=0 toujours possible si une
			// migration ulterieure ressuscite le DEFAULT 0).
			if allZeroXPTotal(history) {
				slog.DebugContext(ctx, "friends_xp: skipped, all_zero",
					"gamertag", f.gamertag, "rows", len(history))
				continue
			}
			results = append(results, domain.FriendXPHistory{
				Gamertag: f.gamertag,
				History:  history,
			})
		}
		return results, nil
	}
}

// allZeroXPTotal retourne true si tous les XPTotal de l'historique sont <= 0.
// Helper utilise par buildFriendsXPLoader pour eviter d'exposer une courbe
// plate a 0 cote frontend (ex: ami avec uniquement des rows rank-only).
func allZeroXPTotal(history []domain.XPHistoryPoint) bool {
	for _, p := range history {
		if p.XPTotal > 0 {
			return false
		}
	}
	return true
}

// Achievements retourne un AchievementsService pour le joueur identifié par slug.
//
// Le service merge deux sources :
//   - AchievementsRepo (player_achievements dans stats.duckdb du joueur)
//   - MetadataRepo.GetAchievementDefinitions (xbox_achievement_definitions dans
//     metadata.duckdb partagée)
//
// Aucun DataAdapter requis : les achievements ne sont pas des données canoniques
// de match, donc l'accès direct via repos suit le même pattern que Career.
func (r *ServiceRegistry) Achievements(ctx context.Context, slug string) (port.AchievementsService, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	repo := duckdb.NewAchievementsRepo(pdb)
	metaRepo := duckdb.NewMetadataRepo(pdb)
	return service.NewAchievementsService(repo, metaRepo).WithTitleSlug(pdb.TitleSlug), nil
}

// TitleDataAdapter retourne un games.TitleDataAdapter player-scoped pour le
// joueur courant.
//
// Phase C+ du plan multi-titres : cette méthode est le point d'injection
// utilisé par les handlers /api/v1/players/{slug}/... pour consommer la
// couche canonique. Le PlayerDB est résolu avec son CareerRepo, ce qui
// active la capability career.progression pour ce DataAdapter.
//
// Title-agnostic (MT-09) : la factory player-scoped est résolue par slug via
// r.playerDataBuilders. Retourne ErrTitleNotResolved si AUCUN DataAdapter
// player-scoped n'est enregistré pour le titre courant (quel qu'il soit).
func (r *ServiceRegistry) TitleDataAdapter(ctx context.Context, slug string) (games.TitleDataAdapter, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	// MT-09 : lookup de la factory player-scoped par titre (title-agnostic).
	build, ok := r.playerDataBuilders[pdb.TitleSlug]
	if !ok {
		return nil, fmt.Errorf("%w: %q (aucun DataAdapter player-scoped enregistré pour ce titre)",
			games.ErrTitleNotResolved, pdb.TitleSlug)
	}
	return build(pdb), nil
}

// CareerLiveCtx retourne un CareerLiveService configuré pour le joueur slug.
// Utilisé par le cron SpartanCustomizationCron qui itère sur tous les joueurs
// du pool toutes les 8h pour rafraîchir la customisation Spartan.
func (r *ServiceRegistry) CareerLiveCtx(ctx context.Context, slug string) (*service.CareerLiveService, *duckdb.PlayerDB, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, nil, err
	}
	homeRepo := r.newHomeRepo(pdb)
	return r.newCareerLiveService(pdb, homeRepo), pdb, nil
}

// newCareerLiveService construit le service live carrière (XP + Spartan ID)
// pour ce joueur. Le cache est process-level (partagé entre joueurs) ; les
// dépendances DB sont scopées à la PlayerDB de ce joueur.
//
// Sans tokens dans le contexte (HomeCtx legacy), le service retombe
// silencieusement sur le fallback DB (LoadLastCareerRank), garantissant que
// la home reste fonctionnelle même sans auth.
func (r *ServiceRegistry) newCareerLiveService(pdb *duckdb.PlayerDB, homeRepo *duckdb.HomeRepo) *service.CareerLiveService {
	repo := duckdb.NewCareerLiveRepo(pdb)
	factory := service.CareerFetcherFactoryFromTokens(10)
	return service.NewCareerLiveService(repo, homeRepo, factory, r.careerLiveCache)
}

// CSRCoverageProvider implémente handlers.CSRCoverageFactory : résout slug →
// (provider coverage, xuid). Utilisé par l'endpoint /_diag/csr-coverage/{slug}
// (Phase 9 du plan pipeline CSR).
func (r *ServiceRegistry) CSRCoverageProvider(ctx context.Context, slug string) (handlers.CSRCoverageProvider, string, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, "", err
	}
	return duckdb.NewCSRCoverageRepo(pdb), pdb.XUID, nil
}

// ProgressionDiagProvider implémente handlers.ProgressionDiagFactory : résout
// slug → provider diag pipeline V2 (streaks/records/milestones). Utilisé par
// l'endpoint /_diag/progression/{slug} (Phase 4 plan stabilisation 2026-05-22).
func (r *ServiceRegistry) ProgressionDiagProvider(ctx context.Context, slug string) (handlers.ProgressionDiagProvider, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return duckdb.NewProgressionDiagRepo(pdb), nil
}

// PrestigeTelemetryDiagProvider implémente handlers.PrestigeTelemetryDiagFactory :
// résout slug → agrégateur de la télémétrie Prestige par origine du défi. Utilisé
// par l'endpoint /_diag/prestige/telemetry/{slug} (calage coach, ADR 0020).
func (r *ServiceRegistry) PrestigeTelemetryDiagProvider(ctx context.Context, slug string) (handlers.PrestigeTelemetryDiagProvider, error) {
	pdb, err := r.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return duckdb.NewPrestigeTelemetryDiagRepo(pdb), nil
}
