/**
 * Tests — HomeSpartanIdentityBanner : bannière SYNTHÉTISÉE (emblème + nameplate
 * recolorisés) pour les titres déclarant la capability `spartan_customizer`.
 *
 * Gating par CAPABILITY (pas par slug) : un titre qui déclare `spartan_customizer`
 * n'a pas de vraie bannière (Halo 5 = render full-body), on synthétise donc via le
 * nameplate recolorisé. Sinon : bannière image en fond (ou backdrop dégradé sans image).
 */
import { describe, it, expect, afterEach } from 'vitest'
import { act, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeSpartanIdentity, TitleSummary } from '@/lib/api/types'
import { HomeSpartanIdentityBanner } from './HomeSpartanIdentityBanner'

const identity: HomeSpartanIdentity = {
  spartan_id: 'OKLM',
  banner_image_url: 'https://cdn.test/render.png',
  emblem_image_url: 'https://cdn.test/emblem.png',
}

const title = (slug: string, caps: string[]): TitleSummary =>
  ({
    slug,
    name: slug,
    status: 'active',
    capabilities: caps,
    is_default: false,
    effective_hp_to_kill: 225,
  }) as unknown as TitleSummary

function renderBanner() {
  renderWithProviders(
    <HomeSpartanIdentityBanner
      spartanIdentity={identity}
      playerName="JGtm"
      playerSlug="jgtm"
      highestCSR={null}
      highestLUSR={null}
      hasRankedHistory={false}
      hasUnrankedHistory={false}
      hasPrivacyWarning={false}
      identityUnavailableLabel="—"
    />,
  )
}

describe('HomeSpartanIdentityBanner — bannière synthétisée (capability spartan_customizer)', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
    })
  })

  it('capability présente : bannière synthétisée (nameplate recolorisé, pas le render full-body)', () => {
    act(() => {
      useAppShellStore.setState({
        currentTitleSlug: 'halo_5',
        availableTitles: [title('halo_5', ['spartan_customizer'])],
      })
    })
    renderBanner()
    // Le nameplate recolorisé sert de fond (+ scrim) → pas le backdrop dégradé.
    expect(screen.getByTestId('home-spartan-nameplate-scrim')).toBeInTheDocument()
    expect(screen.queryByTestId('home-spartan-synthesized-backdrop')).not.toBeInTheDocument()
    // L'emblème est recolorisé (canvas), pas le render <img> brut.
    expect(screen.queryByTestId('home-spartan-emblem-image')).not.toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-gamertag')).toHaveTextContent('JGtm')
  })

  it('capability absente + bannière image : fond image, ni nameplate ni backdrop synthétisés', () => {
    act(() => {
      useAppShellStore.setState({
        currentTitleSlug: 'halo_infinite',
        availableTitles: [title('halo_infinite', [])],
      })
    })
    renderBanner()
    expect(screen.queryByTestId('home-spartan-nameplate-scrim')).not.toBeInTheDocument()
    expect(screen.queryByTestId('home-spartan-synthesized-backdrop')).not.toBeInTheDocument()
    // Emblème rond classique (render <img>).
    expect(screen.getByTestId('home-spartan-emblem-image')).toBeInTheDocument()
  })
})
