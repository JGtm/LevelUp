/**
 * Tests composant — FilmArchivePanel (archive des films : conserver, où, occupation).
 */
import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { filmArchiveFixture } from '@/test/handlers'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { FilmArchivePanel } from './FilmArchivePanel'

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
    useAppShellStore.setState({ locale: 'fr' })
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

  it('affiche tel quel le refus du serveur (rétention finie)', async () => {
    const refus = 'Sans archive des films, la rétention des rejeux doit rester illimitée (0).'
    serveurPatch({ status: 400, message: refus })
    renderWithProviders(<FilmArchivePanel />)
    await screen.findByText(/Conserver les films/)
    fireEvent.click(screen.getByRole('button', { name: '' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(refus)
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

  it('dans l’assistant, ne rend pas l’occupation ni le titre de carte', async () => {
    renderWithProviders(<FilmArchivePanel variant="wizard" />)
    expect(await screen.findByText(/Conserver les films/)).toBeInTheDocument()
    expect(screen.getByText(/Ce choix se change plus tard dans les réglages/)).toBeInTheDocument()
    expect(screen.queryByText('Archive des films')).not.toBeInTheDocument()
    expect(screen.queryByText('Occupation')).not.toBeInTheDocument()
  })
})
