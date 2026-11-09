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
import { SaveStatusBadge } from './SaveStatusBadge'
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
    useSaveFeedbackStore.getState().clear()
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

  it('désactiver la détection envoie le PATCH et confirme par le badge de la page', async () => {
    serveurPatch()
    renderWithProviders(
      <>
        <DesktopIntegrationCard />
        <SaveStatusBadge />
      </>,
    )
    await screen.findByText('Détecter le jeu en cours')
    fireEvent.click(screen.getAllByRole('button')[0])
    await waitFor(() => expect(patches).toEqual([{ game_process_watch: false }]))
    expect(await screen.findByText('✓ Enregistré')).toBeInTheDocument()
    expect(screen.queryByText('Enregistré.')).not.toBeInTheDocument()
  })

  it('activer les notifications Windows envoie le PATCH (désactivées par défaut)', async () => {
    serveurPatch()
    renderWithProviders(<DesktopIntegrationCard />)
    await screen.findByText('Notifications Windows')
    fireEvent.click(screen.getAllByRole('button')[1])
    await waitFor(() => expect(patches).toEqual([{ native_notifications: true }]))
  })

  it('activer la réduction dans la zone de notification envoie le PATCH (croix = quitter par défaut)', async () => {
    serveurPatch()
    renderWithProviders(<DesktopIntegrationCard />)
    await screen.findByText('Réduire dans la zone de notification à la fermeture')
    fireEvent.click(screen.getAllByRole('button')[2])
    await waitFor(() => expect(patches).toEqual([{ close_to_tray: true }]))
  })

  it('activer le démarrage avec Windows envoie le PATCH (désactivé par défaut)', async () => {
    serveurPatch()
    renderWithProviders(<DesktopIntegrationCard />)
    await screen.findByText('Démarrer avec Windows')
    fireEvent.click(screen.getAllByRole('button')[3])
    await waitFor(() => expect(patches).toEqual([{ start_with_windows: true }]))
  })

  it('activer la tâche planifiée envoie le PATCH et dit son état (désactivée par défaut)', async () => {
    serveurPatch()
    server.use(
      http.get(`${API}/settings/desktop`, () =>
        HttpResponse.json({ game_running: false, scheduled_refresh: { state: 'absent' } }),
      ),
    )
    renderWithProviders(<DesktopIntegrationCard />)
    expect(await screen.findByText('Tâche planifiée : absente.')).toBeInTheDocument()
    fireEvent.click(screen.getAllByRole('button')[4])
    await waitFor(() => expect(patches).toEqual([{ scheduled_refresh: true }]))
  })

  it.each([
    ['present', undefined, 'Tâche planifiée : présente.'],
    ['unknown', undefined, 'Tâche planifiée : pas encore vérifiée.'],
    ['error', 'schtasks_failed', /Le planificateur de Windows n'a pas répondu/],
    ['error', 'unsupported_path', /ne sait pas protéger/],
  ])("l'état %s de la tâche planifiée (%s) est dit en français", async (state, code, text) => {
    server.use(
      http.get(`${API}/settings/desktop`, () =>
        HttpResponse.json({ game_running: false, scheduled_refresh: { state, error_code: code } }),
      ),
    )
    renderWithProviders(<DesktopIntegrationCard />)
    expect(await screen.findByText(text)).toBeInTheDocument()
  })

  it('sans la capacité de la coquille, les notifications Windows ne sont pas offertes', async () => {
    useAppShellStore.setState({ capabilities: testCapabilities({ can_watch_game_process: true }) })
    renderWithProviders(<DesktopIntegrationCard />)
    await screen.findByText('Détecter le jeu en cours')
    expect(screen.queryByText('Notifications Windows')).not.toBeInTheDocument()
  })

  it("affiche tel quel un refus du serveur, avec le badge d'échec de la page", async () => {
    serveurPatch({ status: 400, message: 'Ce réglage n’existe que dans l’application de bureau.' })
    renderWithProviders(
      <>
        <DesktopIntegrationCard />
        <SaveStatusBadge />
      </>,
    )
    await screen.findByText('Détecter le jeu en cours')
    fireEvent.click(screen.getAllByRole('button')[0])
    expect(await screen.findByText('✗ Erreur lors de la sauvegarde')).toBeInTheDocument()
    expect(screen.getByText('Ce réglage n’existe que dans l’application de bureau.')).toBeInTheDocument()
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
