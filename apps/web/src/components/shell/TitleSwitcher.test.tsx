import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup, waitFor } from '@testing-library/react'

// Navigate-first (Phase 4a) : le TitleSwitcher NAVIGUE (le layout `t/$titleSlug`
// bascule). On mocke donc useNavigate + applyActiveTitle (fallback sans-joueur) et
// on ré-cible les anciennes assertions switchTitle sur navigate.
const navigateMock = vi.fn()
const applyActiveTitleMock = vi.fn<(slug: string) => Promise<void>>(() => Promise.resolve())

vi.mock('@tanstack/react-router', () => ({ useNavigate: () => navigateMock }))
vi.mock('@/lib/title-routing/applyActiveTitle', () => ({
  applyActiveTitle: (slug: string) => applyActiveTitleMock(slug),
}))

import { TitleSwitcher } from './TitleSwitcher'
import { useAppShellStore } from '@/stores/appShellStore'

const HALO = {
  slug: 'halo_infinite',
  name: 'Halo Infinite',
  status: 'active' as const,
  capabilities: [],
  is_default: true,
  effective_hp_to_kill: 225,
}
const SOON = {
  slug: 'halo_mcc',
  name: 'Halo MCC',
  status: 'coming_soon' as const,
  capabilities: [],
  is_default: false,
  effective_hp_to_kill: 225,
}
const OTHER = {
  slug: 'halo_3',
  name: 'Halo 3',
  status: 'active' as const,
  capabilities: [],
  is_default: false,
  effective_hp_to_kill: 225,
}
const PLAYER = {
  player_slug: 'p1', gamertag: 'P1', xuid: '1', waypoint_player: 'P1', is_demo: false, sync_enabled: true,
}

describe('TitleSwitcher (PMT-8 / MT-22)', () => {
  beforeEach(() => {
    navigateMock.mockClear()
    applyActiveTitleMock.mockClear()
    applyActiveTitleMock.mockImplementation(() => Promise.resolve())
    useAppShellStore.setState({ locale: 'fr', isTitleSwitching: false, currentPlayer: PLAYER, availablePlayers: [PLAYER] })
  })
  afterEach(() => cleanup())

  it('NO-OP mono-titre : ne rend rien avec moins de 2 titres', () => {
    useAppShellStore.setState({ availableTitles: [HALO], currentTitleSlug: 'halo_infinite' })
    const { container } = render(<TitleSwitcher />)
    expect(container).toBeEmptyDOMElement()
  })

  it('liste les titres ; coming_soon désactivé + « Bientôt disponible »', () => {
    useAppShellStore.setState({ availableTitles: [HALO, SOON], currentTitleSlug: 'halo_infinite' })
    render(<TitleSwitcher />)
    expect(screen.getByText('Halo Infinite')).toBeInTheDocument()
    expect(screen.getByText('Halo MCC')).toBeInTheDocument()
    expect(screen.getByText('Bientôt disponible')).toBeInTheDocument()
    expect(screen.getByRole('menuitemradio', { name: /Halo MCC/ })).toBeDisabled()
  })

  it('clic sur coming_soon ne navigue PAS', () => {
    useAppShellStore.setState({ availableTitles: [HALO, SOON], currentTitleSlug: 'halo_infinite' })
    render(<TitleSwitcher />)
    fireEvent.click(screen.getByRole('menuitemradio', { name: /Halo MCC/ }))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it('clic sur un titre actif non courant navigue vers le segment du titre cible', () => {
    useAppShellStore.setState({ availableTitles: [HALO, OTHER], currentTitleSlug: 'halo_infinite' })
    render(<TitleSwitcher />)
    fireEvent.click(screen.getByRole('menuitemradio', { name: 'Halo 3' }))
    // Navigate-first : segment du titre CIBLE + playerSlug du titre COURANT (le
    // layout bascule, resolvePlayerFallback re-cible si besoin).
    expect(navigateMock).toHaveBeenCalledWith({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
      params: { titleSlug: 'halo_3', playerSlug: 'p1' },
    })
  })

  it('clic sur le titre courant ne navigue pas', () => {
    useAppShellStore.setState({ availableTitles: [HALO, OTHER], currentTitleSlug: 'halo_infinite' })
    render(<TitleSwitcher />)
    fireEvent.click(screen.getByRole('menuitemradio', { name: 'Halo Infinite' }))
    expect(navigateMock).not.toHaveBeenCalled()
  })

  it('aucun joueur → fallback applyActiveTitle direct puis navigate vers l’index', async () => {
    // availablePlayers vide + aucun joueur courant → pas de route `/t/{slug}` nue :
    // le fallback bascule directement puis renvoie à l'index (onboarding).
    useAppShellStore.setState({
      availableTitles: [HALO, OTHER],
      currentTitleSlug: 'halo_infinite',
      currentPlayer: null,
      availablePlayers: [],
    })
    render(<TitleSwitcher />)
    fireEvent.click(screen.getByRole('menuitemradio', { name: 'Halo 3' }))
    expect(applyActiveTitleMock).toHaveBeenCalledWith('halo_3')
    await waitFor(() => expect(navigateMock).toHaveBeenCalledWith({ to: '/' }))
  })

  // Revue UX H5 : « on met pas assez en valeur le titre actif ». Le titre courant
  // porte l'emphase token primary (bg + texte) en plus de aria-checked ; les autres
  // non.
  it('le titre actif est mis en valeur (aria-checked + emphase token primary)', () => {
    useAppShellStore.setState({ availableTitles: [HALO, OTHER], currentTitleSlug: 'halo_infinite' })
    render(<TitleSwitcher />)

    const current = screen.getByRole('menuitemradio', { name: 'Halo Infinite' })
    expect(current).toHaveAttribute('aria-checked', 'true')
    expect(current.className).toContain('text-primary')
    expect(current.className).toContain('bg-primary/10')

    const other = screen.getByRole('menuitemradio', { name: 'Halo 3' })
    expect(other).toHaveAttribute('aria-checked', 'false')
    expect(other.className).not.toContain('text-primary')
  })
})
