/**
 * Kit des tests de LA RANGÉE DES ÉQUIPES dans la vraie page (`ReplayPage.row*.test.tsx`) : la largeur du corps du
 * cadre est posée par le test (jsdom ne met rien en page) et suivie par un observateur de taille simulé ; un match
 * à `nbCamps` camps de `parCamp` places avec calque de score ; les sélecteurs de la rangée, de la colonne et des
 * bandeaux de score. Chaque fichier de test pose lui-même le `vi.mock` du routeur (il est hissé par fichier) et
 * appelle `installerLesDoublesDeLaRangee` dans son `beforeEach`.
 */
import { act } from '@testing-library/react'
import { vi } from 'vitest'

import { largeurRequiseEnteteScore } from '../model/cotesEnteteScore'
import { largeurRequiseRangee, largeurRequiseRangeeGrandeEquipe } from '../model/rowLayout'
import { mountReplayPage, stubReplayPage } from './replayPageHarness'
import { scoreboardRow } from './scoreboardRow'
import { testReplayDoc } from './testDoc'

/** La largeur du corps du cadre que le test pose ; 0 = non mesurée. */
let largeurCorps = 0
/** Les observateurs de taille que la page a posés sur un corps de cadre. */
const observateurs: { cible: Element; rappel: () => void }[] = []

class ObservateurDeTaille {
  private readonly rappel: () => void
  constructor(rappel: () => void) {
    this.rappel = rappel
  }
  observe(cible: Element) {
    observateurs.push({ cible, rappel: this.rappel })
  }
  unobserve() {}
  disconnect() {
    for (let i = observateurs.length - 1; i >= 0; i--) if (observateurs[i].rappel === this.rappel) observateurs.splice(i, 1)
  }
}

/** La fenêtre change de taille : le corps du cadre prend cette largeur et ses observateurs sont avertis. */
export function redimensionner(largeur: number) {
  largeurCorps = largeur
  act(() => {
    for (const { cible, rappel } of [...observateurs]) if (cible.hasAttribute('data-replay-body')) rappel()
  })
}

/** À appeler dans le `beforeEach` du fichier de test : remet à zéro la largeur et les observateurs, pose les doubles. */
export function installerLesDoublesDeLaRangee() {
  window.localStorage.clear()
  largeurCorps = 0
  observateurs.length = 0
  stubReplayPage()
  vi.stubGlobal('ResizeObserver', ObservateurDeTaille)
  const original = HTMLElement.prototype.getBoundingClientRect
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
    if (!this.hasAttribute('data-replay-body')) return original.call(this)
    return { width: largeurCorps, height: 0, top: 0, left: 0, right: largeurCorps, bottom: 0, x: 0, y: 0, toJSON: () => ({}) }
  })
}

/** Une élimination du joueur 0 sur le joueur 5 à `secondes` : le fil de la base, que la page recale sur le film. */
export function elimination(secondes: number) {
  return { event_type: 'kill', actor_xuid: 'X0', victim_xuid: 'X5', victim_gamertag: 'Joueur5', event_time_ms: secondes * 1_000 }
}

/**
 * Un match à `nbCamps` camps de `parCamp` places (équipe 0, 1, ...), avec calque de score pour les deux premiers.
 * `highlights` : les événements de la base (les éliminations du fil).
 */
export function match(parCamp: number, nbCamps = 2, highlights: object[] = []) {
  const n = parCamp * nbCamps
  const equipe = (i: number) => (nbCamps === 1 ? -1 : Math.floor(i / parCamp))
  const serie = (teamId: number, v: number) => {
    const points = [{ t: 0, v: 0 }, { t: 10, v }]
    return { teamId, rounds: [{ round: 0, points }], total: points }
  }
  const doc = testReplayDoc({
    frameCount: 300,
    frameIntervalMs: 100,
    originMs: 0,
    roster: Array.from({ length: n }, (_, i) => ({
      xuid: `X${i}`, filmIndex: i, seat: i % parCamp, name: `Joueur${i}`, team: equipe(i),
    })),
    tracks: Array.from({ length: n }, (_, i) => ({
      slot: 512 + i, team: equipe(i), xuid: `X${i}`, startFrame: 0, endFrame: 290, points: [{ t: 0, x: i, y: i, sh: 1, hp: 1 }],
    })),
    ...(nbCamps === 2 ? { scoreTimeline: { teams: [serie(0, 7), serie(1, 3)], players: [] } } : {}),
  })
  const scoreboard = Array.from({ length: n }, (_, i) =>
    scoreboardRow(`X${i}`, `Joueur${i}`, equipe(i) === 0 ? 'a' : 'b', { is_me: i === 0 }),
  )
  const context = {
    header: { map_ui: 'Detachment', mode_ui: 'Assassin en équipe', playlist_label: 'Partie rapide', start_time_label: '10 nov. 2026' },
    scoreboard,
    highlight_events: highlights,
    media: null,
    source_unavailable: false,
  }
  return { doc, context }
}

export async function mount({ doc, context }: ReturnType<typeof match>, largeur: number) {
  largeurCorps = largeur
  return mountReplayPage(doc, context)
}

export const rangee = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-teams-row]')
export const colonneDesEquipes = (c: HTMLElement) => c.querySelector<HTMLElement>('[data-replay-roster-column]')
export const bandeaux = (c: HTMLElement) => [...c.querySelectorAll<HTMLElement>('[data-replay-score-banner]')]
/** Les largeurs demandées AU CINÉMA (marge de corps de 24 px) ; le Plein cadre (4 px) en demande 40 de moins. */
export const L4 = largeurRequiseRangee(4, 'cinema')
export const LGE = largeurRequiseRangeeGrandeEquipe('cinema')
export const LARGEUR_REQUISE_ENTETE_SCORE_PX = largeurRequiseEnteteScore('cinema')