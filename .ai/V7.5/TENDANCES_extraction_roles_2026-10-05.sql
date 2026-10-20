-- Prendre / Défendre / Tenir par match pour JGtm et pour son équipe (classification de
-- narrative/objective_roles.go ; prises de drapeau NETTES depuis match_flag_grabs_net_latest).
WITH o AS (
  SELECT o.match_id, o.xuid,
    COALESCE(o.flag_captures,0) + COALESCE(o.flag_capture_assists,0) + COALESCE(o.flag_steals,0) + COALESCE(o.flag_returners_killed,0)
      + COALESCE(o.zone_captures,0) + COALESCE(o.zone_offensive_kills,0) + COALESCE(o.skull_grabs,0)
      + COALESCE(o.power_seeds_deposited,0) + COALESCE(o.power_seeds_stolen,0)
      + COALESCE(o.extraction_initiations_completed,0) + COALESCE(o.extraction_conversions_completed,0) + COALESCE(o.successful_extractions,0)
      + COALESCE(o.vip_kills,0) + COALESCE(o.vip_assists,0) + COALESCE(g.flag_grabs_net,0) AS take,
    COALESCE(o.flag_returns,0) + COALESCE(o.flag_secures,0) + COALESCE(o.flag_carriers_killed,0)
      + COALESCE(o.zone_secures,0) + COALESCE(o.zone_defensive_kills,0) + COALESCE(o.skull_carriers_killed,0)
      + COALESCE(o.power_seed_carriers_killed,0) + COALESCE(o.extraction_conversions_denied,0) + COALESCE(o.kills_as_vip,0) AS defend,
    COALESCE(o.time_as_flag_carrier_seconds,0) + COALESCE(o.time_in_zones_seconds,0) + COALESCE(o.time_as_skull_carrier_seconds,0)
      + COALESCE(o.time_as_power_seed_carrier_seconds,0) + COALESCE(o.time_as_power_seed_driver_seconds,0) + COALESCE(o.time_as_vip_seconds,0) AS hold
  FROM match_objective_stats_latest o
  LEFT JOIN match_flag_grabs_net_latest g ON g.match_id = o.match_id AND g.xuid = o.xuid
),
me AS (SELECT mp.match_id, mp.team_id FROM match_participants mp WHERE mp.xuid = '2533274823110022'
       AND mp.match_id IN (SELECT match_id FROM o WHERE xuid = '2533274823110022')),
team AS (
  SELECT me.match_id, count(*) FILTER (WHERE COALESCE(mp.present_at_completion, true)) AS ts, sum(o.take) AS tt, sum(o.defend) AS td, sum(o.hold) AS th
  FROM me JOIN match_participants mp ON mp.match_id = me.match_id AND mp.team_id = me.team_id
  LEFT JOIN o ON o.match_id = mp.match_id AND o.xuid = mp.xuid
  GROUP BY me.match_id
)
SELECT epoch(COALESCE(r.start_time_utc, r.start_time AT TIME ZONE 'UTC'))::BIGINT AS t,
  p.take AS pt, p.defend AS pd, round(p.hold) AS ph, team.tt, team.td, round(team.th) AS th, team.ts
FROM o p JOIN team USING (match_id) JOIN match_registry r USING (match_id)
WHERE p.xuid = '2533274823110022'
ORDER BY 1;
