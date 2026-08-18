/**
 * Tests — placementShapes : l'arithmétique des formes de pose, et leur tracé.
 *
 * TROIS FAMILLES SONT NÉES DANS CE FICHIER (lot du 2026-08-18) — la balise du translocateur,
 * le traqueur de menaces, le champ de réparation — et ce que ces tests verrouillent, c'est ce
 * que chacune AFFIRME :
 *  - la BALISE ne pulse pas et ne pointe nulle part (rien de mesuré ne bat, aucune orientation
 *    n'est publiée) ;
 *  - le TRAQUEUR n'émet QU'UNE impulsion, qui ne se rejoue jamais — c'est sa différence de
 *    nature avec le ping du capteur, et elle est testée à l'âge d'un second ping ;
 *  - le CHAMP ne diffuse aucune ONDE, et sa borne est pointillée parce que son rayon est
 *    déclaré ; depuis le 2026-08-18 il porte une CROIX DE PHARMACIE qui respire — un signe
 *    d'activité, jamais une cadence (le film n'en publie aucune).
 */
import { describe, expect, it } from 'vitest'

import {
  beaconDiamond,
  REPAIR_FIELD_RADIUS_M,
  repairCrossAlpha,
  SEEKER_IMPULSE_MS,
  seekerImpulse,
  seekerImpulseActive,
} from './placementShapes'
import {
  BEACON_ID,
  draw,
  FIELD_ID,
  painted,
  pose,
  projected,
  SEEKER_ID,
  TIME,
} from './test/placementFixtures'
import { SENSOR_PING_MS, SENSOR_RADIUS_M, SENSOR_SWEEP_MS } from './threatSensor'

describe('seekerImpulseActive — UNE impulsion, et sa fenêtre ne se rejoue pas', () => {
  it('ouverte à l’âge zéro, fermée au terme de sa course', () => {
    expect(seekerImpulseActive(0)).toBe(true)
    expect(seekerImpulseActive(SEEKER_IMPULSE_MS - 1)).toBe(true)
    expect(seekerImpulseActive(SEEKER_IMPULSE_MS)).toBe(false)
  })

  it('un âge négatif (pose pas encore née) ne l’ouvre pas', () => {
    expect(seekerImpulseActive(-1)).toBe(false)
  })

  it('elle NE REVIENT JAMAIS — c’est la différence de nature avec le ping du capteur', () => {
    // Le capteur repinge à chaque période ; le traqueur, lui, reste éteint pour toujours.
    for (const t of [SENSOR_PING_MS, 2 * SENSOR_PING_MS, 60_000]) {
      expect(seekerImpulseActive(t), `age ${t}`).toBe(false)
      expect(seekerImpulse(t, false), `age ${t}`).toBeNull()
    }
  })
})

describe('seekerImpulse — la course de l’onde', () => {
  it('part du centre et atteint le bord : monotone, jamais au-delà de 1', () => {
    const reaches = [0, 0.25, 0.5, 0.75, 0.99].map(
      (f) => seekerImpulse(f * SEEKER_IMPULSE_MS, false)?.reach ?? -1,
    )
    expect(reaches[0]).toBeCloseTo(0, 6)
    for (let i = 1; i < reaches.length; i++) expect(reaches[i]).toBeGreaterThan(reaches[i - 1])
    expect(reaches[reaches.length - 1]).toBeLessThan(1)
  })

  it('s’efface en s’ouvrant : l’opacité décroît quand la course avance', () => {
    const a = seekerImpulse(0, false)
    const b = seekerImpulse(SEEKER_IMPULSE_MS * 0.9, false)
    expect(a && b && a.alpha > b.alpha).toBe(true)
  })

  it('FONCTION DU TEMPS : le même âge rend toujours la même image', () => {
    expect(seekerImpulse(123, false)).toEqual(seekerImpulse(123, false))
  })

  it('mouvement réduit : un anneau plein et immobile, pas d’onde', () => {
    const debut = seekerImpulse(1, true)
    const fin = seekerImpulse(SEEKER_IMPULSE_MS - 1, true)
    expect(debut?.reach).toBe(1)
    expect(fin?.reach).toBe(1)
    expect(fin?.alpha).toBe(debut?.alpha)
  })

  it('elle emprunte le RYTHME de l’onde du capteur, et c’est écrit', () => {
    expect(SEEKER_IMPULSE_MS).toBe(SENSOR_SWEEP_MS)
  })
})

describe('beaconDiamond — un losange qui ne pointe nulle part', () => {
  it('quatre sommets, tous à la même distance du centre', () => {
    const c = { x: 30, y: -12 }
    const pts = beaconDiamond(c, 5)
    expect(pts).toHaveLength(4)
    for (const p of pts) expect(Math.hypot(p.x - c.x, p.y - c.y)).toBeCloseTo(5, 6)
  })

  it('symétrique par ses deux axes : son barycentre EST le centre', () => {
    const c = { x: 4, y: 9 }
    const pts = beaconDiamond(c, 7)
    expect(pts.reduce((s, p) => s + p.x, 0) / 4).toBeCloseTo(c.x, 6)
    expect(pts.reduce((s, p) => s + p.y, 0) / 4).toBeCloseTo(c.y, 6)
  })
})

describe('la BALISE du translocateur — un marqueur à demeure', () => {
  const beacon = () => pose({ family: 'translocator_beacon', id: BEACON_ID })

  it('un losange fermé et son cœur, à la position de la pose', () => {
    const ops = draw([beacon()])
    const c = projected(5, 5)
    expect(ops.filter((o) => o.op === 'closePath')).toHaveLength(1)
    expect(ops.filter((o) => o.op === 'lineTo')).toHaveLength(3)
    const dot = ops.filter((o) => o.op === 'arc')
    expect(dot).toHaveLength(1)
    expect(dot[0].args.slice(0, 2)).toEqual([c.x, c.y])
  })

  it('elle NE PULSE PAS : la même image à tout âge de sa fenêtre', () => {
    const sommets = [10, 50, 100].map((frame) =>
      draw([beacon()], { ...TIME, frame })
        .filter((o) => o.op === 'lineTo')
        .map((o) => o.args[0] as number),
    )
    expect(sommets[1]).toEqual(sommets[0])
    expect(sommets[2]).toEqual(sommets[0])
  })

  it('le losange ne pointe nulle part, une fois tracé comme au calcul', () => {
    const c = projected(5, 5)
    const sommets = draw([beacon()])
      .filter((o) => o.op === 'moveTo' || o.op === 'lineTo')
      .map((o) => ({ x: o.args[0] as number, y: o.args[1] as number }))
    expect(sommets).toHaveLength(4)
    // La somme des écarts au centre est nulle sur les deux axes : aucun biais directionnel.
    expect(sommets.reduce((s, p) => s + (p.x - c.x), 0)).toBeCloseTo(0, 6)
    expect(sommets.reduce((s, p) => s + (p.y - c.y), 0)).toBeCloseTo(0, 6)
  })
})

describe('le TRAQUEUR de menaces — une seule impulsion, puis plus rien', () => {
  const seeker = () => pose({ family: 'threat_seeker', id: SEEKER_ID, t0: 50, t1: 200 })

  it("une onde pendant l'impulsion, et une seule", () => {
    // 2 images après t0 = 200 ms, à mi-course des 400 ms de l'impulsion.
    const ops = draw([seeker()], { ...TIME, frame: 52 })
    expect(ops.filter((o) => o.op === 'arc')).toHaveLength(1)
    expect(ops.filter((o) => o.op === 'stroke')).toHaveLength(1)
    // Aucun remplissage : le traqueur n'a pas de ZONE, on ne lui invente pas de portée.
    expect(ops.filter((o) => o.op === 'fill')).toHaveLength(0)
  })

  it("l'impulsion PASSÉE, plus rien du tout — ni zone, ni anneau, ni point", () => {
    const apres = 50 + SEEKER_IMPULSE_MS / TIME.frameMs
    for (const frame of [apres, apres + 10, 200]) {
      expect(painted([seeker()], { ...TIME, frame }), `image ${frame}`).toBe(0)
    }
  })

  it('elle ne se rejoue pas : rien à l’âge d’un second ping de capteur (1,8 s)', () => {
    expect(painted([seeker()], { ...TIME, frame: 50 + SENSOR_PING_MS / TIME.frameMs })).toBe(0)
  })

  it('mouvement réduit : un anneau IMMOBILE au rayon plein, la famille reste visible', () => {
    const time = { ...TIME, frame: 52, reducedMotion: true }
    const arcs = draw([seeker()], time).filter((o) => o.op === 'arc')
    expect(arcs).toHaveLength(1)
    const suivant = draw([seeker()], { ...time, frame: 53 }).filter((o) => o.op === 'arc')
    // Le rayon ne bouge pas d'une image à l'autre : aucune course.
    expect(suivant[0].args[2]).toBe(arcs[0].args[2])
  })
})

describe('le CHAMP de réparation — un disque au rayon déclaré', () => {
  const field = () => pose({ family: 'repair_field', id: FIELD_ID })

  it('un disque et son anneau, au rayon DÉCLARÉ converti à l’échelle', () => {
    const ops = draw([field()])
    const arcs = ops.filter((o) => o.op === 'arc')
    expect(arcs).toHaveLength(2)
    for (const a of arcs) expect(a.args[2]).toBeCloseTo(REPAIR_FIELD_RADIUS_M * 10, 6)
    expect(ops.some((o) => o.op === 'fill')).toBe(true)
  })

  it('sa borne est POINTILLÉE : ce rayon n’est pas une valeur publiée', () => {
    expect(draw([field()]).some((o) => o.op === 'setLineDash')).toBe(true)
  })

  it('il reste SOUS la portée officielle du capteur : les deux disques ne se confondent pas', () => {
    expect(REPAIR_FIELD_RADIUS_M).toBeGreaterThan(0)
    expect(REPAIR_FIELD_RADIUS_M).toBeLessThan(SENSOR_RADIUS_M)
  })

  it('aucune onde, à aucun âge : un champ soigne, il n’émet pas', () => {
    for (const frame of [10, 52, 60, 100]) {
      const arcs = draw([field()], { ...TIME, frame }).filter((o) => o.op === 'arc')
      expect(arcs, `image ${frame}`).toHaveLength(2)
    }
  })

  /**
   * V8 (demande utilisateur du 2026-08-18) — LA CROIX DE PHARMACIE QUI RESPIRE.
   *
   * Ce qui est vérifié : la croix EXISTE (deux rectangles, un seul remplissage), elle tient
   * DANS le cercle, et sa respiration ne touche QUE son opacité — la zone, elle, ne bouge pas.
   */
  it('porte une CROIX : deux rectangles croisés, entièrement dans le cercle', () => {
    const ops = draw([field()])
    const rects = ops.filter((o) => o.op === 'rect')
    expect(rects).toHaveLength(2)
    const rayon = REPAIR_FIELD_RADIUS_M * 10
    for (const r of rects) {
      const [x, y, w, h] = r.args as number[]
      const centre = projected(5, 5)
      expect(Math.hypot(x - centre.x, y - centre.y)).toBeLessThan(rayon)
      expect(Math.hypot(x + w - centre.x, y + h - centre.y)).toBeLessThan(rayon)
    }
  })

  it('la respiration ne touche QUE la croix : les deux arcs gardent leur rayon', () => {
    for (const frame of [10, 20, 30]) {
      const arcs = draw([field()], { ...TIME, frame }).filter((o) => o.op === 'arc')
      for (const a of arcs) expect(a.args[2]).toBeCloseTo(REPAIR_FIELD_RADIUS_M * 10, 6)
    }
  })

  it('repairCrossAlpha respire entre deux bornes, et se REJOUE à l’identique', () => {
    const a0 = repairCrossAlpha(0, false)
    const a900 = repairCrossAlpha(900, false)
    expect(a900).toBeGreaterThan(a0)
    // Fonction du temps, pas animation à état : une période plus tard, la même image.
    expect(repairCrossAlpha(1_800, false)).toBeCloseTo(a0, 6)
    for (const t of [0, 300, 900, 1_500, 5_000]) {
      expect(repairCrossAlpha(t, false)).toBeGreaterThanOrEqual(0.55)
      expect(repairCrossAlpha(t, false)).toBeLessThanOrEqual(0.95)
    }
  })

  it('mouvement réduit : la croix NE respire plus, mais elle reste bien visible', () => {
    expect(repairCrossAlpha(0, true)).toBe(0.95)
    expect(repairCrossAlpha(900, true)).toBe(0.95)
  })
})
