/**
 * ReplayPlayerPanelBody.fixation.test.tsx — fixations du corps commun de la fiche du joueur.
 *
 * Ce que les fixations couvrent : cinq corps de fiche, chacun dans son fichier de `__fixtures__/` — un joueur
 * complet (feuille lue, compteurs de l'instant), un joueur sans ligne de feuille, un bot avec feuille, un joueur mort
 * à l'image lue (l'encadré « Éliminé » à la place des jauges), et le joueur enrichi du match de `instantFixture`
 * (toutes les sections, à la dernière image). Le HTML sérialisé est celui du corps seul (sans conteneur) : identité,
 * actions, bloc « À l'instant », sections.
 *
 * LE BLOC « À L'INSTANT » A SA PROPRE FIXATION (`replayPlayerNow.<cas>.html`) : son sous-arbre, extrait du corps, doit
 * rester identique à l'octet quel que soit ce que la fiche ajoute autour de lui.
 *
 * Aucune valeur ne varie : l'image et l'instant sont fixés par les props, et les identifiants générés par React
 * (`useId`) sont normalisés avant la comparaison, car leur numéro dépend de l'ordre des rendus.
 *
 * Régénération : `npx vitest run <ce fichier> -u`, pour les fixations du corps seulement. Jamais pour faire passer une
 * divergence, et jamais pour celles du bloc « À l'instant ».
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import { buildPlayers, type ReplayPlayer } from '@/lib/replay/rosterLogic'

import {
  INSTANT_ICONS, INSTANT_LAST_FRAME, instantDoc, instantFeed, instantPlayer, instantPlayers,
} from '../test/instantFixture'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'
import { createReplayAntagonists } from '../model/replayAntagonists'
import { ReplayPlayerPanelBody } from './ReplayPlayerPanelBody'

/** Alpha vit sur [0, 100] ; Echo meurt à l'image 30 et ne revient qu'à 130 ; Marc est un bot vivant. */
const DOC = testReplayDoc({
  frameIntervalMs: 100,
  coverage: { medals: { decoded: 0, published: 0, noOrigin: 0, beforeOrigin: 0, afterEnd: 0, unknownName: 0 } } as never,
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

/** Une feuille complète : précision, dégâts, score, éliminations à la tête. */
const FEUILLE: MatchScoreboardRow = scoreboardRow('A', 'Alpha', 'a', {
  kills: 7, deaths: 3, assists: 2, score: 1800,
  accuracy: 42.5, shots_hit: 85, shots_fired: 200,
  damage_dealt: 3457.2, damage_taken: 2100,
  headshot_kills: 4, power_weapon_kills: 2, melee_kills: 1,
  objective: { flag_captures: 2, flag_returns: 0 } as never,
  weapon_kills: [{ weapon_id: 9, label: 'BR75', kills: 6 }, { weapon_id: 12, kills: 1 }],
})

function joueur(xuid: string, board?: MatchScoreboardRow): { player: ReplayPlayer; players: ReplayPlayer[] } {
  const base = buildPlayers(DOC, [])
  const p = base.find((x) => x.xuid === xuid)
  if (!p) throw new Error(`joueur ${xuid} absent du document de test`)
  const player = { ...p, board }
  return { player, players: base.map((x) => (x.xuid === xuid ? player : x)) }
}

/** Le HTML du corps monté sur ce joueur, à l'image donnée (un dixième de seconde par image). */
function htmlDuCorps(who: { player: ReplayPlayer; players: ReplayPlayer[] }, frame: number, over: { solo?: boolean } = {}): string {
  const vue = render(
    <ReplayPlayerPanelBody
      player={who.player}
      players={who.players}
      feed={[]}
      weaponIcons={new Map()}
      antagonists={createReplayAntagonists([], [])}
      solo={over.solo ?? false}
      onSolo={vi.fn()}
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

/** Le sous-arbre du bloc « À l'instant », sans section imbriquée : de son ouverture à sa première fermeture. */
function blocALInstant(html: string): string {
  const m = /<section aria-label="À l'instant" data-replay-player-now="".*?<\/section>/s.exec(html)
  if (!m) throw new Error("le bloc « À l'instant » est absent du corps")
  return m[0]
}

const fixture = (nom: string) => `./__fixtures__/${nom}.html`
const temoinDuBloc = (cas: string) => readFileSync(resolve(__dirname, '__fixtures__', `replayPlayerNow.${cas}.html`), 'utf8')

describe('ReplayPlayerPanelBody — fixations du corps', () => {
  it('un joueur complet : feuille lue, compteurs de l’instant, Seul pressé', async () => {
    const html = htmlDuCorps(joueur('A', FEUILLE), 50, { solo: true })
    expect(html).toContain('aria-label="Alpha"')
    expect(html).toContain('Totaux du match')
    expect(blocALInstant(html)).toBe(temoinDuBloc('complet'))
    await expect(html).toMatchFileSnapshot(fixture('replayPlayerPanelBody.complet'))
  })

  it('un joueur sans ligne de feuille : pas de totaux, jamais une liste de zéros', async () => {
    const html = htmlDuCorps(joueur('A'), 50)
    expect(html).not.toContain('Totaux du match')
    expect(html).not.toContain('Pas de statistiques de match pour ce joueur')
    expect(blocALInstant(html)).toBe(temoinDuBloc('sansFeuille'))
    await expect(html).toMatchFileSnapshot(fixture('replayPlayerPanelBody.sansFeuille'))
  })

  it('un bot avec feuille : ses totaux, sous la clé de roster synthétique', async () => {
    const html = htmlDuCorps(joueur('bot:Marc', scoreboardRow('bid(2.0)', 'Marc', 'b', { accuracy: 30, shots_hit: 3, shots_fired: 10 })), 50)
    expect(html).toContain('aria-label="Marc"')
    expect(blocALInstant(html)).toBe(temoinDuBloc('bot'))
    await expect(html).toMatchFileSnapshot(fixture('replayPlayerPanelBody.bot'))
  })

  it('un joueur mort à l’image lue : l’encadré « Éliminé » à la place des jauges', async () => {
    const html = htmlDuCorps(joueur('E', scoreboardRow('E', 'Echo', 'b', { accuracy: 20, shots_hit: 2, shots_fired: 10 })), 50)
    expect(html).toContain('aria-label="Echo"')
    expect(blocALInstant(html)).toBe(temoinDuBloc('mort'))
    await expect(html).toMatchFileSnapshot(fixture('replayPlayerPanelBody.mort'))
  })

  it('le joueur enrichi du match de la fixture : toutes les sections, à la dernière image', async () => {
    const doc = instantDoc()
    const players = instantPlayers(doc)
    const vue = render(
      <ReplayPlayerPanelBody
        player={instantPlayer(players, 'A')}
        players={players}
        feed={instantFeed()}
        weaponIcons={INSTANT_ICONS}
        antagonists={createReplayAntagonists([], [])}
        solo={false}
        onSolo={vi.fn()}
        doc={doc}
        frame={INSTANT_LAST_FRAME}
        nowMs={INSTANT_LAST_FRAME * 100}
        playWindow={null}
        allegiance={filmAllegianceOf(doc, [], 'A')}
        locale="fr"
      />,
    )
    const html = vue.container.innerHTML.replace(/:r[0-9a-z]+:/g, ':id:')
    expect(html).toContain('Éliminations par arme')
    await expect(html).toMatchFileSnapshot(fixture('replayPlayerPanelBody.enrichi'))
  })
})
