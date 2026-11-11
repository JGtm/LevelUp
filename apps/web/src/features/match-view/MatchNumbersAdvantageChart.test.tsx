/**
 * Tests — MatchNumbersAdvantageChart (la supériorité numérique de la vue match).
 *
 * Le COMPTE est éprouvé dans `lib/replay/aliveBalanceLogic.test.ts` ; ici, le COMPOSANT : il
 * est masqué sans film, il dit pourquoi quand le film ne permet pas de tracer, il compte les
 * bots, et il trace sur l'axe de « Score dans le temps » avec les jetons divergents.
 *
 * Le film du témoin : 4 985 images à 100 ms, origine à 12 s, coup d'envoi à 30 s — l'axe du
 * gameplay vaut `image x 100 - 18 000`. Équipe 0 (le joueur) : `me` (vit tout le match) et
 * `amie` (meurt à l'image 1 000). Équipe 1 : `ennemi` (meurt à 2 000) et le bot `Zed` (vit
 * tout le match).
 */
import { describe, expect, it, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { MATCH_VIEW_TEXT } from './i18n'
import { MatchNumbersAdvantageChart } from './MatchNumbersAdvantageChart'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(--ac-${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const options = vi.hoisted(() => [] as Record<string, any>[]) // eslint-disable-line @typescript-eslint/no-explicit-any
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: Record<string, unknown> }) => {
    options.push(option)
    return <div data-testid="echarts-stub" />
  },
}))

// Le calcul est ESPIONNÉ (et reste le vrai) : on compte ses appels pour prouver la mémoïsation.
const compte = vi.hoisted(() => ({ appels: 0 }))
vi.mock('@/lib/replay/aliveBalanceLogic', async (orig) => {
  const reel = await orig<typeof import('@/lib/replay/aliveBalanceLogic')>()
  return {
    ...reel,
    buildAliveBalance: (...a: Parameters<typeof reel.buildAliveBalance>) => {
      compte.appels++
      return reel.buildAliveBalance(...a)
    },
  }
})

const artefact = vi.hoisted(() => ({ current: undefined as unknown, loading: false }))
vi.mock('@/lib/replay/queries', () => ({
  useMatchReplay: () => ({ data: artefact.current, isLoading: artefact.loading }),
}))

const { normalizeReplayDocument } = await import('@/lib/replay/replayNormalize')

const t = MATCH_VIEW_TEXT.fr
const ORIGIN_MS = 12_000
const T0_MS = 30_000
const LAST = 4_984

const SCOREBOARD = [
  { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
  { xuid: 'amie', gamertag: 'Amie', team_side: 't0', is_me: false },
  { xuid: 'ennemi', gamertag: 'Autre', team_side: 't1', is_me: false },
] as unknown as MatchScoreboardRow[]

const ROSTER = [
  { xuid: 'me', team: 0 },
  { xuid: 'amie', team: 0 },
  { xuid: 'ennemi', team: 1 },
  { xuid: '', bot: true, name: 'Zed', team: 1 },
]

const TRACKS = [
  { slot: 0, xuid: 'me', startFrame: 0, endFrame: LAST },
  { slot: 1, xuid: 'amie', startFrame: 0, endFrame: 1_000 },
  { slot: 2, xuid: 'ennemi', startFrame: 0, endFrame: 2_000 },
  { slot: 3, bot: 'Zed', startFrame: 0, endFrame: LAST },
]

function poserArtefact(over: Partial<ReplayDocument> | null) {
  artefact.loading = false
  artefact.current = over
    ? normalizeReplayDocument({
        frameCount: 4985,
        frameIntervalMs: 100,
        originMs: ORIGIN_MS,
        roster: ROSTER,
        tracks: TRACKS.map((tr) => ({ ...tr, points: null })),
        ...over,
      } as unknown as ReplayDocument)
    : undefined
}

function afficher(
  opts: {
    replayAvailable?: boolean
    meXUID?: string | null
    scoreboard?: MatchScoreboardRow[]
    locale?: 'fr' | 'en'
  } = {},
) {
  options.length = 0
  return render(
    <MatchNumbersAdvantageChart
      playerSlug="joueur"
      matchId="m1"
      replayAvailable={opts.replayAvailable ?? true}
      scoreboard={opts.scoreboard ?? SCOREBOARD}
      meXUID={opts.meXUID === undefined ? 'me' : opts.meXUID}
      t0Ms={T0_MS}
      t={MATCH_VIEW_TEXT[opts.locale ?? 'fr']}
    />,
  )
}

async function derniereOption() {
  return waitFor(() => {
    const o = options.at(-1)
    if (!o?.series) throw new Error('option non rendue')
    return o
  })
}

async function aideDuTitre(view: ReturnType<typeof render>): Promise<string> {
  const user = userEvent.setup()
  await user.hover(view.getByRole('button', { name: /informations/i }))
  return view.getByRole('tooltip').textContent ?? ''
}

/** La valeur de l'option (moitié positive + moitié négative) à un instant d'échantillonnage. */
function valeurA(o: Record<string, any>, ms: number): number { // eslint-disable-line @typescript-eslint/no-explicit-any
  const pos = o.series[0].data.find((d: number[]) => d[0] === ms)
  const neg = o.series[1].data.find((d: number[]) => d[0] === ms)
  return pos[1] + neg[1]
}

describe('MatchNumbersAdvantageChart — quand la carte apparaît', () => {
  it('est MASQUÉE sans film : `replay_available` faux, rien à l’écran', () => {
    poserArtefact({})
    expect(afficher({ replayAvailable: false }).container.firstChild).toBeNull()
  })

  it('affiche la carte, son titre neutre et le bouton d’information', async () => {
    poserArtefact({})
    const view = afficher()
    expect(view.getByText(t.numbersAdvantageTitle)).toBeTruthy()
    expect(await aideDuTitre(view)).toBe(t.numbersAdvantageInfo)
  })

  it('montre le chargement pendant la lecture du film', () => {
    poserArtefact(null)
    artefact.loading = true
    expect(afficher().getByTestId('chart-card-loading')).toBeTruthy()
  })
})

describe('MatchNumbersAdvantageChart — quand elle ne peut pas être tracée, elle le dit', () => {
  it('film indisponible malgré `replay_available` : état vide, cause « vies non datées »', () => {
    poserArtefact(null)
    const view = afficher()
    expect(view.getByText(t.numbersAdvantageUnavailableFilm)).toBeTruthy()
    expect(view.queryByTestId('echarts-stub')).toBeNull()
  })

  it('sans origine du film : pas d’horloge, état vide', () => {
    poserArtefact({ originMs: undefined } as never)
    expect(afficher().getByText(t.numbersAdvantageUnavailableFilm)).toBeTruthy()
  })

  it('joueur de la page absent du film : camp non établi, l’état vide le dit', () => {
    poserArtefact({})
    const view = afficher({ meXUID: 'inconnu' })
    expect(view.getByText(t.numbersAdvantageUnavailableCamps)).toBeTruthy()
    expect(view.queryByTestId('echarts-stub')).toBeNull()
  })

  it('sans joueur référent (null) : même état vide', () => {
    poserArtefact({})
    expect(afficher({ meXUID: null }).getByText(t.numbersAdvantageUnavailableCamps)).toBeTruthy()
  })
})

describe('MatchNumbersAdvantageChart — ce que l’option ECharts contient', () => {
  it('compte les BOTS : un bot en vie est un joueur en vie', async () => {
    poserArtefact({})
    afficher()
    const o = await derniereOption()
    // 100 s de gameplay = image 1 180 : `amie` est morte (fin 1 000), `ennemi` et le bot Zed
    // vivent : 1 contre 2 -> -1. Sans le bot, l'écart vaudrait 0.
    expect(valeurA(o, 100_000)).toBe(-1)
    // 300 s = image 3 180 : `ennemi` est mort aussi, reste le bot : 1 contre 1.
    expect(valeurA(o, 300_000)).toBe(0)
    // Au coup d'envoi : 2 contre 2.
    expect(valeurA(o, 0)).toBe(0)
  })

  it('échantillonne chaque seconde, du coup d’envoi à la fin, en escalier', async () => {
    poserArtefact({})
    afficher()
    const o = await derniereOption()
    const xs: number[] = o.series[0].data.map((d: number[]) => d[0])
    expect(xs.slice(0, 3)).toEqual([0, 1_000, 2_000])
    expect(xs.at(-1)).toBe(480_400)
    expect(o.series.every((s: { step: string }) => s.step === 'end')).toBe(true)
    expect(o.xAxis).toMatchObject({ type: 'value', min: 0, max: 480_400 })
  })

  it('UN SEUL axe des ordonnées, des jetons divergents, la légende en bas', async () => {
    poserArtefact({})
    afficher()
    const o = await derniereOption()
    expect(Array.isArray(o.yAxis)).toBe(false)
    expect(o.series[0].areaStyle.color).toBe('var(--ac-divergent-pos)')
    expect(o.series[1].areaStyle.color).toBe('var(--ac-divergent-neg)')
    expect(o.legend.bottom).toBe(0)
    expect(o.legend.data.map((e: { name: string }) => e.name)).toEqual([
      t.numbersAdvantageLegendAlly,
      t.numbersAdvantageLegendEnemy,
    ])
  })

  it('l’infobulle lit l’instant du curseur : effectifs en vie et écart signé', async () => {
    poserArtefact({})
    afficher()
    const o = await derniereOption()
    const html = o.tooltip.formatter([{ axisValue: 100_500 }]) as string
    expect(html).toContain('<b>1:40</b>')
    expect(html).toContain(`${t.combatTeamLabel} : <b>1 en vie</b>`)
    expect(html).toContain(`${t.combatEnemyLabel} : <b>2 en vie</b>`)
    expect(html).toContain(`${t.divergentGapLabel} : <b>−1</b>`)
  })

  it('le calcul est mémoïsé : un rendu de plus avec les mêmes entrées ne le refait pas', async () => {
    poserArtefact({})
    compte.appels = 0
    const view = afficher()
    await derniereOption()
    const avant = compte.appels
    expect(avant).toBeGreaterThan(0)
    view.rerender(
      <MatchNumbersAdvantageChart
        playerSlug="joueur"
        matchId="m1"
        replayAvailable
        scoreboard={SCOREBOARD}
        meXUID="me"
        t0Ms={T0_MS}
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(compte.appels).toBe(avant)
  })
})

describe('MatchNumbersAdvantageChart — plus de deux équipes', () => {
  it('statue : contre la meilleure équipe adverse, et le DIT dans l’aide du titre', async () => {
    poserArtefact({
      roster: [...ROSTER, { xuid: 'tiers', team: 2 }],
      tracks: [...TRACKS, { slot: 4, xuid: 'tiers', startFrame: 0, endFrame: LAST }].map((tr) => ({
        ...tr,
        points: null,
      })),
    } as never)
    const view = afficher()
    expect(await aideDuTitre(view)).toBe(`${t.numbersAdvantageInfo} ${t.divergentMultiTeamNote}`)
  })
})

describe('MatchNumbersAdvantageChart — langue', () => {
  it('EN : titre et légende passent en anglais', async () => {
    poserArtefact({})
    const view = afficher({ locale: 'en' })
    expect(view.getByText(MATCH_VIEW_TEXT.en.numbersAdvantageTitle)).toBeTruthy()
    const o = await derniereOption()
    expect(o.legend.data[0].name).toBe(MATCH_VIEW_TEXT.en.numbersAdvantageLegendAlly)
  })
})
