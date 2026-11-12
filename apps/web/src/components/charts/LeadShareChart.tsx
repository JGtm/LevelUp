/**
 * LeadShareChart — la carte de la MAINMISE, commune à Sessions, Séries temporelles et Escouade.
 *
 * Titre neutre + bouton d'information + graphique + légende en bas, un seul axe (convention des
 * graphiques du projet). L'option vit dans `leadShareOption.ts` (pur, testé) ; ce composant ne
 * fait que poser les points dans un `ChartCard`.
 *
 * Aucun match mesuré : l'état vide en dit la cause (le film ou le score ne permet pas de calculer
 * la Mainmise), jamais une carte de zéros.
 *
 * QUI GARDE LA CARTE. Les pages qui la posent la gardent par la capability `replay` (le titre décode le
 * film) : c'est la seule que le bootstrap sert à la page, la clé fine `film.lead_share` (ce que le
 * serveur sert) n'y arrive pas. Un titre qui a `replay` sans `film.lead_share` ne sert aucune mesure :
 * la carte tombe alors sur l'état vide ci-dessous, jamais sur des zéros (test dédié).
 */
import { useCallback, useMemo, type ReactNode } from 'react'

import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { LeadSharePoint } from '@/lib/charts/leadShare/model'
import { useLeadShareText } from '@/lib/charts/leadShare/text'
import { useAppShellStore } from '@/stores/appShellStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import { buildLeadShareOption, type LeadShareVariant } from './leadShareOption'

export interface LeadShareChartProps {
  /** Titre de la carte ; absent : « Mainmise ». */
  title?: string
  points: LeadSharePoint[]
  variant?: LeadShareVariant
  compact?: boolean
  height?: number
  /** Ligne de résumé sous le titre (moyenne des matchs mesurés...). */
  summary?: ReactNode
  loading?: boolean
}

export function LeadShareChart({
  title,
  points,
  variant = 'bars',
  compact = false,
  height = 260,
  summary,
  loading,
}: LeadShareChartProps) {
  const text = useLeadShareText()
  const locale = useAppShellStore((s) => s.locale)
  const series = useMemo<ChartSeries<LeadSharePoint>[]>(
    () => (points.some((p) => p.share != null) ? [{ key: 'lead-share', datapoints: points }] : []),
    [points],
  )
  const buildOption = useCallback(
    (s: ChartSeries<LeadSharePoint>[]) => buildLeadShareOption(s[0]?.datapoints ?? [], { text, locale, variant, compact }),
    [text, locale, variant, compact],
  )
  return (
    <div className="space-y-2" data-testid="lead-share-chart">
      <ChartCard
        title={
          <span className="inline-flex items-center gap-2">
            {title ?? text.name}
            <InfoTooltip content={text.help} />
          </span>
        }
        series={series}
        buildOption={buildOption}
        height={height}
        loading={loading}
        emptyMessage={text.emptyNoMeasure}
      />
      {summary ? <p className="px-1 text-xs text-muted-foreground tabular-nums">{summary}</p> : null}
    </div>
  )
}
