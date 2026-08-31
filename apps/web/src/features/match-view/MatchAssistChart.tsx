/**
 * MatchAssistChart — « qui est l'assistant de qui ».
 *
 * Barres empilées horizontales, MIROIR du graphe des antagonistes :
 *   - 1 ligne par ASSISTANT (groupées par équipe via assistStackedSeries)
 *   - segments empilés = les TUEURS qu'il a assistés
 *   - infobulle : nombre d'assistances, « dont N volées » quand il y en a, et la part
 *     moyenne de participation quand elle est mesurée (même vocabulaire que le kill feed
 *     du rejeu — « part », jamais « dégâts »)
 *
 * Source : `combat_tab.assist_pairs` (paires agrégées et comptées par le backend Go
 * depuis `match_kill_events_latest`).
 *
 * LES DEUX ÉTATS VIDES NE SE CONFONDENT PAS, et c'est la raison d'être de ce
 * composant plutôt qu'un simple clone :
 *
 *   bloc ABSENT              le match n'a aucune ligne de film — on ne rend RIEN.
 *   measured_deaths === 0    le film est là, mais aucune mort n'est LISIBLE ligne à
 *                            ligne : soit l'assistance n'y est pas mesurée, soit elle
 *                            l'est sans être publiable (cas BTB — nommer deux joueurs
 *                            sur une mort n'y est pas permis). Le contrat ne sépare pas
 *                            les deux causes, le libellé les couvre donc toutes deux.
 *   pairs vide, mesuré > 0   MESURÉ : personne n'a assisté personne.
 *                            « Aucune assistance sur ce match ».
 *
 * Écrire « aucune assistance » sur le deuxième cas fabriquerait un fait jamais
 * observé (doctrine des trois états de `match_kill_events`).
 */
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type { MatchAssistPairs, MatchScoreboardRow } from '@/lib/api/types'
import { assistAvgPctLookup, assistStackedSeries, assistStolenKey, assistStolenLookup } from './_chartSeries'
import type { MatchViewText } from './i18n'

// Mêmes 11 tokens que le graphe des antagonistes, et pour la même raison : distance
// perceptuelle maximale sur la roue des teintes. La palette est PARTAGÉE volontairement —
// les deux graphes se lisent l'un sous l'autre et un joueur y garde une teinte cohérente
// dans les cas où il apparaît des deux côtés.
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
  t: MatchViewText
}

export function MatchAssistChart({ block, scoreboard, meXUID, t }: Props) {
  // Porte 1 — le bloc est absent : aucune ligne de film pour ce match (ou titre sans
  // décodeur). Rien à dire, donc rien de rendu : pas de cadre vide, pas de message.
  if (!block) return null

  // `pairs` est un tableau NULLABLE au contrat (huma sérialise ainsi toute tranche Go) :
  // on le comble ici, à la frontière, une seule fois.
  const pairs = block.pairs ?? []
  const series = assistStackedSeries(pairs, scoreboard, meXUID)
  const stolen = assistStolenLookup(pairs)
  const avgPct = assistAvgPctLookup(pairs)

  // Porte 2 — lisible ou non. Les deux libellés sont distincts et le choix se fait sur le
  // DÉNOMINATEUR, jamais sur la longueur de la liste.
  const emptyMessage = block.measured_deaths === 0 ? t.assistNotUsable : t.assistNoData

  // Couleurs par tueur assisté — cycle ASSIST_TOKENS (11 teintes distinctes).
  const killerSet = new Set<string>()
  for (const s of series) {
    for (const dp of s.datapoints) {
      for (const key of Object.keys(dp.components)) killerSet.add(key)
    }
  }
  const componentHexColors: Record<string, string> = {}
  Array.from(killerSet).forEach((gt, idx) => {
    componentHexColors[gt] = resolveToken(ASSIST_TOKENS[idx % ASSIST_TOKENS.length])
  })

  const assistantCount = series.length > 0 ? series[0].datapoints.length : 0
  const height = Math.max(240, 80 + 24 * assistantCount)
  return (
    <BarStackedChart
      title={t.assistTitle}
      height={height}
      orientation="horizontal"
      series={series}
      emptyMessage={emptyMessage}
      componentHexColors={componentHexColors}
      tooltipHideZero
      tooltipComponentNote={(assistant, killer) => {
        const key = assistStolenKey(assistant, killer)
        const notes: string[] = []
        const n = stolen.get(key)
        if (n) notes.push(t.assistStolenNote(n))
        const pct = avgPct.get(key)
        if (pct != null) notes.push(t.assistAvgShareNote(pct))
        return notes.length > 0 ? notes.join(' · ') : undefined
      }}
    />
  )
}
