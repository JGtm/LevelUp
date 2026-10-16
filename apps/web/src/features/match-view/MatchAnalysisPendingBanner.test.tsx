/**
 * Tests — bandeau INFO « Analyse du film en cours » de la vue match.
 */
import { describe, expect, it } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import { MATCH_VIEW_TEXT } from './i18n'
import { MatchAnalysisPendingBanner } from './MatchAnalysisPendingBanner'

describe('MatchAnalysisPendingBanner', () => {
  it('FR : titre, texte exact, « Mise à jour automatique » — role status, teinte info', () => {
    render(<MatchAnalysisPendingBanner t={MATCH_VIEW_TEXT.fr} />)
    const status = screen.getByRole('status')
    expect(within(status).getByText('Analyse du film en cours')).toBeInTheDocument()
    expect(
      within(status).getByText(
        "Les frags assistés, l'arme de chaque frag, les citations et le rejeu 2D apparaîtront ici d'eux-mêmes d'ici quelques instants.",
      ),
    ).toBeInTheDocument()
    expect(within(status).getByText('Mise à jour automatique')).toBeInTheDocument()
    // Teinte INFO, pas avertissement.
    expect(status.className).toContain('border-info')
    expect(status.className).not.toMatch(/warning|destructive/)
  })

  it('EN', () => {
    render(<MatchAnalysisPendingBanner t={MATCH_VIEW_TEXT.en} />)
    expect(screen.getByText('Film analysis in progress')).toBeInTheDocument()
    expect(screen.getByText('Automatic update')).toBeInTheDocument()
  })

  it('le point pulse seulement hors prefers-reduced-motion', () => {
    render(<MatchAnalysisPendingBanner t={MATCH_VIEW_TEXT.fr} />)
    const animated = document.querySelectorAll('[class*="animate-"]')
    expect(animated.length).toBeGreaterThan(0)
    for (const el of Array.from(animated)) {
      for (const cls of el.className.split(/\s+/).filter((c) => c.includes('animate-'))) {
        expect(cls.startsWith('motion-safe:')).toBe(true)
      }
    }
  })

  it('FR : ni « en général » ni « quelques minutes »', () => {
    const fr = [MATCH_VIEW_TEXT.fr.analysisPendingTitle, MATCH_VIEW_TEXT.fr.analysisPendingBody].join(' ')
    expect(fr).not.toMatch(/en général|quelques minutes/i)
  })
})
