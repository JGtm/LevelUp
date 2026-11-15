/**
 * SquadAppuiCard — « Appui » en diagramme de flux : en-têtes de colonnes en mini-phrases avec
 * flèche, nœuds dans l'ordre du roster et à la couleur de la page, infobulle d'un ruban à la voix
 * active avec la part sur le total serveur, état vide nommé, FR et EN.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadAssistPair, SquadAssistPairs } from '@/lib/api/types'

import { SquadAppuiCard } from './SquadAppuiCard'

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => <div data-testid="echarts-stub">{JSON.stringify(option)}</div>,
}))

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

const paire = (assist: string, killer: string, count: number, stolen = 0): SquadAssistPair => ({
  assist_xuid: `x_${assist}`,
  assist_gamertag: assist,
  killer_xuid: `x_${killer}`,
  killer_gamertag: killer,
  assist_count: count,
  stolen_count: stolen,
})

const BLOC: SquadAssistPairs = {
  matches_measured: 8,
  matches_total: 10,
  total_assists: 56,
  pairs: [paire('Madina', 'JGtm', 14, 3), paire('JGtm', 'Madina', 9)],
}

const ROSTER = ['JGtm', 'Madina']
const COULEURS = { JGtm: '#2255cc', Madina: '#aa3322' }

type Series = {
  type: string
  data: { name: string; itemStyle: { color: string }; tip: string }[]
  links: { source: string; target: string; lineStyle: { color: string }; tip: string }[]
}
async function sankey(): Promise<Series> {
  const text = (await screen.findByTestId('echarts-stub')).textContent ?? ''
  return (JSON.parse(text) as { series: Series[] }).series[0]
}

describe('SquadAppuiCard', () => {
  it('titre « Appui », en-têtes « Assistant : prépare le frag » et « Finisseur : conclut le frag », flèche de sens', () => {
    renderWithProviders(<SquadAppuiCard block={BLOC} roster={ROSTER} colorByPlayer={COULEURS} />)
    expect(screen.getByText('Appui')).toBeTruthy()
    const headers = screen.getByTestId('assist-flow-headers')
    expect(headers.textContent).toContain('Assistant : prépare le frag')
    expect(headers.textContent).toContain('Finisseur : conclut le frag')
    expect(headers.textContent).toContain('→')
  })

  it('aucun « larbin » ni « patron » à l’écran', () => {
    const { container } = renderWithProviders(<SquadAppuiCard block={BLOC} roster={ROSTER} colorByPlayer={COULEURS} />)
    expect(container.textContent).not.toMatch(/larbin|patron/i)
  })

  it('nœuds dans l’ordre du roster, à la couleur de la page ; ruban à la couleur de l’assistant', async () => {
    renderWithProviders(<SquadAppuiCard block={BLOC} roster={ROSTER} colorByPlayer={COULEURS} />)
    const s = await sankey()
    expect(s.type).toBe('sankey')
    expect(s.data.map((n) => n.name)).toEqual(['assistant:JGtm', 'assistant:Madina', 'finisher:JGtm', 'finisher:Madina'])
    expect(s.data[0].itemStyle.color).toBe('#2255cc')
    expect(s.data[3].itemStyle.color).toBe('#aa3322')
    const ruban = s.links.find((l) => l.source === 'assistant:Madina')!
    expect(ruban.lineStyle.color).toBe('#aa3322')
  })

  it('infobulle d’un ruban : voix active, part sur le TOTAL SERVEUR, puis « dont N… »', async () => {
    renderWithProviders(<SquadAppuiCard block={BLOC} roster={ROSTER} colorByPlayer={COULEURS} />)
    const ruban = (await sankey()).links.find((l) => l.source === 'assistant:Madina')!
    // 14 sur les 56 assistances de l'escouade : 25 %.
    expect(ruban.tip.replace(/\p{Zs}/gu, ' ')).toBe(
      'Madina a assisté JGtm sur 14 frags (25 % des assistances de l&#39;escouade)<br/>dont 3 où l’assistant a fait plus de dégâts que le finisseur',
    )
  })

  it('infobulle d’un nœud : « JGtm : 9 assistances données » / « JGtm : 14 frags conclus sur une assistance »', async () => {
    renderWithProviders(<SquadAppuiCard block={BLOC} roster={ROSTER} colorByPlayer={COULEURS} />)
    const s = await sankey()
    expect(s.data.find((n) => n.name === 'assistant:JGtm')!.tip).toBe('JGtm : 9 assistances données')
    expect(s.data.find((n) => n.name === 'finisher:JGtm')!.tip).toBe('JGtm : 14 frags conclus sur une assistance')
  })

  it('EN : titre, en-têtes et phrases (« assister », « finisher »)', async () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<SquadAppuiCard block={BLOC} roster={ROSTER} colorByPlayer={COULEURS} />)
    expect(screen.getByText('Support')).toBeTruthy()
    expect(screen.getByTestId('assist-flow-headers').textContent).toContain('Assister: sets up the kill')
    const ruban = (await sankey()).links.find((l) => l.source === 'assistant:Madina')!
    expect(ruban.tip.replace(/\p{Zs}/gu, ' ')).toBe(
      "Madina assisted JGtm on 14 kills (25% of the squad&#39;s assists)<br/>including 3 where the assister dealt more damage than the finisher",
    )
  })

  it('aucune paire : l’état vide nommé, sans en-têtes ni diagramme', () => {
    renderWithProviders(<SquadAppuiCard block={{ ...BLOC, pairs: [] }} roster={ROSTER} colorByPlayer={COULEURS} />)
    expect(screen.getByText("Aucune assistance entre membres de l'escouade sur les matchs de la sélection.")).toBeTruthy()
    expect(screen.queryByTestId('assist-flow-headers')).toBeNull()
    expect(screen.queryByTestId('echarts-stub')).toBeNull()
  })
})
