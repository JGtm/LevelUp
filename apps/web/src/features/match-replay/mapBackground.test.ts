/**
 * mapBackground.test.ts — la pose du fond de carte.
 *
 * Le calage employé ici est celui de RIDGELINE (Cliffhanger), lu dans l'asset versionné
 * `data/titles/halo_infinite/reference/map_backgrounds/ridgeline.json`, et les bornes sont
 * celles de l'artefact de rejeu `000d5950`. Des chiffres inventés ne diraient rien de la
 * seule superposition qu'on puisse aller vérifier à l'écran.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayBounds, ReplayMapBackgroundCalibration } from '@/lib/api/types'

import { backgroundRect, coversPlayedArea, worldExtent, worldToPixel } from './mapBackground'

const RIDGELINE: ReplayMapBackgroundCalibration = {
  metersPerPixel: 0.092,
  originX: -57.30403518676758,
  originY: 78.87359236145019,
  widthPx: 1633,
  heightPx: 1627,
  convention: 'xMonde = originX + (px + 0.5) * metersPerPixel ; yMonde = originY - (py + 0.5) * metersPerPixel',
}

/** Zone parcourue par les joueurs sur 000d5950 (bounds de l'artefact). */
const JOUE: ReplayBounds = { minX: -7.02, minY: -25.14, maxX: 43.79, maxY: 30.35 }

describe('worldExtent', () => {
  it("étend l'image vers l'est et vers le sud depuis le coin haut-gauche", () => {
    const ext = worldExtent(RIDGELINE)
    expect(ext.minX).toBeCloseTo(-57.304, 3)
    expect(ext.maxX).toBeCloseTo(-57.304 + 1633 * 0.092, 3)
    expect(ext.maxY).toBeCloseTo(78.874, 3)
    expect(ext.minY).toBeCloseTo(78.874 - 1627 * 0.092, 3)
    // L'ordre est le point : maxY est en HAUT, minY en bas — l'image descend quand le
    // monde monte. L'inverser retournerait la carte.
    expect(ext.maxY).toBeGreaterThan(ext.minY)
  })
})

describe('worldToPixel', () => {
  it("applique la convention publiée dans le sidecar", () => {
    // Le coin haut-gauche du monde tombe sur le pixel (0,0).
    expect(worldToPixel(RIDGELINE, RIDGELINE.originX, RIDGELINE.originY)).toMatchObject({
      px: 0,
      py: 0,
      inside: true,
    })
    // Un point 9,2 m à l'est et 9,2 m au sud tombe 100 pixels plus loin sur chaque axe.
    const p = worldToPixel(RIDGELINE, RIDGELINE.originX + 9.2, RIDGELINE.originY - 9.2)
    expect(p.px).toBe(100)
    expect(p.py).toBe(100)
    expect(p.inside).toBe(true)
  })

  it('signale le hors-cadre sans ramener le point au bord', () => {
    const p = worldToPixel(RIDGELINE, RIDGELINE.originX - 50, RIDGELINE.originY)
    expect(p.inside).toBe(false)
    expect(p.px).toBeLessThan(0)
  })

  it('refuse un calage sans échelle', () => {
    expect(worldToPixel({ ...RIDGELINE, metersPerPixel: 0 }, 0, 0).inside).toBe(false)
  })
})

describe('coversPlayedArea', () => {
  it('accepte le couple réel Cliffhanger / 000d5950', () => {
    expect(coversPlayedArea(RIDGELINE, JOUE)).toBe(true)
  })

  it("REFUSE un fond qui ne contient pas le terrain — c'est le témoin de repères disjoints", () => {
    // Même image, décalée de 300 m : elle ne recouvre plus rien de ce qui a été joué.
    const decale = { ...RIDGELINE, originX: RIDGELINE.originX + 300 }
    expect(coversPlayedArea(decale, JOUE)).toBe(false)
    // Et une image dix fois trop petite non plus (facteur d'échelle d'une autre carte).
    expect(coversPlayedArea({ ...RIDGELINE, metersPerPixel: 0.0092 }, JOUE)).toBe(false)
  })
})

describe('backgroundRect', () => {
  const W = 800
  const H = 480
  const PAD = 24

  it('pose les deux coins de l\'image dans le même cadrage que les trajectoires', () => {
    const rect = backgroundRect(RIDGELINE, JOUE, W, H, PAD)
    expect(rect).not.toBeNull()
    if (!rect) return
    // L'image couvre 150 m de large pour une zone jouée de 50,8 m : le rectangle dessiné
    // déborde donc largement du canvas, et déborde des DEUX côtés (la zone jouée est à
    // l'intérieur). C'est exactement ce que le canvas est censé rogner.
    expect(rect.x).toBeLessThan(0)
    expect(rect.y).toBeLessThan(0)
    expect(rect.x + rect.width).toBeGreaterThan(W)
    expect(rect.y + rect.height).toBeGreaterThan(H)
  })

  it("garde le ratio de l'image : un pixel reste carré", () => {
    const rect = backgroundRect(RIDGELINE, JOUE, W, H, PAD)
    if (!rect) throw new Error('rect attendu')
    const ratioImage = RIDGELINE.widthPx / RIDGELINE.heightPx
    expect(rect.width / rect.height).toBeCloseTo(ratioImage, 5)
  })

  it('place la zone jouée AU BON ENDROIT dans le rectangle du fond', () => {
    // LE CONTRÔLE QUI COMPTE : le coin bas-gauche de la zone jouée doit retomber, dans le
    // rectangle du fond, à la position que la convention pixel lui assigne. C'est ce qui
    // dit que le fond et les trajectoires parlent du même repère.
    const rect = backgroundRect(RIDGELINE, JOUE, W, H, PAD)
    if (!rect) throw new Error('rect attendu')
    const { px, py } = worldToPixel(RIDGELINE, JOUE.minX, JOUE.minY)
    const attenduX = rect.x + (px / RIDGELINE.widthPx) * rect.width
    const attenduY = rect.y + (py / RIDGELINE.heightPx) * rect.height

    // Position du même point par la projection des trajectoires (worldToCanvas), reproduite
    // ici sans dépendre de son implémentation : cadrage « fit » centré avec marge.
    const bw = JOUE.maxX - JOUE.minX
    const bh = JOUE.maxY - JOUE.minY
    const scale = Math.min((W - 2 * PAD) / bw, (H - 2 * PAD) / bh)
    const offsetX = (W - bw * scale) / 2
    const offsetY = (H - bh * scale) / 2
    const trajX = offsetX + (JOUE.minX - JOUE.minX) * scale
    const trajY = offsetY + (JOUE.maxY - JOUE.minY) * scale

    // Tolérance d'un pixel image : worldToPixel tronque à l'entier.
    const tol = (rect.width / RIDGELINE.widthPx) * 1.5
    expect(Math.abs(attenduX - trajX)).toBeLessThan(tol)
    expect(Math.abs(attenduY - trajY)).toBeLessThan(tol)
  })

  it('rend null sur un calage inexploitable plutôt que de poser le fond au hasard', () => {
    expect(backgroundRect({ ...RIDGELINE, metersPerPixel: 0 }, JOUE, W, H, PAD)).toBeNull()
    expect(backgroundRect({ ...RIDGELINE, widthPx: 0 }, JOUE, W, H, PAD)).toBeNull()
    expect(backgroundRect({ ...RIDGELINE, heightPx: 0 }, JOUE, W, H, PAD)).toBeNull()
  })
})
