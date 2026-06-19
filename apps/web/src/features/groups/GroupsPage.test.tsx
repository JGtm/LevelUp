/**
 * Tests — GroupsPage (Phase 3 multi-groupes).
 *
 * Couvre : rendu des groupes + membres + badge propriétaire, création (POST /groups),
 * génération d'invitation (POST /groups/{id}/invites) + copie du lien /join?invite=.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { GroupsPage } from './GroupsPage'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { Group } from '@/lib/api/types'

const GROUP: Group = {
  id: 'grp_1',
  name: 'Famille',
  owner_xuid: 'alice-x',
  members: [
    { xuid: 'alice-x', gamertag: 'Alice', role: 'owner', joined_at: '' },
    { xuid: 'bob-x', gamertag: 'Bob', role: 'member', joined_at: '' },
  ],
  created_at: '',
  updated_at: '',
}

function renderPage(groups: Group[] = [GROUP]) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  qc.setQueryData(['groups'], groups)
  return render(
    <QueryClientProvider client={qc}>
      <GroupsPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  // Owner = alice-x → boutons propriétaire visibles.
  useAppShellStore.setState({ locale: 'fr', linkedHaloIdentity: { xuid: 'alice-x', gamertag: 'Alice' } })
})

describe('GroupsPage — rendu', () => {
  it('affiche le groupe, ses membres et le badge propriétaire', () => {
    renderPage()
    expect(screen.getByText('Famille')).toBeTruthy()
    expect(screen.getByText('Alice')).toBeTruthy()
    expect(screen.getByText('Bob')).toBeTruthy()
    expect(screen.getByText(/propriétaire/i)).toBeTruthy()
    // Propriétaire → boutons Renommer/Supprimer présents.
    expect(screen.getByText(/Renommer/i)).toBeTruthy()
    expect(screen.getByText(/Supprimer/i)).toBeTruthy()
  })

  it('liste vide → message dédié', () => {
    renderPage([])
    expect(screen.getByText(/aucun groupe/i)).toBeTruthy()
  })
})

describe('GroupsPage — création', () => {
  it('POST /groups avec le nom saisi', async () => {
    let payload: { name?: string } | null = null
    server.use(
      http.post('/api/v1/groups', async ({ request }) => {
        payload = (await request.json()) as { name?: string }
        return HttpResponse.json({ ...GROUP, id: 'grp_2', name: payload?.name ?? '' }, { status: 201 })
      }),
      http.get('/api/v1/groups', () => HttpResponse.json([GROUP])),
    )
    renderPage()
    fireEvent.change(screen.getByPlaceholderText(/nouveau groupe/i), { target: { value: 'Amis' } })
    fireEvent.click(screen.getByRole('button', { name: /^Créer$/i }))
    await waitFor(() => expect(payload).toEqual({ name: 'Amis' }))
  })
})

describe('GroupsPage — invitation', () => {
  it('POST /groups/{id}/invites et copie le lien /join?invite=', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })
    server.use(
      http.post('/api/v1/groups/grp_1/invites', () =>
        HttpResponse.json({ code: 'ABC12345', created_by: 'Alice', created_at: '', expires_at: '', group_id: 'grp_1' }, { status: 201 }),
      ),
    )
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /Inviter un ami/i }))
    await waitFor(() => expect(writeText).toHaveBeenCalled())
    const link = writeText.mock.calls[0][0] as string
    expect(link.endsWith('/join?invite=ABC12345')).toBe(true)
  })
})
