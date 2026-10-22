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
	// dispenseOperateur (D-2 de .ai/V7.5/PLAN_BACKLOG_2026-10-15.md) : cmd/diag_*, cmd/audit_* et les
	// backfills qui réécrivent le match lu lui-même (la Campagne n'y change aucun agrégat).
	// Réservée à cmd/ (décision du superviseur du 2026-10-15) : un lecteur de internal/sync ou
	// internal/ops est corrigé ou classé mono-match / ensemble fourni / sémantique, jamais ici.
	dispenseOperateur = "outil opérateur"
)

var categoriesDispenseCampagne = map[string]bool{
	dispenseMonoMatch: true, dispenseEnsembleFourni: true, dispenseCallSite: true, dispenseSemantique: true,
	dispenseOperateur: true,
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
	"internal/platform/duckdb/csr_coverage_repo.go:CSRCoverageRepo.controlAgainstReference": {dispenseSemantique, "2026-10-14",
		"contrôle croisé référence des playlists classées <-> CSR par match (alerte A3) : la Campagne n'a ni playlist de la référence ni CSR, elle ne contribue à aucun des deux compteurs. Diagnostic de couverture, pas un affichage."},
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
		"liste de BORNAGE des lectures de résumés par match, pas un agrégat d'affichage : elle garde la population exacte des lectures " +
			"qu'elle borne (QKillsBetweenPlayers, kill_events_source.go : frags entre deux joueurs, Explorer et Comparaison ; " +
			"stats de rencontre de la vue match), dont la règle Campagne reste celle de la lecture bornée."},
	"internal/platform/duckdb/coordination_repo.go:QCoordinationFragsOfficiels": {dispenseEnsembleFourni, "2026-10-09",
		"base de « frags appuyés » : WHERE match_id IN (%s) sur les matchs du scope de la page (service/coordination_block.go:286), " +
			"ceux de Sessions et des Séries temporelles (timeseries_service_sections.go:189), et seuls comptent ceux qui portent une " +
			"ligne d'appui (analysis/coordination, aucun film en Campagne)."},
	"internal/platform/duckdb/explorer_repo_resolve.go:qResolveParParticipant": {dispenseSemantique, "2026-10-09",
		"résolution d'un NOM en xuid (ExplorerRepo.ResolveXUIDByGamertag) : même population que la vue des noms v_gamertag_lookup, " +
			"qui lit tous les participants (analysis/identity.go, GamertagLookupViewSQL) ; aucun agrégat de matchs."},

	// ── E2 (plan .ai/V7.5/PLAN_BACKLOG_2026-10-15.md, 2026-10-15) : critère élargi à killer_victim_pairs,
	// match_kill_events(_latest) et medals_earned (D-1), racines élargies à sync, ops et cmd. ──

	// lecteurs de médailles / frags sur une liste de matchs fournie par la page
	"internal/platform/duckdb/explorer_repo.go:ExplorerRepo.GetMedalCountsForMatches": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (liste) : matchs communs de GetCommonMatches (explorer_repo.go:54, Q19 résolu Campagne) via service/explorer_service_target.go:355-368."},
	"internal/platform/duckdb/explorer_repo_medals.go:ExplorerRepo.GetTopMedalsForMatches": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (liste) : matchs récents de GetTargetRecentMatches (explorer_repo.go:292-293, Q19c résolu Campagne) via service/explorer_target_medals.go:108."},
	"internal/platform/duckdb/home_repo_assisted_frags.go:Q26lHomeAssistedFragsTpl": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (lot) : tuiles de l'accueil (service/home_service_enrichment.go:107), lignes de PlayerMatchesRepo exclues à player_matches_repo.go:124."},
	"internal/platform/duckdb/queries_home_citations.go:Q26hMatchMedalsTemplate": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (lot) : mêmes tuiles de l'accueil (home_repo_medals_citations.go:90), exclues à player_matches_repo.go:124."},
	"internal/platform/duckdb/medals_by_xuid_repo.go:buildMedalsByXUIDQuery": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (filtres.MatchIDs) fournis : matchs de tendances (service/trends_service.go:76-94, LoadPlayerMatches exclu) ou matchs partagés de l'escouade (service/squad_service_v2.go:324)."},
	"internal/platform/duckdb/queries_squad.go:Q32dSquadAssistPairsTemplate": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (portée de la page) : allSquadRows de l'Escouade (service/teammates/teammates_squad_assist_pairs.go:64-69, firstBloodScope), lignes déjà exclues."},
	"internal/platform/duckdb/squad_repo_kill_log.go:Q32eSquadKillLogTemplate": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (portée de la page) : même portée de l'Escouade (service/teammates/teammates_squad_impact_thief.go:31)."},
	"internal/platform/duckdb/tactical_repo_coequipiers.go:QTacticalMates": {dispenseEnsembleFourni, "2026-10-15",
		"liste blanche q.Matchs obligatoire (tactical_repo_coequipiers.go:47, exigerLaListe) : matchs du périmètre tactique, dont le jeton est résolu par resolveCampaignExclusion (tactical_repo_univers.go:234)."},
	"internal/platform/duckdb/vehicle_commendation_stats_repo.go:buildHijackMedalSumQuery": {dispenseEnsembleFourni, "2026-10-15",
		"match_id IN (matchIDs) fournis par l'appelant (vehicle_commendation_stats_repo.go:219-246) : matchs de la page Synthèse."},
	"internal/platform/duckdb/queries_career_encounters.go:Q27CareerRivalsTpl": {dispenseEnsembleFourni, "2026-10-15",
		"borné à matchsDeLHistorique (career_repo_rivals.go:106-116), qui résout la Campagne (career_repo_encounters.go:194-195)."},
	"internal/platform/duckdb/queries_match.go:Q14MatchMedals": {dispenseMonoMatch, "2026-10-15",
		"médailles d'UN match (WHERE me.xuid = ? AND me.match_id = ?, queries_match.go:162) : aucun agrégat d'historique."},
	"internal/analysis/identity_annuaire.go:AnnuaireKillFeedLocaliserSQL": {dispenseSemantique, "2026-10-15",
		"localise les matchs où un xuid est NOMMÉ dans le kill-feed pour résoudre un nom (annuaire, squad_repo_annuaire.go:354-358) : même population que la vue des noms, aucun agrégat de matchs (cf. qResolveParParticipant)."},

	// internal/sync : lecteurs d'UN match
	"internal/sync/citations.go:loadMatchStats": {dispenseMonoMatch, "2026-10-15",
		"requête MONO-MATCH (WHERE mp.match_id = ? AND mp.xuid = ?, citations.go:381) : stats d'un match pour ses citations."},
	"internal/sync/citations.go:loadMedalsForMatch": {dispenseMonoMatch, "2026-10-15",
		"requête MONO-MATCH (WHERE match_id = ? AND xuid = ?, citations.go:332)."},
	"internal/sync/citations_weapons_source.go:loadWeaponKillsFromSource": {dispenseMonoMatch, "2026-10-15",
		"requête MONO-MATCH (WHERE k.match_id = ? AND k.feed_killer_xuid = ?, citations_weapons_source.go:87)."},
	"internal/sync/comeback.go:loadMyTeamAndOutcome": {dispenseMonoMatch, "2026-10-15",
		"requête MONO-MATCH (WHERE match_id = ? AND xuid = ?, comeback.go:376) : équipe et issue d'un match."},

	// internal/sync : prédicat classé
	"internal/sync/csr_backfill.go:loadRankedMatchesForCSRBackfill": {dispenseSemantique, "2026-10-15",
		"candidats au rattrapage du CSR par prédicat classé (csr_backfill.go:181-185) : la Campagne n'est jamais classée, Halo 5 dérive is_ranked de la playlist (games/halo_5/mapping.go:102, classifyRanked(HopperId)) que la Campagne n'a pas ; aucun agrégat, le rattrapage écrit le CSR par match."},
	"internal/sync/csr_shared_backfill.go:loadRankedMatchesForSharedCSRBackfill": {dispenseSemantique, "2026-10-15",
		"même prédicat classé (csr_shared_backfill.go:202-206), même preuve (games/halo_5/mapping.go:102)."},

	// internal/sync : sélection de matchs d'un backfill, écritures par match (D-2, outil opérateur)
	"internal/sync/backfill.go:findMatchesInSharedAll": {dispenseEnsembleFourni, "2026-10-15",
		"sélection de travail : matchs à compléter (backfill.go:58), rendus à l'appelant qui les rattrape un à un ; aucune donnée servie ni agrégat."},
	"internal/sync/backfill.go:findMatchesInSharedDB": {dispenseEnsembleFourni, "2026-10-15",
		"sélection de travail : matchs aux participants à compléter (backfill.go:47), rendus à l'appelant ; aucune donnée servie ni agrégat."},
	"internal/sync/engine_backfills.go:loadAllMatchIDsForPlayer": {dispenseEnsembleFourni, "2026-10-15",
		"sélection de travail : matchs sans drapeau de dominance (engine_backfills.go:455-470), rendus au backfill qui le calcule match par match ; aucune donnée servie ni agrégat."},
	"internal/sync/enrichment_discipline.go:computeAndPersistH5DisciplineAwards": {dispenseMonoMatch, "2026-10-15",
		"chaque pénalité (suicide, trahison) ne dépend que des lignes de SON match : GROUP BY match_id (enrichment_discipline.go:45, :55), jointures sur le même match_id (:52-53) ; aucune lecture d'un autre match du joueur."},
	"internal/sync/enrichments.go:queryAnyBotMatchIDs": {dispenseMonoMatch, "2026-10-15",
		"drapeau « coéquipier bot » : auto-jointure sur le même match_id (enrichments.go:118) ; la valeur d'un match ne lit aucun autre match."},
	"internal/sync/enrichments.go:querySignificantBotMatchIDs": {dispenseMonoMatch, "2026-10-15",
		"seuil de présence d'un bot dans UN match : GROUP BY mp_self.match_id (enrichments.go:164), jointures sur le même match_id (:157-160) ; aucun autre match n'entre dans le calcul."},
	"internal/sync/friends_recompute.go:loadMatchesWithFriends": {dispenseMonoMatch, "2026-10-15",
		"is_with_friends d'un match : un ami dans la même équipe de CE match (p2.match_id = p1.match_id, friends_recompute.go:199-202) ; aucun autre match n'entre dans le calcul."},
	"internal/sync/medal_exploit_loader.go:ComputeMedalExploitByMatch": {dispenseMonoMatch, "2026-10-15",
		"score d'exploit calculé match par match : perMatch[matchID] (medal_exploit_loader.go:73, :85) puis score par match ; aucun agrégat inter-matchs."},
	"internal/sync/snapshot/snapshot_readiness_eval.go:loadSnapshotSharedFacts": {dispenseMonoMatch, "2026-10-15",
		"faits d'UN match : sous-requête sur le même match_id (snapshot_readiness_eval.go:209), clé de sortie match_id ; aucun autre match n'entre dans le calcul."},

	// internal/ops
	"internal/ops/archive.go:ArchiveMatches": {dispenseSemantique, "2026-10-15",
		"archivage Parquet de TOUS les matchs antérieurs à la date (archive.go:55-75) : un archivage qui laisserait la Campagne derrière la supprimerait de la lecture sans l'avoir archivée."},
	"internal/ops/archive.go:listArchivableYears": {dispenseSemantique, "2026-10-15",
		"années à archiver (archive.go:140) : même population que l'archivage, Campagne comprise."},
	"internal/ops/archive.go:deleteArchivedParticipants": {dispenseSemantique, "2026-10-15",
		"retire de la base les participants que l'archivage vient d'écrire (archive.go:187) : doit viser la même population, Campagne comprise."},
	"internal/ops/data_quality_examples.go:exampleMatchIDsFor": {dispenseSemantique, "2026-10-15",
		"exemples de match_id d'un inconnu de la qualité des données (data_quality_examples.go:40-70) : le diagnostic doit retrouver un xuid orphelin même en Campagne ; aucun agrégat."},
	"internal/ops/seed_demo.go:sharedTablesWhere": {dispenseEnsembleFourni, "2026-10-15",
		"extraction des tables du corpus démo par matchIDInClause : match_ids du corpus fournis par resolveDemoCorpus (seed_demo.go:421), qui exclut la Campagne de l'ensemble final par dc.excludeCampaign (seed_demo_corpus.go:432), manifeste figé comme sélection dynamique, toutes voies (solo, escouade, classé, média) ; les rejeux figés n'y échappent pas : Halo 5 n'a aucun décodeur de film (capabilities.toml:65)."},
	"internal/ops/seed_demo_anonymize_check.go:loadRealIdentities": {dispenseSemantique, "2026-10-15",
		"rassemble les gamertags RÉELS à rechercher dans la démo (seed_demo_anonymize_check.go:96-125), tables d'alias et de participants entières : une identité de Campagne doit aussi être détectée ; aucun agrégat."},
	"internal/ops/seed_demo_corpus.go:selectRecentRankedMatchIDs": {dispenseSemantique, "2026-10-15",
		"prédicat classé (seed_demo_corpus.go:146-155, is_ranked ou « ranked » dans la playlist) : la Campagne n'est jamais classée, Halo 5 dérive is_ranked de la playlist (games/halo_5/mapping.go:102)."},
	"internal/ops/seed_demo_prestige.go:loadDemoCorpusStats": {dispenseEnsembleFourni, "2026-10-15",
		"agrège le corpus démo déjà extrait (matchIDs fournis, seed_demo_prestige.go:147-160) ; le corpus est celui de resolveDemoCorpus (seed_demo.go:421), Campagne exclue de l'ensemble final, manifeste figé compris, par dc.excludeCampaign (seed_demo_corpus.go:432, E6.2b-C2)."},
	"internal/ops/seed_demo_replay_picks.go:loadDemoReplayCandidates": {dispenseSemantique, "2026-10-15",
		"candidats au rejeu démo : seuls les matchs dont le cache porte un film (seed_demo_replay_picks.go:248) ; Halo 5 n'a aucun décodeur de film (config/titles/halo_5/mappings/capabilities.toml:65), donc aucune ligne de Campagne."},

	// cmd : outils opérateur (D-2) — diag_*, audit_*, backfills et reconstruction de la ligne lue
	"cmd/audit_coverage/main.go:auditPlayer": {dispenseOperateur, "2026-10-15",
		"cmd/audit_* : audit de couverture, lecture seule."},
	"cmd/backfill_all/main.go:loadMissingPSAMatches": {dispenseOperateur, "2026-10-15",
		"backfill : sélectionne les matchs sans PSA, puis les réécrit (main.go:115-129)."},
	"cmd/backfill_kda_accuracy/main.go:main": {dispenseOperateur, "2026-10-15",
		"backfill : UPDATE match_participants SET kda, accuracy sur le match lu (main.go:111)."},
	"cmd/backfill_participation_info/main.go:updateParticipationInfo": {dispenseOperateur, "2026-10-15",
		"backfill : UPDATE ciblé des colonnes de participation du match lu (main.go:220)."},
	"cmd/backfill_quit_timestamps/main.go:updateQuitTimestamps": {dispenseOperateur, "2026-10-15",
		"backfill : UPDATE des horodatages de départ du match lu (main.go:223)."},
	"cmd/backfill_time_played/main.go:commitUpdates": {dispenseOperateur, "2026-10-15",
		"backfill : UPDATE time_played_seconds du match lu (main.go:197)."},
	"cmd/diag_backfill_dryrun/main.go:countMatchParticipants": {dispenseOperateur, "2026-10-15", "cmd/diag_* : essai à blanc d'un backfill, lecture seule."},
	"cmd/diag_backfill_dryrun/main.go:countRankedForPlayer":   {dispenseOperateur, "2026-10-15", "cmd/diag_* : essai à blanc d'un backfill, lecture seule."},
	"cmd/diag_csr/main.go:diagUnrankedDates":                  {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic CSR, lecture seule."},
	"cmd/diag_csr/main.go:diagUnrankedDatesXUID":              {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic CSR, lecture seule."},
	"cmd/diag_expected_kd/main.go:main":                       {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic, lecture seule."},
	"cmd/diag_highlight_match/main.go:main":                   {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic d'UN match, lecture seule."},
	"cmd/diag_kpm/main.go:main":                               {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic, lecture seule."},
	"cmd/diag_lusr_player/loaders.go:loadMatches":             {dispenseOperateur, "2026-10-15", "cmd/diag_* : rejeu diagnostique du LUSR, lecture seule."},
	"cmd/diag_lusr_volatility/main.go:loadTrajectory":         {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic de volatilité, lecture seule."},
	"cmd/diag_lusr_volatility/main.go:printPlayerCoverage":    {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic de couverture, lecture seule."},
	"cmd/diag_medals/main.go:main":                            {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic des médailles, lecture seule."},
	"cmd/diag_orphan_session/main.go:main":                    {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic de sessions orphelines, lecture seule."},
	"cmd/diag_perfsim/load.go:universeSQL":                    {dispenseOperateur, "2026-10-15", "cmd/diag_* : simulation du score de performance, lecture seule."},
	"cmd/diag_recent_match_sync/main.go:inspectMatch":         {dispenseOperateur, "2026-10-15", "cmd/diag_* : inspection d'UN match, lecture seule."},
	"cmd/diag_session_map_winrate/main.go:computeSquad":       {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic, lecture seule."},
	"cmd/diag_session_map_winrate/main.go:findSessionsOnDate": {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic, lecture seule."},
	"cmd/diag_session_map_winrate/main.go:loadHistory":        {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic, lecture seule."},
	"cmd/diag_session_map_winrate/main.go:matchHasAllXUIDs":   {dispenseOperateur, "2026-10-15", "cmd/diag_* : diagnostic, lecture seule."},
	"cmd/h5-roster-refetch/insert.go:insertParticipant": {dispenseOperateur, "2026-10-15",
		"reconstruit les participants manquants du match lu (INSERT, insert.go:66) ; réécrit le match lui-même."},
	"cmd/h5-roster-topup/main.go:insertParticipant": {dispenseOperateur, "2026-10-15",
		"complète les participants manquants du match lu (INSERT, main.go:387) ; réécrit le match lui-même."},
	"cmd/levelup/cmd_backfill_h5_kill_mechanics.go:applyMechanicUpdates": {dispenseOperateur, "2026-10-15",
		"backfill : UPDATE des mécaniques de frag du match lu (cmd_backfill_h5_kill_mechanics.go:376)."},
	"cmd/seed-medal/main.go:main": {dispenseOperateur, "2026-10-15",
		"script de développement (//go:build ignore) qui pose un badge sur le dernier match lu (main.go:1-14) : réécrit le match lui-même."},

	// ── E5.6 (plan backlog 2026-10-16) : lecteurs du registre sur fenêtre de temps, sans `xuid = ?` ──
	"cmd/diag_film_avail/main.go:lireCandidats": {dispenseOperateur, "2026-10-17",
		"cmd/diag_* : mesure, par mois, des matchs sans voie de film (main.go:123, `>= ?` sur le mois), lecture seule : la mesure porte sur le registre entier, Campagne comprise."},
	"cmd/lusr_v2_squad_estimate/main.go:loadWindowMatches": {dispenseOperateur, "2026-10-17",
		"outil d'estimation du LUSR v2 par escouade (cmd/, main.go:135, `start_time >= ?`) : lecture seule, n'écrit rien ; l'estimation porte sur les matchs de tous les joueurs de la fenêtre, pas sur un affichage joueur."},
}
