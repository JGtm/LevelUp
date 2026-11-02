/**
 * SectionHeader — garde-rail de l'en-tête de page : CHAQUE section de `L1_SECTIONS` (sauf l'accueil,
 * dont la bannière d'identité porte le titre) rend un `PageHeader` au titre de la section, avec un
 * chapeau fr et en. Une section nouvelle sans chapeau, ou dont ni `SectionHeader` ni son layout ne
 * rend d'en-tête, fait échouer ce test.
 *
 * La section dont le layout rend lui-même l'en-tête (Escouade) est cadenassée par un rendu réel de
 * `SquadLayout` (`features/squad/SquadLayout.nav.test.tsx`).
 *
 * Sont aussi cadenassés ici les onglets d'Ascension (5e onglet « Tactique » derrière la capacité
 * `replay`, « un onglet mort ment ») et le remplacement Citations -> Commendations des titres sans
 * citations calculées.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { resolveRoutePath } from '@/test/routeLinkMock'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { KNOWN_LOCALES } from '@/lib/i18n/generated/locales'

let mockPathname = ''

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
vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { show_progression: true } }),
}))

import { PAGE_HEADER_TABS_HEIGHT_CLASS } from '@/components/foundations/pageHeaderLogic'
import { SectionHeader } from './SectionHeader'
import { sectionWithHeader } from './sectionHeaderLogic'
import { L1_SECTIONS } from './navL1Sections'

const ALL_CAPABILITIES = ['career', 'lusr', 'media', 'replay', 'season_pass', 'world.leaderboard', 'citations']

function setTitle(capabilities: string[], locale: 'fr' | 'en' = 'fr') {
  useAppShellStore.setState({
    locale,
    currentTitleSlug: 'un_titre',
    availableTitles: [
      { slug: 'un_titre', name: 'T', status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
    ] as unknown as ReturnType<typeof useAppShellStore.getState>['availableTitles'],
  })
}

function pathOf(template: string): string {
  return resolveRoutePath(template, { titleSlug: 'un_titre', playerSlug: 'p' })
}

beforeEach(() => {
  mockPathname = ''
  setTitle(ALL_CAPABILITIES)
})
afterEach(() => useAppShellStore.setState({ locale: 'fr', availableTitles: [] }))

describe('SectionHeader : chaque section L1 rend son en-tête', () => {
  const rendues = L1_SECTIONS.filter((s) => s.key !== 'home' && !s.headerInLayout)

  it('couvre toutes les sections hors accueil et hors en-tête de layout', () => {
    expect(rendues.map((s) => s.key)).toEqual(['stats', 'career', 'ascension', 'community', 'media', 'explorer'])
  })

  for (const section of rendues) {
    it(`${section.key} : un h1 au libellé de la section et son chapeau`, () => {
      mockPathname = pathOf(section.defaultPath)
      renderWithProviders(<SectionHeader />)
      const titres = screen.getAllByRole('heading', { level: 1 })
      expect(titres).toHaveLength(1)
      expect(titres[0]).toHaveTextContent(formatMessage(commonManifest, section.labelKey, 'fr'))
      expect(screen.getByText(formatMessage(commonManifest, section.leadKey!, 'fr'))).toBeInTheDocument()
    })
  }

  it('l’accueil n’a pas d’en-tête : sa bannière d’identité en tient lieu', () => {
    const home = L1_SECTIONS.find((s) => s.key === 'home')!
    expect(home.leadKey).toBeUndefined()
    mockPathname = pathOf(home.defaultPath)
    const { container } = renderWithProviders(<SectionHeader />)
    expect(container).toBeEmptyDOMElement()
  })

  it('une page hors section (détail d’un match) n’a pas d’en-tête de section', () => {
    mockPathname = '/t/un_titre/players/p/matches/abc'
    const { container } = renderWithProviders(<SectionHeader />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('SectionHeader : une section dont le layout rend l’en-tête', () => {
  const dansLeLayout = L1_SECTIONS.filter((s) => s.headerInLayout)

  it('SectionHeader n’en pose pas, pour ne pas le doubler', () => {
    for (const section of dansLeLayout) {
      mockPathname = pathOf(section.defaultPath)
      const { container, unmount } = renderWithProviders(<SectionHeader />)
      expect(container).toBeEmptyDOMElement()
      unmount()
    }
  })

  it('la liste de ces sections est connue : chacune a son rendu réel d’en-tête dans son layout', () => {
    expect(dansLeLayout.map((s) => s.key)).toEqual(['squad'])
  })
})

describe('chapeaux : parité des langues, phrases sans possessif ni pronom', () => {
  const sections = L1_SECTIONS.filter((s) => s.key !== 'home')

  it('chaque section hors accueil porte une clé de chapeau', () => {
    for (const section of sections) expect(section.leadKey, section.key).toMatch(/^common\.page_lead\./)
  })

  it('chaque chapeau existe dans toutes les langues, non vide, et se distingue du titre', () => {
    for (const section of sections) {
      for (const locale of KNOWN_LOCALES) {
        const lead = formatMessage(commonManifest, section.leadKey!, locale)
        const title = formatMessage(commonManifest, section.labelKey, locale)
        expect(lead.trim().length, `${section.key}/${locale}`).toBeGreaterThan(10)
        expect(lead, `${section.key}/${locale}`).not.toBe(title)
        // Phrase factuelle : jamais « vos », « votre », « your ».
        expect(lead.toLowerCase(), `${section.key}/${locale}`).not.toMatch(/\b(vos?|votre|mes?|mon|your|my)\b/)
      }
    }
  })
})

describe('SectionHeader : onglets', () => {
  it('Ascension, titre AVEC `replay` : cinq onglets, Tactique en dernier', () => {
    mockPathname = '/t/un_titre/players/p/ascension'
    renderWithProviders(<SectionHeader />)
    expect(screen.getAllByRole('link').map((a) => a.textContent)).toEqual([
      'Profil',
      'Objectifs',
      'Entraînement',
      'Réalisations',
      'Tactique',
    ])
  })

  it('Ascension, titre SANS `replay` : pas d’onglet Tactique — un onglet mort ment', () => {
    setTitle(['lusr'])
    mockPathname = '/t/un_titre/players/p/ascension'
    renderWithProviders(<SectionHeader />)
    expect(screen.queryByText('Tactique')).toBeNull()
    expect(screen.getAllByRole('link')).toHaveLength(4)
  })

  it('Ascension : exactement un onglet actif, et c’est le bon', () => {
    mockPathname = '/t/un_titre/players/p/ascension/tactique'
    renderWithProviders(<SectionHeader />)
    const actifs = screen.getAllByRole('link').filter((a) => a.getAttribute('aria-current') === 'page')
    expect(actifs.map((a) => a.textContent)).toEqual(['Tactique'])
  })

  it('Ascension, sur la route Profil : « Profil » seul est actif', () => {
    mockPathname = '/t/un_titre/players/p/ascension'
    renderWithProviders(<SectionHeader />)
    const actifs = screen.getAllByRole('link').filter((a) => a.getAttribute('aria-current') === 'page')
    expect(actifs.map((a) => a.textContent)).toEqual(['Profil'])
  })

  it('en anglais, le titre et les onglets suivent la langue', () => {
    setTitle(ALL_CAPABILITIES, 'en')
    mockPathname = '/t/un_titre/players/p/ascension'
    renderWithProviders(<SectionHeader />)
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('Ascension')
    expect(screen.getByText('Tactical')).toBeInTheDocument()
  })

  it('Carrière : titre avec citations calculées, onglet Citations vers la page calculée', () => {
    mockPathname = '/t/un_titre/players/p/career'
    renderWithProviders(<SectionHeader />)
    const citations = screen.getByRole('link', { name: 'Citations' })
    expect(citations).toHaveAttribute('href', '/t/un_titre/players/p/career/citations')
  })

  it('Carrière : titre sans citations calculées, l’onglet mène aux commendations natives', () => {
    setTitle(['career'])
    mockPathname = '/t/un_titre/players/p/career'
    renderWithProviders(<SectionHeader />)
    const onglet = screen.getByRole('link', { name: 'Citations' })
    expect(onglet).toHaveAttribute('href', '/t/un_titre/players/p/career/commendations')
    expect(screen.queryByText('Pass saisonnier')).toBeNull()
  })

  it('Communauté : l’onglet Classements disparaît sans la capacité world.leaderboard', () => {
    setTitle([])
    mockPathname = '/t/un_titre/players/p/community/relations'
    renderWithProviders(<SectionHeader />)
    expect(screen.queryByText('Classements')).toBeNull()
    expect(screen.getByRole('link', { name: 'Relations' })).toHaveAttribute('aria-current', 'page')
  })

  it('Médias et Explorer : pas d’onglets, mais la ligne est réservée (même hauteur que Carrière)', () => {
    mockPathname = '/t/un_titre/players/p/career'
    const carriere = renderWithProviders(<SectionHeader />)
    const hauteurCarriere = carriere.container.querySelector(`.${PAGE_HEADER_TABS_HEIGHT_CLASS}`)
    expect(hauteurCarriere).not.toBeNull()
    carriere.unmount()
    for (const path of ['/t/un_titre/players/p/media', '/t/un_titre/players/p/explorer']) {
      mockPathname = path
      const { container, unmount } = renderWithProviders(<SectionHeader />)
      expect(container.querySelector('nav')).toBeNull()
      expect(container.querySelector(`.${PAGE_HEADER_TABS_HEIGHT_CLASS}`)).not.toBeNull()
      unmount()
    }
  })
})

describe('SectionHeader : titre sans la capacité de la section', () => {
  it.each([
    ['ascension', 'lusr'],
    ['career', 'career'],
    ['media', 'media'],
  ])('%s : sans la capacité %s, ni titre ni onglets au-dessus de « fonction indisponible »', (section, capability) => {
    setTitle(ALL_CAPABILITIES.filter((c) => c !== capability))
    mockPathname = `/t/un_titre/players/p/${section}`
    const { container } = renderWithProviders(<SectionHeader />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('sectionWithHeader', () => {
  it('rend la section du chemin, et null pour l’accueil et pour un chemin hors section', () => {
    expect(sectionWithHeader(L1_SECTIONS, pathOf('/{-$lang}/t/$titleSlug/players/$playerSlug/career'), ALL_CAPABILITIES)?.key).toBe('career')
    expect(sectionWithHeader(L1_SECTIONS, pathOf('/{-$lang}/t/$titleSlug/players/$playerSlug/home'), ALL_CAPABILITIES)).toBeNull()
    expect(sectionWithHeader(L1_SECTIONS, '/settings', ALL_CAPABILITIES)).toBeNull()
  })

  it('une section absente de la navigation (réglage show_progression coupé) garde son en-tête par URL', () => {
    const sansAscension = L1_SECTIONS.filter((s) => s.key !== 'ascension')
    expect(
      sectionWithHeader(sansAscension, pathOf('/{-$lang}/t/$titleSlug/players/$playerSlug/ascension'), ALL_CAPABILITIES)?.key,
    ).toBe('ascension')
  })

  it('ce repli respecte la capacité de la section : titre sans `lusr`, pas d’en-tête d’Ascension', () => {
    const sansAscension = L1_SECTIONS.filter((s) => s.key !== 'ascension')
    const chemin = pathOf('/{-$lang}/t/$titleSlug/players/$playerSlug/ascension')
    expect(sectionWithHeader(sansAscension, chemin, ['career'])).toBeNull()
    // Capacités pas encore chargées : tout est permis (comme la navigation).
    expect(sectionWithHeader(sansAscension, chemin, null)?.key).toBe('ascension')
  })
})
