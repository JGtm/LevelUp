/**
 * Unit tests pour les helpers purs de session-detail.
 *
 * Couvre les fonctions sans dépendance React/store (formatters + outcome mapping).
 * `useSessionT` n'est pas testé ici (hook React qui dépend de `appShellStore` —
 * couvert indirectement par `SessionDetailPage.test.tsx`).
 */
import { describe, expect, it } from 'vitest'

import {
  formatPercent,
  formatShortDateTime,
  matchOutcomeTone,
  outcomeIntToKey,
  parseDelta,
} from './_shared'

describe('outcomeIntToKey', () => {
  it('mappe les codes canoniques outcome (ADR 0006)', () => {
    expect(outcomeIntToKey(2)).toBe('win')
    expect(outcomeIntToKey(3)).toBe('loss')
    expect(outcomeIntToKey(1)).toBe('tie')
    expect(outcomeIntToKey(4)).toBe('dnf')
  })

  it('retourne null pour valeurs hors enum ou nulles', () => {
    expect(outcomeIntToKey(null)).toBeNull()
    expect(outcomeIntToKey(0)).toBeNull()
    expect(outcomeIntToKey(99)).toBeNull()
  })
})

describe('parseDelta', () => {
  it('parse une string numérique signée', () => {
    expect(parseDelta('+1.5')).toBe(1.5)
    expect(parseDelta('-0.3')).toBe(-0.3)
    expect(parseDelta('0')).toBe(0)
  })

  it('retourne null pour null, vide ou non numérique', () => {
    expect(parseDelta(null)).toBeNull()
    expect(parseDelta('')).toBeNull()
    expect(parseDelta('n/a')).toBeNull()
  })
})

describe('formatPercent', () => {
  it("formate une valeur 0..100 avec 1 décimale + suffixe '%'", () => {
    expect(formatPercent(64.83)).toBe('64.8%')
    expect(formatPercent(0)).toBe('0.0%')
    expect(formatPercent(100)).toBe('100.0%')
  })

  it("retourne '—' pour null", () => {
    expect(formatPercent(null)).toBe('—')
  })
})

describe('formatShortDateTime', () => {
  it('formate un ISO datetime en jj/mm HH:MM (locale FR)', () => {
    // 21 avril 2026 19:45 UTC → la locale FR est utilisée par Intl avec timezone
    // implicite ; on vérifie juste que le format porte bien des séparateurs et
    // 4 paires de chiffres (sans dépendre du timezone du runner).
    const out = formatShortDateTime('2026-04-21T19:45:00Z')
    expect(out).toMatch(/^\d{2}\/\d{2}\s\d{2}:\d{2}$/)
  })

  it("retourne la valeur d'origine si le parsing échoue", () => {
    expect(formatShortDateTime('not-a-date')).toBe('not-a-date')
  })
})

describe('matchOutcomeTone', () => {
  it("retourne une couleur sémantique pour outcomes connus (style.color = CSS var)", () => {
    const win = matchOutcomeTone(2)
    expect(win.className).toBe('font-medium')
    expect(win.style?.color).toMatch(/^var\(--ac-outcome-/)

    const loss = matchOutcomeTone(3)
    expect(loss.className).toBe('font-medium')
    expect(loss.style?.color).toMatch(/^var\(--ac-outcome-/)
  })

  it("retourne text-muted-foreground et pas de style pour null ou outcome inconnu", () => {
    expect(matchOutcomeTone(null)).toEqual({ className: 'text-muted-foreground' })
    expect(matchOutcomeTone(99)).toEqual({ className: 'text-muted-foreground' })
  })
})
