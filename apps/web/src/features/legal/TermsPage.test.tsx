/**
 * Tests — TermsPage + parité FR/EN des conditions d'utilisation.
 */
import { describe, it, expect, afterEach, vi } from 'vitest'
import type { ComponentPropsWithoutRef } from 'react'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { TermsPage } from './TermsPage'
import { getTermsText, TERMS_UPDATED_AT } from './termsI18n'
import { getPrivacyText } from './i18n'
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

const POOL_HEADING = { fr: 'Mise en commun des jetons', en: 'Token pooling' } as const

describe('TermsPage', () => {
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })

  it.each(['fr', 'en'] as const)('rend titre, date et toutes les sections en %s', (locale) => {
    useAppShellStore.setState({ locale })
    renderWithProviders(<TermsPage />)
    const text = getTermsText(locale)
    expect(screen.getByRole('heading', { level: 1, name: text.title })).toBeInTheDocument()
    expect(screen.getByText(new RegExp(TERMS_UPDATED_AT[locale]))).toBeInTheDocument()
    for (const section of text.sections) {
      expect(screen.getByRole('heading', { level: 2, name: section.heading })).toBeInTheDocument()
    }
  })

  it('le jeton de contact devient un lien mailto', () => {
    renderWithProviders(<TermsPage />)
    const email = privacyContactEmail()
    expect(screen.getByRole('link', { name: email })).toHaveAttribute('href', `mailto:${email}`)
  })
})

describe('conditions d’utilisation — parité FR/EN', () => {
  it('même structure dans les deux langues', () => {
    const fr = getTermsText('fr')
    const en = getTermsText('en')
    expect(en.sections).toHaveLength(fr.sections.length)
    expect(en.intro).toHaveLength(fr.intro.length)
    fr.sections.forEach((section, i) => {
      expect(en.sections[i].paragraphs?.length ?? 0).toBe(section.paragraphs?.length ?? 0)
      expect(en.sections[i].bullets?.length ?? 0).toBe(section.bullets?.length ?? 0)
    })
  })

  it('aucun titre ni paragraphe vide', () => {
    for (const locale of ['fr', 'en'] as const) {
      const text = getTermsText(locale)
      expect(text.title.trim()).not.toBe('')
      for (const section of text.sections) {
        expect(section.heading.trim()).not.toBe('')
        for (const p of [...(section.paragraphs ?? []), ...(section.bullets ?? [])]) {
          expect(p.trim().length).toBeGreaterThan(10)
        }
      }
    }
  })
})

describe('mise en commun des jetons — cohérence des textes', () => {
  it.each(['fr', 'en'] as const)(
    'les conditions et la confidentialité décrivent la mise en commun en %s',
    (locale) => {
      const heading = POOL_HEADING[locale]
      const terms = getTermsText(locale).sections.find((s) => s.heading === heading)
      const privacy = getPrivacyText(locale).sections.find((s) => s.heading === heading)
      expect(terms, 'section dans les conditions').toBeDefined()
      expect(privacy, 'section dans la confidentialité').toBeDefined()
    },
  )

  it.each(['fr', 'en'] as const)(
    'les conditions énoncent le caractère facultatif, le retrait et la synchro plus lente en %s',
    (locale) => {
      const section = getTermsText(locale).sections.find((s) => s.heading === POOL_HEADING[locale])
      const flat = [...(section?.paragraphs ?? []), ...(section?.bullets ?? [])].join(' ')
      expect(flat).toMatch(locale === 'fr' ? /jamais une condition/ : /never a condition/)
      expect(flat).toMatch(locale === 'fr' ? /retrait/ : /withdrawal/)
      expect(flat).toMatch(locale === 'fr' ? /plus lente/ : /slower/)
    },
  )
})
