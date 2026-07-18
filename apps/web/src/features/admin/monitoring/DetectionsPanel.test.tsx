/**
 * Tests DetectionsPanel — W4 (revue 2026-07) : annuler le prompt de note (retour null)
 * ne doit déclencher AUCUNE mutation de statut. Une validation (texte, même vide) part.
 */
import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { DetectionsPanel } from './DetectionsPanel'

const openDetection = {
  fingerprint: 'fp-1',
  level: 'error',
  module: 'sync',
  message: 'boom',
  count: 3,
  first_seen: '2026-07-17T10:00:00Z',
  last_seen: '2026-07-17T11:00:00Z',
  status: 'open',
}

function mockDetectionsList() {
  server.use(
    http.get('/api/v1/admin/monitoring/detections', () =>
      HttpResponse.json({ generated_at: '2026-07-17T11:00:00Z', detections: [openDetection], open_count: 1 }),
    ),
  )
}

describe('DetectionsPanel — prompt cancel (W4)', () => {
  beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
  afterEach(() => {
    vi.restoreAllMocks()
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('Annuler le prompt (null) → aucune mutation PATCH', async () => {
    mockDetectionsList()
    let patched = false
    server.use(
      http.patch('/api/v1/admin/monitoring/detections/:fp', () => {
        patched = true
        return HttpResponse.json({ fingerprint: 'fp-1', ok: true, status: 'resolved' })
      }),
    )
    vi.spyOn(window, 'prompt').mockReturnValue(null)

    renderWithProviders(<DetectionsPanel />)

    const resolveBtn = await screen.findByRole('button', { name: /Résoudre/i })
    fireEvent.click(resolveBtn)

    // Laisser une éventuelle mutation partir (elle ne DOIT PAS partir).
    await new Promise((r) => setTimeout(r, 60))
    expect(window.prompt).toHaveBeenCalled()
    expect(patched).toBe(false)
  })

  it('Valider le prompt (texte) → mutation PATCH envoyée', async () => {
    mockDetectionsList()
    let patchedStatus: string | undefined
    server.use(
      http.patch('/api/v1/admin/monitoring/detections/:fp', async ({ request }) => {
        const body = (await request.json()) as { status?: string }
        patchedStatus = body.status
        return HttpResponse.json({ fingerprint: 'fp-1', ok: true, status: 'resolved' })
      }),
    )
    vi.spyOn(window, 'prompt').mockReturnValue('note test')

    renderWithProviders(<DetectionsPanel />)

    const resolveBtn = await screen.findByRole('button', { name: /Résoudre/i })
    fireEvent.click(resolveBtn)

    await waitFor(() => expect(patchedStatus).toBe('resolved'))
  })
})
