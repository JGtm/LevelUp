/**
 * SquadObjectiveStatsPanel — panneau « Objectifs de l'escouade » (V72-03).
 *
 * KPI sobres du CUMUL escouade des stats objectifs (CTF/Zones/Oddball) sur les matchs
 * partagés du scope : somme des agrégats par joueur (header.objective_stats_by_xuid).
 * Double porte : capability `objective_stats` (useCapability) + data-driven (n'affiche
 * que les KPI dont le cumul escouade > 0 ; rien si aucun match à objectif). Tokens
 * sémantiques (aucun hex). NB : distinct de SquadObjectivesPanel (défis/coach).
 */
import type { ObjectiveAggregate } from '@/lib/api/types'
import { useCapability } from '@/lib/capabilities/capabilities'
import { formatDurationMMSS } from '@/lib/formatters/duration'

import type { SquadText } from './i18n'

interface Props {
  statsByXuid?: Record<string, ObjectiveAggregate>
  texts: SquadText
  numLoc: string
}

/** Somme field-by-field des agrégats objectif de tous les membres de l'escouade. */
function squadTotal(statsByXuid: Record<string, ObjectiveAggregate>): ObjectiveAggregate {
  const t: Required<ObjectiveAggregate> = {
    flag_captures: 0,
    flag_returns: 0,
    flag_steals: 0,
    flag_carrier_seconds: 0,
    zone_captures: 0,
    zone_secures: 0,
    zone_seconds: 0,
    skull_grabs: 0,
    skull_carrier_seconds: 0,
  }
  for (const a of Object.values(statsByXuid)) {
    t.flag_captures += a.flag_captures ?? 0
    t.flag_returns += a.flag_returns ?? 0
    t.flag_steals += a.flag_steals ?? 0
    t.flag_carrier_seconds += a.flag_carrier_seconds ?? 0
    t.zone_captures += a.zone_captures ?? 0
    t.zone_secures += a.zone_secures ?? 0
    t.zone_seconds += a.zone_seconds ?? 0
    t.skull_grabs += a.skull_grabs ?? 0
    t.skull_carrier_seconds += a.skull_carrier_seconds ?? 0
  }
  return t
}

export function SquadObjectiveStatsPanel({ statsByXuid, texts, numLoc }: Props) {
  const hasObjectiveStats = useCapability('objective_stats')
  if (!hasObjectiveStats || !statsByXuid || Object.keys(statsByXuid).length === 0) return null

  const total = squadTotal(statsByXuid)
  const o = texts.objectives
  const num = (v: number) => v.toLocaleString(numLoc)

  const cards: { label: string; value: string }[] = []
  if ((total.flag_captures ?? 0) > 0) cards.push({ label: o.flagCaptures, value: num(total.flag_captures!) })
  if ((total.flag_returns ?? 0) > 0) cards.push({ label: o.flagReturns, value: num(total.flag_returns!) })
  if ((total.flag_steals ?? 0) > 0) cards.push({ label: o.flagSteals, value: num(total.flag_steals!) })
  if ((total.flag_carrier_seconds ?? 0) > 0)
    cards.push({ label: o.flagCarrierTime, value: formatDurationMMSS(total.flag_carrier_seconds) })
  if ((total.zone_captures ?? 0) > 0) cards.push({ label: o.zoneCaptures, value: num(total.zone_captures!) })
  if ((total.zone_secures ?? 0) > 0) cards.push({ label: o.zoneSecures, value: num(total.zone_secures!) })
  if ((total.zone_seconds ?? 0) > 0)
    cards.push({ label: o.zoneTime, value: formatDurationMMSS(total.zone_seconds) })
  if ((total.skull_grabs ?? 0) > 0) cards.push({ label: o.skullGrabs, value: num(total.skull_grabs!) })
  if ((total.skull_carrier_seconds ?? 0) > 0)
    cards.push({ label: o.skullCarrierTime, value: formatDurationMMSS(total.skull_carrier_seconds) })

  if (cards.length === 0) return null

  return (
    <section aria-label={o.title}>
      <p className="mb-2 text-sm font-semibold text-foreground">{o.title}</p>
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
