/**
 * Tests composant — ExportForDesktopCard (jeton d'export vers l'application de bureau, site hébergé, lot L22).
 */
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { act, screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { ExportForDesktopCard } from './ExportForDesktopCard'

const API = '/api/v1'
const TOKEN = 'jeton-' + 'e9'.repeat(14)

describe('ExportForDesktopCard', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities({ can_export_for_desktop: true }) })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('sans la capacité d’export (application de bureau), rien n’est rendu', () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_export_for_desktop: false }) })
    const { container } = renderWithProviders(<ExportForDesktopCard />)
    expect(container).toBeEmptyDOMElement()
  })

  it('n’affiche aucun jeton avant le clic', () => {
    renderWithProviders(<ExportForDesktopCard />)
    expect(screen.getByRole('button', { name: "Générer un jeton d'export" })).toBeInTheDocument()
    expect(screen.queryByTestId('export-token')).not.toBeInTheDocument()
  })

  it('génère le jeton, l’affiche une fois avec son échéance et la consigne', async () => {
    renderWithProviders(<ExportForDesktopCard />)
    fireEvent.click(screen.getByRole('button', { name: "Générer un jeton d'export" }))

    expect(await screen.findByTestId('export-token')).toHaveTextContent(TOKEN)
    expect(screen.getByText(/Valable 15 minutes, à usage unique/)).toBeInTheDocument()
    expect(
      screen.getByText("Collez-le dans l'application de bureau, Réglages > Application > Importer depuis le site."),
    ).toBeInTheDocument()
    // Un seul jeton à la fois : le bouton de génération laisse la place au jeton.
    expect(screen.queryByRole('button', { name: "Générer un jeton d'export" })).not.toBeInTheDocument()
  })

  it('« Copier » met le jeton dans le presse-papier et le confirme', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    vi.stubGlobal('navigator', { ...navigator, clipboard: { writeText } })
    renderWithProviders(<ExportForDesktopCard />)
    fireEvent.click(screen.getByRole('button', { name: "Générer un jeton d'export" }))
    await screen.findByTestId('export-token')

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Copier' }))
    })

    expect(writeText).toHaveBeenCalledWith(TOKEN)
    expect(await screen.findByRole('button', { name: 'Copié' })).toBeInTheDocument()
  })

  it('« Masquer le jeton » l’efface de l’écran', async () => {
    renderWithProviders(<ExportForDesktopCard />)
    fireEvent.click(screen.getByRole('button', { name: "Générer un jeton d'export" }))
    await screen.findByTestId('export-token')
    fireEvent.click(screen.getByRole('button', { name: 'Masquer le jeton' }))
    await waitFor(() => expect(screen.queryByTestId('export-token')).not.toBeInTheDocument())
    expect(screen.queryByText(TOKEN)).not.toBeInTheDocument()
  })

  it('la limite d’un export par heure (429) est écrite depuis son code', async () => {
    server.use(
      http.post(`${API}/export/desktop/token`, () =>
        HttpResponse.json(
          { status: 429, code: 'export_rate_limited', message: 'MESSAGE DU SERVEUR' },
          { status: 429 },
        ),
      ),
    )
    renderWithProviders(<ExportForDesktopCard />)
    fireEvent.click(screen.getByRole('button', { name: "Générer un jeton d'export" }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Un export par heure.')
    expect(screen.queryByText('MESSAGE DU SERVEUR')).not.toBeInTheDocument()
    expect(screen.queryByTestId('export-token')).not.toBeInTheDocument()
  })

  it('une autre erreur donne le texte générique', async () => {
    server.use(
      http.post(`${API}/export/desktop/token`, () =>
        HttpResponse.json({ status: 500, code: 'internal_error', message: 'x' }, { status: 500 }),
      ),
    )
    renderWithProviders(<ExportForDesktopCard />)
    fireEvent.click(screen.getByRole('button', { name: "Générer un jeton d'export" }))
    expect(await screen.findByRole('alert')).toHaveTextContent("Le jeton n'a pas pu être généré.")
  })

  it('parle anglais quand la langue est en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<ExportForDesktopCard />)
    fireEvent.click(screen.getByRole('button', { name: 'Generate an export token' }))
    expect(await screen.findByText(/Valid for 15 minutes, single-use/)).toBeInTheDocument()
  })
})
