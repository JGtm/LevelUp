/**
 * ReplayPlayerPanelBody.fixation.test.tsx — fixations du corps commun de la fiche du joueur.
 *
 * Ce que les fixations couvrent : quatre corps de fiche, chacun dans son fichier de
 * `__fixtures__/` — un joueur complet (feuille lue, compteurs de l'instant), un joueur sans ligne de
 * feuille (le bloc pointillé), un bot avec feuille, un joueur mort à l'image lue (l'encadré « Éliminé »
 * à la place des jauges). Le HTML sérialisé est celui du corps seul (sans conteneur) : tête, actions,
 * bloc « À l'instant », listes de totaux. Elles servent de témoin « avant » à tout lot qui ajoute une section au corps.
 *
 * Aucune valeur ne varie : l'image et l'instant sont fixés par les props, et les identifiants générés
 * par React (`useId`) sont normalisés avant la comparaison, car leur numéro dépend de l'ordre des rendus.
 *
 * Régénération : `npx vitest run <ce fichier> -u`, à la création seulement. Jamais pour faire passer une divergence.
 */
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import { ReplayPlayerPanelBody } from './ReplayPlayerPanelBody'

/** Alpha vit sur [0, 100] ; Echo meurt à l'image 30 et ne revient qu'à 130 ; Marc est un bot vivant. */
const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [
    { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
    { xuid: 'E', filmIndex: 1, name: 'Echo', team: 1 },
    { xuid: '', filmIndex: 2, name: 'Marc [bot]', bot: true, team: 1 },
  ],
  tracks: [
    {
      slot: 512, team: -1, xuid: 'A', startFrame: 0, endFrame: 100,
      points: [{ t: 0, x: 1, y: 1, sh: 1, hp: 1 }, { t: 40, x: 2, y: 2, sh: 0.6, hp: 0.7 }],
    },
    { slot: 513, team: -1, xuid: 'E', startFrame: 0, endFrame: 30, points: [{ t: 0, x: 3, y: 3, sh: 1, hp: 1 }] },
    { slot: 514, team: -1, xuid: 'E', startFrame: 130, endFrame: 200, points: [{ t: 130, x: 3, y: 3, sh: 1, hp: 1 }] },
    { slot: 515, team: -1, xuid: 'bot:Marc', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 4, y: 4, sh: 0.5, hp: 1 }] },
  ],
  weaponLabels: { '0xAAAA': { fr: "Fusil d'assaut", en: 'Assault rifle' }, '0xBBBB': { fr: 'SPNKr', en: 'SPNKr' } },
  loadouts: [
    { t: 0, slot: 512, w: ['0xAAAA', '0xBBBB'] },
    { t: 0, slot: 515, w: ['0xAAAA', '0xBBBB'] },
  ],
  grenadeLabels: [{ fr: 'Fragmentation', en: 'Frag' }, { fr: 'Plasma', en: 'Plasma' }],
  inventory: [
    { t: 0, slot: 512, d: 0, am: [{ mag: 36, res: 198 }, { mag: 2, res: 4 }], g: [2, 0] },
    { t: 0, slot: 515, d: 0, am: [{ mag: 8, res: 24 }, { mag: 3, res: 6 }], g: [0, 0] },
  ],
  scoreTimeline: {
    teams: null,
    players: [
      {
        xuid: 'A',
        score: { rounds: null, total: [{ t: 5, v: 120 }, { t: 40, v: 350 }] },
        kills: { rounds: null, total: [{ t: 5, v: 1 }, { t: 40, v: 3 }] },
        deaths: { rounds: null, total: [{ t: 20, v: 2 }] },
        assists: { rounds: null, total: [{ t: 40, v: 4 }] },
      },
    ],
  } as never,
})
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'A')

/** Une feuille complète : précision, dégâts, détail du score, objectif et armes. */
const FEUILLE: MatchScoreboardRow = scoreboardRow('A', 'Alpha', 'a', {
  kills: 7, deaths: 3, assists: 2, score: 1800,
  accuracy: 42.5, shots_hit: 85, shots_fired: 200,
  damage_dealt: 3457.2, damage_taken: 2100,
  headshot_kills: 4, power_weapon_kills: 2, melee_kills: 1,
  objective: { flag_captures: 2, flag_returns: 0 } as never,
  weapon_kills: [{ weapon_id: 9, label: 'BR75', kills: 6 }, { weapon_id: 12, kills: 1 }],
})

function joueur(xuid: string, board?: MatchScoreboardRow): ReplayPlayer {
  const p = buildPlayers(DOC, []).find((x) => x.xuid === xuid)
  if (!p) throw new Error(`joueur ${xuid} absent du document de test`)
  return { ...p, board }
}

/** Le HTML du corps monté sur ce joueur, à l'image donnée (un dixième de seconde par image). */
function htmlDuCorps(player: ReplayPlayer, frame: number, over: { follow?: boolean } = {}): string {
  const vue = render(
    <ReplayPlayerPanelBody
      player={player}
      actions={{ follow: over.follow ?? false, solo: false, hide: false }}
      onToggle={vi.fn()}
      doc={DOC}
      frame={frame}
      nowMs={frame * 100}
      playWindow={null}
      allegiance={ALLEGIANCE}
      locale="fr"
    />,
  )
  return vue.container.innerHTML.replace(/:r[0-9a-z]+:/g, ':id:')
}

describe('ReplayPlayerPanelBody — fixations du corps', () => {
  it('un joueur complet : feuille lue, compteurs de l’instant, Suivre pressé', async () => {
    const html = htmlDuCorps(joueur('A', FEUILLE), 50, { follow: true })
    expect(html).toContain('aria-label="Alpha"')
    expect(html).toContain('Précision sur le match')
    await expect(html).toMatchFileSnapshot('./__fixtures__/replayPlayerPanelBody.complet.html')
  })

  it('un joueur sans ligne de feuille : le bloc pointillé, jamais une liste de zéros', async () => {
    const html = htmlDuCorps(joueur('A'), 50)
    expect(html).toContain('Pas de statistiques de match pour ce joueur')
    await expect(html).toMatchFileSnapshot('./__fixtures__/replayPlayerPanelBody.sansFeuille.html')
  })

  it('un bot avec feuille : ses listes, sous la clé de roster synthétique', async () => {
    const html = htmlDuCorps(joueur('bot:Marc', scoreboardRow('bid(2.0)', 'Marc', 'b', { accuracy: 30, shots_hit: 3, shots_fired: 10 })), 50)
    expect(html).toContain('aria-label="Marc"')
    await expect(html).toMatchFileSnapshot('./__fixtures__/replayPlayerPanelBody.bot.html')
  })

  it('un joueur mort à l’image lue : l’encadré « Éliminé » à la place des jauges', async () => {
    const html = htmlDuCorps(joueur('E', scoreboardRow('E', 'Echo', 'b', { accuracy: 20, shots_hit: 2, shots_fired: 10 })), 50)
    expect(html).toContain('aria-label="Echo"')
    await expect(html).toMatchFileSnapshot('./__fixtures__/replayPlayerPanelBody.mort.html')
  })
})
