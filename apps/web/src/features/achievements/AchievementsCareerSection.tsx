/**
 * AchievementsCareerSection — section achievements Xbox.
 *
 * layout="carousel" (défaut) : scroll horizontal, cartes fixes w-56.
 * layout="sidebar"           : colonne verticale compacte, overflow-y-auto,
 *                              cartes pleine largeur — pour un slot droit
 *                              aux côtés des charts XP/LUSR.
 */
import { useState } from 'react'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAchievementsPage } from './queries'
import { AchievementCard } from './AchievementCard'
import { ACHIEVEMENTS_TEXT, type AchievementsLocale, type AchievementsText } from './i18n'

type StatusFilter = 'all' | 'unlocked' | 'in-progress' | 'not-started'
type DateSort = 'default' | 'asc' | 'desc'

interface Props {
  playerSlug: string
  layout?: 'carousel' | 'sidebar'
  /** Si fourni, seuls les achievements dont xbox_title_id correspond (ou est vide) sont affichés. */
  filterXboxTitleId?: string
}

export function AchievementsCareerSection({ playerSlug, layout = 'carousel', filterXboxTitleId }: Props) {
  const locale = useAppShellStore((s) => s.locale) as AchievementsLocale
  const t = ACHIEVEMENTS_TEXT[locale]
  const { data, isLoading, isError, refetch } = useAchievementsPage(playerSlug)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all')
  const [dateSort, setDateSort] = useState<DateSort>('default')

  if (isLoading || isError || !data) {
    return renderShellWith(
      t,
      layout,
      isError ? (
        <div className="py-2 text-center">
          <p className="text-sm font-medium text-destructive">{t.loadError}</p>
          <button onClick={() => refetch()} className="mt-1 text-xs text-primary underline">
            {t.retry}
          </button>
        </div>
      ) : null,
    )
  }

  if (data.summary.total_count === 0) {
    return renderShellWith(
      t,
      layout,
      <EmptyStateNotice title={t.empty} description={t.emptyHint} />,
    )
  }

  const baseList = filterXboxTitleId
    ? data.achievements.filter((a) => !a.xbox_title_id || a.xbox_title_id === filterXboxTitleId)
    : data.achievements

  const statusFiltered =
    statusFilter === 'all' ? baseList
    : statusFilter === 'unlocked' ? baseList.filter((a) => a.unlocked)
    : statusFilter === 'in-progress'
      ? baseList.filter((a) => !a.unlocked && (a.current_progress ?? 0) > 0)
      : baseList.filter((a) => !a.unlocked && (a.current_progress ?? 0) === 0)

  const visible =
    dateSort === 'default'
      ? statusFiltered
      : [...statusFiltered].sort((a, b) => {
          const at = a.unlocked_at ? new Date(a.unlocked_at).getTime() : null
          const bt = b.unlocked_at ? new Date(b.unlocked_at).getTime() : null
          if (at === null && bt === null) return 0
          if (at === null) return 1
          if (bt === null) return -1
          return dateSort === 'asc' ? at - bt : bt - at
        })

  const summary = data.summary

  if (layout === 'sidebar') {
    return (
      <div className="relative flex flex-col rounded-lg border border-border bg-card">
        <div className="flex items-baseline justify-between gap-2 border-b border-border px-3 py-2">
          <span className="text-sm font-medium">{t.sectionTitle}</span>
          <span className="text-xs text-muted-foreground">
            {summary.unlocked_count}/{summary.total_count} · {summary.completion_pct.toFixed(0)} %
          </span>
        </div>
        <div className="flex items-center justify-between px-3 pb-1 pt-1">
          <span className="text-xs text-muted-foreground">
            {summary.earned_gamerscore} / {summary.total_gamerscore} G
          </span>
          <div className="flex items-center gap-1.5">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
              className="cursor-pointer border-0 bg-transparent text-[10px] text-muted-foreground outline-none"
            >
              <option value="all">{t.filterAll}</option>
              <option value="unlocked">{t.filterUnlocked}</option>
              <option value="in-progress">{t.filterInProgress}</option>
              <option value="not-started">{t.filterNotStarted}</option>
            </select>
            <select
              value={dateSort}
              onChange={(e) => setDateSort(e.target.value as DateSort)}
              className="cursor-pointer border-0 bg-transparent text-[10px] text-muted-foreground outline-none"
            >
              <option value="default">{t.sortDefault}</option>
              <option value="asc">{t.sortDateAsc}</option>
              <option value="desc">{t.sortDateDesc}</option>
            </select>
          </div>
        </div>
        <div className="p-3 pt-1">
          <div
            className="flex flex-col gap-2 overflow-y-auto"
            style={{ maxHeight: '640px' }}
            role="list"
            aria-label={t.sectionTitle}
          >
            {visible.map((a) => (
              <div role="listitem" key={a.achievement_id}>
                <AchievementCard achievement={a} locale={locale} fixedWidth={false} />
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="relative rounded-lg border border-border bg-card">
      <div className="flex flex-col gap-2 border-b border-border px-3 py-2 md:flex-row md:items-center md:justify-between">
        <span className="text-sm font-medium">{t.sectionTitle}</span>
        <SummaryInline summary={summary} t={t} />
      </div>
      <div className="p-3">
        <div
          className="flex snap-x snap-mandatory gap-3 overflow-x-auto pb-2"
          role="list"
          aria-label={t.sectionTitle}
        >
          {visible.map((a) => (
            <div role="listitem" key={a.achievement_id}>
              <AchievementCard achievement={a} locale={locale} />
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function renderShellWith(t: AchievementsText, layout: 'carousel' | 'sidebar', body: React.ReactNode) {
  if (layout === 'sidebar') {
    return (
      <div className="relative rounded-lg border border-border bg-card">
        <div className="border-b border-border px-3 py-2 text-sm font-medium">{t.sectionTitle}</div>
        <div className="p-3">{body}</div>
      </div>
    )
  }
  return (
    <div className="relative rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2 text-sm font-medium">{t.sectionTitle}</div>
      <div className="p-3">{body}</div>
    </div>
  )
}

interface SummaryInlineProps {
  summary: {
    unlocked_count: number
    total_count: number
    earned_gamerscore: number
    total_gamerscore: number
    completion_pct: number
  }
  t: AchievementsText
}

function SummaryInline({ summary, t }: SummaryInlineProps) {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
      <span>
        <span className="text-muted-foreground">{t.summaryUnlocked} : </span>
        <span className="font-semibold text-foreground">
          {summary.unlocked_count} / {summary.total_count}
        </span>
      </span>
      <span aria-hidden="true">·</span>
      <span>
        <span className="text-muted-foreground">{t.summaryGamerscore} : </span>
        <span className="font-semibold text-foreground">
          {summary.earned_gamerscore} / {summary.total_gamerscore} G
        </span>
      </span>
      <span aria-hidden="true">·</span>
      <span>
        <span className="text-muted-foreground">{t.summaryCompletion} : </span>
        <span className="font-semibold text-primary">{summary.completion_pct.toFixed(1)} %</span>
      </span>
    </div>
  )
}
