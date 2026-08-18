/**
 * Tests — objectivesLayer (objectifs statiques du mode : normalisation, calque, pulses).
 *
 * CE QU'ILS PROTÈGENT : la boîte se dessine ORIENTÉE (les coins suivent le Forward servi,
 * pas les axes du monde), le calque n'écrit JAMAIS de texte (la lettre A/B/C n'existe
 * dans aucune donnée décodée — garde du lot 4.3), et un pulse ne se pose que sur une
 * position RELUE de l'auteur, jamais au hasard.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayMapObjectives } from '@/lib/api/types'

import {
  buildObjectivePulses,
  drawObjectivePulses,
  drawObjectivesLayer,
  flagPulsesRetired,
  normalizeMapObjectives,
  OBJECTIVE_TEAM_NEUTRAL,
} from './objectivesLayer'
import { drawZoneStates, zoneElementsOf, zoneStateAt } from './zoneStatesLayer'
import { testReplayDoc } from './test/testDoc'

const MO: ReplayMapObjectives = {
  zones: [
    {
      // Boîte tournée : Forward le long de +Y monde -> halfX porte sur Y, halfY sur X.
      role: 'strongholds_zone', team: OBJECTIVE_TEAM_NEUTRAL,
      x: 5, y: 5, z: 1, family: 'box', halfX: 2, halfY: 1, fwdX: 0, fwdY: 1,
    },
    {
      role: 'flag_delivery', team: 1,
      x: 8, y: 2, z: 1, family: 'cylinder', radius: 3, fwdX: 1, fwdY: 0,
    },
  ],
  markers: [
    { role: 'flag_spawn', team: 0, x: 1, y: 9, z: 1 },
    { role: 'flag_delivery', team: 0, x: 1, y: 8, z: 1 },
  ],
}

describe('normalizeMapObjectives', () => {
  it('résout la nullabilité, classe zones et marqueurs, normalise le Forward', () => {
    const els = normalizeMapObjectives(MO)
    expect(els).toHaveLength(4)
    expect(els.filter((e) => e.kind === 'zone')).toHaveLength(2)
    expect(els.filter((e) => e.kind === 'marker')).toHaveLength(2)
    const box = els[0]
    expect(box.family).toBe('box')
    expect(box.fwd).toEqual({ x: 0, y: 1 })
  })

  it('document sans calque = liste vide, jamais une erreur', () => {
    expect(normalizeMapObjectives(null)).toHaveLength(0)
    expect(normalizeMapObjectives(undefined)).toHaveLength(0)
    expect(normalizeMapObjectives({})).toHaveLength(0)
  })

  it('un Forward dégénéré retombe sur (1,0) — boîte alignée, jamais NaN', () => {
    const els = normalizeMapObjectives({
      zones: [{ role: 'r', team: -1, x: 0, y: 0, z: 0, family: 'box', halfX: 1, halfY: 1, fwdX: 0, fwdY: 0 }],
    })
    expect(els[0].fwd).toEqual({ x: 1, y: 0 })
  })
})

/** Contexte canvas enregistreur : jsdom n'implémente pas getContext. */
function mockCtx() {
  const calls: { method: string; args: unknown[] }[] = []
  const record = (method: string) => (...args: unknown[]) => {
    calls.push({ method, args })
  }
  const ctx = {
    globalAlpha: 1,
    fillStyle: '',
    strokeStyle: '',
    lineWidth: 1,
    beginPath: record('beginPath'),
    moveTo: record('moveTo'),
    lineTo: record('lineTo'),
    closePath: record('closePath'),
    arc: record('arc'),
    fill: record('fill'),
    stroke: record('stroke'),
    strokeText: record('strokeText'),
    fillText: record('fillText'),
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, calls }
}

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 480 + 48, height: 480 + 48, pad: 24 }

describe('drawObjectivesLayer', () => {
  it("n'écrit JAMAIS de texte — la lettre A/B/C n'existe pas dans la donnée", () => {
    const { ctx, calls } = mockCtx()
    drawObjectivesLayer(ctx, normalizeMapObjectives(MO), VIEW, { colorOfTeam: () => '#123456' })
    expect(calls.filter((c) => c.method === 'fillText' || c.method === 'strokeText')).toHaveLength(0)
  })

  it('la boîte est ORIENTÉE : ses coins suivent le Forward servi', () => {
    const { ctx, calls } = mockCtx()
    const box = normalizeMapObjectives({ zones: MO.zones ? [MO.zones[0]] : [] })
    drawObjectivesLayer(ctx, box, VIEW, { colorOfTeam: () => '#123456' })
    // VIEW cadre le monde [0,10]² sur 480 px utiles : 48 px par mètre, Y inversé.
    // Centre (5,5), fwd=(0,1), halfX=2 (porte sur Y), halfY=1 (porte sur X) :
    // coins monde (4,7) (4,3) (6,3) (6,7) -> le premier coin canvas est (24+4*48, 24+(10-7)*48).
    const pts = calls.filter((c) => c.method === 'moveTo' || c.method === 'lineTo').map((c) => c.args)
    expect(pts).toHaveLength(4)
    const xs = pts.map((p) => p[0] as number).sort((a, b) => a - b)
    const ys = pts.map((p) => p[1] as number).sort((a, b) => a - b)
    expect(xs[0]).toBeCloseTo(24 + 4 * 48, 5)
    expect(xs[3]).toBeCloseTo(24 + 6 * 48, 5)
    expect(ys[0]).toBeCloseTo(24 + 3 * 48, 5)
    expect(ys[3]).toBeCloseTo(24 + 7 * 48, 5)
  })

  it('le cylindre se projette en cercle au RAYON MONDE (pixels = rayon × échelle)', () => {
    const { ctx, calls } = mockCtx()
    const cyl = normalizeMapObjectives({ zones: MO.zones ? [MO.zones[1]] : [] })
    drawObjectivesLayer(ctx, cyl, VIEW, { colorOfTeam: () => '#123456' })
    const arcs = calls.filter((c) => c.method === 'arc')
    expect(arcs).toHaveLength(1)
    expect(arcs[0].args[2]).toBeCloseTo(3 * 48, 5) // rayon 3 m × 48 px/m
  })

  it('une livraison ponctuelle gagne un ANNEAU, une apparition reste un losange', () => {
    const { ctx, calls } = mockCtx()
    const markers = normalizeMapObjectives({ markers: MO.markers })
    drawObjectivesLayer(ctx, markers, VIEW, { colorOfTeam: () => '#123456' })
    // 2 losanges (closePath) mais UN seul anneau (arc) : celui de la livraison.
    expect(calls.filter((c) => c.method === 'closePath')).toHaveLength(2)
    expect(calls.filter((c) => c.method === 'arc')).toHaveLength(1)
  })

  it('la couleur vient du team servi — le neutre passe par -1', () => {
    const vus: number[] = []
    const { ctx } = mockCtx()
    drawObjectivesLayer(ctx, normalizeMapObjectives(MO), VIEW, {
      colorOfTeam: (team) => {
        vus.push(team)
        return '#123456'
      },
    })
    expect(vus).toContain(OBJECTIVE_TEAM_NEUTRAL)
    expect(vus).toContain(0)
    expect(vus).toContain(1)
  })
})

describe('buildObjectivePulses', () => {
  const doc = testReplayDoc({
    frameIntervalMs: 100,
    tracks: [
      {
        slot: 1, team: -1, xuid: 'A',
        points: [{ t: 0, x: 1, y: 9 }, { t: 100, x: 1, y: 9 }],
        startFrame: 0, endFrame: 100,
      },
    ],
    objectives: [
      { t: 10, xuid: 'A', stat: 'flag_grabs', timeMs: 1_000 },
      { t: 20, xuid: 'INCONNU', stat: 'flag_grabs', timeMs: 2_000 },
      { t: 900, xuid: 'A', stat: 'flag_captures', timeMs: 90_000 },
    ],
  })

  it("pose le pulse sur l'élément le plus proche de l'AUTEUR à l'instant de l'action", () => {
    const pulses = buildObjectivePulses(doc, normalizeMapObjectives(MO))
    // L'action du joueur A (en 1,9) matche le spawn (1,9) — pas la zone (5,5).
    const p = pulses.find((x) => x.frame === 10)
    expect(p).toBeDefined()
    expect(p?.x).toBe(1)
    expect(p?.y).toBe(9)
    expect(p?.team).toBe(0)
  })

  it('écarte les actions sans position relue (auteur inconnu ou hors fenêtre)', () => {
    const pulses = buildObjectivePulses(doc, normalizeMapObjectives(MO))
    expect(pulses.some((p) => p.frame === 20)).toBe(false) // xuid inconnu des vies
    expect(pulses.some((p) => p.frame === 900)).toBe(false) // bien après la fin de vie
  })

  it('sans éléments servis : aucun pulse (mode sans objectifs)', () => {
    expect(buildObjectivePulses(doc, [])).toHaveLength(0)
  })

  // VERROU DE LA REVUE R1 (2026-08-18) : `objectives[].t` est DÉJÀ une frame du document —
  // le Go retranche l'origine depuis le lot A phase 1 (`63b90583c`, `scoreClock.frameOf`).
  // Le client la retranchait une SECONDE fois : les pulses s'allumaient `originMs` trop tôt.
  // La fixture porte donc la sortie Go actuelle : `t` recalé, `timeMs` sur l'horloge du film.
  it("ne retranche PLUS l'origine — `t` est déjà une frame du document", () => {
    const decale = testReplayDoc({
      frameIntervalMs: 100,
      originMs: 3_000, // 30 frames — le Go les a déjà retranchées pour produire `t`
      tracks: [
        {
          slot: 1, team: -1, xuid: 'A',
          points: [{ t: 0, x: 1, y: 9 }, { t: 100, x: 1, y: 9 }],
          startFrame: 0, endFrame: 100,
        },
      ],
      // t = (7 000 − 3 000) / 100 = 40 : la frame du document, pas celle du film.
      objectives: [{ t: 40, xuid: 'A', stat: 'flag_grabs', timeMs: 7_000 }],
    })
    const pulses = buildObjectivePulses(decale, normalizeMapObjectives(MO))
    expect(pulses).toHaveLength(1)
    expect(pulses[0].frame).toBe(40) // et non 10, qui serait la seconde soustraction
  })

  // Le second visage du même défaut : une action dont la frame est INFÉRIEURE à l'origine
  // était purement et simplement jetée (`frame < 0`). Elle doit sortir, à sa frame.
  it("une action des premières secondes du document n'est plus jetée", () => {
    const tot = testReplayDoc({
      frameIntervalMs: 100,
      originMs: 3_000, // 30 frames
      tracks: [
        {
          slot: 1, team: -1, xuid: 'A',
          points: [{ t: 0, x: 1, y: 9 }, { t: 100, x: 1, y: 9 }],
          startFrame: 0, endFrame: 100,
        },
      ],
      objectives: [{ t: 5, xuid: 'A', stat: 'flag_grabs', timeMs: 3_500 }],
    })
    const pulses = buildObjectivePulses(tot, normalizeMapObjectives(MO))
    expect(pulses).toHaveLength(1)
    expect(pulses[0].frame).toBe(5)
  })
})

/**
 * LE SUBSTITUT DU DRAPEAU EST RETIRÉ QUAND L'OBJET EST PUBLIÉ (lot 3.1, schéma 15).
 *
 * Le pulse de CTF posait l'action sur l'élément statique le plus proche de son auteur — un socle
 * voisin, jamais le drapeau. Depuis que `flagCarries` publie l'objet (position, porteur, état,
 * image par image), le garder ferait dire deux choses différentes au même écran.
 *
 * LES AUTRES FAMILLES RESTENT, et c'est la moitié qui compte : zones et crâne n'ont AUCUN objet
 * vivant publié — leur pulse est encore ce qu'on a de mieux.
 */
describe('buildObjectivePulses — la famille DRAPEAU se retire devant l’objet vivant', () => {
  const actions = [
    { t: 10, xuid: 'A', stat: 'flag_grabs', timeMs: 1_000 },
    { t: 11, xuid: 'A', stat: 'flag_captures', timeMs: 1_100 },
    { t: 12, xuid: 'A', stat: 'zone_captures', timeMs: 1_200 },
  ]
  const tracks = [
    {
      slot: 1, team: -1, xuid: 'A',
      points: [{ t: 0, x: 1, y: 9 }, { t: 100, x: 1, y: 9 }],
      startFrame: 0, endFrame: 100,
    },
  ]
  const drapeaux = [
    { team: 0, spans: [{ state: 'home', t0: 0, t1: 100, xuid: null, x: 1, y: 9 }] },
  ]

  it('AVEC des drapeaux publiés : aucune action `flag_*` ne fait de pulse, la zone en fait un', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      tracks: tracks as never,
      objectives: actions,
      flagCarries: drapeaux as never,
    })
    const pulses = buildObjectivePulses(doc, normalizeMapObjectives(MO))
    expect(pulses.map((p) => p.frame)).toEqual([12])
  })

  it('SANS drapeau publié : le substitut garde son rôle (artefact plus ancien)', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      tracks: tracks as never,
      objectives: actions,
    })
    const pulses = buildObjectivePulses(doc, normalizeMapObjectives(MO))
    expect(pulses.map((p) => p.frame)).toEqual([10, 11, 12])
  })

  it('le retrait NE dépend PAS de la bascule d’affichage : il suit la DONNÉE', () => {
    const doc = testReplayDoc({
      frameIntervalMs: 100,
      tracks: tracks as never,
      objectives: actions,
      flagCarries: drapeaux as never,
    })
    expect(flagPulsesRetired(doc)).toBe(true)
    expect(flagPulsesRetired(testReplayDoc({ tracks: tracks as never }))).toBe(false)
  })
})

describe('drawObjectivePulses', () => {
  const pulses = [{ frame: 10, x: 5, y: 5, team: 1 }]

  it("dessine dans la fenêtre, s'ouvre avec l'âge, rien hors fenêtre", () => {
    const { ctx, calls } = mockCtx()
    drawObjectivePulses(ctx, pulses, VIEW, { frame: 12, hold: 14 }, { colorOfTeam: () => '#123456' }, false)
    const arcs = calls.filter((c) => c.method === 'arc')
    expect(arcs).toHaveLength(1)
    const r1 = arcs[0].args[2] as number

    const encore = mockCtx()
    drawObjectivePulses(encore.ctx, pulses, VIEW, { frame: 20, hold: 14 }, { colorOfTeam: () => '#123456' }, false)
    const r2 = encore.calls.filter((c) => c.method === 'arc')[0].args[2] as number
    expect(r2).toBeGreaterThan(r1) // l'anneau S'OUVRE

    const dehors = mockCtx()
    drawObjectivePulses(dehors.ctx, pulses, VIEW, { frame: 40, hold: 14 }, { colorOfTeam: () => '#123456' }, false)
    expect(dehors.calls.filter((c) => c.method === 'arc')).toHaveLength(0)
  })

  it('sous mouvement réduit : anneau statique, pas d’animation', () => {
    const a = mockCtx()
    drawObjectivePulses(a.ctx, pulses, VIEW, { frame: 11, hold: 14 }, { colorOfTeam: () => '#123456' }, true)
    const b = mockCtx()
    drawObjectivePulses(b.ctx, pulses, VIEW, { frame: 23, hold: 14 }, { colorOfTeam: () => '#123456' }, true)
    const ra = a.calls.filter((c) => c.method === 'arc')[0].args[2]
    const rb = b.calls.filter((c) => c.method === 'arc')[0].args[2]
    expect(ra).toBe(rb)
  })
})

/** L'état d'une zone tel que l'artefact le publie (schéma 16), déjà normalisé (`gauge` comblée à vide). */
const ZONE_STATES = [
  {
    zoneRef: 0,
    key: 0x67f43ac3,
    spans: [
      { t0: 0, t1: 9, owner: null, active: false },
      { t0: 10, t1: 19, owner: 0, active: false, progress: 0.75 },
      { t0: 20, t1: 40, owner: 1, active: false },
    ],
    gauge: [],
  },
  { zoneRef: 1, spans: [{ t0: 5, t1: 40, owner: null, active: true, progress: 0.5 }], gauge: [] },
]

describe('zoneStateAt', () => {
  it('rend l’intervalle qui couvre la frame, bornes INCLUSES', () => {
    expect(zoneStateAt(ZONE_STATES, 0, 10)?.owner).toBe(0)
    expect(zoneStateAt(ZONE_STATES, 0, 19)?.owner).toBe(0)
    expect(zoneStateAt(ZONE_STATES, 0, 20)?.owner).toBe(1)
  })

  it('« personne ne la tient » est une MESURE : owner null, pas un état absent', () => {
    const now = zoneStateAt(ZONE_STATES, 0, 3)
    expect(now).not.toBeNull()
    expect(now?.owner).toBeNull()
    expect(now?.progress).toBeNull()
  })

  it('rend null hors de tout intervalle, et pour une zone sans état', () => {
    expect(zoneStateAt(ZONE_STATES, 0, 41)).toBeNull()
    expect(zoneStateAt(ZONE_STATES, 7, 10)).toBeNull()
    expect(zoneStateAt([], 0, 10)).toBeNull()
  })

  it('porte la zone ACTIVE et la progression telles quelles', () => {
    const now = zoneStateAt(ZONE_STATES, 1, 30)
    expect(now?.active).toBe(true)
    expect(now?.progress).toBe(0.5)
  })
})

describe('zoneElementsOf', () => {
  it('rend les zones SURFACIQUES dans l’ordre servi — celui que zoneRef indexe', () => {
    const zones = zoneElementsOf(normalizeMapObjectives(MO))
    expect(zones).toHaveLength(2)
    expect(zones.every((z) => z.kind === 'zone')).toBe(true)
    expect(zones[0].family).toBe('box')
    expect(zones[1].family).toBe('cylinder')
  })
})

describe('drawZoneStates', () => {
  const style = { colorOfOwner: (team: number) => (team === 0 ? '#allié' : '#adverse'), neutral: '#neutre' }
  const zones = () => zoneElementsOf(normalizeMapObjectives(MO))
  /** L'entrée du calque telle que `useZoneStates` la rend : jointure ACCORDÉE sauf dit autrement. */
  const layer = (zoneElements = zones(), joinable = true) => ({ zoneElements, joinable, style })

  it("n'écrit JAMAIS de texte, comme le calque statique", () => {
    const { ctx, calls } = mockCtx()
    drawZoneStates(ctx, layer(), ZONE_STATES, VIEW, 10)
    expect(calls.filter((c) => c.method === 'fillText' || c.method === 'strokeText')).toHaveLength(0)
  })

  it('une zone TENUE est remplie ET cerclée à l’encre de son camp', () => {
    const { ctx, calls } = mockCtx()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 10)
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(1)
    expect(calls.filter((c) => c.method === 'stroke').length).toBeGreaterThanOrEqual(1)
  })

  it('une zone que PERSONNE ne tient garde le liseré seul — aucun remplissage', () => {
    const { ctx, calls } = mockCtx()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 3)
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(0)
    expect(calls.filter((c) => c.method === 'stroke')).toHaveLength(1)
  })

  it('une zone sans état à cette frame n’est PAS repeinte : elle reste au trait faible', () => {
    const { ctx, calls } = mockCtx()
    drawZoneStates(ctx, layer(), ZONE_STATES, VIEW, 41)
    expect(calls.filter((c) => c.method === 'fill' || c.method === 'stroke')).toHaveLength(0)
  })

  it('la progression ajoute un ARC, et seulement quand la jauge est publiée', () => {
    const avec = mockCtx()
    drawZoneStates(avec.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 10)
    expect(avec.calls.filter((c) => c.method === 'arc')).toHaveLength(1)
    const sans = mockCtx()
    drawZoneStates(sans.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 25)
    expect(sans.calls.filter((c) => c.method === 'arc')).toHaveLength(0)
  })

  it('camp inconnu (aucune ligne « moi ») : encre NEUTRE, jamais une couleur devinée', () => {
    const { ctx, calls } = mockCtx()
    const aveugle = { colorOfOwner: () => null, neutral: '#neutre' }
    drawZoneStates(ctx, { ...layer([zones()[0]]), style: aveugle }, [ZONE_STATES[0]], VIEW, 10)
    // Aucun remplissage : une zone TENUE par un camp qu'on ne sait pas situer garde le liseré
    // seul. Les deux tracés sont le contour et l'arc de progression, tous deux à l'encre neutre.
    expect(calls.filter((c) => c.method === 'fill')).toHaveLength(0)
    expect(calls.filter((c) => c.method === 'stroke')).toHaveLength(2)
  })

  it('sans état publié, le calque ne dessine rien du tout', () => {
    const { ctx, calls } = mockCtx()
    drawZoneStates(ctx, layer(), [], VIEW, 10)
    expect(calls).toHaveLength(0)
  })

  // VERROU DE LA REVUE R1-7 : `zoneRef` est un index figé à la cuisson, la liste servie est
  // reconstruite à la requête. Quand le catalogue de l'artefact ne joint pas la liste servie,
  // le calque VIVANT ne touche PAS au contexte — pas un trait, pas même un `beginPath` : teinter
  // la mauvaise zone serait une erreur invisible et crédible. Retirer la garde du calque fait
  // échouer ce cas, avec exactement les mêmes états et les mêmes zones que le cas « tenue ».
  it('jointure REFUSÉE (catalogue différent de la liste servie) : le calque ne peint rien', () => {
    const { ctx, calls } = mockCtx()
    drawZoneStates(ctx, layer([zones()[0]], false), [ZONE_STATES[0]], VIEW, 10)
    expect(calls).toHaveLength(0)
  })
})
