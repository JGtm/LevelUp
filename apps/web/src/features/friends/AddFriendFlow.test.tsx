/**
 * Tests — AddFriendFlow modale + hook useAddFriend (§3 plan Squad/Sessions).
 *
 * Couvre : titre dynamique avec gamertag, fermeture click backdrop, déjà ami,
 * appel PUT /players/{slug}/friends via mock MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AddFriendModal } from './AddFriendFlow'
import { server } from '@/test/setup'
import { queryKeys } from '@/lib/query/keys'

const PLAYER_SLUG = 'JGtm'

function renderWithClient(ui: React.ReactElement, initialFriends: string[] = []) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // Préremplir le cache pour éviter l'attente du GET (sinon le 1er render lit
  // une liste undefined et le check "déjà ami" est faussé).
  qc.setQueryData(queryKeys.playerFriends(PLAYER_SLUG), {
    xuid: 'xuid_me',
    gamertags: initialFriends,
    can_edit: true,
  })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

describe('AddFriendModal — UI', () => {
  it('ne se monte pas quand open=false', () => {
    renderWithClient(
      <AddFriendModal playerSlug={PLAYER_SLUG} gamertag="Alice" open={false} onClose={vi.fn()} locale="fr" />,
    )
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('affiche le titre avec le gamertag fourni', () => {
    renderWithClient(
      <AddFriendModal playerSlug={PLAYER_SLUG} gamertag="Alice" open={true} onClose={vi.fn()} locale="fr" />,
    )
    expect(screen.getByText(/Ajouter Alice comme ami/i)).toBeTruthy()
  })

  it('appelle onClose au clic sur Annuler', () => {
    const onClose = vi.fn()
    renderWithClient(
      <AddFriendModal playerSlug={PLAYER_SLUG} gamertag="Alice" open={true} onClose={onClose} locale="fr" />,
    )
    fireEvent.click(screen.getByText(/Annuler/i))
    expect(onClose).toHaveBeenCalled()
  })

  it('appelle onClose au clic sur le backdrop', () => {
    const onClose = vi.fn()
    renderWithClient(
      <AddFriendModal playerSlug={PLAYER_SLUG} gamertag="Alice" open={true} onClose={onClose} locale="fr" />,
    )
    const dialog = screen.getByRole('dialog')
    fireEvent.click(dialog)
    expect(onClose).toHaveBeenCalled()
  })

  it('locale EN affiche le texte anglais', () => {
    renderWithClient(
      <AddFriendModal playerSlug={PLAYER_SLUG} gamertag="Bob" open={true} onClose={vi.fn()} locale="en" />,
    )
    expect(screen.getByText(/Add Bob as a friend/i)).toBeTruthy()
    expect(screen.getByText(/Cancel/i)).toBeTruthy()
  })
})

describe('AddFriendModal — soumission', () => {
  it('PUT la liste du joueur et invoque onSuccess + onClose au clic sur Confirmer', async () => {
    let putPayload: { gamertags?: string[] } | null = null
    server.use(
      http.put(`/api/v1/players/${PLAYER_SLUG}/friends`, async ({ request }) => {
        putPayload = (await request.json()) as { gamertags?: string[] }
        return HttpResponse.json({
          xuid: 'xuid_me',
          gamertags: putPayload?.gamertags ?? [],
          can_edit: true,
        })
      }),
    )
    const onClose = vi.fn()
    const onSuccess = vi.fn()
    renderWithClient(
      <AddFriendModal
        playerSlug={PLAYER_SLUG}
        gamertag="Charlie"
        open={true}
        onClose={onClose}
        locale="fr"
        onSuccess={onSuccess}
      />,
      [], // friends initiaux vides
    )
    fireEvent.click(screen.getByRole('button', { name: /^Ajouter$/i }))
    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith('Charlie')
      expect(onClose).toHaveBeenCalled()
    })
    expect(putPayload).toEqual({ gamertags: ['Charlie'] })
  })

  it('détecte un gamertag déjà ami et ferme sans PUT', async () => {
    let putCalled = false
    server.use(
      http.put(`/api/v1/players/${PLAYER_SLUG}/friends`, () => {
        putCalled = true
        return HttpResponse.json({ xuid: 'xuid_me', gamertags: [], can_edit: true })
      }),
    )
    const onClose = vi.fn()
    const onSuccess = vi.fn()
    renderWithClient(
      <AddFriendModal
        playerSlug={PLAYER_SLUG}
        gamertag="Alice"
        open={true}
        onClose={onClose}
        locale="fr"
        onSuccess={onSuccess}
      />,
      ['alice'], // déjà dans la liste (case-insensitive)
    )
    fireEvent.click(screen.getByRole('button', { name: /^Ajouter$/i }))
    await waitFor(() => {
      expect(onClose).toHaveBeenCalled()
    })
    expect(onSuccess).not.toHaveBeenCalled()
    expect(putCalled).toBe(false)
  })
})
