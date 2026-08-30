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

/**
 * Oddball à deux manches : t0 gagne 100/78 puis 100/43.
 *
 * LA BASCULE TOMBE AU FRAME 50, la FIN de la manche 1 (correctif du 2026-08-29) — plus au frame
 * 100, qui est le premier point de la manche 2, c'est-à-dire la REPRISE après l'entracte. Sur
 * les témoins réels, cet entracte dure 19 à 34 s : le message paraissait par-dessus la manche
 * suivante, déjà commencée (cf. l'en-tête de `roundsLogic`).
 */
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
    <ReplayRoundBreakOverlay doc={docOf(ODDBALL_TEAMS)} frame={50} locale="fr" {...over} />,
  )
}

describe('ReplayRoundBreakOverlay — quand et quoi', () => {
  it('affiche « Manche 1 terminée » à la FIN de la manche, pas à la reprise', () => {
    renderOverlay({ frame: 50 })
    expect(screen.getByText('Manche 1 terminée')).toBeInTheDocument()
  })

  it('reste affiché dans sa fenêtre (30 frames à 10 fps)', () => {
    renderOverlay({ frame: 79 })
    expect(screen.getByText('Manche 1 terminée')).toBeInTheDocument()
  })

  it('rien avant la bascule', () => {
    const { container } = renderOverlay({ frame: 49 })
    expect(container).toBeEmptyDOMElement()
  })

  // ET LE FRAME 100 EST DEDANS : c'est la REPRISE de la manche 2, l'endroit d'où le message
  // partait avant le 2026-08-29. Ce cas épingle le correctif — s'il repartait de là, la fenêtre
  // du message serait rouverte 5 s (50 frames) après sa fermeture.
  it('rien une fois la fenêtre passée — reprise de la manche suivante comprise', () => {
    const { container } = renderOverlay({ frame: 80 })
    expect(container).toBeEmptyDOMElement()
    expect(renderOverlay({ frame: 100 }).container).toBeEmptyDOMElement()
  })

  it('rien bien après la fenêtre', () => {
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
      frame: 50,
    })
    expect(container).toBeEmptyDOMElement()
  })
})

describe('ReplayRoundBreakOverlay — les deux langues', () => {
  it('dit « Round 1 over » sous la locale EN', () => {
    renderOverlay({ locale: 'en', frame: 50 })
    expect(screen.getByText('Round 1 over')).toBeInTheDocument()
  })
})

describe('ReplayRoundBreakOverlay — la fenêtre est nommée', () => {
  it('expose une durée d\'affichage brève et non nulle', () => {
    expect(ROUND_BREAK_WINDOW_MS).toBeGreaterThan(0)
    expect(ROUND_BREAK_WINDOW_MS).toBeLessThanOrEqual(5000)
  })
})
