/**
 * PageHeader : un h1 unique, un chapeau, une zone titre de hauteur minimale fixe, une ligne d'onglets
 * de hauteur fixe RÉSERVÉE même sans onglets, collante quand elle en porte, onglets filtrés par
 * capacité de titre, un seul onglet actif.
 *
 * Rendu avec un VRAI routeur (historique mémoire) : le marquage actif d'un `Link` (`aria-current`,
 * `data-status`) est décidé par le routeur, un `Link` simulé ne montrerait pas le défaut où tout
 * préfixe de chemin allume l'onglet racine d'une sous-page.
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  RouterProvider,
} from '@tanstack/react-router'

import { useAppShellStore } from '@/stores/appShellStore'
import { PageHeader } from './PageHeader'
import {
  activeTabKey,
  PAGE_HEADER_TABS_HEIGHT_CLASS,
  PAGE_HEADER_TABS_HEIGHT_REM,
  PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS,
  STICKY_BELOW_TABS_CLASS,
  type PageHeaderTab,
} from './pageHeaderLogic'

const BASE = '/{-$lang}/t/$titleSlug/players/$playerSlug'
const TABS: PageHeaderTab[] = [
  { key: 'progression', label: 'Progression', to: `${BASE}/career` },
  { key: 'citations', label: 'Citations', to: `${BASE}/career/citations` },
  { key: 'season-pass', label: 'Pass saisonnier', to: `${BASE}/career/season-pass`, capability: 'season_pass' },
]
const HOME = `/t/un_titre/players/p/career`

function setCapabilities(capabilities: string[]) {
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: 'un_titre',
    availableTitles: [
      { slug: 'un_titre', name: 'T', status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
    ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
  })
}

/** Rend l'en-tête dans un routeur réel dont les routes sont celles des onglets (+ une sous-page). */
async function renderAt(pathname: string, tabs?: readonly PageHeaderTab[]) {
  const root = createRootRoute({
    component: () => (
      <>
        <PageHeader title="Carrière" lead="Progression de rang." tabs={tabs} tabsAriaLabel="Onglets" />
        <Outlet />
      </>
    ),
  })
  const paths = [
    ...TABS.map((t) => t.to as string),
    `${BASE}/career/citations/detail`,
    `${BASE}/media`,
  ]
  const children = paths.map((path) => createRoute({ getParentRoute: () => root, path, component: () => null }))
  const router = createRouter({
    routeTree: root.addChildren(children),
    history: createMemoryHistory({ initialEntries: [pathname] }),
  })
  await router.load()
  const view = render(<RouterProvider router={router} />)
  await screen.findByRole('heading', { level: 1 })
  return view
}

function actifs(): string[] {
  return screen
    .queryAllByRole('link')
    .filter((a) => a.getAttribute('aria-current') === 'page')
    .map((a) => a.textContent ?? '')
}

beforeEach(() => setCapabilities(['season_pass']))

describe('PageHeader : titre et chapeau', () => {
  it('rend UN seul h1 (le titre) et le chapeau', async () => {
    await renderAt(HOME)
    const titres = screen.getAllByRole('heading', { level: 1 })
    expect(titres).toHaveLength(1)
    expect(titres[0]).toHaveTextContent('Carrière')
    expect(screen.getByText('Progression de rang.')).toBeInTheDocument()
  })
})

describe('PageHeader : aucune page ne saute', () => {
  function zoneTitre(container: HTMLElement): HTMLElement {
    return container.querySelector('h1')!.parentElement as HTMLElement
  }
  function ligneOnglets(container: HTMLElement): HTMLElement {
    return container.querySelector(`.${PAGE_HEADER_TABS_HEIGHT_CLASS}`) as HTMLElement
  }

  it('la zone titre porte la hauteur minimale partagée', async () => {
    const { container } = await renderAt(HOME)
    expect(PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS).toMatch(/^min-h-/)
    expect(zoneTitre(container).className).toContain(PAGE_HEADER_TITLE_MIN_HEIGHT_CLASS)
  })

  it('la ligne d’onglets a la même hauteur et le même trait avec et sans onglets', async () => {
    const sans = await renderAt('/t/un_titre/players/p/media')
    const ligneSans = ligneOnglets(sans.container)
    expect(ligneSans).not.toBeNull()
    expect(ligneSans.className).toContain('border-b')
    expect(sans.container.querySelector('nav')).toBeNull()
    const zoneSans = zoneTitre(sans.container).className
    sans.unmount()
    const avec = await renderAt(HOME, TABS)
    const ligneAvec = ligneOnglets(avec.container)
    expect(zoneTitre(avec.container).className).toBe(zoneSans)
    expect(avec.container.querySelector('nav')).not.toBeNull()
    // Même hauteur et même trait, dans une enveloppe de même gouttière.
    expect(ligneAvec.className).toContain(PAGE_HEADER_TABS_HEIGHT_CLASS)
    expect(ligneAvec.className).toContain('border-b')
    expect(ligneAvec.parentElement!.className).toContain('px-6')
    expect(ligneSans.parentElement!.className).toContain('px-6')
  })

  it('la hauteur réservée à la ligne d’onglets est celle dont se décalent les bandeaux collants', () => {
    // h-11 = 2,75 rem = 11 x 0,25 rem : la classe, le décalage et la valeur en rem disent la même chose.
    const unites = Number(PAGE_HEADER_TABS_HEIGHT_CLASS.replace('h-', ''))
    expect(STICKY_BELOW_TABS_CLASS).toBe(`top-${unites}`)
    expect(PAGE_HEADER_TABS_HEIGHT_REM).toBe(unites * 0.25)
  })
})

describe('PageHeader : ligne d’onglets collante', () => {
  it('avec onglets : collée en haut, fond opaque, sous la barre de progression et les tiroirs (z-50)', async () => {
    const { container } = await renderAt(HOME, TABS)
    const colle = container.querySelector('nav')!.parentElement as HTMLElement
    expect(colle).toHaveClass('sticky', 'top-0', 'z-30', 'bg-background')
    // Frère de l'en-tête, pas son enfant : collante dans l'en-tête seul, elle partirait avec lui.
    expect(container.querySelector('header')!.contains(colle)).toBe(false)
  })

  it('sans onglets : la ligne vide réserve la place sans coller', async () => {
    const { container } = await renderAt('/t/un_titre/players/p/media')
    expect(container.querySelector('.sticky')).toBeNull()
  })
})

describe('PageHeader : onglets', () => {
  it('filtre les onglets par capacité de titre', async () => {
    setCapabilities([])
    await renderAt(HOME, TABS)
    expect(screen.getAllByRole('link').map((a) => a.textContent)).toEqual(['Progression', 'Citations'])
  })

  it('résout les liens avec le titre et le joueur courants', async () => {
    await renderAt(HOME, TABS)
    expect(screen.getAllByRole('link').map((a) => a.getAttribute('href'))).toEqual([
      '/t/un_titre/players/p/career',
      '/t/un_titre/players/p/career/citations',
      '/t/un_titre/players/p/career/season-pass',
    ])
  })

  it('sur une sous-page, un seul onglet est « page courante » : Progression ne l’est PAS sur /career/citations', async () => {
    await renderAt(`${HOME}/citations`, TABS)
    expect(actifs()).toEqual(['Citations'])
    const progression = screen.getByRole('link', { name: 'Progression' })
    expect(progression).not.toHaveAttribute('aria-current')
    expect(progression).not.toHaveAttribute('data-status', 'active')
  })

  it('l’onglet racine reste actif sur sa propre route, et seul', async () => {
    await renderAt(HOME, TABS)
    expect(actifs()).toEqual(['Progression'])
    expect(screen.getByRole('link', { name: 'Citations' })).not.toHaveAttribute('data-status', 'active')
  })

  it('une sous-page d’un onglet garde cet onglet seul allumé', async () => {
    await renderAt(`${HOME}/citations/detail`, TABS)
    expect(actifs()).toEqual(['Citations'])
    expect(screen.getByRole('link', { name: 'Progression' })).not.toHaveAttribute('aria-current')
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
