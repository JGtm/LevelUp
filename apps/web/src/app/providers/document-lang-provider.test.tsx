/**
 * document-lang-provider.test.tsx — garde-rail : la locale applicative doit se
 * refléter sur `document.documentElement.lang` en BCP-47 (fr -> fr-FR,
 * en -> en-US). Empêche une régression vers un `lang` figé côté shell HTML
 * (a11y + format des contrôles natifs sous Firefox).
 */
import { describe, it, expect, afterEach } from 'vitest'
import { render } from '@testing-library/react'
import { useAppShellStore } from '@/stores/appShellStore'
import { DocumentLangProvider } from './document-lang-provider'

afterEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  document.documentElement.removeAttribute('lang')
})

describe('DocumentLangProvider', () => {
  it('pose lang="fr-FR" pour la locale fr', () => {
    useAppShellStore.setState({ locale: 'fr' })
    render(
      <DocumentLangProvider>
        <span />
      </DocumentLangProvider>,
    )
    expect(document.documentElement.lang).toBe('fr-FR')
  })

  it('pose lang="en-US" pour la locale en', () => {
    useAppShellStore.setState({ locale: 'en' })
    render(
      <DocumentLangProvider>
        <span />
      </DocumentLangProvider>,
    )
    expect(document.documentElement.lang).toBe('en-US')
  })

  it('met à jour lang au changement de locale via setLocale (fr -> en)', () => {
    useAppShellStore.setState({ locale: 'fr' })
    const { rerender } = render(
      <DocumentLangProvider>
        <span />
      </DocumentLangProvider>,
    )
    expect(document.documentElement.lang).toBe('fr-FR')

    useAppShellStore.getState().setLocale('en')
    rerender(
      <DocumentLangProvider>
        <span />
      </DocumentLangProvider>,
    )
    expect(document.documentElement.lang).toBe('en-US')
  })
})
