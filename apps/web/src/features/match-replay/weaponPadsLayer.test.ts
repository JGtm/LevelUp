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

const IMAGE = { width: 40, height: 16 } as unknown as CanvasImageSource
/** Une vignette prête à poser : son corps et son liseré (cf. PadIcon). */
const ICON = { fill: IMAGE, outline: IMAGE }

function style(over: Partial<PadStyle> = {}): PadStyle {
  return {
    ink: 'encre',
    fill: 'remplissage',
    outline: 'contour',
    iconOf: () => ICON,
    scaleOf: (weapon) => (KEYS[weapon] === 'hinf_s7_sniper' ? 'power' : 'classic'),
    // A13 : l'encre de la NATURE du socle, résolue par l'appelant comme toutes les autres.
    inkOf: () => 'nature',
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

/**
 * LE TRACÉ, VERSION UNIQUE (verdict du 2026-08-18) : un POINT qui dit l'état, la VIGNETTE
 * dessous (remplie et cernée), le COMPTE À REBOURS dessus. L'anneau qui enfermait la vignette
 * n'existe plus.
 */
describe('le tracé — point, vignette dessous, compte à rebours dessus', () => {
  // DEUX ARCS DEPUIS A13, ET C'EST UNE SEULE MARQUE : le point, puis la BORDURE de sa nature
  // posée autour. Ils sont CONCENTRIQUES — ce que le cas « un socle, une marque » vérifie chez
  // l'appelant en comptant les centres distincts. Le point, lui, garde sa règle : plein.
  it('PLEIN : le point est REMPLI, la bordure l’entoure, et la vignette à pleine encre', () => {
    const ops = draw([pad()], 50)
    expect(count(ops, 'arc')).toBe(2)
    expect(count(ops, 'drawImage')).toBeGreaterThan(0)
    // Un point PLEIN se remplit : il n'est ni tracé ni pointillé. Le seul `stroke` est la bordure.
    expect(count(ops, 'setLineDash')).toBe(0)
    expect(count(ops, 'stroke')).toBe(1)
    expect(count(ops, 'fill')).toBe(1)
    expect(Math.max(...valuesOf(ops, 'globalAlpha'))).toBeGreaterThan(0.9)
  })

  it('INCERTAIN : la vignette RESTE, en fantôme, et le point passe au pointillé', () => {
    const ops = draw([pad()], 110)
    expect(count(ops, 'drawImage'), "l'incertitude ne se masque pas").toBeGreaterThan(0)
    const dashes = ops.filter((o) => o.op === 'setLineDash').map((o) => o.args[0] as number[])
    expect(dashes[0]?.length).toBeGreaterThan(0)
    expect(count(ops, 'fill')).toBe(0)
    // Aucune opacité pleine : ni le point ni la vignette n'affirment une présence.
    expect(Math.max(...valuesOf(ops, 'globalAlpha').slice(0, -1))).toBeLessThan(0.6)
  })

  it('VIDE : plus de vignette, mais le point ET sa bordure restent — le lieu ne disparaît pas', () => {
    const ops = draw([pad()], 200)
    expect(count(ops, 'drawImage')).toBe(0)
    // Le point et sa bordure, concentriques : la marque du lieu tient sans ce qu'il portait.
    expect(count(ops, 'arc')).toBe(2)
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
    const ops = draw([pad({ cycle })], 320, { outline: '' })
    expect(count(ops, 'fillText')).toBe(1)
    expect(count(ops, 'strokeText')).toBe(0)
  })

  // AMENDÉ LE 2026-08-26 (retour utilisateur : « j'ai l'impression qu'il y en a deux »). Ce cas
  // verrouillait le GLYPHE DE REPLI : sans vignette, le calque posait un disque plein du même
  // rayon que le point, une dizaine de pixels sous lui. Deux ronds empilés pour UN socle — et
  // c'est exactement ce que l'œil comptait deux fois. Le repli est supprimé : le point porte
  // déjà la position ET l'état. Ce qui reste garanti, et qui était le vrai sujet du cas :
  // aucune image n'est empruntée à une arme voisine.
  it('SANS VIGNETTE : le point SEUL, et surtout jamais l’icône d’une arme voisine', () => {
    const ops = draw([pad()], 50, { iconOf: () => null })
    expect(count(ops, 'drawImage')).toBe(0)
    // Le point et sa bordure, CONCENTRIQUES : une seule marque pour un seul socle. Ce qui a
    // disparu, c'est le disque décalé — pas un second cercle au même endroit.
    expect(count(ops, 'arc')).toBe(2)
    expect(count(ops, 'fill')).toBe(1)
  })

  /**
   * LE LISERÉ : la même forme reposée tout autour, à l'encre du FOND. C'est lui qui détache la
   * vignette d'un fond de carte clair comme d'un fond sombre — le « contour noir » demandé.
   */
  it('la vignette est CERNÉE : son liseré est reposé huit fois autour du corps', () => {
    const ops = draw([pad()], 50)
    expect(count(ops, 'drawImage')).toBe(9)
  })

  it('une image FINIE (non masque) se pose telle quelle, sans liseré inventé', () => {
    const ops = draw([pad()], 50, { iconOf: () => ({ fill: IMAGE, outline: null }) })
    expect(count(ops, 'drawImage')).toBe(1)
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

  /**
   * LA GÉOMÉTRIE DE LA VERSION UNIQUE : le POINT est à la position monde du socle, la
   * VIGNETTE entièrement SOUS lui, le COMPTE À REBOURS entièrement AU-DESSUS. L'ordre
   * vertical EST la règle du verdict.
   */
  it('point au socle, et compte à rebours DESSUS', () => {
    const cycle = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }
    const ops = draw([pad({ x: 2, y: 8, cycle })], 320)
    const c = worldToCanvas({ x: 2, y: 8 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    const [px, py, rayon] = ops.find((o) => o.op === 'arc')!.args as number[]
    expect(px).toBeCloseTo(c.x, 6)
    expect(py).toBeCloseTo(c.y, 6)
    const [, , ty] = ops.find((o) => o.op === 'fillText')!.args as number[]
    expect(ty).toBeLessThan(c.y - rayon)
  })

  // RÈGLE INVERSÉE LE 2026-08-26, et c'est le même retour utilisateur. La vignette était posée
  // ENTIÈREMENT SOUS le point : un rond ici, une image dix pixels plus bas, deux marques pour un
  // socle. Elle est désormais CENTRÉE sur le point — un socle, une marque. Le point ne disparaît
  // pas pour autant : il reste dessous et continue de porter l'ÉTAT par sa forme (plein,
  // pointillé, discret), ce que la vignette ne sait pas dire.
  it('la vignette est CENTRÉE sur le point, jamais décalée dessous', () => {
    const ops = draw([pad({ x: 2, y: 8 })], 50)
    const c = worldToCanvas({ x: 2, y: 8 }, VIEW.bounds, VIEW.width, VIEW.height, VIEW.pad)
    // Le CORPS est le dernier posé (le liseré vient d'abord, tout autour).
    const images = ops.filter((o) => o.op === 'drawImage')
    const [, dx, dy, w, h] = images[images.length - 1].args as number[]
    expect(dx + w / 2).toBeCloseTo(c.x, 6)
    expect(dy + h / 2).toBeCloseTo(c.y, 6)
  })

  it('aucune COULEUR n’est écrite ici : les encres viennent toutes de l’appelant', () => {
    const cycle = { medianS: 40, p10S: 40, p90S: 40, gaps: 2, missing: 0 }
    const ops = draw([pad({ cycle })], 320)
    const encres = ops
      .filter((o) => o.op === 'set fillStyle' || o.op === 'set strokeStyle')
      .map((o) => o.args[0])
    expect(encres.length).toBeGreaterThan(0)
    // `nature` rejoint la liste depuis A13 : c'est l'encre de la FAMILLE du socle, et elle
    // vient de l'appelant exactement comme les trois autres — ce cas garde son sujet entier.
    for (const e of encres) expect(['encre', 'remplissage', 'contour', 'nature']).toContain(e)
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
