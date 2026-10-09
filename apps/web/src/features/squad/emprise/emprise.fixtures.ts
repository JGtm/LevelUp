/**
 * emprise.fixtures.ts — la soirée témoin du 22/09/2026 (JGtm, Chocoboflor, Madina97294, sept
 * matchs) au format du contrat : bloc `squad_emprise` et historique de matchs. Chiffres de la
 * maquette de l'onglet (`.ai/V7.5/MAQUETTE_ONGLET_TACTIQUE_ESCOUADE_2026-09-26.html` : MATCHES,
 * ITEMS, ROLES, PISTES, FIL_SERIES), relevés en lecture seule dans la base réelle ; les mêmes
 * que le test d'intégration Go `TestSquadEmprise_TemoinsDu2209` (lot L4). Aucune lecture de
 * base : des données figées.
 *
 * Témoins : bonus 12 / 8, armes spéciales 23 / 29 ; fiches JGtm 3 bonus · 9 armes spéciales ;
 * bonus perdus 2 sur 12 et 2 sur 8 ; Starboard « Victoire 3–0 » + « Domination » ; Detachment
 * sans film.
 */
import type {
  SquadEmpriseBlock,
  SquadEmpriseEvening,
  SquadEmpriseHabit,
  SquadEmpriseMatch,
  SquadEmpriseObject,
  SquadEmpriseObjectShare,
  SquadMatchHistoryRow,
} from '@/lib/api/types'

export const XUID = { jgtm: 'x-jgtm', choco: 'x-choco', madina: 'x-madina' } as const
type Who = Partial<Record<'jgtm' | 'choco' | 'madina' | 'rest', number>>

/** Noms des armes du registre (langue de la requête : FR). */
export const WEAPONS: Record<string, string> = {
  a1000001: 'M41 SPNKr',
  a1000002: 'S7 Sniper',
  a1000003: 'Épée à énergie',
  a1000004: 'Needler',
  a1000005: 'Rayon de Sentinelle',
  a1000006: 'Fusil électrique',
  a1000007: 'Empaleur',
  b2000001: 'Déchiqueteur',
  b2000002: 'BR75',
  b2000003: 'VK78 Commando',
  b2000004: 'Carabine à impulsion',
  b2000005: 'Carabine Vestige',
  b2000006: 'MK50 Sidekick',
  b2000007: 'Ravageur',
  b2000008: 'Calcineur',
  b2000009: 'CQS48 Bulldog',
  b2000010: 'Bandit EVO',
  b2000011: 'Disrupteur',
  b2000012: 'Fusil traqueur',
  b2000013: 'MA5K Avenger',
}

const SPNKR = 'a1000001'
const SNIPER = 'a1000002'
const SWORD = 'a1000003'
const NEEDLER = 'a1000004'
const SENTINEL = 'a1000005'
const SHOCK = 'a1000006'
const SKEWER = 'a1000007'
const SHREDDER = 'b2000001'
const BR75 = 'b2000002'
const VK78 = 'b2000003'
const PULSE = 'b2000004'
const VESTIGE = 'b2000005'
const SIDEKICK = 'b2000006'
const RAVAGER = 'b2000007'
const CINDERSHOT = 'b2000008'
const BULLDOG = 'b2000009'
const BANDIT = 'b2000010'
const DISRUPTOR = 'b2000011'
const STALKER = 'b2000012'
const AVENGER = 'b2000013'
const CAMO = 'powerup_camo'
const OVERSHIELD = 'powerup_overshield'

interface ItemSpec {
  us: number
  them: number
  who?: Who
  pads?: number
  kept?: Who
  dropped?: Who
}

interface MatchSpec {
  id: string
  start: string
  map: string
  mode: string
  outcome: number
  score: string
  dom?: number
  pwk: [number, number]
  film: boolean
  items: Partial<Record<'powerup' | 'power_weapon' | 'rack', Record<string, ItemSpec>>>
}

const WIN = 2
const LOSS = 3

/** Les sept matchs (heures de Paris 21:23 → 22:29, en UTC). */
const MATCHES: MatchSpec[] = [
  {
    id: 'm1', start: '2026-09-22T19:23:00Z', map: 'Starboard', mode: 'Drapeau', outcome: WIN, score: '3–0', dom: 1, pwk: [0, 4], film: true,
    items: {
      powerup: { [OVERSHIELD]: { us: 5, them: 2, who: { choco: 2, jgtm: 2, madina: 1 }, pads: 10 } },
      power_weapon: { [SWORD]: { us: 0, them: 2 }, [SHOCK]: { us: 2, them: 2, who: { rest: 2 } } },
      rack: { [VK78]: { us: 2, them: 0, who: { jgtm: 1, rest: 1 } }, [BULLDOG]: { us: 0, them: 1 } },
    },
  },
  {
    id: 'm2', start: '2026-09-22T19:36:00Z', map: 'Curfew', mode: 'Assassin', outcome: WIN, score: '50–48', pwk: [12, 4], film: true,
    items: {
      powerup: { [CAMO]: { us: 4, them: 0, who: { madina: 3, jgtm: 1 }, pads: 7, dropped: { madina: 1 } } },
      power_weapon: {
        [SPNKR]: { us: 3, them: 0, who: { choco: 1, jgtm: 1, rest: 1 } },
        [SENTINEL]: { us: 1, them: 2, who: { rest: 1 } },
        [SKEWER]: { us: 2, them: 0, who: { rest: 2 } },
      },
      rack: {
        [SHREDDER]: { us: 2, them: 1, who: { madina: 2 } },
        [VK78]: { us: 3, them: 0, who: { rest: 3 } },
        [PULSE]: { us: 1, them: 3, who: { rest: 1 } },
        [VESTIGE]: { us: 1, them: 1, who: { jgtm: 1 } },
      },
    },
  },
  {
    id: 'm3', start: '2026-09-22T19:46:00Z', map: 'Origin', mode: 'Drapeau', outcome: LOSS, score: '1–3', pwk: [8, 11], film: true,
    items: {
      power_weapon: { [SPNKR]: { us: 2, them: 6, who: { jgtm: 2 } }, [SNIPER]: { us: 3, them: 1, who: { jgtm: 2, madina: 1 } } },
      rack: { [BR75]: { us: 0, them: 7 }, [SIDEKICK]: { us: 2, them: 0, who: { jgtm: 2 } } },
    },
  },
  {
    id: 'm4', start: '2026-09-22T19:59:00Z', map: 'Solution', mode: 'Assassin', outcome: WIN, score: '50–17', dom: 1, pwk: [5, 0], film: true,
    items: {
      power_weapon: { [SNIPER]: { us: 2, them: 0, who: { madina: 2 } } },
      rack: {
        [SHREDDER]: { us: 4, them: 1, who: { choco: 2, madina: 2 } },
        [RAVAGER]: { us: 0, them: 1 },
        [BANDIT]: { us: 1, them: 0, who: { rest: 1 } },
        [DISRUPTOR]: { us: 0, them: 1 },
      },
    },
  },
  { id: 'm5', start: '2026-09-22T20:08:00Z', map: 'Detachment', mode: 'Assassin', outcome: LOSS, score: '45–50', pwk: [9, 13], film: false, items: {} },
  {
    id: 'm6', start: '2026-09-22T20:19:00Z', map: 'Shogun', mode: 'Assassin', outcome: LOSS, score: '37–50', pwk: [4, 10], film: true,
    items: {
      powerup: { [CAMO]: { us: 2, them: 3, who: { choco: 1, madina: 1 }, pads: 8, kept: { choco: 1 } } },
      power_weapon: {
        [SPNKR]: { us: 1, them: 3, who: { rest: 1 } },
        [SWORD]: { us: 1, them: 2, who: { rest: 1 } },
        [SENTINEL]: { us: 1, them: 0, who: { choco: 1 } },
      },
      rack: {
        [SHREDDER]: { us: 1, them: 0, who: { choco: 1 } },
        [VK78]: { us: 0, them: 1 },
        [PULSE]: { us: 2, them: 0, who: { rest: 2 } },
        [RAVAGER]: { us: 0, them: 1 },
        [CINDERSHOT]: { us: 1, them: 1, who: { rest: 1 } },
        [STALKER]: { us: 0, them: 1 },
        [AVENGER]: { us: 0, them: 1 },
      },
    },
  },
  {
    id: 'm7', start: '2026-09-22T20:29:00Z', map: 'Catalyst', mode: 'Assassin', outcome: LOSS, score: '33–50', pwk: [9, 12], film: true,
    items: {
      powerup: { [OVERSHIELD]: { us: 1, them: 3, who: { rest: 1 }, pads: 8 } },
      power_weapon: {
        [SPNKR]: { us: 3, them: 4, who: { jgtm: 2, choco: 1 } },
        [SNIPER]: { us: 0, them: 2 },
        [SWORD]: { us: 1, them: 1, who: { jgtm: 1 } },
        [NEEDLER]: { us: 1, them: 4, who: { jgtm: 1 } },
      },
      rack: { [VESTIGE]: { us: 0, them: 1 }, [SIDEKICK]: { us: 1, them: 0, who: { madina: 1 } } },
    },
  },
]

const PLAYERS = [
  { xuid: XUID.jgtm, gamertag: 'JGtm' },
  { xuid: XUID.choco, gamertag: 'Chocoboflor' },
  { xuid: XUID.madina, gamertag: 'Madina97294' },
]
const OWNERS = ['jgtm', 'choco', 'madina'] as const

function shares(resource: string, spec: ItemSpec): SquadEmpriseObjectShare[] {
  const bonus = resource === 'powerup'
  const part = (k: keyof Who, xuid?: string): SquadEmpriseObjectShare => ({
    ...(xuid ? { xuid } : {}),
    taken: spec.who?.[k] ?? 0,
    ...(bonus ? { kept: spec.kept?.[k] ?? 0, dropped: spec.dropped?.[k] ?? 0 } : {}),
  })
  return [...OWNERS.map((k) => part(k, XUID[k])), part('rest')]
}

function object(resource: string, key: string, spec: ItemSpec): SquadEmpriseObject {
  return {
    resource,
    key,
    ...(resource === 'powerup' ? {} : { weapon_key: `w_${key}`, label: WEAPONS[key] }),
    ...(spec.pads != null ? { pads_emptied: spec.pads } : {}),
    taken: { us: spec.us, them: spec.them },
    squad: shares(resource, spec),
  }
}

function match(spec: MatchSpec): SquadEmpriseMatch {
  const resources = (['powerup', 'power_weapon', 'rack'] as const).flatMap((resource) => {
    const items = spec.items[resource]
    if (!items) return []
    const objects = Object.entries(items).map(([key, it]) => object(resource, key, it))
    const us = objects.reduce((a, o) => a + o.taken.us, 0)
    const them = objects.reduce((a, o) => a + o.taken.them, 0)
    return [{ resource, taken: { us, them }, objects }]
  })
  return {
    match_id: spec.id,
    has_film: spec.film,
    team_known: true,
    ...(spec.film ? { tiers: 'measured' } : {}),
    resources,
    power_weapon_kills: { us: spec.pwk[0], them: spec.pwk[1] },
  }
}

const RANK: Record<string, number> = { powerup: 0, power_weapon: 1, rack: 2 }

/** La soirée objet par objet : la somme des matchs, dans l'ordre du serveur. */
function eveningObjects(matches: SquadEmpriseMatch[]): SquadEmpriseObject[] {
  const acc = new Map<string, SquadEmpriseObject>()
  for (const m of matches) {
    for (const r of m.resources ?? []) {
      for (const o of r.objects ?? []) {
        const cur = acc.get(o.key)
        if (!cur) {
          acc.set(o.key, { ...o, taken: { ...o.taken }, squad: (o.squad ?? []).map((s) => ({ ...s })) })
          continue
        }
        cur.taken.us += o.taken.us
        cur.taken.them += o.taken.them
        if (o.pads_emptied != null) cur.pads_emptied = (cur.pads_emptied ?? 0) + o.pads_emptied
        ;(o.squad ?? []).forEach((s, i) => {
          const c = cur.squad![i]
          c.taken += s.taken
          if (s.kept != null) c.kept = (c.kept ?? 0) + s.kept
          if (s.dropped != null) c.dropped = (c.dropped ?? 0) + s.dropped
        })
      }
    }
  }
  return [...acc.values()].sort(
    (a, b) =>
      RANK[a.resource] - RANK[b.resource] ||
      b.taken.us - a.taken.us ||
      b.taken.us + b.taken.them - (a.taken.us + a.taken.them) ||
      a.key.localeCompare(b.key),
  )
}

const EMPRISE_MATCHES = MATCHES.map(match)

function evening(label: string, start: string, bonus: [number, number], power: [number, number]): SquadEmpriseEvening {
  const share = (resource: string, [us, them]: [number, number]) => ({ resource, taken: { us, them }, share: us / (us + them) })
  return {
    session_label: label,
    start_time: start,
    match_count: 6,
    measured_matches: 6,
    comparable: true,
    families: ['Assassin', 'Drapeau'],
    shares: [share('powerup', bonus), share('power_weapon', power)],
  }
}

/**
 * « Par rapport à d'habitude » (maquette, `renderHabChart('habPrises', …)`) : cinq soirées
 * précédentes filmées puis ce soir, familles Assassin et Drapeau. La maquette ne publie que les
 * parts ; les comptes des soirées précédentes sont CHOISIS pour les reproduire (58,3 % = 7 sur
 * 12, …). Ce soir : les prises du bilan, 12 / 8 (60 %) et 23 / 29 (44,2 %).
 */
export const HABIT_2209: SquadEmpriseHabit = {
  families: ['Assassin', 'Drapeau'],
  current: evening('22/09', '2026-09-22T19:23:00Z', [12, 8], [23, 29]),
  previous: [
    evening('28/07', '2026-07-28T19:00:00Z', [7, 5], [10, 10]),
    evening('31/07', '2026-07-31T19:00:00Z', [28, 15], [20, 17]),
    evening('27/08', '2026-08-27T19:00:00Z', [6, 2], [11, 12]),
    evening('01/09', '2026-09-01T19:00:00Z', [13, 11], [23, 17]),
    evening('07/09', '2026-09-07T19:00:00Z', [6, 4], [20, 21]),
  ],
}

/** Le bloc `squad_emprise` de la soirée du 22/09. */
export const EMPRISE_2209: SquadEmpriseBlock = {
  matches_total: 7,
  matches_measured: 6,
  players: PLAYERS,
  resources: [
    {
      resource: 'powerup',
      taken: { us: 12, them: 8 },
      matches_measured: 6,
      outcomes: {
        us: { taken: 12, used: 10, kept: 1, dropped: 1 },
        them: { taken: 8, used: 6, kept: 1, dropped: 1 },
      },
    },
    { resource: 'power_weapon', taken: { us: 23, them: 29 }, matches_measured: 6 },
  ],
  objects: eveningObjects(EMPRISE_MATCHES),
  matches: EMPRISE_MATCHES,
  production: [
    // Maquette (PRODUCTIVITE) et L4.6 : frags pendant l'effet 8 / 5, temps d'effet 2 min 39 /
    // 1 min 53 ; rendements calculés comme le Go (frags par minute d'effet).
    {
      resource: 'powerup',
      kills: { us: 8, them: 5 },
      exposure: { kind: 'effect_ms', value: { us: 159_000, them: 113_000 }, kills: { us: 8, them: 5 } },
      yield_us: 8 / (159_000 / 60_000),
      yield_them: 5 / (113_000 / 60_000),
      relative_gap: 8 / (159_000 / 60_000) / (5 / (113_000 / 60_000)) - 1,
    },
    // Frags aux armes spéciales 47 / 54 (feuille, sept matchs) ; prises 23 / 29 ; rendement sur
    // les seuls matchs dont les prises sont mesurées (règle L4 : Detachment en sort, 38 / 41).
    {
      resource: 'power_weapon',
      kills: { us: 47, them: 54 },
      exposure: { kind: 'pickups', value: { us: 23, them: 29 }, kills: { us: 38, them: 41 } },
      yield_us: 38 / 23,
      yield_them: 41 / 29,
      relative_gap: 38 / 23 / (41 / 29) - 1,
    },
  ],
  habit: HABIT_2209,
}

/** L'historique de matchs de la page (résultat, score, dominance, carte, mode). */
export const HISTORY_2209: SquadMatchHistoryRow[] = MATCHES.map((m) => ({
  match_id: m.id,
  start_time: m.start,
  map_ui: m.map,
  mode_ui: m.mode,
  outcome: m.outcome,
  score_label: m.score,
  ...(m.dom ? { dominance_flag: m.dom } : {}),
  kills: 0,
  deaths: 0,
  assists: 0,
}))
