/**
 * instantFixture.ts — UN MATCH ÉCRIT À LA MAIN POUR LA FICHE « À L'INSTANT » : assez de données pour que chaque
 * section existe, un joueur dont tout est lu (Alpha), un joueur dont les tirs ne sont pas lus (Echo) et un bot.
 *
 * Pas de 100 ms, 400 images. Alpha vit de 0 à 150, meurt, revient de 170 à 400 ; il fait quatre éliminations (à
 * 2, 6, 10 et 20 s), gagne trois médailles, perd son bouclier trois fois, lance trois grenades, ramasse trois armes
 * et deux grenades, sprinte deux fois et saute trois fois. Son bilan de tirs dit 35 tirs du jeu pour 7 écrits par
 * le film : les tirs estimés sont cinq fois les tirs écrits. Les points par action de chaque joueur égalent le score
 * de sa feuille : Echo (400) devance Alpha (300) devant le bot (100).
 *
 * Alpha, au dernier instant : 4 éliminations dont 1 à la tête sur 2 mesurées, 3 médailles (Doublé ×2, Parfait).
 */
import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'
import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { WeaponIconsByKey } from '../model/killSourceIcons'
import type { MedalEvent, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import { scoreboardRow } from './scoreboardRow'
import { testReplayDoc } from './testDoc'

/** La dernière image du match de la fixture. */
export const INSTANT_LAST_FRAME = 399

type Doc = ReplayDocument

const point = (t: number, sh: number, hp: number) => ({ t, x: 0, y: 0, sh, hp })

const action = (t: number, xuid: string, points: number) => ({ t, xuid, name: 'killPersonalScore', points })

const shot = (t: number, slot: number, w: string) => ({ t, slot, x: 0, y: 0, w })
const hit = (t: number, shooter: string, victim: string) => ({ t, shooter, victim, effect: 'e', w: '0xAAAA' })
const damage = (t: number, attacker: string, victim: string, shield: number, body: number, w: string) => ({
  t, attacker, victim, effect: 'e', shield, body, w,
})

const grenade = (t: number, rank: number, slot: number) => ({ t, slot, i: 0, x: 0, y: 0, rank, s: 'throw' })
const pickup = (t: number, slot: number, kind: string, extra: Record<string, unknown> = {}) => ({
  t, slot, w: '000012ab', kind, class: 0, xuid: 'A', ...extra,
})

const sprint = (slot: number, t0: number, t1: number) => ({ slot, kind: 'sprint', t0, t1 })
const jump = (slot: number, t0: number, t1: number) => ({ slot, kind: 'jumpDerived', t0, t1 })

/** Le document de la fixture ; un test en surcharge ce qu'il observe. */
export function instantDoc(over: Partial<Doc> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    frameCount: 400,
    roster: [
      { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
      { xuid: 'E', filmIndex: 1, name: 'Echo', team: 1 },
      { xuid: '', bot: true, filmIndex: 2, name: 'Marc', team: 1 },
    ],
    tracks: [
      {
        slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 150,
        points: [point(0, 1, 1), point(40, 0, 0.6), point(60, 1, 1), point(120, 0, 0.4), point(150, 0, 0)],
      },
      { slot: 513, team: 0, xuid: 'A', startFrame: 170, endFrame: 400, points: [point(170, 1, 1), point(300, 0, 0.5)] },
      { slot: 514, team: 1, xuid: 'E', startFrame: 0, endFrame: 400, points: [point(0, 1, 1)] },
      { slot: 515, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 400, points: [point(0, 1, 1)] },
    ],
    scoreTimeline: {
      teams: null,
      players: [
        {
          xuid: 'A',
          score: { rounds: null, total: [{ t: 20, v: 100 }, { t: 60, v: 200 }, { t: 100, v: 300 }] },
          kills: { rounds: null, total: [{ t: 20, v: 1 }, { t: 60, v: 2 }, { t: 100, v: 3 }, { t: 200, v: 4 }] },
          deaths: { rounds: null, total: [{ t: 150, v: 1 }] },
          assists: { rounds: null, total: [{ t: 60, v: 1 }] },
          stats: { headshots: { rounds: null, total: [{ t: 20, v: 1 }, { t: 100, v: 2 }] } },
        },
      ],
    } as never,
    coverage: {
      scoreActions: { complete: true },
      stances: { scanned: true, absent: false, jumpEpisodes: 10 },
      medals: { decoded: 0, published: 0, noOrigin: 0, beforeOrigin: 0, afterEnd: 0, unknownName: 0 },
    } as never,
    scoreActions: [
      action(20, 'A', 100), action(60, 'A', 100), action(100, 'A', 100),
      action(25, 'E', 200), action(50, 'E', 200), action(40, 'bid(7.0)', 100),
    ],
    weaponLabels: { '0xAAAA': { fr: "Fusil d'assaut", en: 'Assault rifle' }, '0xBBBB': { fr: 'SPNKr', en: 'SPNKr' } },
    grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }, { fr: 'Plasma', en: 'Plasma' }],
    abilityLabels: { '6': { fr: 'Capteur de menaces', en: 'Threat sensor', family: 'sensor' } },
    equipmentChanges: [
      { t: 30, slot: 512, kind: 'taken', r: 6, from: -1 },
      { t: 80, slot: 512, kind: 'spent', r: -1, from: 6 },
    ],
    shotStats: [
      { xuid: 'A', shots: 35, shotEvents: 7, messages: 0, hits: 12, damageDealt: 800, damageTaken: 500, coverage: 0.9, duplicates: 1, sufficient: true },
      { xuid: 'E', shots: 10, shotEvents: 2, messages: 0, hits: 0, damageDealt: 0, damageTaken: 0, coverage: 0.2, duplicates: 1, sufficient: false },
    ],
    shots: [
      shot(10, 512, '0xAAAA'), shot(20, 512, '0xAAAA'), shot(30, 512, '0xAAAA'), shot(40, 512, '0xAAAA'),
      shot(90, 512, '0xBBBB'), shot(200, 513, '0xAAAA'), shot(210, 513, '0xAAAA'), shot(15, 514, '0xAAAA'),
    ],
    hits: [hit(11, 'A', 'E'), hit(21, 'A', 'E'), hit(31, 'A', 'E'), hit(41, 'A', 'E'), hit(91, 'A', 'E'), hit(201, 'A', 'E')],
    damage: [
      damage(11, 'A', 'E', 20, 0, '0xAAAA'),
      damage(41, 'A', 'E', 30, 10, '0xAAAA'),
      damage(91, 'A', 'E', 0, 100, '0xBBBB'),
      damage(12, 'E', 'A', 25, 0, '0xAAAA'),
    ],
    grenades: [grenade(10, 0, 512), grenade(50, 0, 512), grenade(90, 1, 512), grenade(210, 0, 513)],
    pickups: [
      pickup(20, 512, 'weapon'), pickup(40, 512, 'weapon'), pickup(220, 513, 'weapon'),
      pickup(60, 512, 'grenade', { class: 2 }), pickup(230, 513, 'grenade', { class: 2 }),
    ],
    stances: [sprint(512, 10, 30), sprint(513, 180, 200), jump(512, 15, 18), jump(512, 100, 102), jump(513, 190, 192)],
    medalLabels: {
      'Double Kill': { fr: 'Doublé', en: 'Double Kill', img: '/medals/double.png', difficulty: 'Normal' },
      Perfect: {
        fr: 'Parfait', en: 'Perfect', difficulty: 'Legendary',
        sprite: { sheet: '/medals/sprites.png', left: 74, top: 148, width: 74, height: 74 },
      },
    },
    ...over,
  } as Partial<Doc>)
}

/** La feuille du match : Alpha (300), Echo (400) et le bot (100), deux camps. */
export function instantSheet(): MatchScoreboardRow[] {
  return [
    scoreboardRow('A', 'Alpha', 't0', {
      team_name: 'Eagle', score: 300, kills: 7, deaths: 3, assists: 2,
      accuracy: 41.2, shots_hit: 118, shots_fired: 288, damage_dealt: 2140, damage_taken: 1610, headshot_kills: 6,
    }),
    scoreboardRow('E', 'Echo', 't1', { team_name: 'Cobra', score: 400, kills: 5, deaths: 4, assists: 0 }),
    scoreboardRow('bid(7.0)', 'Marc', 't1', { team_name: 'Cobra', is_bot: true, score: 100, kills: 0, deaths: 2, assists: 0 }),
  ]
}

/** Le lobby de la fixture, joint à sa feuille (`instantSheet()` par défaut). */
export function instantPlayers(doc = instantDoc(), sheet: MatchScoreboardRow[] = instantSheet()): ReplayPlayer[] {
  return buildPlayers(doc, sheet)
}

/** Un joueur du lobby par sa clé de roster (`A`, `E`, `bot:Marc`). */
export function instantPlayer(players: readonly ReplayPlayer[], xuid: string): ReplayPlayer {
  const p = players.find((x) => x.xuid === xuid)
  if (!p) throw new Error(`joueur absent de la fixture : ${xuid}`)
  return p
}

const medal = (name: string, label: string, xuid: string, tMs: number, imageUrl = ''): MedalEvent => ({
  tMs, xuid, gamertag: '', name, label, description: '', imageUrl,
})

function killEntry(
  replayMs: number,
  xuid: string,
  weaponKey: string,
  weaponClass: ReplayKill['weaponClass'],
  extra: { label?: string; headshot?: boolean; medals?: MedalEvent[] } = {},
): ReplayFeedEntry {
  const kill = {
    xuid, replayMs, weaponKey, weaponClass, weaponLabel: extra.label ?? '', headshot: extra.headshot,
    medals: extra.medals ?? [],
  } as unknown as ReplayKill
  return { key: `k${xuid}${replayMs}`, replayMs, kill, medal: null, death: null }
}

const DOUBLE = (tMs: number) => medal('Double Kill', 'Doublé', 'A', tMs, '/medals/double.png')

/**
 * Le fil de la fixture : quatre éliminations d'Alpha (une à la tête, une non, deux dont le coup fatal n'est pas
 * lu), une d'Echo, et les médailles d'Alpha — Doublé à 6 s et à 20 s, Parfait seul à 6 s, une médaille inconnue
 * de la table à 20 s.
 */
export function instantFeed(): ReplayFeedEntry[] {
  const parfait: ReplayFeedEntry = {
    key: 'medaille-parfait', replayMs: 6000, kill: null, medal: medal('Perfect', 'Parfait', 'A', 6000), death: null,
  }
  const inconnue: ReplayFeedEntry = {
    key: 'medaille-inconnue', replayMs: 20_000, kill: null, medal: medal('Mystery Medal', '', 'A', 20_000), death: null,
  }
  return [
    killEntry(2000, 'A', 'hinf_ma40_ar', 'shoulder', { label: "Fusil d'assaut", headshot: true }),
    killEntry(3000, 'E', 'hinf_ma40_ar', 'shoulder', { label: "Fusil d'assaut", headshot: false }),
    killEntry(6000, 'A', 'hinf_ma40_ar', 'shoulder', { label: "Fusil d'assaut", headshot: false, medals: [DOUBLE(6000)] }),
    parfait,
    killEntry(10_000, 'A', 'hinf_spnkr', 'heavy', { label: 'SPNKr' }),
    killEntry(20_000, 'A', 'hinf_frag_grenade', 'grenade', { medals: [DOUBLE(20_000)] }),
    inconnue,
  ]
}

/** Le référentiel des pictogrammes d'arme : il nomme la grenade que le fil ne nomme pas. */
export const INSTANT_ICONS: WeaponIconsByKey = new Map([
  ['hinf_frag_grenade', { imageUrl: '/g.png', tinted: true, label: 'Fragmentation' }],
])
