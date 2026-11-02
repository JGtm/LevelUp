/**
 * Tests composant — DesktopIntegrationCard (intégration au poste de l'application de bureau, lot L21).
 */
import { describe, it, expect, afterEach, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { DesktopIntegrationCard } from './DesktopIntegrationCard'

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
      return HttpResponse.json({ game_process_watch: true, ...body })
    }),
  )
}

describe('DesktopIntegrationCard', () => {
  beforeEach(() => {
    patches = []
    useAppShellStore.setState({ locale: 'fr', capabilities: testCapabilities(desktopCapabilities()) })
  })
  afterEach(() => {
    useAppShellStore.setState({ locale: 'fr', capabilities: null })
  })

  it('montre la détection du jeu, activée par défaut, et l’état « jeu non lancé »', async () => {
    renderWithProviders(<DesktopIntegrationCard />)
    expect(await screen.findByText('Détecter le jeu en cours')).toBeInTheDocument()
    expect(await screen.findByText("Le jeu n'est pas lancé.")).toBeInTheDocument()
  })

  it('dit quand le jeu tourne : les tâches lourdes attendent', async () => {
    server.use(http.get(`${API}/settings/desktop`, () => HttpResponse.json({ game_running: true })))
    renderWithProviders(<DesktopIntegrationCard />)
    expect(await screen.findByText(/Le jeu tourne/)).toBeInTheDocument()
  })

  it('désactiver la détection envoie le PATCH et confirme', async () => {
    serveurPatch()
    renderWithProviders(<DesktopIntegrationCard />)
    await screen.findByText('Détecter le jeu en cours')
    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(patches).toEqual([{ game_process_watch: false }]))
    expect(await screen.findByText('Enregistré.')).toBeInTheDocument()
  })

  it('affiche tel quel un refus du serveur', async () => {
    serveurPatch({ status: 400, message: 'Ce réglage n’existe que dans l’application de bureau.' })
    renderWithProviders(<DesktopIntegrationCard />)
    await screen.findByText('Détecter le jeu en cours')
    fireEvent.click(screen.getByRole('button'))
    expect(await screen.findByRole('alert')).toHaveTextContent('Ce réglage n’existe que dans l’application de bureau.')
  })

  it('sans la capacité, la ligne de détection du jeu n’est pas offerte', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_watch_game_process: false }) })
    renderWithProviders(<DesktopIntegrationCard />)
    await screen.findByText('Intégration au poste')
    expect(screen.queryByText('Détecter le jeu en cours')).not.toBeInTheDocument()
  })

  it('parle anglais quand la langue est en', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<DesktopIntegrationCard />)
    expect(await screen.findByText('Detect the running game')).toBeInTheDocument()
  })
})
