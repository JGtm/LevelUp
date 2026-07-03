package duckdb

const Q26CareerTopEncountersTpl = `
WITH my_history AS (
    SELECT match_id, team_id, outcome
    FROM match_participants
    WHERE xuid = ?
),
encounters AS (
    SELECT
        p.xuid,
        p.match_id,
        p.team_id  AS opp_team_id,
        h.team_id  AS my_team_id,
        h.outcome  AS my_outcome,
        COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time
    FROM my_history h
    JOIN match_participants p
        ON p.match_id = h.match_id
       AND p.xuid <> ?
       AND p.xuid NOT LIKE 'bid(%%'
    LEFT JOIN match_registry r ON r.match_id = p.match_id
),
encounter_stats AS (
    SELECT
        e.xuid,
        COUNT(DISTINCT e.match_id) AS count_together,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id THEN e.match_id END) AS ally_count,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id THEN e.match_id END) AS enemy_count,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id AND %s THEN e.match_id END) AS wins_as_ally,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id AND %s THEN e.match_id END) AS losses_as_ally,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id AND %s THEN e.match_id END) AS wins_vs_enemy,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id AND %s THEN e.match_id END) AS losses_vs_enemy,
        MAX(e.start_time) AS last_seen_at
    FROM encounters e
    GROUP BY e.xuid
),
kv_stats AS (
    SELECT
        opp_xuid AS xuid,
        SUM(kills_by_me)   AS kills_dealt,
        SUM(kills_by_them) AS deaths_suffered
    FROM (
        SELECT
            CASE WHEN kv.killer_xuid = ? THEN kv.victim_xuid ELSE kv.killer_xuid END AS opp_xuid,
            CASE WHEN kv.killer_xuid = ? THEN kv.kill_count   ELSE 0               END AS kills_by_me,
            CASE WHEN kv.victim_xuid = ? THEN kv.kill_count   ELSE 0               END AS kills_by_them
        FROM killer_victim_pairs kv
        WHERE kv.killer_xuid = ? OR kv.victim_xuid = ?
    ) t
    GROUP BY opp_xuid
)
SELECT
    es.xuid,
    -- es.xuid (encounter_stats) peut être orphelin de la vue → fallback masqué
    -- "Joueur ####" (jamais de xuid brut, miroir de analysis.MaskedXuidLabelSQL).
    COALESCE(vg.gamertag, ('Joueur ' || RIGHT(es.xuid, 4))) AS gamertag,
    es.count_together,
    es.ally_count,
    es.enemy_count,
    es.wins_as_ally,
    es.losses_as_ally,
    es.wins_vs_enemy,
    es.losses_vs_enemy,
    COALESCE(kv.kills_dealt, 0)    AS kills_dealt,
    COALESCE(kv.deaths_suffered,0) AS deaths_suffered,
    es.last_seen_at
FROM encounter_stats es
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = es.xuid
LEFT JOIN kv_stats kv ON kv.xuid = es.xuid
WHERE 1=1 %s
ORDER BY es.count_together DESC, es.xuid ASC
LIMIT 10`

// Q27CareerRivalsTpl : Career — top frags (souffre-douleur) ou top morts (némésis).
//
// Format string : %s à remplacer par "frags" ou "deaths" pour ORDER BY.
//
// Paramètres :
//
//	?1 = xuid joueur (CASE killer→victim)
//	?2 = xuid joueur (SUM frags : kills par moi)
//	?3 = xuid joueur (SUM deaths : kills par lui)
//	?4 = xuid joueur (filtre WHERE killer_xuid)
//	?5 = xuid joueur (filtre WHERE victim_xuid)
//	?6 = xuid joueur (exclusion self dans final WHERE)
//
// Source : SUM(kill_count) sur shared.killer_victim_pairs (1 row par kill,
// cf. comment migration steps_shared.go:118-120). COUNT(DISTINCT match_id)
// pour le nombre de matchs partagés. Bots exclus via xuid NOT LIKE 'bid(%%'.
const Q27CareerRivalsTpl = `
WITH pairs AS (
    SELECT
        CASE WHEN kv.killer_xuid = ? THEN kv.victim_xuid ELSE kv.killer_xuid END AS opp_xuid,
        SUM(CASE WHEN kv.killer_xuid = ? THEN kv.kill_count ELSE 0 END) AS frags,
        SUM(CASE WHEN kv.victim_xuid = ? THEN kv.kill_count ELSE 0 END) AS deaths,
        COUNT(DISTINCT kv.match_id) AS match_count
    FROM killer_victim_pairs kv
    WHERE kv.killer_xuid = ? OR kv.victim_xuid = ?
    GROUP BY opp_xuid
)
SELECT
    p.opp_xuid AS xuid,
    -- opp_xuid (killer_victim_pairs) peut être orphelin de la vue → fallback masqué
    -- "Joueur ####" (jamais de xuid brut, miroir de analysis.MaskedXuidLabelSQL).
    COALESCE(vg.gamertag, ('Joueur ' || RIGHT(p.opp_xuid, 4))) AS gamertag,
    p.frags,
    p.deaths,
    p.match_count
FROM pairs p
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = p.opp_xuid
WHERE p.opp_xuid <> ?
  AND p.opp_xuid NOT LIKE 'bid(%%'
ORDER BY %s DESC, p.match_count DESC, p.opp_xuid ASC
LIMIT 10`

// Q28RelationsTpl : hub Communauté > Relations. Généralisation de Q26 — TOUS
// les joueurs récurrents (HAVING count_together >= 2), sans LIMIT, avec en plus
// MIN(start_time) AS first_seen et les KDA moyens en allié / en ennemi.
//
// Format string : 4 %s = winExpr, lossExpr, winExpr, lossExpr (title-aware,
// fallback "e.my_outcome = 2/3" byte-identique Halo). PAS de clause d'exclusion
// friends ni de LIMIT (le hub Relations affiche tout le monde).
//
// Placeholders ? (ordre, tous = xuid du joueur courant) :
//
//	?1 my_history.WHERE xuid = ?
//	?2 encounters JOIN p.xuid <> ?
//	?3 kv_stats CASE killer_xuid = ? → victim
//	?4 kv_stats CASE killer_xuid = ? → kills_by_me
//	?5 kv_stats CASE victim_xuid = ? → kills_by_them
//	?6 kv_stats WHERE killer_xuid = ?
//	?7 kv_stats WHERE OR victim_xuid = ?
//
// Colonnes SELECT (15, scannées dans cet ordre) : xuid, gamertag,
// count_together, ally_count, enemy_count, wins_as_ally, losses_as_ally,
// wins_vs_enemy, losses_vs_enemy, kills_dealt, deaths_suffered, avg_kda_with,
// avg_kda_against, first_seen_at, last_seen_at.
const Q28RelationsTpl = `
WITH my_history AS (
    SELECT match_id, team_id, outcome
    FROM match_participants
    WHERE xuid = ?
),
encounters AS (
    SELECT
        p.xuid,
        p.match_id,
        p.team_id  AS opp_team_id,
        h.team_id  AS my_team_id,
        h.outcome  AS my_outcome,
        p.kda      AS opp_kda,
        COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time
    FROM my_history h
    JOIN match_participants p
        ON p.match_id = h.match_id
       AND p.xuid <> ?
       AND p.xuid NOT LIKE 'bid(%%'
    LEFT JOIN match_registry r ON r.match_id = p.match_id
),
encounter_stats AS (
    SELECT
        e.xuid,
        COUNT(DISTINCT e.match_id) AS count_together,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id THEN e.match_id END) AS ally_count,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id THEN e.match_id END) AS enemy_count,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id AND %s THEN e.match_id END) AS wins_as_ally,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id AND %s THEN e.match_id END) AS losses_as_ally,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id AND %s THEN e.match_id END) AS wins_vs_enemy,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id AND %s THEN e.match_id END) AS losses_vs_enemy,
        AVG(CASE WHEN e.opp_team_id  = e.my_team_id THEN e.opp_kda END) AS avg_kda_with,
        AVG(CASE WHEN e.opp_team_id <> e.my_team_id THEN e.opp_kda END) AS avg_kda_against,
        MIN(e.start_time) AS first_seen_at,
        MAX(e.start_time) AS last_seen_at
    FROM encounters e
    GROUP BY e.xuid
    HAVING COUNT(DISTINCT e.match_id) >= 2
),
kv_stats AS (
    SELECT
        opp_xuid AS xuid,
        SUM(kills_by_me)   AS kills_dealt,
        SUM(kills_by_them) AS deaths_suffered
    FROM (
        SELECT
            CASE WHEN kv.killer_xuid = ? THEN kv.victim_xuid ELSE kv.killer_xuid END AS opp_xuid,
            CASE WHEN kv.killer_xuid = ? THEN kv.kill_count   ELSE 0               END AS kills_by_me,
            CASE WHEN kv.victim_xuid = ? THEN kv.kill_count   ELSE 0               END AS kills_by_them
        FROM killer_victim_pairs kv
        WHERE kv.killer_xuid = ? OR kv.victim_xuid = ?
    ) t
    GROUP BY opp_xuid
)
SELECT
    es.xuid,
    COALESCE(vg.gamertag, ('Joueur ' || RIGHT(es.xuid, 4))) AS gamertag,
    es.count_together,
    es.ally_count,
    es.enemy_count,
    es.wins_as_ally,
    es.losses_as_ally,
    es.wins_vs_enemy,
    es.losses_vs_enemy,
    COALESCE(kv.kills_dealt, 0)    AS kills_dealt,
    COALESCE(kv.deaths_suffered,0) AS deaths_suffered,
    es.avg_kda_with,
    es.avg_kda_against,
    es.first_seen_at,
    es.last_seen_at
FROM encounter_stats es
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = es.xuid
LEFT JOIN kv_stats kv ON kv.xuid = es.xuid
ORDER BY es.count_together DESC, es.xuid ASC`

// Q28RelationsScopedTpl : variante de Q28RelationsTpl restreinte à un
// sous-ensemble de match_id (segmentation serveur Phase 2). Le périmètre
// (expérience/classé, saison/période, playlist/mode, vue solo/escouade) est
// calculé EN AMONT par FiltersService.ResolveMatchIDs (cross-DB : player DB
// pour is_with_friends + shared pour le reste) ; SQL ne fait que restreindre
// par match_id. Deux points de restriction symétriques :
//   - my_history : limite l'historique du joueur au scope (alimente encounters)
//   - kv_stats   : limite les frags/morts échangés au scope (sinon les duels
//     d'autres périmètres pollueraient kills_dealt/deaths_suffered)
//
// Format string : 6 %s — winExpr, lossExpr, winExpr, lossExpr (cf. Q28RelationsTpl),
// puis 2 clauses IN injectées : ?h (my_history) et ?k (kv_stats). Chaque clause a
// la forme " AND match_id IN (?,?,…)".
//
// Placeholders ? (ordre) :
//
//	?1 my_history.WHERE xuid = ?
//	?…  scope IN (my_history)  — N placeholders du set
//	?  encounters JOIN p.xuid <> ?
//	?  kv_stats CASE killer_xuid = ? → victim
//	?  kv_stats CASE killer_xuid = ? → kills_by_me
//	?  kv_stats CASE victim_xuid = ? → kills_by_them
//	?  kv_stats WHERE killer_xuid = ?
//	?  kv_stats WHERE OR victim_xuid = ?
//	?…  scope IN (kv_stats)     — N placeholders du set
//
// Le binding exact est construit par buildRelationsQuery.
const Q28RelationsScopedTpl = `
WITH my_history AS (
    SELECT match_id, team_id, outcome
    FROM match_participants
    WHERE xuid = ?%s
),
encounters AS (
    SELECT
        p.xuid,
        p.match_id,
        p.team_id  AS opp_team_id,
        h.team_id  AS my_team_id,
        h.outcome  AS my_outcome,
        p.kda      AS opp_kda,
        COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time
    FROM my_history h
    JOIN match_participants p
        ON p.match_id = h.match_id
       AND p.xuid <> ?
       AND p.xuid NOT LIKE 'bid(%%'
    LEFT JOIN match_registry r ON r.match_id = p.match_id
),
encounter_stats AS (
    SELECT
        e.xuid,
        COUNT(DISTINCT e.match_id) AS count_together,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id THEN e.match_id END) AS ally_count,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id THEN e.match_id END) AS enemy_count,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id AND %s THEN e.match_id END) AS wins_as_ally,
        COUNT(DISTINCT CASE WHEN e.opp_team_id  = e.my_team_id AND %s THEN e.match_id END) AS losses_as_ally,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id AND %s THEN e.match_id END) AS wins_vs_enemy,
        COUNT(DISTINCT CASE WHEN e.opp_team_id <> e.my_team_id AND %s THEN e.match_id END) AS losses_vs_enemy,
        AVG(CASE WHEN e.opp_team_id  = e.my_team_id THEN e.opp_kda END) AS avg_kda_with,
        AVG(CASE WHEN e.opp_team_id <> e.my_team_id THEN e.opp_kda END) AS avg_kda_against,
        MIN(e.start_time) AS first_seen_at,
        MAX(e.start_time) AS last_seen_at
    FROM encounters e
    GROUP BY e.xuid
    HAVING COUNT(DISTINCT e.match_id) >= 2
),
kv_stats AS (
    SELECT
        opp_xuid AS xuid,
        SUM(kills_by_me)   AS kills_dealt,
        SUM(kills_by_them) AS deaths_suffered
    FROM (
        SELECT
            CASE WHEN kv.killer_xuid = ? THEN kv.victim_xuid ELSE kv.killer_xuid END AS opp_xuid,
            CASE WHEN kv.killer_xuid = ? THEN kv.kill_count   ELSE 0               END AS kills_by_me,
            CASE WHEN kv.victim_xuid = ? THEN kv.kill_count   ELSE 0               END AS kills_by_them
        FROM killer_victim_pairs kv
        WHERE (kv.killer_xuid = ? OR kv.victim_xuid = ?)%s
    ) t
    GROUP BY opp_xuid
)
SELECT
    es.xuid,
    COALESCE(vg.gamertag, ('Joueur ' || RIGHT(es.xuid, 4))) AS gamertag,
    es.count_together,
    es.ally_count,
    es.enemy_count,
    es.wins_as_ally,
    es.losses_as_ally,
    es.wins_vs_enemy,
    es.losses_vs_enemy,
    COALESCE(kv.kills_dealt, 0)    AS kills_dealt,
    COALESCE(kv.deaths_suffered,0) AS deaths_suffered,
    es.avg_kda_with,
    es.avg_kda_against,
    es.first_seen_at,
    es.last_seen_at
FROM encounter_stats es
LEFT JOIN v_gamertag_lookup vg ON vg.xuid = es.xuid
LEFT JOIN kv_stats kv ON kv.xuid = es.xuid
ORDER BY es.count_together DESC, es.xuid ASC`

// QRelationsPlayerWinRateTpl : WR HISTORIQUE (tout-temps) du joueur — référence
// du « lift » de la carte Noyau dur. Décisifs uniquement (win OU loss ; nuls/dnf
// exclus). NON scopé volontairement (moyenne de carrière). Title-aware via
// outcomeSQLEq (fallback "outcome = 2/3" Halo).
//
// Format string (3 %s) : winExpr, winExpr, lossExpr. Placeholder ? : ?1 = xuid joueur.
const QRelationsPlayerWinRateTpl = `
SELECT
    COUNT(*) FILTER (WHERE %s)            AS wins,
    COUNT(*) FILTER (WHERE (%s) OR (%s))  AS decided
FROM match_participants
WHERE xuid = ?`

// QRelationsCoreFormTpl : issues des derniers matchs du joueur joués AVEC au
// moins un membre du noyau dur (même équipe), ordonnées récent→ancien (le repo
// inverse en ancien→récent pour la frise). DISTINCT par match (un match avec 2
// fidèles ne compte qu'une fois). Title-aware via outcomeSQLEq.
//
// Format string (4 %s) : scopeIn (my_matches, " AND mp.match_id IN (…)"),
// coreIn (placeholders du noyau), winExpr, lossExpr (sur la colonne `outcome`).
// Placeholders ? : ?1 = xuid joueur (my_matches), puis scope, puis ?= xuid joueur
// (exclusion self dans with_core), puis les xuid du noyau, puis ?= limit.
const QRelationsCoreFormTpl = `
WITH my_matches AS (
    SELECT mp.match_id, mp.team_id, mp.outcome AS outcome,
           COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time
    FROM match_participants mp
    LEFT JOIN match_registry r ON r.match_id = mp.match_id
    WHERE mp.xuid = ?%s
),
with_core AS (
    SELECT DISTINCT h.match_id, h.outcome AS outcome, h.start_time
    FROM my_matches h
    JOIN match_participants c
        ON c.match_id = h.match_id
       AND c.team_id  = h.team_id
       AND c.xuid <> ?
       AND c.xuid IN (%s)
)
SELECT CASE WHEN %s THEN 'win' WHEN %s THEN 'loss' ELSE 'other' END AS outcome_label
FROM with_core
ORDER BY start_time DESC NULLS LAST
LIMIT ?`

// Q22 : Sessions — chargement des matchs pour le calcul des sessions.
// Parametre : ?1 = xuid du joueur.
// Retourne 6 colonnes : match_id, start_time, teammates_sig, is_ranked,
// time_played_seconds, end_time (NULL si absent).
const Q22SessionMatches = `
SELECT
    mp.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    -- Signature des coeequipiers : XUIDs tries concatenes (hors joueur lui-meme)
    (SELECT string_agg(t.xuid, ',' ORDER BY t.xuid)
     FROM match_participants t
     WHERE t.match_id = mp.match_id AND t.team_id = mp.team_id AND t.xuid <> ?
    )                                                   AS teammates_sig,
    COALESCE(r.is_ranked, FALSE)                       AS is_ranked,
    mp.time_played_seconds,
    CASE WHEN mp.time_played_seconds IS NOT NULL
         THEN COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') + INTERVAL (mp.time_played_seconds || ' seconds')
         ELSE NULL
    END                                                 AS end_time
FROM match_participants mp
JOIN match_registry r ON r.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') ASC`

// Q23 : Stats series — chargement des matchs avec metriques pour perf score.
// Parametre : ?1 = xuid du joueur.
// Retourne toutes les colonnes necessaires a compute_relative_performance_score.
// Q23StatsMatchesShared : partie shared-only de Q23 (exécutée sur SharedReader).
// La partie player_match_enrichment est chargée séparément via Q23StatsMatchesPlayerEnrich
// puis mergée côté Go (cf. StatsRepo.LoadStatsMatches, ADR 0016).
const Q23StatsMatchesShared = `
SELECT
    mp.match_id,
    COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') AS start_time,
    mp.outcome,
    COALESCE(mp.kills, 0)              AS kills,
    COALESCE(mp.deaths, 0)             AS deaths,
    COALESCE(mp.assists, 0)            AS assists,
    mp.kda,
    mp.accuracy,
    mp.personal_score,
    mp.damage_dealt,
    mp.damage_taken,
    mp.time_played_seconds,
    mp.team_mmr,
    mp.enemy_mmr,
    mp.kills_expected,
    mp.deaths_expected,
    mp.rank,
    COALESCE(r.is_ranked, FALSE)     AS is_ranked,
    COALESCE(r.playlist_name, '')    AS playlist_name,
    COALESCE(r.pair_name, '')        AS pair_name,
    mp.team_id,
    CASE
        WHEN COALESCE(mp.damage_dealt, 0) > 0 THEN
            ? * (COALESCE(mp.kills, 0) + COALESCE(mp.assists, 0) / 3.0) / mp.damage_dealt
        ELSE 0.0
    END AS offensive_conversion,
    CASE
        WHEN COALESCE(mp.damage_taken, 0) > 0 AND COALESCE(mp.deaths, 0) > 0 THEN
            mp.damage_taken / (? * mp.deaths)
        ELSE 0.0
    END AS defensive_resistance
FROM match_participants mp
JOIN match_registry r ON r.match_id = mp.match_id
WHERE mp.xuid = ?
ORDER BY COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC') ASC`

// Q23StatsMatchesPlayerEnrich : Phase B de Q23 — performance_score + session
// + label depuis player_match_enrichment pour les match_ids résultants de
// Phase A. Le template %s reçoit les placeholders IN ?... .
const Q23StatsMatchesPlayerEnrichTpl = `
SELECT match_id, performance_score, session_id, session_label
FROM player_match_enrichment_latest
WHERE match_id IN (%s)`

// Q24 : LUSR — chargement du rating par match depuis match_skill_rank.
// Aucun parametre : la player DB (stats.duckdb) est mono-joueur, match_skill_rank
// ne contient que ses propres ratings — pas de filtre xuid necessaire.
const Q24LUSRHistory = `
SELECT
    msr.match_id,
    msr.rating_value,
    msr.rating_deviation,
    msr.playlist_group
FROM match_skill_rank msr
ORDER BY msr.match_id ASC`

// Q25 : Stats series — participants d un ensemble de matchs (pour LUSR enemy strength).
// Parametre : ?1 = xuid joueur (pour filtrer son equipe).
// Utilise les matchs de Q23 comme sous-requete.
const Q25MatchParticipants = `
SELECT
    mp.match_id,
    mp.xuid,
    mp.team_id,
    mp.kills_expected,
    mp.deaths_expected
FROM match_participants mp
WHERE mp.match_id IN (
    SELECT DISTINCT mp2.match_id
    FROM match_participants mp2
    WHERE mp2.xuid = ?
)
ORDER BY mp.match_id, mp.xuid`

// Q26csrSnapshots : récupère les snapshots CSR du joueur pour la saison courante.
// Le filtre season_id évite qu'un snapshot d'une saison passée (même playlist)
// supplante la valeur courante dans l'index par playlist_id. Garde-fou : si le
// paramètre saison est vide (config absente), aucun filtre n'est appliqué —
// comportement historique préservé.
// Triés par alltime_value DESC pour avoir les meilleures playlists en premier.
const Q26csrSnapshots = `
SELECT
    playlist_id,
    COALESCE(playlist_name, ''),
    COALESCE(queue, ''),
    COALESCE(input, ''),
    COALESCE(season_id, ''),
    COALESCE(current_value, 0),
    COALESCE(current_tier, ''),
    COALESCE(current_sub_tier, 0),
    COALESCE(current_measurement_remaining, 0),
    COALESCE(season_value, 0),
    COALESCE(season_tier, ''),
    COALESCE(season_sub_tier, 0),
    COALESCE(alltime_value, 0),
    COALESCE(alltime_tier, ''),
    COALESCE(alltime_sub_tier, 0)
FROM player_csr_snapshots_latest
WHERE (? = '' OR season_id = ?)
ORDER BY alltime_value DESC, current_value DESC`

// QPlaylistsCatalogRanked : liste les playlists ranked actives du catalogue
// pour un titre donné (metadata.duckdb). Utilisé par la page Carrière pour
// afficher toutes les playlists classées du joueur, y compris celles sans
// snapshot dans player_csr_snapshots (placement à 0 match joué).
//
// is_ranked est désormais fiable : le seed migration "seed_ranked_playlists_catalog"
// le force depuis la référence rankedplaylists (source de vérité), et tous les
// chemins d'écriture s'y conforment. Le hack historique STRPOS(name,'ranked') —
// qui ne rattrapait que les playlists dont le nom contenait littéralement
// "ranked" — est supprimé.
const QPlaylistsCatalogRanked = `
SELECT playlist_asset_id, COALESCE(name_canonical, '')
FROM playlists_catalog
WHERE title_slug = ?
  AND is_active = TRUE
  AND is_ranked = TRUE
ORDER BY name_canonical`

// Q26csrAlltimePeak : CANDIDATS du meilleur CSR alltime, toutes playlists.
// Utilisé par la home page pour remplacer la lecture depuis match_skill_rank.
//
// Renvoie TOUS les snapshots éligibles (palier renseigné OU valeur > 0) ; la
// SÉLECTION du pic est faite EN GO (pickBestCSRAlltime) via l'ordinal canonique
// analysis.CSRTierOrdinal — source UNIQUE de l'ordre des paliers (pas de CASE SQL
// dupliqué). Raison : certains titres (Halo 5) exposent le palier (DesignationId →
// "Diamond", sub_tier) mais une valeur numérique = 0 (l'API ne la fournit pas).
// Exiger value > 0 masquait un vrai pic (ex. Diamant V) → "Non classé" erroné.
// Title-agnostic : titre avec valeur (Infinite, départage) ET tier-only (H5).
const Q26csrAlltimePeak = `
SELECT
    alltime_value,
    alltime_tier,
    alltime_sub_tier
FROM player_csr_snapshots_latest
WHERE (alltime_tier IS NOT NULL AND TRIM(alltime_tier) != '')
   OR (alltime_value IS NOT NULL AND alltime_value > 0)`
