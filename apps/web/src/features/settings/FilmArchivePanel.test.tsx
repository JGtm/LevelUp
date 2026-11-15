/**
 * Tests composant — FilmArchivePanel (archive des films : conserver, où, occupation).
 */
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { filmArchiveFixture } from '@/test/handlers'
import { server } from '@/test/setup'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import { FilmArchivePanel } from './FilmArchivePanel'
import { useSaveFeedbackStore } from './saveFeedbackStore'

const API = '/api/v1'

/** Corps des PATCH /settings reçus, dans l'ordre. */
let patches: Record<string, unknown>[] = []

function serveurPatch(reponse?: { status: number; message: string }) {
  server.use(
    http.patch(`${API}/settings`, async ({ request }) => {
      const body = (await request.json()) as Record<string, unknown>
      patches.push(body)
      if (reponse) {
        return HttpResponse.json(
          { status: reponse.status, code: 'refus', message: reponse.message },
          { status: reponse.status },
        )
      }
      return HttpResponse.json({ archive_films: true, film_archive_dir: '', ...body })
    }),
  )
}

describe('FilmArchivePanel', () => {
  beforeEach(() => {
    patches = []
    useAppShellStore.setState({ locale: 'fr' })
  })
  afterEach(() => {
    vi.unstubAllGlobals()
    useSaveFeedbackStore.getState().clear()
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('affiche l’occupation : nombre de films, taille et espace du volume', async () => {
    server.use(
      http.get(`${API}/settings/film-archive`, () =>
        HttpResponse.json({
          ...filmArchiveFixture,
          films: 1,
          bytes: 2_500_000_000,
          volume: { free_bytes: 40_000_000_000, total_bytes: 100_000_000_000, used_percent: 60, status: 'ok' },
        }),
      ),
    )
    renderWithProviders(<FilmArchivePanel />)
    expect(await screen.findByText(/1 film · 2,5 Go/)).toBeInTheDocument()
    expect(screen.getByText(/40 Go libres sur 100 Go/)).toBeInTheDocument()
  })

  it('signale un volume presque plein', async () => {
    server.use(
      http.get(`${API}/settings/film-archive`, () =>
        HttpResponse.json({
          ...filmArchiveFixture,
          volume: { free_bytes: 5_000_000_000, total_bytes: 100_000_000_000, used_percent: 95, status: 'critical' },
        }),
      ),
    )
    renderWithProviders(<FilmArchivePanel />)
    expect(await screen.findByText('Volume saturé')).toBeInTheDocument()
  })

  it('désactiver la conservation envoie archive_films=false', async () => {
    serveurPatch()
    renderWithProviders(<FilmArchivePanel />)
    await screen.findByText(/Conserver les films/)
    fireEvent.click(screen.getByRole('button', { name: '' }))
    await waitFor(() => expect(patches).toEqual([{ archive_films: false }]))
  })

  it('un enregistrement réussi passe par le badge partagé, sans texte sous la carte', async () => {
    serveurPatch()
    renderWithProviders(<FilmArchivePanel />)
    await screen.findByText(/Conserver les films/)
    fireEvent.click(screen.getByRole('button', { name: '' }))
    await waitFor(() => expect(useSaveFeedbackStore.getState().status).toBe('saved'))
    expect(screen.queryByText('Enregistré.')).not.toBeInTheDocument()
  })

  it('dans l’assistant, le badge partagé est monté par la carte et montre l’enregistrement', async () => {
    serveurPatch()
    renderWithProviders(<FilmArchivePanel variant="wizard" />)
    await screen.findByText(/Conserver les films/)
    fireEvent.click(screen.getByRole('button', { name: '' }))
    expect(await screen.findByText('✓ Enregistré')).toBeInTheDocument()
  })

  it('affiche tel quel le refus du serveur (rétention finie)', async () => {
    const refus = 'Sans archive des films, la rétention des rejeux doit rester illimitée (0).'
    serveurPatch({ status: 400, message: refus })
    renderWithProviders(<FilmArchivePanel />)
    await screen.findByText(/Conserver les films/)
    fireEvent.click(screen.getByRole('button', { name: '' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(refus)
    expect(useSaveFeedbackStore.getState().status).toBe('error')
  })

  it('enregistre le dossier saisi, et seulement s’il a changé', async () => {
    serveurPatch()
    renderWithProviders(<FilmArchivePanel />)
    const enregistrer = await screen.findByRole('button', { name: 'Enregistrer le dossier' })
    expect(enregistrer).toBeDisabled()
    fireEvent.change(screen.getByLabelText("Dossier de l'archive"), { target: { value: ' D:\\Films ' } })
    expect(enregistrer).toBeEnabled()
    fireEvent.click(enregistrer)
    await waitFor(() => expect(patches).toEqual([{ film_archive_dir: 'D:\\Films' }]))
  })

  it('rappelle la commande de déplacement quand l’archive contient des films', async () => {
    server.use(
      http.get(`${API}/settings/film-archive`, () =>
        HttpResponse.json({ ...filmArchiveFixture, films: 12, bytes: 1_000_000 }),
      ),
    )
    renderWithProviders(<FilmArchivePanel />)
    expect(await screen.findByText(/levelup film archive-move/)).toBeInTheDocument()
  })

  it('demande le redémarrage quand le dossier enregistré n’est pas encore celui qui sert', async () => {
    server.use(
      http.get(`${API}/settings/film-archive`, () =>
        HttpResponse.json({ ...filmArchiveFixture, configured_dir: 'D:\\Films', restart_required: true }),
      ),
    )
    renderWithProviders(<FilmArchivePanel />)
    expect(await screen.findByText(/Redémarrer le programme/)).toBeInTheDocument()
  })

  it('est rédigée en anglais quand la langue est en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<FilmArchivePanel />)
    expect(await screen.findByText('Keep films')).toBeInTheDocument()
    expect(screen.getByText('Film archive')).toBeInTheDocument()
  })

  it('« Parcourir… » remplit le champ et enregistre le dossier choisi, aussi dans l’assistant', async () => {
    serveurPatch()
    useAppShellStore.setState({ capabilities: testCapabilities(desktopCapabilities()) })
    const invoke = vi.fn().mockResolvedValue('D:\\Films choisis')
    vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
    renderWithProviders(<FilmArchivePanel variant="wizard" />)
    await screen.findByText(/Conserver les films/)

    fireEvent.click(screen.getByRole('button', { name: 'Parcourir…' }))

    await waitFor(() => expect(patches).toEqual([{ film_archive_dir: 'D:\\Films choisis' }]))
    expect(invoke).toHaveBeenCalledWith('pick_folder', { initial: '' })
  })

  it('une annulation de « Parcourir… » laisse le champ et les réglages inchangés', async () => {
    serveurPatch()
    useAppShellStore.setState({ capabilities: testCapabilities(desktopCapabilities()) })
    const invoke = vi.fn().mockResolvedValue(null)
    vi.stubGlobal('__TAURI_INTERNALS__', { invoke })
    renderWithProviders(<FilmArchivePanel />)
    const champ = await screen.findByLabelText("Dossier de l'archive")
    fireEvent.change(champ, { target: { value: 'D:\\Saisi' } })

    fireEvent.click(screen.getByRole('button', { name: 'Parcourir…' }))

    await waitFor(() => expect(invoke).toHaveBeenCalledWith('pick_folder', { initial: 'D:\\Saisi' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Parcourir…' })).toBeEnabled())
    expect(champ).toHaveValue('D:\\Saisi')
    expect(patches).toEqual([])
  })

  it('sans la capacité, pas de bouton « Parcourir… »', async () => {
    renderWithProviders(<FilmArchivePanel />)
    await screen.findByLabelText("Dossier de l'archive")
    expect(screen.queryByRole('button', { name: 'Parcourir…' })).not.toBeInTheDocument()
  })

  it('dans l’assistant, ne rend pas l’occupation ni le titre de carte', async () => {
    renderWithProviders(<FilmArchivePanel variant="wizard" />)
    expect(await screen.findByText(/Conserver les films/)).toBeInTheDocument()
    expect(screen.getByText(/Ce choix se change plus tard dans les réglages/)).toBeInTheDocument()
    expect(screen.queryByText('Archive des films')).not.toBeInTheDocument()
    expect(screen.queryByText('Occupation')).not.toBeInTheDocument()
  })
})
