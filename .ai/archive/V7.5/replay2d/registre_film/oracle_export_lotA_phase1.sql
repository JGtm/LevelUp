-- oracle_export_lotA_phase1.sql — EXPORT LECTURE SEULE des evenements d'objectif de la base,
-- pour le controle A.1.2 (accord des actions publiees avec `match_objective_events`).
--
-- LECTURE SEULE, AUCUNE ECRITURE : la base partagee est ouverte en `-readonly` (equivalent CLI de
-- `OpenReadForQuery`). Rejouer avec :
--   duckdb -readonly data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb \
--     -c ".read <ce fichier>"   (depuis le worktree PRINCIPAL, qui porte les donnees)
COPY (
  SELECT match_id, seq, time_ms, objective_type, event_type, team_id, value, source, confidence
  FROM match_objective_events
  WHERE substr(match_id,1,8) IN ('64e8adfa','530820e5','53ce4390')
  ORDER BY match_id, seq
) TO 'C:/Users/Guillaume/Projects/LevelUp-wt-score-film/.ai/V7.5/replay2d/registre_film/oracle_lotA_objective_events.tsv'
  (HEADER, DELIMITER E'\t');
