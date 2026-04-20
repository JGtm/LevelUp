import { useParams } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SeasonPassStatus, SeasonPassTrackSummary } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getPalmaresText, normalizePalmaresLocale } from './i18n'
import { PalmaresShell } from './PalmaresShell'
import { useSeasonPassPage } from './queries'

function statusVariant(status: SeasonPassStatus) {
  switch (status) {
    case 'active':
      return 'default' as const
    case 'completed':
      return 'success' as const
    case 'in_progress':
      return 'secondary' as const
    default:
      return 'outline' as const
  }
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className="border-dashed">
      <CardContent className="pt-5">
        <p className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
        <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
      </CardContent>
    </Card>
  )
}

function ProgressBar({ value }: { value?: number | null }) {
  const width = value == null ? 0 : Math.max(0, Math.min(100, value))
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
      <div className="h-full rounded-full bg-primary transition-[width]" style={{ width: `${width}%` }} />
    </div>
  )
}

function SeasonPassCard({ pass, intlLocale, statusLabel, labels }: {
  pass: SeasonPassTrackSummary
  intlLocale: string
  statusLabel: string
  labels: { premium: string; active: string; rank: string; progress: string }
}) {
  const background = pass.background_image_url ?? pass.image_url ?? null
  const rankValue = pass.max_rank ? `${pass.current_rank}/${pass.max_rank}` : pass.current_rank.toLocaleString(intlLocale)

  return (
    <Card className="relative overflow-hidden border-border/70 bg-card/95 shadow-sm">
      {background && (
        <div
          className="absolute inset-0 bg-cover bg-center opacity-15"
          style={{ backgroundImage: `url(${background})` }}
          aria-hidden="true"
        />
      )}
      <div className="absolute inset-0 bg-gradient-to-br from-background via-background/92 to-background/75" aria-hidden="true" />
      <CardContent className="relative flex h-full flex-col gap-4 pt-6">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <p className="text-xs uppercase tracking-[0.24em] text-muted-foreground">{pass.reward_track_path.split('/').slice(-1)[0]}</p>
            <h3 className="mt-2 text-lg font-semibold text-foreground">{pass.name}</h3>
          </div>
          <div className="flex flex-wrap gap-2">
            {pass.is_active && <Badge variant="default">{labels.active}</Badge>}
            {pass.is_owned && <Badge variant="outline">{labels.premium}</Badge>}
            <Badge variant={statusVariant(pass.status)}>{statusLabel}</Badge>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <p className="text-xs text-muted-foreground">{labels.rank}</p>
            <p className="mt-1 text-xl font-semibold text-foreground">{rankValue}</p>
          </div>
          <div>
            <p className="text-xs text-muted-foreground">{labels.progress}</p>
            <p className="mt-1 text-xl font-semibold text-foreground">
              {pass.completion_percent == null
                ? '—'
                : `${pass.completion_percent.toLocaleString(intlLocale, { maximumFractionDigits: 0 })} %`}
            </p>
          </div>
        </div>

        <ProgressBar value={pass.completion_percent} />
      </CardContent>
    </Card>
  )
}

export function SeasonPassPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)
  const { data, isLoading, isError, error, refetch } = useSeasonPassPage(playerSlug)

  if (isLoading) {
    return (
      <PalmaresShell playerSlug={playerSlug} activeTab="season-pass">
        <div className="flex items-center justify-center py-24">
          <Spinner size="lg" />
        </div>
      </PalmaresShell>
    )
  }

  if (isError || !data) {
    return (
      <PalmaresShell playerSlug={playerSlug} activeTab="season-pass">
        <EmptyStateCard
          title={text.seasonPass.unavailableTitle}
          description={error?.message ?? text.seasonPass.unavailableDescription}
          actionLabel={text.seasonPass.retry}
          onAction={() => refetch()}
        />
      </PalmaresShell>
    )
  }

  const completedCount = data.passes.filter((pass) => pass.status === 'completed').length
  const inProgressCount = data.passes.filter((pass) => pass.status === 'in_progress').length
  const activePass = data.passes.find((pass) => pass.is_active)

  return (
    <PalmaresShell playerSlug={playerSlug} activeTab="season-pass">
      <div className="grid gap-4 xl:grid-cols-4">
        <StatCard label={text.seasonPass.activeCard} value={activePass?.name ?? '—'} />
        <StatCard label={text.seasonPass.completedCard} value={completedCount.toLocaleString(text.intlLocale)} />
        <StatCard label={text.seasonPass.inProgressCard} value={inProgressCount.toLocaleString(text.intlLocale)} />
        <StatCard label={text.seasonPass.challengesCard} value={(data.challenges.total ?? 0).toLocaleString(text.intlLocale)} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{text.seasonPass.challengesTitle}</CardTitle>
        </CardHeader>
        <CardContent>
          {data.challenges.available ? (
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.completed}</p>
                <p className="mt-1 text-lg font-semibold">{(data.challenges.completed ?? 0).toLocaleString(text.intlLocale)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.total}</p>
                <p className="mt-1 text-lg font-semibold">{(data.challenges.total ?? 0).toLocaleString(text.intlLocale)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.xpAvailable}</p>
                <p className="mt-1 text-lg font-semibold">{(data.challenges.xp_available ?? 0).toLocaleString(text.intlLocale)}</p>
              </div>
              <div>
                <p className="text-xs text-muted-foreground">{text.seasonPass.nextExpiry}</p>
                <p className="mt-1 text-lg font-semibold">
                  {data.challenges.next_expiry
                    ? new Date(data.challenges.next_expiry).toLocaleString(text.intlLocale)
                    : text.seasonPass.noExpiry}
                </p>
              </div>
            </div>
          ) : (
            <EmptyStateNotice title={text.seasonPass.challengesUnavailable} description={data.challenges.error_hint ?? text.seasonPass.unavailableDescription} />
          )}
        </CardContent>
      </Card>

      {data.passes.length === 0 ? (
        <EmptyStateCard title={text.seasonPass.noPassesTitle} description={text.seasonPass.noPassesDescription} />
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          {data.passes.map((pass) => (
            <SeasonPassCard
              key={pass.reward_track_path}
              pass={pass}
              intlLocale={text.intlLocale}
              statusLabel={text.seasonPass.status[pass.status] ?? pass.status}
              labels={{
                premium: text.seasonPass.premium,
                active: text.seasonPass.active,
                rank: text.seasonPass.cardRank,
                progress: text.seasonPass.cardProgress,
              }}
            />
          ))}
        </div>
      )}
    </PalmaresShell>
  )
}
