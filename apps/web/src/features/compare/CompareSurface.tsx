import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { Spinner } from '@/components/ui/spinner'
import type { CompareMetricRow, NormalizedPlayerStats } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'

import { getCompareText, normalizeCompareLocale, type CompareText } from './i18n'
import { useCompare } from './queries'

function formatMetricValue(metric: string, value: number | string, text: CompareText) {
  if (typeof value !== 'number') {
    return String(value)
  }
  if (metric === 'win_rate' || metric === 'accuracy') {
    return `${value.toLocaleString(text.intlLocale, { maximumFractionDigits: 1 })} %`
  }
  if (metric === 'matches' || metric === 'csr_current' || metric === 'csr_best' || metric === 'career_rank') {
    return value.toLocaleString(text.intlLocale, { maximumFractionDigits: 0 })
  }
  return value.toLocaleString(text.intlLocale, { minimumFractionDigits: 1, maximumFractionDigits: 2 })
}

function PlayerStatRow({ label, value }: { label: string; value: string }) {
  return (
    <p>
      <span className="font-medium">{label}:</span> {value}
    </p>
  )
}

function PlayerStatsCard({ stats, side, text }: { stats: NormalizedPlayerStats; side: 'A' | 'B'; text: CompareText }) {
  const colorClass = side === 'A' ? 'text-compare-a' : 'text-compare-b'
  const bgClass = side === 'A' ? 'bg-compare-a/10 border-compare-a' : 'bg-compare-b/10 border-compare-b'

  return (
    <Card className={`border ${bgClass}`}>
      <CardHeader className="pb-2">
        <CardTitle className={`text-sm ${colorClass}`}>
          {stats.gamertag}
          {stats.is_local && (
            <Badge variant="secondary" className="ml-2 text-xs">
              {text.localBadge}
            </Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 text-xs text-foreground">
        <PlayerStatRow label={text.stats.matches} value={stats.matches.toLocaleString(text.intlLocale)} />
        <PlayerStatRow label={text.stats.winRate} value={`${(stats.win_rate * 100).toLocaleString(text.intlLocale, { maximumFractionDigits: 1 })} %`} />
        <PlayerStatRow label={text.stats.kda} value={stats.kda.toLocaleString(text.intlLocale, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} />
        <PlayerStatRow label={text.stats.kdr} value={stats.kdr.toLocaleString(text.intlLocale, { minimumFractionDigits: 2, maximumFractionDigits: 2 })} />
        <PlayerStatRow label={text.stats.accuracy} value={`${(stats.accuracy * 100).toLocaleString(text.intlLocale, { maximumFractionDigits: 1 })} %`} />
        <PlayerStatRow label={text.stats.killsPerGame} value={stats.kills_per_game.toLocaleString(text.intlLocale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })} />
        <PlayerStatRow label={text.stats.currentCsr} value={stats.csr_current.toLocaleString(text.intlLocale)} />
      </CardContent>
    </Card>
  )
}

function MetricRowComp({ row, text }: { row: CompareMetricRow; text: CompareText }) {
  const winnerA = row.winner === 'a'
  const winnerB = row.winner === 'b'
  const label = text.metrics[row.metric] ?? row.label_fr

  return (
    <tr className="border-b last:border-0 text-sm">
      <td className={`py-2 pr-4 ${winnerA ? 'font-semibold text-compare-a' : 'text-foreground'}`}>
        {formatMetricValue(row.metric, row.value_a, text)}
      </td>
      <td className="py-2 px-2 text-muted-foreground text-center text-xs">{label}</td>
      <td className={`py-2 pl-4 text-right ${winnerB ? 'font-semibold text-compare-b' : 'text-foreground'}`}>
        {formatMetricValue(row.metric, row.value_b, text)}
      </td>
    </tr>
  )
}

export function CompareSurface({ playerSlug }: { playerSlug: string }) {
  const locale = normalizeCompareLocale(useAppShellStore((state) => state.locale))
  const { data: fieldMappings } = useFieldMappings()
  const text = getCompareText(locale, fieldMappings)
  const [targetGamertag, setTargetGamertag] = useState('')
  const { mutate, data, isPending, isError, error, reset } = useCompare(playerSlug)

  function handleSubmit(event: React.FormEvent) {
    event.preventDefault()
    if (!targetGamertag.trim()) {
      return
    }
    reset()
    mutate({ target_gamertag: targetGamertag.trim() })
  }

  return (
    <div className="space-y-4">
      <form onSubmit={handleSubmit} className="space-y-2 border-b pb-4">
        <label className="block text-sm font-medium text-foreground">{text.formLabel}</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={targetGamertag}
            onChange={(event) => setTargetGamertag(event.target.value)}
            placeholder={text.placeholder}
            className="flex-1 rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
          <button
            type="submit"
            disabled={isPending || !targetGamertag.trim()}
            className="rounded bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {isPending ? '…' : text.submit}
          </button>
        </div>
      </form>

      {isPending && (
        <div className="flex justify-center py-8">
          <Spinner size="lg" label={text.loading} />
        </div>
      )}

      {isError && (
        <EmptyStateCard
          title={error?.message?.includes('404') ? text.notFoundTitle : text.errorTitle}
          description={
            error?.message?.includes('404')
              ? text.notFoundDescription
              : (error?.message ?? text.errorDescription)
          }
        />
      )}

      {data && (
        <div className="space-y-4">
          <PrivacyBanner warning={data.privacy_warning} />
          {data.player_b_partial && !data.privacy_warning && (
            <p className="rounded bg-warning/10 px-3 py-2 text-xs text-warning">
              {text.partialWarning(data.player_b.gamertag)}
            </p>
          )}

          <div className="grid gap-3 md:grid-cols-2">
            <PlayerStatsCard stats={data.player_a} side="A" text={text} />
            <PlayerStatsCard stats={data.player_b} side="B" text={text} />
          </div>

          {data.metrics.length > 0 && (
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm">{text.detailsTitle}</CardTitle>
              </CardHeader>
              <CardContent className="p-0">
                <table className="w-full">
                  <thead>
                    <tr className="border-b text-xs text-muted-foreground">
                      <th className="py-2 pr-4 text-left font-medium text-compare-a">{data.player_a.gamertag}</th>
                      <th className="py-2 px-2 text-center font-medium">{text.metricColumn}</th>
                      <th className="py-2 pl-4 text-right font-medium text-compare-b">{data.player_b.gamertag}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.metrics.map((row) => (
                      <MetricRowComp key={row.metric} row={row} text={text} />
                    ))}
                  </tbody>
                </table>
              </CardContent>
            </Card>
          )}
        </div>
      )}

      {!isPending && !isError && !data && (
        <p className="py-8 text-center text-sm text-muted-foreground">{text.emptyPrompt}</p>
      )}
    </div>
  )
}
