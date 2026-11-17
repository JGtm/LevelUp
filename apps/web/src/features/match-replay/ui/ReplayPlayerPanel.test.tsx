/**
 * Tests — la fiche du joueur dans la colonne du Standard (conteneur `ReplayPlayerPanel`).
 *
 * Ce qu'ils protègent :
 *  1. L'EN-TÊTE : le retour « Équipes » (chevron) à gauche, la capsule « Échap » à droite ; le retour rend la
 *     colonne, pas « Retour au fil ».
 *  2. UNE SEULE ZONE QUI DÉFILE : retour, sélecteur, tête, actions et sections y sont ensemble ; le pied
 *     « Éliminations » est fixé en bas, HORS de cette zone ; la partie basse du corps ne défile pas d'elle-même.
 *  3. ÉCHAP est une couche (marqueur, une seule fermeture, une couche intérieure d'abord) et rend la colonne ; le
 *     focus est rendu à la fiche de la colonne, y compris quand elle n'est de nouveau visible qu'au retrait du
 *     panneau.
 *  4. LE PIED REÇOIT LA SORTIE du panneau : « Fil complet » ferme comme « Équipes ».
 *  5. LE CORPS COMMUN : les sections de la fiche enrichie y sont, dans l'ordre, dans la zone qui défile.
 */
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { useState } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { COTES_FICHE_STANDARD } from '../model/cotesFicheStandard'
import { COTES_COLONNE_STANDARD } from '../model/cotesStandard'
import {
  INSTANT_ICONS, INSTANT_LAST_FRAME, instantDoc, instantFeed, instantPlayer, instantPlayers,
} from '../test/instantFixture'
import { testReplayDoc } from '../test/testDoc'
import { ReplayPlayerPanel, type ReplayPlayerPanelProps } from './ReplayPlayerPanel'

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [{ xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 }],
})
const PLAYER: ReplayPlayer = { xuid: 'A', filmName: 'Alpha', team: 0, lives: [] }

afterEach(cleanup)

function props(over: Partial<ReplayPlayerPanelProps> = {}): ReplayPlayerPanelProps {
  return {
    player: PLAYER,
    actions: { follow: false, solo: false, hide: false },
    onToggle: vi.fn(),
    onClose: vi.fn(),
    doc: DOC,
    frame: 10,
    nowMs: 65_000,
    playWindow: null,
    allegiance: filmAllegianceOf(DOC, [], 'A'),
    locale: 'fr',
    players: [PLAYER],
    feed: [],
    weaponIcons: new Map(),
    column: {
      picker: <div data-testid="selecteur">sélecteur</div>,
      footer: (close) => (
        <div data-testid="pied">
          <button type="button" onClick={close}>
            Fil complet
          </button>
        </div>
      ),
    },
    ...over,
  }
}

function setup(over: Partial<ReplayPlayerPanelProps> = {}) {
  const p = props(over)
  render(<ReplayPlayerPanel {...p} />)
  return p
}

const zone = () => document.querySelector('[data-replay-fiche-scroll]') as HTMLElement
const echap = () => fireEvent.keyDown(window, { key: 'Escape' })

describe('fiche de la colonne — l’en-tête', () => {
  it('le retour « Équipes » à gauche, la capsule « Échap » à droite ; il appelle la sortie', () => {
    const p = setup()
    const retour = screen.getByRole('button', { name: 'Équipes' })
    const capsule = screen.getByText('Échap')
    expect(capsule.tagName).toBe('KBD')
    expect(capsule.className).toContain('ml-auto')
    // Le retour précède la capsule dans l'en-tête.
    expect(retour.compareDocumentPosition(capsule) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(retour.querySelector('svg')?.getAttribute('aria-hidden')).toBe('true')
    expect(retour.style.height).toBe(`${COTES_FICHE_STANDARD.retour.hauteurPx}px`)
    fireEvent.click(retour)
    expect(p.onClose).toHaveBeenCalledTimes(1)
    expect(screen.queryByRole('button', { name: 'Retour au fil' })).toBeNull()
  })

  it('en anglais : « Teams » et « Esc »', () => {
    setup({ locale: 'en' })
    expect(screen.getByRole('button', { name: 'Teams' })).toBeTruthy()
    expect(screen.getByText('Esc')).toBeTruthy()
  })
})

describe('fiche de la colonne — la zone qui défile et le pied fixé', () => {
  it('retour, sélecteur, tête, actions et sections sont tous dans la même zone qui défile', () => {
    setup()
    const z = zone()
    expect(z.className).toContain('overflow-y-auto')
    expect(within(z).getByRole('button', { name: 'Équipes' })).toBeTruthy()
    expect(within(z).getByTestId('selecteur')).toBeTruthy()
    expect(within(z).getByRole('group', { name: 'Alpha' })).toBeTruthy()
    expect(within(z).getByText('Suivre')).toBeTruthy()
    expect(within(z).getByRole('region', { name: "À l'instant" })).toBeTruthy()
  })

  it('le pied est un frère de la zone, jamais un de ses enfants : il reste fixé en bas', () => {
    setup()
    const z = zone()
    const pied = screen.getByTestId('pied')
    expect(z.contains(pied)).toBe(false)
    expect(pied.parentElement).toBe(z.parentElement)
    // Le pied vient après la zone, et c'est la zone qui prend la place restante.
    expect(z.compareDocumentPosition(pied) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(z.className).toContain('flex-1')
    expect(z.className).toContain('min-h-0')
  })

  it('la partie basse du corps ne défile pas d’elle-même : le conteneur seul défile', () => {
    setup()
    const now = document.querySelector('[data-replay-player-now]') as HTMLElement
    const basse = now.parentElement as HTMLElement
    expect(basse.className).not.toContain('overflow-y-auto')
    expect(basse.className).not.toContain('flex-1')
    expect(zone().querySelectorAll('.overflow-y-auto')).toHaveLength(0)
  })

  it('la marge intérieure, l’écart et la marge de barre sont ceux des cotes', () => {
    setup()
    const section = screen.getByRole('region', { name: 'Fiche de Alpha' })
    expect(section.style.padding).toBe(
      `${COTES_COLONNE_STANDARD.margeVerticalePx}px ${COTES_COLONNE_STANDARD.margeLateralePx}px`,
    )
    expect(section.style.gap).toBe(`${COTES_COLONNE_STANDARD.ecartBlocsPx}px`)
    expect(zone().style.gap).toBe(`${COTES_FICHE_STANDARD.ecartBlocsPx}px`)
    expect(zone().style.paddingRight).toBe(`${COTES_FICHE_STANDARD.margeBarrePx}px`)
  })

  it('porte le nom accessible de la fiche et le marqueur de couche d’Échap', () => {
    setup()
    const section = screen.getByRole('region', { name: 'Fiche de Alpha' })
    expect(section.hasAttribute('data-replay-escape-layer')).toBe(true)
    expect(section.hasAttribute('data-replay-fiche-column')).toBe(true)
  })
})

describe('fiche de la colonne — la sortie', () => {
  it('Échap rend la colonne, une seule fois ; une autre touche non', () => {
    const p = setup()
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(p.onClose).not.toHaveBeenCalled()
    echap()
    expect(p.onClose).toHaveBeenCalledTimes(1)
  })

  it('le panneau retiré, plus d’écouteur : Échap ne rappelle plus la page', () => {
    const p = setup()
    cleanup()
    echap()
    expect(p.onClose).not.toHaveBeenCalled()
  })

  it('le focus est rendu à la fiche du joueur à Échap comme à « Équipes »', () => {
    const fiche = document.createElement('div')
    fiche.setAttribute('role', 'button')
    fiche.setAttribute('aria-pressed', 'true')
    fiche.setAttribute('data-replay-player-card', '')
    fiche.tabIndex = 0
    document.body.appendChild(fiche)
    try {
      setup()
      echap()
      expect(document.activeElement).toBe(fiche)
      fiche.blur()
      fireEvent.click(screen.getByRole('button', { name: 'Équipes' }))
      expect(document.activeElement).toBe(fiche)
    } finally {
      fiche.remove()
    }
  })

  it('une couche plus intérieure ouverte se ferme d’abord : la fiche reste', () => {
    const p = setup()
    const autre = document.createElement('div')
    autre.setAttribute('data-replay-escape-layer', '')
    document.body.appendChild(autre)
    try {
      echap()
      expect(p.onClose).not.toHaveBeenCalled()
    } finally {
      autre.remove()
    }
  })

  it('« Fil complet », reçu du conteneur, ferme comme « Équipes »', () => {
    const p = setup()
    fireEvent.click(screen.getByRole('button', { name: 'Fil complet' }))
    expect(p.onClose).toHaveBeenCalledTimes(1)
  })

  it('le focus revient à la fiche de la colonne, même quand elle ne redevient visible qu’au retrait du panneau', () => {
    const fiche = document.createElement('div')
    fiche.setAttribute('role', 'button')
    fiche.setAttribute('aria-pressed', 'true')
    fiche.setAttribute('data-replay-player-card', '')
    fiche.tabIndex = 0
    document.body.appendChild(fiche)
    function Hote() {
      const [ouvert, setOuvert] = useState(true)
      return ouvert ? <ReplayPlayerPanel {...props({ onClose: () => setOuvert(false) })} /> : null
    }
    try {
      render(<Hote />)
      fireEvent.click(screen.getByRole('button', { name: 'Équipes' }))
      expect(screen.queryByRole('region', { name: 'Fiche de Alpha' })).toBeNull()
      expect(document.activeElement).toBe(fiche)
    } finally {
      fiche.remove()
    }
  })

  it('le focus rendu est repris au retrait : une fiche masquée au moment du clic le reçoit ensuite', () => {
    const fiche = document.createElement('div')
    fiche.setAttribute('role', 'button')
    fiche.setAttribute('aria-pressed', 'true')
    fiche.setAttribute('data-replay-player-card', '')
    fiche.tabIndex = 0
    document.body.appendChild(fiche)
    const focus = vi.spyOn(fiche, 'focus')
    function Hote() {
      const [ouvert, setOuvert] = useState(true)
      return ouvert ? <ReplayPlayerPanel {...props({ onClose: () => setOuvert(false) })} /> : null
    }
    try {
      render(<Hote />)
      fireEvent.click(screen.getByRole('button', { name: 'Équipes' }))
      // Une fois au clic (fiche visible), une fois au retrait du panneau (fiche masquée jusque-là).
      expect(focus).toHaveBeenCalledTimes(2)
    } finally {
      fiche.remove()
    }
  })
})

describe('fiche de la colonne — le corps commun, enrichi', () => {
  it('toutes les sections de la fiche, de « À l’instant » à « Gestes », dans la zone qui défile ; le pied reste fixé', () => {
    const doc = instantDoc()
    const players = instantPlayers(doc)
    setup({
      doc, player: instantPlayer(players, 'A'), players, feed: instantFeed(), weaponIcons: INSTANT_ICONS,
      frame: INSTANT_LAST_FRAME, nowMs: INSTANT_LAST_FRAME * 100, allegiance: filmAllegianceOf(doc, [], 'A'),
    })
    const titres = [...zone().querySelectorAll('section[aria-label]')].map((s) => s.getAttribute('aria-label'))
    expect(titres).toEqual([
      "À l'instant", 'Depuis le début du match', 'Éliminations par arme', 'Tirs et dégâts', 'Grenades lancées',
      'Ramassages', 'Équipement', 'Médailles', 'Totaux du match', 'Gestes',
    ])
    expect(zone().contains(screen.getByTestId('pied'))).toBe(false)
    // La colonne fait défiler toute la fiche : le corps ne porte aucun défilement propre, et son nom est à 17 px.
    expect(zone().querySelectorAll('.overflow-y-auto')).toHaveLength(0)
    expect(within(zone()).getByText('Alpha').style.fontSize).toBe('17px')
  })
})
