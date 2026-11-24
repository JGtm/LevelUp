/**
 * Tests — le CORPS COMMUN de la fiche du joueur (`ReplayPlayerPanelBody`), que montent la fenêtre du Cinéma et du
 * Plein cadre et la fiche en colonne du Standard.
 *
 * Ce qu'ils protègent :
 *  1. L'IDENTITÉ : pastille de camp, nom, camp, frags / morts / assistances ; aucune bordure colorée.
 *  2. LES TROIS ACTIONS : des boutons à bascule (`aria-pressed`) dans l'ordre de la maquette, qui rappellent la page ;
 *     la souris ne leur prend pas le focus, Espace reste « Lecture ».
 *  3. L'ORDRE DES SECTIONS, de « À l'instant » à « Gestes », sur un joueur dont tout est lu ; la fiche SUIT l'image lue.
 *  4. RIEN D'INVENTÉ : une valeur absente n'a pas de ligne, une section sans valeur n'existe pas, aucun texte de
 *     remplacement — lecture partielle, bot, joueur mort, sans feuille, début et fin de match.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { Antagonists, AntagonistsAt } from '@/lib/antagonists/antagonists'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import {
  INSTANT_ICONS, INSTANT_LAST_FRAME, instantDoc, instantFeed, instantPlayer, instantPlayers, instantSheet,
} from '../test/instantFixture'
import { createReplayAntagonists } from '../model/replayAntagonists'
import { ReplayPlayerPanelBody, type ReplayPlayerPanelBodyProps } from './ReplayPlayerPanelBody'

const DOC = instantDoc()
const FEED = instantFeed()
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'A')

/** Les sections de la fiche, dans l'ordre où elles sont écrites. */
const SECTIONS_COMPLETES = [
  "À l'instant",
  'Antagonistes',
  'Depuis le début du match',
  'Éliminations par arme',
  'Tirs et dégâts',
  'Grenades lancées',
  'Ramassages',
  'Équipement',
  'Médailles',
  'Totaux du match',
  'Gestes',
]

function bodyProps(over: Partial<ReplayPlayerPanelBodyProps> = {}, sheet = instantSheet()): ReplayPlayerPanelBodyProps {
  const players = instantPlayers(DOC, sheet)
  return {
    player: instantPlayer(players, 'A'),
    solo: false,
    onSolo: vi.fn(),
    doc: DOC,
    frame: INSTANT_LAST_FRAME,
    nowMs: INSTANT_LAST_FRAME * 100,
    playWindow: null,
    allegiance: ALLEGIANCE,
    locale: 'fr',
    players,
    feed: FEED,
    weaponIcons: INSTANT_ICONS,
    antagonists: createReplayAntagonists([], []),
    ...over,
  }
}

function setup(over: Partial<ReplayPlayerPanelBodyProps> = {}, sheet = instantSheet()) {
  const props = bodyProps(over, sheet)
  const vue = render(<ReplayPlayerPanelBody {...props} />)
  return { props, ...vue }
}

/** Le lobby de la fixture, avec ce joueur à la place du sien. */
function joueur(xuid: string, sheet = instantSheet()): { player: ReplayPlayer; players: ReplayPlayer[] } {
  const players = instantPlayers(DOC, sheet)
  return { player: instantPlayer(players, xuid), players }
}

/** Les titres des sections rendues, dans l'ordre du document. */
const sections = (root: ParentNode = document) =>
  [...root.querySelectorAll('section[aria-label]')].map((s) => s.getAttribute('aria-label'))

describe('ReplayPlayerPanelBody — l’identité', () => {
  it('nomme le joueur, son camp, et montre ses compteurs du film à l’instant lu', () => {
    setup()
    expect(screen.getByRole('group', { name: 'Alpha' })).toBeTruthy()
    expect(screen.getByText('Alpha').tagName).toBe('B')
    expect(screen.getByText('Équipe Eagle')).toBeTruthy()
    expect(screen.getByTitle("Frags / morts / assistances à l'instant lu").textContent).toBe('4/1/1')
  })

  it('sans compteurs du film, la tête retombe sur les totaux de la feuille et le dit en infobulle', () => {
    setup({ doc: instantDoc({ scoreTimeline: undefined }) })
    expect(screen.getByTitle('Frags / morts / assistances du match').textContent).toBe('7/3/2')
  })

  it('aucune bordure colorée à gauche : l’ancienne tête n’existe plus', () => {
    const { container } = setup()
    expect(container.innerHTML).not.toMatch(/border-left|border-l-2|borderLeft/)
    expect(container.querySelector('[data-fiche-identite]')).not.toBeNull()
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

  it('le nom est à 17 px dans la colonne (`outer`), à 16 px dans la fenêtre (`internal`)', () => {
    const { unmount } = render(<ReplayPlayerPanelBody {...bodyProps()} scroll="outer" />)
    expect(screen.getByText('Alpha').style.fontSize).toBe('17px')
    unmount()
    render(<ReplayPlayerPanelBody {...bodyProps()} scroll="internal" />)
    expect(screen.getByText('Alpha').style.fontSize).toBe('16px')
  })
})

describe('ReplayPlayerPanelBody — l’action Seul', () => {
  it('« Seul » est la seule action de la fiche : un bouton à bascule, ni Suivre ni Masquer', () => {
    setup({ solo: true })
    const group = screen.getByRole('group', { name: 'Alpha' })
    const buttons = within(group).getAllByRole('button')
    expect(buttons.map((b) => b.textContent)).toEqual(['Seul · tout réafficher'])
    expect(buttons.map((b) => b.getAttribute('aria-pressed'))).toEqual(['true'])
    expect(screen.queryByRole('button', { name: 'Caméra : suivre' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Masquer' })).toBeNull()
  })

  it('un clic rappelle la page', () => {
    const { props } = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Seul' }))
    expect(props.onSolo).toHaveBeenCalledTimes(1)
  })

  it('l’infobulle de Seul dit que l’œil d’une tuile réaffiche un joueur', () => {
    setup()
    expect(screen.getByRole('button', { name: 'Seul' }).getAttribute('title')).toContain('L\'œil d\'une tuile en réaffiche un.')
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

describe('ReplayPlayerPanelBody — un joueur dont tout est lu, à la fin du match', () => {
  it('toutes les sections, dans l’ordre de la fiche, « À l’instant » juste après les actions', () => {
    const { container } = setup()
    expect(sections(container)).toEqual(SECTIONS_COMPLETES)
    const actions = screen.getByRole('group', { name: 'Alpha' })
    const bloc = screen.getByRole('region', { name: "À l'instant" })
    expect(actions.compareDocumentPosition(bloc) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    // L'horloge du bloc est celle de l'instant lu : 39,9 s de film.
    expect(within(bloc).getByText('0:39')).toBeTruthy()
  })

  it('la section « Gestes » est la dernière, « Totaux du match » l’avant-dernière, toutes deux sous un filet', () => {
    const { container } = setup()
    const titres = sections(container)
    expect(titres.slice(-2)).toEqual(['Totaux du match', 'Gestes'])
    for (const nom of ['Totaux du match', 'Gestes']) {
      const section = screen.getByRole('region', { name: nom })
      expect(section.parentElement?.className).toContain('border-t')
    }
  })

  it('« Tirs et dégâts », « Ramassages » et « Gestes » portent la capsule « estimation »', () => {
    setup()
    for (const nom of ['Tirs et dégâts', 'Ramassages', 'Gestes']) {
      expect(within(screen.getByRole('region', { name: nom })).getByText('estimation')).toBeTruthy()
    }
  })

  it('les valeurs de la fiche : FDA, rang, précision estimée, éliminations, équipement, médailles', () => {
    setup()
    const depuis = screen.getByRole('region', { name: 'Depuis le début du match' })
    expect(within(depuis).getByText('FDA').nextElementSibling?.textContent).toBe('3,33')
    expect(within(depuis).getByText('Score').nextElementSibling?.textContent).toBe('300 · 2e sur 3')
    const tirs = screen.getByRole('region', { name: 'Tirs et dégâts' })
    expect(within(tirs).getByText('Précision').nextElementSibling?.textContent).toBe('17 % · 6 / 35')
    const kills = screen.getByRole('table', { name: 'Éliminations par arme' })
    expect(within(kills).getByText('Lourde')).toBeTruthy()
    expect(within(screen.getByRole('region', { name: 'Médailles' })).getByText('×2')).toBeTruthy()
    const totaux = screen.getByRole('region', { name: 'Totaux du match' })
    expect(within(totaux).getByText('Précision').nextElementSibling?.textContent).toBe('41 % · 118 / 288')
    expect(within(totaux).getByText('Éliminations à la tête').nextElementSibling?.textContent).toBe('6')
  })

  it('en anglais, les titres et les libellés suivent', () => {
    const { container } = setup({ locale: 'en' })
    expect(sections(container)).toEqual([
      'Right now', 'Antagonists', 'Since the start of the match', 'Kills by weapon', 'Shots and damage', 'Grenades thrown',
      'Pickups', 'Equipment', 'Medals', 'Match totals', 'Moves',
    ])
  })
})

describe('ReplayPlayerPanelBody — la fiche suit l’image lue', () => {
  it('deux images, deux fiches : les valeurs changent, les sections qui n’existent pas encore apparaissent', () => {
    const props = bodyProps({ frame: 25, nowMs: 2500 })
    const { rerender } = render(<ReplayPlayerPanelBody {...props} />)
    const depuis = () => screen.getByRole('region', { name: 'Depuis le début du match' })
    expect(within(depuis()).getByText('Réapparitions').nextElementSibling?.textContent).toBe('0')
    expect(within(depuis()).getByText('Boucliers brisés').nextElementSibling?.textContent).toBe('0 · 3 sur le match')
    expect(within(depuis()).getByText('Recharges complètes').nextElementSibling?.textContent).toBe('0 · 1 sur le match')
    expect(screen.queryByRole('region', { name: 'Équipement' })).toBeNull()
    expect(screen.queryByRole('region', { name: 'Médailles' })).toBeNull()
    rerender(<ReplayPlayerPanelBody {...props} frame={INSTANT_LAST_FRAME} nowMs={INSTANT_LAST_FRAME * 100} />)
    expect(within(depuis()).getByText('Réapparitions').nextElementSibling?.textContent).toBe('1')
    expect(within(depuis()).getByText('Boucliers brisés').nextElementSibling?.textContent).toBe('3 · 3 sur le match')
    expect(within(depuis()).getByText('Recharges complètes').nextElementSibling?.textContent).toBe('1 · 1 sur le match')
    expect(within(depuis()).getByText('Différentiel').nextElementSibling?.textContent).toBe('−2 · −2 sur le match')
    expect(screen.getByRole('region', { name: 'Équipement' })).toBeTruthy()
    expect(screen.getByRole('region', { name: 'Médailles' })).toBeTruthy()
  })

  it('au début du match : des zéros mesurés dans les tableaux, presque aucune autre section', () => {
    const { container } = setup({ frame: 0, nowMs: 0 })
    expect(sections(container)).toEqual([
      "À l'instant", 'Antagonistes', 'Depuis le début du match', 'Éliminations par arme', 'Tirs et dégâts', 'Totaux du match',
    ])
    const kills = screen.getByRole('table', { name: 'Éliminations par arme' })
    expect(within(kills).getAllByRole('row')[1].textContent).toContain('0')
    expect(screen.queryByRole('img', { name: /Arme d'épaule/ })).toBeNull()
  })
})

describe('ReplayPlayerPanelBody — rien d’inventé', () => {
  it('lecture partielle : ni score ni rang, ni « Tirs et dégâts », aucun texte de remplacement', () => {
    const sheet = instantSheet().map((r) => (r.xuid === 'E' ? { ...r, score: 999 } : r))
    const { player, players } = joueur('E', sheet)
    const { container } = setup({ player, players }, sheet)
    const titres = sections(container)
    expect(titres).not.toContain('Tirs et dégâts')
    expect(screen.queryByText('Score', { selector: 'dt' })).not.toBeNull() // le total de la feuille, dans les totaux
    const depuis = screen.getByRole('region', { name: 'Depuis le début du match' })
    expect(within(depuis).queryByText('Score')).toBeNull()
    expect(container.textContent).not.toMatch(/—|inconnu|non mesur|non lu|indisponible|n\/a/i)
  })

  it('un joueur sans ligne de feuille : ni totaux ni score, ni bloc de remplacement ; la fiche demeure', () => {
    const sheet = instantSheet().filter((r) => r.xuid !== 'A')
    const { player, players } = joueur('A', sheet)
    const { container } = setup({ player, players }, sheet)
    expect(sections(container)).not.toContain('Totaux du match')
    expect(screen.queryByText(/Pas de statistiques/)).toBeNull()
    expect(screen.getByRole('button', { name: 'Seul' })).toBeTruthy()
    expect(screen.getByRole('region', { name: "À l'instant" })).toBeTruthy()
    const depuis = screen.getByRole('region', { name: 'Depuis le début du match' })
    expect(within(depuis).queryByText('Score')).toBeNull()
  })

  it('un bot : ses totaux de feuille, aucune médaille ni tir, pas d’équipement', () => {
    const { player, players } = joueur('bot:Marc')
    const { container } = setup({ player, players })
    expect(screen.getByRole('group', { name: 'Marc' })).toBeTruthy()
    const titres = sections(container)
    expect(titres).toContain("À l'instant")
    for (const absent of ['Tirs et dégâts', 'Médailles', 'Éliminations par arme', 'Grenades lancées', 'Ramassages']) {
      expect(titres).not.toContain(absent)
    }
  })

  it('un joueur mort à l’image lue : l’encadré « Éliminé » du bloc, la fiche entière demeure', () => {
    const { container } = setup({ frame: 160, nowMs: 16_000 })
    const bloc = screen.getByRole('region', { name: "À l'instant" })
    expect(within(bloc).getByText(/Éliminé/i)).toBeTruthy()
    expect(within(bloc).queryByRole('meter')).toBeNull()
    const depuis = screen.getByRole('region', { name: 'Depuis le début du match' })
    expect(within(depuis).queryByText('Vie en cours')).toBeNull()
    expect(sections(container)).toContain('Éliminations par arme')
  })

  it('un document sans aucune donnée (fil vide, pas de tirs) : l’identité, les actions, le bloc et les antagonistes (état vide)', () => {
    const doc = instantDoc({ scoreTimeline: undefined, shotStats: [], stances: [], pickups: [], grenades: [], equipmentChanges: [] })
    const { player, players } = joueur('A', [])
    const { container } = setup({ doc, player, players, feed: [], frame: 0, nowMs: 0 }, [])
    expect(sections(container)).toEqual(["À l'instant", 'Antagonistes', 'Depuis le début du match'])
  })
})

describe('ReplayPlayerPanelBody — la section « Antagonistes »', () => {
  it('se pose après « À l’instant » et avant « Depuis le début du match »', () => {
    const { container } = setup()
    const titres = sections(container)
    expect(titres.indexOf('Antagonistes')).toBe(titres.indexOf("À l'instant") + 1)
    expect(titres.indexOf('Antagonistes')).toBe(titres.indexOf('Depuis le début du match') - 1)
  })

  it('lit l’index du joueur de la fiche à l’instant lu', () => {
    const at = vi.fn<(t: number) => AntagonistsAt>(() => ({ nemesis: { holder: 'E', count: 2, tiedWith: [] }, victim: null }))
    const forSubject = vi.fn<(key: string | null) => Antagonists>(() => ({ changes: [], at }))
    setup({ nowMs: 4_200, antagonists: { forSubject, forViewpoint: () => ({ changes: [], at }) } })
    expect(forSubject).toHaveBeenCalledWith('A')
    expect(at).toHaveBeenCalledWith(4_200)
    expect(screen.getByRole('region', { name: 'Antagonistes' }).textContent).toContain('Echo, 2 éliminations')
  })

  it('sans adversaire à l’instant, la section existe et le dit', () => {
    setup()
    expect(screen.getByRole('region', { name: 'Antagonistes' }).textContent).toContain('Aucun antagoniste à cet instant.')
  })
})
