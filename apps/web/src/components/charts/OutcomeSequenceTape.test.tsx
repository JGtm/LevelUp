import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { OutcomeSequenceTape, type OutcomeSequenceLabels } from './OutcomeSequenceTape'
import {
  asDominance,
  clusterDominanceNotches,
  matchIndexAtX,
  notchCoreWidth,
  notchGutterWidth,
  toRuns,
  type DominanceValue,
  type OutcomePoint,
  type Run,
} from './outcomeSequence'

// Mock du wrapper ECharts (canvas absent en jsdom) : on capture les props passées
// pour vérifier le câblage `onEvents`/`onChartReady` selon la présence de la prop.
const captured: Array<Record<string, unknown>> = []
vi.mock('echarts-for-react', () => ({
  default: (props: Record<string, unknown>) => {
    captured.push(props)
    return <div data-testid="tape-stub" />
  },
}))

const labels: OutcomeSequenceLabels = { win: 'V', loss: 'D', tie: 'N', dnf: 'DNF' }

function pt(matchId: string, outcome: OutcomePoint['outcome'] = 'win'): OutcomePoint {
  return { outcome, matchId }
}

describe('matchIndexAtX', () => {
  const runs: Run[] = [
    { outcome: 'win', count: 2, matches: [pt('m1'), pt('m2')] },
    { outcome: 'loss', count: 1, matches: [pt('m3', 'loss')] },
  ]

  it('résout le match par index global (offset intra-run)', () => {
    expect(matchIndexAtX(runs, 0.5)?.matchId).toBe('m1')
    expect(matchIndexAtX(runs, 1.9)?.matchId).toBe('m2')
    expect(matchIndexAtX(runs, 2.1)?.matchId).toBe('m3')
  })

  it('borne les X hors limites (négatif → premier, ≥ xMax → dernier)', () => {
    expect(matchIndexAtX(runs, -5)?.matchId).toBe('m1')
    expect(matchIndexAtX(runs, 99)?.matchId).toBe('m3')
    // xMax exact (3) est hors [0, total-1] → borné au dernier.
    expect(matchIndexAtX(runs, 3)?.matchId).toBe('m3')
  })

  it('renvoie null sur runs vides', () => {
    expect(matchIndexAtX([], 0)).toBeNull()
  })

  it('gère un run unique', () => {
    const single: Run[] = [{ outcome: 'win', count: 1, matches: [pt('only')] }]
    expect(matchIndexAtX(single, 0)?.matchId).toBe('only')
    expect(matchIndexAtX(single, 42)?.matchId).toBe('only')
  })
})

describe('OutcomeSequenceTape — non-régression sans prop', () => {
  beforeEach(() => {
    captured.length = 0
  })

  it("ne câble aucun onEvents/onChartReady quand onMatchClick est absent", async () => {
    render(<OutcomeSequenceTape matches={[pt('a'), pt('b', 'loss')]} labels={labels} />)
    await screen.findByTestId('tape-stub')
    const props = captured[captured.length - 1]
    expect(props.onEvents).toBeUndefined()
    expect(props.onChartReady).toBeUndefined()
  })

  it('câble onEvents.click + onChartReady quand onMatchClick est fourni', async () => {
    render(
      <OutcomeSequenceTape matches={[pt('a'), pt('b', 'loss')]} labels={labels} onMatchClick={vi.fn()} />,
    )
    await screen.findByTestId('tape-stub')
    const props = captured[captured.length - 1]
    expect(props.onEvents).toBeDefined()
    expect((props.onEvents as { click?: unknown }).click).toBeTypeOf('function')
    expect(props.onChartReady).toBeTypeOf('function')
  })
})

// GAP #1 (glue non testée) : la résolution du match cliqué dans handleClick — le
// chemin nominal convertFromPixel→matchIndexAtX ET le chemin dégradé (dataIndex)
// quand la conversion pixel est indisponible ou lève. Un bug dans le fallback
// (mauvais dataIndex, mauvais matchId, ou absence de garde Number.isFinite)
// partirait en prod sans être attrapé — matchIndexAtX seul ne le couvre pas.
describe('OutcomeSequenceTape — handleClick (résolution du match)', () => {
  // runs déduits : win×2 [m1,m2], loss×1 [m3].
  const clickMatches = [pt('m1'), pt('m2'), pt('m3', 'loss')]

  function setup() {
    captured.length = 0
    const onMatchClick = vi.fn()
    render(<OutcomeSequenceTape matches={clickMatches} labels={labels} onMatchClick={onMatchClick} />)
    const props = captured[captured.length - 1]
    const click = (props.onEvents as { click: (p: unknown) => void }).click
    const onChartReady = props.onChartReady as (chart: unknown) => void
    return { onMatchClick, click, onChartReady }
  }

  it('résout via convertFromPixel→matchIndexAtX (chemin nominal)', async () => {
    const { onMatchClick, click, onChartReady } = setup()
    await screen.findByTestId('tape-stub')
    // Chart dont convertFromPixel renvoie X=1.9 → index global 1 → m2.
    onChartReady({ convertFromPixel: () => [1.9, 0] })
    click({ dataIndex: 0, event: { offsetX: 50, offsetY: 10 } })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m2')
  })

  it('retombe sur runs[dataIndex].matches[0] quand convertFromPixel lève', async () => {
    const { onMatchClick, click, onChartReady } = setup()
    await screen.findByTestId('tape-stub')
    onChartReady({
      convertFromPixel: () => {
        throw new Error('boom')
      },
    })
    // dataIndex 1 = run loss → premier match m3 (chemin dégradé).
    click({ dataIndex: 1, event: { offsetX: 5, offsetY: 5 } })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m3')
  })

  it('retombe sur dataIndex quand convertFromPixel renvoie une valeur non finie', async () => {
    const { onMatchClick, click, onChartReady } = setup()
    await screen.findByTestId('tape-stub')
    onChartReady({ convertFromPixel: () => [Number.NaN, 0] })
    click({ dataIndex: 0, event: { offsetX: 5, offsetY: 5 } })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m1')
  })

  it('utilise le fallback dataIndex sans chart prêt (offset absent)', async () => {
    const { onMatchClick, click } = setup()
    await screen.findByTestId('tape-stub')
    // onChartReady jamais appelé (chartRef null) + pas d'event → fallback pur.
    click({ dataIndex: 1 })
    expect(onMatchClick).toHaveBeenCalledExactlyOnceWith('m3')
  })

  it("n'émet rien quand ni la conversion ni le dataIndex ne résolvent", async () => {
    const { onMatchClick, click } = setup()
    await screen.findByTestId('tape-stub')
    click({}) // pas de dataIndex, pas d'event
    expect(onMatchClick).not.toHaveBeenCalled()
  })
})

// ─── Encoches de dominance (v7.3 lot 2, item 3.5) ───────────────────────────
//
// Le drapeau est OPTIONNEL : les consommateurs qui ne le fournissent pas doivent
// garder un rendu strictement identique (même exigence que `onMatchClick`).

function flagged(i: number, dominance?: DominanceValue): OutcomePoint {
  return { outcome: 'win', matchId: `m${i}`, dominance }
}

describe('encoches de dominance — géométrie (helpers purs)', () => {
  it('cœur = perMatchW × 0,5, borné à [1,8 ; 5] px', () => {
    expect(notchCoreWidth(6)).toBe(3) // facteur 0,5 dans la bande utile
    expect(notchCoreWidth(8)).toBe(4)
    expect(notchCoreWidth(3.6)).toBe(1.8) // pile sur le plancher
    expect(notchCoreWidth(2)).toBe(1.8) // sous le plancher → écrêté
    expect(notchCoreWidth(0)).toBe(1.8)
    expect(notchCoreWidth(10)).toBe(5) // pile sur le plafond
    expect(notchCoreWidth(400)).toBe(5) // au-dessus → écrêté
  })

  it('gouttière 1 px, amincie à 0,75 px sous 4 px/match', () => {
    expect(notchGutterWidth(20)).toBe(1)
    expect(notchGutterWidth(4)).toBe(1) // borne inclusive
    expect(notchGutterWidth(3.9)).toBe(0.75)
    expect(notchGutterWidth(0.5)).toBe(0.75)
  })
})

describe('clusterDominanceNotches — fusion des encoches voisines', () => {
  it('aucun drapeau → aucune encoche', () => {
    expect(clusterDominanceNotches([flagged(0), flagged(1)], 20)).toEqual([])
  })

  it('densité confortable → une encoche par match, drapeau conservé', () => {
    const notches = clusterDominanceNotches([flagged(0, 1), flagged(1), flagged(2, 3)], 20)
    expect(notches).toEqual([
      { center: 0.5, flag: 1, size: 1 },
      { center: 2.5, flag: 3, size: 1 },
    ])
  })

  it('pas de fusion juste au-dessus du seuil (2,1 px/match)', () => {
    const notches = clusterDominanceNotches([flagged(0, 1), flagged(1, 1)], 2.1)
    expect(notches).toHaveLength(2)
  })

  it('fusion sous ~2 px/match, MÊME drapeau → couleur conservée', () => {
    const notches = clusterDominanceNotches([flagged(0, 3), flagged(1, 3)], 2)
    expect(notches).toEqual([{ center: 1, flag: 3, size: 2 }])
  })

  it('fusion de drapeaux MÊLÉS → neutre (flag null), aucun drapeau élu au hasard', () => {
    const notches = clusterDominanceNotches([flagged(0, 1), flagged(1, 4)], 1)
    expect(notches).toEqual([{ center: 1, flag: null, size: 2 }])
  })

  it('deux groupes assez éloignés restent deux encoches distinctes', () => {
    const matches = [flagged(0, 1), flagged(1, 1), flagged(2), flagged(3), flagged(4), flagged(5, 2)]
    // Empreinte à 1 px/match = 1,8 + 2×0,75 = 3,3 px ; l'écart 1→5 vaut 4 px.
    const notches = clusterDominanceNotches(matches, 1)
    expect(notches).toEqual([
      { center: 1, flag: 1, size: 2 },
      { center: 5.5, flag: 2, size: 1 },
    ])
  })

  it('densité dégénérée (0 px/match) → tout fusionne, pas de division par zéro', () => {
    // 10 matchs, drapeaux aux extrémités : à 0 px/match ils occupent le même pixel.
    const matches = Array.from({ length: 10 }, (_, i) =>
      flagged(i, i === 0 || i === 9 ? 1 : undefined),
    )
    expect(clusterDominanceNotches(matches, 0)).toEqual([{ center: 5, flag: 1, size: 2 }])
  })
})

type RenderChild = { type: string; shape?: Record<string, number>; style?: Record<string, unknown> }

function renderChildren(
  matches: OutcomePoint[],
  opts: { pxPerMatch: number; dominanceLabels?: Record<number, string> },
): RenderChild[] {
  captured.length = 0
  render(
    <OutcomeSequenceTape
      matches={matches}
      labels={labels}
      dominanceLabels={opts.dominanceLabels}
    />,
  )
  const props = captured[captured.length - 1]
  const option = props.option as {
    series: Array<{ renderItem: (p: unknown, a: unknown) => { children: RenderChild[] } }>
  }
  const api = { coord: (v: number[]) => [v[0] * opts.pxPerMatch, 50] }
  return option.series[0].renderItem({ dataIndex: 0 }, api).children
}

describe('OutcomeSequenceTape — dominance', () => {
  beforeEach(() => {
    captured.length = 0
  })

  it('toRuns propage le drapeau tel quel', () => {
    const runs = toRuns([
      { outcome: 'win', matchId: 'm1', dominance: 3 },
      { outcome: 'win', matchId: 'm2' },
    ])
    expect(runs).toHaveLength(1)
    expect(runs[0].matches[0].dominance).toBe(3)
    expect(runs[0].matches[1].dominance).toBeUndefined()
  })

  it('asDominance ne retient que 1..5 (0, null et codes inconnus → undefined)', () => {
    expect(asDominance(3)).toBe(3)
    expect(asDominance(0)).toBeUndefined()
    expect(asDominance(null)).toBeUndefined()
    expect(asDominance(undefined)).toBeUndefined()
    expect(asDominance(9)).toBeUndefined()
  })

  // La bande elle-même est le PREMIER rect ; chaque encoche en ajoute deux
  // (gouttière puis cœur). Aucun `polygon` n'est plus produit (fin des losanges).
  const notchRects = (children: RenderChild[]) => children.slice(1).filter((c) => c.type === 'rect')

  it('dessine une gouttière + un cœur par match porteur d’un drapeau', () => {
    const children = renderChildren([flagged(1, 1), flagged(2), flagged(3, 3)], { pxPerMatch: 20 })
    expect(children.filter((c) => c.type === 'polygon')).toHaveLength(0)
    const rects = notchRects(children)
    expect(rects).toHaveLength(4)
    // Gouttière (1 px de chaque côté à 20 px/match) plus large que le cœur (5 px).
    expect(rects[0].shape?.width).toBe(7)
    expect(rects[1].shape?.width).toBe(5)
    // Encoche TRAVERSANTE : 14 px de bande + 2 × 3 px de dépassement.
    expect(rects[0].shape?.height).toBe(20)
    expect(rects[1].shape?.height).toBe(20)
    // Gouttière et cœur sont concentriques.
    const gutterMid = (rects[0].shape as { x: number; width: number })
    const coreMid = (rects[1].shape as { x: number; width: number })
    expect(gutterMid.x + gutterMid.width / 2).toBeCloseTo(coreMid.x + coreMid.width / 2, 6)
  })

  it('le seuil dur de 6 px/match a DISPARU : encoche dessinée à 4 px comme à 1 px', () => {
    const dense = renderChildren([flagged(1, 1), flagged(2), flagged(3)], { pxPerMatch: 4 })
    expect(notchRects(dense)).toHaveLength(2)
    const denser = renderChildren([flagged(1, 1), flagged(2), flagged(3)], { pxPerMatch: 1 })
    expect(notchRects(denser)).toHaveLength(2)
  })

  it('bande très serrée : deux drapeaux voisins fusionnent en UNE encoche', () => {
    const children = renderChildren([flagged(1, 1), flagged(2, 1), flagged(3)], { pxPerMatch: 1 })
    expect(notchRects(children)).toHaveLength(2)
  })

  it('consommateur SANS drapeau → aucune encoche (non-régression)', () => {
    const children = renderChildren([pt('a'), pt('b')], { pxPerMatch: 20 })
    expect(notchRects(children)).toHaveLength(0)
    expect(children.filter((c) => c.type === 'polygon')).toHaveLength(0)
  })

  it('tooltip : suffixe le match avec le libellé du drapeau quand il est fourni', () => {
    captured.length = 0
    render(
      <OutcomeSequenceTape
        matches={[{ outcome: 'win', matchId: 'm1', map: 'Aquarius', dominance: 3 }]}
        labels={labels}
        dominanceLabels={{ 3: 'Remontada' }}
      />,
    )
    const props = captured[captured.length - 1]
    const option = props.option as {
      series: Array<{ data: Array<{ run: Run }> }>
      tooltip: { formatter: (p: unknown) => string }
    }
    const html = option.tooltip.formatter({ data: option.series[0].data[0] })
    expect(html).toContain('Remontada')
  })

  it('tooltip : sans libellé fourni, la ligne du match reste inchangée', () => {
    captured.length = 0
    render(
      <OutcomeSequenceTape
        matches={[{ outcome: 'win', matchId: 'm1', map: 'Aquarius', dominance: 3 }]}
        labels={labels}
      />,
    )
    const props = captured[captured.length - 1]
    const option = props.option as {
      series: Array<{ data: Array<{ run: Run }> }>
      tooltip: { formatter: (p: unknown) => string }
    }
    const html = option.tooltip.formatter({ data: option.series[0].data[0] })
    expect(html).toContain('Aquarius')
    expect(html).not.toContain('—')
  })
})
