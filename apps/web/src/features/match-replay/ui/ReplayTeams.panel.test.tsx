/**
 * Tests — la fiche de joueur est un bouton qui ouvre le panneau (lot 4 du plan « maquettes du
 * rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. LA PAGE DEMANDE LA FICHE-BOUTON : sans `panel`, aucune fiche n'est cliquable (la fixation DOM
 *     `ReplayTeams.dom4v4` tient nœud pour nœud) ; avec, chaque fiche d'un joueur présent l'est.
 *  2. ACCESSIBLE : `role="button"`, nom accessible, `aria-pressed` sur la fiche dont le panneau est
 *     ouvert, focalisable ; Entrée et Espace activent, Espace ne remonte pas au raccourci de lecture,
 *     la souris ne prend pas le focus.
 *  3. LE CLIC RAPPELLE LA PAGE avec le joueur de la fiche — un bot par sa clé de roster.
 *  4. L'INFOBULLE DE LA FICHE RESTE (`title`), et la place libre n'est jamais cliquable.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { ReplayTeams } from './ReplayTeams'
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

function setup(panel?: { xuid: string | null; onSelect: (xuid: string) => void }) {
  return render(
    <ReplayTeams doc={DOC} scoreboard={SCOREBOARD} frame={10} locale="fr" allegiance={ALLEGIANCE} panel={panel} />,
  )
}

describe('ReplayTeams — fiches inertes sans panneau', () => {
  it('sans `panel`, aucune fiche n’est un bouton', () => {
    setup()
    expect(screen.queryAllByRole('button')).toEqual([])
  })
})

describe('ReplayTeams — fiches-boutons', () => {
  it('chaque fiche d’un joueur présent est un bouton nommé, la choisie est enfoncée', () => {
    setup({ xuid: 'B', onSelect: vi.fn() })
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
    const onSelect = vi.fn()
    setup({ xuid: null, onSelect })
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }))
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir la fiche de Marc' }))
    expect(onSelect).toHaveBeenNthCalledWith(1, 'A')
    expect(onSelect).toHaveBeenNthCalledWith(2, 'bot:Marc')
  })

  it('Entrée et Espace activent ; la fiche est focalisable au clavier', () => {
    const onSelect = vi.fn()
    setup({ xuid: null, onSelect })
    const fiche = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    expect(fiche.tabIndex).toBe(0)
    fireEvent.keyDown(fiche, { key: 'Enter' })
    fireEvent.keyDown(fiche, { key: ' ' })
    expect(onSelect).toHaveBeenCalledTimes(2)
    fireEvent.keyDown(fiche, { key: 'a' })
    expect(onSelect).toHaveBeenCalledTimes(2)
  })

  it('Espace ne remonte pas au raccourci de lecture, et la souris ne prend pas le focus', () => {
    setup({ xuid: null, onSelect: vi.fn() })
    const fiche = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    expect(fireEvent.mouseDown(fiche)).toBe(false)
    const global = vi.fn()
    window.addEventListener('keydown', global)
    fireEvent.keyDown(fiche, { key: ' ' })
    window.removeEventListener('keydown', global)
    expect(global).not.toHaveBeenCalled()
  })

  it('les infobulles (title) de la colonne sont celles d’avant : la fiche-bouton n’en retire aucune', () => {
    const titles = (c: HTMLElement) => Array.from(c.querySelectorAll('[title]')).map((el) => el.getAttribute('title'))
    const inerte = setup()
    const avant = titles(inerte.container)
    inerte.unmount()
    const bouton = setup({ xuid: null, onSelect: vi.fn() })
    expect(avant.length).toBeGreaterThan(0)
    expect(titles(bouton.container)).toEqual(avant)
  })

  it('la fiche choisie porte un anneau, qui ne change aucune cote (ombre)', () => {
    setup({ xuid: 'A', onSelect: vi.fn() })
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }).className).toContain('ring-2')
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' }).className).not.toContain('ring-2 ring-ring')
  })
})
