/**
 * MatchViewTabChronology.t0.test.tsx — LE CÂBLAGE DE `t0Ms`, ET RIEN D'AUTRE.
 *
 * POURQUOI CE FICHIER (2026-09-06, revue R1 du lot v2 D, constat C1). La correction P0-7
 * — « Frags cumulés » et « Score dans le temps » sur le MÊME axe temporel — tient à trois
 * lignes de câblage : `t0Ms={header.t0_ms}` dans `MatchViewPage`, puis `t0Ms={t0Ms}` sur
 * chacune des DEUX lectures du bloc de score. La revue a retiré les deux dernières : la suite
 * complète est restée verte (6 376 tests). Le seul chaînon entre la correction et l'écran
 * n'avait aucun témoin — les tests d'unité de `_scoreCurve` / `_scoreEvents` fabriquent leur
 * horloge, et `MatchViewTabs.test.tsx` remplace les graphes par des stubs.
 *
 * CE QU'IL VÉRIFIE, aux deux niveaux :
 *  1. le CÂBLAGE : chacune des deux lectures reçoit bien `t0Ms` (une prop retirée = rouge) ;
 *  2. l'EFFET : sur le graphe RÉEL, une abscisse est bien recalée de l'écart entre le coup
 *     d'envoi et l'origine du film — sans la prop, elle repasse sur l'axe du film.
 * Le second sans le premier laisserait la page se recâbler mal ; le premier sans le second ne
 * dirait pas ce que la prop change.
 */
import { describe, expect, it, vi } from 'vitest'
import { waitFor } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import { MATCH_VIEW_TEXT } from './i18n'
import { SCORE_TIMELINE_EVENTS } from './scoreTimelineKind'

vi.mock('@/lib/accessibility', async (orig) => {
  const reel = await orig<typeof import('@/lib/accessibility')>()
  return { ...reel, resolveToken: (token: string) => `var(--ac-${token})` }
})

// Un chart ECharts monté en jsdom crashe le canvas — même stub que les autres tests de
// charts de cette feature, mais il ENREGISTRE les options pour qu'on puisse les relire.
const optionsRendues = vi.hoisted(() => [] as unknown[])
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => {
    optionsRendues.push(option)
    return <div data-testid="echarts-stub" />
  },
}))

// L'artefact de rejeu est la seule frontière réseau des deux lectures de score.
const artefact = vi.hoisted(() => ({ current: undefined as unknown }))
vi.mock('@/lib/replay/queries', () => ({
  useMatchReplay: () => ({ data: artefact.current, isLoading: false }),
  useReplayMapBackground: () => ({ data: undefined }),
  useReplayMapCallouts: () => ({ data: undefined }),
  useReplayMapImage: () => ({ data: undefined }),
}))

// Hors périmètre de ce témoin : cette section ouvre sa propre requête.
vi.mock('@/features/engagement/EngagementMatchSection', () => ({
  EngagementMatchSection: () => <div data-testid="engagement-stub" />,
}))

// Les deux lectures du score, remplacées par des ESPIONS : c'est leur PROP qu'on éprouve.
const propsCourbe = vi.hoisted(() => [] as Record<string, unknown>[])
const propsBarres = vi.hoisted(() => [] as Record<string, unknown>[])
vi.mock('./MatchScoreCurveChart', () => ({
  MatchScoreCurveChart: (p: Record<string, unknown>) => {
    propsCourbe.push(p)
    return <div data-testid="courbe-espion" />
  },
}))
vi.mock('./MatchScoreEventsChart', () => ({
  MatchScoreEventsChart: (p: Record<string, unknown>) => {
    propsBarres.push(p)
    return <div data-testid="barres-espion" />
  },
}))

const { MatchViewTabChronology } = await import('./MatchViewTabChronology')
const { MatchScoreCurveChart } = await vi.importActual<
  typeof import('./MatchScoreCurveChart')
>('./MatchScoreCurveChart')
const { normalizeReplayDocument } = await import('@/lib/replay/replayNormalize')

const t = MATCH_VIEW_TEXT.fr

/** L'image zéro du film tombe 12 s après le début du match, le coup d'envoi 30 s après. */
const ORIGIN_MS = 12_000
const T0_MS = 30_000
/** Le film commence donc 18 s AVANT le coup d'envoi : toute abscisse recule d'autant. */
const ECART_MS = T0_MS - ORIGIN_MS

const SCOREBOARD = [
  { xuid: 'me', gamertag: 'Moi', team_side: 't0', is_me: true },
  { xuid: 'ennemi', gamertag: 'Autre', team_side: 't1', is_me: false },
] as unknown as MatchScoreboardRow[]

/** Deux camps, une marque chacun : de quoi lire une abscisse sans ambiguïté. */
const TIMELINE = {
  teams: [
    { teamId: 0, rounds: null, total: [{ t: 400, v: 1 }] },
    { teamId: 1, rounds: null, total: [{ t: 900, v: 1 }] },
  ],
  players: null,
}

function poserArtefact() {
  artefact.current = normalizeReplayDocument({
    frameCount: 4985,
    frameIntervalMs: 100,
    originMs: ORIGIN_MS,
    scoreTimeline: TIMELINE,
  } as unknown as ReplayDocument)
}

function afficherOnglet(scoreTimelineKind?: string) {
  return renderWithProviders(
    <MatchViewTabChronology
      playerSlug="joueur"
      matchId="m1"
      replayAvailable
      impactBadges={[]}
      highlightEvents={[]}
      scoreboard={SCOREBOARD}
      meXUID="me"
      objectiveEvents={undefined}
      matchPositions={undefined}
      tugOfWar={[]}
      cadence={null}
      scoreTimelineKind={scoreTimelineKind}
      t0Ms={T0_MS}
      locale="fr"
      t={t}
    />,
  )
}

describe('onglet Chronologie — le câblage de l’axe commun (P0-7)', () => {
  it('la COURBE reçoit le coup d’envoi de la page', () => {
    poserArtefact()
    propsCourbe.length = 0
    afficherOnglet()
    expect(propsCourbe.length).toBeGreaterThan(0)
    expect(propsCourbe.at(-1)?.t0Ms).toBe(T0_MS)
  })

  it('les BARRES d’instants aussi — la branche sœur du même bloc', () => {
    poserArtefact()
    propsBarres.length = 0
    afficherOnglet(SCORE_TIMELINE_EVENTS)
    expect(propsBarres.length).toBeGreaterThan(0)
    expect(propsBarres.at(-1)?.t0Ms).toBe(T0_MS)
  })
})

describe('onglet Chronologie — ce que la prop change à l’écran', () => {
  /**
   * Rend le graphe RÉEL et rend l'abscisse de la première MARQUE de l'option ECharts.
   * `points[0]` est toujours le coup d'envoi lui-même (`[0, score à l'entame]`,
   * `_scoreCurve.stepPoints`) : c'est le point SUIVANT qui porte l'instant converti.
   */
  async function premiereAbscisse(t0Ms: number | undefined): Promise<number> {
    poserArtefact()
    optionsRendues.length = 0
    renderWithProviders(
      <MatchScoreCurveChart
        playerSlug="joueur"
        matchId="m1"
        replayAvailable
        scoreboard={SCOREBOARD}
        meXUID="me"
        t0Ms={t0Ms}
        t={t}
      />,
    )
    const opt = await waitFor(() => {
      const derniere = optionsRendues.at(-1) as { series?: { data?: number[][] }[] } | undefined
      if (!derniere?.series?.length) throw new Error('option non rendue')
      return derniere
    })
    return opt.series![0].data![1][0]
  }

  it('avec le coup d’envoi, la marque recule du countdown d’avant-match', async () => {
    const avec = await premiereAbscisse(T0_MS)
    const sans = await premiereAbscisse(undefined)
    // Sans la prop, l'horloge retient `t0Ms = 0` : la marque repasse sur l'axe du film,
    // exactement le défaut P0-7. L'écart entre les deux est le countdown lui-même.
    expect(sans - avec).toBe(T0_MS)
    // Et la valeur câblée est celle calculée à la main : image 400 = 40 000 ms de film,
    // moins les 18 000 ms qui séparent l'image zéro du coup d'envoi.
    expect(avec).toBe(400 * 100 - ECART_MS)
  })
})
