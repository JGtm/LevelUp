/**
 * Tests du sélecteur de joueur avec présence.
 *
 * Ce qui compte ici : la manette n'apparaît QUE pour un joueur réellement en
 * jeu, le compteur d'amis QUE s'il y en a, la bascule de joueur reste celle de
 * la NavL1, et le menu se pilote au clavier (c'est un `role="menu"`, plus un
 * `<select>` natif qui offrait tout cela gratuitement).
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import type { PlayerSummary } from '@/lib/api/types'

import { PlayerSwitcher } from './PlayerSwitcher'

interface PresencePayload {
  players: Array<{
    player_slug: string
    gamertag: string
    in_game: boolean
    title_slug?: string
    title_name?: string
  }> | null
  friends_in_game: number
}

function mockPresence(payload: PresencePayload) {
  server.use(http.get('/api/v1/presence', () => HttpResponse.json(payload)))
}

function asPlayer(slug: string, gamertag: string): PlayerSummary {
  return { player_slug: slug, gamertag, xuid: `xuid-${slug}` } as PlayerSummary
}

const JGTM = asPlayer('jgtm', 'JGtm')
const MADINA = asPlayer('madina', 'Madina')

function renderSwitcher(players: PlayerSummary[], onPlayerChange = vi.fn()) {
  renderWithProviders(
    <PlayerSwitcher
      players={players}
      currentPlayer={players[0] ?? null}
      locale="fr"
      onPlayerChange={onPlayerChange}
    />,
  )
}

describe('PlayerSwitcher', () => {
  beforeEach(() => {
    useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', locale: 'fr' })
    mockPresence({ players: [], friends_in_game: 0 })
  })

  it('ouvre un menu listant tous les joueurs disponibles', () => {
    renderSwitcher([JGTM, MADINA])

    fireEvent.click(screen.getByRole('button', { name: 'Sélectionner un joueur' }))

    const items = screen.getAllByRole('menuitem')
    expect(items).toHaveLength(2)
    expect(items[0]).toHaveTextContent('JGtm')
    expect(items[1]).toHaveTextContent('Madina')
  })

  it('bascule de joueur via le callback de la NavL1 et referme le menu', () => {
    const onPlayerChange = vi.fn()
    renderSwitcher([JGTM, MADINA], onPlayerChange)

    fireEvent.click(screen.getByRole('button', { name: 'Sélectionner un joueur' }))
    fireEvent.click(screen.getByRole('menuitem', { name: /Madina/ }))

    expect(onPlayerChange).toHaveBeenCalledWith('madina')
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('marque à la manette le SEUL joueur en jeu, avec le titre réel', async () => {
    mockPresence({
      players: [
        { player_slug: 'jgtm', gamertag: 'JGtm', in_game: false },
        { player_slug: 'madina', gamertag: 'Madina', in_game: true, title_slug: 'halo_5', title_name: 'Halo 5' },
      ],
      friends_in_game: 0,
    })
    renderSwitcher([JGTM, MADINA])

    fireEvent.click(screen.getByRole('button', { name: 'Sélectionner un joueur' }))

    await waitFor(() => {
      expect(screen.getByTestId('in-game-madina')).toBeInTheDocument()
    })
    expect(screen.queryByTestId('in-game-jgtm')).not.toBeInTheDocument()
    // Le titre RÉEL est nommé : un joueur peut jouer à un autre titre que celui affiché.
    expect(screen.getByTestId('in-game-madina')).toHaveAttribute('aria-label', 'En jeu sur Halo 5')
  })

  it("nomme « En jeu » sans titre quand le titre n'est pas connu", async () => {
    mockPresence({
      players: [{ player_slug: 'jgtm', gamertag: 'JGtm', in_game: true }],
      friends_in_game: 0,
    })
    renderSwitcher([JGTM, MADINA])

    await waitFor(() => {
      expect(screen.getByTestId('in-game-jgtm')).toHaveAttribute('aria-label', 'En jeu')
    })
  })

  it('affiche le compteur d\'amis en jeu avec son libellé accessible pluralisé', async () => {
    mockPresence({ players: [], friends_in_game: 3 })
    renderSwitcher([JGTM, MADINA])

    await waitFor(() => {
      expect(screen.getByTestId('friends-in-game-badge')).toHaveAttribute('aria-label', '3 amis en jeu')
    })
    expect(screen.getByTestId('friends-in-game-badge')).toHaveTextContent('3')
  })

  it('accorde le libellé au singulier pour un seul ami', async () => {
    mockPresence({ players: [], friends_in_game: 1 })
    renderSwitcher([JGTM, MADINA])

    await waitFor(() => {
      expect(screen.getByTestId('friends-in-game-badge')).toHaveAttribute('aria-label', '1 ami en jeu')
    })
  })

  it('ne rend AUCUN compteur quand personne n\'est en jeu', async () => {
    renderSwitcher([JGTM, MADINA])

    // Laisse la requête se résoudre avant de conclure à l'absence.
    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Sélectionner un joueur' })).toBeInTheDocument()
    })
    expect(screen.queryByTestId('friends-in-game-badge')).not.toBeInTheDocument()
  })

  it('reste muet si la présence est indisponible (endpoint en erreur)', async () => {
    server.use(http.get('/api/v1/presence', () => new HttpResponse(null, { status: 503 })))
    renderSwitcher([JGTM, MADINA])

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Sélectionner un joueur' })).toBeInTheDocument()
    })
    expect(screen.queryByTestId('friends-in-game-badge')).not.toBeInTheDocument()
    expect(screen.queryByTestId('in-game-jgtm')).not.toBeInTheDocument()
  })

  it('expose un déclencheur de menu ARIA (haspopup + expanded)', () => {
    renderSwitcher([JGTM, MADINA])

    const trigger = screen.getByRole('button', { name: 'Sélectionner un joueur' })
    expect(trigger).toHaveAttribute('aria-haspopup', 'menu')
    expect(trigger).toHaveAttribute('aria-expanded', 'false')

    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
  })

  it('ouvre au clavier (flèche bas) et donne le focus au premier joueur', () => {
    renderSwitcher([JGTM, MADINA])

    const trigger = screen.getByRole('button', { name: 'Sélectionner un joueur' })
    fireEvent.keyDown(trigger, { key: 'ArrowDown' })

    const items = screen.getAllByRole('menuitem')
    expect(items[0]).toHaveFocus()
  })

  it('navigue entre les joueurs aux flèches et boucle', () => {
    renderSwitcher([JGTM, MADINA])

    fireEvent.click(screen.getByRole('button', { name: 'Sélectionner un joueur' }))
    const menu = screen.getByRole('menu')
    const items = screen.getAllByRole('menuitem')

    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(items[1]).toHaveFocus()
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(items[0]).toHaveFocus() // boucle
    fireEvent.keyDown(menu, { key: 'ArrowUp' })
    expect(items[1]).toHaveFocus()
  })

  it('ferme sur Échap et rend le focus au déclencheur', () => {
    renderSwitcher([JGTM, MADINA])

    const trigger = screen.getByRole('button', { name: 'Sélectionner un joueur' })
    fireEvent.click(trigger)
    fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' })

    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    expect(trigger).toHaveFocus()
  })

  it('ferme au clic en dehors', () => {
    renderSwitcher([JGTM, MADINA])

    fireEvent.click(screen.getByRole('button', { name: 'Sélectionner un joueur' }))
    expect(screen.getByRole('menu')).toBeInTheDocument()

    fireEvent.mouseDown(document.body)
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('reste un simple libellé avec un seul joueur, manette et compteur compris', async () => {
    mockPresence({
      players: [{ player_slug: 'jgtm', gamertag: 'JGtm', in_game: true, title_name: 'Halo Infinite' }],
      friends_in_game: 2,
    })
    renderSwitcher([JGTM])

    // Pas de menu à ouvrir (comportement d'origine du span préservé).
    expect(screen.queryByRole('button', { name: 'Sélectionner un joueur' })).not.toBeInTheDocument()
    expect(screen.getByText('JGtm')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByTestId('in-game-jgtm')).toBeInTheDocument()
    })
    expect(screen.getByTestId('friends-in-game-badge')).toHaveAttribute('aria-label', '2 amis en jeu')
  })

  it('traduit les libellés en anglais', async () => {
    mockPresence({
      players: [{ player_slug: 'jgtm', gamertag: 'JGtm', in_game: true, title_name: 'Halo Infinite' }],
      friends_in_game: 2,
    })
    renderWithProviders(
      <PlayerSwitcher players={[JGTM, MADINA]} currentPlayer={JGTM} locale="en" onPlayerChange={vi.fn()} />,
    )

    await waitFor(() => {
      expect(screen.getByTestId('in-game-jgtm')).toHaveAttribute('aria-label', 'In game on Halo Infinite')
    })
    expect(screen.getByTestId('friends-in-game-badge')).toHaveAttribute('aria-label', '2 friends in game')
    expect(screen.getByRole('button', { name: 'Select a player' })).toBeInTheDocument()
  })

  it('comble un `players: null` du contrat sans planter', async () => {
    mockPresence({ players: null, friends_in_game: 0 })
    renderSwitcher([JGTM, MADINA])

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Sélectionner un joueur' })).toBeInTheDocument()
    })
    expect(screen.queryByTestId('in-game-jgtm')).not.toBeInTheDocument()
  })
})
