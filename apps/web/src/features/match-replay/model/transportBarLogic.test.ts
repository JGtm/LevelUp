/**
 * Tests — les deux décisions pures de la barre de lecture d'une ligne : sur combien de lignes elle se range (seuil
 * et marge de bascule), et sous quelle forme elle se montre (surimpression, lecture, survol, focus, geste, préférence).
 */
import { describe, expect, it } from 'vitest'

import { COTES_BARRE_COMPACTE as C } from './cotesBarreCompacte'
import type { ReplayFrame } from './replayFrame'
import { MARGE_BASCULE_PX } from './rowLayout'
import {
  barForm,
  barLines,
  overlaysTransportBar,
  requiredWidthOneLine,
  type BarFormInput,
} from './transportBarLogic'

/** Quatorze groupes de 40 px : de quoi poser un seuil lisible. */
const GROUPS = Array.from({ length: 14 }, () => 40)
const REQUIRED = requiredWidthOneLine(GROUPS)

describe('requiredWidthOneLine — la frise garde sa largeur minimale', () => {
  it('somme les groupes, un écart par élément, les deux marges, et la frise à son minimum avec ses marges', () => {
    expect(REQUIRED).toBe(
      14 * 40 + 14 * C.ecartPx + 2 * C.margeXPx + C.friseLargeurMinPx + 2 * C.friseMargePx,
    )
  })

  it('sans aucun groupe, il reste la frise et les marges de la barre', () => {
    expect(requiredWidthOneLine([])).toBe(2 * C.margeXPx + C.friseLargeurMinPx + 2 * C.friseMargePx)
  })
})

describe('barLines — une ligne tant que la frise tient, sinon deux', () => {
  it('juste à la largeur requise : une ligne', () => {
    expect(barLines({ availableWidth: REQUIRED, groupWidths: GROUPS }, null)).toBe(1)
  })

  it('un pixel sous la largeur requise : deux lignes', () => {
    expect(barLines({ availableWidth: REQUIRED - 1, groupWidths: GROUPS }, null)).toBe(2)
  })

  it('une ligne mémorisée la quitte dès que la largeur requise manque : aucune marge de ce côté', () => {
    expect(barLines({ availableWidth: REQUIRED - 1, groupWidths: GROUPS }, 1)).toBe(2)
    expect(barLines({ availableWidth: REQUIRED, groupWidths: GROUPS }, 1)).toBe(1)
  })

  it('deux lignes mémorisées ne reviennent à une ligne qu\'avec la marge de bascule en plus', () => {
    expect(barLines({ availableWidth: REQUIRED, groupWidths: GROUPS }, 2)).toBe(2)
    expect(barLines({ availableWidth: REQUIRED + MARGE_BASCULE_PX - 1, groupWidths: GROUPS }, 2)).toBe(2)
    expect(barLines({ availableWidth: REQUIRED + MARGE_BASCULE_PX, groupWidths: GROUPS }, 2)).toBe(1)
  })

  it('une largeur qui oscille autour du seuil ne fait pas battre la barre', () => {
    let lines = barLines({ availableWidth: REQUIRED - 1, groupWidths: GROUPS }, null)
    for (const delta of [0, 5, -1, 10, -3, 20, 0]) {
      lines = barLines({ availableWidth: REQUIRED + delta, groupWidths: GROUPS }, lines)
      expect(lines).toBe(2)
    }
  })

  it('une largeur très étroite : deux lignes, avec ou sans mémoire', () => {
    expect(barLines({ availableWidth: 120, groupWidths: GROUPS }, null)).toBe(2)
    expect(barLines({ availableWidth: 120, groupWidths: GROUPS }, 1)).toBe(2)
    expect(barLines({ availableWidth: 120, groupWidths: GROUPS }, 2)).toBe(2)
  })

  it('une largeur ou des groupes pas encore mesurés : une ligne', () => {
    expect(barLines({ availableWidth: 0, groupWidths: GROUPS }, null)).toBe(1)
    expect(barLines({ availableWidth: 0, groupWidths: GROUPS }, 2)).toBe(1)
    expect(barLines({ availableWidth: 900, groupWidths: [] }, null)).toBe(1)
  })

  it('un groupe de plus (« Plan tourné ») relève le seuil', () => {
    const plusUn = [...GROUPS, 118]
    expect(requiredWidthOneLine(plusUn)).toBeGreaterThan(REQUIRED)
    expect(barLines({ availableWidth: REQUIRED, groupWidths: plusUn }, null)).toBe(2)
  })
})

const FRAMES: readonly ReplayFrame[] = ['standard', 'cinema', 'plein-cadre']

function input(over: Partial<BarFormInput> = {}): BarFormInput {
  return { frame: 'plein-cadre', thinPreferred: true, playing: true, hovered: false, focused: false, recentGesture: false, ...over }
}

describe('overlaysTransportBar — la barre n\'est posée sur le plan qu\'au Plein cadre, préférence allumée', () => {
  it.each(FRAMES)('%s', (frame) => {
    expect(overlaysTransportBar(frame, true)).toBe(frame === 'plein-cadre')
    expect(overlaysTransportBar(frame, false)).toBe(false)
  })
})

describe('barForm — fine seulement en lecture, au repos, au Plein cadre', () => {
  it('lecture, ni survol, ni focus, ni geste : fine', () => {
    expect(barForm(input())).toBe('fine')
  })

  it('pause : pleine', () => {
    expect(barForm(input({ playing: false }))).toBe('pleine')
  })

  it('survol : pleine', () => {
    expect(barForm(input({ hovered: true }))).toBe('pleine')
  })

  it('focus du clavier dans la barre : pleine, un élément focalisé n\'est jamais dans la forme réduite', () => {
    expect(barForm(input({ focused: true }))).toBe('pleine')
  })

  it('touche récente (ou survol tout juste quitté) : pleine', () => {
    expect(barForm(input({ recentGesture: true }))).toBe('pleine')
  })

  it('préférence éteinte : toujours pleine, même en lecture et au repos', () => {
    expect(barForm(input({ thinPreferred: false }))).toBe('pleine')
  })

  it.each(['standard', 'cinema'] as const)('%s : toujours pleine, la barre y est fixe', (frame) => {
    expect(barForm(input({ frame }))).toBe('pleine')
  })
})
