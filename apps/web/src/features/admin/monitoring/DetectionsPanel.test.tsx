/**
 * Tests DetectionsPanel — flux de note in-app (dialog, plus de prompt() natif) :
 * annuler le dialog ne déclenche AUCUNE mutation ; valider envoie le PATCH de
 * statut (avec la note saisie). Reprend la sémantique W4 (annulation = zéro effet).
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

describe('DetectionsPanel — note dialog', () => {
  beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
  afterEach(() => {
    vi.restoreAllMocks()
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('Annuler le dialog → aucune mutation PATCH', async () => {
    mockDetectionsList()
    let patched = false
    server.use(
      http.patch('/api/v1/admin/monitoring/detections/:fp', () => {
        patched = true
        return HttpResponse.json({ fingerprint: 'fp-1', ok: true, status: 'resolved' })
      }),
    )

    renderWithProviders(<DetectionsPanel />)

    const resolveBtn = await screen.findByRole('button', { name: /Résoudre/i })
    fireEvent.click(resolveBtn)

    // Le dialog s'ouvre ; on annule.
    const cancelBtn = await screen.findByRole('button', { name: /Annuler/i })
    fireEvent.click(cancelBtn)

    // Laisser une éventuelle mutation partir (elle ne DOIT PAS partir).
    await new Promise((r) => setTimeout(r, 60))
    expect(patched).toBe(false)
  })

  it('Valider le dialog → mutation PATCH envoyée avec la note', async () => {
    mockDetectionsList()
    let patchedStatus: string | undefined
    let patchedNote: string | undefined
    server.use(
      http.patch('/api/v1/admin/monitoring/detections/:fp', async ({ request }) => {
        const body = (await request.json()) as { status?: string; note?: string }
        patchedStatus = body.status
        patchedNote = body.note
        return HttpResponse.json({ fingerprint: 'fp-1', ok: true, status: 'resolved' })
      }),
    )

    renderWithProviders(<DetectionsPanel />)

    const resolveBtn = await screen.findByRole('button', { name: /Résoudre/i })
    fireEvent.click(resolveBtn)

    const textarea = await screen.findByRole('textbox')
    fireEvent.change(textarea, { target: { value: 'note test' } })

    const confirmBtn = await screen.findByRole('button', { name: /Valider/i })
    fireEvent.click(confirmBtn)

    await waitFor(() => expect(patchedStatus).toBe('resolved'))
    expect(patchedNote).toBe('note test')
  })
})
