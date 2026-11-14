/**
 * MatchAssistChart — le diagramme de flux des assistances, ses deux états vides et ses bots.
 *
 * Ce que ces tests cadenassent : un bloc absent (pas de film porteur de l'assistance) ne
 * rend rien du tout — aucune mention « non mesuré » ; un bloc à liste vide dit « aucune
 * assistance » ; un bot assistant ou finisseur compte comme un joueur ; les deux colonnes sont
 * nommées en mini-phrases (« Assistant : prépare le frag » / « Finisseur : conclut le frag »)
 * avec une flèche de sens ; les phrases d'infobulle sont à la voix active, sujet = l'assistant.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchAssistChart } from './MatchAssistChart'
import { MATCH_VIEW_TEXT } from './i18n'
import type { MatchAssistPair, MatchAssistPairs, MatchScoreboardRow } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `var(${token})`,
  resolveToken: (token: string) => `var(${token})`,
}))

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

const pair = (over: Partial<MatchAssistPair> = {}): MatchAssistPair => ({
  assist_xuid: 'X_A',
  assist_gamertag: 'Alice',
  killer_xuid: 'X_B',
  killer_gamertag: 'Bob',
  assist_count: 2,
  stolen_count: 0,
  ...over,
})

const block = (over: Partial<MatchAssistPairs> = {}): MatchAssistPairs => ({
  pairs: [pair()],
  ...over,
})

const emptyScoreboard: MatchScoreboardRow[] = []

type SankeyOption = {
  series: Array<{
    type: string
    data: Array<{ name: string; value: number; tip: string; label: { position: string; formatter: string } }>
    links: Array<{ source: string; target: string; value: number; tip: string }>
  }>
}

async function sankey(): Promise<SankeyOption['series'][number]> {
  const text = (await screen.findByTestId('echarts-stub')).textContent ?? ''
  return (JSON.parse(text) as SankeyOption).series[0]
}

describe('MatchAssistChart — états vides', () => {
  it('ne rend RIEN quand le bloc est absent (aucune ligne de film)', () => {
    const { container } = render(
      <MatchAssistChart block={undefined} scoreboard={emptyScoreboard} meXUID={null} locale="fr" t={MATCH_VIEW_TEXT.fr} />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('dit « aucune assistance » quand le bloc est là et que la liste est vide, sans mention « non mesuré »', () => {
    const { container } = render(
      <MatchAssistChart
        block={block({ pairs: [] })}
        scoreboard={emptyScoreboard}
        meXUID={null}
        locale="fr"
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.assistNoData)).toBeTruthy()
    expect(container.textContent).not.toMatch(/mesur/i)
    // Aucune colonne à nommer quand il n'y a rien à relier.
    expect(screen.queryByTestId('assist-flow-headers')).toBeNull()
  })

  it('traite `pairs: null` (tableau nullable du contrat) comme une liste vide', () => {
    render(
      <MatchAssistChart
        block={{ pairs: null } as unknown as MatchAssistPairs}
        scoreboard={emptyScoreboard}
        meXUID={null}
        locale="fr"
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.assistNoData)).toBeTruthy()
  })
})

describe('MatchAssistChart — rendu', () => {
  // ChartCard charge ECharts en différé (React.lazy) : le stub n'apparaît qu'après
  // résolution — d'où le `findByTestId` plutôt qu'un `getByTestId` synchrone.
  it('rend le titre, les deux en-têtes de colonne avec leur flèche, et un nœud par joueur et par côté', async () => {
    render(
      <MatchAssistChart
        block={block({
          pairs: [pair({ assist_count: 3 }), pair({ assist_xuid: 'X_C', assist_gamertag: 'Carol', assist_count: 1 })],
        })}
        scoreboard={emptyScoreboard}
        meXUID={null}
        locale="fr"
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.assistTitle)).toBeTruthy()
    const headers = screen.getByTestId('assist-flow-headers')
    expect(headers.textContent).toContain('Assistant : prépare le frag')
    expect(headers.textContent).toContain('Finisseur : conclut le frag')
    expect(headers.textContent).toContain('→')
    const s = await sankey()
    expect(s.type).toBe('sankey')
    // Alice et Carol à gauche, Bob à droite : trois nœuds, deux rubans.
    expect(s.data.map((n) => n.name)).toEqual(['assistant:X_A', 'assistant:X_C', 'finisher:X_B'])
    expect(s.links.map((l) => l.value)).toEqual([3, 1])
  })

  it('étiquette de nœud : nom et total ; côté gauche pour les assistants, côté droit pour les finisseurs', async () => {
    render(
      <MatchAssistChart block={block({ pairs: [pair({ assist_count: 3 })] })} scoreboard={emptyScoreboard} meXUID={null} locale="fr" t={MATCH_VIEW_TEXT.fr} />,
    )
    const [left, right] = (await sankey()).data
    expect(left.label).toEqual({ position: 'left', formatter: 'Alice · 3' })
    expect(right.label).toEqual({ position: 'right', formatter: 'Bob · 3' })
  })

  it('infobulle d’un ruban : voix active, sujet = l’assistant ; « dont N… » seulement s’il y en a ; part moyenne mesurée', async () => {
    render(
      <MatchAssistChart
        block={block({ pairs: [pair({ assist_count: 14, stolen_count: 3, avg_assist_pct: 45 })] })}
        scoreboard={emptyScoreboard}
        meXUID={null}
        locale="fr"
        t={MATCH_VIEW_TEXT.fr}
      />,
    )
    const [link] = (await sankey()).links
    expect(link.tip).toBe(
      'Alice a assisté Bob sur 14 frags<br/>dont 3 où l’assistant a fait plus de dégâts que le finisseur<br/>part moyenne 45 %',
    )
    expect(link.tip).not.toMatch(/été assisté/)
  })

  it('infobulle d’un ruban sans élimination volée ni part moyenne : la seule phrase', async () => {
    render(
      <MatchAssistChart block={block({ pairs: [pair({ assist_count: 1 })] })} scoreboard={emptyScoreboard} meXUID={null} locale="fr" t={MATCH_VIEW_TEXT.fr} />,
    )
    expect((await sankey()).links[0].tip).toBe('Alice a assisté Bob sur 1 frag')
  })

  it('infobulle d’un nœud : assistances données (gauche) / frags conclus sur une assistance (droite)', async () => {
    render(
      <MatchAssistChart block={block({ pairs: [pair({ assist_count: 9 })] })} scoreboard={emptyScoreboard} meXUID={null} locale="fr" t={MATCH_VIEW_TEXT.fr} />,
    )
    const [left, right] = (await sankey()).data
    expect(left.tip).toBe('Alice : 9 assistances données')
    expect(right.tip).toBe('Bob : 9 frags conclus sur une assistance')
  })

  it('EN : « assister » / « finisher », mêmes phrases à la voix active', async () => {
    render(
      <MatchAssistChart
        block={block({ pairs: [pair({ assist_count: 2, stolen_count: 1 })] })}
        scoreboard={emptyScoreboard}
        meXUID={null}
        locale="en"
        t={MATCH_VIEW_TEXT.en}
      />,
    )
    expect(screen.getByTestId('assist-flow-headers').textContent).toContain('Assister: sets up the kill')
    expect(screen.getByTestId('assist-flow-headers').textContent).toContain('Finisher: completes the kill')
    const s = await sankey()
    expect(s.links[0].tip).toBe('Alice assisted Bob on 2 kills<br/>including 1 where the assister dealt more damage than the finisher')
    expect(s.data[0].tip).toBe('Alice: 2 assists given')
    expect(s.data[1].tip).toBe('Bob: 2 kills finished on an assist')
  })
})
