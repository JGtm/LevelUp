/**
 * matchEmprise.fixtures.ts — LES DEUX TÉMOINS de la Vue match aux formes de l'Emprise, réduits à ce
 * que lisent les cartes (`.ai/V7.5/MESURES_MATCHVIEW_2026-10-06.md`, §3) :
 *   - `STARBOARD` (22/09, Drapeau à Starboard, 4 contre 4, journal des morts publiable) : bonus 5–2
 *     (Surbouclier, 10 socles vidés), armes spéciales 0–2 (Épée à énergie), râteliers 4–3 (Fusil
 *     électrique 2–2, VK78 Commando 2–0, CQS48 Bulldog 0–1) ; fiches JGtm, XL JACOB, Madina97294,
 *     Chocoboflor ; temps d'effet 71,6 s contre 0 ; vies au seuil de 18 m.
 *   - `FLOOD_GULCH` (24/07, BTB, 12 contre 12, journal non publiable) : armes spéciales 19–11 (S7
 *     Sniper 11–8, SPNKr à combustible 8–3), râteliers 29–40, 31 prises non identifiées (19 / 12),
 *     aucun bonus pris publié mais 166,7 s d'effet contre 216,9 s.
 */
import type { MatchEmpriseBlock, MatchLivesNearTeammate, MatchScoreboardRow, SquadEmpriseObject } from '@/lib/api/types'

const c = (us: number, them: number) => ({ us, them })

export const XUID = { jgtm: 'x-jgtm', jacob: 'x-jacob', madina: 'x-madina', choco: 'x-choco' } as const

function obj(
  resource: string,
  key: string,
  label: string,
  taken: { us: number; them: number },
  squad: { xuid: string; taken: number }[],
  extra: Partial<SquadEmpriseObject> = {},
): SquadEmpriseObject {
  return { resource, key, label, taken, squad, ...extra }
}

const STARBOARD_OBJECTS: SquadEmpriseObject[] = [
  obj('powerup', 'overshield', 'Surbouclier', c(5, 2), [
    { xuid: XUID.jgtm, taken: 2 },
    { xuid: XUID.madina, taken: 1 },
    { xuid: XUID.choco, taken: 2 },
  ], { pads_emptied: 10 }),
  obj('power_weapon', 'energy_sword', 'Épée à énergie', c(0, 2), []),
  obj('rack', 'shock_rifle', 'Fusil électrique', c(2, 2), [{ xuid: XUID.jacob, taken: 2 }]),
  obj('rack', 'vk78', 'VK78 Commando', c(2, 0), [
    { xuid: XUID.jgtm, taken: 1 },
    { xuid: XUID.jacob, taken: 1 },
  ]),
  obj('rack', 'bulldog', 'CQS48 Bulldog', c(0, 1), []),
]

const byResource = (objects: SquadEmpriseObject[], resource: string) => objects.filter((o) => o.resource === resource)
const sum = (objects: SquadEmpriseObject[]) => objects.reduce((a, o) => c(a.us + o.taken.us, a.them + o.taken.them), c(0, 0))
const resourceOf = (objects: SquadEmpriseObject[], resource: string) => ({
  resource,
  taken: sum(byResource(objects, resource)),
  objects: byResource(objects, resource),
})

export const STARBOARD: MatchEmpriseBlock = {
  matches_total: 1,
  matches_measured: 1,
  kill_journal_publishable: true,
  players: [
    { xuid: XUID.jgtm, gamertag: 'JGtm' },
    { xuid: XUID.jacob, gamertag: 'XL JACOB' },
    { xuid: XUID.madina, gamertag: 'Madina97294' },
    { xuid: XUID.choco, gamertag: 'Chocoboflor' },
  ],
  objects: STARBOARD_OBJECTS,
  resources: [
    { resource: 'powerup', taken: c(5, 2), matches_measured: 1 },
    { resource: 'power_weapon', taken: c(0, 2), matches_measured: 1 },
    { resource: 'rack', taken: c(4, 3), matches_measured: 1 },
  ],
  matches: [
    {
      match_id: 'ab526724',
      has_film: true,
      team_known: true,
      tiers: 'measured',
      resources: ['powerup', 'power_weapon', 'rack'].map((r) => resourceOf(STARBOARD_OBJECTS, r)),
    },
  ],
  production: [
    // Rendements : l'adversaire n'a aucun temps d'effet, l'équipe aucune prise — aucun écart publié.
    { resource: 'powerup', kills: c(2, 0), exposure: { kind: 'effect_ms', value: c(71600, 0), kills: c(2, 0) }, yield_us: 2 / (71600 / 60000) },
    { resource: 'power_weapon', kills: c(0, 4), exposure: { kind: 'pickups', value: c(0, 2), kills: c(0, 4) }, yield_them: 2 },
  ],
}

const FLOOD_OBJECTS: SquadEmpriseObject[] = [
  obj('power_weapon', 's7', 'S7 Sniper', c(11, 8), [{ xuid: XUID.jgtm, taken: 4 }]),
  obj('power_weapon', 'spnkr_fuel', 'SPNKr à combustible', c(8, 3), [{ xuid: XUID.jgtm, taken: 3 }]),
  obj('rack', 'br75', 'BR75', c(8, 17), []),
  obj('rack', 'disruptor', 'Disrupteur', c(14, 4), [{ xuid: XUID.jgtm, taken: 1 }]),
  obj('rack', 'bulldog', 'CQS48 Bulldog', c(2, 12), []),
  obj('rack', 'ma5k', 'MA5K Avenger', c(5, 2), []),
  obj('rack', 'pulse', 'Carabine à impulsion', c(0, 3), []),
  obj('rack', 'plasma_pistol', 'Pistolet à plasma', c(0, 2), []),
]

export const FLOOD_GULCH: MatchEmpriseBlock = {
  matches_total: 1,
  matches_measured: 1,
  kill_journal_publishable: false,
  players: [{ xuid: XUID.jgtm, gamertag: 'JGtm' }],
  objects: FLOOD_OBJECTS,
  resources: [
    { resource: 'power_weapon', taken: c(19, 11), matches_measured: 1 },
    { resource: 'rack', taken: c(29, 40), matches_measured: 1 },
  ],
  matches: [
    {
      match_id: '4f77afc1',
      has_film: true,
      team_known: true,
      tiers: 'measured',
      unclassified_pickups: c(19, 12),
      resources: ['power_weapon', 'rack'].map((r) => resourceOf(FLOOD_OBJECTS, r)),
    },
  ],
  production: [
    // Journal non publiable : aucun frag pendant l'effet lu, rendement de l'adversaire nul, pas d'écart.
    { resource: 'powerup', kills: c(0, 0), exposure: { kind: 'effect_ms', value: c(166700, 216900), kills: c(0, 0) }, yield_us: 0, yield_them: 0 },
    // Armes spéciales : frags de la feuille sur les prises (23 / 19 contre 38 / 11).
    {
      resource: 'power_weapon',
      kills: c(23, 38),
      exposure: { kind: 'pickups', value: c(19, 11), kills: c(23, 38) },
      yield_us: 23 / 19,
      yield_them: 38 / 11,
      relative_gap: 23 / 19 / (38 / 11) - 1,
    },
  ],
}

const side = (lives: number, kills: number) => ({ lives, kills })
const life = (xuid: string, near: [number, number], alone: [number, number]) => ({
  xuid,
  near: side(...near),
  alone: side(...alone),
  excluded_unlocated: 0,
  excluded_no_radar: 0,
  excluded_unpublishable: 0,
  matches_read: 1,
  matches_without_radar: 0,
})

/** Les vies du 22/09 au seuil de 18 m (MESURES §3, I) — l'ordre du repo, pas celui des fiches. */
export const STARBOARD_LIVES: MatchLivesNearTeammate = {
  players: [
    life(XUID.choco, [13, 10], [2, 1]),
    life(XUID.jgtm, [11, 8], [2, 0]),
    life(XUID.madina, [14, 16], [3, 6]),
    life(XUID.jacob, [9, 30], [1, 1]),
  ],
}

/** Le tableau du 22/09 : quatre contre quatre, tous présents à la fin. */
export const STARBOARD_SCOREBOARD = [
  { xuid: XUID.jgtm, gamertag: 'JGtm', team_side: 't0', is_me: true },
  { xuid: XUID.jacob, gamertag: 'XL JACOB', team_side: 't0' },
  { xuid: XUID.madina, gamertag: 'Madina97294', team_side: 't0' },
  { xuid: XUID.choco, gamertag: 'Chocoboflor', team_side: 't0' },
  { xuid: 'x-o1', gamertag: 'UNSCSparton11', team_side: 't1' },
  { xuid: 'x-o2', gamertag: 'NOaimAssist5833', team_side: 't1' },
  { xuid: 'x-o3', gamertag: 'Taiko900BPM', team_side: 't1' },
  { xuid: 'x-o4', gamertag: 'Rianbolis', team_side: 't1' },
] as unknown as MatchScoreboardRow[]
