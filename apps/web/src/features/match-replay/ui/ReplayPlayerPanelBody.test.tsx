/**
 * Tests — le CORPS COMMUN de la fiche du joueur (`ReplayPlayerPanelBody`), que montent la fenêtre du Cinéma et du
 * Plein cadre et la fiche en colonne du Standard.
 *
 * Ce qu'ils protègent :
 *  1. LA TÊTE : le nom, les compteurs au format de la fiche, l'instant qui les date.
 *  2. LES TROIS ACTIONS : des boutons à bascule (`aria-pressed`) dans l'ordre de la maquette, qui
 *     rappellent la page ; la souris ne leur prend pas le focus, Espace reste « Lecture ».
 *  3. LES LISTES SONT DES TOTAUX DE FIN DE MATCH, et le disent (« sur le match »), en FR comme en EN ;
 *     aucune précision « à l'instant ».
 *  3 bis. LE BLOC « À L'INSTANT » est monté entre les actions et les listes, sur l'image lue.
 *  4. RIEN D'INVENTÉ : une valeur non lue est un tiret, une feuille absente ou entièrement non lue un
 *     bloc pointillé — jamais un zéro. Un bot sans feuille est dans ce cas ; avec une feuille, il a ses listes.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import { ReplayPlayerPanelBody, type ReplayPlayerPanelBodyProps } from './ReplayPlayerPanelBody'

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
})
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'A')

function player(board?: MatchScoreboardRow, extra: Partial<ReplayPlayer> = {}): ReplayPlayer {
  return { xuid: 'A', filmName: 'Alpha', team: 0, lives: [], board, ...extra }
}

const FULL = scoreboardRow('A', 'Alpha', 'a', {
  kills: 7, deaths: 3, assists: 2, score: 1800,
  accuracy: 42.5, shots_hit: 85, shots_fired: 200,
  damage_dealt: 3457.2, damage_taken: 2100,
  headshot_kills: 4, power_weapon_kills: 2, melee_kills: 1,
  objective: { flag_captures: 2, flag_returns: 0 } as never,
  weapon_kills: [{ weapon_id: 9, label: 'BR75', kills: 6 }, { weapon_id: 12, kills: 1 }],
})

function bodyProps(over: Partial<ReplayPlayerPanelBodyProps> = {}): ReplayPlayerPanelBodyProps {
  return {
    player: player(FULL),
    actions: { follow: false, solo: false, hide: false },
    onToggle: vi.fn(),
    doc: DOC,
    frame: 10,
    nowMs: 65_000,
    playWindow: null,
    allegiance: ALLEGIANCE,
    locale: 'fr',
    ...over,
  }
}

function setup(over: Partial<ReplayPlayerPanelBodyProps> = {}) {
  const props = bodyProps(over)
  render(<ReplayPlayerPanelBody {...props} />)
  return props
}

describe('ReplayPlayerPanelBody — la tête', () => {
  it('nomme le joueur et montre ses compteurs au format de la fiche, datés par les totaux de la feuille', () => {
    setup()
    expect(screen.getByRole('group', { name: 'Alpha' })).toBeTruthy()
    // Sans film publié pour ce joueur, la fiche retombe sur les totaux de la feuille : 7 / 3 / 2.
    expect(screen.getByTitle(/Frags \/ morts \/ assistances du match/).textContent).toBe('7/3/2')
    expect(screen.getByText('totaux du match')).toBeTruthy()
  })

  it('ce que le conteneur pose à droite de la tête (`headEnd`) y est rendu', () => {
    setup({ headEnd: <button type="button">Fermer</button> })
    expect(screen.getByRole('button', { name: 'Fermer' })).toBeTruthy()
  })

  it('la partie basse défile d’elle-même par défaut, et laisse le conteneur défiler avec `scroll="outer"`', () => {
    const { unmount } = render(<ReplayPlayerPanelBody {...bodyProps()} />)
    const basse = () => (document.querySelector('[data-replay-player-now]') as HTMLElement).parentElement as HTMLElement
    expect(basse().className).toContain('overflow-y-auto')
    unmount()
    render(<ReplayPlayerPanelBody {...bodyProps()} scroll="outer" />)
    expect(basse().className).not.toContain('overflow-y-auto')
  })
})

describe('ReplayPlayerPanelBody — les trois actions', () => {
  it('Suivre, Seul, Masquer : des boutons à bascule, dans cet ordre', () => {
    setup({ actions: { follow: true, solo: false, hide: false } })
    const group = screen.getByRole('group', { name: 'Alpha' })
    const buttons = within(group).getAllByRole('button')
    expect(buttons.map((b) => b.textContent)).toEqual(['Suivre', 'Seul', 'Masquer'])
    expect(buttons.map((b) => b.getAttribute('aria-pressed'))).toEqual(['true', 'false', 'false'])
  })

  it('un clic rappelle la page avec l’action', () => {
    const props = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Seul' }))
    fireEvent.click(screen.getByRole('button', { name: 'Masquer' }))
    fireEvent.click(screen.getByRole('button', { name: 'Suivre' }))
    expect(props.onToggle).toHaveBeenNthCalledWith(1, 'solo')
    expect(props.onToggle).toHaveBeenNthCalledWith(2, 'hide')
    expect(props.onToggle).toHaveBeenNthCalledWith(3, 'follow')
  })

  it('l’infobulle de Suivre dit que la main l’arrête', () => {
    setup()
    expect(screen.getByRole('button', { name: 'Suivre' }).getAttribute('title')).toContain("l'arrête")
  })

  it('la souris ne prend pas le focus et Espace ne remonte pas au raccourci de lecture', () => {
    setup()
    const bouton = screen.getByRole('button', { name: 'Seul' })
    // `fireEvent` rend false quand l'événement a été annulé : le focus n'est pas pris.
    expect(fireEvent.mouseDown(bouton)).toBe(false)
    const global = vi.fn()
    window.addEventListener('keydown', global)
    fireEvent.keyDown(bouton, { key: ' ' })
    window.removeEventListener('keydown', global)
    expect(global).not.toHaveBeenCalled()
  })
})

describe('ReplayPlayerPanelBody — les listes : des totaux de fin de match', () => {
  it('chaque titre dit « sur le match », et aucune précision « à l’instant » n’existe', () => {
    setup()
    for (const titre of [
      'Précision sur le match',
      'Dégâts sur le match',
      'Détail du score sur le match',
      'Objectif sur le match',
      'Éliminations par arme sur le match',
    ]) {
      expect(screen.getByRole('heading', { name: titre })).toBeTruthy()
    }
    // Le bloc « À l'instant » existe (cf. plus bas) mais n'est le titre d'aucune de ces listes.
    expect(screen.queryAllByRole('heading').filter((h) => /à l.instant/i.test(h.textContent ?? ''))).toEqual([])
  })

  it('la précision : pourcentage, touches et tirs, lus sur la feuille', () => {
    setup()
    const liste = screen.getByRole('region', { name: 'Précision sur le match' })
    const lignes = within(liste).getAllByRole('listitem').map((li) => li.textContent)
    expect(lignes).toEqual(['Précision42,5 %', 'Touches85', 'Tirs200'])
  })

  it('un zéro MESURÉ s’écrit 0 ; les dégâts sont arrondis', () => {
    setup()
    const objectif = screen.getByRole('region', { name: 'Objectif sur le match' })
    expect(within(objectif).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      'Captures de drapeau2',
      'Retours de drapeau0',
    ])
    const degats = screen.getByRole('region', { name: 'Dégâts sur le match' })
    expect(within(degats).getAllByRole('listitem')[0].textContent).toBe('Infligés3 457')
  })

  it('une valeur non lue est un tiret, pas un zéro', () => {
    setup({ player: player(scoreboardRow('A', 'Alpha', 'a', { accuracy: 40, shots_hit: undefined, shots_fired: 100 })) })
    const liste = screen.getByRole('region', { name: 'Précision sur le match' })
    expect(within(liste).getAllByRole('listitem').map((li) => li.textContent)).toEqual([
      'Précision40,0 %',
      'Touches—',
      'Tirs100',
    ])
  })

  it('une arme sans nom écrit son numéro, jamais le nom d’une autre', () => {
    setup()
    const armes = screen.getByRole('region', { name: 'Éliminations par arme sur le match' })
    expect(within(armes).getAllByRole('listitem').map((li) => li.textContent)).toEqual(['BR75×6', 'Arme n° 12×1'])
  })

  it('en anglais, les titres le disent aussi (« over the match »)', () => {
    setup({ locale: 'en' })
    expect(screen.getByRole('heading', { name: 'Accuracy over the match' })).toBeTruthy()
    expect(screen.getByRole('heading', { name: 'Score breakdown over the match' })).toBeTruthy()
  })
})

describe('ReplayPlayerPanelBody — le bloc « À l’instant »', () => {
  it('est monté entre les actions et les totaux, avec l’instant lu', () => {
    setup()
    const actions = screen.getByRole('group', { name: 'Alpha' })
    const bloc = screen.getByRole('region', { name: "À l'instant" })
    const premiereListe = screen.getByRole('region', { name: 'Précision sur le match' })
    expect(actions.compareDocumentPosition(bloc) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(bloc.compareDocumentPosition(premiereListe) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    // L'horloge du bloc est celle de la tête : 65 s de film, sans fenêtre de jeu.
    expect(within(bloc).getByText('1:05')).toBeTruthy()
  })

  it('reste là quand le joueur n’a pas de ligne de feuille : les totaux seuls tombent au bloc pointillé', () => {
    setup({ player: player(undefined) })
    expect(screen.getByRole('region', { name: "À l'instant" })).toBeTruthy()
    expect(screen.getByText('Pas de statistiques de match pour ce joueur')).toBeTruthy()
  })
})

describe('ReplayPlayerPanelBody — rien d’inventé', () => {
  it('un joueur sans ligne de feuille : un bloc pointillé, aucune liste, aucun zéro', () => {
    setup({ player: player(undefined) })
    expect(screen.getByText('Pas de statistiques de match pour ce joueur')).toBeTruthy()
    expect(screen.queryByRole('heading', { level: 4 })).toBeNull()
    expect(screen.queryByText('0')).toBeNull()
    // La tête, elle, reste : son nom et les actions sur la carte fonctionnent.
    expect(screen.getByRole('button', { name: 'Suivre' })).toBeTruthy()
  })

  it('une feuille entièrement non lue : le même bloc, pas des listes de tirets', () => {
    setup({ player: player(scoreboardRow('A', 'Alpha', 'a', { score: undefined })) })
    expect(screen.getByText('Pas de statistiques de match pour ce joueur')).toBeTruthy()
  })

  it('un bot (sans xuid, clé de roster synthétique) avec une feuille a ses listes ; sans feuille, le bloc', () => {
    const bot = player(scoreboardRow('bid(2.0)', 'Marc', 'b', { accuracy: 30, shots_hit: 3, shots_fired: 10 }), {
      xuid: 'bot:Marc', bot: true, filmName: 'Marc',
    })
    setup({ player: bot })
    expect(screen.getByRole('group', { name: 'Marc' })).toBeTruthy()
    expect(screen.getByRole('region', { name: 'Précision sur le match' })).toBeTruthy()
  })

  it('un bot sans ligne de feuille : le bloc pointillé', () => {
    setup({ player: player(undefined, { xuid: 'bot:Marc', bot: true, filmName: 'Marc' }) })
    expect(screen.getByText('Pas de statistiques de match pour ce joueur')).toBeTruthy()
  })
})
