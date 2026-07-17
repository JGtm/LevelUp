/**
 * Tests table-driven sur les helpers formatters (P2.6bis).
 * Note : `formatPercent` a son propre fichier de tests (percent.test.ts).
 */
import { describe, it, expect } from 'vitest'

import {
  formatDate,
  formatDateRange,
  formatDateShort,
  formatDateTime,
  formatNumber,
  formatNumberFixed,
  formatRatio,
  formatKDA,
  formatDurationMMSS,
  formatDurationHMS,
  formatDurationMShort,
  formatDurationHM,
  displayRatingLabel,
  formatRankDelta,
  formatOffensiveConversion,
  formatDefensiveResistance,
  effectiveDmgPerFrag,
} from './index'

describe('formatDate', () => {
  it('format ISO en medium FR', () => {
    const result = formatDate('2026-04-29T12:00:00Z', 'fr-FR')
    // Le résultat exact dépend du fuseau ; on vérifie qu'il contient avril
    expect(result).toMatch(/avr|avril/i)
  })

  it('format ISO en short EN', () => {
    const result = formatDate('2026-04-29T12:00:00Z', 'en-US', { dateStyle: 'short' })
    expect(result).toMatch(/4\/29\/26|04\/29\/2026/)
  })

  it('renvoie le fallback sur null/undefined/empty/invalide', () => {
    expect(formatDate(null, 'fr-FR')).toBe('—')
    expect(formatDate(undefined, 'fr-FR')).toBe('—')
    expect(formatDate('', 'fr-FR')).toBe('—')
    expect(formatDate('not-a-date', 'fr-FR')).toBe('—')
  })

  it('respecte un fallback custom', () => {
    expect(formatDate(null, 'fr-FR', undefined, 'N/A')).toBe('N/A')
  })
})

describe('formatDateRange', () => {
  it('factorise mois/année quand la période est dans le même mois (année incluse)', () => {
    const r = formatDateRange('2025-03-03', '2025-03-12', 'fr-FR')
    expect(r).toMatch(/mars/i)
    expect(r).toMatch(/2025/)
    expect(r).toMatch(/3/)
    expect(r).toMatch(/12/)
    // Année factorisée : une seule occurrence de "2025".
    expect(r.match(/2025/g)?.length).toBe(1)
  })

  it('affiche les deux années quand elles diffèrent', () => {
    const r = formatDateRange('2024-03-03', '2025-01-12', 'fr-FR')
    expect(r).toMatch(/2024/)
    expect(r).toMatch(/2025/)
  })

  it('date simple si end absent, égal à start, ou invalide', () => {
    const single = formatDateRange('2025-03-03', null, 'fr-FR')
    expect(single).toMatch(/mars/i)
    expect(single).toMatch(/2025/)
    expect(formatDateRange('2025-03-03', '2025-03-03', 'fr-FR')).toBe(single)
    expect(formatDateRange('2025-03-03', 'not-a-date', 'fr-FR')).toBe(single)
  })

  it('renvoie le fallback sur start invalide', () => {
    expect(formatDateRange(null, '2025-03-12', 'fr-FR')).toBe('—')
    expect(formatDateRange('', null, 'fr-FR')).toBe('—')
    expect(formatDateRange('not-a-date', null, 'fr-FR')).toBe('—')
    expect(formatDateRange(null, null, 'fr-FR', 'N/A')).toBe('N/A')
  })
})

describe('formatDateShort', () => {
  it('format DD/MM FR', () => {
    expect(formatDateShort('2026-04-29')).toMatch(/29\/04/)
  })
})

describe('formatDateTime', () => {
  it('format date+time selon la locale', () => {
    const result = formatDateTime('2026-04-29T12:00:00Z', 'fr-FR')
    // Doit contenir une date et une heure
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}/)
  })

  it('fallback sur null', () => {
    expect(formatDateTime(null, 'fr-FR')).toBe('—')
  })
})

describe('formatNumber', () => {
  it('format avec séparateurs FR', () => {
    expect(formatNumber(12345, 'fr-FR', 0)).toMatch(/12.345/)
    expect(formatNumber(12345.6, 'fr-FR', 1)).toMatch(/12.345,6/)
  })

  it('format avec séparateurs EN', () => {
    expect(formatNumber(12345, 'en-US', 0)).toBe('12,345')
  })

  it('fallback sur null/NaN', () => {
    expect(formatNumber(null, 'fr-FR')).toBe('—')
    expect(formatNumber(NaN, 'fr-FR')).toBe('—')
  })
})

describe('formatNumberFixed', () => {
  it('toFixed sans séparateurs locale', () => {
    expect(formatNumberFixed(12.345, 1)).toBe('12.3')
    expect(formatNumberFixed(12.345, 2)).toBe('12.35')
  })

  it('fallback sur null/NaN/Infinity', () => {
    expect(formatNumberFixed(null)).toBe('—')
    expect(formatNumberFixed(NaN)).toBe('—')
    expect(formatNumberFixed(Infinity)).toBe('—')
  })
})

describe('formatRatio / formatKDA', () => {
  it('2 décimales locale-sensitive', () => {
    expect(formatRatio(2.345, 'fr-FR')).toMatch(/2,35/)
    expect(formatRatio(2.345, 'en-US')).toBe('2.35')
  })

  it('formatKDA est un alias de formatRatio', () => {
    expect(formatKDA(2.345, 'fr-FR')).toBe(formatRatio(2.345, 'fr-FR'))
  })

  it('fallback sur null', () => {
    expect(formatRatio(null, 'fr-FR')).toBe('—')
    expect(formatKDA(null, 'fr-FR')).toBe('—')
  })
})

describe('formatDurationMMSS', () => {
  it('format MM:SS standard', () => {
    expect(formatDurationMMSS(125)).toBe('2:05')
    expect(formatDurationMMSS(3661)).toBe('61:01') // pas d'heure (M >= 60 ok)
    expect(formatDurationMMSS(0)).toBe('-')
  })

  it('fallback sur invalide', () => {
    expect(formatDurationMMSS(undefined)).toBe('-')
    expect(formatDurationMMSS(null)).toBe('-')
    expect(formatDurationMMSS(-5)).toBe('-')
    expect(formatDurationMMSS(NaN)).toBe('-')
  })

  it('respecte un fallback custom', () => {
    expect(formatDurationMMSS(null, '—')).toBe('—')
  })
})

describe('formatDurationHMS', () => {
  it('format HH:MM:SS standard', () => {
    expect(formatDurationHMS(3661)).toBe('1:01:01')
    expect(formatDurationHMS(125)).toBe('0:02:05')
  })

  it('fallback sur invalide', () => {
    expect(formatDurationHMS(undefined)).toBe('-')
    expect(formatDurationHMS(0)).toBe('-')
  })
})

describe('formatDurationMShort', () => {
  it('format XmYYs avec secondes sur 2 chiffres', () => {
    expect(formatDurationMShort(65)).toBe('1m05s')
    expect(formatDurationMShort(37)).toBe('0m37s')
    expect(formatDurationMShort(125)).toBe('2m05s')
  })

  it('fallback sur invalide', () => {
    expect(formatDurationMShort(undefined)).toBe('-')
    expect(formatDurationMShort(null)).toBe('-')
    expect(formatDurationMShort(0)).toBe('-')
    expect(formatDurationMShort(-5)).toBe('-')
    expect(formatDurationMShort(NaN)).toBe('-')
  })
})

describe('formatDurationHM', () => {
  it('format « h min » pour des totaux longs', () => {
    expect(formatDurationHM(2530)).toBe('42 min') // < 1 h → minutes seules
    expect(formatDurationHM(152400)).toBe('42 h 20') // heures + minutes zero-paddées
    expect(formatDurationHM(3600)).toBe('1 h 00')
    expect(formatDurationHM(59)).toBe('0 min') // < 1 min mais > 0 → 0 min
  })

  it('fallback « — » sur invalide', () => {
    expect(formatDurationHM(undefined)).toBe('—')
    expect(formatDurationHM(null)).toBe('—')
    expect(formatDurationHM(0)).toBe('—')
    expect(formatDurationHM(-5)).toBe('—')
    expect(formatDurationHM(NaN)).toBe('—')
  })
})

describe('displayRatingLabel', () => {
  it('collapse toute la famille LUSR vers "LUSR" (v2 transparente)', () => {
    expect(displayRatingLabel('LUSR')).toBe('LUSR')
    expect(displayRatingLabel('LUSR_V2')).toBe('LUSR')
    expect(displayRatingLabel('lusr_v2')).toBe('LUSR')
    expect(displayRatingLabel('lusr')).toBe('LUSR')
  })

  it('conserve CSR (classé) tel quel', () => {
    expect(displayRatingLabel('CSR')).toBe('CSR')
    expect(displayRatingLabel('csr')).toBe('CSR')
  })

  it('renvoie null sur null/undefined/empty', () => {
    expect(displayRatingLabel(null)).toBeNull()
    expect(displayRatingLabel(undefined)).toBeNull()
    expect(displayRatingLabel('')).toBeNull()
  })
})

describe('formatOffensiveConversion', () => {
  it('formate OC en pourcentage entier (valeur × 100)', () => {
    expect(formatOffensiveConversion(0.42)).toBe('42%')
    expect(formatOffensiveConversion(0.835)).toBe('84%')
    expect(formatOffensiveConversion(0)).toBe('0%')
  })

  it('fallback sur null/undefined/NaN', () => {
    expect(formatOffensiveConversion(null)).toBe('—')
    expect(formatOffensiveConversion(undefined)).toBe('—')
    expect(formatOffensiveConversion(NaN)).toBe('—')
    expect(formatOffensiveConversion(null, 'N/A')).toBe('N/A')
  })
})

describe('formatDefensiveResistance', () => {
  it('formate DR en écart au baseline 1.0 ((valeur − 1) × 100)', () => {
    expect(formatDefensiveResistance(1.18)).toBe('+18%')
    expect(formatDefensiveResistance(1)).toBe('+0%')
    expect(formatDefensiveResistance(0.8)).toBe('-20%')
  })

  it('affiche ∞ pour la sentinelle (valeur < 0, aucune mort)', () => {
    expect(formatDefensiveResistance(-1)).toBe('∞')
  })

  it('fallback sur null/undefined/NaN', () => {
    expect(formatDefensiveResistance(null)).toBe('—')
    expect(formatDefensiveResistance(undefined)).toBe('—')
    expect(formatDefensiveResistance(NaN)).toBe('—')
  })
})

describe('effectiveDmgPerFrag', () => {
  it('compte les assists au dénominateur (frags + assists/3)', () => {
    // 2000 dégâts / (10 + 6/3) = 2000/12 ≈ 166.67
    expect(effectiveDmgPerFrag(2000, 10, 6)).toBeCloseTo(2000 / 12, 6)
  })

  it("est l'inverse exact du rendement : OC = 225 / effectiveDmgPerFrag", () => {
    const dpfe = effectiveDmgPerFrag(2000, 10, 6) as number
    // OC officiel = 225 × (10 + 6/3) / 2000 = 1.35
    expect(225 / dpfe).toBeCloseTo((225 * 12) / 2000, 6)
  })

  it('renvoie null si dégâts absents ou dénominateur ≤ 0', () => {
    expect(effectiveDmgPerFrag(null, 10, 6)).toBeNull()
    expect(effectiveDmgPerFrag(undefined, 10, 6)).toBeNull()
    expect(effectiveDmgPerFrag(2000, 0, 0)).toBeNull()
  })
})

describe('formatRankDelta', () => {
  it('CSR : entier signé, ±0 sur zéro', () => {
    expect(formatRankDelta(45, 'CSR')).toBe('+45')
    expect(formatRankDelta(-12, 'csr')).toBe('-12')
    expect(formatRankDelta(0, 'CSR')).toBe('±0')
  })

  it('LUSR : 2 décimales signées, ±0.00 sur zéro (jamais -0)', () => {
    expect(formatRankDelta(1.234, 'LUSR')).toBe('+1.23')
    expect(formatRankDelta(-2.5, 'LUSR')).toBe('-2.50')
    expect(formatRankDelta(0, 'LUSR')).toBe('±0.00')
  })

  it('LUSR_V2 (row audit) est traité comme LUSR, pas CSR', () => {
    expect(formatRankDelta(0, 'LUSR_V2')).toBe('±0.00')
    expect(formatRankDelta(3, 'LUSR_V2')).toBe('+3.00')
  })
})
