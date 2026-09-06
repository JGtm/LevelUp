/**
 * Tests — MatchScoreEventsChart (les points marqués dans le temps).
 *
 * CE QU'ILS PROTÈGENT :
 *   1. LES PORTES. Sans artefact, sans calque, sur un mode sans compteur, ou quand personne
 *      n'a marqué : RIEN. Un graphe de barres sans barre est un cadre vide, et un cadre vide
 *      est une promesse non tenue — répétée sur chaque page de match.
 *   2. LA GARDE D'HORLOGE. Une origine ni résolue ni publiée décale chaque marque d'un écart
 *      INCONNU (3,6 à 50,8 s selon le match) : mieux vaut rien qu'un instant faux.
 *   3. UNE BARRE PAR MARQUE, à l'instant du point, dans la couleur de son camp — les tokens
 *      `team-ally`/`team-enemy` de la palette d'accessibilité, jamais une teinte en dur.
 *   4. LA RÉSERVE DE PIED, identique à celle de la courbe qu'il remplace.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'

import { MATCH_VIEW_TEXT } from './i18n'
import { MatchScoreEventsChart } from './MatchScoreEventsChart'

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

/**
 * Le témoin CTF `530820e5` réduit : 3-0 sur 3 001 images à 100 ms. Le camp qui n'a jamais
 * marqué n'émet AUCUNE série — c'est le film qui dit son zéro en se taisant.
 */
const CTF_3_0 = {
  teams: [
    {
      teamId: 0,
      rounds: null,
      total: [
        { t: 400, v: 1 },
        { t: 1200, v: 2 },
        { t: 2600, v: 3 },
      ],
    },
  ],
  players: null,
}

/**
 * L'HORLOGE DU TÉMOIN : l'image zéro du film tombe 12 s après le début du match, le coup
 * d'envoi 30 s après. Le film commence donc 18 s AVANT le coup d'envoi, et toute abscisse
 * attendue plus bas vaut `image × 100 − 18 000`, calculé à la main.
 */
const ORIGIN_MS = 12_000
const T0_MS = 30_000

function poserArtefact(over: Partial<ReplayDocument> | null) {
  artefact.current = over
    ? normalizeReplayDocument({
        frameCount: 3001,
        frameIntervalMs: 100,
        originMs: ORIGIN_MS,
        scoreTimeline: CTF_3_0,
        ...over,
      } as unknown as ReplayDocument)
    : undefined
}

function afficher(locale: 'fr' | 'en' = 'fr') {
  return render(
    <MatchScoreEventsChart
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

describe('MatchScoreEventsChart — quand la carte apparaît', () => {
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

  it('ne rend rien quand PERSONNE n’a marqué — sans barre, il n’y a rien à montrer', () => {
    poserArtefact({
      scoreTimeline: { teams: [{ teamId: 0, rounds: null, total: [] }], players: null },
    } as never)
    expect(afficher().container.firstChild).toBeNull()
  })

  it('affiche la carte, son titre et sa note de source quand l’artefact porte des marques', async () => {
    poserArtefact({})
    const view = afficher()
    expect(view.getByText(t.scoreEventsTitle)).toBeTruthy()
    expect(view.getByText(new RegExp(t.scoreCurveSource.slice(0, 30)))).toBeTruthy()
    await waitFor(() => expect(view.getByTestId('echarts-stub')).toBeTruthy())
  })
})

describe('MatchScoreEventsChart — ce que l’option ECharts contient', () => {
  async function option(locale: 'fr' | 'en' = 'fr') {
    poserArtefact({})
    const view = afficher(locale)
    return waitFor(() => JSON.parse(view.getByTestId('echarts-stub').textContent ?? '{}'))
  }

  it('pose UNE BARRE par marque, à l’instant du point, sur un axe des temps en valeurs', async () => {
    const opt = await option()
    expect(opt.series).toHaveLength(2)
    expect(opt.series.every((s: { type: string }) => s.type === 'bar')).toBe(true)
    expect(opt.xAxis.type).toBe('value')
    expect(opt.xAxis.min).toBe(0)
    // L'axe court du COUP D'ENVOI à la fin du film, pas jusqu'à la dernière marque :
    // 3 000 images = 300 s de film, dont 18 s avant le coup d'envoi.
    expect(opt.xAxis.max).toBe(282_000)
    // Paliers 400 / 1 200 / 2 600 -> 40 / 120 / 260 s de film, moins 18 s.
    expect(opt.series[0].data).toEqual([
      [22_000, 1, 1],
      [102_000, 1, 2],
      [242_000, 1, 3],
    ])
  })

  it('garde la voie du camp MUET : le film dit son zéro en se taisant (témoin CTF 3-0)', async () => {
    const opt = await option()
    expect(opt.series[1].data).toEqual([])
  })

  it('fixe une largeur de barre en PIXELS — sur un axe de valeurs, l’auto donne des pavés', async () => {
    const opt = await option()
    expect(opt.series.every((s: { barWidth: number }) => s.barWidth > 0)).toBe(true)
  })

  it('prend les tokens allié / adverse, jamais une couleur en dur', async () => {
    const opt = await option()
    expect(opt.series[0].itemStyle.color).toBe('var(--ac-team-ally)')
    expect(opt.series[1].itemStyle.color).toBe('var(--ac-team-enemy)')
  })

  it('pose la légende EN BAS et CENTRÉE, et y nomme les deux camps', async () => {
    const opt = await option()
    expect(opt.legend.bottom).toBe(0)
    expect(opt.legend.left).toBe('center')
    expect(opt.legend.data).toEqual(['Équipe Eagle', 'Équipe Cobra'])
  })
})

describe('MatchScoreEventsChart — ce que la carte DIT de sa mesure', () => {
  it('signale une lecture TRONQUÉE, comme la courbe qu’elle remplace', () => {
    poserArtefact({ coverage: { score: { truncated: true, modeSupported: true } } } as never)
    expect(afficher().getByText(new RegExp(t.scoreCurveTruncated.slice(0, 30)))).toBeTruthy()
  })

  it('ne dit rien de tel quand la lecture est complète', () => {
    poserArtefact({})
    expect(afficher().queryByText(new RegExp(t.scoreCurveTruncated.slice(0, 30)))).toBeNull()
  })

  it('EN : le titre passe en anglais', () => {
    poserArtefact({})
    expect(afficher('en').getByText(MATCH_VIEW_TEXT.en.scoreEventsTitle)).toBeTruthy()
  })
})
