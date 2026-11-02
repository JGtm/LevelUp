/**
 * Tests — AppShell selon la capacité `can_use_compact_chrome` (ADR 0044 : la capacité, jamais le
 * nom du profil). Capacité vraie : barre latérale, ni barre haute ni pied de page. Capacité fausse
 * (le site hébergé) : barre haute et pied de page, comme avant.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { act, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { AppShell } from './AppShell'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Outlet: () => <div data-testid="outlet" />,
    useMatches: () => [],
  }
})

vi.mock('./NavL1', () => ({ NavL1: () => <nav data-testid="nav-l1" /> }))
vi.mock('./SidebarNav', () => ({ SidebarNav: () => <aside data-testid="sidebar-nav" /> }))
vi.mock('./AppFooter', () => ({ AppFooter: () => <footer data-testid="app-footer" /> }))
vi.mock('./ReauthBanner', () => ({ ReauthBanner: () => null }))
vi.mock('./OAuthRefusalBanner', () => ({ OAuthRefusalBanner: () => null }))
vi.mock('./PoolSharingBanner', () => ({ PoolSharingBanner: () => null }))
vi.mock('./TopProgressBar', () => ({
  TopProgressBar: ({ placement }: { placement?: string }) => (
    <div data-testid="progress" data-placement={placement ?? 'defaut'} />
  ),
}))
vi.mock('@/features/notifications/toastBridge', () => ({ NotificationsToastBridge: () => null }))
vi.mock('@/features/asset-drawer', () => ({
  AssetDrawer: () => <div data-testid="asset-drawer" />,
  AssetDockedPanel: () => <aside data-testid="asset-docked" />,
}))
vi.mock('@/features/feedback-drawer', () => ({ FeedbackDrawer: () => null }))

function setCapabilities(desktop: boolean) {
  act(() => {
    useAppShellStore.setState({ capabilities: testCapabilities(desktop ? desktopCapabilities() : {}) })
  })
}

describe('AppShell — chrome selon la capacité', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ capabilities: null })
    })
  })

  it('capacité vraie : barre latérale, ni barre haute ni pied de page, contenu rendu', () => {
    setCapabilities(true)
    renderWithProviders(<AppShell />)
    expect(screen.getByTestId('sidebar-nav')).toBeInTheDocument()
    expect(screen.queryByTestId('nav-l1')).not.toBeInTheDocument()
    expect(screen.queryByTestId('app-footer')).not.toBeInTheDocument()
    expect(screen.getByTestId('outlet')).toBeInTheDocument()
    // La barre de progression se pose contre la barre latérale, pas sous une barre haute absente.
    expect(screen.getByTestId('progress')).toHaveAttribute('data-placement', 'left-60 right-0 top-0')
  })

  it('capacité vraie : le volet des référentiels est un frère du contenu dans la rangée, qui le pousse', () => {
    setCapabilities(true)
    renderWithProviders(<AppShell />)
    const docked = screen.getByTestId('asset-docked')
    const row = docked.parentElement as HTMLElement
    // Même rangée horizontale que la barre latérale et le contenu, à droite du contenu : rien ne se superpose.
    expect(row.className).toContain('flex-row')
    const kids = Array.from(row.children)
    expect(kids.indexOf(screen.getByTestId('sidebar-nav'))).toBeLessThan(
      kids.indexOf(screen.getByTestId('outlet').closest('main')!.parentElement!),
    )
    expect(kids.indexOf(screen.getByTestId('outlet').closest('main')!.parentElement!)).toBeLessThan(kids.indexOf(docked))
    // Le panneau coulissant du site n'est pas monté.
    expect(screen.queryByTestId('asset-drawer')).not.toBeInTheDocument()
  })

  it('capacité fausse (site) : panneau coulissant des référentiels, aucun volet ancré', () => {
    setCapabilities(false)
    renderWithProviders(<AppShell />)
    expect(screen.getByTestId('asset-drawer')).toBeInTheDocument()
    expect(screen.queryByTestId('asset-docked')).not.toBeInTheDocument()
  })

  it('capacité fausse (site) : barre haute et pied de page, aucune barre latérale', () => {
    setCapabilities(false)
    renderWithProviders(<AppShell />)
    expect(screen.getByTestId('nav-l1')).toBeInTheDocument()
    expect(screen.getByTestId('app-footer')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-nav')).not.toBeInTheDocument()
    expect(screen.getByTestId('outlet')).toBeInTheDocument()
    expect(screen.getByTestId('progress')).toHaveAttribute('data-placement', 'defaut')
  })

  it('avant l’hydratation du bootstrap : la disposition du site', () => {
    renderWithProviders(<AppShell />)
    expect(screen.getByTestId('nav-l1')).toBeInTheDocument()
    expect(screen.queryByTestId('sidebar-nav')).not.toBeInTheDocument()
  })
})
