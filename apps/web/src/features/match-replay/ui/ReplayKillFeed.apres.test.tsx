/**
 * ReplayKillFeed.apres.test.tsx — fixations du fil d'éliminations au format d'une ligne de 28 px aux
 * colonnes fixes, et le CONTENU de chaque ligne (instant, lieu, noms, ligne d'événement).
 *
 * LE CONTENU EST CELUI D'AVANT LA REFONTE : avant de remplacer les fixations `replayKillFeed*.avant.html`
 * (DOM de l'ancienne ligne), le contenu textuel de chaque ligne — textes des feuilles et noms accessibles
 * des images, dans l'ordre du document — a été comparé à celui de ces fixations, sur les deux scènes
 * ci-dessous : identique, ligne à ligne. Les tableaux `CONTENU_*` sont ce contenu ; seul l'ordre des colonnes
 * du DOM a changé.
 *
 * Ce que les fixations couvrent : quatre éliminations dans les deux sens et dans les deux camps, plus une
 * ligne d'événement (capture de drapeau, portée par `objectives` du document de test). La seconde fixation
 * rend la même entrée avec le lieu de la victime (`zoneOf`) : trois éliminations nommées, une sans lieu.
 *
 * Aucune valeur ne varie : l'instant est fixé par `nowMs`, aucune date ni identifiant généré n'entre
 * dans le rendu. Régénération : `npx vitest run <ce fichier> -u`, jamais pour faire passer une divergence.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen, within } from '@testing-library/react'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { buildFeedEntries } from '../model/killFeedLogic'
import { testReplayDoc } from '../test/testDoc'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const FRAME_MS = 100

const META = new Map([
  ['me', { gamertag: 'JGtm', ally: true }],
  ['mate', { gamertag: 'Ally02', ally: true }],
  ['foe', { gamertag: 'Cobra01', ally: false }],
  ['foe2', { gamertag: 'Viper', ally: false }],
])

const ROSTER = [
  { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
  { xuid: 'mate', filmIndex: 1, name: 'Ally02', team: 0 },
  { xuid: 'foe', filmIndex: 2, name: 'Cobra01', team: 1 },
  { xuid: 'foe2', filmIndex: 3, name: 'Viper', team: 1 },
]

function kill(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 2_000, xuid: 'me', ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '',
    weaponTinted: false, assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null,
    assistDamagePct: null, victimXuid: 'foe', victimGamertag: 'Cobra01', ...over,
  }
}

/** Quatre éliminations : deux de chaque côté, le joueur regardé dans deux d'entre elles. */
const KILLS: KillEvent[] = [
  kill({ tMs: 2_000, xuid: 'me', victimXuid: 'foe', victimGamertag: 'Cobra01' }),
  kill({ tMs: 5_000, xuid: 'mate', ally: true, victimXuid: 'foe2', victimGamertag: 'Viper' }),
  kill({ tMs: 9_000, xuid: 'foe', ally: false, teamID: 1, victimXuid: 'me', victimGamertag: 'JGtm' }),
  kill({ tMs: 12_000, xuid: 'foe2', ally: false, teamID: 1, victimXuid: 'mate', victimGamertag: 'Ally02' }),
]

/** Une capture de drapeau par le joueur regardé : une ligne d'événement, pas une élimination. */
const DOC = testReplayDoc({
  frameIntervalMs: FRAME_MS,
  roster: ROSTER,
  objectives: [{ stat: 'flag_captures', t: 60, xuid: 'me', timeMs: 60 * FRAME_MS }] as never,
})

const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')

/** Le contenu d'une ligne : les textes des feuilles, dans l'ordre du document, sans les textes réservés aux lecteurs d'écran. */
function contenu(li: Element): string[] {
  const out: string[] = []
  const visit = (el: Element) => {
    if (el.getAttribute('aria-hidden') === 'true' || el.classList.contains('sr-only')) return
    if (el.children.length === 0 && (el.textContent ?? '') !== '') out.push(el.textContent ?? '')
    for (const c of Array.from(el.children)) visit(c)
  }
  visit(li)
  return out
}

const contenus = (racine: HTMLElement) => Array.from(racine.querySelectorAll('li')).map(contenu)

/** Le contenu des lignes de la scène sans lieu, la plus récente en tête. */
const CONTENU_SANS_LIEU = [
  ['0:12', 'Viper', 'Ally02'],
  ['0:09', 'Cobra01', 'JGtm'],
  ['0:06', 'JGtm', 'a capturé le drapeau'],
  ['0:05', 'Ally02', 'Viper'],
  ['0:02', 'JGtm', 'Cobra01'],
]

/** Le contenu des lignes de la scène avec lieu : la quatrième élimination, sans lieu, garde sa colonne (vide). */
const CONTENU_AVEC_LIEU = [
  ['0:12', 'Viper', 'Ally02'],
  ['0:09', 'Pont Est', 'Cobra01', 'JGtm'],
  ['0:06', 'JGtm', 'a capturé le drapeau'],
  ['0:05', 'Rampe Nord', 'Ally02', 'Viper'],
  ['0:02', 'Fer à cheval', 'JGtm', 'Cobra01'],
]

describe('ReplayKillFeed — fixation des lignes du fil (format d’une ligne)', () => {
  it('le HTML du fil (quatre éliminations et une ligne d’événement) est celui de la fixture, et le contenu celui d’avant la refonte', async () => {
    const vue = render(
      <ReplayKillFeed
        entries={buildFeedEntries(KILLS, [], 0, DOC)}
        nowMs={1_000_000}
        playWindow={null}
        scoreboard={[]}
        xuidMeta={META}
        allegiance={ALLEGIANCE}
        locale="fr"
        doc={DOC}
      />,
    )
    expect(screen.getAllByRole('listitem').length).toBeGreaterThanOrEqual(5)
    expect(vue.container.textContent).toMatch(/capturé le drapeau/)
    expect(contenus(vue.container)).toEqual(CONTENU_SANS_LIEU)
    await expect(vue.container.innerHTML).toMatchFileSnapshot('./__fixtures__/replayKillFeed.apres.html')
  })
})

describe('ReplayKillFeed — fixation avec le lieu de la victime (format d’une ligne)', () => {
  it('le même fil avec la colonne du lieu : trois éliminations nommées, une sans lieu', async () => {
    const entries = buildFeedEntries(KILLS, [], 0, DOC)
    const cles = entries.filter((e) => e.kill).map((e) => e.key)
    expect(cles).toHaveLength(4)
    const lieux = new Map<string, string>([
      [cles[0], 'Fer à cheval'],
      [cles[1], 'Rampe Nord'],
      [cles[2], 'Pont Est'],
    ])
    const vue = render(
      <ReplayKillFeed
        entries={entries}
        nowMs={1_000_000}
        playWindow={null}
        scoreboard={[]}
        xuidMeta={META}
        allegiance={ALLEGIANCE}
        locale="fr"
        doc={DOC}
        zoneOf={(key) => lieux.get(key) ?? ''}
      />,
    )
    for (const nom of lieux.values()) expect(within(vue.container).getByText(nom)).toBeTruthy()
    expect(contenus(vue.container)).toEqual(CONTENU_AVEC_LIEU)
    await expect(vue.container.innerHTML).toMatchFileSnapshot('./__fixtures__/replayKillFeed.lieu.apres.html')
  })
})
