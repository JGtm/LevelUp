/**
 * UsageA2Ajustements.test.tsx — LES ÉTATS VIDES du lot A2 (2026-09-21) sur les formes partagées.
 *
 *   - D8 — CHAQUE CAUSE D'ÉTAT VIDE A SA PHRASE ET SON TITRE : « aucune donnée » n'en distinguait
 *     aucune ; l'état vide se dessine par le gabarit canonique `EmptyStateNotice`.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { UsageEmptyNotice } from './UsageEmptyNotice'
import { USAGE_TEXT } from './usageI18n'
import type { UsageEmptyReason } from './usageAvailability'

const t = USAGE_TEXT.fr

describe('UsageEmptyNotice — D8, une phrase par cause', () => {
  const attendu: Record<UsageEmptyReason, string> = {
    'no-film': t.emptyNoFilm,
    'no-pads': t.emptyNoPads,
    'no-objectives': t.emptyNoObjectives,
    'load-failed': t.unavailableLoadFailed,
  }

  it('nomme la cause, et les quatre phrases sont distinctes', () => {
    for (const [reason, phrase] of Object.entries(attendu)) {
      const { unmount } = render(
        <UsageEmptyNotice reason={reason as UsageEmptyReason} t={t} />,
      )
      expect(screen.getByText(phrase)).toBeInTheDocument()
      unmount()
    }
    expect(new Set(Object.values(attendu)).size).toBe(4)
  })

  it('porte sa cause en attribut, pour que la capture dise laquelle', () => {
    const { container } = render(<UsageEmptyNotice reason="no-pads" t={t} />)
    expect(container.querySelector('[data-usage-empty="no-pads"]')).not.toBeNull()
  })

  // 2026-09-22 : l'état vide se dessine comme partout ailleurs dans l'app — le gabarit
  // `EmptyStateNotice` (titre en gras + description en gris dans le cadre pointillé), pas
  // une ligne grise nue. La typographie elle-même est testée chez `empty-state`.
  it('rend le gabarit canonique : un titre court AU-DESSUS de la phrase de la cause', () => {
    const { container } = render(<UsageEmptyNotice reason="no-pads" t={t} />)
    expect(screen.getByText(t.emptyTitleNoPads)).toBeInTheDocument()
    expect(container.querySelector('.border-dashed')).not.toBeNull()
  })

  it('a un titre distinct par cause (deux causes, deux phrases — jusque dans le titre)', () => {
    const titres = [
      t.emptyTitleNoFilm,
      t.emptyTitleNoPads,
      t.emptyTitleNoObjectives,
      t.emptyTitleLoadFailed,
    ]
    expect(new Set(titres).size).toBe(4)
  })
})
