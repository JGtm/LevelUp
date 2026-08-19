/**
 * Tests — zoneStatesLayer (l'état VIVANT des zones, schémas 16-17).
 *
 * CE QU'ILS PROTÈGENT : l'état se lit sur l'intervalle qui couvre la frame (bornes incluses),
 * « personne ne la tient » est une MESURE, le calque n'écrit jamais de texte, il refuse de
 * peindre quand la jointure du catalogue est douteuse — et, depuis le schéma 17, L'ARC SUIT LA
 * SÉRIE DE LA JAUGE EN DIRECT en escalier : jamais le sommet de l'intervalle (le test échoue si
 * l'on y repasse), AUCUN arc sans `gauge`, la valeur TENUE jusqu'au point suivant (une capture figée reste
 * affichée) et retour à rien une seconde après le dernier point de la série.
 *
 * Extraits de `objectivesLayer.test.ts` le 2026-08-18 (lot C-ter volet 3) : le calque a son
 * fichier, ses tests aussi.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayMapObjectives } from '@/lib/api/types'

import { normalizeMapObjectives, OBJECTIVE_TEAM_NEUTRAL } from './objectivesLayer'
import { count, recordingContext, valuesOf } from './test/recordingContext'
import type { ReplayZoneStateReady } from './replayNormalize'
import { drawZoneStates, zoneElementsOf, zoneGaugeAt } from './zoneStatesLayer'

const MO: ReplayMapObjectives = {
  zones: [
    {
      role: 'strongholds_zone', team: OBJECTIVE_TEAM_NEUTRAL,
      x: 5, y: 5, z: 1, family: 'box', halfX: 2, halfY: 1, fwdX: 0, fwdY: 1,
    },
    {
      role: 'flag_delivery', team: 1,
      x: 8, y: 2, z: 1, family: 'cylinder', radius: 3, fwdX: 1, fwdY: 0,
    },
  ],
  markers: [{ role: 'flag_spawn', team: 0, x: 1, y: 9, z: 1 }],
}

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 480 + 48, height: 480 + 48, pad: 24 }

/** Un document à 100 ms par frame : la tenue d'une seconde vaut 10 frames. */
const HOLD = 10

/**
 * L'état d'une zone tel que l'artefact le publie (schéma 17), déjà normalisé. La zone 0 porte
 * une RAMPE de jauge aux frames 12..18 (le sommet 0,75 de l'intervalle [10 ; 19] est atteint à
 * la frame 18) FERMÉE par son retour à zéro à 19 ; puis une seconde capture qui monte (30..32),
 * se FIGE 28 frames à 0,2 (zone contestée : aucun point), reprend à 60 et est abandonnée — retour
 * à zéro à 62.
 */
const ZONE_STATES: ReplayZoneStateReady[] = [
  {
    zoneRef: 0,
    key: 0x67f43ac3,
    spans: [
      { t0: 0, t1: 9, owner: null, active: false },
      { t0: 10, t1: 19, owner: 0, active: false, progress: 0.75 },
      { t0: 20, t1: 40, owner: 1, active: false },
    ],
    gauge: [
      { t: 12, v: 0 }, { t: 14, v: 0.3 }, { t: 16, v: 0.55 }, { t: 18, v: 0.75 }, { t: 19, v: 0 },
      { t: 30, v: 0 }, { t: 32, v: 0.2 }, { t: 60, v: 0.5 }, { t: 62, v: 0 },
    ],
  },
  { zoneRef: 1, spans: [{ t0: 5, t1: 40, owner: null, active: true, progress: 0.5 }], gauge: [] },
]

describe('zoneGaugeAt — l’escalier de la jauge en direct', () => {
  const gauge = ZONE_STATES[0].gauge

  it('rien AVANT le premier point', () => {
    expect(zoneGaugeAt(gauge, 0, HOLD)).toBeNull()
    expect(zoneGaugeAt(gauge, 11, HOLD)).toBeNull()
  })

  it('la dernière valeur dont l’instant est <= frame — un escalier, pas une pente', () => {
    expect(zoneGaugeAt(gauge, 12, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 13, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 14, HOLD)).toBe(0.3)
    expect(zoneGaugeAt(gauge, 15, HOLD)).toBe(0.3)
    expect(zoneGaugeAt(gauge, 18, HOLD)).toBe(0.75)
  })

  it('le retour à zéro FERME la rampe : la valeur retombe à 0 et y reste', () => {
    expect(zoneGaugeAt(gauge, 19, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 25, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 29, HOLD)).toBe(0)
  })

  it('une capture FIGÉE tient sa valeur jusqu’au point suivant — pas d’expiration entre deux points', () => {
    expect(zoneGaugeAt(gauge, 32, HOLD)).toBe(0.2)
    expect(zoneGaugeAt(gauge, 32 + HOLD + 1, HOLD)).toBe(0.2)
    expect(zoneGaugeAt(gauge, 59, HOLD)).toBe(0.2)
    expect(zoneGaugeAt(gauge, 60, HOLD)).toBe(0.5)
    expect(zoneGaugeAt(gauge, 62, HOLD)).toBe(0)
  })

  it('le DERNIER point de la série tient une seconde, puis plus rien', () => {
    expect(zoneGaugeAt(gauge, 62 + HOLD, HOLD)).toBe(0)
    expect(zoneGaugeAt(gauge, 62 + HOLD + 1, HOLD)).toBeNull()
    // Une série qui s'arrête sur un sommet (fin de film, retour à zéro non lu) : même tenue.
    const sommet = [{ t: 12, v: 0 }, { t: 18, v: 0.75 }]
    expect(zoneGaugeAt(sommet, 18 + HOLD, HOLD)).toBe(0.75)
    expect(zoneGaugeAt(sommet, 18 + HOLD + 1, HOLD)).toBeNull()
  })

  it('une série vide (schéma <= 16, ou zone sans rampe) ne rend jamais de valeur', () => {
    expect(zoneGaugeAt([], 15, HOLD)).toBeNull()
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
  const style = {
    colorOfOwner: (team: number) => (team === 0 ? '#allié' : '#adverse'),
    colorOfCapturer: (owner: number) => (owner === 0 ? '#adverse' : '#allié'),
    neutral: '#neutre',
  }
  const zones = () => zoneElementsOf(normalizeMapObjectives(MO))
  /** L'entrée du calque telle que `useZoneStates` la rend : jointure ACCORDÉE sauf dit autrement. */
  const layer = (zoneElements = zones(), joinable = true) => ({ zoneElements, joinable, style, gaugeHoldFrames: HOLD })
  /** L'angle de fin du DERNIER arc émis, ramené à la fraction de tour qu'il couvre. */
  const arcFraction = (ops: { op: string; args: unknown[] }[]) => {
    const arcs = ops.filter((o) => o.op === 'arc')
    const a = arcs[arcs.length - 1].args
    return ((a[4] as number) - (a[3] as number)) / (2 * Math.PI)
  }

  it("n'écrit JAMAIS de texte, comme le calque statique", () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), ZONE_STATES, VIEW, 10)
    expect(count(ops, 'fillText') + count(ops, 'strokeText')).toBe(0)
  })

  it('une zone TENUE est remplie ET cerclée à l’encre de son camp', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 10)
    expect(count(ops, 'fill')).toBe(1)
    expect(count(ops, 'stroke')).toBeGreaterThanOrEqual(1)
  })

  it('une zone que PERSONNE ne tient garde le liseré seul — aucun remplissage', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 3)
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'stroke')).toBe(1)
  })

  // La zone ACTIVE est le SEUL cas où le calque remplit sans propriétaire : la surbrillance dit
  // « c'est ici que ça se joue » là où le liseré seul dirait « personne ne la tient ». Ce cas
  // porte la lecture d'`active` depuis la revue du 2026-08-19 : elle était vérifiée sur la
  // fonction de lecture exportée, que plus rien n'appelait — c'est au RENDU qu'elle se voit.
  it('la zone ACTIVE est en SURBRILLANCE : remplie sans propriétaire, liseré renforcé', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), [ZONE_STATES[1]], VIEW, 30)
    expect(count(ops, 'fill')).toBe(1)
    expect(valuesOf(ops, 'lineWidth')).toContain(3.5)
  })

  it('une zone sans état à cette frame n’est PAS repeinte : elle reste au trait faible', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), ZONE_STATES, VIEW, 80)
    expect(count(ops, 'fill') + count(ops, 'stroke')).toBe(0)
  })

  // LE VERROU DU SCHÉMA 17 : l'arc se remplit avec la VALEUR de la jauge à l'image. À la frame
  // 14 la série dit 0,3 alors que le sommet de l'intervalle dit 0,75 — repasser au sommet fait
  // échouer ce cas, exactement comme dessiner 0,55 à la frame 15 (l'escalier tient 0,3).
  it("l'arc SUIT la série de la jauge, en escalier — jamais le sommet de l'intervalle", () => {
    const a14 = recordingContext()
    drawZoneStates(a14.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    expect(count(a14.ops, 'arc')).toBe(1)
    expect(arcFraction(a14.ops)).toBeCloseTo(0.3, 6)
    const a15 = recordingContext()
    drawZoneStates(a15.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 15)
    expect(arcFraction(a15.ops)).toBeCloseTo(0.3, 6)
    const a18 = recordingContext()
    drawZoneStates(a18.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 18)
    expect(arcFraction(a18.ops)).toBeCloseTo(0.75, 6)
  })

  it("aucun arc AVANT la rampe ni APRÈS son retour à zéro ; l'arc TIENT pendant un blocage", () => {
    const avant = recordingContext()
    drawZoneStates(avant.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 11)
    expect(count(avant.ops, 'arc')).toBe(0)
    // Frame 25 : la jauge est revenue à zéro (point de la frame 19) — rien à tracer.
    const retombe = recordingContext()
    drawZoneStates(retombe.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 25)
    expect(count(retombe.ops, 'arc')).toBe(0)
    // Frame 45 : la capture est FIGÉE à 0,2 depuis la frame 32 — l'arc reste, à 0,2.
    const fige = recordingContext()
    drawZoneStates(fige.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 45)
    expect(count(fige.ops, 'arc')).toBe(1)
    expect(arcFraction(fige.ops)).toBeCloseTo(0.2, 6)
    // Une seconde après le DERNIER point de la série (62, retour à zéro), plus rien.
    const eteint = recordingContext()
    drawZoneStates(eteint.ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 62 + HOLD + 1)
    expect(count(eteint.ops, 'arc')).toBe(0)
  })

  // LA DÉCISION DU PLAN : sur un artefact qui ne porte pas `gauge` (schéma <= 16), il n'y a
  // PLUS D'ARC DU TOUT — même quand l'intervalle publie un sommet. Le sommet statique se lisait
  // comme une jauge ; mieux vaut rien.
  it("sans `gauge`, AUCUN arc — le sommet `progress` de l'intervalle ne le remplace pas", () => {
    const sansJauge = [{ ...ZONE_STATES[0], gauge: [] }]
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), sansJauge, VIEW, 14)
    expect(count(ops, 'arc')).toBe(0)
    // La colline active de la zone 1 publie un sommet (0,5) et aucune série : pas d'arc non plus.
    const colline = recordingContext()
    // (sur la BOÎTE : le contour d'un cylindre est lui-même un `arc`, ce qui brouillerait le compte)
    drawZoneStates(colline.ctx, layer([zones()[0]]), [{ ...ZONE_STATES[1], zoneRef: 0 }], VIEW, 30)
    expect(count(colline.ops, 'arc')).toBe(0)
  })

  it("l'arc prend l'encre du camp QUI CAPTURE (le camp d'en face du propriétaire)", () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), [ZONE_STATES[0]], VIEW, 14)
    // Le propriétaire à la frame 14 est le camp 0 (allié) : l'arc est ADVERSE.
    const inks = valuesOf(ops, 'strokeStyle') as unknown as string[]
    expect(inks[inks.length - 1]).toBe('#adverse')
  })

  it("propriétaire inconnu (zone neutre) : l'arc est NEUTRE, jamais une couleur devinée", () => {
    const neutre = [{ ...ZONE_STATES[0], gauge: [{ t: 2, v: 0.4 }] }]
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), neutre, VIEW, 3)
    expect(count(ops, 'arc')).toBe(1)
    const inks = valuesOf(ops, 'strokeStyle') as unknown as string[]
    expect(inks[inks.length - 1]).toBe('#neutre')
  })

  it("une rampe AVANT le premier intervalle se dessine quand même, à l'encre neutre", () => {
    const tot = [{ ...ZONE_STATES[0], spans: ZONE_STATES[0].spans.slice(1), gauge: [{ t: 2, v: 0.4 }] }]
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]]), tot, VIEW, 3)
    expect(count(ops, 'arc')).toBe(1)
    expect(count(ops, 'fill')).toBe(0)
    const inks = valuesOf(ops, 'strokeStyle') as unknown as string[]
    expect(inks[inks.length - 1]).toBe('#neutre')
  })

  it('camp inconnu (aucune ligne « moi ») : encre NEUTRE, jamais une couleur devinée', () => {
    const { ctx, ops } = recordingContext()
    const aveugle = { colorOfOwner: () => null, colorOfCapturer: () => null, neutral: '#neutre' }
    drawZoneStates(ctx, { ...layer([zones()[0]]), style: aveugle }, [ZONE_STATES[0]], VIEW, 14)
    // Aucun remplissage : une zone TENUE par un camp qu'on ne sait pas situer garde le liseré
    // seul. Les deux tracés sont le contour et l'arc de jauge, tous deux à l'encre neutre.
    expect(count(ops, 'fill')).toBe(0)
    expect(count(ops, 'stroke')).toBe(2)
    expect((valuesOf(ops, 'strokeStyle') as unknown as string[]).every((i) => i === '#neutre')).toBe(true)
  })

  it('sans état publié, le calque ne dessine rien du tout', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer(), [], VIEW, 10)
    expect(ops).toHaveLength(0)
  })

  // VERROU DE LA REVUE R1-7 : `zoneRef` est un index figé à la cuisson, la liste servie est
  // reconstruite à la requête. Quand le catalogue de l'artefact ne joint pas la liste servie,
  // le calque VIVANT ne touche PAS au contexte — pas un trait, pas même un `beginPath`.
  it('jointure REFUSÉE (catalogue différent de la liste servie) : le calque ne peint rien', () => {
    const { ctx, ops } = recordingContext()
    drawZoneStates(ctx, layer([zones()[0]], false), [ZONE_STATES[0]], VIEW, 14)
    expect(ops).toHaveLength(0)
  })
})
