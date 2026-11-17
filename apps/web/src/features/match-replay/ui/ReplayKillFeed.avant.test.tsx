/**
 * ReplayKillFeed.avant.test.tsx — fixations des lignes du fil d'éliminations.
 *
 * Ce que les fixations couvrent : quatre éliminations dans les deux sens et dans les deux camps, plus
 * une ligne d'événement (capture de drapeau, portée par `objectives` du document de test). La seconde
 * fixation rend la même entrée avec le lieu du tueur (`zoneOf`) : trois éliminations nommées, une sans lieu.
 *
 * Aucune valeur ne varie : l'instant est fixé par `nowMs`, aucune date ni identifiant généré n'entre
 * dans le rendu. Si un lancement en faisait apparaître un, le neutraliser dans le test, sans toucher au composant.
 *
 * Régénération : `npx vitest run <ce fichier> -u`, à la création seulement. Jamais pour faire passer une divergence.
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

describe('ReplayKillFeed — fixation des lignes du fil (avant la refonte)', () => {
  it('le HTML du fil (quatre éliminations et une ligne d’événement) est celui de la fixture prise avant la refonte', async () => {
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
    // Les lignes sont là, en nombre : au moins les quatre éliminations et la capture. Sans cela la
    // fixture comparerait une liste vide.
    const lignes = screen.getAllByRole('listitem')
    expect(lignes.length).toBeGreaterThanOrEqual(5)
    expect(screen.getAllByText('Cobra01').length).toBeGreaterThan(0)
    expect(screen.getAllByText('Viper').length).toBeGreaterThan(0)
    expect(vue.container.textContent).toMatch(/capturé le drapeau/)
    await expect(vue.container.innerHTML).toMatchFileSnapshot('./__fixtures__/replayKillFeed.avant.html')
  })
})

describe('ReplayKillFeed — fixation avec le lieu du tueur (avant la refonte)', () => {
  it('le même fil avec la gouttière du lieu : trois éliminations nommées, une sans lieu', async () => {
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
    // Les trois lieux nommés sont dans le rendu : sans eux la gouttière ne serait pas fixée.
    for (const nom of lieux.values()) expect(within(vue.container).getByText(nom)).toBeTruthy()
    await expect(vue.container.innerHTML).toMatchFileSnapshot('./__fixtures__/replayKillFeed.lieu.avant.html')
  })
})
