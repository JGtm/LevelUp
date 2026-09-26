/**
 * Tests — PrivacyPage + parité FR/EN du texte de confidentialité.
 */
import { describe, it, expect, afterEach, vi } from 'vitest'
import type { ComponentPropsWithoutRef } from 'react'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { PrivacyPage } from './PrivacyPage'
import { getPrivacyText, PRIVACY_UPDATED_AT, CONTACT_TOKEN } from './i18n'
import { privacyContactEmail } from '@/lib/appLinks'

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

describe('PrivacyPage', () => {
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('rend le titre, la date de révision et toutes les sections en français', () => {
    renderWithProviders(<PrivacyPage />)
    const text = getPrivacyText('fr')
    expect(screen.getByRole('heading', { level: 1, name: text.title })).toBeInTheDocument()
    expect(screen.getByText(new RegExp(PRIVACY_UPDATED_AT.fr))).toBeInTheDocument()
    for (const section of text.sections) {
      expect(screen.getByRole('heading', { level: 2, name: section.heading })).toBeInTheDocument()
    }
  })

  it('bascule intégralement en anglais avec la locale', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<PrivacyPage />)
    const text = getPrivacyText('en')
    expect(screen.getByRole('heading', { level: 1, name: text.title })).toBeInTheDocument()
    for (const section of text.sections) {
      expect(screen.getByRole('heading', { level: 2, name: section.heading })).toBeInTheDocument()
    }
  })

  it('offre un retour vers l’application', () => {
    renderWithProviders(<PrivacyPage />)
    const back = getPrivacyText('fr').backToApp
    expect(screen.getByRole('link', { name: back })).toHaveAttribute('href', '/')
  })

  it('le jeton de contact devient un lien mailto vers l’alias, dans les deux langues', () => {
    const email = privacyContactEmail()
    for (const locale of ['fr', 'en'] as const) {
      useAppShellStore.setState({ locale })
      const { unmount } = renderWithProviders(<PrivacyPage />)
      expect(screen.getByRole('link', { name: email })).toHaveAttribute(
        'href',
        `mailto:${email}`,
      )
      // Le jeton brut ne doit JAMAIS rester affiché.
      expect(screen.queryByText(new RegExp(CONTACT_TOKEN.replace(/[{}]/g, '\\$&')))).toBeNull()
      unmount()
    }
  })
})

describe('texte de confidentialité — parité FR/EN', () => {
  it('les deux langues ont le même nombre de sections, dans le même ordre', () => {
    const fr = getPrivacyText('fr')
    const en = getPrivacyText('en')
    expect(en.sections).toHaveLength(fr.sections.length)
    expect(en.intro).toHaveLength(fr.intro.length)
    fr.sections.forEach((section, i) => {
      expect(en.sections[i].paragraphs?.length ?? 0).toBe(section.paragraphs?.length ?? 0)
      expect(en.sections[i].bullets?.length ?? 0).toBe(section.bullets?.length ?? 0)
    })
  })

  it('aucun titre ni paragraphe vide', () => {
    for (const locale of ['fr', 'en'] as const) {
      const text = getPrivacyText(locale)
      expect(text.title.trim()).not.toBe('')
      for (const section of text.sections) {
        expect(section.heading.trim()).not.toBe('')
        for (const p of [...(section.paragraphs ?? []), ...(section.bullets ?? [])]) {
          expect(p.trim().length).toBeGreaterThan(10)
        }
      }
    }
  })

  it('les faits vérifiables du code sont bien énoncés (nom du cookie, durée, absence de traceur)', () => {
    const flat = (locale: 'fr' | 'en') =>
      getPrivacyText(locale)
        .sections.flatMap((s) => [s.heading, ...(s.paragraphs ?? []), ...(s.bullets ?? [])])
        .join(' ')
    for (const locale of ['fr', 'en'] as const) {
      expect(flat(locale)).toContain('levelup_session')
      expect(flat(locale)).toContain('HttpOnly')
      expect(flat(locale)).toMatch(/sept jours|seven days/)
    }
  })
})
