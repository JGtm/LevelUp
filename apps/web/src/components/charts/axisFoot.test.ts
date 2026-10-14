import { describe, expect, it } from 'vitest'

import { footLabelStyle, labelsHeight, layoutAxisFoot, lineHeightFor, type AxisFootSpec } from './axisFoot'
import { axisFootProblems } from './axisFoot.check'

const DATES = { lines: 1, lineHeight: 14, margin: 4 }
const MIX = { lines: 2, lineHeight: 12, margin: 6 }

describe('layoutAxisFoot', () => {
  it('somme les blocs dans l’ordre : étiquettes, écart, bande, étiquettes, marge de pied', () => {
    const spec: AxisFootSpec = { above: DATES, band: { height: 10, gap: 4 }, below: MIX, pad: 4 }
    const l = layoutAxisFoot(spec)
    // 18 (dates) + 4 + 10 (bande) + 30 (modes) + 4 = 66
    expect(l.gridBottom).toBe(66)
    expect(l.bandHeight).toBe(10)
    // La bande finit 32 px sous la grille ; il reste 34 px sous elle.
    expect(l.bandBottom).toBe(34)
  })

  it('sans bande ni second bloc : les seules étiquettes et la marge de pied', () => {
    expect(layoutAxisFoot({ above: DATES, band: null, below: null, pad: 8 })).toEqual({
      gridBottom: 26,
      bandBottom: 0,
      bandHeight: 0,
    })
  })

  it('la bande seule : rien à réserver pour des étiquettes absentes', () => {
    expect(layoutAxisFoot({ above: null, band: { height: 10, gap: 8 }, below: null, pad: 12 })).toEqual({
      gridBottom: 30,
      bandBottom: 12,
      bandHeight: 10,
    })
  })

  it('hauteur de ligne dérivée de la police, marge et lignes de l’axe écrites telles que réservées', () => {
    expect(lineHeightFor(10.5)).toBe(14)
    expect(labelsHeight(MIX)).toBe(30)
    expect(footLabelStyle(MIX)).toEqual({ interval: 0, margin: 6, lineHeight: 12 })
  })
})

describe('axisFootProblems', () => {
  const axis = (formatter: (v: string, i: number) => string, margin = 4, lineHeight = 14) => ({
    data: ['0', '1'],
    axisLabel: { formatter, margin, lineHeight },
  })

  it('attrape les nombres écrits à la main avant ce calcul : la bande recouvrait les dates', () => {
    // Pied d'avant : grid.bottom 64, bande 10 px à 46 px du bas, dates d'une ligne (4 + 14 = 18 px).
    const avant = {
      grid: [{ bottom: 64 }, { bottom: 46, height: 10 }],
      xAxis: [axis(() => '{d|12/09}'), axis(() => '{n|1 sur 7}\n{m|B}', 6, 13)],
    }
    const problems = axisFootProblems(avant)
    expect(problems).toHaveLength(1)
    expect(problems[0]).toContain('recouvre')
  })

  it('accepte un pied calculé par layoutAxisFoot', () => {
    const l = layoutAxisFoot({ above: DATES, band: { height: 10, gap: 4 }, below: MIX, pad: 4 })
    const option = {
      grid: [{ bottom: l.gridBottom }, { bottom: l.bandBottom, height: l.bandHeight }],
      xAxis: [axis(() => 'a'), axis(() => 'a\nb', 6, 12)],
    }
    expect(axisFootProblems(option)).toEqual([])
  })

  it('signale des étiquettes plus hautes que le bas de grille et une hauteur de ligne non écrite', () => {
    expect(axisFootProblems({ grid: [{ bottom: 10 }], xAxis: [axis(() => 'a\nb')] })).toHaveLength(1)
    const sansLigne = { grid: [{ bottom: 40 }], xAxis: [{ axisLabel: { formatter: () => 'a', margin: 4 } }] }
    expect(axisFootProblems(sansLigne)[0]).toContain('lineHeight')
  })
})
