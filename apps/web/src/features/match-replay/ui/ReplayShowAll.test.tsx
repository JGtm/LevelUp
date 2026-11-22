/**
 * Tests — « Tout afficher », en tête de la colonne des équipes (cadre Standard, colonne du Cinéma) et de la
 * rangée des équipes (Cinéma, Plein cadre) : visible seulement quand au moins un joueur est masqué, il rappelle
 * la page une fois et ne touche à rien d'autre.
 */
import { useMemo } from 'react'
import type { ComponentProps } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { filmAllegianceOf, NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayRosterColumn } from './ReplayRosterColumn'
import type { PlayerTileActions } from '../model/playerTileActions'
import { ReplayRowFromBoard } from '../test/replayRowFromBoard'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

const DOC = testReplayDoc({
  frameCount: 200,
  roster: [
    { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 },
    { xuid: 'B', filmIndex: 1, seat: 1, name: 'Bravo', team: 1 },
  ],
  tracks: [
    { slot: 512, team: 0, xuid: 'A', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 1, y: 1 }] },
    { slot: 513, team: 1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 2, y: 2 }] },
  ] as never,
})
const SCOREBOARD: MatchScoreboardRow[] = [scoreboardRow('A', 'Alpha', 'a'), scoreboardRow('B', 'Bravo', 'b')]

function actions(hidden: string[] = [], over: Partial<PlayerTileActions> = {}): PlayerTileActions {
  return {
    followedXuid: null, hidden: new Set(hidden), onFollow: vi.fn(), onToggleHidden: vi.fn(), onShowAll: vi.fn(), ...over,
  }
}

type ColumnProps = Omit<ComponentProps<typeof ReplayRosterColumn>, 'players' | 'tracked' | 'allegiance'>

function Colonne(props: ColumnProps) {
  const players = useMemo(() => buildPlayers(props.doc, SCOREBOARD), [props.doc])
  return <ReplayRosterColumn {...props} players={players} tracked={null} allegiance={NO_ALLEGIANCE} />
}

function colonne(tileActions?: PlayerTileActions, locale: 'fr' | 'en' = 'fr') {
  return render(<Colonne doc={DOC} frame={10} locale={locale} onCard={vi.fn()} tileActions={tileActions} />)
}

function rangee(tileActions?: PlayerTileActions, locale: 'fr' | 'en' = 'fr') {
  return render(
    <ReplayRowFromBoard
      doc={DOC}
      scoreboard={SCOREBOARD}
      frame={10}
      locale={locale}
      allegiance={filmAllegianceOf(DOC, SCOREBOARD, 'A')}
      onCard={vi.fn()}
      tileActions={tileActions}
    />,
  )
}

describe.each([
  ['la colonne des équipes', colonne],
  ['la rangée des équipes', rangee],
] as const)('« Tout afficher » — %s', (_nom, monter) => {
  it('absent tant qu’aucun joueur n’est masqué', () => {
    monter(actions())
    expect(screen.queryByRole('button', { name: 'Tout afficher' })).toBeNull()
  })

  it('absent sans actions de tuile (page sans panneau)', () => {
    monter(undefined)
    expect(screen.queryByRole('button', { name: 'Tout afficher' })).toBeNull()
  })

  it('visible dès qu’un joueur est masqué, et un clic rappelle la page une fois', () => {
    const tile = actions(['B'])
    monter(tile)
    const bouton = screen.getByRole('button', { name: 'Tout afficher' })
    fireEvent.click(bouton)
    expect(tile.onShowAll).toHaveBeenCalledTimes(1)
    expect(tile.onToggleHidden).not.toHaveBeenCalled()
  })

  it('en anglais : « Show all »', () => {
    monter(actions(['A', 'B']), 'en')
    expect(screen.getByRole('button', { name: 'Show all' })).toBeTruthy()
  })

  it('Espace et la souris : même contrat que les boutons du rejeu', () => {
    monter(actions(['B']))
    const bouton = screen.getByRole('button', { name: 'Tout afficher' })
    expect(fireEvent.mouseDown(bouton)).toBe(false)
    const global = vi.fn()
    window.addEventListener('keydown', global)
    fireEvent.keyDown(bouton, { key: ' ' })
    window.removeEventListener('keydown', global)
    expect(global).not.toHaveBeenCalled()
  })
})

/** Le flux de la colonne, jusqu'aux lignes (leur contenu est exclu : l'œil d'un joueur masqué y change) : chemin et classes, hors bloc de hauteur nulle de « Tout afficher ». */
function flux(racine: HTMLElement): string[] {
  const out: string[] = []
  const marche = (el: Element, chemin: string, profondeur: number) => {
    if (profondeur > 3) return
    ;[...el.children].forEach((enfant, i) => {
      if (enfant.hasAttribute('data-replay-show-all-slot')) return
      const nom = `${chemin}/${enfant.tagName.toLowerCase()}[${i - [...el.children].slice(0, i).filter((c) => c.hasAttribute('data-replay-show-all-slot')).length}]`
      out.push(`${nom} ${enfant.getAttribute('class') ?? ''}`)
      marche(enfant, nom, profondeur + 1)
    })
  }
  marche(racine, '', 1)
  return out
}

describe('« Tout afficher » — la colonne ne bouge pas quand il apparaît ou disparaît', () => {
  const racineDe = (c: HTMLElement) => c.querySelector('[data-replay-roster-column]') as HTMLElement

  it('aucune ligne ne change de place dans le flux : même structure, mêmes classes, avec ou sans joueur masqué', () => {
    const sans = colonne(actions())
    const fluxSans = flux(racineDe(sans.container))
    sans.unmount()
    const avec = colonne(actions(['B']))
    expect(screen.getByRole('button', { name: 'Tout afficher' })).toBeTruthy()
    expect(flux(racineDe(avec.container))).toEqual(fluxSans)
  })

  it('le bouton est hors du flux : bloc de hauteur nulle, bouton en position absolue, collé à l’en-tête du premier camp', () => {
    const vue = colonne(actions(['B']))
    const racine = racineDe(vue.container)
    const bouton = screen.getByRole('button', { name: 'Tout afficher' })
    const emplacement = bouton.parentElement as HTMLElement
    expect(emplacement.hasAttribute('data-replay-show-all-slot')).toBe(true)
    expect(emplacement.className).toContain('h-0')
    expect(bouton.className).toContain('absolute')
    // Aucun bloc en flux n'est ajouté à la racine : ses enfants sont les camps, rien d'autre.
    expect([...racine.children].every((c) => c.tagName === 'SECTION')).toBe(true)
    const premier = racine.querySelector('section') as HTMLElement
    expect(premier.contains(bouton)).toBe(true)
    expect(emplacement.nextElementSibling?.tagName).toBe('H3')
  })

  it('sans joueur masqué, rien n’est rendu : ni bouton focalisable, ni nom annoncé, ni emplacement', () => {
    const vue = colonne(actions())
    expect(vue.container.querySelector('[data-replay-show-all]')).toBeNull()
    expect(vue.container.querySelector('[data-replay-show-all-slot]')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Tout afficher' })).toBeNull()
    expect(screen.queryByText('Tout afficher')).toBeNull()
  })
})
