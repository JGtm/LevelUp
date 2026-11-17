/**
 * StartCampaignModal — fermeture : Échap ferme la modale, un clic sur le fond aussi, un clic dans la
 * carte non ; pendant l'envoi, ni Échap ni le fond ne ferment.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { StartCampaignModal } from './StartCampaignModal'

const envoi = vi.hoisted(() => ({ isPending: false }))

vi.mock('@/features/ascension/profile/queries', () => ({
  useCampaignMutations: () => ({ start: { isPending: envoi.isPending, error: null, reset: vi.fn(), mutate: vi.fn() } }),
}))

beforeEach(() => {
  envoi.isPending = false
})

function ouvre() {
  const onOpenChange = vi.fn()
  renderWithProviders(
    <StartCampaignModal open playerSlug="p1" axis="kills" axisKind="radar" onOpenChange={onOpenChange} />,
  )
  const dialogue = screen.getByRole('dialog')
  const fond = dialogue.firstElementChild as HTMLElement
  const carte = dialogue.lastElementChild as HTMLElement
  return { onOpenChange, fond, carte }
}

describe('StartCampaignModal — fermeture', () => {
  it('Échap ferme la modale, une seule fois', () => {
    const { onOpenChange } = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onOpenChange).toHaveBeenCalledTimes(1)
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('un clic sur le fond ferme la modale', () => {
    const { onOpenChange, fond } = ouvre()
    fireEvent.mouseDown(fond)
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('un clic dans la carte ne ferme pas la modale', () => {
    const { onOpenChange, carte } = ouvre()
    fireEvent.mouseDown(carte)
    fireEvent.click(carte)
    expect(onOpenChange).not.toHaveBeenCalled()
  })

  it('envoi en cours : ni Échap ni le fond ne ferment', () => {
    envoi.isPending = true
    const { onOpenChange, fond } = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    fireEvent.mouseDown(fond)
    expect(onOpenChange).not.toHaveBeenCalled()
  })
})
