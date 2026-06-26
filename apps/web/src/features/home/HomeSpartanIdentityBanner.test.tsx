/**
 * Tests — HomeSpartanIdentityBanner : bannière SYNTHÉTISÉE pour Halo 5.
 *
 * Halo 5 n'a pas de nameplate ; ce qui arrive en banner_image_url est le render
 * full-body (pas une bannière). Pour H5 on NE l'utilise PAS en fond → backdrop
 * gradient synthétisé. Hors H5, la bannière image reste utilisée en fond.
 */
import { describe, it, expect, afterEach } from 'vitest'
import { act, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeSpartanIdentity } from '@/lib/api/types'
import { HomeSpartanIdentityBanner } from './HomeSpartanIdentityBanner'

const identity: HomeSpartanIdentity = {
  spartan_id: 'OKLM',
  banner_image_url: 'https://cdn.test/render.png',
  emblem_image_url: 'https://cdn.test/emblem.png',
}

function renderBanner() {
  renderWithProviders(
    <HomeSpartanIdentityBanner
      spartanIdentity={identity}
      playerName="JGtm"
      highestCSR={null}
      highestLUSR={null}
      hasRankedHistory={false}
      hasUnrankedHistory={false}
      hasPrivacyWarning={false}
      identityUnavailableLabel="—"
    />,
  )
}

describe('HomeSpartanIdentityBanner — bannière synthétisée H5', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ currentTitleSlug: '' })
    })
  })

  it('Halo 5 : backdrop synthétisé (le render full-body n\'est PAS le fond)', () => {
    act(() => {
      useAppShellStore.setState({ currentTitleSlug: 'halo_5' })
    })
    renderBanner()
    expect(screen.getByTestId('home-spartan-synthesized-backdrop')).toBeInTheDocument()
    // L'emblème et l'identité restent rendus (bannière synthétisée, pas vide).
    expect(screen.getByTestId('home-spartan-emblem-image')).toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-gamertag')).toHaveTextContent('JGtm')
  })

  it('hors Halo 5 : pas de backdrop synthétisé quand une bannière image existe', () => {
    act(() => {
      useAppShellStore.setState({ currentTitleSlug: 'halo_infinite' })
    })
    renderBanner()
    expect(screen.queryByTestId('home-spartan-synthesized-backdrop')).not.toBeInTheDocument()
  })
})
