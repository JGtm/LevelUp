COPY (
  SELECT * FROM match_registry
  WHERE substr(match_id,1,8) IN (
    '64e8adfa','530820e5','53ce4390',
    '0a247154','01e1f945','606d9844','8076f97f',
    '7344d24f','696a9d7c','24dbb67d',
    '000d5950','06dfe6d9')
  ORDER BY match_id
) TO 'C:/Users/Guillaume/Projects/LevelUp-wt-score-film/.ai/V7.5/replay2d/registre_film/oracle_lotA.tsv'
  (HEADER, DELIMITER E'\t');

COPY (
  SELECT * FROM match_participants
  WHERE substr(match_id,1,8) IN (
    '64e8adfa','530820e5','53ce4390',
    '0a247154','01e1f945','606d9844','8076f97f',
    '7344d24f','696a9d7c','24dbb67d',
    '000d5950','06dfe6d9')
  ORDER BY match_id, team_id, xuid
) TO 'C:/Users/Guillaume/Projects/LevelUp-wt-score-film/.ai/V7.5/replay2d/registre_film/oracle_lotA_participants.tsv'
  (HEADER, DELIMITER E'\t');
