/**
 * Tests — les deux actions de la ligne d'un joueur dans la colonne des équipes : les mêmes que celles de la tuile
 * (Suivre, œil), au même contrat — boutons nommés à `aria-pressed`, infobulle, clic qui n'ouvre pas la fiche,
 * visibles au survol ou au focus de la ligne et toujours visibles actifs, nom pâle et mention « masqué ».
 */
import { useMemo } from 'react'
import type { ComponentProps } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayRosterColumn } from './ReplayRosterColumn'
import type { PlayerTileActions } from '../model/playerTileActions'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

type Props = Omit<ComponentProps<typeof ReplayRosterColumn>, 'players' | 'tracked' | 'allegiance'> & {
  scoreboard: MatchScoreboardRow[]
}

function Colonne({ scoreboard, ...rest }: Props) {
  const players = useMemo(() => buildPlayers(rest.doc, scoreboard), [rest.doc, scoreboard])
  return <ReplayRosterColumn {...rest} players={players} tracked={null} allegiance={NO_ALLEGIANCE} />
}

const DOC = testReplayDoc({
  frameCount: 200,
  roster: [
    { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 },
    { xuid: 'B', filmIndex: 1, seat: 1, name: 'Bravo', team: 1 },
    { xuid: '', bot: true, filmIndex: 2, seat: 2, name: 'Marc', team: 1 },
  ],
  tracks: [
    { slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 1, y: 1 }] },
    { slot: 513, team: 1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 2, y: 2 }] },
    { slot: 514, team: 1, xuid: '', bot: 'Marc', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 3, y: 3 }] },
  ] as never,
})
const SCOREBOARD = [scoreboardRow('A', 'Alpha', 'a'), scoreboardRow('B', 'Bravo', 'b')]

function actions(over: Partial<PlayerTileActions> = {}): PlayerTileActions {
  return {
    followedXuid: null, hidden: new Set(), onFollow: vi.fn(), onToggleHidden: vi.fn(), onShowAll: vi.fn(), ...over,
  }
}

function setup(tileActions?: PlayerTileActions, onCard: (xuid: string) => void = vi.fn(), locale: 'fr' | 'en' = 'fr') {
  const view = render(
    <Colonne doc={DOC} scoreboard={SCOREBOARD} frame={10} locale={locale} onCard={onCard} tileActions={tileActions} />,
  )
  return { ...view, onCard }
}

const ligne = (nom: string) => screen.getByRole('button', { name: `Ouvrir la fiche de ${nom}` })

describe('ReplayRosterRow — les actions sont à la demande de la page', () => {
  it('sans `tileActions`, la ligne n’a que son bouton d’ouverture', () => {
    setup()
    expect(screen.queryByRole('button', { name: /Suivre/ })).toBeNull()
    expect(screen.queryByRole('button', { name: /Masquer/ })).toBeNull()
    expect(screen.getAllByRole('button')).toHaveLength(3)
  })

  it('avec `tileActions`, chaque ligne porte deux boutons nommés, un bot compris', () => {
    setup(actions())
    for (const nom of ['Alpha', 'Bravo', 'Marc']) {
      expect(within(ligne(nom)).getByRole('button', { name: `Suivre ${nom} (caméra)` })).toBeTruthy()
      expect(within(ligne(nom)).getByRole('button', { name: `Masquer ${nom}` })).toBeTruthy()
    }
  })
})

describe('ReplayRosterRow — les gestes', () => {
  it('Suivre : un clic demande à suivre ce joueur, sans ouvrir la fiche', () => {
    const tile = actions()
    const { onCard } = setup(tile)
    fireEvent.click(screen.getByRole('button', { name: 'Suivre Bravo (caméra)' }))
    expect(tile.onFollow).toHaveBeenCalledTimes(1)
    expect(tile.onFollow).toHaveBeenCalledWith('B')
    expect(onCard).not.toHaveBeenCalled()
  })

  it('l’œil : un clic masque ce joueur — un bot par sa clé de roster —, sans ouvrir la fiche', () => {
    const tile = actions()
    const { onCard } = setup(tile)
    fireEvent.click(screen.getByRole('button', { name: 'Masquer Alpha' }))
    fireEvent.click(screen.getByRole('button', { name: 'Masquer Marc' }))
    expect(tile.onToggleHidden).toHaveBeenNthCalledWith(1, 'A')
    expect(tile.onToggleHidden).toHaveBeenNthCalledWith(2, 'bot:Marc')
    expect(onCard).not.toHaveBeenCalled()
  })

  it('un clic ailleurs sur la ligne ouvre toujours la fiche', () => {
    const tile = actions()
    const { onCard } = setup(tile)
    fireEvent.click(ligne('Bravo'))
    expect(onCard).toHaveBeenCalledWith('B')
    expect(tile.onFollow).not.toHaveBeenCalled()
    expect(tile.onToggleHidden).not.toHaveBeenCalled()
  })

  it('le joueur suivi : « Ne plus suivre », enfoncé ; un clic arrête le suivi', () => {
    const tile = actions({ followedXuid: 'A' })
    setup(tile)
    const bouton = screen.getByRole('button', { name: 'Ne plus suivre Alpha (caméra)' })
    expect(bouton.getAttribute('aria-pressed')).toBe('true')
    fireEvent.click(bouton)
    expect(tile.onFollow).toHaveBeenCalledWith(null)
  })

  it('le joueur masqué : « Afficher », enfoncé ; un clic le réaffiche', () => {
    const tile = actions({ hidden: new Set(['bot:Marc']) })
    setup(tile)
    const bouton = screen.getByRole('button', { name: 'Afficher Marc' })
    expect(bouton.getAttribute('aria-pressed')).toBe('true')
    fireEvent.click(bouton)
    expect(tile.onToggleHidden).toHaveBeenCalledWith('bot:Marc')
  })

  it('Entrée, Espace et la souris sur un bouton d’action ne rouvrent pas la fiche et ne prennent pas le focus', () => {
    const { onCard } = setup(actions())
    const bouton = screen.getByRole('button', { name: 'Suivre Alpha (caméra)' })
    fireEvent.keyDown(bouton, { key: 'Enter' })
    fireEvent.keyDown(bouton, { key: ' ' })
    expect(onCard).not.toHaveBeenCalled()
    expect(fireEvent.mouseDown(bouton)).toBe(false)
  })
})

describe('ReplayRosterRow — infobulles, visibilité, ligne masquée', () => {
  it.each([
    ['fr', 'Suivre Alpha (caméra)', 'Caméra : suivre\nLa caméra cadre la carte sur Alpha.', 'Masquer Alpha', 'Masquer\nRetire Alpha de la carte : pion, traînée, nom.'],
    ['en', 'Follow Alpha (camera)', 'Camera: follow\nThe camera frames the map on Alpha.', 'Hide Alpha', 'Hide\nRemoves Alpha from the map: marker, trail, name.'],
  ] as const)('%s : titre puis phrase', (locale, nomSuivre, titreSuivre, nomOeil, titreOeil) => {
    setup(actions(), vi.fn(), locale)
    expect(screen.getByRole('button', { name: nomSuivre }).getAttribute('title')).toBe(titreSuivre)
    expect(screen.getByRole('button', { name: nomOeil }).getAttribute('title')).toBe(titreOeil)
  })

  it('au repos cachées (visibles au survol, au focus, à 60 % sans survol) ; actives, toujours visibles', () => {
    setup(actions({ followedXuid: 'A', hidden: new Set(['B']) }))
    const repos = screen.getByRole('button', { name: 'Masquer Alpha' }).className
    expect(repos).toContain('opacity-0')
    expect(repos).toContain('group-hover:opacity-100')
    expect(repos).toContain('group-focus-within:opacity-100')
    expect(repos).toContain('[@media(hover:none)]:opacity-60')
    for (const nom of ['Ne plus suivre Alpha (caméra)', 'Afficher Bravo']) {
      const actif = screen.getByRole('button', { name: nom }).className
      expect(actif).toContain('opacity-100')
      expect(actif).not.toContain('opacity-0')
    }
    expect(ligne('Alpha').className).toMatch(/(^|\s)group(\s|$)/)
  })

  it('la ligne d’un joueur masqué écrit « masqué » et pâlit le nom, sur ce joueur seulement', () => {
    setup(actions({ hidden: new Set(['B']) }))
    expect(within(ligne('Bravo')).getByText('masqué')).toBeTruthy()
    expect(within(ligne('Alpha')).queryByText('masqué')).toBeNull()
    expect(within(ligne('Bravo')).getByText('Bravo', { exact: false }).className).toContain('text-muted-foreground')
  })

  it('les icônes sont en ligne, sans couleur écrite en dur', () => {
    setup(actions({ hidden: new Set(['B']) }))
    for (const nom of ['Suivre Alpha (caméra)', 'Afficher Bravo']) {
      const svg = screen.getByRole('button', { name: nom }).querySelector('svg')
      expect(svg, nom).not.toBeNull()
      expect(svg?.outerHTML).not.toMatch(/#[0-9a-f]{3,8}|rgb\(|hsl\(|oklch\(/i)
    }
  })
})
