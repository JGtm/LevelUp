import { describe, expect, it } from 'vitest'

import { adminRelativeTime, formatDurationMs, formatIntervalMinutes } from './format'

describe('adminRelativeTime', () => {
  const now = new Date('2026-06-11T12:00:00Z').getTime()

  it('retourne un tiret pour les ISO vides ou invalides', () => {
    expect(adminRelativeTime(undefined, 'fr', now)).toBe('—')
    expect(adminRelativeTime('', 'fr', now)).toBe('—')
    expect(adminRelativeTime('pas-une-date', 'en', now)).toBe('—')
  })

  it('couvre les bornes minute/heure/jour en FR', () => {
    expect(adminRelativeTime('2026-06-11T11:59:50Z', 'fr', now)).toBe("à l'instant")
    expect(adminRelativeTime('2026-06-11T11:45:00Z', 'fr', now)).toBe('il y a 15 min')
    expect(adminRelativeTime('2026-06-11T09:00:00Z', 'fr', now)).toBe('il y a 3 h')
    expect(adminRelativeTime('2026-06-09T12:00:00Z', 'fr', now)).toBe('il y a 2 j')
  })

  it('couvre les bornes en EN', () => {
    expect(adminRelativeTime('2026-06-11T11:45:00Z', 'en', now)).toBe('15 min ago')
    expect(adminRelativeTime('2026-06-11T09:00:00Z', 'en', now)).toBe('3 h ago')
  })

  it('bascule sur la date locale au-delà de 7 jours', () => {
    const out = adminRelativeTime('2026-05-01T12:00:00Z', 'fr', now)
    expect(out).toContain('2026')
    expect(out).not.toContain('il y a')
  })
})

describe('formatDurationMs', () => {
  it('retourne un tiret pour les valeurs absentes ou négatives', () => {
    expect(formatDurationMs(undefined, 'fr')).toBe('—')
    expect(formatDurationMs(-5, 'fr')).toBe('—')
    expect(formatDurationMs(Number.NaN, 'en')).toBe('—')
  })

  it('formate ms, secondes (décimale localisée), minutes et heures', () => {
    expect(formatDurationMs(850, 'fr')).toBe('850 ms')
    expect(formatDurationMs(2400, 'fr')).toBe('2,4 s')
    expect(formatDurationMs(2400, 'en')).toBe('2.4 s')
    expect(formatDurationMs(65_000, 'fr')).toBe('1 min 05 s')
    expect(formatDurationMs(120_000, 'fr')).toBe('2 min')
    expect(formatDurationMs(4_320_000, 'fr')).toBe('1 h 12 min')
    expect(formatDurationMs(7_200_000, 'en')).toBe('2 h')
  })
})

describe('formatIntervalMinutes', () => {
  it('gère minutes, heures rondes et mixte', () => {
    expect(formatIntervalMinutes(undefined, 'fr')).toBe('—')
    expect(formatIntervalMinutes(0, 'fr')).toBe('—')
    expect(formatIntervalMinutes(15, 'fr')).toBe('15 min')
    expect(formatIntervalMinutes(360, 'fr')).toBe('6 h')
    expect(formatIntervalMinutes(90, 'en')).toBe('1 h 30 min')
  })
})
