/**
 * Tests — le panneau de joueur prend la place du fil SANS le détruire (lot 4 du plan « maquettes du
 * rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. LE FIL RESTE MONTÉ : ouvrir puis fermer le panneau ne remonte ni la liste ni le menu — le DOM
 *     est le même nœud, donc la position de défilement ET le filtre choisi sont retrouvés.
 *  2. LE FIL N'EST PLUS LU NI ATTEIGNABLE pendant que le panneau est ouvert : `invisible` (boîte
 *     gardée, hors focus et hors lecture d'écran) et `aria-hidden`.
 *  3. LE PANNEAU REMPLIT LA CELLULE du fil, rien d'autre : la carte n'est jamais couverte.
 *  4. LA MISE EN VALEUR D'UNE ÉLIMINATION (lot 2) ET LE PANNEAU COEXISTENT : ouvrir le panneau ne
 *     l'efface pas (elle reste sur la carte), et la ligne choisie est retrouvée au retour.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { buildFeedEntries } from '../model/killFeedLogic'
import { createPlaybackStore, type PlaybackStore } from '../model/playbackStore'
import { FEED_FILTER_KEY } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { ReplayFeedOrPanel } from './ReplayFeedOrPanel'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['foe', { gamertag: 'Cobra01' }],
])

const KILL: KillEvent = {
  tMs: 2_050, xuid: 'me', ally: true, teamID: 0, weaponKey: 'br75', weaponLabel: 'BR75',
  weaponImageUrl: '/br75.png', weaponTinted: true, assistState: '', assistGamertag: '', assistTeamID: null,
  killerDamagePct: null, assistDamagePct: null, victimXuid: 'foe', victimGamertag: 'Cobra01',
}

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [
    { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
    { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
  ],
})

function feed(store: PlaybackStore) {
  return (
    <ReplayKillFeed
      entries={buildFeedEntries([KILL], [], 0, null)}
      nowMs={100_000}
      playWindow={null}
      scoreboard={[]}
      xuidMeta={META}
      allegiance={filmAllegianceOf(DOC, [], 'me')}
      locale="fr"
      doc={DOC}
      playbackStore={store}
    />
  )
}

const PANEL = <div data-testid="panneau">panneau</div>

beforeEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))
afterEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))

describe('ReplayFeedOrPanel', () => {
  it('sans panneau, le fil est affiché normalement', () => {
    render(<ReplayFeedOrPanel feed={<p>fil</p>} panel={null} />)
    const cellule = screen.getByTestId('replay-feed-slot')
    expect(cellule.className).not.toContain('invisible')
    expect(cellule.hasAttribute('aria-hidden')).toBe(false)
    expect(screen.queryByTestId('panneau')).toBeNull()
  })

  it('avec le panneau, le fil reste monté mais invisible et caché aux lecteurs d’écran', () => {
    render(<ReplayFeedOrPanel feed={<p>fil</p>} panel={PANEL} />)
    const cellule = screen.getByTestId('replay-feed-slot')
    expect(screen.getByText('fil')).toBeTruthy()
    expect(cellule.className).toContain('invisible')
    expect(cellule.getAttribute('aria-hidden')).toBe('true')
    expect(screen.getByTestId('panneau').parentElement?.className).toContain('absolute inset-0')
  })

  it('ouvrir puis fermer ne remonte pas le fil : même liste, filtre et défilement retrouvés', () => {
    const store = createPlaybackStore()
    const { rerender } = render(<ReplayFeedOrPanel feed={feed(store)} panel={null} />)

    // Le user choisit le filtre « Éliminations » et fait défiler la liste.
    fireEvent.click(screen.getByRole('button', { name: /Filtrer le fil/ }))
    fireEvent.click(screen.getByRole('menuitemradio', { name: /Éliminations/ }))
    const liste = screen.getByRole('list')
    liste.scrollTop = 120
    const filtreAvant = window.localStorage.getItem(FEED_FILTER_KEY)

    rerender(<ReplayFeedOrPanel feed={feed(store)} panel={PANEL} />)
    expect(screen.getByRole('list', { hidden: true })).toBe(liste)
    expect(liste.scrollTop).toBe(120)

    rerender(<ReplayFeedOrPanel feed={feed(store)} panel={null} />)
    expect(screen.getByRole('list')).toBe(liste)
    expect(liste.scrollTop).toBe(120)
    expect(window.localStorage.getItem(FEED_FILTER_KEY)).toBe(filtreAvant)
  })

  it('la mise en valeur d’une élimination reste posée panneau ouvert, et sa ligne est retrouvée au retour', () => {
    const store = createPlaybackStore()
    const { rerender } = render(<ReplayFeedOrPanel feed={feed(store)} panel={null} />)
    fireEvent.click(screen.getByRole('button', { pressed: false }))
    expect(store.focus()).not.toBeNull()

    rerender(<ReplayFeedOrPanel feed={feed(store)} panel={PANEL} />)
    expect(store.focus()).not.toBeNull()

    rerender(<ReplayFeedOrPanel feed={feed(store)} panel={null} />)
    expect(store.focus()).not.toBeNull()
    expect(screen.getByRole('button', { pressed: true })).toBeTruthy()
  })
})
