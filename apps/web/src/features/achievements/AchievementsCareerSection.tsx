/**
 * AchievementsCareerSection — section achievements Xbox.
 *
 * layout="carousel" (défaut) : scroll horizontal, cartes fixes w-56.
 * layout="sidebar"           : colonne verticale compacte, overflow-y-auto,
 *                              cartes pleine largeur — pour un slot droit
 *                              aux côtés des charts XP/LUSR.
 */
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAchievementsPage } from './queries'
import { AchievementCard } from './AchievementCard'
import { ACHIEVEMENTS_TEXT, type AchievementsLocale, type AchievementsText } from './i18n'

const VISIBLE_LIMIT = 30

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

  const filtered = filterXboxTitleId
    ? data.achievements.filter((a) => !a.xbox_title_id || a.xbox_title_id === filterXboxTitleId)
    : data.achievements
  const visible = filtered.slice(0, VISIBLE_LIMIT)
  const summary = data.summary

  if (layout === 'sidebar') {
    return (
      <Card className="flex flex-col">
        <CardHeader className="pb-2 pt-3">
          <div className="flex items-baseline justify-between gap-2">
            <h2 className="text-sm font-semibold text-foreground">{t.sectionTitle}</h2>
            <span className="text-xs text-muted-foreground">
              {summary.unlocked_count}/{summary.total_count} · {summary.completion_pct.toFixed(0)} %
            </span>
          </div>
          <p className="text-xs text-muted-foreground">
            {summary.earned_gamerscore} / {summary.total_gamerscore} G
          </p>
        </CardHeader>
        <CardContent className="flex-1 overflow-hidden p-2">
          <div
            className="flex flex-col gap-2 overflow-y-auto"
            style={{ maxHeight: '680px' }}
            role="list"
            aria-label={t.sectionTitle}
          >
            {visible.map((a) => (
              <div role="listitem" key={a.achievement_id}>
                <AchievementCard achievement={a} locale={locale} fixedWidth={false} />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className="flex flex-col gap-2 pb-3 md:flex-row md:items-center md:justify-between">
        <h2 className="text-base font-semibold text-foreground">{t.sectionTitle}</h2>
        <SummaryInline summary={summary} t={t} />
      </CardHeader>
      <CardContent>
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
      </CardContent>
    </Card>
  )
}

function renderShellWith(t: AchievementsText, layout: 'carousel' | 'sidebar', body: React.ReactNode) {
  if (layout === 'sidebar') {
    return (
      <Card className="flex flex-col">
        <CardHeader className="pb-2 pt-3">
          <h2 className="text-sm font-semibold text-foreground">{t.sectionTitle}</h2>
        </CardHeader>
        <CardContent>{body}</CardContent>
      </Card>
    )
  }
  return (
    <Card>
      <CardHeader className="pb-3">
        <h2 className="text-base font-semibold text-foreground">{t.sectionTitle}</h2>
      </CardHeader>
      <CardContent>{body}</CardContent>
    </Card>
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
