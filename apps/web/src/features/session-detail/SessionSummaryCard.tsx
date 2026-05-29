/**
 * SessionSummaryCard — bande KPI plate des performances d'une session.
 *
 * Look aligné sur la KPI barre de la page Stats Solo (`_shared/SessionBriefing/KpiGrid`)
 * : grille de petites cards plates (`rounded border bg-card`), pas de Card englobante,
 * pas de titre. L'identité de la session (label, catégorie, nb de matchs, durée) vit
 * désormais dans le header L3 / drawer via `SessionParamPills` — ici on ne garde QUE
 * les indicateurs de performance, pour éviter la redondance.
 *
 * Valeurs colorées par token sémantique (KDA/KDR via kdScale, Perf via perf-tier).
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { kdScale } from '@/lib/accessibility/scales'
import type { SessionCompareEntry } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { formatNumber, useSessionT } from './_shared'

interface Props {
  entry: SessionCompareEntry | null
}

// Seuils perf-tier (alignés sur analysis.PerfTier côté Go).
function perfTierToken(score: number | null): SemanticToken | undefined {
  if (score == null) return undefined
  const tier = score >= 80 ? 1 : score >= 65 ? 2 : score >= 50 ? 3 : score >= 35 ? 4 : 5
  return `perf-tier-${tier}` as SemanticToken
}

export function SessionSummaryCard({ entry }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string => fieldMappings?.fields[key]?.label ?? key
  const t = useSessionT()

  if (!entry) {
    return (
      <EmptyStateNotice
        title={t('session.detail.summary_unavailable_title')}
        description={t('session.detail.summary_unavailable_description')}
      />
    )
  }

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
      <KpiStat label={t('session.detail.stat_wins_losses')} value={`${entry.wins} / ${entry.losses}`} />
      <KpiStat label={t('session.detail.stat_win_rate')} value={`${formatNumber(entry.win_rate, 0)} %`} />
      <KpiStat
        label={labelOf('kda')}
        value={formatNumber(entry.kda, 2)}
        token={entry.kda != null ? kdScale(entry.kda) : undefined}
      />
      <KpiStat label={t('session.detail.stat_kdr')} value={formatNumber(entry.kdr, 2)} token={kdScale(entry.kdr)} />
      <KpiStat label={t('session.detail.stat_kills_per_match')} value={formatNumber(entry.kills_per_match, 1)} />
      <KpiStat
        label={t('session.detail.stat_perf_score')}
        value={formatNumber(entry.performance_score, 1)}
        token={perfTierToken(entry.performance_score)}
      />
    </div>
  )
}

function KpiStat({ label, value, token }: { label: string; value: string; token?: SemanticToken }) {
  return (
    <div className="rounded border border-border bg-card px-3 py-2">
      <p className="text-3xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
      <p className="mt-0.5 text-lg font-bold text-foreground" style={token ? { color: tokenCssVar(token) } : undefined}>
        {value}
      </p>
    </div>
  )
}
