/**
 * Tests — fenêtre « À propos » de l'application de bureau.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useState, type ComponentPropsWithoutRef } from 'react'
import { act, fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer/feedbackDrawer.store'
import {
  GITHUB_LICENSE_URL,
  GITHUB_URL,
  MICROSOFT_CONTENT_NOTICE,
  PAYPAL_URL,
  SPONSORS_URL,
  XBOX_GAME_CONTENT_RULES_URL,
} from '@/lib/appLinks'
import { AboutDialog } from './AboutDialog'

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

const DATA_ROOT = 'C:\\Users\\Test\\AppData\\Local\\LevelUp'

function Harness({ initiallyOpen = true }: { initiallyOpen?: boolean }) {
  const [open, setOpen] = useState(initiallyOpen)
  return (
    <>
      <button type="button" onClick={() => setOpen(true)}>
        ouvrir-test
      </button>
      <AboutDialog open={open} onClose={() => setOpen(false)} />
    </>
  )
}

function hrefOf(name: RegExp): string | null {
  return screen.getByRole('link', { name }).getAttribute('href')
}

describe('AboutDialog', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr', appInfo: { version: '8.1.0', data_root: DATA_ROOT } })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
    useAppShellStore.setState({ locale: 'fr', appInfo: null })
    useFeedbackDrawerStore.setState({ isOpen: false })
  })

  it('ne rend rien tant qu’elle est fermée', () => {
    renderWithProviders(<AboutDialog open={false} onClose={() => {}} />)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('est une modale nommée, avec la version, et le focus sur « Fermer »', () => {
    renderWithProviders(<Harness />)
    const dialog = screen.getByRole('dialog', { name: 'LevelUp' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(screen.getByTestId('about-version')).toHaveTextContent('Version 8.1.0')
    expect(screen.getByRole('button', { name: 'Fermer' })).toHaveFocus()
  })

  it('se ferme par le bouton, par Échap et par un clic sur le fond, et rend le focus au déclencheur', async () => {
    renderWithProviders(<Harness initiallyOpen={false} />)
    const opener = screen.getByRole('button', { name: 'ouvrir-test' })

    for (const close of [
      () => fireEvent.click(screen.getByRole('button', { name: 'Fermer' })),
      () => fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' }),
      () => fireEvent.mouseDown(screen.getByRole('dialog').parentElement as HTMLElement),
    ]) {
      opener.focus()
      fireEvent.click(opener)
      expect(screen.getByRole('dialog')).toBeInTheDocument()
      close()
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
      expect(opener).toHaveFocus()
    }
  })

  it('un clic dans la fenêtre ne la ferme pas', () => {
    renderWithProviders(<Harness />)
    fireEvent.mouseDown(screen.getByRole('dialog'))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('Tab reste dans la fenêtre (dernier vers premier, premier vers dernier)', () => {
    renderWithProviders(<Harness />)
    const dialog = screen.getByRole('dialog')
    const items = Array.from(dialog.querySelectorAll<HTMLElement>('a[href], button:not([disabled])'))
    const first = items[0]
    const last = items[items.length - 1]
    last.focus()
    fireEvent.keyDown(dialog, { key: 'Tab' })
    expect(first).toHaveFocus()
    first.focus()
    fireEvent.keyDown(dialog, { key: 'Tab', shiftKey: true })
    expect(last).toHaveFocus()
  })

  it('Échap ferme la fenêtre même quand le focus est tombé sur le corps de la page', async () => {
    renderWithProviders(<Harness />)
    ;(document.activeElement as HTMLElement).blur()
    expect(document.body).toHaveFocus()
    fireEvent.keyDown(document.body, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
  })

  it('Tab ramène le focus dans la fenêtre quand il en est sorti (corps de la page)', () => {
    renderWithProviders(<Harness />)
    const dialog = screen.getByRole('dialog')
    const items = Array.from(dialog.querySelectorAll<HTMLElement>('a[href], button:not([disabled])'))
    ;(document.activeElement as HTMLElement).blur()
    fireEvent.keyDown(document.body, { key: 'Tab' })
    expect(dialog).toContainElement(document.activeElement as HTMLElement)
    expect(items[0]).toHaveFocus()
    ;(document.activeElement as HTMLElement).blur()
    fireEvent.keyDown(document.body, { key: 'Tab', shiftKey: true })
    expect(items[items.length - 1]).toHaveFocus()
  })

  it('Tab depuis le dernier élément, événement reçu hors de la fenêtre, revient au premier', () => {
    renderWithProviders(<Harness />)
    const dialog = screen.getByRole('dialog')
    const items = Array.from(dialog.querySelectorAll<HTMLElement>('a[href], button:not([disabled])'))
    items[items.length - 1].focus()
    fireEvent.keyDown(document, { key: 'Tab' })
    expect(items[0]).toHaveFocus()
  })

  it('pendant la recherche de mise à jour, le focus reste dans la fenêtre et Échap ferme', async () => {
    let release: (r: Response) => void = () => {}
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise<Response>((resolve) => (release = resolve))))
    renderWithProviders(<Harness />)
    const check = screen.getByRole('button', { name: 'Rechercher une mise à jour' })
    check.focus()
    await act(async () => {
      fireEvent.click(check)
    })
    expect(check).toBeDisabled()
    expect(screen.getByRole('dialog')).toContainElement(document.activeElement as HTMLElement)
    // Chromium renvoie le focus sur le corps quand le bouton focalisé se désactive.
    ;(document.activeElement as HTMLElement).blur()
    fireEvent.keyDown(document.body, { key: 'Escape' })
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    await act(async () => {
      release(new Response(null))
    })
  })

  it('porte les liens du projet, du soutien et des données, par lib/appLinks', () => {
    renderWithProviders(<Harness />)
    expect(hrefOf(/^Nouveautés$/)).toBe('/changelog')
    expect(hrefOf(/^Code source$/)).toBe(GITHUB_URL)
    expect(hrefOf(/Licence AGPL-3\.0/)).toBe(GITHUB_LICENSE_URL)
    expect(hrefOf(/Mentions tierces/)).toBe(
      `${GITHUB_URL}/blob/v8.1.0/apps/desktop/THIRD_PARTY_NOTICES.md`,
    )
    expect(hrefOf(/GitHub Sponsors/)).toBe(SPONSORS_URL)
    expect(hrefOf(/PayPal/)).toBe(PAYPAL_URL)
    expect(hrefOf(/Confidentialité/)).toBe('/privacy')
    expect(hrefOf(/Conditions d'utilisation/)).toBe('/terms')
    // Aucun lien du projet n'ouvre de fenêtre : la coquille envoie les liens web au navigateur par défaut.
    for (const link of screen.getAllByRole('link')) {
      if (link.getAttribute('href')?.startsWith('http') && link.closest('[data-testid="microsoft-notice"]') === null) {
        expect(link).not.toHaveAttribute('target')
      }
    }
  })

  it('« Signaler un problème » ferme la fenêtre et ouvre le tiroir de retour', async () => {
    renderWithProviders(<Harness />)
    fireEvent.click(screen.getByRole('button', { name: 'Signaler un problème' }))
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())
    expect(useFeedbackDrawerStore.getState().isOpen).toBe(true)
  })

  it('montre le dossier des données et le copie avec un retour visuel', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    renderWithProviders(<Harness />)
    expect(screen.getByTestId('about-data-root')).toHaveTextContent(DATA_ROOT)

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Copier' }))
    })

    expect(writeText).toHaveBeenCalledWith(DATA_ROOT)
    expect(await screen.findByRole('button', { name: 'Copié' })).toBeInTheDocument()
  })

  it('une copie refusée ne montre pas de fausse confirmation', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('refusé'))
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    vi.spyOn(console, 'error').mockImplementation(() => {})
    renderWithProviders(<Harness />)

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Copier' }))
    })

    expect(writeText).toHaveBeenCalledTimes(1)
    expect(screen.getByRole('button', { name: 'Copier' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Copié' })).not.toBeInTheDocument()
  })

  it('sans informations d’installation, pas de section de dossier ni de version', () => {
    useAppShellStore.setState({ appInfo: null })
    renderWithProviders(<Harness />)
    expect(screen.queryByTestId('about-data-root')).not.toBeInTheDocument()
    expect(screen.queryByTestId('about-version')).not.toBeInTheDocument()
  })

  it('porte la mention Microsoft imposée, ffmpeg LGPL et les sons retirés sur demande', () => {
    renderWithProviders(<Harness />)
    expect(screen.getByTestId('microsoft-notice')).toHaveTextContent(MICROSOFT_CONTENT_NOTICE)
    expect(hrefOf(/Règles d'usage du contenu de jeu de Microsoft/)).toBe(XBOX_GAME_CONTENT_RULES_URL)
    expect(screen.getByText(/ffmpeg et ffprobe sont livrés sous licence LGPL/)).toBeInTheDocument()
    expect(screen.getByText(/Sons du jeu retirés sur simple demande/)).toBeInTheDocument()
  })

  it('« Rechercher une mise à jour » appelle la coquille et le dit', async () => {
    const fetchSpy = vi.fn().mockResolvedValue(new Response(null))
    vi.stubGlobal('fetch', fetchSpy)
    renderWithProviders(<Harness />)

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Rechercher une mise à jour' }))
    })

    expect(fetchSpy).toHaveBeenCalledTimes(1)
    expect(fetchSpy.mock.calls[0][0]).toBe('http://levelup-shell.localhost/check-update')
    expect(screen.getByRole('status')).toHaveTextContent(/Recherche lancée/)
    // La page reste là : la fenêtre est toujours ouverte et le bouton se réarme.
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Rechercher une mise à jour' })).toBeEnabled()
  })

  it('dit quand la coquille est injoignable', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))
    vi.spyOn(console, 'warn').mockImplementation(() => {})
    renderWithProviders(<Harness />)

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Rechercher une mise à jour' }))
    })

    expect(screen.getByRole('status')).toHaveTextContent(/n'a pas pu être lancée/)
  })

  it('suit la langue de l’interface', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<Harness />)
    expect(screen.getByRole('button', { name: 'Check for updates' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Close' })).toBeInTheDocument()
    expect(screen.getByTestId('about-version')).toHaveTextContent('Version 8.1.0')
    expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument()
  })

  describe('rapport de diagnostic (can_report_diagnostic)', () => {
    const REPORT_PATH = 'C:\\Users\\Test\\AppData\\Local\\LevelUp\\diagnostics\\levelup-diagnostic.zip'

    afterEach(() => {
      useAppShellStore.setState({ capabilities: null })
    })

    it('sans la capacité, le bouton n’est pas offert', () => {
      renderWithProviders(<Harness />)
      expect(screen.queryByRole('button', { name: 'Rapport de diagnostic' })).not.toBeInTheDocument()
    })

    it('crée le rapport, montre le chemin du zip et la consigne de signalement', async () => {
      useAppShellStore.setState({ capabilities: testCapabilities({ can_report_diagnostic: true }) })
      renderWithProviders(<Harness />)
      fireEvent.click(screen.getByRole('button', { name: 'Rapport de diagnostic' }))

      expect(await screen.findByTestId('about-diagnostic-path')).toHaveTextContent(REPORT_PATH)
      expect(screen.getByText('Joignez ce fichier à votre signalement.')).toBeInTheDocument()
    })

    it('« Copier le chemin » met le chemin dans le presse-papier et le confirme', async () => {
      const writeText = vi.fn().mockResolvedValue(undefined)
      vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
      useAppShellStore.setState({ capabilities: testCapabilities({ can_report_diagnostic: true }) })
      renderWithProviders(<Harness />)
      fireEvent.click(screen.getByRole('button', { name: 'Rapport de diagnostic' }))
      await screen.findByTestId('about-diagnostic-path')

      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: 'Copier le chemin' }))
      })

      expect(writeText).toHaveBeenCalledWith(REPORT_PATH)
      expect(await screen.findByRole('button', { name: 'Chemin copié' })).toBeInTheDocument()
    })

    it('presse-papier indisponible : le chemin reste affiché, sans fausse confirmation', async () => {
      vi.stubGlobal('navigator', { ...navigator, clipboard: undefined })
      vi.spyOn(console, 'error').mockImplementation(() => {})
      useAppShellStore.setState({ capabilities: testCapabilities({ can_report_diagnostic: true }) })
      renderWithProviders(<Harness />)
      fireEvent.click(screen.getByRole('button', { name: 'Rapport de diagnostic' }))
      await screen.findByTestId('about-diagnostic-path')

      await act(async () => {
        fireEvent.click(screen.getByRole('button', { name: 'Copier le chemin' }))
      })

      expect(screen.getByTestId('about-diagnostic-path')).toHaveTextContent(REPORT_PATH)
      expect(screen.queryByRole('button', { name: 'Chemin copié' })).not.toBeInTheDocument()
    })

    it('un rapport déjà en cours (409) est dit depuis son code', async () => {
      server.use(
        http.post('/api/v1/desktop/diagnostic', () =>
          HttpResponse.json(
            { status: 409, code: 'diagnostic_already_running', message: 'MESSAGE DU SERVEUR' },
            { status: 409 },
          ),
        ),
      )
      useAppShellStore.setState({ capabilities: testCapabilities({ can_report_diagnostic: true }) })
      renderWithProviders(<Harness />)
      fireEvent.click(screen.getByRole('button', { name: 'Rapport de diagnostic' }))
      expect(await screen.findByRole('alert')).toHaveTextContent('Un rapport est déjà en cours de création.')
      expect(screen.queryByText('MESSAGE DU SERVEUR')).not.toBeInTheDocument()
    })

    it('un échec est dit, en anglais quand la langue est en', async () => {
      server.use(
        http.post('/api/v1/desktop/diagnostic', () =>
          HttpResponse.json({ status: 500, code: 'internal_error', message: 'x' }, { status: 500 }),
        ),
      )
      useAppShellStore.setState({ locale: 'en', capabilities: testCapabilities({ can_report_diagnostic: true }) })
      renderWithProviders(<Harness />)
      fireEvent.click(screen.getByRole('button', { name: 'Diagnostic report' }))
      expect(await screen.findByRole('alert')).toHaveTextContent('The report could not be created.')
    })
  })
})
