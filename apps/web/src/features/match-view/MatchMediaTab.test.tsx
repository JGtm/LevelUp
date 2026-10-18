import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MatchMediaTab } from './MatchMediaTab'
import { MATCH_VIEW_TEXT } from './i18n'
import { getMediaText } from '@/features/media/i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MatchAssociatedMedia } from '@/lib/api/types'

describe('MatchMediaTab - etat vide', () => {
  it.each(['fr', 'en'] as const)('liste vide (%s) : message d etat vide, pas de grille', (locale) => {
    const t = MATCH_VIEW_TEXT[locale]
    renderWithProviders(<MatchMediaTab items={[]} playerSlug="p" matchId="m" t={t} />)
    expect(screen.getByText(t.mediaNoCaptures)).toBeInTheDocument()
    expect(screen.getByText(t.mediaNoCapturesDesc)).toBeInTheDocument()
  })

  it('l état vide pointillé est DANS un bloc parent (Card), jamais nu sous le titre de section', () => {
    const t = MATCH_VIEW_TEXT.fr
    renderWithProviders(<MatchMediaTab items={[]} playerSlug="p" matchId="m" t={t} />)
    const bloc = screen.getByTestId('match-media-empty')
    const notice = bloc.querySelector('.border-dashed')
    expect(notice).not.toBeNull()
    expect(notice).not.toBe(bloc)
    expect(bloc.className).toContain('border')
    expect(bloc.contains(notice)).toBe(true)
  })
})

describe("MatchMediaTab - profil d'un ami", () => {
  it('le coeur reste actif : le like est l action du visiteur, attribuee a son profil', () => {
    useAppShellStore.setState({ writablePlayerSlugs: ['MonCompte'] })
    try {
      const t = MATCH_VIEW_TEXT.fr
      const item: MatchAssociatedMedia = {
        file_id: 'f1',
        file_name: 'clip.mp4',
        file_path: '/media/Ami/clip.mp4',
        kind: 'video',
        liked: false,
      }
      renderWithProviders(<MatchMediaTab items={[item]} playerSlug="Ami" matchId="m" t={t} />)
      const like = screen.getByRole('button', { name: getMediaText('fr').like.ariaLike })
      expect(like).toBeEnabled()
    } finally {
      useAppShellStore.setState({ writablePlayerSlugs: null })
    }
  })
})
