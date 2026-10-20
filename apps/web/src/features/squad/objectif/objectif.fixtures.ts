/**
 * objectif.fixtures.ts — les soirées témoins de la maquette C3EW (relevées en lecture seule dans
 * la base réelle, recopiées ici) au format du contrat : bloc `formes_retenues`, historique de
 * matchs et `squad_objective_history`. Aucune lecture de base : des données figées.
 *
 * - 07/09/2026 : quatre Bases puis trois Drapeau, 1 victoire sur 7 ; les parts par match sont
 *   celles de la maquette (PERMATCH), les totaux par colonne ceux du rapport de force.
 * - 22/09/2026 : deux Drapeau (JGtm, Chocoboflor, Madina97294) ; les totaux par joueur sont
 *   ceux des fiches de la maquette (FICHES), répartis sur les deux matchs.
 */
import type {
  SquadFormesBlock,
  SquadFormesMatch,
  SquadFormesObjectiveColumn,
  SquadMatchHistoryRow,
  SquadObjectiveHistory,
} from '@/lib/api/types'

const OUR_TEAM = 0
const THEIR_TEAM = 1

const col = (key: string, role: string, extra: Partial<SquadFormesObjectiveColumn> = {}): SquadFormesObjectiveColumn => ({
  key,
  role,
  ...extra,
})

/** Les colonnes publiées par famille, dans l'ordre du serveur (rôle, puis narrative). */
export const ZONE_COLUMNS: SquadFormesObjectiveColumn[] = [
  col('zone_captures', 'take'),
  col('zone_offensive_kills', 'take'),
  col('zone_secures', 'defend'),
  col('zone_defensive_kills', 'defend'),
  col('time_in_zones_seconds', 'hold', { duration: true }),
]

export const FLAG_COLUMNS: SquadFormesObjectiveColumn[] = [
  col('flag_captures', 'take'),
  col('flag_capture_assists', 'take'),
  col('flag_steals', 'take'),
  col('flag_returners_killed', 'take'),
  col('flag_grabs_net', 'take', { optional: true }),
  col('flag_returns', 'defend'),
  col('flag_secures', 'defend'),
  col('flag_carriers_killed', 'defend'),
  col('time_as_flag_carrier_seconds', 'hold', { duration: true }),
]

type Values = Record<string, number>

function objectiveMatch(
  id: string,
  start: string,
  map: string,
  family: string,
  columns: SquadFormesObjectiveColumn[],
  players: { xuid: string; team: number; values: Values }[],
): SquadFormesMatch {
  return {
    match_id: id,
    start_time: start,
    map_label: map,
    mode_label: family === 'ctf' ? 'Drapeau' : 'Bases',
    player_team: OUR_TEAM,
    objective: {
      family,
      columns,
      players: players.map((p) => ({ xuid: p.xuid, team_id: p.team, values: p.values })),
    },
  }
}

/** Notre camp (sur JGtm) et l'adversaire (le lobby moins notre camp). */
function campVsLobby(camp: Values, lobby: Values) {
  const them: Values = {}
  for (const k of Object.keys(lobby)) them[k] = lobby[k] - (camp[k] ?? 0)
  return [
    { xuid: 'xj', team: OUR_TEAM, values: camp },
    { xuid: 'xadv', team: THEIR_TEAM, values: them },
  ]
}

// --- 07/09 : [notre camp, lobby] par colonne et par match ------------------------------

type Pair = [number, number]
const zone = (cap: Pair, off: Pair, sec: Pair, def: Pair, time: Pair) => ({
  camp: { zone_captures: cap[0], zone_offensive_kills: off[0], zone_secures: sec[0], zone_defensive_kills: def[0], time_in_zones_seconds: time[0] },
  lobby: { zone_captures: cap[1], zone_offensive_kills: off[1], zone_secures: sec[1], zone_defensive_kills: def[1], time_in_zones_seconds: time[1] },
})
const flag = (take: [Pair, Pair, Pair, Pair], net: Pair, def: [Pair, Pair, Pair], time: Pair) => ({
  camp: {
    flag_captures: take[0][0], flag_capture_assists: take[1][0], flag_steals: take[2][0], flag_returners_killed: take[3][0],
    flag_grabs_net: net[0], flag_returns: def[0][0], flag_secures: def[1][0], flag_carriers_killed: def[2][0],
    time_as_flag_carrier_seconds: time[0],
  },
  lobby: {
    flag_captures: take[0][1], flag_capture_assists: take[1][1], flag_steals: take[2][1], flag_returners_killed: take[3][1],
    flag_grabs_net: net[1], flag_returns: def[0][1], flag_secures: def[1][1], flag_carriers_killed: def[2][1],
    time_as_flag_carrier_seconds: time[1],
  },
})

const SOIREE_0709: { id: string; start: string; map: string; fam: string; v: { camp: Values; lobby: Values }; outcome: number; score: string }[] = [
  { id: 'b1', start: '2026-09-07T19:26:00Z', map: 'Banished Narrows', fam: 'zones_strongholds', v: zone([22, 41], [7, 15], [4, 13], [8, 20], [285.7, 599.9]), outcome: 3, score: '82–200' },
  { id: 'b2', start: '2026-09-07T19:34:00Z', map: 'Isolation', fam: 'zones_strongholds', v: zone([11, 25], [3, 10], [1, 6], [2, 7], [140.8, 316.4]), outcome: 3, score: '31–200' },
  { id: 'b3', start: '2026-09-07T19:42:00Z', map: 'Illusion', fam: 'zones_strongholds', v: zone([22, 48], [7, 16], [3, 9], [5, 14], [401.3, 619.8]), outcome: 2, score: '200–183' },
  { id: 'b4', start: '2026-09-07T19:52:00Z', map: 'Fortress', fam: 'zones_strongholds', v: zone([16, 36], [5, 11], [2, 9], [5, 12], [240.0, 457.8]), outcome: 3, score: '89–200' },
  { id: 'd1', start: '2026-09-07T20:00:00Z', map: 'Origin', fam: 'ctf', v: flag([[1, 4], [0, 3], [6, 11], [0, 2]], [10, 25], [[3, 9], [7, 22], [3, 7]], [86.0, 209.3]), outcome: 3, score: '1–3' },
  { id: 'd2', start: '2026-09-07T20:10:00Z', map: 'Domicile', fam: 'ctf', v: flag([[0, 2], [1, 2], [2, 5], [0, 2]], [6, 17], [[2, 5], [4, 9], [1, 4]], [32.7, 110.1]), outcome: 3, score: '0–3' },
  { id: 'd3', start: '2026-09-07T20:14:00Z', map: 'Absolution', fam: 'ctf', v: flag([[0, 4], [0, 3], [7, 15], [0, 2]], [13, 28], [[2, 8], [13, 23], [6, 6]], [51.8, 196.0]), outcome: 3, score: '0–3' },
]

const SQUAD = [
  { xuid: 'xj', gamertag: 'JGtm' },
  { xuid: 'xc', gamertag: 'Chocoboflor' },
  { xuid: 'xm', gamertag: 'Madina97294' },
]

export function block0709(): SquadFormesBlock {
  return {
    available: true,
    main_xuid: 'xj',
    matches_measured: 7,
    matches_total: 7,
    squad: SQUAD,
    matches: SOIREE_0709.map((m) =>
      objectiveMatch(m.id, m.start, m.map, m.fam, m.fam === 'ctf' ? FLAG_COLUMNS : ZONE_COLUMNS, campVsLobby(m.v.camp, m.v.lobby)),
    ),
  }
}

/** L'historique de matchs de la page (DESC, comme le serveur le rend) — sans la ligne de d2. */
export function history0709(): SquadMatchHistoryRow[] {
  return SOIREE_0709.filter((m) => m.id !== 'd2')
    .map((m) => ({
      match_id: m.id,
      start_time: m.start,
      map_ui: m.map,
      outcome: m.outcome,
      score_label: m.score,
      dominance_flag: m.id === 'b2' ? 2 : 0,
    }) as SquadMatchHistoryRow)
    .reverse()
}

// --- 22/09 : les fiches de la maquette (JGtm, Chocoboflor, Madina97294, reste du camp) ----------

/** [JGtm, Chocoboflor, Madina97294, reste du camp] et le lobby, par colonne, sur la soirée. */
const FICHES_2209: Record<string, { camp: [number, number, number, number]; lobby: number }> = {
  flag_captures: { camp: [4, 0, 0, 0], lobby: 7 },
  flag_capture_assists: { camp: [0, 1, 1, 0], lobby: 5 },
  flag_steals: { camp: [3, 4, 5, 2], lobby: 26 },
  flag_returners_killed: { camp: [0, 0, 1, 1], lobby: 5 },
  flag_grabs_net: { camp: [7, 5, 8, 2], lobby: 48 },
  flag_returns: { camp: [3, 4, 3, 4], lobby: 23 },
  flag_secures: { camp: [5, 2, 4, 0], lobby: 23 },
  flag_carriers_killed: { camp: [2, 0, 4, 2], lobby: 20 },
  time_as_flag_carrier_seconds: { camp: [64.9, 19.7, 44.8, 8.1], lobby: 248.8 },
}

/** Répartit une valeur sur les deux matchs (entiers : moitié haute puis le reste). */
function split(v: number, first: boolean): number {
  const a = Number.isInteger(v) ? Math.ceil(v / 2) : Math.round((v / 2) * 10) / 10
  return first ? a : Math.round((v - a) * 10) / 10
}

export function block2209(): SquadFormesBlock {
  const owners = ['xj', 'xc', 'xm', 'xrest']
  const matches = [
    { id: 's1', start: '2026-09-22T19:23:00Z', map: 'Starboard' },
    { id: 's2', start: '2026-09-22T19:46:00Z', map: 'Origin' },
  ].map((m, mi) => {
    const players = owners.map((xuid, oi) => ({
      xuid,
      team: OUR_TEAM,
      values: Object.fromEntries(Object.entries(FICHES_2209).map(([k, v]) => [k, split(v.camp[oi], mi === 0)])),
    }))
    const adv: Values = {}
    for (const [k, v] of Object.entries(FICHES_2209)) {
      adv[k] = split(v.lobby - v.camp.reduce((a, b) => a + b, 0), mi === 0)
    }
    return objectiveMatch(m.id, m.start, m.map, 'ctf', FLAG_COLUMNS, [...players, { xuid: 'xadv', team: THEIR_TEAM, values: adv }])
  })
  return { available: true, main_xuid: 'xj', matches_measured: 2, matches_total: 2, squad: SQUAD, matches }
}

// --- Historique d'objectif (Go) : 07/09 et ses dix soirées précédentes (HAB_ALL) ---------------

const HAB_ALL: [string, [number, string][], number, number, number, number, number][] = [
  ['2026-04-06', [[2, 'ctf'], [1, 'zones_strongholds']], 3, 1, 51.6, 49.9, 50.5],
  ['2026-04-19', [[2, 'zones_strongholds'], [2, 'ctf']], 4, 2, 50.9, 53.9, 56.8],
  ['2026-04-27', [[5, 'zones_strongholds'], [1, 'ctf']], 6, 2, 48.0, 49.3, 56.7],
  ['2026-05-05', [[3, 'ctf'], [1, 'zones_strongholds']], 4, 2, 49.2, 61.7, 60.2],
  ['2026-05-11', [[2, 'zones_strongholds'], [2, 'ctf']], 4, 2, 52.4, 52.6, 56.2],
  ['2026-05-21', [[6, 'ctf']], 6, 3, 47.6, 49.6, 48.3],
  ['2026-06-03', [[2, 'zones_strongholds'], [2, 'ctf']], 4, 3, 51.5, 52.3, 54.1],
  ['2026-06-09', [[2, 'ctf'], [1, 'zones_strongholds']], 3, 2, 50.8, 55.0, 53.0],
  ['2026-07-28', [[3, 'ctf']], 3, 0, 25.9, 40.1, 24.1],
  ['2026-07-31', [[2, 'ctf'], [1, 'zones_strongholds']], 3, 2, 46.5, 47.5, 54.8],
  ['2026-09-07', [[4, 'zones_strongholds'], [3, 'ctf']], 7, 1, 39.0, 36.8, 43.8],
]

function evening(r: (typeof HAB_ALL)[number]) {
  return {
    session_label: r[0],
    start_time: `${r[0]}T19:00:00Z`,
    objective_matches: r[2],
    wins: r[3],
    families: r[1].map(([n, f]) => ({ family: f, matches: n })),
    take: r[4] / 100,
    defend: r[5] / 100,
    hold: r[6] / 100,
  }
}

export function history0709Evenings(): SquadObjectiveHistory {
  return {
    current: evening(HAB_ALL[10]),
    previous: HAB_ALL.slice(0, 10).map(evening),
    evenings_with_objective: 25,
    evenings_below_minimum: 0,
    min_objective_matches: 3,
  }
}

/** 22/09 : deux matchs à objectif — sous le minimum (24 soirées sur 49 dans le même cas). */
export function history2209Evenings(): SquadObjectiveHistory {
  return {
    current: { session_label: '2026-09-22', start_time: '2026-09-22T19:23:00Z', objective_matches: 2, wins: 1, families: [{ family: 'ctf', matches: 2 }], take: 0.4, defend: 0.48, hold: 0.56 },
    previous: HAB_ALL.slice(1).map(evening),
    evenings_with_objective: 49,
    evenings_below_minimum: 24,
    min_objective_matches: 3,
  }
}
