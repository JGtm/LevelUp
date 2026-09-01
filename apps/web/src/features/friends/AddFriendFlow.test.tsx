/**
 * Tests — AddFriendFlow modale + hook useAddFriend (§3 plan Squad/Sessions).
 *
 * Couvre : titre dynamique avec gamertag, fermeture click backdrop, déjà ami,
 * appel PATCH /settings via mock MSW.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AddFriendModal } from './AddFriendFlow'
import { server } from '@/test/setup'

const SETTINGS_BASE = {
  lang: 'fr',
  discord_lang: 'fr',
  user_timezone: 'UTC',
  normalize_mode_labels: true,
  show_records: true,
  refresh_clears_caches: false,
  career_top_exclude_btb: false,
  media_captures_base_dir: '',
  media_buffer_minutes: 0,
  media_watcher_enabled: false,
  media_watcher_debounce_seconds: 0,
  discord_notifications_enabled: false,
  discord_notify_sync: false,
  discord_notify_backfill: false,
  discord_notify_new_version: false,
  discord_notify_friends: false,
  spnkr_auto_sync_enabled: false,
  spnkr_auto_sync_interval_hours: 0,
  spnkr_auto_sync_interval_minutes: 0,
  watcher_presence_enabled: false,
  watcher_subscribed_players: [],
  spnkr_refresh_with_backfill: false,
  spnkr_refresh_backfill_medals: false,
  spnkr_refresh_backfill_skill: false,
  spnkr_refresh_backfill_aliases: false,
  spnkr_refresh_backfill_personal_scores: false,
  spnkr_refresh_backfill_performance_scores: false,
  spnkr_refresh_backfill_lusr: false,
  spnkr_refresh_backfill_events: false,
  session_gap_minutes: 90,
  session_split_on_ranked_change: false,
  session_team_change_mode: 'group',
  outcome_exclude_bot_matches_from_badges: false,
  outcome_exclude_bot_matches_from_records: false,
  outcome_badge_sensitivity: 'standard',
  can_self_provision: true,
  can_start_initial_sync: true,
  auth_provider: 'sisu',
}

function renderWithClient(
  ui: React.ReactElement,
  initialFriends: string[] = [],
) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  // Préremplir le cache pour éviter l'attente du GET /settings (sinon le
  // 1er render lit settings=undefined et le check "déjà ami" est faussé).
  qc.setQueryData(['settings'], { ...SETTINGS_BASE, friend_gamertags: initialFriends })
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

describe('AddFriendModal — UI', () => {
  it('ne se monte pas quand open=false', () => {
    renderWithClient(
      <AddFriendModal gamertag="Alice" open={false} onClose={vi.fn()} locale="fr" />,
    )
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('affiche le titre avec le gamertag fourni', () => {
    renderWithClient(
      <AddFriendModal gamertag="Alice" open={true} onClose={vi.fn()} locale="fr" />,
    )
    expect(screen.getByText(/Ajouter Alice comme ami/i)).toBeTruthy()
  })

  it('appelle onClose au clic sur Annuler', () => {
    const onClose = vi.fn()
    renderWithClient(
      <AddFriendModal gamertag="Alice" open={true} onClose={onClose} locale="fr" />,
    )
    fireEvent.click(screen.getByText(/Annuler/i))
    expect(onClose).toHaveBeenCalled()
  })

  it('appelle onClose au clic sur le backdrop', () => {
    const onClose = vi.fn()
    renderWithClient(
      <AddFriendModal gamertag="Alice" open={true} onClose={onClose} locale="fr" />,
    )
    const dialog = screen.getByRole('dialog')
    fireEvent.click(dialog)
    expect(onClose).toHaveBeenCalled()
  })

  it('locale EN affiche le texte anglais', () => {
    renderWithClient(
      <AddFriendModal gamertag="Bob" open={true} onClose={vi.fn()} locale="en" />,
    )
    expect(screen.getByText(/Add Bob as a friend/i)).toBeTruthy()
    expect(screen.getByText(/Cancel/i)).toBeTruthy()
  })
})

describe('AddFriendModal — soumission', () => {
  it('PATCH /settings et invoque onSuccess + onClose au clic sur Confirmer', async () => {
    let patchPayload: { friend_gamertags?: string[] } | null = null
    server.use(
      http.patch('/api/v1/settings', async ({ request }) => {
        patchPayload = (await request.json()) as { friend_gamertags?: string[] }
        return HttpResponse.json({ ...SETTINGS_BASE, friend_gamertags: patchPayload?.friend_gamertags ?? [] })
      }),
    )
    const onClose = vi.fn()
    const onSuccess = vi.fn()
    renderWithClient(
      <AddFriendModal
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
    expect(patchPayload).toEqual({ friend_gamertags: ['Charlie'] })
  })

  it('détecte un gamertag déjà ami et ferme sans PATCH', async () => {
    let patchCalled = false
    server.use(
      http.patch('/api/v1/settings', () => {
        patchCalled = true
        return HttpResponse.json(SETTINGS_BASE)
      }),
    )
    const onClose = vi.fn()
    const onSuccess = vi.fn()
    renderWithClient(
      <AddFriendModal
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
    expect(patchCalled).toBe(false)
  })
})
