// Package duckdb â€” queries_home_citations.go : requÃªtes page Home, citations et mÃ©dias.
package duckdb

// Q26 — Home : matchs d'un joueur avec KPIs pour le hero card.
//
// Phase 3.bis plan stabilisation 2026-05-22 : split en 2 queries Go-side pour
// éviter le mix cross-DB (player + shared) qui forçait l'ATTACH shared sur la
// player conn — pattern interdit depuis ADR 0016 (cf. AUDIT §3 + §5).
//
//   - Q26HomeMatchesSharedPart : tables shared uniquement (match_participants,
//     match_registry, medals_earned via CTE perfect). Exécutée via SharedReader.
//     Tri chronologique + LIMIT 150 = source de vérité de la liste.
//   - Q26HomeMatchesPlayerEnrichTpl : tables player uniquement
//     (player_match_enrichment, match_skill_rank), filtrée par match_id IN (%s).
//     Exécutée sur pdb.Player.
//
// Le merge Go-side reconstruit la HomeMatchRow complète. Cf. LoadHomeMatches.
//
// Paramètres :
//   - Q26HomeMatchesSharedPart : ?1 = xuid (CTE perfect), ?2 = xuid (WHERE mp.xuid)
//   - Q26HomeMatchesPlayerEnrichTpl : pas de paramètre, juste IN (%s) match_ids
//
// Le token /*__PERFECT_KILL_IN__*/ est résolu au runtime vers le set de médailles
// « frag parfait » du titre du joueur (perfectKillMedalInClause ; HINF = {1512363953}).
var Q26HomeMatchesSharedPart = `
WITH base AS (
    -- J7 : fenêtre des 150 matchs affichés, calculée UNE fois. perfect ET la
    -- requête principale s'y bornent → même ensemble de matchs (aucun risque de
    -- divergence sur les ex-aequo de start_time) et perfect n'agrège plus tout
    -- l'historique de médailles mais seulement ces 150 matchs. Résultat identique.
    SELECT mp.match_id
    FROM match_participants mp
    JOIN match_registry r ON r.match_id = mp.match_id
    WHERE mp.xuid = ?
    ORDER BY ` + StartTimeCanonicalSQL("r") + ` DESC
    LIMIT 150
),
perfect AS (
    SELECT match_id, COALESCE(SUM(count), 0) AS perfect_kills
    FROM medals_earned
    WHERE xuid = ? AND match_id IN (SELECT match_id FROM base) AND /*__PERFECT_KILL_IN__*/
    GROUP BY match_id
)
SELECT
    mp.match_id,
    ` + StartTimeCanonicalSQL("r") + ` AS start_time,
    COALESCE(r.map_id, '')                                  AS map_id,
    COALESCE(r.map_name, '')                                AS map_name,
    COALESCE(r.map_name_fr, r.map_name, '')                 AS map_name_fr,
    COALESCE(r.pair_id, '')                                 AS pair_id,
    COALESCE(r.pair_name, '')                               AS pair_name,
    COALESCE(r.pair_name_fr, r.pair_name, '')               AS pair_name_fr,
    COALESCE(r.game_variant_id, '')                         AS game_variant_id,
    COALESCE(r.game_variant_name, '')                       AS game_variant_name,
    COALESCE(r.playlist_id, '')                             AS playlist_id,
    COALESCE(r.playlist_name, '')                           AS playlist_name,
    COALESCE(r.playlist_name_fr, r.playlist_name, '')       AS playlist_name_fr,
    COALESCE(r.is_firefight, FALSE)                         AS is_firefight,
    CASE
        WHEN COALESCE(r.is_ranked, FALSE)
            OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
            OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
        THEN TRUE
        ELSE FALSE
    END                                                     AS is_ranked,
    COALESCE(mp.outcome, 0)                                 AS outcome,
    COALESCE(mp.team_id, -1)                                AS team_id,
    COALESCE(r.team_0_score, -1)                            AS team_0_score,
    COALESCE(r.team_1_score, -1)                            AS team_1_score,
    COALESCE(mp.kills, 0)                                   AS kills,
    COALESCE(mp.deaths, 0)                                  AS deaths,
    COALESCE(mp.assists, 0)                                 AS assists,
    mp.kda,
    CASE WHEN COALESCE(mp.deaths, 0) > 0
         THEN CAST(COALESCE(mp.kills, 0) AS DOUBLE) / CAST(mp.deaths AS DOUBLE)
         ELSE CAST(COALESCE(mp.kills, 0) AS DOUBLE) END     AS ratio,
    mp.accuracy,
    mp.avg_life_seconds,
    NULLIF(COALESCE(NULLIF(mp.time_played_seconds, 0), r.playable_duration_seconds), 0) AS time_played_seconds,
    mp.damage_dealt,
    mp.damage_taken,
    mp.team_mmr,
    mp.enemy_mmr,
    mp.rank                                                 AS rank_in_team,
    COALESCE(mp.headshot_kills, 0)                          AS headshot_kills,
    COALESCE(perfect.perfect_kills, 0)                      AS perfect_kills,
    mp.max_killing_spree                                    AS max_killing_spree
FROM match_participants mp
JOIN match_registry r ON r.match_id = mp.match_id
LEFT JOIN perfect ON perfect.match_id = mp.match_id
WHERE mp.xuid = ? AND mp.match_id IN (SELECT match_id FROM base)
ORDER BY ` + StartTimeCanonicalSQL("r") + ` DESC
LIMIT 150`

// Q26HomeMatchesPlayerEnrichTpl : enrichissement player (pme + msr) pour un
// lot de match_ids. À utiliser via fmt.Sprintf(tpl, strings.Join(placeholders, ", ")).
const Q26HomeMatchesPlayerEnrichTpl = `
SELECT
    pme.match_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends,
    COALESCE(pme.dominance_flag, 0)      AS dominance_flag,
    pme.performance_score,
    msr.rating_type,
    msr.rating_value,
    msr.tier,
    COALESCE(msr.sub_tier, 0)            AS sub_tier,
    msr.tier_label,
    msr.rating_delta,
    msr.playlist_group
FROM player_match_enrichment_latest pme
LEFT JOIN match_skill_rank_latest msr ON msr.match_id = pme.match_id
WHERE pme.match_id IN (%s)`

// Q27HomeSessionsPlayerPart : Phase A de Q27 — sessions du joueur depuis
// player_match_enrichment uniquement. Le start_time est récupéré via une
// 2e query SharedReader (cf. Q27HomeSessionsSharedStartTimes).
//
// Phase 3.bis plan stabilisation 2026-05-22 : split de Q27HomeSessions
// (qui mixait player + shared) en 2 phases Go-side. Pattern aligné sur Q26.
const Q27HomeSessionsPlayerPart = `
SELECT
    pme.match_id,
    pme.session_id,
    pme.session_label,
    COALESCE(pme.is_with_friends, FALSE) AS is_with_friends
FROM player_match_enrichment_latest pme
WHERE pme.session_label IS NOT NULL`

// Q27HomeSessionsSharedStartTimesTpl : Phase B de Q27 — start_time pour
// un lot de match_ids depuis match_registry (shared).
var Q27HomeSessionsSharedStartTimesTpl = `
SELECT
    match_id,
    ` + StartTimeCanonicalSQL("") + ` AS start_time
FROM match_registry
WHERE match_id IN (%s)`

// Q26h : Home â€” mÃ©dailles par match pour un joueur, lots de match_id.
// ParamÃ¨tres : ?1 = xuid. Les match_id sont injectÃ©s dynamiquement via IN (%s).
// RequÃªte sur pdb.Player (shared attachÃ©) ; labels rÃ©solus ensuite via metadata.
const Q26hMatchMedalsTemplate = `
SELECT
    me.match_id,
    me.medal_name_id,
    COALESCE(me.count, 1) AS count
FROM medals_earned me
WHERE me.xuid = ?
  AND me.match_id IN (%s)
ORDER BY me.match_id, me.count DESC`

// Q26i : Home â€” citations progressÃ©es par match, avec cumul global au moment du match.
// Les match_id sont injectÃ©s dynamiquement via IN (%s).
// RequÃªte sur pdb.Player uniquement (match_citations est dans stats.duckdb).
const Q26iMatchCitationsTemplate = `
SELECT
    mc.match_id,
    mc.citation_name_norm,
    mc.value                AS match_delta,
    cum.total               AS cumulative_total
FROM match_citations_latest mc
JOIN (
    SELECT citation_name_norm, SUM(value) AS total
    FROM match_citations_latest
    GROUP BY citation_name_norm
) cum ON cum.citation_name_norm = mc.citation_name_norm
WHERE mc.match_id IN (%s)
  AND mc.value > 0
ORDER BY mc.match_id, mc.value DESC`

// Q26j : Home â€” mÃ©tadonnÃ©es citations depuis metadata.duckdb pour un ensemble de norms.
// Les citation_name_norm sont injectÃ©s dynamiquement via IN (%s).
// GROUP BY car une citation peut avoir plusieurs medal_id rows.
//
// GH2-B2/B6 : citation_name_display_en exposé pour la résolution locale-aware du
// nom (les citations Infinite sont des copies de commendations H5, seul le calcul
// diffère → l'EN vient du seed). Sous UI EN, l'appelant remplace le display FR par
// l'EN quand il est non vide.
// GH4 : description_en exposé (source = commendations H5 officielles + trad fidèle
// Infinite). Sous UI EN, l'appelant sert description_en quand non vide, sinon le nom
// seul (description masquée) — jamais le FR (principe GH-5b).
const Q26jCitationMappingsForNormsTemplate = `
SELECT
    citation_name_norm,
    citation_name_display,
    COALESCE(citation_name_display_en, '') AS citation_name_display_en,
    COALESCE(image_path, '')   AS image_path,
    COALESCE(tier_targets, '') AS tier_targets,
    COALESCE(MAX(description), '') AS description,
    COALESCE(MAX(description_en), '') AS description_en
FROM citation_mappings
WHERE citation_name_norm IN (%s)
  AND enabled IS NOT FALSE
GROUP BY citation_name_norm, citation_name_display, citation_name_display_en, image_path, tier_targets`

// Q26b : Home -- nombre total de matchs d un joueur (pas de LIMIT).
// Parametre : ?1 = xuid du joueur.
const Q26bCountPlayerMatches = `
SELECT COUNT(*) FROM match_participants WHERE xuid = ?`

// Q26c : Home -- identitÃ© record compacte depuis career_progression.
// Un seul scan via ARG_MAX â€” remplace les 5 sous-requÃªtes corrÃ©lÃ©es de l'ancienne version.
//
// FILTRE PAR XUID (?1) : career_progression est dans la player DB mais peut, en cas
// de contamination de sync historique, contenir des rows d'un AUTRE joueur. Sans ce
// filtre, ARG_MAX remontait l'identité (emblème/bannière/rang) du mauvais xuid si sa
// row était plus récente — symptôme observé (Chocoboflor affichait l'identité de JGtm).
// Concaténer xuid à une chaîne vide défait le pushdown sur l'index PK (cf.
// career_live_repo.go : index DuckDB connu corrompu, table-scan complet, < 1k rows/joueur).
// ParamÃ¨tre : ?1 = xuid du joueur.
// BANNIÈRE/EMBLÈME/BACKDROP : champs d'apparence INDÉPENDANTS (directive
// produit 2026-07-08) — chacun sert sa dernière valeur non vide (« jamais
// vide »). Pas de couplage bannière↔emblème : un emblème sans nameplate
// upstream (nouvelle génération `<id>-SpartanEmblem`) laisse la dernière
// bannière connue servie.
const Q26cHomeSpartanIdentity = `
SELECT
    ARG_MAX(rank,             recorded_at) FILTER (WHERE rank IS NOT NULL)                                  AS rank,
    COALESCE(ARG_MAX(current_xp,      recorded_at) FILTER (WHERE rank IS NOT NULL), 0)                     AS current_xp,
    COALESCE(ARG_MAX(xp_for_next_rank, recorded_at) FILTER (WHERE rank IS NOT NULL), 0)                    AS xp_for_next_rank,
    COALESCE(ARG_MAX(is_max_rank,     recorded_at) FILTER (WHERE rank IS NOT NULL), FALSE)                  AS is_max_rank,
    ARG_MAX(spartan_id,       recorded_at) FILTER (WHERE NULLIF(TRIM(spartan_id),       '') IS NOT NULL)    AS spartan_id,
    NULLIF(TRIM(ARG_MAX(rank_name,    recorded_at) FILTER (WHERE rank IS NOT NULL)), '')                    AS rank_name,
    NULLIF(TRIM(ARG_MAX(rank_tier,    recorded_at) FILTER (WHERE rank IS NOT NULL)), '')                    AS rank_tier,
    ARG_MAX(banner_image_url,  recorded_at) FILTER (WHERE NULLIF(TRIM(banner_image_url),  '') IS NOT NULL)  AS banner_image_url,
    ARG_MAX(emblem_image_url,  recorded_at) FILTER (WHERE NULLIF(TRIM(emblem_image_url),  '') IS NOT NULL)  AS emblem_image_url,
    ARG_MAX(backdrop_image_url, recorded_at) FILTER (WHERE NULLIF(TRIM(backdrop_image_url),'') IS NOT NULL) AS backdrop_image_url,
    ARG_MAX(adornment_path,   recorded_at) FILTER (WHERE NULLIF(TRIM(adornment_path),   '') IS NOT NULL)    AS adornment_path
FROM career_progression
WHERE xuid || '' = ?`

// Q26d : Home -- assets visuels du rang carriÃ¨re courant depuis metadata.duckdb.
//
// Les libellÃ©s (title FR/EN, next_rank_title) ne sont PAS lus ici : ils
// proviennent du TitleSemanticAdapter (career_rank_translations) cÃ´tÃ© service.
// Le repo storage reste exclusivement responsable des paths d'assets.
//
// ParamÃ¨tre : ?1 = rank_id.
const Q26dHomeCareerRankMeta = `
SELECT
	COALESCE(
		NULLIF(TRIM(large_icon_path), ''),
		NULLIF(TRIM(icon_path), '')
	) AS image_path,
	NULLIF(TRIM(adornment_icon_path), '') AS adornment_path
FROM career_ranks
WHERE rank_id = ?
LIMIT 1`

// Q26e : Home -- meilleur rating historique par type (CSR ou LUSR), avec
// statut de placement par playlist_group.
//
// ParamÃ¨tre : ?1 = rating_type ('CSR' ou 'LUSR', case-insensitive).
//
// Retourne 1 ligne max :
//
//	rating_value, tier_label, tier, sub_tier, placement_remaining
//
// Logique placement (LUSR principalement, CSR secondaire en fallback) :
//   - chaque playlist_group a sa propre phase de placement de 10 matchs
//   - match_count = COUNT(*) des rows par playlist_group pour le type donné
//   - placement_remaining = GREATEST(0, 10 - match_count)
//   - on prÃ©fÃ¨re le meilleur rating d'un groupe matured (match_count >= 10) ;
//     sinon on retourne le meilleur rating du groupe le plus jouÃ© en placement,
//     pour que le badge unranked_(10-remaining).png puisse Ãªtre construit.
//
// **NULL handling — fix bug prod 2026-05-20** : `playlist_group` peut être NULL
// pour les anciennes rows LUSR/CSR (avant introduction de la colonne ou pour
// les CSR fetchés sans group). Sans COALESCE en sentinel, le `JOIN ... ON
// gc.playlist_group = bpg.playlist_group` éliminerait TOUTES les rows à
// playlist_group NULL (sémantique SQL `NULL = NULL` → NULL → exclu de l'inner
// JOIN). C'est exactement le bug observé pour JGtm : 758 rows LUSR existaient
// mais highest_lusr ressortait null. On normalise via COALESCE(..., '_unknown').
//
// Le CASE de classification CSR/LUSR garde le fallback heuristique sur
// playlist_name/pair_name (régression historique is_ranked=FALSE non corrigée).
// Q26ePeakPhaseAPlayer : Phase A (player-only) — rating courant par match.
// Sprint P7 / ADR 0016 (2026-05-20) : exécutée via pdb.Player, sans
// shared. Classification CSR/LUSR faite côté Go après Phase B.
//
// LIT match_skill_rank_LATEST (jamais la table brute) : le substrat est
// append-only (ADR 0026) et contient des rows périmées + des rows d'audit
// LUSR_V2. Lire le brut faisait remonter un pic faux (ex. Onyx fantôme sur un
// joueur Or : valeur LUSR_V2/périmée masquée par la vue). La vue applique la
// priorité CSR > LUSR > LUSR_V2 + most-recent par match_id.
const Q26ePeakPhaseAPlayer = `
SELECT
	msr.match_id,
	msr.playlist_group,
	msr.rating_value,
	msr.rating_type,
	msr.tier,
	msr.sub_tier,
	msr.tier_label,
	COALESCE(msr.updated_at, msr.start_time, msr.created_at) AS recency
FROM match_skill_rank_latest msr
WHERE msr.rating_value IS NOT NULL`

// Q26ePeakPhaseBRegistryTpl : Phase B (shared-only) — registry pour
// classification. Exécutée via pdb.SharedReadDB().Get() — pas de préfixe.
const Q26ePeakPhaseBRegistryTpl = `
SELECT
	mr.match_id,
	COALESCE(mr.is_ranked, FALSE)  AS is_ranked,
	COALESCE(mr.playlist_name, '') AS playlist_name,
	COALESCE(mr.pair_name, '')     AS pair_name
FROM match_registry mr
WHERE mr.match_id IN (%s)`

// Q26g : Home â€” 3 derniÃ¨res playlists distinctes jouÃ©es avec leur dernier rang compÃ©titif.
// ParamÃ¨tre : ?1 = xuid du joueur.
// Retourne (playlist_id, playlist_name, is_ranked, rating_type, rating_value, tier, tier_fr,
//
//	sub_tier, tier_label, measurement_matches_remaining).
//
// playlist_name_fr est rÃ©solu en Go depuis asset_translations (mÃªme source que les tuiles de matchs).
// rating_* sont NULL pour les playlists sans rang calculÃ©.
// measurement_matches_remaining vient de player_csr_snapshots (snapshot le plus rÃ©cent par playlist)
// pour permettre d'Ã©mettre `unranked_N.png` pendant la phase de placement (10 â†’ 0 matchs restants).
// Q26gPlaylistPhaseBShared : Phase B (shared) — top 3 playlists pour xuid
// avec le dernier match_id par playlist. Sprint P7 / ADR 0016 : sans préfixe
// shared. (exécuté via pdb.SharedReadDB().Get()).
//
// Fallback playlist_name : beaucoup de matchs ont playlist_id = NULL dans
// match_registry (Social, anciens matchs non backfillés). On groupe sur
// COALESCE(playlist_id, playlist_name) pour capturer ces cas et retourner
// jusqu'à 3 playlists distinctes même sans playlist_id renseigné.
var Q26gPlaylistPhaseBShared = `
WITH scoped AS (
	SELECT
		COALESCE(NULLIF(TRIM(r.playlist_id), ''), NULLIF(TRIM(r.playlist_name), '')) AS group_key,
		NULLIF(TRIM(r.playlist_id), '') AS playlist_id,
		r.playlist_name AS playlist_name,
		CASE
			WHEN COALESCE(r.is_ranked, FALSE)
				OR STRPOS(LOWER(COALESCE(r.playlist_name, '')), 'ranked') > 0
				OR STRPOS(LOWER(COALESCE(r.pair_name, '')), 'ranked') > 0
			THEN 1 ELSE 0
		END AS is_ranked_flag,
		r.match_id AS match_id,
		r.season_id AS season_id,
		` + StartTimeCanonicalSQL("r") + ` AS played_at,
		ROW_NUMBER() OVER (
			PARTITION BY COALESCE(NULLIF(TRIM(r.playlist_id), ''), NULLIF(TRIM(r.playlist_name), ''))
			ORDER BY ` + StartTimeCanonicalSQL("r") + ` DESC
		) AS rn_recent
	FROM match_participants mp
	JOIN match_registry r ON r.match_id = mp.match_id
	WHERE mp.xuid = ?
	  AND COALESCE(NULLIF(TRIM(r.playlist_id), ''), NULLIF(TRIM(r.playlist_name), '')) IS NOT NULL
),
recent AS (
	SELECT * FROM scoped WHERE rn_recent <= 30
),
per_playlist AS (
	SELECT
		group_key,
		COALESCE(MAX(playlist_id), '') AS playlist_id,
		COALESCE(MAX(playlist_name), '') AS playlist_name,
		MAX(is_ranked_flag) > 0 AS is_ranked,
		MAX(played_at) AS last_played,
		ARG_MAX(match_id, played_at) AS last_match_id,
		ARG_MAX(season_id, played_at) AS last_season_id,
		-- Liste ordonnée (plus récent → plus ancien) des match_ids récents de la
		-- playlist : fix S5, permet à la Phase A1/Go de retomber sur le dernier
		-- match QUI a une ligne MSR exploitable si le tout dernier n'en a pas.
		STRING_AGG(match_id, ',' ORDER BY played_at DESC) AS recent_match_ids
	FROM recent
	GROUP BY group_key
)
SELECT playlist_id, playlist_name, is_ranked, last_played, last_match_id, last_season_id, recent_match_ids
FROM per_playlist
ORDER BY last_played DESC
LIMIT 3`

// Q26gPlaylistPhaseAMSRTpl : Phase A1 (player) — rating + tier des last_match_id.
//
// Robustesse multi-titre (fix S5) : un même match_id porte souvent plusieurs
// lignes (Infinite : 1 CSR réel ; Halo 5 : CSR=0 placeholder + LUSR + LUSR_V2,
// car la sync H5 écrit rating_value=0 pour respecter le NOT NULL). Deux gardes :
//  1. on ignore les lignes `CSR AND rating_value=0` (placeholder inexploitable
//     qui, sinon, écrasait la vraie valeur LUSR dans le map Go keyé par match_id) ;
//  2. on ne garde qu'UNE ligne par match_id via ROW_NUMBER, priorité
//     CSR(réel) > LUSR > LUSR_V2 — même ordre que la vue match_skill_rank_latest,
//     mais débarrassé du placeholder CSR=0. Title-agnostique : sur Infinite,
//     la seule ligne est le CSR réel → no-op.
const Q26gPlaylistPhaseAMSRTpl = `
WITH exploitable AS (
	SELECT
		match_id,
		rating_type,
		rating_value,
		NULLIF(TRIM(tier), '')        AS tier,
		NULLIF(TRIM(tier_fr), '')     AS tier_fr,
		COALESCE(sub_tier, 0)         AS sub_tier,
		NULLIF(TRIM(tier_label), '')  AS tier_label,
		ROW_NUMBER() OVER (
			PARTITION BY match_id
			ORDER BY CASE UPPER(COALESCE(rating_type, ''))
				WHEN 'CSR' THEN 0
				WHEN 'LUSR' THEN 1
				WHEN 'LUSR_V2' THEN 2
				ELSE 3
			END, written_at DESC, id DESC
		) AS rn
	-- Lecture BRUTE volontaire (allowlist B8) : le filtre H5 NOT (CSR AND
	-- rating_value=0) (placeholder Halo 5) doit s'appliquer AVANT le choix de la
	-- ligne ; la vue match_skill_rank_latest a deja tranche (CSR>LUSR) et
	-- masquerait la vraie LUSR. Le tie-break written_at/id rend ce latest manuel
	-- deterministe (anti-ADR-0026).
	FROM match_skill_rank
	WHERE match_id IN (%s)
	  AND rating_value IS NOT NULL
	  AND NOT (UPPER(COALESCE(rating_type, '')) = 'CSR' AND rating_value = 0)
)
SELECT match_id, rating_value, tier, tier_fr, sub_tier, tier_label
FROM exploitable
WHERE rn = 1`

// Q26gPlaylistPhaseASnapshotTpl : Phase A2 (player) — placement remaining
// par playlist_id depuis player_csr_snapshots (snapshot le plus récent).
const Q26gPlaylistPhaseASnapshotTpl = `
WITH ranked AS (
	SELECT
		playlist_id,
		current_measurement_remaining,
		ROW_NUMBER() OVER (
			PARTITION BY playlist_id
			ORDER BY fetched_at DESC, season_id DESC
		) AS rn
	FROM player_csr_snapshots_latest
	WHERE playlist_id IN (%s)
)
SELECT playlist_id, current_measurement_remaining
FROM ranked
WHERE rn = 1`

// Q27HomeSessions : DÉPRÉCIÉ — split en 2 phases via
// Q27HomeSessionsPlayerPart + Q27HomeSessionsSharedStartTimesTpl
// (Phase 3.bis plan stabilisation 2026-05-22). Conservé en commentaire
// comme référence historique. Code mort retiré.

// Q28 : Home â€” medias recents depuis media_files + media_match_associations.
// Parametre : ?1 = LIMIT (nombre de medias).
// Retourne uniquement les medias actifs, triés par date de modification desc.
//
// Phase 3 plan stabilisation 2026-05-22 : migré vers pdb.SharedSocial. Le
// schéma shared_social.media_files + media_match_associations diffère de
// l'ancien player.media_files :
//   - JOIN via `mma.media_file_id = mf.id` (au lieu de file_path/media_path)
//   - match_start_time absent → retourné NULL (consumer doit dériver depuis
//     match_registry si besoin via une autre query Go-side)
const Q28RecentMedia = `
SELECT
    mf.file_name,
    mma.match_id,
    NULL::TIMESTAMP AS match_start_time
FROM media_files mf
LEFT JOIN media_match_associations_latest mma ON mma.media_file_id = mf.id
WHERE mf.status = 'active'
ORDER BY mf.mtime DESC
LIMIT ?`

// Q26k : Home â€” arme favorite (kills totaux) du joueur toutes armes confondues.
// ParamÃ¨tre : ?1 = xuid.
// RequÃªte sur pdb.Player (shared attachÃ©). Label rÃ©solu ensuite via pdb.Metadata.
const Q26kFavoriteWeapon = `
SELECT
    wk.effective_weapon_id AS weapon_id,
    COUNT(*)               AS total_kills
FROM v_weapon_kills wk
WHERE wk.xuid = ?
  AND wk.effective_weapon_id NOT IN (0, 1, 2)
GROUP BY wk.effective_weapon_id
ORDER BY total_kills DESC
LIMIT 1`

// =============================================================================
// Sprint 13 â€” Citations + MÃ©dias
// =============================================================================

// Q34 : Citations â€” mappings de citation depuis metadata.duckdb.
// ParamÃ¨tre : aucun. RequÃªte sur pdb.Metadata (pas pdb.Player).
