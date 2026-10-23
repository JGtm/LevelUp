/**
 * TimeseriesObjectiveCard — les tuiles du CUMUL objectif (CTF/Zones/Oddball) du joueur suivi sur le
 * périmètre (data.objective_stats), en tête du bloc « Objectif » de l'onglet Usages.
 *
 * Le bloc qui la monte n'existe que si le périmètre a au moins un match à objectif ; ses états vides
 * (aucun match à objectif, lecture en échec) sont les siens. Seules les actions qui valent plus de
 * zéro ont une tuile ; sans aucune, la rangée se retire : la fiche « Part du joueur à l'objectif »,
 * dessous, garde ses lignes à zéro et dit déjà le constat. Tokens sémantiques.
 */
import type { ObjectiveAggregate } from '@/lib/api/types'
import { formatDurationMMSS } from '@/lib/formatters/duration'
import { formatMessage } from '@/lib/i18n/format'
import { timeseriesManifest, type TimeseriesManifestKey } from '@/lib/i18n/generated/timeseries'
import type { Locale } from '@/lib/i18n/locale'

interface Props {
  stats?: ObjectiveAggregate | null
  locale: Locale
}

export function TimeseriesObjectiveCard({ stats, locale }: Props) {
  const t = (key: TimeseriesManifestKey) => formatMessage(timeseriesManifest, key, locale)
  const numLoc = locale === 'en' ? 'en-US' : 'fr-FR'
  const num = (v: number) => v.toLocaleString(numLoc)

  const cards: { label: string; value: string }[] = []
  if (stats) pushObjectiveCards(cards, stats, t, num)
  if (cards.length === 0) return null

  return (
    <div
      aria-label={t('timeseries.objectives.title')}
      className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4"
      data-testid="timeseries-objectives"
      role="group"
    >
      {cards.map((c) => (
        <div key={c.label} className="rounded-lg border border-border bg-card px-3 py-2">
          <div className="text-xs text-muted-foreground">{c.label}</div>
          <div className="text-lg font-semibold tabular-nums text-foreground">{c.value}</div>
        </div>
      ))}
    </div>
  )
}

/** Les KPI du cumul d'objectif (drapeau, zones, crânes) : seuls ceux qui valent plus de zéro. */
function pushObjectiveCards(
  cards: { label: string; value: string }[],
  stats: ObjectiveAggregate,
  t: (key: TimeseriesManifestKey) => string,
  num: (v: number) => string,
): void {
  if ((stats.flag_captures ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_captures'), value: num(stats.flag_captures!) })
  if ((stats.flag_returns ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_returns'), value: num(stats.flag_returns!) })
  if ((stats.flag_steals ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_steals'), value: num(stats.flag_steals!) })
  if ((stats.flag_carrier_seconds ?? 0) > 0) cards.push({ label: t('timeseries.objectives.flag_carrier_time'), value: formatDurationMMSS(stats.flag_carrier_seconds) })
  if ((stats.zone_captures ?? 0) > 0) cards.push({ label: t('timeseries.objectives.zone_captures'), value: num(stats.zone_captures!) })
  if ((stats.zone_secures ?? 0) > 0) cards.push({ label: t('timeseries.objectives.zone_secures'), value: num(stats.zone_secures!) })
  if ((stats.zone_seconds ?? 0) > 0) cards.push({ label: t('timeseries.objectives.zone_time'), value: formatDurationMMSS(stats.zone_seconds) })
  if ((stats.skull_grabs ?? 0) > 0) cards.push({ label: t('timeseries.objectives.skull_grabs'), value: num(stats.skull_grabs!) })
  if ((stats.skull_carrier_seconds ?? 0) > 0) cards.push({ label: t('timeseries.objectives.skull_carrier_time'), value: formatDurationMMSS(stats.skull_carrier_seconds) })
}
