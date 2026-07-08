/**
 * NavL1MobileMenu.test.tsx — tiroir de navigation mobile (hamburger gauche).
 */
import type { ComponentPropsWithoutRef } from 'react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { NavL1MobileMenu } from './NavL1MobileMenu'
import type { L1Section } from './navL1Sections'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ children, to, ...props }: ComponentPropsWithoutRef<'a'> & { to: string }) => (
      <a href={to} {...props}>
        {children}
      </a>
    ),
  }
})

// Libellés référencés par clé i18n (résolus au rendu) — locale 'fr' fixée dans
// beforeEach ⇒ 'Accueil' / 'Solo' / 'Sessions' rendus.
const SECTIONS: L1Section[] = [
  {
    key: 'home',
    labelKey: 'common.nav.section_home',
    defaultPath: '/players/$playerSlug/home',
    matchPathname: (p) => p.includes('/home'),
  },
  {
    key: 'stats',
    labelKey: 'common.nav.section_solo',
    defaultPath: '/players/$playerSlug/stats/timeseries',
    matchPathname: (p) => p.includes('/stats'),
    tabs: [
      { key: 'sessions', labelKey: 'common.nav.tab_sessions', path: '/players/$playerSlug/stats/sessions' },
    ],
  },
]

const resolvePath = (tpl: string) => tpl.replace('$playerSlug', 'test-player')

function setup(pathname = '/players/test-player/home') {
  return renderWithProviders(
    <NavL1MobileMenu sections={SECTIONS} pathname={pathname} resolvePath={resolvePath} />,
  )
}

describe('NavL1MobileMenu', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('affiche le bouton hamburger', () => {
    setup()
    expect(screen.getByRole('button', { name: 'Ouvrir le menu de navigation' })).toBeInTheDocument()
  })

  it('le panneau est fermé au départ (aria-hidden)', () => {
    setup()
    expect(screen.getByRole('menu', { hidden: true })).toHaveAttribute('aria-hidden', 'true')
  })

  it('ouvre le panneau au clic sur le hamburger', () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir le menu de navigation' }))
    expect(screen.getByRole('menu')).toHaveAttribute('aria-hidden', 'false')
  })

  it('rend les sections et leurs onglets', () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir le menu de navigation' }))
    expect(screen.getByRole('menuitem', { name: 'Accueil' })).toHaveAttribute(
      'href',
      '/players/test-player/home',
    )
    expect(screen.getByRole('menuitem', { name: 'Sessions' })).toHaveAttribute(
      'href',
      '/players/test-player/stats/sessions',
    )
  })

  it('marque la section active via aria-current', () => {
    setup('/players/test-player/home')
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir le menu de navigation' }))
    expect(screen.getByRole('menuitem', { name: 'Accueil' })).toHaveAttribute('aria-current', 'page')
  })

  it('se ferme au clic sur un lien', () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir le menu de navigation' }))
    fireEvent.click(screen.getByRole('menuitem', { name: 'Accueil' }))
    expect(screen.getByRole('menu', { hidden: true })).toHaveAttribute('aria-hidden', 'true')
  })

  it('se ferme avec la touche Escape', () => {
    setup()
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir le menu de navigation' }))
    expect(screen.getByRole('menu')).toHaveAttribute('aria-hidden', 'false')
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.getByRole('menu', { hidden: true })).toHaveAttribute('aria-hidden', 'true')
  })
})
