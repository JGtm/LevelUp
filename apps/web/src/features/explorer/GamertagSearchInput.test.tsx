/**
 * Tests d'intégration GamertagSearchInput.
 *
 * Vérifie l'harmonisation avec GamertagCombobox :
 *  - Affiche les Joueurs configurés en tête (même comportement que Combobox)
 *  - Appelle onSelect au clic
 *  - Affiche le message vide unifié
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { PlayerSummary, GamertagSearchResponse } from '@/lib/api/types'

import { GamertagSearchInput } from './GamertagSearchInput'

function asPlayer(gamertag: string): PlayerSummary {
  return {
    player_slug: gamertag.toLowerCase(),
    gamertag,
    xuid: `xuid-${gamertag}`,
  } as PlayerSummary
}

describe('GamertagSearchInput', () => {
  beforeEach(() => {
    useAppShellStore.setState({
      availablePlayers: [asPlayer('AlphaPlayer'), asPlayer('BravoGamer')],
    })
  })

  it('affiche les Joueurs configurés en tête quand on tape', () => {
    renderWithProviders(<GamertagSearchInput onSelect={() => {}} />)
    const input = screen.getByPlaceholderText(/Rechercher un joueur/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'Alpha' } })

    expect(screen.getByText('Joueurs configurés')).toBeInTheDocument()
    expect(screen.getByText('AlphaPlayer')).toBeInTheDocument()
  })

  it('appelle onSelect au clic sur une suggestion', () => {
    const onSelect = vi.fn()
    renderWithProviders(<GamertagSearchInput onSelect={onSelect} />)
    const input = screen.getByPlaceholderText(/Rechercher un joueur/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'Alpha' } })

    fireEvent.click(screen.getByText('AlphaPlayer'))
    expect(onSelect).toHaveBeenCalledWith('AlphaPlayer')
  })

  it('affiche le message vide unifié quand 0 résultat', async () => {
    useAppShellStore.setState({ availablePlayers: [] })
    server.use(
      http.get('*/directory/gamertags/search', () =>
        HttpResponse.json<GamertagSearchResponse>({ query: 'xq', items: [] }),
      ),
    )
    renderWithProviders(<GamertagSearchInput onSelect={() => {}} />)
    const input = screen.getByPlaceholderText(/Rechercher un joueur/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'xq' } })

    await waitFor(
      () => {
        expect(screen.getByText(/Aucun joueur trouvé pour "xq"/)).toBeInTheDocument()
      },
      { timeout: 2000 },
    )
  })

  // ─── Harmonisation Face à Face : recherche d'un joueur inconnu ───────────────

  it('affiche le bouton "Rechercher" pour un gamertag hors suggestions, et appelle onSelect au clic', () => {
    const onSelect = vi.fn()
    renderWithProviders(<GamertagSearchInput onSelect={onSelect} />)
    const input = screen.getByPlaceholderText(/Rechercher un joueur/i)
    fireEvent.focus(input)
    // 'Zz' ne correspond à aucun joueur configuré (Alpha/Bravo) → saisie libre.
    fireEvent.change(input, { target: { value: 'Zz' } })

    const button = screen.getByText(/Rechercher "Zz"/)
    expect(button).toBeInTheDocument()
    fireEvent.click(button)
    expect(onSelect).toHaveBeenCalledWith('Zz')
  })

  it('Entrée recherche le texte exact quand le joueur est inconnu', () => {
    const onSelect = vi.fn()
    renderWithProviders(<GamertagSearchInput onSelect={onSelect} />)
    const input = screen.getByPlaceholderText(/Rechercher un joueur/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'Zz' } })
    fireEvent.keyDown(input, { key: 'Enter' })
    expect(onSelect).toHaveBeenCalledWith('Zz')
  })

  // ─── Repli live Xbox explicite (V72-24) ──────────────────────────────────────

  it('propose « Rechercher sur Xbox » pour un joueur inconnu et affiche le résultat live au clic', async () => {
    useAppShellStore.setState({ availablePlayers: [] })
    server.use(
      http.get('*/directory/gamertags/search', ({ request }) => {
        const live = new URL(request.url).searchParams.get('live')
        if (live === '1') {
          return HttpResponse.json<GamertagSearchResponse>({
            query: 'NeverSeen',
            items: [{ gamertag: 'NeverSeen', xuid: 'xuid-live', score: 0, exact_match: true }],
          })
        }
        // Recherche locale (typeahead) : rien, mais aucun blocage live.
        return HttpResponse.json<GamertagSearchResponse>({ query: 'NeverSeen', items: [] })
      }),
    )
    const onSelect = vi.fn()
    renderWithProviders(<GamertagSearchInput onSelect={onSelect} />)
    const input = screen.getByPlaceholderText(/Rechercher un joueur/i)
    fireEvent.focus(input)
    fireEvent.change(input, { target: { value: 'NeverSeen' } })

    // Le bouton apparaît une fois la recherche locale revenue bredouille.
    const btn = await screen.findByText('Rechercher sur Xbox', undefined, { timeout: 2000 })
    fireEvent.click(btn)

    // Le joueur résolu côté Xbox devient sélectionnable.
    const hit = await screen.findByText('NeverSeen', undefined, { timeout: 2000 })
    fireEvent.click(hit)
    expect(onSelect).toHaveBeenCalledWith('NeverSeen')
  })
})
