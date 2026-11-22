/**
 * Tests — la fiche du joueur en fenêtre (conteneur `ReplayPlayerPopover`, Cinéma et Plein cadre).
 *
 * Ce qu'ils protègent : la fenêtre monte le MÊME corps que la colonne du Standard (mêmes sections, dans le même
 * ordre) ; elle fait 330 px de large, son nom est à 16 px, la tête et les actions restent en place et la partie
 * basse défile d'elle-même quand la fiche dépasse la hauteur de la fenêtre ; la croix et Échap la ferment.
 * Le placement (ancrage, bornes) est couvert par `ReplayPage.fenetre.test.tsx`.
 */
import { cleanup, fireEvent, render, screen, within } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { COTES_FENETRE_FICHE } from '../model/cotesFenetreFiche'
import {
  INSTANT_ICONS, INSTANT_LAST_FRAME, instantDoc, instantFeed, instantPlayer, instantPlayers,
} from '../test/instantFixture'
import { ReplayPlayerPopover, type ReplayPlayerPopoverProps } from './ReplayPlayerPopover'

afterEach(cleanup)

function props(over: Partial<ReplayPlayerPopoverProps> = {}): ReplayPlayerPopoverProps {
  const doc = instantDoc()
  const players = instantPlayers(doc)
  return {
    player: instantPlayer(players, 'A'),
    actions: { follow: false, solo: false, hide: false },
    onToggle: vi.fn(),
    onClose: vi.fn(),
    doc,
    frame: INSTANT_LAST_FRAME,
    nowMs: INSTANT_LAST_FRAME * 100,
    playWindow: null,
    allegiance: filmAllegianceOf(doc, [], 'A'),
    locale: 'fr',
    players,
    feed: instantFeed(),
    weaponIcons: INSTANT_ICONS,
    ...over,
  }
}

const fenetre = () => document.querySelector('[data-replay-fiche-popover]') as HTMLElement

describe('fiche en fenêtre — le corps commun', () => {
  it('toutes les sections de la fiche, comme la colonne du Standard', () => {
    render(<ReplayPlayerPopover {...props()} />)
    const titres = [...fenetre().querySelectorAll('section[aria-label]')]
      .map((s) => s.getAttribute('aria-label'))
      // La fenêtre est elle-même une section : son nom accessible est celui de la fiche.
      .filter((t) => t !== 'Fiche de Alpha')
    expect(titres).toEqual([
      "À l'instant", 'Depuis le début du match', 'Éliminations par arme', 'Tirs et dégâts', 'Grenades lancées',
      'Ramassages', 'Équipement', 'Médailles', 'Totaux du match', 'Gestes',
    ])
  })

  it('330 px de large, nom à 16 px, capsule « Échap » et croix à droite de l’identité', () => {
    render(<ReplayPlayerPopover {...props()} />)
    expect(fenetre().style.width).toBe(`${COTES_FENETRE_FICHE.largeurPx}px`)
    expect(COTES_FENETRE_FICHE.largeurPx).toBe(330)
    expect(within(fenetre()).getByText('Alpha').style.fontSize).toBe('16px')
    const identite = fenetre().querySelector('[data-fiche-identite]') as HTMLElement
    expect(within(identite).getByText('Échap')).toBeTruthy()
    expect(within(identite).getByRole('button', { name: 'Fermer' })).toBeTruthy()
  })

  it('la fiche défile dans la fenêtre : tête et actions fixes, partie basse qui défile', () => {
    render(<ReplayPlayerPopover {...props()} />)
    expect(fenetre().className).toContain('overflow-hidden')
    const basse = (fenetre().querySelector('[data-replay-player-now]') as HTMLElement).parentElement as HTMLElement
    expect(basse.className).toContain('overflow-y-auto')
    expect(basse.className).toContain('min-h-0')
    // L'identité et les actions sont hors de la région qui défile.
    expect(basse.contains(fenetre().querySelector('[data-fiche-identite]'))).toBe(false)
    expect(basse.contains(within(fenetre()).getByRole('group', { name: 'Alpha' }))).toBe(false)
    expect(fenetre().style.maxHeight).not.toBe('')
  })

  it('son haut est à 8 px sous le haut du bloc du plan, pas sous la fiche', () => {
    const bloc = document.createElement('div')
    bloc.setAttribute('data-replay-plan-block', '')
    bloc.getBoundingClientRect = () => ({ top: 200, bottom: 700, left: 0, right: 900, width: 900, height: 500, x: 0, y: 200, toJSON: () => ({}) })
    const fiche = document.createElement('button')
    fiche.setAttribute('data-replay-player-card', '')
    fiche.setAttribute('aria-pressed', 'true')
    fiche.getBoundingClientRect = () => ({ top: 100, bottom: 170, left: 140, right: 260, width: 120, height: 70, x: 140, y: 100, toJSON: () => ({}) })
    document.body.append(bloc, fiche)
    render(<ReplayPlayerPopover {...props()} />)
    expect(fenetre().style.top).toBe(`${200 + COTES_FENETRE_FICHE.margeHautPlanPx}px`)
    expect(COTES_FENETRE_FICHE.margeHautPlanPx).toBe(8)
    bloc.remove()
    fiche.remove()
  })

  it('la croix et Échap ferment la fenêtre', () => {
    const p = props()
    render(<ReplayPlayerPopover {...p} />)
    fireEvent.click(screen.getByRole('button', { name: 'Fermer' }))
    expect(p.onClose).toHaveBeenCalledTimes(1)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(p.onClose).toHaveBeenCalledTimes(2)
  })
})
