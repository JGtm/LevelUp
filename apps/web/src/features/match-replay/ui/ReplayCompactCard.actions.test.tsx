/**
 * Tests — les deux actions de la fiche compacte d'une Grande équipe (rangée du Cinéma : forme « stacked » ;
 * colonne du Standard : forme « line »), les mêmes que celles de la tuile (D-6, E.8) : Suivre et l'œil,
 * `aria-pressed`, noms accessibles, infobulle (titre puis phrase), clic sans ouverture de fiche, visibles au survol
 * ou au focus de la fiche et toujours visibles actifs (60 % sans survol), nom pâle, œil barré quand le joueur est
 * masqué. Elles viennent de `ReplayTileActionButtons` : aucun second composant.
 */
import { useMemo } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { filmAllegianceOf, NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayRosterColumn } from './ReplayRosterColumn'
import { ReplayTeamsRow } from './ReplayTeamsRow'
import type { PlayerTileActions } from '../model/playerTileActions'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** Six places par camp : au-delà de `PLACES_PAR_CAMP_MAX_RANGEE`, les fiches sont compactes. */
const NOMS = ['Alpha', 'Bravo', 'Charlie', 'Delta', 'Echo', 'Foxtrot', 'Golf', 'Hotel', 'India', 'Juliett', 'Kilo', 'Lima']
const XUIDS = NOMS.map((_, i) => `X${i}`)
const DOC = testReplayDoc({
  frameCount: 200,
  roster: NOMS.map((name, i) => ({ xuid: XUIDS[i], filmIndex: i, seat: i, name, team: i < 6 ? 0 : 1 })),
  tracks: NOMS.map((_, i) => ({
    slot: 512 + i, team: i < 6 ? 0 : 1, xuid: XUIDS[i], startFrame: 0, endFrame: 100, points: [{ t: 0, x: i, y: i, sh: 0.4, hp: 0.7 }],
  })) as never,
})
const SCOREBOARD: MatchScoreboardRow[] = NOMS.map((n, i) => scoreboardRow(XUIDS[i], n, i < 6 ? 'a' : 'b'))

function actions(over: Partial<PlayerTileActions> = {}): PlayerTileActions {
  return {
    followedXuid: null, hidden: new Set(), onFollow: vi.fn(), onToggleHidden: vi.fn(), onShowAll: vi.fn(), ...over,
  }
}

function Rangee({ tileActions, onCard, locale }: { tileActions?: PlayerTileActions; onCard: (x: string) => void; locale: 'fr' | 'en' }) {
  const players = useMemo(() => buildPlayers(DOC, SCOREBOARD), [])
  return (
    <ReplayTeamsRow
      doc={DOC} players={players} frame={10} locale={locale} allegiance={filmAllegianceOf(DOC, SCOREBOARD, XUIDS[0])}
      tracked={null} onCard={onCard} tileActions={tileActions} score={null} grandeEquipe
    />
  )
}

function Colonne({ tileActions, onCard, locale }: { tileActions?: PlayerTileActions; onCard: (x: string) => void; locale: 'fr' | 'en' }) {
  const players = useMemo(() => buildPlayers(DOC, SCOREBOARD), [])
  return (
    <ReplayRosterColumn
      doc={DOC} players={players} frame={10} locale={locale} allegiance={NO_ALLEGIANCE}
      tracked={null} onCard={onCard} tileActions={tileActions}
    />
  )
}

describe.each([
  ['la rangée (forme « stacked »)', Rangee],
  ['la colonne (forme « line »)', Colonne],
] as const)('fiche compacte — %s', (_nom, Monter) => {
  function setup(tileActions?: PlayerTileActions, locale: 'fr' | 'en' = 'fr') {
    const onCard = vi.fn()
    const vue = render(<Monter tileActions={tileActions} onCard={onCard} locale={locale} />)
    return { ...vue, onCard }
  }
  const fiche = (nom: string) => screen.getByRole('button', { name: `Ouvrir la fiche de ${nom}` })

  it('sans `tileActions`, aucune action de plus', () => {
    setup()
    expect(screen.queryByRole('button', { name: /Suivre/ })).toBeNull()
    expect(screen.queryByRole('button', { name: /Masquer/ })).toBeNull()
  })

  it('chaque fiche porte deux boutons nommés', () => {
    setup(actions())
    for (const nom of ['Alpha', 'Lima']) {
      expect(within(fiche(nom)).getByRole('button', { name: `Suivre ${nom}` })).toBeTruthy()
      expect(within(fiche(nom)).getByRole('button', { name: `Masquer ${nom}` })).toBeTruthy()
    }
  })

  it('un clic sur un bouton agit sans ouvrir la fiche ; un clic ailleurs l’ouvre', () => {
    const tile = actions()
    const { onCard } = setup(tile)
    fireEvent.click(screen.getByRole('button', { name: 'Suivre Bravo' }))
    fireEvent.click(screen.getByRole('button', { name: 'Masquer Bravo' }))
    expect(tile.onFollow).toHaveBeenCalledWith('X1')
    expect(tile.onToggleHidden).toHaveBeenCalledWith('X1')
    expect(onCard).not.toHaveBeenCalled()
    fireEvent.click(fiche('Bravo'))
    expect(onCard).toHaveBeenCalledWith('X1')
  })

  it('Entrée et Espace sur un bouton ne rouvrent pas la fiche ; la souris ne prend pas le focus', () => {
    const { onCard } = setup(actions())
    const bouton = screen.getByRole('button', { name: 'Suivre Alpha' })
    fireEvent.keyDown(bouton, { key: 'Enter' })
    fireEvent.keyDown(bouton, { key: ' ' })
    expect(onCard).not.toHaveBeenCalled()
    expect(fireEvent.mouseDown(bouton)).toBe(false)
  })

  it('états actifs : suivi « Ne plus suivre », masqué « Afficher », enfoncés ; clic = bascule', () => {
    const tile = actions({ followedXuid: 'X0', hidden: new Set(['X1']) })
    setup(tile)
    const suivi = screen.getByRole('button', { name: 'Ne plus suivre Alpha' })
    const masque = screen.getByRole('button', { name: 'Afficher Bravo' })
    expect(suivi.getAttribute('aria-pressed')).toBe('true')
    expect(masque.getAttribute('aria-pressed')).toBe('true')
    fireEvent.click(suivi)
    expect(tile.onFollow).toHaveBeenCalledWith(null)
  })

  it.each([
    ['fr', 'Suivre Alpha', 'Suivre\nLa caméra cadre la carte sur Alpha.', 'Masquer Alpha', 'Masquer\nRetire Alpha de la carte : pion, traînée, nom.'],
    ['en', 'Follow Alpha', 'Follow\nThe camera frames the map on Alpha.', 'Hide Alpha', 'Hide\nRemoves Alpha from the map: marker, trail, name.'],
  ] as const)('%s : infobulles titre puis phrase', (locale, nomSuivre, titreSuivre, nomOeil, titreOeil) => {
    setup(actions(), locale)
    expect(screen.getByRole('button', { name: nomSuivre }).getAttribute('title')).toBe(titreSuivre)
    expect(screen.getByRole('button', { name: nomOeil }).getAttribute('title')).toBe(titreOeil)
  })

  it('visibilité : cachés au repos (survol, focus, 60 % sans survol), actifs toujours visibles ; la fiche est le groupe', () => {
    setup(actions({ followedXuid: 'X0', hidden: new Set(['X1']) }))
    const repos = screen.getByRole('button', { name: 'Masquer Charlie' }).className
    expect(repos).toContain('opacity-0')
    expect(repos).toContain('group-hover:opacity-100')
    expect(repos).toContain('group-focus-within:opacity-100')
    expect(repos).toContain('[@media(hover:none)]:opacity-60')
    for (const nom of ['Ne plus suivre Alpha', 'Afficher Bravo']) {
      const actif = screen.getByRole('button', { name: nom }).className
      expect(actif).toContain('opacity-100')
      expect(actif).not.toContain('opacity-0')
    }
    expect(fiche('Alpha').className).toMatch(/(^|\s)group(\s|$)/)
  })

  it('la fiche d’un joueur masqué a le nom pâle (sur ce joueur seulement) et l’œil barré', () => {
    setup(actions({ hidden: new Set(['X1']) }))
    expect(within(fiche('Bravo')).getByText('Bravo').className).toContain('text-muted-foreground')
    expect(within(fiche('Alpha')).getByText('Alpha').className).not.toContain('text-muted-foreground')
    const barre = screen.getByRole('button', { name: 'Afficher Bravo' }).querySelector('svg')?.innerHTML
    const ouvert = screen.getByRole('button', { name: 'Masquer Alpha' }).querySelector('svg')?.innerHTML
    expect(barre).not.toBe(ouvert)
  })

  it('les boutons sont ceux des tuiles : 16 px, icônes en ligne sans couleur en dur', () => {
    setup(actions())
    const bouton = screen.getByRole('button', { name: 'Suivre Alpha' })
    expect(bouton.className).toContain('h-4')
    expect(bouton.className).toContain('w-4')
    expect(bouton.querySelector('svg')?.outerHTML).not.toMatch(/#[0-9a-f]{3,8}|rgb\(|hsl\(|oklch\(/i)
    expect(bouton.closest('[data-replay-tile-actions]')).not.toBeNull()
  })
})
