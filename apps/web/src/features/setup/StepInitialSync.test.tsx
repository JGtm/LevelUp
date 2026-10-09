/**
 * StepInitialSync — un refus du lancement s'affiche dans la page (aucune éjection).
 *
 * Session valide sans jetons Halo : le serveur répond 403 `halo_tokens_missing` (jamais
 * 401 `auth_required`, que la coquille traiterait comme une session expirée).
 */
import type { ReactNode } from 'react'
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSetupFlowStore } from '@/stores/setupFlowStore'
import { StepInitialSync } from './StepInitialSync'

// Link sans contexte routeur : un ancre suffit au rendu de l'étape.
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, Link: ({ children }: { children: ReactNode }) => <a>{children}</a> }
})

describe('StepInitialSync — refus du lancement', () => {
  beforeEach(() => {
    useSetupFlowStore.getState().reset()
    useAppShellStore.setState({ activeSyncJobId: null, locale: 'fr' })
  })

  it('403 halo_tokens_missing → message dédié, aucun événement de session expirée', async () => {
    let ejections = 0
    const onAuth = () => { ejections++ }
    window.addEventListener('levelup:auth-required', onAuth)
    server.use(
      http.post('*/sync/initial', () =>
        HttpResponse.json(
          { code: 'halo_tokens_missing', message: 'Aucun jeton Halo', retryable: false },
          { status: 403 },
        ),
      ),
    )

    renderWithProviders(<StepInitialSync playerSlug="test-player" />)
    await userEvent.click(screen.getByRole('button'))

    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/Aucun compte Xbox n'est relié/)
    })
    window.removeEventListener('levelup:auth-required', onAuth)
    expect(ejections).toBe(0)
  })
})
