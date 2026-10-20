-- Oracle du lot B item B.0.3 (prolongation) — UN SEUL export, LECTURE SEULE.
-- Question posee : quels matchs AYANT UN FILM DANS LE CACHE LOCAL sont flagues
-- « Prolongation » par le chemin de production (analysis.ResolveOvertime) ?
--   elapsed = MEDIAN(time_played_seconds) des participants present_at_beginning
--             AND present_at_completion, repli MAX (duckdb/elapsed_seconds.go)
--   flag    = elapsed > regulation_seconds + 40 (analysis.OvertimeMarginSeconds)
-- La table de reglement est celle de config/titles/halo_infinite/mappings/regulation.toml
-- (9 variantes) : une variante absente n'est JAMAIS flaguee, c'est le contrat du fichier.
-- La base n'est jamais ecrite ; la liste des films est jointe depuis un CSV du cache.
ATTACH 'C:/Users/Guillaume/Projects/LevelUp/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb' AS s (READ_ONLY);

CREATE OR REPLACE TEMP VIEW cache_films AS
  SELECT short8 FROM read_csv_auto('C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Projects-LevelUp/e3c97844-8479-4bb5-915f-496872b71a0a/scratchpad/cache_films_bp.csv');

CREATE OR REPLACE TEMP VIEW regulation AS
  SELECT * FROM (VALUES
    ('CTF:Arena', 720), ('CTF:Arena Neutral Flag', 720), ('Team Slayer:Arena', 720),
    ('Slayer:Arena', 720), ('Slayer:Arena Super Fiesta', 720), ('Slayer:Arena Tactical', 720),
    ('Strongholds:Arena', 720), ('Arena:Slayer', 720), ('Arena:Team Slayer', 720)
  ) AS t(game_variant_name, regulation_seconds);

CREATE OR REPLACE TEMP VIEW elapsed AS
  SELECT p.match_id,
         COALESCE(
           MEDIAN(p.time_played_seconds) FILTER (
             WHERE COALESCE(p.present_at_beginning, FALSE)
               AND COALESCE(p.present_at_completion, FALSE)),
           MAX(p.time_played_seconds)) AS elapsed_seconds
  FROM s.match_participants p
  GROUP BY p.match_id;

COPY (
  SELECT substr(r.match_id,1,8) AS short8, r.match_id, r.game_variant_name, r.map_name,
         g.regulation_seconds, CAST(round(e.elapsed_seconds) AS INTEGER) AS elapsed_seconds,
         CAST(round(e.elapsed_seconds) - g.regulation_seconds AS INTEGER) AS over_seconds,
         (round(e.elapsed_seconds) > g.regulation_seconds + 40) AS is_overtime,
         r.duration_seconds, r.playable_duration_seconds, r.team_0_score, r.team_1_score,
         r.player_count, r.start_time_utc
  FROM s.match_registry r
  JOIN cache_films c ON c.short8 = substr(r.match_id,1,8)
  JOIN regulation g ON g.game_variant_name = trim(r.game_variant_name)
  LEFT JOIN elapsed e ON e.match_id = r.match_id
  WHERE r.is_firefight = false
  ORDER BY is_overtime DESC, over_seconds DESC
) TO 'C:/Users/Guillaume/Projects/LevelUp-wt-joueur-moteur/.ai/V7.5/replay2d/registre_film/oracle_lotB_overtime.tsv'
  (HEADER, DELIMITER E'\t');
