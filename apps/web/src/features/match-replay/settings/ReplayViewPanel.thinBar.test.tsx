/**
 * ReplayViewPanel.thinBar.test.tsx — LE PANNEAU DE LA VUE : l'option « Barre de lecture fine », là où vivent les
 * options voisines (« Fil des éliminations »). Elle n'est proposée que si le cadre peut poser la barre sur le plan.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { REPLAY_TEXT } from '../i18n/i18n'
import { renderPanel } from './replayViewPanelFixtures'

const NAME = REPLAY_TEXT.fr.frames.thinBar

describe('ReplayViewPanel — barre de lecture fine', () => {
  it("absente quand le panneau n'en reçoit pas", () => {
    renderPanel()
    expect(screen.queryByRole('switch', { name: NAME })).toBeNull()
  })

  it("absente quand le cadre ne peut pas poser la barre sur le plan", () => {
    renderPanel({ thinBar: { available: false, show: true, onToggle: vi.fn() } })
    expect(screen.queryByRole('switch', { name: NAME })).toBeNull()
  })

  it("présente au Plein cadre, avec l'état retenu, juste après le fil des éliminations", () => {
    const onToggle = vi.fn()
    renderPanel({
      planFeed: { available: true, show: true, onToggle: vi.fn() },
      thinBar: { available: true, show: true, onToggle },
    })
    const feed = screen.getByRole('switch', { name: REPLAY_TEXT.fr.frames.planFeed })
    const thin = screen.getByRole('switch', { name: NAME })
    expect(thin).toHaveAttribute('aria-checked', 'true')
    expect(feed.compareDocumentPosition(thin) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    fireEvent.click(thin)
    expect(onToggle).toHaveBeenCalledTimes(1)
  })
})
