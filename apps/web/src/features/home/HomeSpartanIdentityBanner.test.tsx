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
import type { RenderResult } from '@testing-library/react'
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

function setTitle(currentTitleSlug: string, availableTitles: TitleSummary[]) {
  act(() => {
    useAppShellStore.setState({ currentTitleSlug, availableTitles })
  })
}

function renderBanner(): RenderResult {
  return renderWithProviders(
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

/** Toute src d'asset title-scopé (nameplate/emblème recolorisé) présente dans le DOM. */
function maskSrcs(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('[data-mask-src]')).map(
    (el) => el.getAttribute('data-mask-src') ?? '',
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

describe('HomeSpartanIdentityBanner — portes transitoires fail-closed (anti-fuite V72-29)', () => {
  afterEach(() => {
    setTitle('halo_infinite', [])
  })

  it('availableTitles vide (re-bootstrap) : AUCUNE synthèse même si currentTitleSlug=halo_5', () => {
    // Fenêtre transitoire au switch : le store pointe encore halo_5 mais availableTitles
    // n'est pas (encore) résolu → useCapabilityStrict = false → pas de visuel Halo 5.
    setTitle('halo_5', [])
    const { container } = renderBanner()
    expect(screen.queryByTestId('home-spartan-nameplate-scrim')).not.toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-emblem-image')).toBeInTheDocument()
    // Aucun asset title-scopé (nameplate/emblème recolorisés) rendu.
    expect(maskSrcs(container)).toHaveLength(0)
  })

  it('currentTitleSlug absent de availableTitles (désync) : fail-closed, aucun asset halo_5', () => {
    setTitle('halo_5', [title('halo_infinite', ['spartan_customizer'])])
    const { container } = renderBanner()
    expect(screen.queryByTestId('home-spartan-nameplate-scrim')).not.toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-emblem-image')).toBeInTheDocument()
    expect(maskSrcs(container).some((s) => s.includes('halo_5'))).toBe(false)
  })

  it('titre résolu SANS spartan_customizer (Infinite) : bannière image, pas de synthèse', () => {
    setTitle('halo_infinite', [title('halo_infinite', [])])
    const { container } = renderBanner()
    expect(screen.queryByTestId('home-spartan-nameplate-scrim')).not.toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-emblem-image')).toBeInTheDocument()
    expect(maskSrcs(container)).toHaveLength(0)
  })

  it('titre résolu AVEC spartan_customizer (H5) : synthèse via assets /titles/halo_5/', () => {
    setTitle('halo_5', [title('halo_5', ['spartan_customizer'])])
    const { container } = renderBanner()
    expect(screen.getByTestId('home-spartan-nameplate-scrim')).toBeInTheDocument()
    expect(screen.queryByTestId('home-spartan-emblem-image')).not.toBeInTheDocument()
    const srcs = maskSrcs(container)
    // Nameplate + emblème recolorisés, TOUS sous /titles/halo_5/ (jamais un autre titre).
    expect(srcs.length).toBeGreaterThan(0)
    expect(srcs.every((s) => s.startsWith('/titles/halo_5/'))).toBe(true)
  })

  it('switch à chaud H5 -> Infinite : aucun asset /titles/halo_5/ ne reste dans le DOM', () => {
    setTitle('halo_5', [title('halo_5', ['spartan_customizer'])])
    const { container } = renderBanner()
    // État initial H5 : synthèse active, assets halo_5 présents.
    expect(maskSrcs(container).some((s) => s.includes('/titles/halo_5/'))).toBe(true)

    // Bascule Infinite (re-render via subscription store).
    setTitle('halo_infinite', [title('halo_infinite', [])])
    expect(screen.queryByTestId('home-spartan-nameplate-scrim')).not.toBeInTheDocument()
    expect(screen.getByTestId('home-spartan-emblem-image')).toBeInTheDocument()
    expect(maskSrcs(container).some((s) => s.includes('halo_5'))).toBe(false)
  })
})
