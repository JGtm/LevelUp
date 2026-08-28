/**
 * Tests — ReplayRoundBreakOverlay (le message inter-manche « Manche N terminée »).
 *
 * CE QU'ILS PROTÈGENT : le message paraît DANS SA FENÊTRE autour de la bascule et nulle part
 * ailleurs, il nomme la manche qui vient de se terminer, il se traduit, et il obéit à la garde
 * d'horloge du calque de score comme le bandeau et l'écran de fin.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import { normalizeScoreTimeline } from '@/lib/replay/scoreTimeline'

import { ReplayRoundBreakOverlay, ROUND_BREAK_WINDOW_MS } from './ReplayRoundBreakOverlay'
import type { ReplayDocumentReady } from './replayNormalize'

/** Une manche : numéro et paliers `[frame, valeur]`. */
function manche(round: number, points: Array<[number, number]>) {
  return { round, points: points.map(([t, v]) => ({ t, v })) }
}

/** Oddball à deux manches : t0 gagne 100/78 puis 100/43. Bascule de manche au frame 100. */
const ODDBALL_TEAMS = [
  {
    teamId: 0,
    rounds: [manche(0, [[0, 0], [50, 100]]), manche(1, [[100, 0], [150, 100]])],
    total: [[0, 0], [50, 100], [150, 200]].map(([t, v]) => ({ t, v })),
  },
  {
    teamId: 1,
    rounds: [manche(0, [[0, 0], [50, 78]]), manche(1, [[100, 0], [150, 43]])],
    total: [[0, 0], [50, 78], [150, 121]].map(([t, v]) => ({ t, v })),
  },
]

/**
 * Un document réduit à ce que l'overlay lui demande : le calque de score, l'origine (garde
 * d'horloge) et la cadence (fenêtre en frames). `frameIntervalMs: 100` → 10 fps, donc la
 * fenêtre de 3000 ms fait 30 frames.
 */
function docOf(teams: unknown[], extra: Record<string, unknown> = {}): ReplayDocumentReady {
  return {
    frameIntervalMs: 100,
    originMs: 0,
    scoreTimeline: normalizeScoreTimeline({ teams, players: [] } as never),
    ...extra,
  } as unknown as ReplayDocumentReady
}

function renderOverlay(over: Partial<Parameters<typeof ReplayRoundBreakOverlay>[0]> = {}) {
  return render(
    <ReplayRoundBreakOverlay doc={docOf(ODDBALL_TEAMS)} frame={100} locale="fr" {...over} />,
  )
}

describe('ReplayRoundBreakOverlay — quand et quoi', () => {
  it('affiche « Manche 1 terminée » à la bascule', () => {
    renderOverlay({ frame: 100 })
    expect(screen.getByText('Manche 1 terminée')).toBeInTheDocument()
  })

  it('reste affiché dans sa fenêtre (30 frames à 10 fps)', () => {
    renderOverlay({ frame: 129 })
    expect(screen.getByText('Manche 1 terminée')).toBeInTheDocument()
  })

  it('rien avant la bascule', () => {
    const { container } = renderOverlay({ frame: 99 })
    expect(container).toBeEmptyDOMElement()
  })

  it('rien une fois la fenêtre passée', () => {
    const { container } = renderOverlay({ frame: 130 })
    expect(container).toBeEmptyDOMElement()
  })
})

describe('ReplayRoundBreakOverlay — ce qui ne déclenche rien', () => {
  it('mode à manche unique : jamais de message', () => {
    const slayer = [
      { teamId: 0, rounds: [manche(0, [[0, 0], [500, 43]])], total: [{ t: 0, v: 0 }, { t: 500, v: 43 }] },
    ]
    const { container } = renderOverlay({ doc: docOf(slayer), frame: 250 })
    expect(container).toBeEmptyDOMElement()
  })

  it('horloge du film non recalée : rien (garde de scoreTimelineOf)', () => {
    const { container } = renderOverlay({
      doc: docOf(ODDBALL_TEAMS, { originMs: undefined, coverage: { originResolved: false } }),
      frame: 100,
    })
    expect(container).toBeEmptyDOMElement()
  })
})

describe('ReplayRoundBreakOverlay — les deux langues', () => {
  it('dit « Round 1 over » sous la locale EN', () => {
    renderOverlay({ locale: 'en', frame: 100 })
    expect(screen.getByText('Round 1 over')).toBeInTheDocument()
  })
})

describe('ReplayRoundBreakOverlay — la fenêtre est nommée', () => {
  it('expose une durée d\'affichage brève et non nulle', () => {
    expect(ROUND_BREAK_WINDOW_MS).toBeGreaterThan(0)
    expect(ROUND_BREAK_WINDOW_MS).toBeLessThanOrEqual(5000)
  })
})
