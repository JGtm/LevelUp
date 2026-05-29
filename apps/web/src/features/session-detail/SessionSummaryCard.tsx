/**
 * SessionSummaryCard — bande KPI plate des performances d'une session.
 *
 * Look aligné sur la section "Mes stats sur cette session" des pages Solo/Escouade
 * (`_shared/SessionBriefing/KpiGrid`) : grille de petites cards plates
 * (`rounded border bg-card`), pas de Card englobante, pas de titre. L'identité de la
 * session (label, catégorie, nb de matchs, durée) vit dans le header L3 / drawer via
 * `SessionParamPills`.
 *
 * Les KPIs redondants avec le donut d'issues (Victoires/Défaites, Taux de victoire)
 * et avec le tableau (KDR) ont été retirés. À la place : Rendement/Résistance (OC/DR,
 * même composite que la home) et Durée de vie moyenne (même rendu que "Mes stats").
 */
import { OffDefComposite } from '@/components/ui/off-def-composite'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { kdScale } from '@/lib/accessibility/scales'
import { formatDurationMMSS } from '@/lib/formatters'
import type { SessionCompareEntry } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { formatNumber, formatRankDelta, rankDeltaToken, useSessionT } from './_shared'

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

  const hasOffDef = entry.avg_oc != null || entry.avg_dr != null

  return (
    <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
      <KpiStat
        label={labelOf('kda')}
        value={formatNumber(entry.kda, 2)}
        token={entry.kda != null ? kdScale(entry.kda) : undefined}
      />
      <KpiStat label={t('session.detail.stat_kills_per_match')} value={formatNumber(entry.kills_per_match, 1)} />
      <KpiStat
        label={t('session.detail.stat_avg_life')}
        value={entry.avg_life_seconds != null ? formatDurationMMSS(entry.avg_life_seconds) : '—'}
      />
      <KpiStat
        label={t('session.detail.stat_perf_score')}
        value={formatNumber(entry.performance_score, 1)}
        token={perfTierToken(entry.performance_score)}
      />
      {/* Rendement / Résistance : composite identique à la home (OffDefComposite). */}
      <div className="col-span-2 rounded border border-border bg-card px-3 py-2">
        <p className="text-3xs font-medium uppercase tracking-wide text-muted-foreground">
          {t('session.detail.stat_off_def')}
        </p>
        <div className="mt-1.5">
          {hasOffDef ? (
            <OffDefComposite offensiveConversion={entry.avg_oc} defensiveResistance={entry.avg_dr} align="start" />
          ) : (
            <p className="text-lg font-bold text-muted-foreground">—</p>
          )}
        </div>
      </div>
      {entry.skill_rating_delta != null && entry.skill_rating_type ? (
        <KpiStat
          label={`Δ ${entry.skill_rating_type.toUpperCase()}`}
          value={formatRankDelta(entry.skill_rating_delta, entry.skill_rating_type)}
          token={rankDeltaToken(entry.skill_rating_delta)}
        />
      ) : null}
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
