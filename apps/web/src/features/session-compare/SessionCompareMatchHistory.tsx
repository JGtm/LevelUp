/**
 * SessionCompareMatchHistory — tableau des matchs par session avec onglets A/B.
 * Chart 14 du mock session_compare.
 */
import { useState } from 'react'

import type { SessionDetailMatchRow } from '@/lib/api/types'

// Outcome codes : 2=win, 3=loss (Halo standard).
const OUTCOME_WIN = 2
const OUTCOME_LOSS = 3

export interface SessionCompareMatchHistoryProps {
  matchesA: SessionDetailMatchRow[]
  matchesB: SessionDetailMatchRow[]
  labels: {
    title: string
    tabA: string
    tabB: string
    colDate: string
    colKDA: string
    colMode: string
    colPerf: string
    win: string
    loss: string
    other: string
    empty: string
  }
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  } catch {
    return iso
  }
}

function OutcomeBadge({ outcome, labels }: { outcome: number | null | undefined; labels: { win: string; loss: string; other: string } }) {
  if (outcome === OUTCOME_WIN)
    return <span className="rounded px-1.5 py-0.5 text-[10px] font-bold bg-success/20 text-success">{labels.win}</span>
  if (outcome === OUTCOME_LOSS)
    return <span className="rounded px-1.5 py-0.5 text-[10px] font-bold bg-destructive/20 text-destructive">{labels.loss}</span>
  return <span className="text-muted-foreground text-xs">{labels.other}</span>
}

function MatchTable({
  matches,
  labels,
  side,
}: {
  matches: SessionDetailMatchRow[]
  labels: SessionCompareMatchHistoryProps['labels']
  side: 'a' | 'b'
}) {
  const colorClass = side === 'a' ? 'text-compare-a' : 'text-compare-b'

  if (matches.length === 0) {
    return <p className="text-sm text-muted-foreground italic text-center py-4">{labels.empty}</p>
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-xs text-muted-foreground">
            <th className="py-2 pr-3 text-left">{labels.colDate}</th>
            <th className="py-2 pr-3 text-center">{''}</th>
            <th className={`py-2 pr-3 text-right ${colorClass}`}>{labels.colKDA}</th>
            <th className="py-2 pr-3 text-left">{labels.colMode}</th>
            <th className={`py-2 text-right ${colorClass}`}>{labels.colPerf}</th>
          </tr>
        </thead>
        <tbody>
          {matches.map((m) => {
            const kda = m.kda != null ? m.kda.toFixed(2) : `${m.kills}/${m.deaths}/${m.assists}`
            const mode = [m.pair_name, m.playlist_name].filter(Boolean).join(' · ') || '—'
            const perf = m.performance_score != null ? m.performance_score.toFixed(1) : '—'
            return (
              <tr key={m.match_id} className="border-b last:border-0 hover:bg-muted/20">
                <td className="py-1.5 pr-3 text-xs text-muted-foreground whitespace-nowrap">
                  {formatDate(m.start_time)}
                </td>
                <td className="py-1.5 pr-3 text-center">
                  <OutcomeBadge outcome={m.outcome} labels={labels} />
                </td>
                <td className="py-1.5 pr-3 text-right tabular-nums font-mono text-xs">
                  {kda}
                </td>
                <td className="py-1.5 pr-3 text-xs text-muted-foreground max-w-[140px] truncate">
                  {mode}
                </td>
                <td className="py-1.5 text-right tabular-nums text-xs">
                  {perf}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

export function SessionCompareMatchHistory({
  matchesA,
  matchesB,
  labels,
}: SessionCompareMatchHistoryProps) {
  const [activeTab, setActiveTab] = useState<'a' | 'b'>('a')

  const tabBase = 'px-4 py-2 text-sm font-medium border-b-2 transition-colors'
  const tabActive = 'border-foreground text-foreground'
  const tabInactive = 'border-transparent text-muted-foreground hover:text-foreground'

  return (
    <div>
      <div className="flex border-b mb-4">
        <button
          className={`${tabBase} ${activeTab === 'a' ? `${tabActive} text-compare-a border-compare-a` : tabInactive}`}
          onClick={() => setActiveTab('a')}
        >
          {labels.tabA}
          <span className="ml-1.5 text-xs opacity-60">({matchesA.length})</span>
        </button>
        <button
          className={`${tabBase} ${activeTab === 'b' ? `${tabActive} text-compare-b border-compare-b` : tabInactive}`}
          onClick={() => setActiveTab('b')}
        >
          {labels.tabB}
          <span className="ml-1.5 text-xs opacity-60">({matchesB.length})</span>
        </button>
      </div>
      {activeTab === 'a' ? (
        <MatchTable matches={matchesA} labels={labels} side="a" />
      ) : (
        <MatchTable matches={matchesB} labels={labels} side="b" />
      )}
    </div>
  )
}
