/**
 * sessionEmprise.fixtures.ts — les colonnes de session témoins de la page Sessions (relevés
 * `.ai/V7.5/MESURES_SESSIONS_2026-10-06.md`, §0 à §3) :
 *   - `session2209()` : la soirée d'escouade du 22/09 (7 matchs, 6 filmés, deux CTF) — Emprise et
 *     objectif des fixtures de l'Escouade de la même soirée (`EMPRISE_2209`, `block2209`), frags par
 *     classe et six premiers outils de JGtm (65 frags : BR75 22, MK50 Sidekick 13, M41 SPNKr 11…),
 *     vies 56 près / 18 seul (37 / 4 frags) ;
 *   - `session0709()` : le 07/09 (4 Bases, 3 CTF) — l'objectif seul (`block0709`) ;
 *   - `sessionSolo()` : le 22/09 en solo (6 Super Fiesta, aucun match à objectif, aucune prise) —
 *     72 frags (Needler 13, Rayon de Sentinelle 10…), temps d'effet 804,9 s / 705 s pour 50 / 36
 *     frags, frags aux armes spéciales 91 / 101, vies 53 près / 5 seul (46 / 11 frags).
 * Aucune lecture de base : des données figées.
 */
import { EMPRISE_2209, HISTORY_2209 } from '@/features/squad/emprise/emprise.fixtures'
import { block0709, block2209 } from '@/features/squad/objectif/objectif.fixtures'
import type {
  FragClassEntry,
  SessionCompareEntry,
  SessionDetailMatchRow,
  SoloEmpriseBlock,
  SquadWeaponToolLine,
} from '@/lib/api/types'

import type { SessionColumnBlocks } from './sessionEmprise.logic'

export const ME_GAMERTAG = 'JGtm'

const classes = (by: Record<string, number>): FragClassEntry[] =>
  Object.entries(by).map(([c, kills]) => ({ class: c, kills, authoritative: true, roles: [] }))

const weapon = (label: string, cls: string, kills: number): SquadWeaponToolLine => ({
  kind: 'weapon',
  weapon_key: label,
  label,
  class: cls,
  kills_by_player: { [ME_GAMERTAG]: kills },
  total_squad: kills,
})

const tool = (kind: string, cls: string, kills: number): SquadWeaponToolLine => ({
  kind,
  class: cls,
  kills_by_player: { [ME_GAMERTAG]: kills },
  total_squad: kills,
})

function entry(label: string, frags: Record<string, number>, lines: SquadWeaponToolLine[]): SessionCompareEntry {
  const total = Object.values(frags).reduce((a, b) => a + b, 0)
  return {
    session_label: label,
    start_time: '2026-09-22T19:23:00Z',
    end_time: '2026-09-22T20:40:00Z',
    total_matches: 7,
    wins: 3,
    losses: 4,
    with_friends: true,
    frag_distribution: { total_kills: total, classes: classes(frags) },
    weapon_tools: { players: [ME_GAMERTAG], lines },
  } as unknown as SessionCompareEntry
}

/** Les lignes de match du 22/09 (score et dominance de l'historique de l'Escouade). */
export const MATCHES_2209: SessionDetailMatchRow[] = HISTORY_2209.map((h) => ({
  match_id: h.match_id,
  start_time: h.start_time ?? '',
  map_name: h.map_ui,
  mode_ui: h.mode_ui,
  outcome: h.outcome,
  score_label: h.score_label,
  ...(h.dominance_flag ? { dominance_flag: h.dominance_flag } : {}),
})) as unknown as SessionDetailMatchRow[]

export function session2209(): SessionColumnBlocks {
  return {
    entry: entry(
      '2026-09-22 21h23',
      { shoulder: 28, sidearm: 13, melee: 6, grenade: 5, heavy: 11, environmental: 2 },
      [
        weapon('BR75', 'shoulder', 22),
        weapon('MK50 Sidekick', 'sidearm', 13),
        weapon('M41 SPNKr', 'heavy', 11),
        tool('melee', 'melee', 6),
        weapon('Grenade frag', 'grenade', 2),
        weapon('Bandit EVO', 'shoulder', 2),
        weapon('S7 Sniper', 'heavy', 1),
        tool('unattributed', 'unattributed', 8),
      ],
    ),
    matches: MATCHES_2209,
    emprise: { ...EMPRISE_2209, maps: null } as SoloEmpriseBlock,
    lives: {
      near: { lives: 56, kills: 37 },
      alone: { lives: 18, kills: 4 },
      excluded_unlocated: 0,
      excluded_no_radar: 0,
      excluded_unpublishable: 0,
      matches_read: 6,
      matches_without_radar: 0,
    },
    formes: block2209(),
    emblemUrl: 'https://example.test/emblem.png',
  }
}

export function session0709(): SessionColumnBlocks {
  return {
    entry: entry('2026-09-07 21h26', { sidearm: 28, melee: 8, heavy: 3, grenade: 2, shoulder: 9 }, [
      weapon('MK50 Sidekick', 'sidearm', 28),
      tool('melee', 'melee', 8),
    ]),
    matches: [],
    formes: block0709(),
  }
}

const SOLO_IDS = ['z1', 'z2', 'z3', 'z4', 'z5', 'z6']
/** Heures UTC des six matchs (19:27 à 20:19, heure de Paris) et résultats du relevé §1 ; cartes anonymisées (quatre cartes, deux rejouées). */
const SOLO_STARTS = ['17:27', '17:36', '17:46', '17:57', '18:06', '18:19']
const SOLO_MAPS = ['Carte A', 'Carte B', 'Carte C', 'Carte D', 'Carte D', 'Carte B']
const SOLO_OUTCOMES = [2, 3, 3, 2, 2, 2]
const SOLO_PWK: [number, number][] = [[18, 15], [14, 17], [12, 20], [16, 16], [15, 18], [16, 15]]

export function sessionSolo(): SessionColumnBlocks {
  const emprise = {
    matches_total: 6,
    matches_measured: 6,
    players: [{ xuid: 'xj', gamertag: ME_GAMERTAG }],
    resources: [],
    objects: [],
    matches: SOLO_IDS.map((id, i) => ({
      match_id: id,
      has_film: true,
      team_known: true,
      tiers: 'measured',
      vehicles: 'not_measured',
      resources: [],
      power_weapon_kills: { us: SOLO_PWK[i][0], them: SOLO_PWK[i][1] },
    })),
    production: [
      {
        resource: 'powerup',
        kills: { us: 50, them: 36 },
        exposure: { kind: 'effect_ms', value: { us: 804_900, them: 705_000 }, kills: { us: 50, them: 36 } },
        yield_us: 50 / (804_900 / 60_000),
        yield_them: 36 / (705_000 / 60_000),
        relative_gap: 50 / (804_900 / 60_000) / (36 / (705_000 / 60_000)) - 1,
      },
      { resource: 'power_weapon', kills: { us: 91, them: 101 } },
    ],
    maps: null,
  } as unknown as SoloEmpriseBlock
  return {
    entry: entry('2026-09-22 19h27', { shoulder: 36, melee: 5, heavy: 16, sidearm: 11, grenade: 3, unattributed: 1 }, [
      weapon('Needler', 'shoulder', 13),
      weapon('Rayon de Sentinelle', 'heavy', 10),
      weapon('Ravageur', 'shoulder', 8),
      weapon('Fusil traqueur', 'shoulder', 7),
      weapon('Disrupteur', 'sidearm', 5),
      tool('melee', 'melee', 5),
    ]),
    matches: SOLO_IDS.map((id, i) => ({
      match_id: id,
      start_time: `2026-09-22T${SOLO_STARTS[i]}:00Z`,
      map_name: SOLO_MAPS[i],
      mode_ui: 'Mode témoin',
      outcome: SOLO_OUTCOMES[i],
    })) as unknown as SessionDetailMatchRow[],
    emprise,
    lives: {
      near: { lives: 53, kills: 46 },
      alone: { lives: 5, kills: 11 },
      excluded_unlocated: 1,
      excluded_no_radar: 0,
      excluded_unpublishable: 0,
      matches_read: 6,
      matches_without_radar: 0,
    },
    formes: { available: true, main_xuid: 'xj', matches_measured: 6, matches_total: 6, squad: [{ xuid: 'xj', gamertag: ME_GAMERTAG }], matches: [] },
  }
}
