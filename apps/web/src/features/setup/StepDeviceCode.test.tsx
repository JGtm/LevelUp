/**
 * Tests composant — StepDeviceCode : le message d'échec du device-flow vient du code
 * réellement posé par le serveur sur la tentative (`error_code`), en FR et en EN.
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { StepDeviceCode } from './StepDeviceCode'

const failWith = (errorCode: string) =>
  server.use(
    http.get('/api/v1/auth/device-flow/:attemptId', () =>
      HttpResponse.json({ attempt_id: 'att-1', status: 'failed', error_code: errorCode }),
    ),
  )

describe('StepDeviceCode — échec du device-flow', () => {
  const reset = () => {
    useSetupFlowStore.getState().reset()
    useAppShellStore.setState({ locale: 'fr' })
  }
  beforeEach(reset)
  afterEach(reset)

  const fr: Array<[string, string]> = [
    ['device_flow_acquire_error', "Erreur lors de l'authentification Microsoft."],
    ['halo_exchange_error', "Impossible d'obtenir un accès Halo. Veuillez réessayer."],
    ['identity_missing', 'Impossible de résoudre le Gamertag. Réessaie dans quelques instants.'],
    ['instance_locked', 'Cette instance est fermée aux nouvelles inscriptions.'],
    ['invite_invalid', "Cette invitation n'est plus valable. Demande un nouveau lien à la personne qui t'a invité."],
    ['link_failed', "Échec de l'authentification."],
  ]
  it.each(fr)('%s : message FR', async (code, message) => {
    failWith(code)
    renderWithProviders(<StepDeviceCode />)
    expect(await screen.findByText(message, {}, { timeout: 4000 })).toBeInTheDocument()
  })

  const en: Array<[string, string]> = [
    ['device_flow_acquire_error', 'Error during Microsoft authentication.'],
    ['identity_missing', 'Unable to resolve the Gamertag. Try again shortly.'],
    ['code_inconnu', 'Authentication failed.'],
  ]
  it.each(en)('%s : message EN', async (code, message) => {
    useAppShellStore.setState({ locale: 'en' })
    failWith(code)
    renderWithProviders(<StepDeviceCode />)
    expect(await screen.findByText(message, {}, { timeout: 4000 })).toBeInTheDocument()
  })
})

describe('StepDeviceCode — succès puis 404 tardif', () => {
  beforeEach(() => {
    useSetupFlowStore.getState().reset()
    useAppShellStore.setState({ locale: 'fr' })
  })
  afterEach(() => {
    useSetupFlowStore.getState().reset()
  })

  it('une fois autorisé : un retour de focus ne relance ni start ni flow sur un 404 attempt_not_found', async () => {
    // Le serveur fait tourner la session au succès : un retour de focus refait la requête de
    // statut sur l'ancienne tentative, qui répond 404. Sans garde, l'effet de récupération
    // relançait `start` (jusqu'à 3 fois).
    let startCount = 0
    let pollCount = 0
    server.use(
      http.post('/api/v1/auth/device-flow/start', () => {
        startCount += 1
        return HttpResponse.json({ attempt_id: `att-${startCount}`, user_code: 'ABCD-1234', verification_uri: 'https://microsoft.com/devicelogin', expires_in: 900, poll_interval_sec: 5 })
      }),
      http.get('/api/v1/auth/device-flow/:attemptId', () => {
        pollCount += 1
        if (pollCount === 1) {
          return HttpResponse.json({ attempt_id: 'att-1', status: 'authorized', gamertag: 'TestPlayer', xuid: '0000000000000001', error: null })
        }
        return HttpResponse.json({ code: 'attempt_not_found', message: 'tentative introuvable', retryable: false }, { status: 404 })
      }),
    )

    renderWithProviders(<StepDeviceCode />)
    expect(await screen.findByText('✓ Authentification réussie !', {}, { timeout: 4000 })).toBeInTheDocument()

    // Retour de focus (TanStack Query écoute visibilitychange sur window).
    window.dispatchEvent(new Event('visibilitychange'))
    await waitFor(() => expect(pollCount).toBeGreaterThanOrEqual(2), { timeout: 4000 })
    // Laisse le temps à l'effet de récupération de réagir au 404.
    await new Promise((resolve) => setTimeout(resolve, 200))

    expect(startCount).toBe(1)
  })
})
