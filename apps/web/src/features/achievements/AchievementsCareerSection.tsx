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
type CategoryFilter = 'all' | 'multiplayer' | 'campaign' | 'other'
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
  // Multijoueur par défaut : c'est la catégorie pertinente pour le dashboard.
  // Sans effet sur un titre sans mapping (filtre inactif tant que hasCategories est faux).
  const [categoryFilter, setCategoryFilter] = useState<CategoryFilter>('multiplayer')
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

  const achievements = data.achievements ?? []
  const baseList = filterXboxTitleId
    ? achievements.filter((a) => !a.xbox_title_id || a.xbox_title_id === filterXboxTitleId)
    : achievements

  // Titre sans mapping de catégories (champ absent partout) → filtre masqué.
  const hasCategories = achievements.some((a) => !!a.category)

  const categoryFiltered =
    !hasCategories || categoryFilter === 'all'
      ? baseList
      : baseList.filter((a) => a.category === categoryFilter)

  const statusFiltered =
    statusFilter === 'all' ? categoryFiltered
    : statusFilter === 'unlocked' ? categoryFiltered.filter((a) => a.unlocked)
    : statusFilter === 'in-progress'
      ? categoryFiltered.filter((a) => !a.unlocked && (a.current_progress ?? 0) > 0)
      : categoryFiltered.filter((a) => !a.unlocked && (a.current_progress ?? 0) === 0)

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
      <div className="relative flex h-full flex-col rounded-lg border border-border bg-card">
        <div className="flex items-baseline justify-between gap-2 border-b border-border px-3 py-2">
          <span className="text-sm font-medium">{t.sectionTitle}</span>
          <span className="text-xs text-muted-foreground">
            {summary.unlocked_count}/{summary.total_count} · {summary.completion_pct.toFixed(0)} %
          </span>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-x-2 gap-y-1 px-3 pb-1 pt-1">
          <span className="whitespace-nowrap text-xs tabular-nums text-muted-foreground">
            {summary.earned_gamerscore} / {summary.total_gamerscore} G
          </span>
          <div className="flex items-center gap-1.5">
            {hasCategories ? (
              <select
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value as CategoryFilter)}
                className="cursor-pointer border-0 bg-card text-2xs text-muted-foreground outline-none"
              >
                <option value="all">{t.filterCategoryAll}</option>
                <option value="multiplayer">{t.filterCategoryMultiplayer}</option>
                <option value="campaign">{t.filterCategoryCampaign}</option>
                <option value="other">{t.filterCategoryOther}</option>
              </select>
            ) : null}
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
              className="cursor-pointer border-0 bg-card text-2xs text-muted-foreground outline-none"
            >
              <option value="all">{t.filterAll}</option>
              <option value="unlocked">{t.filterUnlocked}</option>
              <option value="in-progress">{t.filterInProgress}</option>
              <option value="not-started">{t.filterNotStarted}</option>
            </select>
            <select
              value={dateSort}
              onChange={(e) => setDateSort(e.target.value as DateSort)}
              className="cursor-pointer border-0 bg-card text-2xs text-muted-foreground outline-none"
            >
              <option value="default">{t.sortDefault}</option>
              <option value="asc">{t.sortDateAsc}</option>
              <option value="desc">{t.sortDateDesc}</option>
            </select>
          </div>
        </div>
        <div className="flex min-h-0 flex-1 flex-col p-3 pt-1">
          <div
            className="flex h-full max-h-[70vh] flex-col gap-2 overflow-y-auto xl:max-h-none"
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
