/**
 * BattlePassRewardLightbox — fermeture : Échap ferme la visionneuse, un clic sur le voile aussi, un
 * clic dans le bloc central ou sur une vignette latérale non ; les flèches gardent leur écouteur.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { BattlePassRewardLightbox, type RewardLightboxData } from './BattlePassRewardLightbox'

const REWARDS: RewardLightboxData[] = [
  { title: 'Premier', rank: 1 },
  { title: 'Deuxième', rank: 2 },
  { title: 'Troisième', rank: 3 },
]

function ouvre() {
  const onClose = vi.fn()
  renderWithProviders(<BattlePassRewardLightbox rewards={REWARDS} startIndex={1} onClose={onClose} />)
  const voile = screen.getByTestId('battle-pass-reward-lightbox')
  const bloc = screen.getByRole('heading', { name: 'Deuxième' }).closest('div.rounded-2xl') as HTMLElement
  return { onClose, voile, bloc }
}

describe('BattlePassRewardLightbox — fermeture', () => {
  it('Échap ferme la visionneuse, une seule fois', () => {
    const { onClose } = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic sur le voile ferme la visionneuse', () => {
    const { onClose, voile } = ouvre()
    fireEvent.click(voile)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('un clic dans le bloc central ne ferme pas la visionneuse', () => {
    const { onClose, bloc } = ouvre()
    fireEvent.mouseDown(bloc)
    fireEvent.click(bloc)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('une vignette latérale navigue sans fermer', () => {
    const { onClose } = ouvre()
    const [precedente] = screen.getAllByRole('button', { name: /précédent/i })
    fireEvent.mouseDown(precedente)
    fireEvent.click(precedente)
    expect(onClose).not.toHaveBeenCalled()
  })

  it('les flèches gardent leur écouteur : elles naviguent sans fermer', () => {
    const { onClose } = ouvre()
    fireEvent.keyDown(window, { key: 'ArrowRight' })
    expect(screen.getByRole('heading', { name: 'Troisième' })).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })
})
