/**
 * MatchTugOfWarChart — match_view.10.
 *
 * Deux panneaux liés par l'axe temps :
 *   1. Barres divergentes par bin (mon équipe en haut, adverses en bas).
 *   2. Kill feed chronologique (acteur, victime, instant) — scrollable.
 *
 * Les bins viennent de `combat_tab.tug_of_war` ; le kill feed dérive de
 * `combat_tab.highlight_events` (kill uniquement) + lookup gamertag via
 * `team_tab.scoreboard` (allié vs adversaire selon `team_side`).
 */
import { Card, CardContent } from '@/components/ui/card'
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import type {
  MatchHighlightEvent,
  MatchScoreboardRow,
  MatchTugOfWarBin,
} from '@/lib/api/types'
import {
  TUG_OF_WAR_LABELS,
  formatBinSeconds,
  tugOfWarStackedSeries,
} from './_chartSeries'

interface Props {
  bins: MatchTugOfWarBin[]
  events: MatchHighlightEvent[]
  scoreboard: MatchScoreboardRow[]
  /** XUID du joueur principal pour distinguer les feeds allié vs adverse. */
  meXUID: string | null
}

interface KillFeedEntry {
  timeMS: number
  actorXUID: string
  actorLabel: string
  isAlly: boolean
  isMe: boolean
}

function buildKillFeed(
  events: MatchHighlightEvent[],
  scoreboard: MatchScoreboardRow[],
  meXUID: string | null,
): KillFeedEntry[] {
  const xuidToRow = new Map<string, MatchScoreboardRow>()
  for (const r of scoreboard) xuidToRow.set(r.xuid, r)

  const meRow = meXUID ? xuidToRow.get(meXUID) : undefined
  const allyTeamSide = meRow?.team_side ?? null

  const out: KillFeedEntry[] = []
  for (const e of events) {
    if ((e.event_type ?? '').toLowerCase() !== 'kill') continue
    if (e.event_time_ms == null || !e.actor_xuid) continue
    const row = xuidToRow.get(e.actor_xuid)
    const isAlly = allyTeamSide != null && row?.team_side === allyTeamSide
    out.push({
      timeMS: e.event_time_ms,
      actorXUID: e.actor_xuid,
      actorLabel: row?.gamertag ?? `XUID(${e.actor_xuid.slice(-4)})`,
      isAlly,
      isMe: e.actor_xuid === meXUID,
    })
  }
  out.sort((a, b) => a.timeMS - b.timeMS)
  return out
}

export function MatchTugOfWarChart({ bins, events, scoreboard, meXUID }: Props) {
  if (bins.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          Pas de données de dominance disponibles pour ce match.
        </CardContent>
      </Card>
    )
  }

  const series = tugOfWarStackedSeries(bins)
  const feed = buildKillFeed(events, scoreboard, meXUID)

  return (
    <Card>
      <CardContent className="py-4 space-y-4">
        <BarStackedChart
          title="Tug-of-war — dominance par tranche"
          height={260}
          series={series}
          componentColors={{
            [TUG_OF_WAR_LABELS.team]: 'outcome-win',
            [TUG_OF_WAR_LABELS.enemy]: 'outcome-loss',
          }}
          componentOrder={[TUG_OF_WAR_LABELS.team, TUG_OF_WAR_LABELS.enemy]}
        />
        <KillFeed entries={feed} />
      </CardContent>
    </Card>
  )
}

function KillFeed({ entries }: { entries: KillFeedEntry[] }) {
  if (entries.length === 0) {
    return (
      <p className="text-xs italic text-muted-foreground">
        Aucun event de kill horodaté disponible.
      </p>
    )
  }
  return (
    <div className="border-t border-border pt-3">
      <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground">
        Kill feed ({entries.length})
      </p>
      <div className="max-h-48 overflow-y-auto pr-1">
        <ul className="space-y-1 text-xs">
          {entries.map((e, idx) => (
            <li
              key={`${e.timeMS}:${e.actorXUID}:${idx}`}
              className={`flex items-center gap-2 ${e.isMe ? 'font-semibold' : ''}`}
            >
              <span className="w-12 tabular-nums text-muted-foreground">
                {formatBinSeconds(Math.floor(e.timeMS / 1000))}
              </span>
              <span
                className={`inline-block w-2 h-2 rounded-full ${
                  e.isAlly ? 'bg-success' : 'bg-destructive'
                }`}
                aria-hidden="true"
              />
              <span className={e.isAlly ? 'text-success' : 'text-destructive'}>
                {e.actorLabel}
              </span>
              <span className="text-muted-foreground">a fragué</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}
