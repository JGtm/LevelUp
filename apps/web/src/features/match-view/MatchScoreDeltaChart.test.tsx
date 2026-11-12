/**
 * Tests — MatchScoreDeltaChart (l'écart de score de la vue match).
 *
 * CE QU'ILS PROTÈGENT : la carte ne rend RIEN sans film (comme la courbe qu'elle accompagne) ;
 * avec un film qui ne permet pas de tracer l'écart, elle DIT pourquoi (état vide canonique) ;
 * et quand elle trace, l'écart est celui de l'équipe du JOUEUR, sur l'axe des temps de « Score
 * dans le temps », en jetons divergents, avec une infobulle qui lit l'instant du curseur.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

import { MATCH_VIEW_TEXT } from './i18n'
import { MatchScoreDeltaChart } from './MatchScoreDeltaChart'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(--ac-${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

// Un chart ECharts monté en jsdom crashe le canvas : on stubbe le wrapper et on GARDE l'option
// vivante (avec ses fonctions : l'infobulle s'éprouve en l'appelant).
const options = vi.hoisted(() => [] as Record<string, any>[]) // eslint-disable-line @typescript-eslint/no-explicit-any
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: Record<string, unknown> }) => {
    options.push(option)
    return <div data-testid="echarts-stub" />
  },
}))

const artefact = vi.hoisted(() => ({ current: undefined as unknown, loading: false }))
vi.mock('@/lib/replay/queries', () => ({
  useMatchReplay: () => ({ data: artefact.current, isLoading: artefact.loading }),
}))

const { normalizeReplayDocument } = await import('@/lib/replay/replayNormalize')

const t = MATCH_VIEW_TEXT.fr

function feuille(over: Array<Partial<MatchScoreboardRow>> = []): MatchScoreboardRow[] {
  const base = [
    { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
    { xuid: 'ennemi', gamertag: 'Autre', team_side: 't1', is_me: false },
  ]
  return (over.length ? over : base) as unknown as MatchScoreboardRow[]
}

/** Le témoin Slayer réduit : 43/50 sur 4 985 images à 100 ms. */
const SLAYER = {
  teams: [
    { teamId: 0, rounds: null, total: [{ t: 399, v: 1 }, { t: 4886, v: 43 }] },
    { teamId: 1, rounds: null, total: [{ t: 317, v: 2 }, { t: 4908, v: 50 }] },
  ],
  players: null,
}

/** L'image zéro du film tombe 12 s après le début du match, le coup d'envoi 30 s après. */
const ORIGIN_MS = 12_000
const T0_MS = 30_000

function poserArtefact(over: Partial<ReplayDocument> | null) {
  artefact.loading = false
  artefact.current = over
    ? normalizeReplayDocument({
        frameCount: 4985,
        frameIntervalMs: 100,
        originMs: ORIGIN_MS,
        scoreTimeline: SLAYER,
        ...over,
      } as unknown as ReplayDocument)
    : undefined
}

function afficher(
  opts: { scoreboard?: MatchScoreboardRow[]; replayAvailable?: boolean; locale?: 'fr' | 'en' } = {},
) {
  options.length = 0
  return render(
    <MatchScoreDeltaChart
      playerSlug="joueur"
      matchId="m1"
      replayAvailable={opts.replayAvailable ?? true}
      scoreboard={opts.scoreboard ?? feuille()}
      meXUID="me"
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

describe('MatchScoreDeltaChart — quand la carte apparaît', () => {
  it('sans film (`replay_available` faux), la carte reste à l’écran : bloc vide qui dit qu’il faut le rejeu', () => {
    poserArtefact(null) // la requête désactivée ne rend aucune donnée
    const view = afficher({ replayAvailable: false })
    expect(view.getByText(t.scoreDeltaTitle)).toBeTruthy()
    expect(view.getByText(t.divergentNeedsReplay)).toBeTruthy()
    expect(view.queryByTestId('echarts-stub')).toBeNull()
  })

  it('affiche la carte, son titre neutre et le bouton d’information', async () => {
    poserArtefact({})
    const view = afficher()
    expect(view.getByText(t.scoreDeltaTitle)).toBeTruthy()
    expect(await aideDuTitre(view)).toBe(t.scoreDeltaInfo)
    await waitFor(() => expect(view.getByTestId('echarts-stub')).toBeTruthy())
  })

  it('pendant le chargement du film, montre le chargement — pas un état vide trompeur', () => {
    poserArtefact(null)
    artefact.loading = true
    const view = afficher()
    expect(view.getByTestId('chart-card-loading')).toBeTruthy()
    expect(view.queryByText(t.scoreDeltaUnavailableFilm)).toBeNull()
  })
})

describe('MatchScoreDeltaChart — quand l’écart ne peut pas être tracé, la carte le dit', () => {
  it('film absent malgré `replay_available` : l’état vide dit que le score n’est pas daté', () => {
    poserArtefact(null)
    const view = afficher()
    expect(view.getByText(t.scoreDeltaUnavailableFilm)).toBeTruthy()
    expect(view.queryByTestId('echarts-stub')).toBeNull()
  })

  it('film sans calque de score : même état vide', () => {
    poserArtefact({ scoreTimeline: undefined })
    expect(afficher().getByText(t.scoreDeltaUnavailableFilm)).toBeTruthy()
  })

  it('horloge NON SÛRE (origine ni résolue ni publiée) : état vide, jamais une courbe décalée', () => {
    poserArtefact({ originMs: undefined, coverage: { originResolved: false } } as never)
    const view = afficher()
    expect(view.getByText(t.scoreDeltaUnavailableFilm)).toBeTruthy()
    expect(view.queryByTestId('echarts-stub')).toBeNull()
  })

  it('mode SANS compteur de score (`modeSupported` faux) : état vide', () => {
    poserArtefact({ coverage: { score: { modeSupported: false } } } as never)
    expect(afficher().getByText(t.scoreDeltaUnavailableFilm)).toBeTruthy()
  })

  it('CAMP DU JOUEUR NON RÉSOLU : l’état vide le dit, aucune équipe n’est devinée', () => {
    poserArtefact({})
    const sansMoi = feuille([
      { xuid: 'a', gamertag: 'A', team_side: 't0', is_me: false },
      { xuid: 'b', gamertag: 'B', team_side: 't1', is_me: false },
    ] as Array<Partial<MatchScoreboardRow>>)
    const view = afficher({ scoreboard: sansMoi })
    expect(view.getByText(t.scoreDeltaUnavailableCamps)).toBeTruthy()
    expect(view.queryByTestId('echarts-stub')).toBeNull()
  })

  it('une série du calque sans camp (`teamIdentity` non résolue) : état vide', () => {
    poserArtefact({
      scoreTimeline: {
        teams: [
          { teamId: 0, rounds: null, total: [{ t: 399, v: 1 }] },
          { rounds: null, total: [{ t: 500, v: 1 }] },
        ],
        players: null,
      },
    } as never)
    expect(afficher().getByText(t.scoreDeltaUnavailableCamps)).toBeTruthy()
  })
})

describe('MatchScoreDeltaChart — ce que l’option ECharts contient', () => {
  it('trace l’écart de l’équipe du JOUEUR, du coup d’envoi à la fin, en escalier', async () => {
    poserArtefact({})
    afficher()
    const o = await derniereOption()
    // Équipe 0 = le joueur : 1 puis 43 contre 2 puis 50. Paliers datés comme « Score dans le
    // temps » (image x 100 - 18 000) : 13 700 (équipe 1 : 2), 21 900 (équipe 0 : 1), 470 600
    // (équipe 0 : 43), 472 800 (équipe 1 : 50), fin 480 400.
    expect(o.series[0].data).toEqual([
      [0, 0], [13_700, 0], [21_900, 0], [470_600, 41], [472_800, 0], [480_400, 0],
    ])
    expect(o.series[1].data).toEqual([
      [0, 0], [13_700, -2], [21_900, -1], [470_600, 0], [472_800, -7], [480_400, -7],
    ])
    expect(o.series.every((s: { step: string }) => s.step === 'end')).toBe(true)
    expect(o.xAxis).toMatchObject({ type: 'value', min: 0, max: 480_400 })
  })

  it('UN SEUL axe des ordonnées, les couleurs d’équipe de la courbe, la légende en bas aux noms d’équipe', async () => {
    poserArtefact({})
    afficher()
    const o = await derniereOption()
    expect(Array.isArray(o.yAxis)).toBe(false)
    expect(o.series[0].areaStyle.color).toBe('var(--ac-team-ally)')
    expect(o.series[1].areaStyle.color).toBe('var(--ac-team-enemy)')
    expect(o.legend.bottom).toBe(0)
    // Les noms sont ceux de « Score dans le temps » : l'équipe du joueur d'abord (Eagle = t0).
    const noms = o.legend.data.map((e: { name: string }) => e.name)
    expect(noms[0]).toContain('Eagle')
    expect(noms[1]).toContain('Cobra')
  })

  it('l’infobulle lit l’instant du curseur : score des deux camps et écart signé', async () => {
    poserArtefact({})
    afficher()
    const o = await derniereOption()
    // 1:00 de gameplay : équipe 0 a 1, équipe 1 a 2 -> -1. (Le point tenu est celui de 21 900.)
    const html = o.tooltip.formatter([{ axisValue: 60_000 }]) as string
    expect(html).toContain('<b>1:00</b>')
    expect(html).toContain('<b>1</b>')
    expect(html).toContain('<b>2</b>')
    expect(html).toContain(`${t.divergentGapLabel} : <b>−1</b>`)
    // Le camp qui mène passe devant : +41.
    expect(o.tooltip.formatter([{ axisValue: 471_000 }])).toContain('<b>+41</b>')
  })

  it('échappe le nom d’équipe de la donnée dans l’infobulle', async () => {
    poserArtefact({})
    afficher({
      scoreboard: feuille([
        { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true, team_name: '<img src=x>' },
        { xuid: 'ennemi', gamertag: 'Autre', team_side: 't1', is_me: false },
      ] as Array<Partial<MatchScoreboardRow>>),
    })
    const o = await derniereOption()
    const html = o.tooltip.formatter([{ axisValue: 60_000 }]) as string
    expect(html).not.toContain('<img')
    expect(html).toContain('&lt;img src=x&gt;')
  })
})

describe('MatchScoreDeltaChart — plus de deux équipes', () => {
  function troisCamps() {
    poserArtefact({
      scoreTimeline: {
        teams: [
          { teamId: 0, rounds: null, total: [{ t: 400, v: 4 }] },
          { teamId: 1, rounds: null, total: [{ t: 500, v: 2 }] },
          { teamId: 2, rounds: null, total: [{ t: 600, v: 6 }] },
        ],
        players: null,
      },
    } as never)
    return afficher({
      scoreboard: feuille([
        { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
        { xuid: 'b', gamertag: 'B', team_side: 't1', is_me: false },
        { xuid: 'c', gamertag: 'C', team_side: 't2', is_me: false },
      ] as Array<Partial<MatchScoreboardRow>>),
    })
  }

  it('statue : écart contre la meilleure équipe adverse, et le DIT dans l’aide du titre', async () => {
    const view = troisCamps()
    expect(await aideDuTitre(view)).toBe(`${t.scoreDeltaInfo} ${t.divergentMultiTeamNote}`)
  })

  it('le dessous prend une encre NEUTRE et la légende nomme « la meilleure équipe adverse »', async () => {
    troisCamps()
    const o = await derniereOption()
    expect(o.series[0].areaStyle.color).toBe('var(--ac-team-ally)')
    expect(o.series[1].areaStyle.color).not.toBe('var(--ac-team-enemy)')
    expect(o.legend.data[1].name).toBe(t.divergentBestOpponent)
  })

  it('l’infobulle nomme « la meilleure équipe adverse », pas un camp arbitraire', async () => {
    troisCamps()
    const o = await derniereOption()
    const html = o.tooltip.formatter([{ axisValue: 90_000 }]) as string
    expect(html).toContain(t.divergentBestOpponent)
  })
})

describe('MatchScoreDeltaChart — langue', () => {
  it('EN : titre, aide et légende passent en anglais', async () => {
    poserArtefact({})
    const view = afficher({ locale: 'en' })
    expect(view.getByText(MATCH_VIEW_TEXT.en.scoreDeltaTitle)).toBeTruthy()
    const o = await derniereOption()
    expect(o.legend.data[0].name).toContain('Eagle')
  })
})
