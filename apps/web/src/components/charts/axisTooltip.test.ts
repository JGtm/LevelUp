import { afterEach, describe, expect, it } from 'vitest'

import { useAppShellStore } from '@/stores/appShellStore'

import { axisTooltipFormatter, stackTopFirstOrder, tipNumber } from './axisTooltip'

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const NNBSP = ' '

describe('tipNumber', () => {
  it('suit la langue de l application (séparateurs et décimale)', () => {
    useAppShellStore.setState({ locale: 'fr' })
    expect(tipNumber(1234.5, 1)).toBe(`1${NNBSP}234,5`)
    useAppShellStore.setState({ locale: 'en' })
    expect(tipNumber(1234.5, 1)).toBe('1,234.5')
  })
  it('décimales fixes, signe typographique, zéro sans signe', () => {
    expect(tipNumber(3, 1, true)).toBe('+3,0')
    expect(tipNumber(-1.25, 1, true)).toBe('−1,3')
    expect(tipNumber(0, 1, true)).toBe('0,0')
    expect(tipNumber(-4)).toBe('−4')
  })
  it('une valeur non finie ne s écrit pas NaN', () => {
    expect(tipNumber(Number.NaN)).toBe('—')
  })
})

describe('axisTooltipFormatter', () => {
  const param = (seriesIndex: number, seriesName: string, value: unknown, extra = {}) => ({
    seriesIndex,
    seriesName,
    value,
    color: `c${seriesIndex}`,
    axisValueLabel: '#2\nStreets',
    ...extra,
  })
  const fmt = axisTooltipFormatter({ value: (r) => tipNumber(r.value, 1) })

  it('titre = repère de l axe sur une ligne ; une ligne par série qui a une valeur', () => {
    const html = fmt([param(0, 'A', 1.5), param(1, 'B', null), param(2, 'C', undefined), param(3, 'D', 2)])
    expect(html).toContain('#2 · Streets')
    expect(html).toContain('>A<')
    expect(html).toContain('>D<')
    expect(html).not.toContain('>B<')
    expect(html).not.toContain('>C<')
    expect(html).not.toContain('NaN')
  })

  it('aucune valeur : pas d infobulle', () => {
    expect(fmt([param(0, 'A', null)])).toBe('')
  })

  it('ordre de la légende par défaut, valeur décroissante sur demande', () => {
    const rows = [param(0, 'A', 1), param(1, 'B', 3), param(2, 'C', 2)]
    const names = (html: string) => [...html.matchAll(/<span>([A-C])<\/span>/g)].map((m) => m[1])
    expect(names(fmt(rows))).toEqual(['A', 'B', 'C'])
    expect(names(axisTooltipFormatter({ value: () => '', order: 'value-desc' })(rows))).toEqual(['B', 'C', 'A'])
  })

  it('un couple [x, y] lit son dernier terme', () => {
    expect(fmt([param(0, 'A', [3, 7])])).toContain('7,0')
  })

  it('échappe les noms (gamertag, carte)', () => {
    const html = fmt([param(0, '<img src=x onerror=1>', 1, { axisValueLabel: '<b>' })])
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;img')
    expect(html).toContain('&lt;b&gt;')
  })

  it('ligne de détail et pied', () => {
    const html = axisTooltipFormatter({
      value: () => 'v',
      detail: () => 'dégâts 12',
      footer: () => '<div>pied</div>',
    })([param(0, 'A', 1)])
    expect(html).toContain('dégâts 12')
    expect(html.endsWith('<div>pied</div>')).toBe(true)
  })
})

describe('stackTopFirstOrder', () => {
  const names = (html: string) => [...html.matchAll(/<span>(\w+)<\/span>/g)].map((m) => m[1])

  it('dans chaque pile : positives de la plus haute à la plus basse, puis négatives de l axe vers le bas', () => {
    const order = stackTopFirstOrder([
      { name: 'J1 frags', stack: 'J1' },
      { name: 'J1 bonus', stack: 'J1' },
      { name: 'J1 morts', stack: 'J1' },
      { name: 'J2 frags', stack: 'J2' },
      { name: 'J2 bonus', stack: 'J2' },
    ])
    const values = [3, 1, -2, 4, 2]
    const html = axisTooltipFormatter({ value: () => '', order })(
      values.map((value, i) => ({ seriesIndex: i, seriesName: `s${i}`, value, color: '' })),
    )
    expect(names(html)).toEqual(['s1', 's0', 's2', 's4', 's3'])
  })

  it('un zéro tracé en négatif (-0) reste sous l axe', () => {
    const order = stackTopFirstOrder([
      { name: 'frags', stack: 'J' },
      { name: 'morts', stack: 'J' },
    ])
    const html = axisTooltipFormatter({ value: () => '', order })([
      { seriesIndex: 1, seriesName: 'morts', value: -0, color: '' },
      { seriesIndex: 0, seriesName: 'frags', value: 0, color: '' },
    ])
    expect(names(html)).toEqual(['frags', 'morts'])
  })
})
