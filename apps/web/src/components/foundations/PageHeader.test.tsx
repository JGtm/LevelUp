/**
 * PageHeader : un h1 unique, un chapeau, une hauteur de zone titre FIXE (aucun saut de page avec ou
 * sans sous-titre ni actions), des onglets à hauteur fixe filtrés par capacité de titre.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'

let mockPathname = '/t/un_titre/players/p/career'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({
      children,
      to,
      params,
      ...props
    }: ComponentPropsWithoutRef<'a'> & { to: string; params?: Record<string, string | undefined> }) => (
      <a href={resolveRoutePath(to, params)} {...props}>
        {children}
      </a>
    ),
    useParams: () => ({ playerSlug: 'p' }),
    useRouterState: ({ select }: { select: (s: unknown) => unknown }) =>
      select({ location: { pathname: mockPathname } }),
  }
})

import { PageHeader } from './PageHeader'
import {
  activeTabKey,
  PAGE_HEADER_TABS_HEIGHT_CLASS,
  PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS,
  type PageHeaderTab,
} from './pageHeaderLogic'

const BASE = '/{-$lang}/t/$titleSlug/players/$playerSlug'
const TABS: PageHeaderTab[] = [
  { key: 'progression', label: 'Progression', to: `${BASE}/career` },
  { key: 'citations', label: 'Citations', to: `${BASE}/career/citations` },
  { key: 'season-pass', label: 'Pass saisonnier', to: `${BASE}/career/season-pass`, capability: 'season_pass' },
]

function setCapabilities(capabilities: string[]) {
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: 'un_titre',
    availableTitles: [
      { slug: 'un_titre', name: 'T', status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
    ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
  })
}

beforeEach(() => {
  mockPathname = '/t/un_titre/players/p/career'
  setCapabilities(['season_pass'])
})

describe('PageHeader : titre et chapeau', () => {
  it('rend UN seul h1 (le titre) et le chapeau', () => {
    renderWithProviders(<PageHeader title="Carrière" lead="Progression de rang." />)
    const titres = screen.getAllByRole('heading', { level: 1 })
    expect(titres).toHaveLength(1)
    expect(titres[0]).toHaveTextContent('Carrière')
    expect(screen.getByText('Progression de rang.')).toBeInTheDocument()
  })

  it('le sous-titre et les actions n’apparaissent que fournis', () => {
    const { rerender } = renderWithProviders(<PageHeader title="T" lead="L" />)
    expect(screen.queryByText('Sous-titre')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Action' })).toBeNull()
    rerender(<PageHeader title="T" lead="L" subtitle="Sous-titre" actions={<button type="button">Action</button>} />)
    expect(screen.getByText('Sous-titre')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument()
  })
})

describe('PageHeader : aucune page ne saute', () => {
  function zoneTitre(container: HTMLElement): HTMLElement {
    return container.querySelector('h1')!.parentElement!.parentElement as HTMLElement
  }

  it('la zone titre porte la même hauteur minimale avec et sans sous-titre ni actions', () => {
    const nu = renderWithProviders(<PageHeader title="T" lead="L" />)
    const complet = renderWithProviders(
      <PageHeader title="T" lead="L" subtitle="S" actions={<span>A</span>} />,
    )
    expect(PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS).toMatch(/^min-h-/)
    expect(zoneTitre(nu.container).className).toContain(PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS)
    expect(zoneTitre(complet.container).className).toContain(PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS)
  })

  it('la ligne d’onglets a une hauteur fixe quand elle est présente, et n’existe pas sans onglets', () => {
    const sans = renderWithProviders(<PageHeader title="T" lead="L" />)
    expect(sans.container.querySelector('nav')).toBeNull()
    const avec = renderWithProviders(<PageHeader title="T" lead="L" tabs={TABS} />)
    expect(avec.container.querySelector('nav')?.className).toContain(PAGE_HEADER_TABS_HEIGHT_CLASS)
  })
})

describe('PageHeader : onglets', () => {
  it('filtre les onglets par capacité de titre', () => {
    setCapabilities([])
    renderWithProviders(<PageHeader title="T" lead="L" tabs={TABS} />)
    expect(screen.getAllByRole('link').map((a) => a.textContent)).toEqual(['Progression', 'Citations'])
  })

  it('résout les liens avec le titre et le joueur courants', () => {
    renderWithProviders(<PageHeader title="T" lead="L" tabs={TABS} />)
    expect(screen.getAllByRole('link').map((a) => a.getAttribute('href'))).toEqual([
      '/t/un_titre/players/p/career',
      '/t/un_titre/players/p/career/citations',
      '/t/un_titre/players/p/career/season-pass',
    ])
  })

  it('un seul onglet actif (aria-current), l’exact avant le préfixe', () => {
    mockPathname = '/t/un_titre/players/p/career/citations'
    renderWithProviders(<PageHeader title="T" lead="L" tabs={TABS} />)
    const actifs = screen.getAllByRole('link').filter((a) => a.getAttribute('aria-current') === 'page')
    expect(actifs.map((a) => a.textContent)).toEqual(['Citations'])
  })
})

describe('activeTabKey', () => {
  it('exact, puis plus long préfixe de segments, sinon aucun', () => {
    expect(activeTabKey('/career', TABS)).toBe('progression')
    expect(activeTabKey('/career/citations', TABS)).toBe('citations')
    expect(activeTabKey('/career/citations/detail', TABS)).toBe('citations')
    expect(activeTabKey('/careers', TABS)).toBeNull()
    expect(activeTabKey('/stats/timeseries', TABS)).toBeNull()
    expect(activeTabKey(null, TABS)).toBeNull()
  })
})
