/**
 * Tests d'intégration GamertagCombobox.
 *
 * Vérifie les invariants UX :
 *  - Affiche le groupe « Joueurs configurés » avec étoile
 *  - Affiche « Coéquipiers fréquents » quand fournis
 *  - Affiche le message vide unifié quand recherche serveur retourne 0
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { PlayerSummary, GamertagSearchResponse } from '@/lib/api/types'

import { GamertagCombobox } from './GamertagCombobox'

function asPlayer(gamertag: string): PlayerSummary {
  return {
    player_slug: gamertag.toLowerCase(),
    gamertag,
    xuid: `xuid-${gamertag}`,
  } as PlayerSummary
}

describe('GamertagCombobox', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      availablePlayers: [asPlayer('AlphaPlayer'), asPlayer('BravoGamer')],
    })
  })

  it('affiche le groupe "Joueurs configurés" au focus', () => {
    renderWithProviders(
      <GamertagCombobox selected={[]} onChange={() => {}} />,
    )
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher un gamertag/i))
    expect(screen.getByText('Joueurs configurés')).toBeInTheDocument()
    expect(screen.getByText('AlphaPlayer')).toBeInTheDocument()
    expect(screen.getByText('BravoGamer')).toBeInTheDocument()
  })

  it('affiche le groupe "Coéquipiers fréquents" quand fournis', () => {
    renderWithProviders(
      <GamertagCombobox
        selected={[]}
        onChange={() => {}}
        frequentOptions={[
          { gamertag: 'CharlieX', xuid: 'xuid-CharlieX', encounter_count: 12, last_seen_at: null },
        ]}
      />,
    )
    fireEvent.focus(screen.getByPlaceholderText(/Rechercher un gamertag/i))
    expect(screen.getByText('Coéquipiers fréquents')).toBeInTheDocument()
    expect(screen.getByText('CharlieX')).toBeInTheDocument()
    expect(screen.getByText('12×')).toBeInTheDocument()
  })

  it('affiche le message vide unifié quand 0 résultat serveur', async () => {
    useAppShellStore.setState({ availablePlayers: [] })
    server.use(
      http.get('*/directory/gamertags/search', () =>
        HttpResponse.json<GamertagSearchResponse>({ query: 'xq', items: [] }),
      ),
    )

    renderWithProviders(
      <GamertagCombobox selected={[]} onChange={() => {}} />,
    )
    const input = screen.getByPlaceholderText(/Rechercher un gamertag/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'xq' } })

    await waitFor(
      () => {
        expect(screen.getByText(/Aucun joueur trouvé pour "xq"/)).toBeInTheDocument()
      },
      { timeout: 2000 },
    )
  })

  it('respecte la limite max', () => {
    let selected = ['AlphaPlayer']
    const onChange = (v: string[]) => {
      selected = v
    }
    renderWithProviders(
      <GamertagCombobox selected={selected} onChange={onChange} max={1} />,
    )
    const input = screen.getByPlaceholderText('') as HTMLInputElement
    expect(input.disabled).toBe(true)
    expect(screen.getByText('1/1')).toBeInTheDocument()
  })
})
