/**
 * Tests — la fiche de joueur est un bouton qui ouvre le panneau (rangée des équipes, forme « fiches »).
 *
 * Ce qu'ils protègent :
 *  1. LA PAGE DEMANDE LA FICHE-BOUTON : sans `onCard`, aucune fiche n'est cliquable (nœud pour nœud) ; avec,
 *     chaque fiche d'un joueur présent l'est.
 *  2. ACCESSIBLE : `role="button"`, nom accessible, `aria-pressed` sur la fiche dont le panneau est ouvert,
 *     focalisable ; Entrée et Espace activent, Espace ne remonte pas au raccourci de lecture, la souris ne
 *     prend pas le focus.
 *  3. LE CLIC RAPPELLE LA PAGE avec le joueur de la fiche — un bot par sa clé de roster.
 *  4. L'INFOBULLE DE LA FICHE RESTE (`title`).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { ReplayRowFromBoard } from '../test/replayRowFromBoard'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

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
const ALLEGIANCE = filmAllegianceOf(DOC, SCOREBOARD, 'A')

function setup(onCard?: (xuid: string) => void, selectedXuid: string | null = null) {
  return render(
    <ReplayRowFromBoard
      doc={DOC}
      scoreboard={SCOREBOARD}
      frame={10}
      locale="fr"
      allegiance={ALLEGIANCE}
      onCard={onCard}
      selectedXuid={selectedXuid}
    />,
  )
}

describe('ReplayPlayerCard — fiches inertes sans geste de clic', () => {
  it('sans `onCard`, aucune fiche n’est un bouton', () => {
    setup()
    expect(screen.queryAllByRole('button')).toEqual([])
  })
})

describe('ReplayPlayerCard — fiches-boutons', () => {
  it('chaque fiche d’un joueur présent est un bouton nommé, la choisie est enfoncée', () => {
    setup(vi.fn(), 'B')
    const boutons = screen.getAllByRole('button')
    expect(boutons.map((b) => b.getAttribute('aria-label')).sort()).toEqual([
      'Ouvrir la fiche de Alpha',
      'Ouvrir la fiche de Bravo',
      'Ouvrir la fiche de Marc',
    ])
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' }).getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }).getAttribute('aria-pressed')).toBe('false')
  })

  it('un clic rappelle la page avec le joueur — un bot par sa clé de roster', () => {
    const onCard = vi.fn()
    setup(onCard)
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }))
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir la fiche de Marc' }))
    expect(onCard).toHaveBeenNthCalledWith(1, 'A')
    expect(onCard).toHaveBeenNthCalledWith(2, 'bot:Marc')
  })

  it('Entrée et Espace activent ; la fiche est focalisable au clavier', () => {
    const onCard = vi.fn()
    setup(onCard)
    const fiche = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    expect(fiche.tabIndex).toBe(0)
    fireEvent.keyDown(fiche, { key: 'Enter' })
    fireEvent.keyDown(fiche, { key: ' ' })
    expect(onCard).toHaveBeenCalledTimes(2)
    fireEvent.keyDown(fiche, { key: 'a' })
    expect(onCard).toHaveBeenCalledTimes(2)
  })

  it('Espace ne remonte pas au raccourci de lecture, et la souris ne prend pas le focus', () => {
    setup(vi.fn())
    const fiche = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    expect(fireEvent.mouseDown(fiche)).toBe(false)
    const global = vi.fn()
    window.addEventListener('keydown', global)
    fireEvent.keyDown(fiche, { key: ' ' })
    window.removeEventListener('keydown', global)
    expect(global).not.toHaveBeenCalled()
  })

  it('les infobulles (title) des fiches sont celles d’avant : la fiche-bouton n’en retire aucune', () => {
    const titles = (c: HTMLElement) => Array.from(c.querySelectorAll('[title]')).map((el) => el.getAttribute('title'))
    const inerte = setup()
    const avant = titles(inerte.container)
    inerte.unmount()
    const bouton = setup(vi.fn())
    expect(avant.length).toBeGreaterThan(0)
    expect(titles(bouton.container)).toEqual(avant)
  })

  it('la fiche choisie porte un anneau, qui ne change aucune cote (ombre)', () => {
    setup(vi.fn(), 'A')
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }).className).toContain('ring-2')
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' }).className).not.toContain('ring-2 ring-ring')
  })
})
