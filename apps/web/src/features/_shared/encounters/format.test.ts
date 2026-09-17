/**
 * Tests — features/_shared/encounters/format.
 *
 * Ces quatre formats vivaient EN DOUBLE (vue match + explorateur) et n'étaient couverts nulle
 * part : la centralisation les met sous test au passage. Chaque cas verrouille une DÉCISION,
 * pas un chiffre : ce qui est une absence (tiret), ce qui est une mesure (le zéro), où
 * basculent les paliers d'ancienneté, et le singulier français de chacun.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { formatKDCross, formatKDRatio, formatRelativeEN, formatRelativeFor, formatRelativeFR } from './format'

describe('formatKDCross — frags/morts en croisé', () => {
  it('rend le tiret QUAND LES DEUX côtés sont absents, jamais sur un seul', () => {
    expect(formatKDCross(null, null)).toBe('—')
    expect(formatKDCross(undefined, undefined)).toBe('—')
    expect(formatKDCross(3, null)).toBe('3/0')
    expect(formatKDCross(null, 4)).toBe('0/4')
  })
  it('le zéro est une MESURE et s’affiche', () => {
    expect(formatKDCross(0, 0)).toBe('0/0')
    expect(formatKDCross(7, 2)).toBe('7/2')
  })
})

describe('formatKDRatio — frags ÷ morts', () => {
  it('rend le tiret dès qu’un côté est absent (là où le croisé, lui, affiche 0)', () => {
    expect(formatKDRatio(null, 2)).toBe('—')
    expect(formatKDRatio(3, null)).toBe('—')
    expect(formatKDRatio(undefined, undefined)).toBe('—')
  })
  it('zéro mort : ∞ avec des frags, tiret sans — 0/0 n’est pas un ratio infini', () => {
    expect(formatKDRatio(5, 0)).toBe('∞')
    expect(formatKDRatio(0, 0)).toBe('—')
  })
  it('deux décimales, toujours', () => {
    expect(formatKDRatio(3, 2)).toBe('1.50')
    expect(formatKDRatio(1, 3)).toBe('0.33')
  })
})

// Le temps est FIGÉ : ces formats lisent `Date.now()`, un test qui dépendrait de l'horloge
// réelle serait instable aux bornes (59 min 59 s -> « il y a 1 h »).
const MAINTENANT = new Date('2026-09-17T12:00:00.000Z')
/** Un ISO daté de `ms` millisecondes AVANT l'instant figé. */
function ilYA(ms: number): string {
  return new Date(MAINTENANT.getTime() - ms).toISOString()
}
const MIN = 60_000
const HEURE = 60 * MIN
const JOUR = 24 * HEURE

beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(MAINTENANT)
})
afterEach(() => {
  vi.useRealTimers()
})

describe('formatRelativeFR — ancienneté en français', () => {
  it('une date illisible rend le tiret, jamais « Invalid Date »', () => {
    expect(formatRelativeFR('pas-une-date')).toBe('—')
    expect(formatRelativeFR('')).toBe('—')
  })
  it('minutes, puis heures', () => {
    expect(formatRelativeFR(ilYA(10_000))).toBe("à l'instant")
    expect(formatRelativeFR(ilYA(5 * MIN))).toBe('il y a 5 min')
    expect(formatRelativeFR(ilYA(HEURE))).toBe('il y a 1 h')
    expect(formatRelativeFR(ilYA(5 * HEURE))).toBe('il y a 5 h')
  })
  it('jours, semaines, mois, années — avec leurs singuliers', () => {
    expect(formatRelativeFR(ilYA(JOUR))).toBe('hier')
    expect(formatRelativeFR(ilYA(3 * JOUR))).toBe('il y a 3 j')
    expect(formatRelativeFR(ilYA(7 * JOUR))).toBe('il y a 1 sem.')
    expect(formatRelativeFR(ilYA(21 * JOUR))).toBe('il y a 3 sem.')
    // LA BASCULE SEMAINES -> MOIS EST À 5 SEMAINES ARRONDIES, pas à 30 jours : un mois
    // calendaire se dit encore « 4 sem. ». Comportement d'origine, pinné tel quel.
    expect(formatRelativeFR(ilYA(30 * JOUR))).toBe('il y a 4 sem.')
    expect(formatRelativeFR(ilYA(35 * JOUR))).toBe('il y a 1 mois')
    expect(formatRelativeFR(ilYA(90 * JOUR))).toBe('il y a 3 mois')
    expect(formatRelativeFR(ilYA(365 * JOUR))).toBe('il y a 1 an')
    expect(formatRelativeFR(ilYA(3 * 365 * JOUR))).toBe('il y a 3 ans')
  })
})

describe('formatRelativeEN — mêmes paliers, mots anglais', () => {
  it('une date illisible rend le MÊME tiret', () => {
    expect(formatRelativeEN('pas-une-date')).toBe('—')
  })
  it('des minutes aux années', () => {
    expect(formatRelativeEN(ilYA(10_000))).toBe('just now')
    expect(formatRelativeEN(ilYA(5 * MIN))).toBe('5 min ago')
    expect(formatRelativeEN(ilYA(5 * HEURE))).toBe('5 h ago')
    expect(formatRelativeEN(ilYA(JOUR))).toBe('yesterday')
    expect(formatRelativeEN(ilYA(3 * JOUR))).toBe('3 d ago')
    expect(formatRelativeEN(ilYA(21 * JOUR))).toBe('3 w ago')
    expect(formatRelativeEN(ilYA(30 * JOUR))).toBe('4 w ago')
    expect(formatRelativeEN(ilYA(35 * JOUR))).toBe('1 mo ago')
    expect(formatRelativeEN(ilYA(90 * JOUR))).toBe('3 mo ago')
    expect(formatRelativeEN(ilYA(365 * JOUR))).toBe('1 y ago')
    expect(formatRelativeEN(ilYA(3 * 365 * JOUR))).toBe('3 y ago')
  })
})

describe('formatRelativeFor — le ternaire que les deux vues écrivaient', () => {
  it('rend le formateur de la langue demandée', () => {
    expect(formatRelativeFor('fr')).toBe(formatRelativeFR)
    expect(formatRelativeFor('en')).toBe(formatRelativeEN)
  })
})
