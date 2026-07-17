/**
 * Unit tests pour les helpers purs de session-detail.
 *
 * Couvre les fonctions sans dépendance React/store (formatters + outcome mapping).
 * `useSessionT` n'est pas testé ici (hook React qui dépend de `appShellStore` —
 * couvert indirectement par `SessionDetailPage.test.tsx`).
 */
import { describe, expect, it } from 'vitest'

import { formatPercent, matchOutcomeTone, parseDelta } from './_shared'

// Le mapping outcome int → clé ('win'/'loss'/'tie'/'dnf', défaut null) a migré
// vers `@/lib/outcome` (`outcomeCodeToValue`) — testé dans `lib/outcome.test.ts`.

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
