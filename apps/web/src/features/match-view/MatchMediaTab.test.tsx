import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MatchMediaTab } from './MatchMediaTab'
import { MATCH_VIEW_TEXT } from './i18n'

describe('MatchMediaTab - etat vide', () => {
  it.each(['fr', 'en'] as const)('liste vide (%s) : message d etat vide, pas de grille', (locale) => {
    const t = MATCH_VIEW_TEXT[locale]
    renderWithProviders(<MatchMediaTab items={[]} playerSlug="p" matchId="m" t={t} />)
    expect(screen.getByText(t.mediaNoCaptures)).toBeInTheDocument()
    expect(screen.getByText(t.mediaNoCapturesDesc)).toBeInTheDocument()
  })
})
