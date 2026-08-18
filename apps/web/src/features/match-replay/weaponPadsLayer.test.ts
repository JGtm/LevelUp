/**
 * Tests — weaponPadsLayer : LES TROIS ÉTATS, LE COMPTE À REBOURS, LA TAILLE, LE SURVOL.
 *
 * CE QUE CE FICHIER VERROUILLE, et chaque point correspond à une clause écrite du plan :
 *  - PLEIN de `t0` à `tLow`, INCERTAIN jusqu'à `tHigh`, VIDE ensuite — et l'incertitude ne se
 *    masque PAS (l'icône reste, en fantôme). Une icône qui s'éteindrait pile à `tLow`
 *    affirmerait une datation que la source n'a pas ;
 *  - un socle « jamais vidé » (`tHigh` = fin du rejeu) finit INCERTAIN, jamais vide ;
 *  - AUCUN compte à rebours sans cycle établi — ni chiffre, ni tiret ;
 *  - la TAILLE suit l'arme : puissance grande, classique petite, et l'anneau est le repère
 *    qui reste quand le socle est vide ;
 *  - le SURVOL se rejoue sur la donnée, y compris sur un socle VIDE (c'est là qu'on inspecte).
 *
 * Le contexte enregistreur observe la GÉOMÉTRIE ÉMISE, jamais un pixel (cf. recordingContext).
 */
import { describe, expect, it } from 'vitest'

import { count, recordingContext, valuesOf } from './test/recordingContext'
import type { ReplayWeaponPadReady } from './replayNormalize'
import { worldToCanvas } from './replayLogic'
import {
  drawWeaponPadsLayer,
  padAt,
  padOccupancyAt,
  padRadiusPx,
  padRespawnSecondsAt,
  padStateAt,
  type PadStyle,
} from './weaponPadsLayer'

/** 10 m de côté sur 100 px : 10 px par mètre — le même cadrage que les tests de poses. */
const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 100, height: 100, pad: 0 }

/** Une image de 100 ms : 10 images = 1 s, ce qui rend les comptes lisibles à l'œil nu. */
const FRAME_MS = 100

/** Le S7 Sniper est une arme de PUISSANCE ; le BR75 une classique (weaponPadFamilies). */
const SNIPER = '0x0A1992BC'
const BR75 = '0x2B1824D5'
const KEYS: Record<string, string> = { [SNIPER]: 'hinf_s7_sniper', [BR75]: 'hinf_br75' }

const ICON = { width: 40, height: 16 } as unknown as CanvasImageSource

function style(over: Partial<PadStyle> = {}): PadStyle {
  return {
    ink: 'encre',
    labelStroke: 'contour',
    iconOf: () => ICON,
    scaleOf: (weapon) => (KEYS[weapon] === 'hinf_s7_sniper' ? 'power' : 'classic'),
    countdownLabel: (s) => `${Math.ceil(s)} s`,
    ...over,
  }
}

function pad(over: Partial<ReplayWeaponPadReady> = {}): ReplayWeaponPadReady {
  return {
    x: 5,
    y: 5,
    weapon: SNIPER,
    spawns: [0],
    presence: [{ t0: 0, tLow: 100, tHigh: 120 }],
    ...over,
  }
}

function draw(pads: ReplayWeaponPadReady[], frame: number, over: Partial<PadStyle> = {}) {
  const { ops, ctx } = recordingContext()
  drawWeaponPadsLayer(ctx, pads, VIEW, { frame, frameMs: FRAME_MS, k: 1 }, style(over))
  return ops
}

describe('padStateAt — les trois états, et leurs frontières', () => {
  const p = pad()

  it('PLEIN de l’apparition au dernier instant prouvé', () => {
    expect(padStateAt(p, 0)).toBe('full')
    expect(padStateAt(p, 99)).toBe('full')
  })

  it('INCERTAIN entre la dernière preuve de présence et la première preuve d’absence', () => {
    expect(padStateAt(p, 100)).toBe('uncertain')
    expect(padStateAt(p, 119)).toBe('uncertain')
  })

  it('VIDE dès que l’absence est prouvée', () => {
    expect(padStateAt(p, 120)).toBe('empty')
    expect(padStateAt(p, 500)).toBe('empty')
  })

  it('VIDE avant la première apparition : le socle n’a rien porté du tout', () => {
    const tardif = pad({ spawns: [200], presence: [{ t0: 200, tLow: 300, tHigh: 320 }] })
    expect(padStateAt(tardif, 0)).toBe('empty')
    expect(padOccupancyAt(tardif, 0)).toBeNull()
    expect(padStateAt(tardif, 200)).toBe('full')
  })

  it('une RÉAPPARITION reprend à plein : c’est la dernière occupation qui gouverne', () => {
    const deux = pad({
      spawns: [0, 300],
      presence: [
        { t0: 0, tLow: 100, tHigh: 120 },
        { t0: 300, tLow: 400, tHigh: 420 },
      ],
    })
    expect(padStateAt(deux, 250)).toBe('empty')
    expect(padStateAt(deux, 300)).toBe('full')
    expect(padOccupancyAt(deux, 350)?.t0).toBe(300)
  })

  it('un socle JAMAIS VIDÉ reste PLEIN jusqu’au bout : aucune absence n’a été prouvée', () => {
    // Cas mesuré sur `bcb6d393` : 8 occupations sur 28 s'achèvent ainsi (tHigh = tLow = fin).
    const jamais = pad({ presence: [{ t0: 0, tLow: 3464, tHigh: 3464 }] })
    expect(padStateAt(jamais, 3463)).toBe('full')
    expect(padStateAt(jamais, 3464)).toBe('full')
    expect(padRespawnSecondsAt(jamais, 3464, FRAME_MS)).toBeNull()
  })
})

describe('padRespawnSecondsAt — le compte à rebours n’existe qu’avec un cycle établi', () => {
  const cycle = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }

  it('SANS cycle : rien, à aucun instant — pas même un zéro', () => {
    const p = pad()
    expect(p.cycle).toBeUndefined()
    for (const f of [0, 100, 120, 200, 5000]) expect(padRespawnSecondsAt(p, f, FRAME_MS)).toBeNull()
  })

  it('AVEC cycle : il part de tHigh, la borne HAUTE de la disparition', () => {
    const p = pad({ cycle })
    // tHigh = 120 ; 40 s = 400 images. À l'instant de tHigh, tout le cycle reste.
    expect(padRespawnSecondsAt(p, 120, FRAME_MS)).toBeCloseTo(40, 6)
    expect(padRespawnSecondsAt(p, 320, FRAME_MS)).toBeCloseTo(20, 6)
  })

  it('AVEC cycle : rien tant que le socle n’est pas VIDE — l’attente n’a pas commencé', () => {
    const p = pad({ cycle })
    expect(padRespawnSecondsAt(p, 50, FRAME_MS)).toBeNull()
    expect(padRespawnSecondsAt(p, 110, FRAME_MS)).toBeNull()
  })

  it('le compte ÉPUISÉ s’efface : jamais un nombre négatif ni un zéro qui traîne', () => {
    const p = pad({ cycle })
    expect(padRespawnSecondsAt(p, 520, FRAME_MS)).toBeNull()
    expect(padRespawnSecondsAt(p, 900, FRAME_MS)).toBeNull()
  })

  it('une durée d’image inconnue n’invente pas de compte', () => {
    expect(padRespawnSecondsAt(pad({ cycle }), 200, 0)).toBeNull()
  })
})

describe('le tracé — anneau, vignette, fantôme, compte à rebours', () => {
  it('PLEIN : anneau PLEIN et vignette à pleine encre', () => {
    const ops = draw([pad()], 50)
    expect(count(ops, 'arc')).toBe(1)
    expect(count(ops, 'drawImage')).toBe(1)
    // `setLineDash([])` : l'anneau du socle plein n'est pas pointillé.
    const dashes = ops.filter((o) => o.op === 'setLineDash').map((o) => o.args[0] as number[])
    expect(dashes[0]).toEqual([])
    expect(Math.max(...valuesOf(ops, 'globalAlpha'))).toBeGreaterThan(0.9)
  })

  it('INCERTAIN : la vignette RESTE, en fantôme, et l’anneau passe au pointillé', () => {
    const ops = draw([pad()], 110)
    expect(count(ops, 'drawImage'), "l'incertitude ne se masque pas").toBe(1)
    const dashes = ops.filter((o) => o.op === 'setLineDash').map((o) => o.args[0] as number[])
    expect(dashes[0]?.length).toBeGreaterThan(0)
    // Aucune opacité pleine : ni l'anneau ni la vignette n'affirment une présence.
    expect(Math.max(...valuesOf(ops, 'globalAlpha').slice(0, -1))).toBeLessThan(0.5)
  })

  it('VIDE : plus de vignette, mais l’anneau reste — le lieu ne disparaît pas', () => {
    const ops = draw([pad()], 200)
    expect(count(ops, 'drawImage')).toBe(0)
    expect(count(ops, 'arc')).toBe(1)
  })

  it('VIDE avec cycle : le compte à rebours s’écrit, cerné pour rester lisible', () => {
    const cycle = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }
    const ops = draw([pad({ cycle })], 320)
    expect(ops.filter((o) => o.op === 'fillText').map((o) => o.args[0])).toEqual(['20 s'])
    expect(count(ops, 'strokeText')).toBe(1)
  })

  it('VIDE sans cycle : AUCUN texte — ni chiffre, ni tiret', () => {
    const ops = draw([pad()], 320)
    expect(count(ops, 'fillText')).toBe(0)
    expect(count(ops, 'strokeText')).toBe(0)
  })

  it('sans contour de thème, le compte à rebours s’écrit quand même (sans cerne)', () => {
    const cycle = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }
    const ops = draw([pad({ cycle })], 320, { labelStroke: '' })
    expect(count(ops, 'fillText')).toBe(1)
    expect(count(ops, 'strokeText')).toBe(0)
  })

  it('SANS VIGNETTE : un glyphe neutre, jamais l’icône d’une arme voisine', () => {
    const ops = draw([pad()], 50, { iconOf: () => null })
    expect(count(ops, 'drawImage')).toBe(0)
    // Deux arcs : l'anneau, puis le disque du glyphe.
    expect(count(ops, 'arc')).toBe(2)
    expect(count(ops, 'fill')).toBe(1)
  })

  it('la TAILLE suit l’arme : une arme de puissance est plus grande qu’une classique', () => {
    const puissance = draw([pad()], 50)
    const classique = draw([pad({ weapon: BR75 })], 50)
    const rayon = (ops: ReturnType<typeof draw>) =>
      (ops.find((o) => o.op === 'arc')?.args[2] as number) ?? 0
    expect(rayon(puissance)).toBeGreaterThan(rayon(classique))
    const hauteur = (ops: ReturnType<typeof draw>) =>
      (ops.find((o) => o.op === 'drawImage')?.args[4] as number) ?? 0
    expect(hauteur(puissance)).toBeGreaterThan(hauteur(classique))
  })

  it('la vignette est CENTRÉE sur la position monde du socle', () => {
    const ops = draw([pad({ x: 2, y: 8 })], 50)
    const c = worldToCanvas({ x: 2, y: 8 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const [, dx, dy, w, h] = ops.find((o) => o.op === 'drawImage')!.args as number[]
    expect(dx + w / 2).toBeCloseTo(c.x, 6)
    expect(dy + h / 2).toBeCloseTo(c.y, 6)
  })

  it('aucune COULEUR n’est écrite ici : les encres viennent toutes de l’appelant', () => {
    const ops = draw([pad()], 50)
    const encres = ops
      .filter((o) => o.op === 'set fillStyle' || o.op === 'set strokeStyle')
      .map((o) => o.args[0])
    expect(encres.length).toBeGreaterThan(0)
    for (const e of encres) expect(['encre', 'contour']).toContain(e)
  })

  it('aucun socle : rien n’est dessiné, pas même un cadre vide (témoin 000d5950)', () => {
    expect(draw([], 50)).toHaveLength(0)
  })
})

describe('padAt — le survol se rejoue sur la donnée', () => {
  const s = style()
  const at = (x: number, y: number) =>
    worldToCanvas({ x, y }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)

  it('trouve le socle sous le pointeur', () => {
    const p = pad()
    expect(padAt([p], VIEW, s, 1, at(5, 5))).toBe(p)
  })

  it('ne trouve rien loin du socle', () => {
    expect(padAt([pad()], VIEW, s, 1, at(1, 1))).toBeNull()
  })

  it('un socle VIDE se survole aussi : c’est là qu’on veut savoir ce qui manque', () => {
    const p = pad()
    expect(padStateAt(p, 300)).toBe('empty')
    expect(padAt([p], VIEW, s, 1, at(5, 5))).toBe(p)
  })

  it('le PLUS PROCHE l’emporte quand deux socles se recouvrent', () => {
    const a = pad({ x: 5, y: 5 })
    const b = pad({ x: 5.4, y: 5, weapon: BR75 })
    expect(padAt([a, b], VIEW, s, 1, at(5.4, 5))).toBe(b)
    expect(padAt([a, b], VIEW, s, 1, at(5, 5))).toBe(a)
  })

  it('la zone sensible ne descend pas sous un plancher visable', () => {
    // L'anneau d'une arme classique fait 5,5 px : la zone sensible est relevée à 9 px.
    expect(padRadiusPx(pad({ weapon: BR75 }), s, 1)).toBeLessThan(9)
    expect(padAt([pad({ weapon: BR75 })], VIEW, s, 1, { x: 50 + 8, y: 50 })).not.toBeNull()
  })
})
