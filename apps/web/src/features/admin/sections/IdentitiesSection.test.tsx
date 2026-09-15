/**
 * IdentitiesSection.test.tsx — l'annuaire des joueurs : lignes, badges
 * d'anomalies, état vide, et l'interrupteur « Instance fermée ».
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { AdminIdentitiesResponse } from '@/lib/api/types'
import { IdentitiesSection } from './IdentitiesSection'

const dataRef: { current: AdminIdentitiesResponse | undefined } = { current: undefined }
const lockRef = { locked: false, isPending: false, isError: false, setLocked: vi.fn() }

vi.mock('../management/identitiesQueries', () => ({
  useAdminIdentities: () => ({
    data: dataRef.current,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    isFetching: false,
  }),
}))

vi.mock('../management/useInstanceLock', () => ({
  useInstanceLock: () => lockRef,
}))

function makeData(): AdminIdentitiesResponse {
  return {
    generated_at: '2026-09-15T12:00:00Z',
    counts: { identities: 2, warnings: 1, infos: 1 },
    identities: [
      {
        xuid: '999',
        gamertag: 'Inconnu',
        profiles: [],
        account: { username: 'inconnu', role: 'user' },
        token: { has_refresh_token: true, reauth_required: false },
        watched: [],
        anomalies: [
          { code: 'account_without_profile', severity: 'warning', detail: 'inconnu' },
        ],
      },
      {
        xuid: '111',
        gamertag: 'Spartan',
        profiles: [
          {
            title_slug: 'halo_infinite',
            key: 'Spartan',
            sync_enabled: true,
            auth_only: false,
            dir_exists: true,
            db_exists: true,
          },
        ],
        watched: ['halo_infinite'],
        anomalies: [{ code: 'profile_without_account', severity: 'info' }],
      },
    ],
  }
}

function rowTexts(): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => tr.textContent ?? '')
}

beforeEach(() => {
  lockRef.locked = false
  lockRef.isPending = false
  lockRef.isError = false
  lockRef.setLocked = vi.fn()
})

describe('IdentitiesSection', () => {
  it('rend une ligne par identité, avec le libellé FR de chaque anomalie', () => {
    dataRef.current = makeData()
    render(<IdentitiesSection />)

    const rows = rowTexts()
    expect(rows).toHaveLength(2)
    // Les identités à regarder passent devant (tri par défaut sur les warning).
    expect(rows[0]).toContain('Inconnu')
    expect(rows[0]).toContain('Compte sans profil')
    expect(rows[1]).toContain('Spartan')
    expect(rows[1]).toContain('Profil sans compte')
    // Le profil est rendu par son titre, le suivi live aussi.
    expect(rows[1]).toContain('halo_infinite')
    // Le xuid est affiché tel quel (copiable).
    expect(screen.getByText('999')).toBeTruthy()
  })

  it('affiche les compteurs d en-tête', () => {
    dataRef.current = makeData()
    render(<IdentitiesSection />)
    expect(screen.getByText('identités')).toBeTruthy()
    expect(screen.getByText('à regarder')).toBeTruthy()
  })

  it('état vide : aucune identité, pas de tableau', () => {
    dataRef.current = { generated_at: '', counts: {}, identities: [] }
    render(<IdentitiesSection />)
    expect(screen.getByText('Aucune identité.')).toBeTruthy()
    expect(document.querySelector('tbody')).toBeNull()
  })

  it('code d anomalie inconnu du web : affiché brut plutôt que muet', () => {
    const data = makeData()
    data.identities = [
      {
        xuid: '111',
        gamertag: 'Spartan',
        profiles: [],
        watched: [],
        anomalies: [{ code: 'code_du_futur', severity: 'warning' }],
      },
    ]
    dataRef.current = data
    render(<IdentitiesSection />)
    expect(screen.getByText('code_du_futur')).toBeTruthy()
  })
})

describe('IdentitiesSection — interrupteur « Instance fermée »', () => {
  it('reflète l état courant du verrou', () => {
    dataRef.current = makeData()
    lockRef.locked = true
    render(<IdentitiesSection />)
    const box = screen.getByRole('checkbox') as HTMLInputElement
    expect(box.checked).toBe(true)
  })

  it('un clic demande la bascule avec la bonne valeur', () => {
    dataRef.current = makeData()
    render(<IdentitiesSection />)
    fireEvent.click(screen.getByRole('checkbox'))
    expect(lockRef.setLocked).toHaveBeenCalledWith(true)
  })

  it('désactivé pendant l envoi (pas de double bascule)', () => {
    dataRef.current = makeData()
    lockRef.isPending = true
    render(<IdentitiesSection />)
    expect((screen.getByRole('checkbox') as HTMLInputElement).disabled).toBe(true)
    expect(screen.getByText('Enregistrement…')).toBeTruthy()
  })

  it('échec de la bascule : message visible, jamais un silence', () => {
    dataRef.current = makeData()
    lockRef.isError = true
    render(<IdentitiesSection />)
    expect(screen.getByText("Le verrou n'a pas pu être modifié.")).toBeTruthy()
  })
})
