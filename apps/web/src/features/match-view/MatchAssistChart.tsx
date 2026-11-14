/**
 * MatchAssistChart — « qui assiste qui », en diagramme de flux.
 *
 * MÊME FORME QUE « APPUI » (Escouade › Synergies) : à question identique, forme identique
 * (`features/_shared/assists/AssistFlowChart`). À gauche les ASSISTANTS (« Assistant : prépare le
 * frag »), à droite les FINISSEURS (« Finisseur : conclut le frag »), un ruban par couple dont la
 * largeur est le nombre d'assistances ; chaque joueur apparaît des deux côtés, à sa couleur, et le
 * ruban prend celle de l'assistant. Infobulle d'un ruban, à la voix active : « X a assisté Y sur N
 * frags », « dont N où l'assistant a fait plus de dégâts que le finisseur » quand il y en a, et la
 * part moyenne de participation quand elle est mesurée (même vocabulaire que le kill feed du
 * rejeu — « part », jamais « dégâts »). Les assistants ennemis sont en tête, comme sur le graphe
 * des antagonistes.
 *
 * Source : `combat_tab.assist_pairs` (paires agrégées et comptées par le backend Go
 * depuis `match_kill_events_latest`).
 *
 * DEUX ÉTATS :
 *
 *   bloc ABSENT   le match n'a pas de film, ou son film ne porte pas l'assistance : on ne
 *                 rend RIEN (aucune mention « non mesuré », règle des bases).
 *   pairs vide    le film porte l'assistance : personne n'a assisté personne.
 *                 « Aucune assistance sur ce match ».
 *
 * Les bots y sont des acteurs comme les autres (nommés par le film, `matchAssistFlow.logic`).
 */
import { useMemo } from 'react'

import { AssistFlowChart } from '@/features/_shared/assists/AssistFlowChart'
import type { AssistFlowChartOpts } from '@/features/_shared/assists/assistFlowOption'
import { ASSIST_FLOW_TEXT } from '@/features/_shared/assists/assistFlowStrings'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { MatchAssistPairs, MatchScoreboardRow } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import type { MatchViewText } from './i18n'
import { matchAssistFlow } from './matchAssistFlow.logic'

// Mêmes 11 tokens que le graphe des antagonistes, et pour la même raison : distance
// perceptuelle maximale sur la roue des teintes. La palette est PARTAGÉE volontairement —
// les deux graphes se lisent l'un à côté de l'autre et un joueur y garde une teinte cohérente.
const ASSIST_TOKENS: SemanticToken[] = [
  'outcome-loss',
  'chart-series-7',
  'chart-series-6',
  'chart-series-8',
  'perf-tier-2',
  'compare-a',
  'narrative-humiliation',
  'narrative-debacle',
  'narrative-dominant',
  'narrative-contre-remontada',
  'perf-tier-4',
]

interface Props {
  block: MatchAssistPairs | undefined
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
  locale: Locale
  t: MatchViewText
}

export function MatchAssistChart({ block, scoreboard, meXUID, locale, t }: Props) {
  // `pairs` est un tableau NULLABLE au contrat (huma sérialise ainsi toute tranche Go) :
  // on le comble ici, à la frontière, une seule fois.
  const pairs = block?.pairs
  const flow = useMemo(() => matchAssistFlow(pairs ?? [], scoreboard, meXUID), [pairs, scoreboard, meXUID])

  // Une teinte par joueur (par nom affiché), la même des deux côtés : cycle ASSIST_TOKENS, dans
  // l'ordre des nœuds (assistants puis finisseurs).
  const chart = useMemo<AssistFlowChartOpts>(() => {
    const colors = new Map<string, string>()
    for (const n of flow.nodes) {
      if (!colors.has(n.name)) colors.set(n.name, resolveToken(ASSIST_TOKENS[colors.size % ASSIST_TOKENS.length]))
    }
    return { colorOf: (name) => colors.get(name) ?? '', fmtShare: String, texts: ASSIST_FLOW_TEXT[locale] }
  }, [flow.nodes, locale])

  // Le bloc est absent : pas de film porteur de l'assistance pour ce match (ou titre sans
  // décodeur). Rien à dire, donc rien de rendu : pas de cadre vide, pas de message.
  if (!block) return null

  const flowText = ASSIST_FLOW_TEXT[locale]
  return (
    <AssistFlowChart
      flow={flow}
      chart={chart}
      headers={{ assistant: flowText.assistantHeader, finisher: flowText.finisherHeader }}
      emptyMessage={t.assistNoData}
      frame={{ title: t.assistTitle }}
      testId="match-assist-flow"
    />
  )
}
