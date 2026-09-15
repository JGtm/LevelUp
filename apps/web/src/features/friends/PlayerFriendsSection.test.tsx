/**
 * Tests — PlayerFriendsSection (page « Amis et groupes »).
 *
 * Couvre : lecture de la liste du joueur actif, mode LECTURE SEULE piloté par
 * `can_edit` (et non par un 403 interprété côté front), retrait d'un ami (PUT de
 * la liste complète), et l'absence de joueur actif.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { PlayerFriendsSection } from './PlayerFriendsSection'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { queryKeys } from '@/lib/query/keys'

const SLUG = 'test-player'

function setActivePlayer(slug: string | null) {
  useAppShellStore.setState({
    locale: 'fr',
    currentPlayer: slug
      ? {
          player_slug: slug,
          gamertag: 'TestPlayer',
          xuid: '0000000000000001',
          waypoint_player: 'TestPlayer',
          is_demo: false,
          sync_enabled: true,
        }
      : null,
  })
}

function renderSection(friends: { gamertags: string[]; can_edit: boolean }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(queryKeys.playerFriends(SLUG), {
    xuid: '0000000000000001',
    gamertags: friends.gamertags,
    can_edit: friends.can_edit,
  })
  return {
    qc,
    ...render(
      <QueryClientProvider client={qc}>
        <PlayerFriendsSection />
      </QueryClientProvider>,
    ),
  }
}

beforeEach(() => {
  setActivePlayer(SLUG)
})

describe('PlayerFriendsSection', () => {
  it('affiche les amis du joueur actif', () => {
    renderSection({ gamertags: ['Charlie', 'Delta'], can_edit: true })
    expect(screen.getByText('Charlie')).toBeTruthy()
    expect(screen.getByText('Delta')).toBeTruthy()
    expect(screen.getByText(/Amis de TestPlayer/i)).toBeTruthy()
  })

  it('liste vide : message dédié, pas de ligne fantôme', () => {
    renderSection({ gamertags: [], can_edit: true })
    expect(screen.getByText(/Aucun ami pour l'instant/i)).toBeTruthy()
  })

  it('can_edit=false : lecture seule, ni saisie ni bouton de retrait', () => {
    renderSection({ gamertags: ['Charlie'], can_edit: false })
    expect(screen.getByText(/lecture seule/i)).toBeTruthy()
    expect(screen.queryByPlaceholderText(/Gamertag à ajouter/i)).toBeNull()
    expect(screen.queryByRole('button', { name: /Retirer Charlie/i })).toBeNull()
  })

  it('can_edit=true : saisie et retrait disponibles', () => {
    renderSection({ gamertags: ['Charlie'], can_edit: true })
    expect(screen.getByPlaceholderText(/Gamertag à ajouter/i)).toBeTruthy()
    expect(screen.getByRole('button', { name: /Retirer Charlie/i })).toBeTruthy()
    expect(screen.queryByText(/lecture seule/i)).toBeNull()
  })

  it('retirer un ami envoie la liste COMPLÈTE restante', async () => {
    let putPayload: { gamertags?: string[] } | null = null
    server.use(
      http.put(`/api/v1/players/${SLUG}/friends`, async ({ request }) => {
        putPayload = (await request.json()) as { gamertags?: string[] }
        return HttpResponse.json({
          xuid: '0000000000000001',
          gamertags: putPayload?.gamertags ?? [],
          can_edit: true,
        })
      }),
    )
    renderSection({ gamertags: ['Charlie', 'Delta'], can_edit: true })
    fireEvent.click(screen.getByRole('button', { name: /Retirer Charlie/i }))
    await waitFor(() => {
      expect(putPayload).toEqual({ gamertags: ['Delta'] })
    })
  })

  it("le bouton d'ajout ouvre la confirmation, la saisie seule ne l'ouvre pas", () => {
    renderSection({ gamertags: [], can_edit: true })
    const input = screen.getByPlaceholderText(/Gamertag à ajouter/i)
    fireEvent.change(input, { target: { value: 'Echo' } })
    expect(screen.queryByRole('dialog')).toBeNull()
    fireEvent.click(screen.getByRole('button', { name: /^Ajouter$/i }))
    expect(screen.getByRole('dialog')).toBeTruthy()
  })

  it('aucun joueur actif : message explicite, aucune requête', () => {
    setActivePlayer(null)
    const spy = vi.fn()
    server.use(
      http.get('/api/v1/players/:slug/friends', () => {
        spy()
        return HttpResponse.json({ xuid: '', gamertags: [], can_edit: false })
      }),
    )
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={qc}>
        <PlayerFriendsSection />
      </QueryClientProvider>,
    )
    expect(screen.getByText(/Aucun joueur sélectionné/i)).toBeTruthy()
    expect(spy).not.toHaveBeenCalled()
  })
})
