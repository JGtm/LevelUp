/**
 * Tests — equipmentUsageLogic (les usages d'équipement agrégés par joueur et par équipe).
 *
 * CE QU'ILS PROTÈGENT, dans l'ordre des pièges du domaine :
 *   - le PONT slot -> joueur -> équipe : un joueur est ses VIES, et son camp vient du
 *     SCOREBOARD (le film n'en porte aucun, `Track.Team` vaut -1) ;
 *   - le joueur HORS SCOREBOARD garde sa ligne, SANS équipe — le trou se montre ;
 *   - l'ANONYME reste anonyme : les socles de bonus vidés ne descendent sur aucune ligne ;
 *   - ce qui est mesuré sans propriétaire est COMPTÉ à part, jamais versé au hasard ;
 *   - répulseur et propulseur n'ouvrent AUCUNE colonne de pose : aucune grandeur, aucun zéro
 *     (l'usage du propulseur est mesuré depuis le schéma 38, mais il se lit sur la carte).
 *
 * Les fixtures passent par `testReplayDoc`, la seule porte du document de test (garde-rail
 * `testDoc.guard.test.ts`) : elles décrivent un document de TRANSPORT, comme le serveur l'envoie.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { buildEquipmentUsage, tallyIsEmpty } from './equipmentUsageLogic'
import { testReplayDoc } from './test/testDoc'

/** Une vie : le slot, son propriétaire, et deux points pour que la fenêtre existe. */
function vie(slot: number, xuid: string, start = 0, end = 100) {
  return {
    slot,
    xuid,
    team: -1,
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

/** Une pose d'équipement (les champs que l'agrégation lit). */
function pose(family: string, origin: string, owner: number, id = '0xaaaa') {
  return { family, origin, owner, id, t0: 10, t1: 20, x: 0, y: 0 }
}

const SB: MatchScoreboardRow[] = [
  { xuid: 'a1', gamertag: 'Alpha', team_side: 't0' },
  { xuid: 'a2', gamertag: 'Bravo', team_side: 't0' },
  { xuid: 'b1', gamertag: 'Charlie', team_side: 't1' },
] as MatchScoreboardRow[]

/**
 * LE TÉMOIN. Trois joueurs au scoreboard (deux camps) plus un QUATRIÈME que le film voit vivre
 * et que le scoreboard ignore ; un slot de caméra (vie sans xuid) qui porte pourtant des gestes.
 *
 * Frames à 100 ms : un épisode de 50 frames dure 5 000 ms — les durées sont donc lisibles à
 * l'œil dans les attentes ci-dessous.
 */
function temoin(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
      { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
      { filmIndex: 2, xuid: 'b1', name: 'Charlie' },
      { filmIndex: 3, xuid: 'orphelin', name: 'Delta' },
    ],
    tracks: [
      vie(1, 'a1'),
      vie(2, 'a2'),
      vie(3, 'b1'),
      vie(4, 'orphelin'),
      // Une vie SANS propriétaire : caméra ou spectateur de fin de partie.
      { slot: 9, team: -1, startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }] },
    ],
    ...over,
  } as Partial<ReplayDocument>)
}

describe('buildEquipmentUsage — le pont slot -> joueur -> équipe', () => {
  it('attribue les tractions de grappin au propriétaire de la vie, pas au slot', () => {
    const doc = temoin({
      grappleLines: [
        { slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 },
        { slot: 1, t0: 8, t1: 12, ax: 0, ay: 0 },
        { slot: 3, t0: 2, t1: 6, ax: 0, ay: 0 },
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const parNom = new Map(u.byPlayer.map((r) => [r.name, r]))
    expect(parNom.get('Alpha')?.grapplePulls).toBe(2)
    expect(parNom.get('Charlie')?.grapplePulls).toBe(1)
    expect(parNom.get('Bravo')?.grapplePulls).toBe(0)
    expect(u.columns.grapple).toBe(true)
  })

  it('range les joueurs par camp du SCOREBOARD et somme chaque camp', () => {
    const doc = temoin({
      grappleLines: [
        { slot: 1, t0: 1, t1: 5, ax: 0, ay: 0 },
        { slot: 2, t0: 1, t1: 5, ax: 0, ay: 0 },
        { slot: 3, t0: 1, t1: 5, ax: 0, ay: 0 },
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const t0 = u.byTeam.find((g) => g.side === 't0')
    const t1 = u.byTeam.find((g) => g.side === 't1')
    expect(t0?.players.map((p) => p.name)).toEqual(['Alpha', 'Bravo'])
    expect(t0?.total.grapplePulls).toBe(2)
    expect(t1?.total.grapplePulls).toBe(1)
  })

  it('garde le joueur HORS SCOREBOARD, sans équipe — le trou se montre, il ne se comble pas', () => {
    const u = buildEquipmentUsage(temoin(), SB)
    const sansEquipe = u.byTeam.find((g) => g.side === null)
    expect(sansEquipe?.players.map((p) => p.name)).toEqual(['Delta'])
    expect(sansEquipe?.players[0].side).toBeNull()
    // Et il n'a été versé dans AUCUN camp nommé.
    expect(u.byTeam.filter((g) => g.side !== null).flatMap((g) => g.players).map((p) => p.name))
      .toEqual(['Alpha', 'Bravo', 'Charlie'])
  })

  it('sans scoreboard du tout, personne n’a d’équipe et rien n’est deviné', () => {
    const u = buildEquipmentUsage(temoin(), undefined)
    expect(u.byTeam).toHaveLength(1)
    expect(u.byTeam[0].side).toBeNull()
    expect(u.byPlayer).toHaveLength(4)
  })

  it('n’ouvre aucune ligne pour une entrée de roster que le film n’a jamais vue vivre', () => {
    const doc = temoin({
      roster: [
        { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
        { filmIndex: 9, xuid: 'jamais_vu', name: 'Echo' },
      ],
      tracks: [vie(1, 'a1')],
    } as Partial<ReplayDocument>)
    expect(buildEquipmentUsage(doc, SB).byPlayer.map((r) => r.name)).toEqual(['Alpha'])
  })
})

describe('buildEquipmentUsage — les épisodes d’état actif (camouflage, surbouclier)', () => {
  it('compte les épisodes ET cumule leur durée, famille par famille', () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 60 },
        { slot: 1, fam: 'camo', t0: 80, t1: 100 },
        { slot: 1, fam: 'overshield', t0: 20, t1: 50 },
        { slot: 3, fam: 'overshield', t0: 0, t1: 10 },
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    // 50 + 20 frames a 100 ms = 7 000 ms de camouflage, en 2 episodes. Aucun `k` sur la
    // fixture : kills reste a 0 (kills=0 et killsRead=false se distinguent au niveau de la
    // couverture, pas ici — cf. describe dedie plus bas).
    expect(alpha?.episodes.camo).toEqual({ count: 2, ms: 7000, kills: 0 })
    expect(alpha?.episodes.overshield).toEqual({ count: 1, ms: 3000, kills: 0 })
    expect(u.columns.episodes).toEqual(['camo', 'overshield'])
  })

  it('n’ouvre la colonne que pour la famille réellement portée', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 1, fam: 'overshield', t0: 10, t1: 20 }],
    } as Partial<ReplayDocument>)
    expect(buildEquipmentUsage(doc, SB).columns.episodes).toEqual(['overshield'])
  })

  it('écarte une famille d’épisode que le document ne mesure pas (aucun libellé, aucun sens)', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 1, fam: 'inconnue', t0: 10, t1: 20 }],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.columns.episodes).toEqual([])
    expect(u.hasData).toBe(false)
  })

  it('borne à zéro un épisode dont les bornes sont inversées, sans le perdre', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 60, t1: 10 }],
    } as Partial<ReplayDocument>)
    const alpha = buildEquipmentUsage(doc, SB).byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.episodes.camo).toEqual({ count: 1, ms: 0, kills: 0 })
  })
})

describe('buildEquipmentUsage — les frags sous effet actif (LOT F.2)', () => {
  it('somme les frags du porteur sur TOUS les épisodes de la famille', () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 60, k: 2 },
        { slot: 1, fam: 'camo', t0: 80, t1: 100, k: 1 },
        { slot: 1, fam: 'overshield', t0: 20, t1: 50, k: 3 },
      ],
    } as Partial<ReplayDocument>)
    const alpha = buildEquipmentUsage(doc, SB).byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.episodes.camo.kills).toBe(3)
    expect(alpha?.episodes.overshield.kills).toBe(3)
  })

  it('un épisode sans `k` (omitempty) compte comme zéro, jamais comme une absence', () => {
    const doc = temoin({
      equipmentEpisodes: [{ slot: 1, fam: 'camo', t0: 10, t1: 20 }],
    } as Partial<ReplayDocument>)
    const alpha = buildEquipmentUsage(doc, SB).byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.episodes.camo.kills).toBe(0)
  })

  it('un total d’équipe additionne les kills de ses joueurs, comme le reste du tally', () => {
    const doc = temoin({
      equipmentEpisodes: [
        { slot: 1, fam: 'camo', t0: 10, t1: 20, k: 2 },
        { slot: 2, fam: 'camo', t0: 10, t1: 20, k: 1 },
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const t0 = u.byTeam.find((g) => g.side === 't0')
    expect(t0?.total.episodes.camo.kills).toBe(3)
  })
})

describe('buildEquipmentUsage — poses déployées et objets lâchés', () => {
  it('sépare les DÉPLOIEMENTS des LÂCHERS, famille par famille', () => {
    const doc = temoin({
      equipmentPlacements: [
        pose('sensor', 'deployed', 1),
        pose('sensor', 'deployed', 1),
        pose('repair_field', 'deployed', 3),
        pose('sensor', 'dropped', 1),
        pose('powerup_overshield', 'dropped', 2),
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const parNom = new Map(u.byPlayer.map((r) => [r.name, r]))
    expect(parNom.get('Alpha')?.deployed).toEqual({ sensor: 2 })
    expect(parNom.get('Alpha')?.dropped).toEqual({ sensor: 1 })
    expect(parNom.get('Charlie')?.deployed).toEqual({ repair_field: 1 })
    expect(parNom.get('Bravo')?.dropped).toEqual({ powerup_overshield: 1 })
    // Ordre ÉCRIT : celui des tables de référence, pas celui de la rencontre dans le film.
    expect(u.columns.deployed).toEqual(['sensor', 'repair_field'])
    expect(u.columns.dropped).toEqual(['sensor', 'powerup_overshield'])
  })

  it('ne compte QUE les panneaux du mur déployé — l’appareil et ses panneaux font UN mur', () => {
    const doc = temoin({
      equipmentPlacements: [
        // 0x528fce46 = les PANNEAUX (cf. WALL_PANEL_IDS) ; l'autre id est l'appareil qui vole.
        pose('wall', 'deployed', 1, '0x528fce46'),
        pose('wall', 'deployed', 1, '0x8e2dc574'),
      ],
    } as Partial<ReplayDocument>)
    const alpha = buildEquipmentUsage(doc, SB).byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.deployed).toEqual({ wall: 1 })
  })

  it('écarte une pose d’origine NON ÉTABLIE : ni déploiement, ni lâcher', () => {
    const doc = temoin({
      equipmentPlacements: [pose('sensor', 'unknown', 1), { ...pose('sensor', '', 1), origin: undefined }],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.columns.deployed).toEqual([])
    expect(u.columns.dropped).toEqual([])
    expect(u.hasData).toBe(false)
  })

  it('n’ouvre AUCUNE grandeur de POSE pour le répulseur et le propulseur', () => {
    const doc = temoin({
      equipmentPlacements: [
        pose('repulsor', 'deployed', 1),
        pose('thruster', 'deployed', 1),
        pose('repulsor', 'dropped', 1),
        pose('grenade_frag', 'deployed', 1),
        pose('grapple', 'dropped', 1),
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.columns.deployed).toEqual([])
    expect(u.columns.dropped).toEqual([])
    expect(u.hasData).toBe(false)
  })

  it('range une pose SANS poseur mesuré (owner -1) hors des lignes de joueur', () => {
    const doc = temoin({
      equipmentPlacements: [pose('sensor', 'deployed', -1), pose('sensor', 'deployed', 1)],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.unattributed.deployed).toEqual({ sensor: 1 })
    expect(u.byPlayer.reduce((n, r) => n + (r.deployed.sensor ?? 0), 0)).toBe(1)
  })
})

describe('buildEquipmentUsage — grenades lancées', () => {
  it('compte les lancers par RANG du catalogue, dans l’ordre des rangs', () => {
    const doc = temoin({
      grenades: [
        // `i` = l'index de joueur du film (0 = Alpha, 2 = Charlie au roster du témoin).
        { slot: 0, rank: 1, t: 5, i: 0, s: 'x', x: 0, y: 0 },
        { slot: 0, rank: 0, t: 6, i: 0, s: 'x', x: 0, y: 0 },
        { slot: 0, rank: 0, t: 7, i: 0, s: 'x', x: 0, y: 0 },
        { slot: 0, rank: 1, t: 8, i: 2, s: 'x', x: 0, y: 0 },
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    const alpha = u.byPlayer.find((r) => r.name === 'Alpha')
    expect(alpha?.grenades).toEqual({ 0: 2, 1: 1 })
    expect(u.columns.grenades).toEqual([0, 1])
    expect(u.byTeam.find((g) => g.side === 't1')?.total.grenades).toEqual({ 1: 1 })
  })

  /**
   * LE PIÈGE DU CANAL, vérifié sur pièces le 2026-08-25 : `Grenade.slot` est « le biped lanceur
   * QUAND IL EST CONNU (0 sinon) » (grenades.go), l'auteur est `Grenade.i`. Sur quatre témoins
   * du cache, 65/70, 108/143 et 123/130 lancers portent un slot ABSENT des pistes. Joindre par
   * le slot verserait tous ces lancers au propriétaire du slot 0 dès qu'il existe.
   */
  it('joint par l’INDEX DE FILM, jamais par le slot — un slot menteur ne trompe personne', () => {
    const doc = temoin({
      // Slot 3 = la vie de Charlie ; l'auteur écrit dans le film est Alpha (index 0).
      grenades: [{ slot: 3, rank: 0, t: 5, i: 0, s: 'x', x: 0, y: 0 }],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.byPlayer.find((r) => r.name === 'Alpha')?.grenades).toEqual({ 0: 1 })
    expect(u.byPlayer.find((r) => r.name === 'Charlie')?.grenades).toEqual({})
  })

  it('range un lancer dont l’index de film n’est à personne dans les orphelins', () => {
    const doc = temoin({
      grenades: [{ slot: 0, rank: 0, t: 5, i: 77, s: 'x', x: 0, y: 0 }],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.unattributed.grenades).toEqual({ 0: 1 })
    expect(u.hasData).toBe(false)
  })

  it('un lancer d’une entrée de roster SANS vie va aux orphelins, jamais dans le vide', () => {
    const doc = temoin({
      roster: [
        { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
        { filmIndex: 5, xuid: 'jamais_vu', name: 'Echo' },
      ],
      tracks: [vie(1, 'a1')],
      grenades: [{ slot: 0, rank: 0, t: 5, i: 5, s: 'x', x: 0, y: 0 }],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.byPlayer.map((r) => r.name)).toEqual(['Alpha'])
    expect(u.unattributed.grenades).toEqual({ 0: 1 })
  })
})

describe('buildEquipmentUsage — le canal ANONYME et les gestes sans propriétaire', () => {
  it('compte les socles de BONUS vidés, par famille, au niveau du MATCH', () => {
    const doc = temoin({
      weaponPads: [
        { weapon: '0xdead', x: 0, y: 0, spawns: [], presence: [] },
        { weapon: 'powerup_overshield', x: 1, y: 1, spawns: [], presence: [] },
        { weapon: 'powerup_camo', x: 2, y: 2, spawns: [], presence: [] },
      ],
      padPickups: [
        { pad: 1, tLow: 10, tHigh: 30, xuid: null },
        { pad: 1, tLow: 60, tHigh: 80, xuid: null },
        { pad: 2, tLow: 20, tHigh: 40, xuid: null },
        // Un socle d'ARME : ce n'est pas un bonus, il ne compte pas ici.
        { pad: 0, tLow: 10, tHigh: 20, xuid: null },
        // Index hors bornes : ne compte pour rien, jamais pour un socle voisin.
        { pad: 42, tLow: 10, tHigh: 20, xuid: null },
      ],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.powerupPickups).toEqual({ powerup_overshield: 2, powerup_camo: 1 })
    expect(u.powerupPickupsTotal).toBe(3)
    // ET IL RESTE ANONYME : aucune ligne de joueur n'en porte trace.
    expect(u.byPlayer.every((r) => tallyIsEmpty(r))).toBe(true)
  })

  it('un vidage de socle suffit à ouvrir la section, même sans aucun geste attribué', () => {
    const doc = temoin({
      weaponPads: [{ weapon: 'powerup_camo', x: 0, y: 0, spawns: [], presence: [] }],
      padPickups: [{ pad: 0, tLow: 1, tHigh: 2, xuid: null }],
    } as Partial<ReplayDocument>)
    expect(buildEquipmentUsage(doc, SB).hasData).toBe(true)
  })

  it('compte à part ce qui vient d’une vie SANS propriétaire (caméra, spectateur)', () => {
    const doc = temoin({
      grappleLines: [{ slot: 9, t0: 1, t1: 5, ax: 0, ay: 0 }],
      grenades: [{ slot: 9, rank: 2, t: 5, i: 99, s: 'x', x: 0, y: 0 }],
      equipmentEpisodes: [{ slot: 9, fam: 'camo', t0: 0, t1: 10 }],
    } as Partial<ReplayDocument>)
    const u = buildEquipmentUsage(doc, SB)
    expect(u.unattributed.grapplePulls).toBe(1)
    expect(u.unattributed.grenades).toEqual({ 2: 1 })
    expect(u.unattributed.episodes.camo).toEqual({ count: 1, ms: 1000, kills: 0 })
    expect(u.byPlayer.every((r) => tallyIsEmpty(r))).toBe(true)
    // Rien d'ATTRIBUÉ : la section reste fermée, il n'y a aucune ligne à écrire.
    expect(u.hasData).toBe(false)
  })
})

describe('buildEquipmentUsage — les dénominateurs de couverture', () => {
  it('recopie les dénominateurs du document, sans en recalculer aucun', () => {
    const doc = temoin({
      coverage: {
        equipment: {
          tracksTotal: 90,
          camoLives: 3,
          camoEpisodes: 4,
          overshieldLives: 2,
          overshieldEpisodes: 2,
          killsRead: true,
        },
        grapple: { pulls: 12, pullLives: 7, lightReads: 20, heavyReads: 14, unpairedFires: 2, brokenBodies: 0 },
        placements: { byFamilyOrigin: { 'sensor/deployed': 5 } },
        groundWeapons: { powerupPads: 2 },
      },
    } as unknown as Partial<ReplayDocument>)
    const cov = buildEquipmentUsage(doc, SB).coverage
    expect(cov.tracksTotal).toBe(90)
    expect(cov.episodeLives).toEqual({ camo: 3, overshield: 2 })
    expect(cov.grapplePulls).toBe(12)
    expect(cov.grapplePullLives).toBe(7)
    expect(cov.placementsByFamilyOrigin).toEqual({ 'sensor/deployed': 5 })
    expect(cov.powerupPads).toBe(2)
    expect(cov.killsRead).toBe(true)
  })

  it('killsRead faux (jointure non tentée) se distingue d’une jointure lue à zéro', () => {
    const doc = temoin({
      coverage: {
        equipment: {
          tracksTotal: 10,
          camoLives: 0,
          camoEpisodes: 0,
          overshieldLives: 0,
          overshieldEpisodes: 0,
          killsRead: false,
        },
      },
    } as unknown as Partial<ReplayDocument>)
    expect(buildEquipmentUsage(doc, SB).coverage.killsRead).toBe(false)
  })

  it('un artefact sans bloc de couverture ne rend AUCUN dénominateur inventé', () => {
    const cov = buildEquipmentUsage(temoin(), SB).coverage
    expect(cov).toEqual({
      tracksTotal: 0,
      episodeLives: { camo: 0, overshield: 0 },
      grapplePulls: 0,
      grapplePullLives: 0,
      placementsByFamilyOrigin: {},
      powerupPads: 0,
      killsRead: false,
    })
  })
})

describe('buildEquipmentUsage — la double porte', () => {
  it('un document vide ne porte aucune donnée : la section ne doit rien rendre', () => {
    const u = buildEquipmentUsage(testReplayDoc(), undefined)
    expect(u.hasData).toBe(false)
    expect(u.byPlayer).toEqual([])
    expect(u.columns).toEqual({ grapple: false, episodes: [], deployed: [], dropped: [], grenades: [] })
  })
})
