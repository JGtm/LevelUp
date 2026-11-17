/**
 * playerEquipmentStats.test.ts — L'ÉQUIPEMENT D'UN JOUEUR DU DÉBUT DU MATCH À L'IMAGE LUE.
 *
 * Trois familles de preuves. Des matchs écrits à la main (le témoin partagé des tests du bilan, pas de
 * 100 ms) donnent les comptes exacts : début de match, pris puis utilisé, pris puis lâché en mourant, pris
 * et encore porté, deux familles, famille sans colonne, bot, slot recyclé. Un cliquet vérifie que les
 * comptes ne reculent jamais d'une image à l'autre. Enfin, à la dernière image, chaque joueur égale la
 * colonne de la grille du match (`usageColumnGroups`, le vrai code des colonnes) famille par famille, sur un
 * jeu de scénarios qui reprend ceux des tests du bilan et sur les huit documents publiés par Go
 * (`fixtures/go/`).
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'
import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import { normalizeReplayDocument, type ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { REPLAY_TEXT } from '../i18n/i18n'
import { pose, SB, temoin, vie } from '../test/equipmentUsageFixtures'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { scoreboardRow } from '../test/scoreboardRow'
import { usageColumnGroups } from './equipmentUsageColumns'
import { buildEquipmentUsage } from './equipmentUsageLogic'
import {
  equipmentSeriesOf,
  playerEquipmentStatsAt,
  readEquipmentSeries,
  type EquipmentFamilyStats,
} from './playerEquipmentStats'

const LABELS = {
  '1': { fr: 'Camouflage', en: 'Active camo', family: 'powerup_camo' },
  '5': { fr: 'Mur de protection', en: 'Drop wall', family: 'wall' },
  '6': { fr: 'Capteur de menaces', en: 'Threat sensor', family: 'sensor' },
  '7': { fr: 'Traqueur de menaces', en: 'Threat seeker', family: 'threat_seeker' },
  '10': { fr: 'Répulseur', en: 'Repulsor', family: 'repulsor' },
  '12': { fr: 'Grappin', en: 'Grappleshot', family: 'grapple' },
}

/** Les champs de document que ces tests écrivent : la table de familles, les changements d'équipement, les poses. */
function docWith(over: Record<string, unknown>): ReplayDocumentReady {
  return temoin({ abilityLabels: LABELS, ...over } as unknown as Partial<ReplayDocument>)
}

const taken = (t: number, slot: number, r: number) => ({ t, slot, kind: 'taken', r, from: -1 })
const spent = (t: number, slot: number, from: number) => ({ t, slot, kind: 'spent', r: -1, from })
/** Une pose d'équipement datée : `pose` du témoin partagé la date à 10. */
const poseAt = (t0: number, family: string, origin: string, owner: number, id?: string) => ({
  ...pose(family, origin, owner, id),
  t0,
  t1: t0 + 10,
})

/** Le lecteur d'un document : `lire(xuid, frame)` répond pour le joueur de ce xuid de roster. */
function lecteur(doc: ReplayDocumentReady, sheet: MatchScoreboardRow[] = SB) {
  const players = buildPlayers(doc, sheet)
  const index = readEquipmentSeries(doc, players)
  return (xuid: string, frame: number) => {
    const player = players.find((p) => p.xuid === xuid) as ReplayPlayer
    return playerEquipmentStatsAt(equipmentSeriesOf(index, player), frame)
  }
}

const famille = (family: string, pris: number, used: number, dropped: number): EquipmentFamilyStats => ({
  family,
  taken: pris,
  used,
  dropped,
})

describe('playerEquipmentStats — début de match', () => {
  const lire = lecteur(docWith({ equipmentChanges: [taken(5, 1, 6)] }))

  it('avant le premier geste, aucun objet pris : une liste vide mesurée, pas une absence', () => {
    expect(lire('a1', 0)).toEqual({ families: [] })
    expect(lire('a1', -1)).toEqual({ families: [] })
    expect(lire('a1', 4)).toEqual({ families: [] })
  })

  it('un joueur qui ne touche à rien a une liste vide à toute image', () => {
    expect(lire('b1', 0)).toEqual({ families: [] })
    expect(lire('b1', 10_000)).toEqual({ families: [] })
  })

  it('le geste daté à l’image lue compte (image comprise)', () => {
    expect(lire('a1', 5)).toEqual({ families: [famille('sensor', 1, 0, 0)] })
  })
})

describe('playerEquipmentStats — pris, utilisé, perdu', () => {
  it('un capteur pris puis consommé : utilisé à l’image de la consommation, pas avant', () => {
    const lire = lecteur(docWith({ equipmentChanges: [taken(5, 1, 6), spent(8, 1, 6)] }))
    expect(lire('a1', 7)).toEqual({ families: [famille('sensor', 1, 0, 0)] })
    expect(lire('a1', 8)).toEqual({ families: [famille('sensor', 1, 1, 0)] })
    expect(lire('a1', 100)).toEqual({ families: [famille('sensor', 1, 1, 0)] })
  })

  it('un mur pris puis lâché en mourant : perdu à l’image du lâcher', () => {
    const lire = lecteur(
      docWith({
        equipmentChanges: [taken(5, 1, 5)],
        equipmentPlacements: [poseAt(30, 'wall', 'dropped', 1)],
      }),
    )
    expect(lire('a1', 29)).toEqual({ families: [famille('wall', 1, 0, 0)] })
    expect(lire('a1', 30)).toEqual({ families: [famille('wall', 1, 0, 1)] })
  })

  it('un mur pris puis posé : utilisé sur la pose de ses panneaux, pas sur l’appareil', () => {
    const lire = lecteur(
      docWith({
        equipmentChanges: [taken(5, 1, 5)],
        equipmentPlacements: [
          poseAt(20, 'wall', 'deployed', 1, '0x8e2dc574'),
          poseAt(30, 'wall', 'deployed', 1, '0x528fce46'),
        ],
      }),
    )
    expect(lire('a1', 25)).toEqual({ families: [famille('wall', 1, 0, 0)] })
    expect(lire('a1', 30)).toEqual({ families: [famille('wall', 1, 1, 0)] })
  })

  it('un objet pris et encore porté à l’image lue : pris 1, utilisé 0, perdu 0', () => {
    const lire = lecteur(docWith({ equipmentChanges: [taken(5, 1, 6)] }))
    expect(lire('a1', 199)).toEqual({ families: [famille('sensor', 1, 0, 0)] })
  })

  it('un capteur seulement posé (pose déployée) n’est pas utilisé : une pose sur un objet porté est un lâcher volontaire', () => {
    const lire = lecteur(
      docWith({
        equipmentChanges: [taken(5, 1, 6)],
        equipmentPlacements: [poseAt(20, 'sensor', 'deployed', 1)],
      }),
    )
    expect(lire('a1', 100)).toEqual({ families: [famille('sensor', 1, 0, 0)] })
  })

  it('une consommation sur chaîne de compteur trouée ne compte pas comme un usage', () => {
    const lire = lecteur(
      docWith({ equipmentChanges: [taken(5, 1, 6), { ...spent(8, 1, 6), gap: 1 }] }),
    )
    expect(lire('a1', 100)).toEqual({ families: [famille('sensor', 1, 0, 0)] })
  })

  it('un bonus utilisé sans prise datée (équipement de réapparition) compte comme pris, comme la grille', () => {
    const lire = lecteur(docWith({ equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 10, t1: 60 }] }))
    expect(lire('a1', 9)).toEqual({ families: [] })
    expect(lire('a1', 10)).toEqual({ families: [famille('camo', 1, 1, 0)] })
  })

  it('le bonus pris puis lâché est ponté sur le vocabulaire de pose (`powerup_camo`)', () => {
    const lire = lecteur(
      docWith({
        equipmentChanges: [taken(5, 2, 1)],
        equipmentPlacements: [poseAt(40, 'powerup_camo', 'dropped', 2)],
      }),
    )
    expect(lire('a2', 40)).toEqual({ families: [famille('camo', 1, 0, 1)] })
  })

  it('les gestes d’un joueur ne vont jamais à un autre', () => {
    const lire = lecteur(docWith({ equipmentChanges: [taken(5, 1, 6)] }))
    expect(lire('a2', 100)).toEqual({ families: [] })
    expect(lire('b1', 100)).toEqual({ families: [] })
  })
})

describe('playerEquipmentStats — plusieurs familles', () => {
  const lire = lecteur(
    docWith({
      // Le bonus est utilisé AVANT la prise du capteur : l'ordre de la liste reste celui de la grille.
      equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 10, t1: 60 }],
      equipmentChanges: [taken(20, 1, 6), spent(30, 1, 6), taken(40, 1, 7)],
    }),
  )

  it('rend les familles dans l’ordre de la grille du match, pas dans celui du temps', () => {
    expect(lire('a1', 100)?.families.map((f) => f.family)).toEqual(['sensor', 'threat_seeker', 'camo'])
  })

  it('compte chaque famille à part, à chaque image', () => {
    expect(lire('a1', 15)).toEqual({ families: [famille('camo', 1, 1, 0)] })
    expect(lire('a1', 35)).toEqual({ families: [famille('sensor', 1, 1, 0), famille('camo', 1, 1, 0)] })
    expect(lire('a1', 100)).toEqual({
      families: [famille('sensor', 1, 1, 0), famille('threat_seeker', 1, 0, 0), famille('camo', 1, 1, 0)],
    })
  })
})

describe('playerEquipmentStats — famille sans colonne', () => {
  const lire = lecteur(
    docWith({
      equipmentChanges: [taken(5, 1, 10), spent(8, 1, 10), taken(6, 1, 12), taken(7, 1, 6)],
      equipmentPlacements: [poseAt(20, 'repulsor', 'deployed', 1), poseAt(25, 'repulsor', 'dropped', 1)],
    }),
  )

  it('le répulseur et le grappin n’ont aucune entrée, ni zéro ni compte : ils ne sont pas au bilan', () => {
    const end = lire('a1', 100)
    expect(end?.families.map((f) => f.family)).toEqual(['sensor'])
    expect(end?.families.some((f) => f.family === 'repulsor' || f.family === 'grapple')).toBe(false)
  })
})

describe('playerEquipmentStats — bot', () => {
  const doc = docWith({
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
      { filmIndex: 8, xuid: '', bot: true, name: 'Marc', team: 1 },
    ],
    tracks: [vie(1, 'a1'), { ...vie(8, ''), bot: 'Marc' }],
    equipmentChanges: [taken(5, 8, 6), spent(9, 8, 6), taken(6, 1, 5)],
  })
  const sheet = [
    scoreboardRow('a1', 'Alpha', 't0'),
    scoreboardRow('bid(7.0)', 'Marc', 't1', { is_bot: true }),
  ]
  const players = buildPlayers(doc, sheet)
  const index = readEquipmentSeries(doc, players)
  const bot = players.find((p) => p.bot) as ReplayPlayer

  it('se lit par sa clé de roster `bot:<nom>`', () => {
    expect(bot.xuid).toBe('bot:Marc')
    expect(index.has('bot:Marc')).toBe(true)
    expect(playerEquipmentStatsAt(equipmentSeriesOf(index, bot), 9)).toEqual({
      families: [famille('sensor', 1, 1, 0)],
    })
  })

  it('se lit aussi par le xuid de sa ligne de feuille', () => {
    expect(bot.board?.xuid).toBe('bid(7.0)')
    expect(index.get('bid(7.0)')).toBe(index.get('bot:Marc'))
  })

  it('les gestes du bot ne vont pas à l’humain, et inversement', () => {
    const humain = players.find((p) => p.xuid === 'a1') as ReplayPlayer
    expect(playerEquipmentStatsAt(equipmentSeriesOf(index, humain), 100)).toEqual({
      families: [famille('wall', 1, 0, 0)],
    })
  })
})

describe('playerEquipmentStats — attribution', () => {
  it('rend les gestes à la VIE qui occupe le slot à l’image du geste, pas au dernier occupant', () => {
    const doc = docWith({
      tracks: [vie(1, 'a1', 0, 50), vie(1, 'a2', 60, 150)],
      equipmentChanges: [taken(20, 1, 6), taken(100, 1, 5)],
    })
    const lire = lecteur(doc)
    expect(lire('a1', 199)).toEqual({ families: [famille('sensor', 1, 0, 0)] })
    expect(lire('a2', 199)).toEqual({ families: [famille('wall', 1, 0, 0)] })
  })

  it('une pose sans poseur mesuré (owner -1) ne revient à personne', () => {
    const lire = lecteur(docWith({ equipmentPlacements: [poseAt(20, 'sensor', 'dropped', -1)] }))
    for (const xuid of ['a1', 'a2', 'b1', 'orphelin']) expect(lire(xuid, 100)).toEqual({ families: [] })
  })

  it('un joueur sans ligne (équipe tue par le film) n’a pas de réponse : null, jamais « aucun objet »', () => {
    const doc = docWith({
      roster: [
        { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
        { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
      ],
      equipmentChanges: [taken(5, 2, 6)],
    })
    const lire = lecteur(doc)
    expect(lire('a2', 100)).toBeNull()
    expect(lire('a1', 100)).toEqual({ families: [] })
  })

  it('la série absente rend null à toute image', () => {
    expect(playerEquipmentStatsAt(null, 0)).toBeNull()
  })
})

describe('playerEquipmentStats — les comptes ne reculent jamais', () => {
  it('chaque famille ne fait que croître d’une image à l’autre, et pris vaut au moins utilisé + perdu', () => {
    const doc = docWith({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 30 },
        { slot: 1, fam: 'camo', t0: 70, t1: 90 },
      ],
      equipmentChanges: [
        taken(5, 1, 6), taken(5, 1, 6), spent(12, 1, 6), taken(30, 1, 5), spent(35, 1, 6), taken(60, 1, 1),
        taken(80, 1, 7),
      ],
      equipmentPlacements: [
        poseAt(40, 'wall', 'deployed', 1, '0x528fce46'),
        poseAt(50, 'sensor', 'dropped', 1),
        poseAt(95, 'powerup_camo', 'dropped', 1),
        poseAt(96, 'wall', 'dropped', 1),
      ],
    })
    const lire = lecteur(doc)
    let previous = new Map<string, EquipmentFamilyStats>()
    for (let f = 0; f <= 200; f++) {
      const families = lire('a1', f)?.families ?? []
      const now = new Map(families.map((x) => [x.family, x]))
      for (const [family, before] of previous) {
        const after = now.get(family)
        expect(after, `${family} à l'image ${f}`).toBeDefined()
        expect(after!.taken).toBeGreaterThanOrEqual(before.taken)
        expect(after!.used).toBeGreaterThanOrEqual(before.used)
        expect(after!.dropped).toBeGreaterThanOrEqual(before.dropped)
      }
      for (const x of families) expect(x.taken).toBeGreaterThanOrEqual(x.used + x.dropped)
      previous = now
    }
    expect(previous.size).toBeGreaterThan(2)
  })
})

/** Les familles que la grille du match affiche pour un joueur, lues sur le vrai code des colonnes. */
function grille(doc: ReplayDocumentReady, sheet: MatchScoreboardRow[] | undefined): Map<string, EquipmentFamilyStats[]> {
  const usage = buildEquipmentUsage(doc, sheet)
  const columns = usageColumnGroups(usage, REPLAY_TEXT.fr)
    .flatMap((g) => g.columns)
    .filter((c) => c.key.startsWith('equipment.'))
  const out = new Map<string, EquipmentFamilyStats[]>()
  for (const row of usage.byPlayer) {
    const families: EquipmentFamilyStats[] = []
    for (const c of columns) {
      const segments = c.segments?.(row)
      if (!segments) continue
      const of = (key: string) => segments.find((s) => s.key === key)?.value ?? 0
      families.push(famille(c.key.slice('equipment.'.length), c.value(row) ?? 0, of('used'), of('dropped')))
    }
    out.set(row.xuid, families)
  }
  return out
}

/** Compare, joueur par joueur, la dernière image de la lecture à la grille ; rend le nombre de familles comparées. */
function compare(doc: ReplayDocumentReady, sheet: MatchScoreboardRow[] | undefined, label: string): number {
  const players = buildPlayers(doc, sheet ?? [])
  const index = readEquipmentSeries(doc, players)
  const expected = grille(doc, sheet)
  let compared = 0
  for (const [xuid, families] of expected) {
    const player = players.find((p) => p.xuid === xuid) as ReplayPlayer
    const end = playerEquipmentStatsAt(equipmentSeriesOf(index, player), doc.frameCount + 1_000)
    expect(end?.families, `${label} / ${xuid}`).toEqual(families)
    compared += families.length
  }
  return compared
}

describe('playerEquipmentStats — fin de match égale à la grille du match (scénarios des tests du bilan)', () => {
  const SCENARIOS: Record<string, Record<string, unknown>> = {
    'poses déployées et lâchers par famille': {
      equipmentPlacements: [
        pose('sensor', 'deployed', 1), pose('sensor', 'deployed', 1), pose('repair_field', 'deployed', 3),
        pose('sensor', 'dropped', 1), pose('powerup_overshield', 'dropped', 2),
      ],
    },
    'objets engendrés du capteur, du champ et du mur': {
      carriedEquipment: [
        { slot: 1, r: 6, t0: 5, t1: 150, tracked: [{ t0: 10, t1: 40, land: 14, x: 1, y: 2, z: 3 }, { t0: 60, t1: 90 }] },
        { slot: 3, r: 7, t0: 5, t1: 150, tracked: [{ t0: 20, t1: 80, land: 24, x: 1, y: 2, z: 3 }] },
        { slot: 1, r: 5, t0: 5, t1: 150, tracked: [{ t0: 30, t1: 50 }, { t0: 70, t1: 90 }] },
      ],
    },
    'capteur hérité : pose déployée et objet engendré': {
      carriedEquipment: [{ slot: 1, r: 6, t0: 5, t1: 150, tracked: [{ t0: 10, t1: 40 }] }],
      equipmentPlacements: [pose('sensor', 'deployed', 1)],
    },
    'panneaux du mur et appareil': {
      equipmentPlacements: [pose('wall', 'deployed', 1, '0x528fce46'), pose('wall', 'deployed', 1, '0x8e2dc574')],
    },
    'origine de pose non établie': {
      equipmentPlacements: [pose('sensor', 'unknown', 1), { ...pose('sensor', '', 1), origin: undefined }],
    },
    'répulseur, propulseur, grenades et grappin': {
      equipmentChanges: [taken(5, 1, 10), taken(6, 1, 12)],
      equipmentPlacements: [
        pose('repulsor', 'deployed', 1), pose('thruster', 'deployed', 1), pose('repulsor', 'dropped', 1),
        pose('grenade_frag', 'deployed', 1), pose('grapple', 'dropped', 1),
      ],
      grappleLines: [{ slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 }],
    },
    'pose sans poseur mesuré': {
      equipmentPlacements: [pose('sensor', 'deployed', -1), pose('sensor', 'deployed', 1)],
    },
    'mur pris, posé, lâché': {
      equipmentChanges: [taken(5, 1, 5), taken(6, 2, 5), taken(7, 3, 5)],
      equipmentPlacements: [pose('wall', 'deployed', 1, '0x528fce46'), pose('wall', 'dropped', 2)],
    },
    'capteur pris puis seulement posé': {
      equipmentChanges: [taken(5, 1, 6)],
      equipmentPlacements: [pose('sensor', 'deployed', 1)],
    },
    'capteur pris puis consommé, chaîne trouée': {
      equipmentChanges: [taken(5, 1, 6), spent(8, 1, 6), taken(9, 2, 6), { ...spent(12, 2, 6), gap: 1 }],
    },
    'bonus activé, pris, lâché': {
      equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 10, t1: 60 }, { slot: 2, fam: 'overshield', t0: 12, t1: 40 }],
      equipmentChanges: [taken(5, 1, 1), taken(6, 2, 1)],
      equipmentPlacements: [pose('powerup_camo', 'dropped', 2)],
    },
    'prise sans famille connue': {
      abilityLabels: undefined,
      equipmentChanges: [taken(5, 1, 5)],
    },
    'label sans famille': {
      abilityLabels: { '5': { fr: 'Mur de protection', en: 'Drop wall' } },
      equipmentChanges: [taken(5, 1, 5)],
    },
    'slot recyclé entre deux joueurs': {
      tracks: [vie(1, 'a1', 0, 50), vie(1, 'a2', 60, 150), vie(3, 'b1')],
      equipmentChanges: [taken(20, 1, 6), taken(100, 1, 5), spent(110, 1, 5)],
      equipmentPlacements: [poseAt(30, 'sensor', 'dropped', 1), poseAt(120, 'wall', 'dropped', 1)],
    },
  }

  for (const [name, over] of Object.entries(SCENARIOS)) {
    it(name, () => {
      compare(docWith(over), SB, name)
    })
  }

  it('les scénarios ne sont pas vides : ils comparent de vraies familles', () => {
    const total = Object.entries(SCENARIOS).reduce((n, [name, over]) => n + compare(docWith(over), SB, name), 0)
    expect(total).toBeGreaterThan(10)
  })
})

describe('playerEquipmentStats — fin de match égale à la grille du match (documents réels publiés par Go)', () => {
  it('chaque document, joueur par joueur, famille par famille', () => {
    let compared = 0
    for (const entry of goFixtureEntries()) {
      const doc = normalizeReplayDocument(loadGoFixture(entry).doc)
      compared += compare(doc, undefined, `${entry.film} (${entry.build})`)
    }
    // Sans matière dans les documents réels, l'égalité ne prouverait rien.
    expect(compared).toBeGreaterThan(0)
  })
})
