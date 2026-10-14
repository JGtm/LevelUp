// Package archlint — campaign_exclusion_guard_dispenses_test.go : dispenses du garde
// d'exclusion de la Campagne (campaign_exclusion_guard_test.go, backlog 2026-09-26, B4).
//
// Une dispense = un lecteur des matchs d'un joueur admis SANS exclusion de la Campagne. Elle
// porte une catégorie, une date et la pièce lue (fichier:ligne de l'appelant ou de la
// capability). Le garde échoue sur une dispense qui ne désigne plus un lecteur non exclu.
package archlint

type dispenseCampagne struct {
	categorie string
	date      string
	raison    string
}

const (
	dispenseMonoMatch      = "mono-match"
	dispenseEnsembleFourni = "ensemble fourni par l'appelant"
	dispenseCallSite       = "exclu au call site"
	dispenseSemantique     = "sémantique"
)

var categoriesDispenseCampagne = map[string]bool{
	dispenseMonoMatch: true, dispenseEnsembleFourni: true, dispenseCallSite: true, dispenseSemantique: true,
}

// dispensesExclusionCampagne : clé = chemin.go:Recv.Func ou chemin.go:Nom (depuis apps/go-api).
var dispensesExclusionCampagne = map[string]dispenseCampagne{
	// ── Reprises À L'IDENTIQUE de l'ancien garde de platform/duckdb (item H1, 2026-07-18) ──
	"internal/platform/duckdb/queries_career_encounters.go:Q25MatchParticipants": {dispenseEnsembleFourni, "2026-07-18",
		"sur-lecture inoffensive : les participants sont joints EN GO au set de matchs de Q23 (déjà purgé de la Campagne) ; les lignes de matchs Campagne surnuméraires ne sont jamais restituées."},
	"internal/platform/duckdb/queries_career_encounters.go:QRelationsPlayerWinRateTpl": {dispenseCallSite, "2026-07-18",
		"filtré au CALL SITE via excludeCampaignByMatchID (relations_core_engagement_repo.go, queryPlayerWinRate) — la clause vit hors de la constante."},
	"internal/platform/duckdb/queries_match.go:Q25NeighborMatchesTemplate": {dispenseCallSite, "2026-07-18",
		"filtré au CALL SITE : la clause Campagne est injectée dans /*EXTRA_WHERE*/ via excludeCampaignClause (match_view_repo_neighbors_skill.go)."},
	"internal/platform/duckdb/queries_match.go:Q17PlayerMatchStats": {dispenseMonoMatch, "2026-07-18",
		"requête MONO-MATCH (WHERE match_id = ? AND xuid = ?) : aucune agrégation d'historique → la Campagne ne peut pas polluer un affichage de liste."},
	"internal/platform/duckdb/queries_match.go:Q17bIsParticipant": {dispenseMonoMatch, "2026-07-18",
		"check de participation à UN match (EXISTS WHERE match_id = ? AND xuid = ?) — pas de listing ni d'agrégat."},
	"internal/platform/duckdb/queries_match_detail.go:Q26MatchExpectedStats": {dispenseMonoMatch, "2026-07-18",
		"requête MONO-MATCH (WHERE match_id = ? AND xuid = ?) — pas d'agrégation d'historique."},

	// ── B4.4 : dispenses prévues par le plan ──
	"internal/platform/duckdb/engagement_score_repo_queries.go:EngagementScoreRepo.LoadMatchEngagementContext": {dispenseMonoMatch, "2026-09-27",
		"requête MONO-MATCH (engagement_score_repo_queries.go:53, WHERE mr.match_id = ? AND mp.xuid = ?) : contexte d'UN match, aucun agrégat."},
	"internal/platform/duckdb/csr_coverage_repo.go:CSRCoverageRepo.countRankedMatchesInRegistry": {dispenseSemantique, "2026-09-27",
		"prédicat classé (csr_coverage_repo.go:109-111) : la Campagne n'est jamais classée — Halo 5 dérive is_ranked de la playlist " +
			"(games/halo_5/mapping.go:102, classifyRanked(HopperId)), que la Campagne n'a pas. Diagnostic de couverture CSR, pas un affichage."},
	"internal/platform/duckdb/media_repo_filters.go:MediaRepo.LoadMatchCandidatesForMedia": {dispenseSemantique, "2026-09-27",
		"appariement clip → match par fenêtre de capture (media_repo_filters.go:255) : un clip tourné en Campagne doit pouvoir se rattacher à son match."},
	"internal/api/wire/registry_monitoring_freshness.go:ServiceRegistry.lastMatchByXUID": {dispenseSemantique, "2026-09-27",
		"fraîcheur admin, dernier match persisté par joueur (registry_monitoring_freshness.go:173) : la Campagne est une activité réelle du compte."},
	"internal/analysis/match_filter.go:BuildNeighborsWhereClause": {dispenseCallSite, "2026-09-27",
		"faux positif : fragment WithPlayerXuid (match_filter.go:196) injecté dans /*EXTRA_WHERE*/ avec excludeCampaignClause " +
			"(platform/duckdb/match_view_repo_neighbors_skill.go:72)."},

	// ── B4.5 : autres entrants révélés par le balayage ──
	"internal/platform/duckdb/player_matches_repo.go:playerMatchesSharedBaseSelect": {dispenseCallSite, "2026-09-27",
		"SELECT de base de PlayerMatchesRepo : le seul appelant buildSharedQuery pose analysis.SQLExcludeCampaignVariants (player_matches_repo.go:124)."},
	"internal/platform/duckdb/player_matches_repo.go:appendPlayerMatchSetFilters": {dispenseSemantique, "2026-09-27",
		"anti-filtre ExcludeFriendsXUIDs (player_matches_repo.go:207) : l'ensemble des matchs à RETIRER ; le seul appelant buildSharedQuery " +
			"exclut la Campagne de la lecture (player_matches_repo.go:124)."},
	"internal/platform/duckdb/queries_squad.go:Q32bMainTeamParticipantsTemplate": {dispenseEnsembleFourni, "2026-09-27",
		"WHERE p.match_id IN (%s) : matchs fournis par l'appelant (squad_repo_synthesis.go:29) — accueil : lignes de PlayerMatchesRepo " +
			"(service/home_service.go:259, exclues à player_matches_repo.go:124) ; Escouade : matchs de la page (service/teammates/teammates_service_intersect.go:328)."},
	"internal/platform/duckdb/queries_squad.go:Q42MapStatsSquadExtraExclusionFrag": {dispenseCallSite, "2026-09-27",
		"anti-jointure ajoutée à Q42MapStatsForSquadSharedTpl (squad_repo_mapstats.go:105), dont le jeton est résolu (squad_repo_mapstats.go:95)."},
	"internal/platform/duckdb/relation_assists_repo.go:buildRelationAssistsQuery": {dispenseSemantique, "2026-09-27",
		"Q28c ne lit que les matchs mesurés au film (relation_assists_repo.go:74, assist_known) ; hors film, assist_known vaut FAUX " +
			"(migration/steps_shared_kill_events_credit_base.go:253) et Halo 5 n'a pas de film (config/titles/halo_5/mappings/capabilities.toml:65) : " +
			"aucune ligne de Campagne. partnerClause (:153) = joueurs d'UN match."},
	"internal/platform/duckdb/replay_facts_repo.go:ReplayFactsRepo.LinkTargetsForMatches": {dispenseEnsembleFourni, "2026-09-27",
		"lien vers la page de rejeu des matchs dont le film vient d'être cuit (api/wire/registry_replay_notify.go:208, b.MatchIDs) ; aucun agrégat."},
	"internal/platform/duckdb/tactical_repo_univers.go:clauseCoequipier": {dispenseCallSite, "2026-09-27",
		"fragment ajouté par clausePerimetre à QTacticalUnivers / QTacticalMaps, dont le jeton est résolu (tactical_repo_univers.go:215, tactical_repo.go:118)."},

	// ── Explorer, onglet « Joueur » (ADR 0036, 2026-10-09) ──
	"internal/platform/duckdb/perimetre_liste.go:QMatchsOuJoue": {dispenseSemantique, "2026-10-09",
		"liste de BORNAGE sous les fenêtres `_latest` du kill-feed, pas un agrégat d'affichage : elle garde la population exacte des lectures " +
			"qu'elle borne (QKillsBetweenPlayersBorne, kill_events_source.go ; Q28c, relation_assists_repo.go), qui ne filtrent pas la Campagne."},
	"internal/platform/duckdb/coordination_repo.go:QCoordinationFragsOfficiels": {dispenseEnsembleFourni, "2026-10-09",
		"base de « frags appuyés » : WHERE match_id IN (%s) sur les matchs du scope de la page (service/coordination_block.go:286), " +
			"ceux de Sessions et des Séries temporelles (timeseries_service_sections.go:189), et seuls comptent ceux qui portent une " +
			"ligne d'appui (analysis/coordination, aucun film en Campagne)."},
	"internal/platform/duckdb/explorer_repo_resolve.go:qResolveParParticipant": {dispenseSemantique, "2026-10-09",
		"résolution d'un NOM en xuid (ExplorerRepo.ResolveXUIDByGamertag) : même population que la vue des noms v_gamertag_lookup, " +
			"qui lit tous les participants (analysis/identity.go, GamertagLookupViewSQL) ; aucun agrégat de matchs."},
}
