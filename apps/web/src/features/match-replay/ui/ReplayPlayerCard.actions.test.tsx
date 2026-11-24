/**
 * Tests — les deux actions de la tuile d'un joueur (rangée des équipes, forme « fiches ») : Suivre (la cible) et
 * l'œil (Masquer / Afficher), qui apparaissent au survol ou au focus de la tuile et restent visibles quand elles
 * sont actives.
 *
 * Ce qu'ils protègent :
 *  1. OPT-IN : sans `tileActions`, la tuile n'a aucun bouton de plus.
 *  2. DEUX BOUTONS PAR TUILE, accessibles : `aria-pressed`, noms « Suivre X » / « Ne plus suivre X » / « Masquer X »
 *     / « Afficher X », infobulle (titre, puis phrase) dans la langue de la page.
 *  3. LE CLIC SUR UN BOUTON N'OUVRE PAS LA FICHE : ni la souris, ni Entrée, ni Espace ; Espace reste « Lecture »
 *     et la souris ne prend pas le focus (`PRESS_GUARDS`).
 *  4. VISIBILITÉ par classes : cachés au repos, visibles au survol et au focus de la tuile, toujours visibles
 *     actifs, à 60 % sur un écran sans survol.
 *  5. LA TUILE D'UN JOUEUR MASQUÉ : nom pâle et mention « masqué ».
 *  6. ICÔNES EN LIGNE, sans couleur en dur.
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import type { PlayerTileActions } from '../model/playerTileActions'
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

function actions(over: Partial<PlayerTileActions> = {}): PlayerTileActions {
  return {
    followedXuid: null, hidden: new Set(), onFollow: vi.fn(), onToggleHidden: vi.fn(), onShowAll: vi.fn(), ...over,
  }
}

function setup(tileActions?: PlayerTileActions, onCard: (xuid: string) => void = vi.fn(), locale: 'fr' | 'en' = 'fr') {
  const view = render(
    <ReplayRowFromBoard
      doc={DOC}
      scoreboard={SCOREBOARD}
      frame={10}
      locale={locale}
      allegiance={ALLEGIANCE}
      onCard={onCard}
      tileActions={tileActions}
    />,
  )
  return { ...view, onCard }
}

describe('ReplayPlayerCard — les actions sont à la demande de la page', () => {
  it('sans `tileActions`, la tuile n’a que son bouton d’ouverture', () => {
    setup()
    expect(screen.queryByRole('button', { name: /Suivre/ })).toBeNull()
    expect(screen.queryByRole('button', { name: /Masquer/ })).toBeNull()
    expect(screen.getAllByRole('button')).toHaveLength(3)
  })

  it('avec `tileActions`, chaque tuile d’un joueur présent porte deux boutons nommés, un bot compris', () => {
    setup(actions())
    for (const nom of ['Alpha', 'Bravo', 'Marc']) {
      expect(screen.getByRole('button', { name: `Suivre ${nom} (caméra)` })).toBeTruthy()
      expect(screen.getByRole('button', { name: `Masquer ${nom}` })).toBeTruthy()
    }
  })
})

describe('ReplayPlayerCard — les gestes', () => {
  it('Suivre : un clic demande à suivre ce joueur, sans ouvrir la fiche', () => {
    const tile = actions()
    const { onCard } = setup(tile)
    fireEvent.click(screen.getByRole('button', { name: 'Suivre Alpha (caméra)' }))
    expect(tile.onFollow).toHaveBeenCalledTimes(1)
    expect(tile.onFollow).toHaveBeenCalledWith('A')
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

  it('un clic ailleurs sur la tuile ouvre toujours la fiche', () => {
    const tile = actions()
    const { onCard } = setup(tile)
    fireEvent.click(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }))
    expect(onCard).toHaveBeenCalledWith('A')
    expect(tile.onFollow).not.toHaveBeenCalled()
    expect(tile.onToggleHidden).not.toHaveBeenCalled()
  })

  it('le joueur suivi : « Ne plus suivre », enfoncé ; un clic arrête le suivi', () => {
    const tile = actions({ followedXuid: 'B' })
    setup(tile)
    const bouton = screen.getByRole('button', { name: 'Ne plus suivre Bravo (caméra)' })
    expect(bouton.getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByRole('button', { name: 'Suivre Alpha (caméra)' }).getAttribute('aria-pressed')).toBe('false')
    fireEvent.click(bouton)
    expect(tile.onFollow).toHaveBeenCalledWith(null)
  })

  it('le joueur masqué : « Afficher », enfoncé ; un clic le réaffiche', () => {
    const tile = actions({ hidden: new Set(['B']) })
    setup(tile)
    const bouton = screen.getByRole('button', { name: 'Afficher Bravo' })
    expect(bouton.getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByRole('button', { name: 'Masquer Alpha' }).getAttribute('aria-pressed')).toBe('false')
    fireEvent.click(bouton)
    expect(tile.onToggleHidden).toHaveBeenCalledWith('B')
  })

  it('plusieurs joueurs masqués à la fois : chaque tuile le dit', () => {
    setup(actions({ hidden: new Set(['A', 'bot:Marc']) }))
    expect(screen.getByRole('button', { name: 'Afficher Alpha' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Afficher Marc' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Masquer Bravo' })).toBeTruthy()
  })
})

describe('ReplayPlayerCard — clavier et souris (PRESS_GUARDS)', () => {
  it('Entrée et Espace sur un bouton d’action ne rouvrent pas la fiche', () => {
    const { onCard } = setup(actions())
    const bouton = screen.getByRole('button', { name: 'Suivre Alpha (caméra)' })
    fireEvent.keyDown(bouton, { key: 'Enter' })
    fireEvent.keyDown(bouton, { key: ' ' })
    expect(onCard).not.toHaveBeenCalled()
  })

  it('Espace sur un bouton d’action ne remonte pas au raccourci de lecture', () => {
    setup(actions())
    const global = vi.fn()
    window.addEventListener('keydown', global)
    fireEvent.keyDown(screen.getByRole('button', { name: 'Masquer Alpha' }), { key: ' ' })
    window.removeEventListener('keydown', global)
    expect(global).not.toHaveBeenCalled()
  })

  it('la souris ne prend pas le focus sur un bouton d’action', () => {
    setup(actions())
    expect(fireEvent.mouseDown(screen.getByRole('button', { name: 'Suivre Alpha (caméra)' }))).toBe(false)
    expect(fireEvent.mouseDown(screen.getByRole('button', { name: 'Masquer Alpha' }))).toBe(false)
  })

  it('les deux boutons sont dans l’ordre de tabulation, après la tuile', () => {
    setup(actions())
    const tuile = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    const suivre = screen.getByRole('button', { name: 'Suivre Alpha (caméra)' })
    const oeil = screen.getByRole('button', { name: 'Masquer Alpha' })
    expect(suivre.tabIndex).toBe(0)
    expect(oeil.tabIndex).toBe(0)
    expect(tuile.contains(suivre)).toBe(true)
    expect(tuile.compareDocumentPosition(suivre) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(suivre.compareDocumentPosition(oeil) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })
})

describe('ReplayPlayerCard — les infobulles', () => {
  it.each([
    ['fr', 'Caméra : suivre', 'La caméra cadre la carte sur Alpha.', 'Masquer', 'Retire Alpha de la carte : pion, traînée, nom.'],
    ['en', 'Camera: follow', 'The camera frames the map on Alpha.', 'Hide', 'Removes Alpha from the map: marker, trail, name.'],
  ] as const)('%s : titre puis phrase, à l’état de repos', (locale, suivre, phraseSuivre, masquer, phraseMasquer) => {
    setup(actions(), vi.fn(), locale)
    const nom = locale === 'fr' ? 'Suivre Alpha (caméra)' : 'Follow Alpha (camera)'
    const nomOeil = locale === 'fr' ? 'Masquer Alpha' : 'Hide Alpha'
    expect(screen.getByRole('button', { name: nom }).getAttribute('title')).toBe(`${suivre}\n${phraseSuivre}`)
    expect(screen.getByRole('button', { name: nomOeil }).getAttribute('title')).toBe(`${masquer}\n${phraseMasquer}`)
  })

  it.each([
    ['fr', 'Ne plus suivre Alpha (caméra)', 'Ne plus suivre\nLa carte revient en vue entière.', 'Afficher Bravo', 'Afficher\nBravo réapparaît sur la carte.'],
    ['en', 'Stop following Alpha (camera)', 'Stop following\nThe map goes back to the full view.', 'Show Bravo', 'Show\nBravo reappears on the map.'],
  ] as const)('%s : titre puis phrase, à l’état actif', (locale, nomSuivi, titreSuivi, nomMasque, titreMasque) => {
    setup(actions({ followedXuid: 'A', hidden: new Set(['B']) }), vi.fn(), locale)
    expect(screen.getByRole('button', { name: nomSuivi }).getAttribute('title')).toBe(titreSuivi)
    expect(screen.getByRole('button', { name: nomMasque }).getAttribute('title')).toBe(titreMasque)
  })
})

describe('ReplayPlayerCard — la visibilité des actions', () => {
  it('au repos : cachées, visibles au survol et au focus de la tuile, à 60 % sans survol', () => {
    setup(actions())
    for (const nom of ['Suivre Alpha (caméra)', 'Masquer Alpha']) {
      const classes = screen.getByRole('button', { name: nom }).className
      expect(classes).toContain('opacity-0')
      expect(classes).toContain('group-hover:opacity-100')
      expect(classes).toContain('group-focus-within:opacity-100')
      expect(classes).toContain('[@media(hover:none)]:opacity-60')
    }
  })

  it('une action active reste visible', () => {
    setup(actions({ followedXuid: 'A', hidden: new Set(['B']) }))
    for (const nom of ['Ne plus suivre Alpha (caméra)', 'Afficher Bravo']) {
      const classes = screen.getByRole('button', { name: nom }).className
      expect(classes).toContain('opacity-100')
      expect(classes).not.toContain('opacity-0')
      expect(classes).not.toContain('[@media(hover:none)]:opacity-60')
    }
  })

  it('la tuile porte le groupe qui commande l’apparition', () => {
    setup(actions())
    expect(screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' }).className).toMatch(/(^|\s)group(\s|$)/)
  })
})

describe('ReplayPlayerCard — la tuile d’un joueur masqué', () => {
  it('écrit la mention « masqué » et pâlit le nom, sur ce joueur seulement', () => {
    setup(actions({ hidden: new Set(['B']) }))
    const bravo = screen.getByRole('button', { name: 'Ouvrir la fiche de Bravo' })
    expect(within(bravo).getByText('masqué')).toBeTruthy()
    expect(within(bravo).getByText('Bravo').className).toContain('text-muted-foreground')
    const alpha = screen.getByRole('button', { name: 'Ouvrir la fiche de Alpha' })
    expect(within(alpha).queryByText('masqué')).toBeNull()
    expect(within(alpha).getByText('Alpha').className).not.toContain('text-muted-foreground')
  })

  it('la mention est « hidden » en anglais', () => {
    setup(actions({ hidden: new Set(['B']) }), vi.fn(), 'en')
    expect(screen.getByText('hidden')).toBeTruthy()
  })
})

describe('ReplayPlayerCard — les icônes', () => {
  it('chaque bouton porte une icône en ligne, sans couleur écrite en dur', () => {
    setup(actions({ followedXuid: 'A', hidden: new Set(['B']) }))
    for (const nom of ['Ne plus suivre Alpha (caméra)', 'Masquer Alpha', 'Afficher Bravo']) {
      const svg = screen.getByRole('button', { name: nom }).querySelector('svg')
      expect(svg, nom).not.toBeNull()
      expect(svg?.getAttribute('aria-hidden')).toBe('true')
      expect(svg?.outerHTML).not.toMatch(/#[0-9a-f]{3,8}|rgb\(|hsl\(|oklch\(/i)
    }
  })

  it('l’œil est barré quand le joueur est masqué : l’icône change', () => {
    setup(actions({ hidden: new Set(['B']) }))
    const ouvert = screen.getByRole('button', { name: 'Masquer Alpha' }).querySelector('svg')?.innerHTML
    const barre = screen.getByRole('button', { name: 'Afficher Bravo' }).querySelector('svg')?.innerHTML
    expect(ouvert).not.toBe(barre)
  })
})
