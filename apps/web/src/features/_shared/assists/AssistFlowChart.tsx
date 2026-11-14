/**
 * AssistFlowChart — le diagramme de flux « assistant -> finisseur », partagé par « Appui »
 * (Escouade › Synergies) et « Assistances » (page Match) : à question identique, forme identique.
 *
 * En-tête : deux mini-phrases, une par colonne (« Assistant : prépare le frag » à gauche,
 * « Finisseur : conclut le frag » à droite), et une flèche qui dit le sens de lecture, de gauche
 * à droite. Dessous : le graphe (`assistFlowOption.ts`). Un cadre optionnel (`frame`) reproduit
 * celui d'un `ChartCard` titré, pour la page Match où le graphe est une carte ; dans une
 * `SectionCard` (Escouade) le graphe est nu.
 */
import { useMemo, useRef } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'

import type { AssistFlow } from './assistFlow.logic'
import {
  assistFlowHeight,
  buildAssistFlowOption,
  DEFAULT_FLOW_WIDTH,
  type AssistFlowChartOpts,
} from './assistFlowOption'
import { useElementWidth } from './useElementWidth'

export interface AssistFlowHeaders {
  assistant: string
  finisher: string
}

interface Props {
  flow: AssistFlow
  /** Les options du graphe, SANS la largeur : le composant la mesure. */
  chart: Omit<AssistFlowChartOpts, 'width'>
  headers: AssistFlowHeaders
  /** Message de l'état vide (aucune assistance). */
  emptyMessage: string
  /** Titre d'une carte autonome (page Match) ; absent : le graphe est nu. */
  frame?: { title: string }
  testId?: string
}

/** Le sens de lecture : de l'assistant vers le finisseur. */
const FLOW_ARROW = '→'

export function AssistFlowChart({ flow, chart, headers, emptyMessage, frame, testId }: Props) {
  const series = useMemo(() => (flow.links.length > 0 ? [{ key: 'assist-flow', datapoints: flow.links }] : []), [flow.links])
  const ref = useRef<HTMLDivElement>(null)
  const width = useElementWidth(ref, DEFAULT_FLOW_WIDTH)
  const buildOption = useMemo(() => () => buildAssistFlowOption(flow, { ...chart, width }), [flow, chart, width])
  const body = (
    <div ref={ref} data-testid={testId}>
      {flow.links.length > 0 && (
        <div
          className="flex items-center justify-between gap-2 px-3 pt-2 text-xs font-medium text-muted-foreground"
          data-testid="assist-flow-headers"
        >
          <span>{headers.assistant}</span>
          <span aria-hidden="true" className="text-base leading-none">
            {FLOW_ARROW}
          </span>
          <span className="text-right">{headers.finisher}</span>
        </div>
      )}
      <ChartCard
        series={series}
        buildOption={buildOption}
        height={assistFlowHeight(flow)}
        emptyMessage={emptyMessage}
        frameless
      />
    </div>
  )
  if (!frame) return body
  return (
    <div className="rounded-lg border border-border bg-card">
      {/* Même bandeau de titre qu'un ChartCard titré : les cartes voisines de la rangée s'alignent. */}
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">{frame.title}</div>
      {body}
    </div>
  )
}
