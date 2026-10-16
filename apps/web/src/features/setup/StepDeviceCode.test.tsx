/**
 * Tests composant — StepDeviceCode : le message d'échec du device-flow vient du code
 * réellement posé par le serveur sur la tentative (`error_code`), en FR et en EN.
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
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
