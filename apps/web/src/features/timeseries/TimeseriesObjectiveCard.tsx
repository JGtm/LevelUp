/**
 * TimeseriesObjectiveCard — carte sobre « Objectifs » du bandeau Timeseries (V72-03).
 *
 * KPI du CUMUL objectif (CTF/Zones/Oddball) du joueur suivi sur le scope courant
 * (data.objective_stats). Choix P4 (documenté au Journal du plan) : KPI agrégé de scope
 * plutôt que des champs par-match sur TimeseriesMatchRow — le tableau par-match de la
 * page est alimenté par un payload distinct (ExplorerMatchRow) et un graphe objectif
 * dédié serait disproportionné pour le périmètre « sobre ». Double porte : capability
 * `objective_stats` (useCapability) + data-driven (KPI > 0 seulement). Tokens sémantiques.
 */
import type { ObjectiveAggregate } from '@/lib/api/types'
import { useCapability } from '@/lib/capabilities/capabilities'
import { formatDurationMMSS } from '@/lib/formatters/duration'
import { formatMessage } from '@/lib/i18n/format'
import { timeseriesManifest, type TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'

interface Props {
  stats?: ObjectiveAggregate | null
  locale: Locale
}

export function TimeseriesObjectiveCard({ stats, locale }: Props) {
  const hasObjectiveStats = useCapability('objective_stats')
  if (!hasObjectiveStats || !stats) return null

  const t = (key: TimeseriesManifestKey) => formatMessage(timeseriesManifest, key, locale)
  const numLoc = locale === 'en' ? 'en-US' : 'fr-FR'
  const num = (v: number) => v.toLocaleString(numLoc)

  const cards: { label: string; value: string }[] = []
  if ((stats.flag_captures ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_captures'), value: num(stats.flag_captures!) })
  if ((stats.flag_returns ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_returns'), value: num(stats.flag_returns!) })
  if ((stats.flag_steals ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_steals'), value: num(stats.flag_steals!) })
  if ((stats.flag_carrier_seconds ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_carrier_time'), value: formatDurationMMSS(stats.flag_carrier_seconds) })
  if ((stats.zone_captures ?? 0) > 0) cards.push({ label: t('timeseries.objectives.zone_captures'), value: num(stats.zone_captures!) })
  if ((stats.zone_secures ?? 0) > 0) cards.push({ label: t('timeseries.objectives.zone_secures'), value: num(stats.zone_secures!) })
  if ((stats.zone_seconds ?? 0) > 0) cards.push({ label: t('timeseries.objectives.zone_time'), value: formatDurationMMSS(stats.zone_seconds) })
  if ((stats.skull_grabs ?? 0) > 0) cards.push({ label: t('timeseries.objectives.skull_grabs'), value: num(stats.skull_grabs!) })
  if ((stats.skull_carrier_seconds ?? 0) > 0) cards.push({ label: t('timeseries.objectives.skull_carrier_time'), value: formatDurationMMSS(stats.skull_carrier_seconds) })

  if (cards.length === 0) return null

  return (
    <section aria-label={t('timeseries.objectives.title')} className="px-6 pb-2">
      <p className="mb-2 text-sm font-semibold text-foreground">{t('timeseries.objectives.title')}</p>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4">
        {cards.map((c) => (
          <div key={c.label} className="rounded-lg border border-border bg-card px-3 py-2">
            <div className="text-xs text-muted-foreground">{c.label}</div>
            <div className="text-lg font-semibold tabular-nums text-foreground">{c.value}</div>
          </div>
        ))}
      </div>
    </section>
  )
}
