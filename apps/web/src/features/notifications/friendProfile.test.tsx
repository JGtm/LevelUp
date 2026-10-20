/**
 * Notifications sur le profil d'un ami : réservées à leur propriétaire, lecture comprise
 * (ADR 0029). Aucune page, aucun réglage par catégorie, aucun toast, et aucune requête
 * vers les routes de notifications de l'ami (le serveur les refuserait en 403).
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { NotificationsPage } from './NotificationsPage'
import { NotificationsSettingsTab } from './NotificationsSettingsTab'
import { NotificationsToastBridge } from './toastBridge'
import { getNotificationsText } from './i18n'

const FRIEND = 'Ami'
const OWN = 'MonCompte'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: FRIEND, titleSlug: 'halo_infinite' }),
    useNavigate: () => vi.fn(),
  }
})

/** Compte les requêtes vers les routes de notifications du profil de l'ami. */
function spyFriendNotifications(): { calls: string[] } {
  const spy = { calls: [] as string[] }
  server.use(
    http.all(`/api/v1/players/${FRIEND}/notifications*`, ({ request }) => {
      spy.calls.push(`${request.method} ${new URL(request.url).pathname}`)
      return HttpResponse.json({ code: 'player_forbidden' }, { status: 403 })
    }),
  )
  return spy
}

function setFriendProfile() {
  useAppShellStore.setState({
    locale: 'fr',
    writablePlayerSlugs: [OWN],
    currentPlayer: { player_slug: FRIEND, gamertag: FRIEND } as unknown as ReturnType<
      typeof useAppShellStore.getState
    >['currentPlayer'],
  })
}

describe("Notifications sur le profil d'un ami", () => {
  beforeEach(setFriendProfile)
  afterEach(() => {
    useAppShellStore.setState({ writablePlayerSlugs: null, currentPlayer: null })
  })

  it.each(['fr', 'en'] as const)('page dédiée (%s) : message de confidentialité, aucune requête', async (locale) => {
    useAppShellStore.setState({ locale })
    const spy = spyFriendNotifications()
    renderWithProviders(<NotificationsPage />)
    expect(await screen.findByText(getNotificationsText(locale).privateToOwner)).toBeInTheDocument()
    expect(screen.queryByText(getNotificationsText(locale).dropdownMarkAllRead)).toBeNull()
    await new Promise((r) => setTimeout(r, 50))
    expect(spy.calls).toEqual([])
  })

  it('réglages : catégories remplacées par le message, préférences jamais lues', async () => {
    const spy = spyFriendNotifications()
    renderWithProviders(<NotificationsSettingsTab />)
    const t = getNotificationsText('fr')
    expect(await screen.findByText(t.privateToOwner)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: t.settingsTestButton })).toBeNull()
    await new Promise((r) => setTimeout(r, 50))
    expect(spy.calls).toEqual([])
  })

  it('toasts : aucune requête vers les notifications de l ami', async () => {
    const spy = spyFriendNotifications()
    renderWithProviders(<NotificationsToastBridge />)
    await new Promise((r) => setTimeout(r, 50))
    expect(spy.calls).toEqual([])
  })
})

describe('Notifications sur son propre profil', () => {
  afterEach(() => {
    useAppShellStore.setState({ writablePlayerSlugs: null, currentPlayer: null })
  })

  it('la page dédiée lit la liste', async () => {
    useAppShellStore.setState({ locale: 'fr', writablePlayerSlugs: [FRIEND] })
    let listed = 0
    server.use(
      http.get(`/api/v1/players/${FRIEND}/notifications`, () => {
        listed++
        return HttpResponse.json({ items: [] })
      }),
    )
    renderWithProviders(<NotificationsPage />)
    await waitFor(() => expect(listed).toBeGreaterThan(0))
    expect(screen.queryByText(getNotificationsText('fr').privateToOwner)).toBeNull()
  })
})
