// Package service — orchestration du chargement parallèle + assemblage
// séquentiel des données Match View.
//
// Extrait de match_view_service.go (audit #1 god files). Sépare le routage
// errgroup vers les repos (loadMatchViewDataParallel) du builder façade
// (buildMatchViewFromData) qui appelle les builders par onglet définis dans
// match_view_builders_*.go.
package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis/timeline"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/service/killsourceload"

	"golang.org/x/sync/errgroup"
)

// matchViewData centralise les payloads chargés en parallèle pour un match.
// Champ par champ aligné sur les sources repo (Q12/Q14/Q17/Q21/...) — voir
// loadMatchViewDataParallel pour la sémantique de chaque slot.
type matchViewData struct {
	stats      *domain.PlayerMatchStatsRaw
	enrich     *domain.MatchEnrichmentRaw
	scoreboard []domain.ScoreboardRaw
	medals     []domain.MedalRaw
	events     []domain.EventRaw
	// canonicalEvents : chargés via port.HighlightEventsRepository si câblé
	// (chunk MV4.A, loader unifié Phase 0). Sinon, conversion à la volée
	// depuis events (chunk MV2 legacy). Consommés par les builders narrative
	// (cadence + impact 8 rôles).
	canonicalEvents []canonical.HighlightEvent
	// encounterStats : stats riches par encounter (chunk MV4.C') chargées
	// via Q23b. Permet narrative.ComputeEncounterBadges (ally_plus +
	// tough_enemy). Optionnel — degradation gracieuse vers badge ordinal
	// seul si la repo retourne nil.
	encounterStats []domain.EncounterStatsRaw
	// killSources : source de dégât par (tueur, instant), pour l'arme du kill feed
	// (Q21b). Vide si le titre n'a pas de décodeur de film ou si le match n'y est pas
	// passé — le feed s'affiche alors sans icône d'arme.
	killSources []domain.KillSourceRaw
	// killAssists : assistance par (tueur, instant), pour l'assistant du kill feed
	// (Q21c). Une mort absente de la tranche reste « on ne sait pas » — jamais
	// « pas d'assistant ».
	killAssists []domain.KillAssistRaw
	// assistPairs / assistScope : agrégat (assistant → tueur assisté) du match et la
	// PORTÉE de sa lecture (Q21d). assistScope.MatchDeaths à 0 = aucune ligne de film :
	// le builder n'émet alors aucun bloc. Sans clé temporelle — donc jamais recalé T0
	// (cf. correctMatchViewEventsT0, qui ne le touche pas).
	assistPairs []domain.MatchAssistPairRaw
	assistScope domain.MatchAssistScopeRaw
	kvPairs     []domain.KVPairRaw
	// kvPairsFeed : COPIE des paires killer→victim corrigée T0, réservée à la
	// décoration du kill feed (clé exacte tueur+instant contre les events corrigés).
	// kvPairs reste sur l'horloge brute : tug-of-war et KD timeline en dépendent.
	kvPairsFeed []domain.KVPairRaw
	skillRank   *domain.SkillRankRaw
	// sharedCSRs : CSR de tous les participants depuis shared.match_csrs_latest.
	// Nil si match non-ranked ou table absente. Utilisé comme fallback pour les
	// joueurs non-trackés dans buildTeamTabFull.
	sharedCSRs     map[string]*domain.SkillRankRaw
	encounters     []domain.EncounterRaw
	media          []domain.MediaAssocRaw
	expected       *domain.ExpectedStatsRaw
	bulkMedals     []domain.BulkMedalRaw
	bulkWeapons    []domain.BulkWeaponKillRaw
	matchCitations []domain.CitationMatchViewRow
	richCitations  []domain.HomeMatchCitationRaw
	histRows       []domain.MatchHistAvgRow
	// killSourceClasses : kills du joueur AGREGES PAR CLASSE depuis la source de degat
	// — ceux que l attribution arme-a-feu ne voit pas (repulseur, bobines, chute).
	// A ne pas confondre avec `killSources` ci-dessus : celui-la est la source PAR MORT
	// pour l icone du kill feed, celui-ci est un COMPTE PAR JOUEUR pour le sunburst.
	// Vide = titre sans decodeur de film, match jamais decode, ou aucune de ces morts.
	killSourceClasses []port.KillSourceClassRow
	objectiveScore    int
}

// loadMatchViewDataParallel lance en parallèle (errgroup) tous les chargements
// secondaires nécessaires à GetMatchView. Toutes les erreurs sont loggées en
// warn et ignorées (dégradation gracieuse, le builder gère les nil/vides).
//
// Seul `g.Wait()` peut remonter une erreur (annulation ctx).
//
//nolint:funlen,cyclop // routage errgroup linéaire ~20 chargements concurrents — découper davantage ne réduit pas la complexité réelle.
func (s *MatchViewService) loadMatchViewDataParallel(ctx context.Context, matchID string) (matchViewData, error) {
	var d matchViewData

	g, gctx := errgroup.WithContext(ctx)

	goLoad(gctx, g, matchID, "stats", func() error {
		var e error
		d.stats, e = s.repo.GetPlayerMatchStats(gctx, s.xuid, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "enrichment", func() error {
		var e error
		d.enrich, e = s.repo.GetMatchEnrichment(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "scoreboard", func() error {
		var e error
		d.scoreboard, e = s.repo.GetMatchScoreboard(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "objective score", func() error {
		var e error
		// Score PSA catégorie 'objective' pour le joueur courant — consommé
		// UNIQUEMENT par le résiduel de l'axe Score du radar (décision 1, plan
		// PLAN_AXE_OBJECTIFS_INDEX ; l'axe Objectif vient de l'index par
		// opportunité sur le bloc Q12). Dégradation silencieuse à 0.
		d.objectiveScore, e = s.repo.GetMatchObjectiveScore(gctx, s.xuid, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "medals", func() error {
		var e error
		d.medals, e = s.repo.GetMatchMedals(gctx, s.xuid, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "events", func() error {
		var e error
		d.events, e = s.repo.GetMatchEvents(gctx, matchID)
		return e
	})
	// MV4.A : chargement parallèle des events via le loader unifié si câblé.
	// Si l'eventsRepo n'est pas injecté, canonicalEvents reste nil et les
	// builders narrative retomberont sur la conversion à la volée (chunk MV2).
	if s.eventsRepo != nil {
		g.Go(func() error {
			filters := port.HighlightEventFilters{MatchIDs: []string{matchID}}
			if e := filters.Validate(); e != nil {
				slog.WarnContext(gctx, "match_view: HighlightEventFilters invalides",
					"match_id", matchID, "err", e)
				return nil
			}
			canonicalEv, e := s.eventsRepo.Load(gctx, filters)
			if e != nil {
				if !errors.Is(e, games.ErrCapabilityNotSupported) {
					slog.WarnContext(gctx, "match_view: Load highlight events echec",
						"match_id", matchID, "err", e)
				}
				return nil
			}
			d.canonicalEvents = canonicalEv
			return nil
		})
	}

	// MV4.B' : awards chargés après l'errgroup principal car ils dépendent du
	// scoreboard (xuids). Voir l'appel `s.loadAwardsForScoreboard(...)` plus
	// bas, après `g.Wait()`.
	goLoad(gctx, g, matchID, "kill_sources", func() error {
		var e error
		d.killSources, e = s.repo.GetMatchKillSources(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "kill_assists", func() error {
		var e error
		d.killAssists, e = s.repo.GetMatchKillAssists(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "assist_pairs", func() error {
		var e error
		d.assistPairs, d.assistScope, e = s.repo.GetMatchAssistPairs(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "kv_pairs", func() error {
		var e error
		d.kvPairs, e = s.repo.GetMatchKVPairs(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "skill_rank", func() error {
		var e error
		d.skillRank, e = s.repo.GetMatchSkillRank(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "shared_csrs", func() error {
		var e error
		d.sharedCSRs, e = s.repo.GetMatchSharedCSRs(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "encounters", func() error {
		var e error
		d.encounters, e = s.repo.GetMatchEncounters(gctx, matchID, s.xuid)
		return e
	})
	// MV4.C' : chargement parallele des stats encounter riches (Q23b).
	goLoad(gctx, g, matchID, "encounter_stats", func() error {
		var e error
		d.encounterStats, e = s.repo.GetMatchEncounterStats(gctx, matchID, s.xuid)
		return e
	})
	goLoad(gctx, g, matchID, "media", func() error {
		var e error
		// Q24 retourne tous les auteurs (cross-joueur) : un coéquipier peut
		// avoir uploadé un media pour ce match.
		d.media, e = s.repo.GetMatchMedia(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "expected_stats", func() error {
		var e error
		d.expected, e = s.repo.GetMatchExpectedStats(gctx, matchID, s.xuid)
		return e
	})
	goLoad(gctx, g, matchID, "bulk_medals", func() error {
		var e error
		d.bulkMedals, e = s.repo.GetMatchBulkMedals(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "bulk_weapons", func() error {
		var e error
		d.bulkWeapons, e = s.repo.GetMatchBulkWeaponKills(gctx, matchID)
		return e
	})
	goLoad(gctx, g, matchID, "hist_avg", func() error {
		var e error
		d.histRows, e = s.repo.GetHistoryForAvg(gctx, s.xuid)
		return e
	})
	// Le GATE de capability est pose au CABLAGE (wire), la ou le TitleDataAdapter et sa
	// CapabilityMap sont disponibles : `film.kill_source` est une capability DATA-LEVEL
	// (games.CapabilityKey, capabilities.toml), pas une capability title-level. Ici, un
	// repo non nil VEUT DIRE que le titre l a. Zero comparaison de slug.
	if s.killSourceRepo != nil {
		goLoad(gctx, g, matchID, "kill_source_classes", func() error {
			var e error
			d.killSourceClasses, e = s.loadMatchKillSourceClasses(gctx, matchID)
			return e
		})
	}
	if s.citationsRepo != nil {
		goLoad(gctx, g, matchID, "citations", func() error {
			var e error
			d.matchCitations, e = s.citationsRepo.LoadMatchCitationsForView(gctx, matchID)
			return e
		})
		goLoad(gctx, g, matchID, "rich citations", func() error {
			var e error
			d.richCitations, e = s.citationsRepo.LoadMatchCitationsRich(gctx, matchID)
			return e
		})
	}

	if err := g.Wait(); err != nil {
		return matchViewData{}, err
	}
	return d, nil
}

// goLoad lance un chargement best-effort dans l'errgroup g : exécute load(), logge
// un WARN "match_view: <label> indisponible" sur erreur (jamais fatal) puis retourne
// nil. Centralise ~18 blocs g.Go copiés (K2e, CR §5) et passe à slog.WarnContext.
func goLoad(gctx context.Context, g *errgroup.Group, matchID, label string, load func() error) {
	g.Go(func() error {
		if err := load(); err != nil {
			slog.WarnContext(gctx, "match_view: "+label+" indisponible", "match_id", matchID, "err", err)
		}
		return nil
	})
}

// tugDurationMS retourne la durée (ms) servant à binner la Dominance (tug-of-war).
// playable_duration_seconds (API) est prioritaire — Infinite le fournit toujours,
// donc son comportement est inchangé. Fallback title-agnostic quand il est NULL
// (cas de 100 % des matchs Halo 5, dont l'adapter n'écrit jamais ce champ) : la
// durée de gameplay dérivée (duration_seconds − T0 via headerGameplayDurationSeconds),
// sinon 0. Sans ce fallback, ComputeTugOfWar retourne nil (durée ≤ 0) et la
// Dominance est vide sur tout Halo 5 alors que les paires killer/victim existent.
func tugDurationMS(meta *domain.MatchMetaRaw) int64 {
	if meta == nil {
		return 0
	}
	if meta.PlayableDurationSeconds != nil {
		return *meta.PlayableDurationSeconds * 1000
	}
	if gp := headerGameplayDurationSeconds(meta); gp != nil {
		return *gp * 1000
	}
	return 0
}

// buildMatchViewFromData assemble séquentiellement la MatchViewResponse depuis
// la meta + les données chargées en parallèle. Aucun appel I/O bloquant ici à
// l'exception du lookup IsMatchFavorite (PK indexée, cheap) et du loader
// friendsExtras (qui peut faire des fan-out vers les DBs amies — best-effort).
//
//nolint:funlen,cyclop // assemblage linéaire 4 onglets + header — découper davantage casserait la lisibilité du payload final.
func (s *MatchViewService) buildMatchViewFromData(
	ctx context.Context,
	matchID string,
	meta *domain.MatchMetaRaw,
	d matchViewData,
) domain.MatchViewResponse {
	// Durée pour les bins tug-of-war (Dominance).
	durationMS := tugDurationMS(meta)

	// Correction chronologie T0 (Phase 3 : T0 réel depuis meta.T0Ms, 0 si
	// indisponible). Recale TOUTES les sources d'events au référentiel gameplay
	// (vrai début de match) en un POINT UNIQUE, avant les builders combat. Sans
	// ça la cadence/les rôles (canonicalEvents) seraient sur l'horloge gameplay
	// mais le kill-feed (event_time_ms, axe X des charts KD-cumul/frag-diff/tug)
	// + les badges d'impact (events bruts) resteraient sur l'horloge film
	// (countdown inclus) → incohérence visuelle sur la même page.
	var t0Ms int64
	if meta != nil && meta.T0Ms != nil {
		t0Ms = *meta.T0Ms
	}
	correctMatchViewEventsT0(&d, matchID, timeline.BuildForMatchMs(durationMS, t0Ms))

	// IsFavorite : lookup synchrone (cheap, indexé sur PK player_slug+match_id).
	// Dégradation gracieuse si socialRepo nil ou shared_social indisponible.
	isFavorite := false
	if s.socialRepo != nil && s.playerSlug != "" {
		if fav, ferr := s.socialRepo.IsMatchFavorite(ctx, s.playerSlug, matchID); ferr == nil {
			isFavorite = fav
		} else {
			slog.WarnContext(ctx, "match_view: IsMatchFavorite échoué",
				"match_id", matchID, "player", s.playerSlug, "err", ferr)
		}
	}

	// expected_assists — uniquement pour le joueur suivi (is_me), jamais pour
	// les autres lignes du scoreboard.
	// Chaîne de résolution :
	//   1. Modèle personnel OLS (player_assists_model dans stats.duckdb)
	//   2. Fallback modèle populationnel (assists_model_coefs dans metadata.duckdb)
	if d.stats != nil && meta != nil && meta.GameVariantName != nil {
		v := computeExpectedAssists(ctx, s.repo, s.metadataRepo, *meta.GameVariantName, d.stats)
		if v != nil {
			if d.expected == nil {
				d.expected = &domain.ExpectedStatsRaw{}
			}
			d.expected.AssistsExpected = v
			// Propager aussi sur la ligne is_me du scoreboard (expander PlayerDetailPanel).
			for i := range d.scoreboard {
				if d.scoreboard[i].XUID == s.xuid {
					d.scoreboard[i].AssistsExpected = v
					break
				}
			}
		}
	}

	header := buildMatchHeader(ctx, matchID, meta, d.stats, d.enrich, d.scoreboard, s.assetURL, isFavorite)
	// Flag « Prolongation » : résolu APRÈS le header (la table réglementaire est
	// portée par le service, pas par le builder — buildMatchHeader est déjà à la
	// limite de paramètres). Titre sans table → no-op.
	applyMatchHeaderOvertime(&header, meta, s.regulationSeconds)
	// Score de l'en-tête : points ou MANCHES. Ici et pas dans le builder, pour la même
	// raison que la ligne au-dessus — la table `[rounds_decide]` est portée par le
	// service. Table absente → lecture en points, comportement d'avant le 2026-08-29.
	applyMatchHeaderScore(&header, meta, d.stats, s.roundsDecide)
	// Présence de l'artefact de rejeu 2D : un os.Stat, jamais une lecture. Même
	// raison d'être ici que le flag « Prolongation » — la dépendance est portée par
	// le service, pas par le builder.
	applyMatchHeaderReplay(ctx, &header, matchID, s.replaySvc)
	// ModeCategory : catégorie custom résolue depuis pair_name (taxonomie injectée,
	// WithModeTaxonomy) — pour que la garde Fiesta du rejeu 2D corrèle sur la
	// résolution de mode de l'app plutôt que deviner sur ModeUI/PlaylistLabel.
	applyMatchHeaderModeCategory(&header, meta, s.modeTaxonomy)
	rank := buildRankBlock(d.skillRank, s.assetURL)
	curDurSec := 0
	if meta != nil && meta.DurationSeconds != nil {
		curDurSec = int(*meta.DurationSeconds)
	}
	summary := buildSummaryTabFull(d.stats, d.medals, d.expected, d.histRows, meta, s.titleSlug, d.richCitations, curDurSec)
	// Proba de victoire pré-match (LUSR v2) → card « Résultat attendu ». Source :
	// match_skill_rank_latest.expected_win_prob via d.skillRank (même lecture que le
	// player-matches scan). Best-effort : nil pour les matchs pré-v2 / sans donnée.
	if d.skillRank != nil && d.skillRank.ExpectedWinProb != nil {
		summary.ExpectedStats.ExpectedWinProb = d.skillRank.ExpectedWinProb
	}
	// Propage l'expected K/D LOCAL (modèle count∝durée, Halo 5) sur la ligne is_me
	// du scoreboard → le drawer (expander) affiche attendu vs réel sur les 3 stats,
	// pas seulement les assists. Limité au is_me (seul joueur dont l'historique est
	// chargé ici). Doit précéder buildTeamTabFull (qui projette d.scoreboard).
	if summary.ExpectedStats.LocallyEstimated {
		for i := range d.scoreboard {
			if d.scoreboard[i].XUID == s.xuid {
				d.scoreboard[i].KillsExpected = summary.ExpectedStats.ExpectedKills
				d.scoreboard[i].DeathsExpected = summary.ExpectedStats.ExpectedDeaths
				d.scoreboard[i].LocallyEstimated = true
				break
			}
		}
	}
	combat := buildCombatTabFull(matchID, d.bulkWeapons, d.events, d.canonicalEvents, d.kvPairs, d.scoreboard, s.xuid, durationMS)
	// L'arme du kill et l'équipe du tueur se posent APRÈS l'assemblage : ce sont des
	// décorations du feed, pas des entrées du calcul de dominance (les bins, les vagues
	// et les cumuls ne dépendent d'aucune des deux). Les séparer garde buildCombatTabFull
	// à sa responsabilité et rend la décoration testable seule.
	decorateKillFeed(ctx, combat.HighlightEvents, killFeedInputs{
		sources:    d.killSources,
		assists:    d.killAssists,
		victims:    d.kvPairsFeed,
		scoreboard: d.scoreboard,
		assetURL:   s.assetURL,
	})
	// L'identité des médailles se pose par le même modèle : une décoration du feed,
	// résolue contre le référentiel du titre (best-effort).
	decorateMedalEvents(ctx, combat.HighlightEvents, s.repo, s.assetURL)
	// Les paires d'assistance sont un AGRÉGAT PAR MATCH, pas une décoration du feed :
	// elles sortent déjà comptées de Q21d et n'ont besoin que du scoreboard pour nommer
	// le tueur. Posées ici pour la même raison que FragDistribution — hors de
	// buildCombatTabFull, dont la signature est déjà à la limite de paramètres.
	combat.AssistPairs = buildAssistPairs(ctx, d.assistPairs, d.assistScope, d.scoreboard)
	// Extras per-friend (panneau d'expander scoreboard) : best-effort, on
	// charge depuis chaque player DB d'ami configuré. Si pas de loader injecté
	// → map vide (section "Local" inactive sauf pour `is_me`).
	var friendsExtras map[string]port.FriendMatchExtras
	if s.friendsExtras != nil {
		xuids := make([]string, 0, len(d.scoreboard))
		for _, sb := range d.scoreboard {
			if sb.XUID != "" && sb.XUID != s.xuid {
				xuids = append(xuids, sb.XUID)
			}
		}
		if len(xuids) > 0 {
			gvn := ""
			if meta != nil && meta.GameVariantName != nil {
				gvn = *meta.GameVariantName
			}
			friendsExtras = s.friendsExtras(ctx, matchID, gvn, xuids)
		}
	}
	// Même modèle local (count∝durée) pour les AMIS TRACKÉS : leur historique
	// complet est dans shared (joueur synchronisé), chargeable par xuid. On le
	// charge et on pose l'expected K/D sur leur ligne → le drawer affiche attendu
	// vs réel pour eux aussi (cohérent avec is_me). Limité aux xuids présents dans
	// friendsExtras (synchronisés ; l'historique d'un non-tracké ne contiendrait
	// que les matchs communs avec l'escouade → échantillon biaisé). Skip si l'API
	// a déjà fourni les K/D (Infinite).
	if curDurSec > 60 && len(friendsExtras) > 0 {
		// J3 : collecter les xuids amis éligibles, charger tout leur historique en
		// UN seul appel (GetHistoryForAvgBulk) au lieu de ~8 GetHistoryForAvg
		// séquentiels, puis réappliquer le même filtre d'éligibilité pour écrire
		// l'expected K/D. Best-effort : bulk en échec → map vide → aucun expected
		// (dégradation identique à la voie unitaire).
		var needXUIDs []string
		seen := make(map[string]bool)
		eligible := func(i int) bool {
			xuid := d.scoreboard[i].XUID
			if xuid == s.xuid || d.scoreboard[i].KillsExpected != nil {
				return false
			}
			_, tracked := friendsExtras[xuid]
			return tracked
		}
		for i := range d.scoreboard {
			if !eligible(i) {
				continue
			}
			if xuid := d.scoreboard[i].XUID; !seen[xuid] {
				seen[xuid] = true
				needXUIDs = append(needXUIDs, xuid)
			}
		}
		if len(needXUIDs) > 0 {
			histByXUID, err := s.repo.GetHistoryForAvgBulk(ctx, needXUIDs)
			if err == nil {
				for i := range d.scoreboard {
					if !eligible(i) {
						continue
					}
					fh := histByXUID[d.scoreboard[i].XUID]
					if len(fh) == 0 {
						continue
					}
					if ek, ed, ok := localExpectedKD(fh, meta, curDurSec); ok {
						d.scoreboard[i].KillsExpected = ek
						d.scoreboard[i].DeathsExpected = ed
						d.scoreboard[i].LocallyEstimated = true
					}
				}
			}
		}
	}
	team := buildTeamTabFull(d.scoreboard, d.kvPairs, d.encounters, d.encounterStats, d.bulkMedals, d.bulkWeapons, s.xuid, s.titleSlug, d.enrich, d.skillRank, friendsExtras, d.sharedCSRs, s.assetURL)
	// Halo 5 persisté : libellés d'équipe « Rouge/Bleu » depuis team_colors (no-op HINF
	// et si le référentiel est vide → le front garde son libellé existant).
	s.applyTeamNames(ctx, team.Scoreboard)
	// FragDistribution v2 du viewer (sunburst classe→rôle) : construite après le
	// scoreboard (compteurs natifs melee/grenade/spartan de la ligne is_me) + les bulk
	// weapon kills du viewer (classes gun). hasMechanics via capability (jamais slug==).
	combat.FragDistribution = buildViewerFragDistribution(
		findViewerScoreboardRow(team.Scoreboard), d.bulkWeapons, d.killSourceClasses,
		titleHasNativeKillMechanics(s.titleSlug),
	)
	if combat.FragDistribution != nil {
		logFragDistribution(ctx, "match view", s.titleSlug, s.xuid, *combat.FragDistribution)
	}
	mediaTab := buildMediaTab(d.media)

	// MV4.B' : radar calculé depuis le scoreboard (kills/HS/PK/assists/accuracy/
	// deaths/damage/score + bloc objectif Q12). Mêmes formules que le radar squad
	// (loadSynergyMateAxes), appliquées à un seul match. L'axe Objectif = index
	// par opportunité sur row.Obj (retiré si match sans bloc objectif) ; le PSA
	// d.objectiveScore ne sert plus qu'au résiduel de l'axe Score (décision 1,
	// plan PLAN_AXE_OBJECTIFS_INDEX).
	modeFamily := matchModeFamilyFromMeta(meta)
	radarSeries := BuildMatchRadarFromScoreboard(d.scoreboard, s.xuid, d.objectiveScore, modeFamily, games.EffectiveHpToKill(s.titleSlug), games.OffensiveConversionP80(s.titleSlug))
	radar := make([]any, 0, len(radarSeries))
	for _, rs := range radarSeries {
		radar = append(radar, rs)
	}

	// RC6 — détection sync incomplet : le match_registry est OK (sinon on aurait
	// court-circuité plus haut), mais une ou plusieurs sources secondaires sont
	// vides. Le front peut afficher un bandeau dégradé au lieu de l'écran
	// "Match introuvable ou erreur de chargement" full-page.
	partialReasons := detectPartialMatchData(d.stats, d.scoreboard, d.events, d.medals)

	return domain.MatchViewResponse{
		Header:         header,
		Rank:           rank,
		SummaryTab:     summary,
		CombatTab:      combat,
		TeamTab:        team,
		MediaTab:       mediaTab,
		CitationsTab:   buildCitationsTab(d.matchCitations, d.medals, s.titleSlug),
		Radar:          radar,
		IsPartial:      len(partialReasons) > 0,
		PartialReasons: partialReasons,
	}
}

// correctMatchViewEventsT0 recale les sources d'events d'un match au référentiel
// gameplay (T0 retranché) en un point unique, avant les builders combat.
//
// canonicalEvents (cadence + 8 rôles) ET events bruts (evtList/kill-feed +
// badges d'impact) partagent ainsi la même horloge — celle du vrai début de
// match (meta.T0Ms). kvPairs n'est PAS recalé : ses consommateurs ne dépendent
// pas de kv.TimeMS pour l'affichage (killer/victim n'utilise pas le temps ;
// l'axe tug-of-war dérive de la durée du match, pas des temps de frags ; la
// sortie kd_timeline n'est pas rendue en Match View — le front reconstruit la
// courbe K/D depuis event_time_ms). T0=0/inconnu → identité (recalage no-op).
func correctMatchViewEventsT0(d *matchViewData, matchID string, tl domain.MatchTimeline) {
	if len(d.canonicalEvents) > 0 {
		d.canonicalEvents = timeline.CorrectEvents(
			d.canonicalEvents, map[string]domain.MatchTimeline{matchID: tl},
		)
	}
	d.events = timeline.CorrectEventRaws(d.events, tl)
	// Q21b/Q21c s'apparient aux events par clé EXACTE (xuid, time_ms) dans
	// decorateKillFeed : ils DOIVENT subir la même correction que d.events.
	// Sans ça, sur tout match à T0 non nul, aucun kill ne recevait son arme ni
	// son assistance (décalage constant de T0 ms entre les clés — 2026-08-12).
	d.killSources = timeline.CorrectKillSourceRaws(d.killSources, tl)
	d.killAssists = timeline.CorrectKillAssistRaws(d.killAssists, tl)
	// La VICTIME s'apparie par la même clé exacte : copie corrigée, kvPairs intact
	// (tug-of-war et KD timeline restent sur l'horloge brute).
	d.kvPairsFeed = timeline.CorrectKVPairRaws(d.kvPairs, tl)
}

// loadAwardsForScoreboard charge les awards pour tous les xuids du scoreboard
// (chunk MV4.B'). Sérialisé après l'errgroup principal — la liste des xuids
// dépend du scoreboard chargé en parallèle. Dégradation gracieuse :
//
//	awardsRepo nil       -> retourne nil
//	scoreboard vide      -> retourne nil
//	capability absente   -> retourne nil (silencieux)
//	autre erreur         -> log warn + retourne nil
//
// tests utilisent toujours "m1" — la signature reste configurable et le caller prod
// passe le vrai matchID variable.
//
//nolint:unparam // matchID paramétré (caller passe le matchID courant), bien que les
func (s *MatchViewService) loadAwardsForScoreboard(
	ctx context.Context,
	matchID string,
	scoreboard []domain.ScoreboardRaw,
) []port.PersonalScoreAwardRow {
	if s.awardsRepo == nil || len(scoreboard) == 0 {
		return nil
	}
	xuids := extractMatchSquadXUIDs(scoreboard)
	if len(xuids) == 0 {
		return nil
	}
	filters := port.PersonalScoreAwardsFilters{
		MatchIDs: []string{matchID},
		XUIDs:    xuids,
	}
	if err := filters.Validate(); err != nil {
		slog.WarnContext(ctx, "match_view: PersonalScoreAwardsFilters invalides",
			"match_id", matchID, "err", err)
		return nil
	}
	rows, err := s.awardsRepo.LoadPersonalScoreAwards(ctx, s.titleSlug, filters)
	if err != nil {
		if !errors.Is(err, games.ErrCapabilityNotSupported) {
			slog.WarnContext(ctx, "match_view: LoadPersonalScoreAwards echec",
				"match_id", matchID, "err", err)
		}
		return nil
	}
	return rows
}

// matchModeFamilyFromMeta résout la mode family pour le calcul des seuils
// radar (narrative.DefaultThresholds). Best-effort : on inspecte pair_name
// pour identifier slayer / ctf / strongholds / oddball.
//
// Si la pair_name ne match aucun pattern connu, retourne "" (thresholds
// custom neutres).
func matchModeFamilyFromMeta(meta *domain.MatchMetaRaw) string {
	if meta == nil || meta.PairName == nil {
		return ""
	}
	name := strings.ToLower(*meta.PairName)
	switch {
	case strings.Contains(name, "slayer"):
		return "slayer"
	case strings.Contains(name, "ctf") || strings.Contains(name, "capture"):
		return "ctf"
	case strings.Contains(name, "stronghold"):
		return "strongholds"
	case strings.Contains(name, "oddball") || strings.Contains(name, "neutral"):
		return "oddball"
	}
	return ""
}

// pBitMedals est le bit 9 de match_participants.backfill_bits indiquant que les
// médailles ont été fetchées pour ce joueur×match (sync.PBitMedals = 1 << 9).
const pBitMedals = 1 << 9

// detectPartialMatchData inspecte les sources secondaires d'un match et
// retourne la liste des raisons (codes stables) pour lesquelles la vue est
// considérée partielle. Vide si tout est plein.
//
// Codes utilisés (stables — front les mappe à des messages i18n) :
//   - "scoreboard_empty"     → Q12 a renvoyé 0 lignes
//   - "events_empty"         → Q21 a renvoyé 0 highlight events
//   - "player_stats_empty"   → Q17 stats joueur courant absentes (outcome = 0)
//   - "medals_empty"         → Q14 a renvoyé 0 médailles ET le bit PBitMedals
//     n'est pas positionné (médailles jamais fetchées). Si le bit est positionné,
//     0 médaille est un résultat légitime (certains modes n'en attribuent pas).
func detectPartialMatchData(
	stats *domain.PlayerMatchStatsRaw,
	scoreboard []domain.ScoreboardRaw,
	events []domain.EventRaw,
	medals []domain.MedalRaw,
) []string {
	var reasons []string
	if len(scoreboard) == 0 {
		reasons = append(reasons, "scoreboard_empty")
	}
	if len(events) == 0 {
		reasons = append(reasons, "events_empty")
	}
	if stats == nil || stats.OutcomeCode == 0 {
		reasons = append(reasons, "player_stats_empty")
	}
	if len(medals) == 0 {
		medalsFetched := stats != nil && stats.BackfillBits != nil && (*stats.BackfillBits&pBitMedals) != 0
		if !medalsFetched {
			reasons = append(reasons, "medals_empty")
		}
	}
	return reasons
}

// strDeref retourne la valeur d'un *string ou "<nil>" pour les logs structurés.
// Évite les faux-positifs "<nil>" dans slog quand on veut juste tracer le contenu.
func strDeref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

// loadMatchKillSourceClasses charge les kills du joueur courant par source de degat.
//
// Perimetre volontairement etroit : CE match, CE joueur — ce qui satisfait aussi le
// garde-fou anti-scan-complet des filtres. Le chargement lui-meme passe par le foyer
// unique `loadKillSourceClasses` (killsource_load.go), partage avec les agregats.
func (s *MatchViewService) loadMatchKillSourceClasses(
	ctx context.Context, matchID string,
) ([]port.KillSourceClassRow, error) {
	return killsourceload.Load(ctx, s.killSourceRepo, "match view", s.titleSlug,
		[]string{matchID}, []string{s.xuid}), nil
}
