-- Corpus de la phase 0-bis (item A.0b.4) : n >= 3 par mode.
-- Un SEUL export, en lecture seule. La liste des films du cache local est jointe depuis un CSV
-- (951 lignes) : la base n'est jamais ecrite et la requete ne suppose aucun film.
CREATE OR REPLACE TEMP VIEW cache_films AS
  SELECT short8 FROM read_csv_auto('C:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Projects-LevelUp/e3c97844-8479-4bb5-915f-496872b71a0a/scratchpad/cache_films.csv');

CREATE OR REPLACE TEMP VIEW candidats AS
  SELECT r.*
  FROM match_registry r
  JOIN cache_films c ON c.short8 = substr(r.match_id, 1, 8)
  WHERE r.is_firefight = false
    AND r.player_count IS NOT NULL
    AND (
      lower(r.game_variant_name) LIKE '%oddball%'
      OR lower(r.game_variant_name) LIKE '%stronghold%'
      OR lower(r.game_variant_name) LIKE '%king of the hill%'
      OR lower(r.game_variant_name) LIKE '%koth%'
      OR (lower(r.game_variant_name) LIKE '%slayer%'
          AND lower(r.game_variant_name) NOT LIKE '%fiesta%'
          AND lower(r.game_variant_name) NOT LIKE '%btb%')
    );

COPY (
  SELECT * FROM candidats ORDER BY game_variant_name, match_id
) TO 'C:/Users/Guillaume/Projects/LevelUp-wt-score-film/.ai/V7.5/replay2d/registre_film/oracle_lotA_bis.tsv'
  (HEADER, DELIMITER E'\t');

COPY (
  SELECT p.* FROM match_participants p
  WHERE substr(p.match_id, 1, 8) IN (SELECT substr(match_id, 1, 8) FROM candidats)
  ORDER BY p.match_id, p.team_id, p.xuid
) TO 'C:/Users/Guillaume/Projects/LevelUp-wt-score-film/.ai/V7.5/replay2d/registre_film/oracle_lotA_bis_participants.tsv'
  (HEADER, DELIMITER E'\t');
