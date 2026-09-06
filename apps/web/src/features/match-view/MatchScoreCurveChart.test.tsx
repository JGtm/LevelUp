/**
 * Tests — MatchScoreCurveChart (la courbe de score de la vue match).
 *
 * CE QU'ILS PROTÈGENT AVANT TOUT : la carte ne doit RIEN afficher quand l'artefact de rejeu
 * n'existe pas. C'est la règle explicite du plan (« sinon rien, pas de placeholder ») et
 * c'est aussi la réalité de la production, où presque aucun match n'a de film décodé : un
 * cadre vide sur chaque page de match serait une promesse non tenue répétée à l'infini.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'

import { MATCH_VIEW_TEXT } from './i18n'
import { MatchScoreCurveChart } from './MatchScoreCurveChart'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(--ac-${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

// Un chart ECharts monté en jsdom crashe le canvas — on stubbe le wrapper (cf. les autres
// tests de charts de cette feature).
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

// La lecture de l'artefact est la SEULE frontière réseau de ce composant : on la pilote,
// et tout le reste (garde d'horloge, projection en paliers) reste le vrai code.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('@/lib/replay/queries', () => ({
  useMatchReplay: () => ({ data: artefact.current }),
}))

const { normalizeReplayDocument } = await import('@/lib/replay/replayNormalize')

const t = MATCH_VIEW_TEXT.fr

const SCOREBOARD = [
  { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
  { xuid: 'ennemi', gamertag: 'Autre', team_side: 't1', is_me: false },
] as unknown as MatchScoreboardRow[]

/** Le témoin Slayer réduit : 43/50 sur 4 985 images à 100 ms. */
const SLAYER = {
  teams: [
    { teamId: 0, rounds: null, total: [{ t: 399, v: 1 }, { t: 4886, v: 43 }] },
    { teamId: 1, rounds: null, total: [{ t: 317, v: 2 }, { t: 4908, v: 50 }] },
  ],
  players: null,
}

/**
 * L'HORLOGE DU TÉMOIN : l'image zéro du film tombe 12 s après le début du match, le coup
 * d'envoi 30 s après. Le film commence donc 18 s AVANT le coup d'envoi — soit à l'image 180
 * — et toute abscisse attendue plus bas vaut `image × 100 − 18 000`, calculé à la main.
 */
const ORIGIN_MS = 12_000
const T0_MS = 30_000

function poserArtefact(over: Partial<ReplayDocument> | null) {
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

function afficher(locale: 'fr' | 'en' = 'fr') {
  return render(
    <MatchScoreCurveChart
      playerSlug="joueur"
      matchId="m1"
      replayAvailable
      scoreboard={SCOREBOARD}
      meXUID="me"
      t0Ms={T0_MS}
      t={MATCH_VIEW_TEXT[locale]}
    />,
  )
}

describe('MatchScoreCurveChart — quand la carte apparaît', () => {
  it('ne rend RIEN sans artefact : 404 = pas de film, pas de cadre vide', () => {
    poserArtefact(null)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand l’artefact existe mais ne porte pas le calque de score', () => {
    poserArtefact({ scoreTimeline: undefined })
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand le MODE ne porte pas de compteur (coverage.modeSupported)', () => {
    poserArtefact({ coverage: { score: { modeSupported: false } } } as never)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien quand l’origine du film n’est ni résolue ni publiée (garde d’horloge)', () => {
    poserArtefact({ originMs: undefined, coverage: { originResolved: false } } as never)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('ne rend rien SANS ORIGINE PUBLIÉE : l’axe serait décalé de 3,6 à 50,8 s sans le dire', () => {
    poserArtefact({ originMs: undefined } as never)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('affiche la carte, son titre et sa note de source quand l’artefact porte le calque', async () => {
    poserArtefact({})
    const view = afficher()
    expect(view.getByText(t.scoreCurveTitle)).toBeTruthy()
    expect(view.getByText(new RegExp(t.scoreCurveSource.slice(0, 30)))).toBeTruthy()
    await waitFor(() => expect(view.getByTestId('echarts-stub')).toBeTruthy())
  })
})

describe('MatchScoreCurveChart — ce que l’option ECharts contient', () => {
  it('trace DEUX séries en ESCALIER, une par camp, sur un seul axe de valeurs', async () => {
    poserArtefact({})
    const view = afficher()
    const opt = await waitFor(() => JSON.parse(view.getByTestId('echarts-stub').textContent ?? '{}'))
    expect(opt.series).toHaveLength(2)
    expect(opt.series.every((s: { step: string }) => s.step === 'end')).toBe(true)
    expect(Array.isArray(opt.yAxis)).toBe(false)
    expect(opt.xAxis.type).toBe('value')
    // 4 984 images = 498,4 s de film, dont 18 s AVANT le coup d'envoi.
    expect(opt.xAxis.max).toBe(480_400)
  })

  it('borne chaque série au COUP D’ENVOI et à la fin du match, jamais à l’image zéro du film', async () => {
    poserArtefact({})
    const view = afficher()
    const opt = await waitFor(() => JSON.parse(view.getByTestId('echarts-stub').textContent ?? '{}'))
    // Le point de départ est le coup d'envoi (score nul), et non l'image zéro du film :
    // celle-ci se lirait « 0m00s » 18 s trop tôt, juste sous « Frags cumulés » qui, lui,
    // compte depuis le coup d'envoi (registre 2026-09-05, P0-7).
    expect(opt.series[0].data[0]).toEqual([0, 0])
    // Paliers 399 et 4 886 -> 39,9 s et 488,6 s de film, moins 18 s.
    expect(opt.series[0].data).toEqual([
      [0, 0],
      [21_900, 1],
      [470_600, 43],
      [480_400, 43],
    ])
    // Paliers 317 et 4 908 -> 31,7 s et 490,8 s de film, moins 18 s.
    expect(opt.series[1].data).toEqual([
      [0, 0],
      [13_700, 2],
      [472_800, 50],
      [480_400, 50],
    ])
  })

  it('SANS T0 les deux axes retombent ensemble sur celui du match (countdown inconnu)', async () => {
    poserArtefact({})
    const view = render(
      <MatchScoreCurveChart
        playerSlug="joueur"
        matchId="m1"
        replayAvailable
        scoreboard={SCOREBOARD}
        meXUID="me"
        t={t}
      />,
    )
    const opt = await waitFor(() => JSON.parse(view.getByTestId('echarts-stub').textContent ?? '{}'))
    // Le serveur n'a alors rien retranché aux events non plus : + originMs, rien de plus.
    expect(opt.xAxis.max).toBe(498_400 + ORIGIN_MS)
    expect(opt.series[0].data[1]).toEqual([39_900 + ORIGIN_MS, 1])
  })

  it('prend les tokens allié / adverse, jamais une couleur en dur', async () => {
    poserArtefact({})
    const view = afficher()
    const opt = await waitFor(() => JSON.parse(view.getByTestId('echarts-stub').textContent ?? '{}'))
    expect(opt.series[0].lineStyle.color).toBe('var(--ac-team-ally)')
    expect(opt.series[1].lineStyle.color).toBe('var(--ac-team-enemy)')
  })

  it('nomme les deux séries dans la légende — l’identité n’est jamais portée par la seule couleur', async () => {
    poserArtefact({})
    const view = afficher()
    const opt = await waitFor(() => JSON.parse(view.getByTestId('echarts-stub').textContent ?? '{}'))
    expect(opt.legend.data).toEqual(['Équipe Eagle', 'Équipe Cobra'])
  })

  it('pose les RETOURNEMENTS une seule fois, sur la première série', async () => {
    poserArtefact({
      scoreTimeline: {
        teams: [
          { teamId: 0, rounds: null, total: [{ t: 100, v: 1 }, { t: 300, v: 5 }] },
          { teamId: 1, rounds: null, total: [{ t: 50, v: 1 }, { t: 400, v: 9 }] },
        ],
        players: null,
      },
    } as never)
    const view = afficher()
    const opt = await waitFor(() => JSON.parse(view.getByTestId('echarts-stub').textContent ?? '{}'))
    // t0 passe devant à l'image 300, t1 reprend à 400 : deux retournements, tous sur la
    // première série (répétés sur chaque courbe, ils doubleraient chaque trait). Datés sur
    // l'horloge du gameplay comme le reste : 30 s et 40 s de film, moins 18 s.
    expect(opt.series[0].markLine.data.map((d: { xAxis: number }) => d.xAxis)).toEqual([12_000, 22_000])
    expect(opt.series[1].markLine).toBeUndefined()
  })
})

describe('MatchScoreCurveChart — ce que la carte DIT de sa mesure', () => {
  it('signale une lecture TRONQUÉE : une courbe incomplète qui a l’air complète est pire', () => {
    poserArtefact({ coverage: { score: { truncated: true, modeSupported: true } } } as never)
    expect(afficher().getByText(new RegExp(t.scoreCurveTruncated.slice(0, 30)))).toBeTruthy()
  })

  it('ne dit rien de tel quand la lecture est complète', () => {
    poserArtefact({})
    expect(afficher().queryByText(new RegExp(t.scoreCurveTruncated.slice(0, 30)))).toBeNull()
  })

  it('EN : titre et note passent en anglais', () => {
    poserArtefact({})
    const view = afficher('en')
    expect(view.getByText(MATCH_VIEW_TEXT.en.scoreCurveTitle)).toBeTruthy()
  })
})

describe('MatchScoreCurveChart — la lecture décidée par la DONNÉE du titre', () => {
  it('s’efface en `hidden` : le mode marque au frag, « Frags cumulés » vient de le dire', () => {
    poserArtefact({})
    const vue = render(
      <MatchScoreCurveChart
        playerSlug="joueur"
        matchId="m1"
        replayAvailable
        scoreboard={SCOREBOARD}
        meXUID="me"
        scoreTimelineKind="hidden"
        t={t}
      />,
    )
    expect(vue.container.firstChild).toBeNull()
  })

  it('garde la courbe sur toute autre valeur — l’inconnu est le REPLI SÛR, jamais une absence', () => {
    for (const kind of [undefined, 'curve', 'valeur-inconnue']) {
      poserArtefact({})
      const vue = render(
        <MatchScoreCurveChart
          playerSlug="joueur"
          matchId="m1"
          replayAvailable
          scoreboard={SCOREBOARD}
          meXUID="me"
          scoreTimelineKind={kind}
          t={t}
        />,
      )
      expect(vue.getByText(t.scoreCurveTitle)).toBeTruthy()
      vue.unmount()
    }
  })
})
