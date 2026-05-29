import { useParams, useNavigate } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { RelationInsight } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
// P8.12 (revue 2026-04-29) : helpers canoniques (formatPercent/formatKDA/formatDate)
// au lieu d'helpers ad-hoc locaux. Cf. apps/web/src/lib/formatters/.
import { formatPercent, formatKDA, formatDate } from '@/lib/formatters'

import { getPalmaresText, normalizePalmaresLocale } from './i18n'
import { useRelationsPage } from './queries'

type RelationVariant = 'allies' | 'synergy' | 'nemesis' | 'victim' | 'circle'

function OverviewCard({ label, value }: { label: string; value: string }) {
  return (
    <Card className="border-dashed">
      <CardContent className="pt-5">
        <p className="text-xs uppercase tracking-label-md text-muted-foreground">{label}</p>
        <p className="mt-2 text-2xl font-semibold text-foreground">{value}</p>
      </CardContent>
    </Card>
  )
}

function RelationRow({
  playerSlug,
  entry,
  variant,
  locale,
  labels,
}: {
  playerSlug: string
  entry: RelationInsight
  variant: RelationVariant
  locale: string
  labels: { with: string; against: string; winRate: string; avgKDA: string; lastSeen: string }
}) {
  const navigate = useNavigate()
  const teammateLine = `${labels.with} ${entry.teammate_matches}`
  const enemyLine = `${labels.against} ${entry.enemy_matches}`

  let contextLine = `${teammateLine} · ${enemyLine}`
  let metricLine = `${labels.lastSeen} ${formatDate(entry.last_seen_at, locale)}`

  if (variant === 'allies' || variant === 'synergy') {
    contextLine = `${teammateLine} · ${labels.winRate} ${formatPercent(entry.teammate_win_rate, 0)}`
    metricLine = `${labels.avgKDA} ${formatKDA(entry.avg_kda_with, locale)} · ${labels.lastSeen} ${formatDate(entry.last_seen_at, locale)}`
  }
  if (variant === 'nemesis' || variant === 'victim') {
    contextLine = `${enemyLine} · ${labels.winRate} ${formatPercent(entry.enemy_win_rate, 0)}`
    metricLine = `${labels.avgKDA} ${formatKDA(entry.avg_kda_against, locale)} · ${labels.lastSeen} ${formatDate(entry.last_seen_at, locale)}`
  }

  function goToExplorer() {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: entry.gamertag },
    })
  }

  return (
    <div className="flex items-start justify-between gap-4 rounded-2xl border border-border/70 bg-background/70 p-4">
      <div className="min-w-0">
        <button
          type="button"
          className="truncate text-sm font-semibold text-foreground hover:text-primary hover:underline transition-colors"
          onClick={goToExplorer}
          title={`Voir l'historique avec ${entry.gamertag}`}
        >
          {entry.gamertag}
        </button>
        <p className="mt-1 text-sm text-muted-foreground">{contextLine}</p>
        <p className="mt-1 text-xs text-muted-foreground">{metricLine}</p>
      </div>

      <div className="flex shrink-0 flex-wrap justify-end gap-2">
        <Badge variant="outline">{entry.total_matches.toLocaleString(locale)} matchs</Badge>
        {entry.teammate_matches > 0 ? <Badge variant="secondary">{teammateLine}</Badge> : null}
        {entry.enemy_matches > 0 ? <Badge variant="destructive">{enemyLine}</Badge> : null}
      </div>
    </div>
  )
}

function RelationshipGroupCard({
  playerSlug,
  title,
  description,
  items,
  variant,
  locale,
  labels,
  emptyTitle,
  emptyDescription,
}: {
  playerSlug: string
  title: string
  description: string
  items: RelationInsight[]
  variant: RelationVariant
  locale: string
  labels: { with: string; against: string; winRate: string; avgKDA: string; lastSeen: string }
  emptyTitle: string
  emptyDescription: string
}) {
  return (
    <Card className="h-full">
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <p className="text-sm text-muted-foreground">{description}</p>
      </CardHeader>
      <CardContent className="space-y-3">
        {items.length === 0 ? (
          <EmptyStateNotice title={emptyTitle} description={emptyDescription} />
        ) : (
          items.map((entry) => (
            <RelationRow
              key={`${variant}-${entry.xuid}`}
              playerSlug={playerSlug}
              entry={entry}
              variant={variant}
              locale={locale}
              labels={labels}
            />
          ))
        )}
      </CardContent>
    </Card>
  )
}

export function PalmaresRelationsPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)
  const { data, isLoading, isError, error, refetch } = useRelationsPage(playerSlug)

  if (isLoading) {
    return (
      <div className="flex flex-col gap-6 p-6">
        <div className="flex items-center justify-center py-24">
          <Spinner size="lg" />
        </div>
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="flex flex-col gap-6 p-6">
        <EmptyStateCard
          title={text.relations.unavailableTitle}
          description={error?.message ?? text.relations.unavailableDescription}
          actionLabel={text.relations.retry}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const totalEntries =
    data.frequent_allies.length +
    data.best_synergies.length +
    data.nemeses.length +
    data.favorite_victims.length +
    data.closed_circle.length

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="grid gap-4 xl:grid-cols-4" data-testid="palmares-relations-overview">
        <OverviewCard
          label={text.relations.overview.distinctPlayers}
          value={data.overview.distinct_players.toLocaleString(text.intlLocale)}
        />
        <OverviewCard
          label={text.relations.overview.frequentAllies}
          value={data.overview.frequent_allies.toLocaleString(text.intlLocale)}
        />
        <OverviewCard
          label={text.relations.overview.repeatRivals}
          value={data.overview.repeat_rivals.toLocaleString(text.intlLocale)}
        />
        <OverviewCard
          label={text.relations.overview.closedCircle}
          value={data.overview.closed_circle.toLocaleString(text.intlLocale)}
        />
      </div>

      {data.overview.distinct_players === 0 && totalEntries === 0 ? (
        <EmptyStateCard title={text.relations.emptyTitle} description={text.relations.emptyDescription} />
      ) : (
        <div className="grid gap-4 xl:grid-cols-2">
          <RelationshipGroupCard
            playerSlug={playerSlug}
            title={text.relations.sections.frequentAlliesTitle}
            description={text.relations.sections.frequentAlliesDescription}
            items={data.frequent_allies}
            variant="allies"
            locale={text.intlLocale}
            labels={text.relations.labels}
            emptyTitle={text.relations.emptyTitle}
            emptyDescription={text.relations.emptyDescription}
          />
          <RelationshipGroupCard
            playerSlug={playerSlug}
            title={text.relations.sections.bestSynergiesTitle}
            description={text.relations.sections.bestSynergiesDescription}
            items={data.best_synergies}
            variant="synergy"
            locale={text.intlLocale}
            labels={text.relations.labels}
            emptyTitle={text.relations.emptyTitle}
            emptyDescription={text.relations.emptyDescription}
          />
          <RelationshipGroupCard
            playerSlug={playerSlug}
            title={text.relations.sections.nemesesTitle}
            description={text.relations.sections.nemesesDescription}
            items={data.nemeses}
            variant="nemesis"
            locale={text.intlLocale}
            labels={text.relations.labels}
            emptyTitle={text.relations.emptyTitle}
            emptyDescription={text.relations.emptyDescription}
          />
          <RelationshipGroupCard
            playerSlug={playerSlug}
            title={text.relations.sections.victimsTitle}
            description={text.relations.sections.victimsDescription}
            items={data.favorite_victims}
            variant="victim"
            locale={text.intlLocale}
            labels={text.relations.labels}
            emptyTitle={text.relations.emptyTitle}
            emptyDescription={text.relations.emptyDescription}
          />
          <div className="xl:col-span-2">
            <RelationshipGroupCard
              playerSlug={playerSlug}
              title={text.relations.sections.closedCircleTitle}
              description={text.relations.sections.closedCircleDescription}
              items={data.closed_circle}
              variant="circle"
              locale={text.intlLocale}
              labels={text.relations.labels}
              emptyTitle={text.relations.emptyTitle}
              emptyDescription={text.relations.emptyDescription}
            />
          </div>
        </div>
      )}
    </div>
  )
}
