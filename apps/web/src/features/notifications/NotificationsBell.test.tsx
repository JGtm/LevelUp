/**
 * Tests — NotificationsBell : badge DP6 (badge_count) + auto-read DP7
 * (fermeture du dropdown → mark-read des non-lues affichées).
 */
import type { ComponentPropsWithoutRef } from 'react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { NotificationsBell } from './NotificationsBell'
import type { Notification } from './types'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ children, ...props }: ComponentPropsWithoutRef<'a'> & { to: string; params?: unknown }) => (
      <a {...props}>{children}</a>
    ),
    useNavigate: () => vi.fn(),
  }
})

const SLUG = 'test-player'

function notif(id: number, readAt: string | null = null): Notification {
  return {
    id,
    category: 'personal_record',
    severity: 'success',
    title_key: 'notif.personal_record.title',
    source: 'test',
    created_at: new Date().toISOString(),
    read_at: readAt,
  }
}

function mockApi(opts: {
  count: number
  badgeCount?: number
  items?: Notification[]
  onMarkRead?: (ids: number[]) => void
}) {
  server.use(
    http.get(`/api/v1/players/${SLUG}/notifications/unread-count`, () =>
      HttpResponse.json({
        count: opts.count,
        ...(opts.badgeCount !== undefined ? { badge_count: opts.badgeCount } : {}),
        by_category: {},
      }),
    ),
    http.get(`/api/v1/players/${SLUG}/notifications`, () =>
      HttpResponse.json({ items: opts.items ?? [] }),
    ),
    http.post(`/api/v1/players/${SLUG}/notifications/mark-read`, async ({ request }) => {
      const body = (await request.json()) as { ids: number[] }
      opts.onMarkRead?.(body.ids)
      return HttpResponse.json({ updated: body.ids.length })
    }),
  )
}

function setVisibility(state: DocumentVisibilityState) {
  Object.defineProperty(document, 'visibilityState', { configurable: true, get: () => state })
}

describe('NotificationsBell', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
  })

  afterEach(() => {
    setVisibility('visible')
  })

  it('affiche badge_count (DP6), pas le compteur complet', async () => {
    mockApi({ count: 57, badgeCount: 3 })
    renderWithProviders(<NotificationsBell playerSlug={SLUG} />)
    await waitFor(() => {
      expect(screen.getByText('3')).toBeInTheDocument()
    })
    expect(screen.queryByText('57')).toBeNull()
  })

  it('retombe sur count si badge_count absent (rétro-compat)', async () => {
    mockApi({ count: 7 })
    renderWithProviders(<NotificationsBell playerSlug={SLUG} />)
    await waitFor(() => {
      expect(screen.getByText('7')).toBeInTheDocument()
    })
  })

  it('masque le badge quand badge_count=0 même si count>0', async () => {
    mockApi({ count: 12, badgeCount: 0 })
    renderWithProviders(<NotificationsBell playerSlug={SLUG} />)
    // Le libellé aria passe en "vide" quand le badge est à 0.
    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Aucune notification non lue' }),
      ).toBeInTheDocument()
    })
    expect(screen.queryByText('12')).toBeNull()
  })

  it('DP7 : fermeture du dropdown → markRead des non-lues affichées', async () => {
    const markedRead: number[][] = []
    mockApi({
      count: 2,
      badgeCount: 2,
      items: [notif(101), notif(102)],
      onMarkRead: (ids) => markedRead.push(ids),
    })
    renderWithProviders(<NotificationsBell playerSlug={SLUG} />)

    // Ouvrir le dropdown.
    const button = await screen.findByRole('button')
    fireEvent.click(button)

    // Attendre que la liste soit rendue (2 non lues visibles).
    await waitFor(() => {
      expect(screen.getByRole('menu')).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(screen.getAllByRole('menuitem').length).toBeGreaterThanOrEqual(2)
    })

    // Fermer (Escape) → auto-read des 2 ids affichés.
    fireEvent.keyDown(document, { key: 'Escape' })
    await waitFor(() => {
      expect(markedRead.length).toBe(1)
    })
    expect([...markedRead[0]].sort()).toEqual([101, 102])
  })

  it("DP7 : fermeture sans non-lues affichées → aucun appel markRead", async () => {
    const markedRead: number[][] = []
    mockApi({
      count: 0,
      badgeCount: 0,
      items: [notif(201, new Date().toISOString())], // déjà lue
      onMarkRead: (ids) => markedRead.push(ids),
    })
    renderWithProviders(<NotificationsBell playerSlug={SLUG} />)

    const button = await screen.findByRole('button')
    fireEvent.click(button)
    await waitFor(() => {
      expect(screen.getByRole('menu')).toBeInTheDocument()
    })
    fireEvent.keyDown(document, { key: 'Escape' })
    await waitFor(() => {
      expect(screen.queryByRole('menu')).toBeNull()
    })
    expect(markedRead.length).toBe(0)
  })

  it('W5 : visibilitychange hidden (dropdown ouvert) → flush keepalive des ids', async () => {
    const markedRead: number[][] = []
    mockApi({
      count: 2,
      badgeCount: 2,
      items: [notif(101), notif(102)],
      onMarkRead: (ids) => markedRead.push(ids),
    })
    renderWithProviders(<NotificationsBell playerSlug={SLUG} />)

    const button = await screen.findByRole('button')
    fireEvent.click(button)
    await waitFor(() => {
      expect(screen.getByRole('menu')).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(screen.getAllByRole('menuitem').length).toBeGreaterThanOrEqual(2)
    })

    // L'onglet passe en arrière-plan SANS fermer le dropdown (F5/switch d'onglet).
    setVisibility('hidden')
    document.dispatchEvent(new Event('visibilitychange'))

    await waitFor(() => {
      expect(markedRead.length).toBe(1)
    })
    expect([...markedRead[0]].sort((a, b) => a - b)).toEqual([101, 102])

    // Retour au premier plan puis nouvel arrière-plan → PAS de re-flush des mêmes
    // ids (dédup : ref vidé + ids marqués flushés).
    setVisibility('visible')
    document.dispatchEvent(new Event('visibilitychange'))
    setVisibility('hidden')
    document.dispatchEvent(new Event('visibilitychange'))
    await new Promise((r) => setTimeout(r, 20))
    expect(markedRead.length).toBe(1)
  })

  it('W5 : pagehide (dropdown ouvert) → flush keepalive des ids', async () => {
    const markedRead: number[][] = []
    mockApi({
      count: 1,
      badgeCount: 1,
      items: [notif(303)],
      onMarkRead: (ids) => markedRead.push(ids),
    })
    renderWithProviders(<NotificationsBell playerSlug={SLUG} />)

    const button = await screen.findByRole('button')
    fireEvent.click(button)
    await waitFor(() => {
      expect(screen.getAllByRole('menuitem').length).toBeGreaterThanOrEqual(1)
    })

    window.dispatchEvent(new Event('pagehide'))
    await waitFor(() => {
      expect(markedRead.length).toBe(1)
    })
    expect(markedRead[0]).toEqual([303])
  })
})
