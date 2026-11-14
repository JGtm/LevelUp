/**
 * assistFlowOption — l'option ECharts du diagramme de flux « assistant -> finisseur » (série
 * `sankey`) : les ASSISTANTS à gauche, les FINISSEURS à droite, un ruban par couple dont la
 * largeur est le nombre d'assistances. Chaque joueur a un nœud de chaque côté ; le nœud porte la
 * couleur du joueur, le ruban celle de l'assistant (la teinte de celui qui prépare le frag), et
 * le total du nœud est dans son étiquette.
 *
 * Phrases à la VOIX ACTIVE, sujet = l'assistant (« Madina a assisté JGtm sur 14 frags ») : les
 * textes viennent de la page (FR et EN), ce module ne connaît aucun libellé.
 *
 * Aucune couleur ici : la page fournit la couleur de chaque joueur (palette de la page, ou rampe
 * de jetons sur la page Match).
 */
import type { ChartOption } from '@/components/charts/chartOption'
import { CHART_BG, escapeHtml, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'

import type { AssistFlow, FlowLink, FlowNode } from './assistFlow.logic'

export interface AssistFlowTexts {
  /** « Madina a assisté JGtm sur 14 frags ». */
  link: (assistant: string, finisher: string, count: number) => string
  /** La part du ruban, déjà formatée, dite entre parenthèses après la phrase : « 25 % des assistances de l'escouade ». Absente : aucune part à dire. */
  share?: (pct: string) => string
  /** « dont 2 où l'assistant a fait plus de dégâts que le finisseur », quand il y en a. */
  outdamaged: (n: number) => string
  /** « part moyenne 45 % », quand elle est servie. */
  avgShare?: (pct: number) => string
  /** Infobulle d'un nœud de gauche : « JGtm : 9 assistances données ». */
  given: (name: string, n: number) => string
  /** Infobulle d'un nœud de droite : « JGtm : 20 frags conclus sur une assistance ». */
  concluded: (name: string, n: number) => string
}

export interface AssistFlowChartOpts {
  /** Couleur d'un joueur par son nom affiché (le même des deux côtés). */
  colorOf: (name: string) => string
  texts: AssistFlowTexts
  /** Part 0..1 -> texte (« 25 % »), séparateurs de la langue. */
  fmtShare: (v: number) => string
}

/** Marge réservée de chaque côté aux étiquettes des nœuds (px). */
export const LABEL_MARGIN = 150
const NODE_WIDTH = 14
const NODE_GAP = 12
/** Hauteur par nœud de la colonne la plus chargée, plus les marges. */
const ROW_HEIGHT = 34
const PADDING = 24
const MIN_HEIGHT = 220
const LINK_OPACITY = 0.45

/** Hauteur du graphe selon la colonne la plus chargée. */
export function assistFlowHeight(flow: AssistFlow): number {
  const left = flow.nodes.filter((n) => n.side === 'assistant').length
  const right = flow.nodes.length - left
  return Math.max(MIN_HEIGHT, PADDING + Math.max(left, right) * ROW_HEIGHT)
}

function nodeDatum(n: FlowNode, o: AssistFlowChartOpts) {
  const left = n.side === 'assistant'
  const text = left ? o.texts.given(n.name, n.total) : o.texts.concluded(n.name, n.total)
  return {
    name: n.id,
    value: n.total,
    itemStyle: { color: o.colorOf(n.name), borderColor: o.colorOf(n.name) },
    label: { position: left ? 'left' : 'right', formatter: `${n.name} · ${n.total}` },
    tip: escapeHtml(text),
  }
}

function linkTip(l: FlowLink, o: AssistFlowChartOpts): string {
  const sentence = o.texts.link(l.assistantName, l.finisherName, l.count)
  const shareClause = l.share != null && o.texts.share ? ` (${o.texts.share(o.fmtShare(l.share))})` : ''
  const lines = [escapeHtml(sentence + shareClause)]
  if (l.stolen > 0) lines.push(escapeHtml(o.texts.outdamaged(l.stolen)))
  if (l.avgPct != null && o.texts.avgShare) lines.push(escapeHtml(o.texts.avgShare(l.avgPct)))
  return lines.join('<br/>')
}

function linkDatum(l: FlowLink, o: AssistFlowChartOpts) {
  return {
    source: l.source,
    target: l.target,
    value: l.count,
    lineStyle: { color: o.colorOf(l.assistantName) },
    tip: linkTip(l, o),
  }
}

const tipOf = (p: { data?: { tip?: string } }) => p.data?.tip ?? ''

export function buildAssistFlowOption(flow: AssistFlow, opts: AssistFlowChartOpts): ChartOption {
  if (flow.links.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  return {
    backgroundColor: CHART_BG,
    animation: false,
    tooltip: { ...getTooltipBase(tc), trigger: 'item', formatter: tipOf },
    series: [
      {
        type: 'sankey',
        left: LABEL_MARGIN,
        right: LABEL_MARGIN,
        top: PADDING / 2,
        bottom: PADDING / 2,
        nodeWidth: NODE_WIDTH,
        nodeGap: NODE_GAP,
        // Ordre des nœuds = ordre des données : les colonnes suivent l'ordre de la page.
        layoutIterations: 0,
        draggable: false,
        emphasis: { focus: 'adjacency' },
        lineStyle: { opacity: LINK_OPACITY, curveness: 0.5 },
        label: { color: tc.text, fontSize: 12 },
        data: flow.nodes.map((n) => nodeDatum(n, opts)),
        links: flow.links.map((l) => linkDatum(l, opts)),
      },
    ],
  }
}
