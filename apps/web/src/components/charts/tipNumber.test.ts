import { afterEach, describe, expect, it } from 'vitest'

import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import { useAppShellStore } from '@/stores/appShellStore'

import { getTooltipBase } from './_utils'
import { tipNumber, tipNumberAuto } from './tipNumber'

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

describe('tipNumberAuto', () => {
  it('entier sans décimale, sinon deux décimales au plus, jamais de queue de flottant', () => {
    expect(tipNumberAuto(5)).toBe('5')
    expect(tipNumberAuto(1.3333333333333333)).toBe('1,33')
    expect(tipNumberAuto(0.1 + 0.2)).toBe('0,3')
    expect(tipNumberAuto(1234.5)).toBe(`1${NNBSP}234,5`)
    expect(tipNumberAuto(-4)).toBe('−4')
    expect(tipNumberAuto(Number.NaN)).toBe('—')
  })
  it('suit la langue de l application', () => {
    useAppShellStore.setState({ locale: 'en' })
    expect(tipNumberAuto(1234.5678)).toBe('1,234.57')
  })
})

describe('getTooltipBase : valeur par défaut', () => {
  const base = getTooltipBase(getEChartsThemeColors())

  it('un nombre brut ne s écrit jamais avec sa queue de décimales', () => {
    expect(base.valueFormatter(4 / 3)).toBe('1,33')
    expect(base.valueFormatter(7)).toBe('7')
    expect(base.valueFormatter([1.23456, 2])).toBe('1,23, 2')
  })
  it('un texte ou une valeur absente reste lisible', () => {
    expect(base.valueFormatter('-')).toBe('-')
    expect(base.valueFormatter(null)).toBe('-')
  })
  it('un valueFormatter explicite, posé après le spread, le remplace', () => {
    const opt = { ...base, valueFormatter: (v: unknown) => '<' + String(v) + '>' }
    expect(opt.valueFormatter(3)).toBe('<3>')
  })
})
