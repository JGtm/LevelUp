import { useMemo, useState } from 'react'
import type { ChallengeItem } from '@/lib/api/types'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { CADENCE_DAILY_FALLBACK_FR, CADENCE_WEEKLY_FALLBACK_FR } from './fallback.i18n'

type ChallengeCadence = 'daily' | 'weekly' | 'capstone' | null
type ChallengeCategory = 'daily' | 'weekly'
type ChallengeSection = {
  kind: ChallengeCategory
  /** Clé canonique de cadence (alignée sur assets.toml). Le label est résolu au render. */
  cadenceKey: string
  items: ChallengeItem[]
}

function challengeSortScore(item: ChallengeItem): number {
  if (item.progress_percent != null) {
    return item.progress_percent
  }
  if (item.progress_current != null && item.progress_current > 0) {
    return 0.001 + item.progress_current / 10000
  }
  return 0
}

function deriveChallengeCadence(challengePath?: string | null): ChallengeCadence {
  const normalizedPath = (challengePath ?? '').toLowerCase()
  if (!normalizedPath) {
    return null
  }
  if (normalizedPath.includes('dailychallenges')) {
    return 'daily'
  }
  if (
    normalizedPath.includes('weeklychallenges') ||
    normalizedPath.includes('winterchallenges') ||
    normalizedPath.includes('seasonalchallenges') ||
    normalizedPath.includes('eventchallenges') ||
    normalizedPath.includes('operationchallenges') ||
    normalizedPath.includes('fracturechallenges')
  ) {
    return 'weekly'
  }
  if (normalizedPath.includes('ultimate') || normalizedPath.includes('capstone')) {
    return 'capstone'
  }
  return null
}

// Catégorie de défi → clé de cadence dans assets.toml.
// Le libellé est résolu au runtime par le composant via useAssetLabel('cadence', key).
function cadenceKeyOf(category: ChallengeCategory): string {
  return category === 'daily' ? 'daily' : 'weekly'
}

function challengeCategoryClasses(category: ChallengeCategory): string {
  if (category === 'daily') {
    return 'border-amber-500/25 bg-amber-500/8'
  }
  return 'border-sky-500/25 bg-sky-500/8'
}

function challengeCategoryForItem(item: ChallengeItem): ChallengeCategory {
  if (deriveChallengeCadence(item.challenge_path) === 'daily') {
    return 'daily'
  }
  return 'weekly'
}

function buildChallengeSections(items: ChallengeItem[]): ChallengeSection[] {
  const dailyItems: ChallengeItem[] = []
  const weeklyItems: ChallengeItem[] = []

  for (const item of items) {
    if (challengeCategoryForItem(item) === 'daily') {
      dailyItems.push(item)
      continue
    }
    weeklyItems.push(item)
  }

  const sections: ChallengeSection[] = []
  if (dailyItems.length > 0) {
    sections.push({
      kind: 'daily',
      cadenceKey: cadenceKeyOf('daily'),
      items: dailyItems,
    })
  }
  if (weeklyItems.length > 0) {
    sections.push({
      kind: 'weekly',
      cadenceKey: cadenceKeyOf('weekly'),
      items: weeklyItems,
    })
  }
  return sections
}

function challengeSectionDividerClasses(): string {
  return 'bg-white/90'
}

function ChallengeThumb({
  imageUrl,
  title,
}: {
  imageUrl?: string | null
  title: string
}) {
  const [imageFailed, setImageFailed] = useState(false)

  if (!imageUrl || imageFailed) {
    return (
      <div
        data-testid="home-challenge-thumb"
        className="flex h-16 w-16 shrink-0 items-center justify-center rounded-md text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground"
      >
        Défi
      </div>
    )
  }

  return (
    <div data-testid="home-challenge-thumb" className="h-16 w-16 shrink-0 overflow-hidden rounded-md">
      <img
        src={imageUrl}
        alt={title}
        className="h-full w-full object-cover"
        onError={() => setImageFailed(true)}
      />
    </div>
  )
}

export function HomeChallengesList({ items }: { items: ChallengeItem[] }) {
  const sortedItems = useMemo(
    () => [...items].sort((left, right) => {
      const scoreDelta = challengeSortScore(right) - challengeSortScore(left)
      if (scoreDelta !== 0) {
        return scoreDelta
      }
      const currentDelta = (right.progress_current ?? 0) - (left.progress_current ?? 0)
      if (currentDelta !== 0) {
        return currentDelta
      }
      return left.title.localeCompare(right.title, 'fr')
    }),
    [items],
  )
  const sections = useMemo(() => buildChallengeSections(sortedItems), [sortedItems])

  return (
    <div className="space-y-5">
      {sections.map((section) => (
        <ChallengeSection key={section.kind} section={section} />
      ))}
    </div>
  )
}

// ChallengeSection : sub-component pour pouvoir appeler useAssetLabel par section.
function ChallengeSection({ section }: { section: ChallengeSection }) {
  // Phase 4 plan finition multi-titres : libellé cadence via assets.toml.
  const cadenceLabel = useAssetLabel('cadence', section.cadenceKey)
  const fallback = section.cadenceKey === 'daily' ? CADENCE_DAILY_FALLBACK_FR : CADENCE_WEEKLY_FALLBACK_FR
  const label = cadenceLabel !== section.cadenceKey ? cadenceLabel : fallback
  return (
    <section data-testid={`home-challenge-section-${section.kind}`} className="space-y-3">
      <div className="space-y-2">
        <p data-testid="home-challenge-section-title" className="text-[11px] font-semibold uppercase tracking-[0.18em] text-foreground/90">
          {label}
        </p>
            <div className={`h-px w-full rounded-full ${challengeSectionDividerClasses()}`} />
          </div>

          <div className="space-y-3">
            {section.items.map((item) => {
              const progressPercent = Math.max(0, Math.min(100, item.progress_percent ?? 0))
              const current = item.progress_current ?? 0
              const target = item.progress_target

              return (
                <div
                  key={item.tracking_id ?? item.challenge_path}
                  data-testid="home-challenge-item"
                  className={`flex items-stretch gap-3 rounded-lg border p-3 ${challengeCategoryClasses(section.kind)}`}
                >
                  <ChallengeThumb imageUrl={item.image_url} title={item.title} />

                  <div className="flex min-h-16 min-w-0 flex-1 flex-col justify-between gap-2">
                    <div className="min-w-0">
                      <p data-testid="home-challenge-title" className="truncate text-[15px] font-semibold leading-tight text-foreground">
                        {item.title}
                      </p>
                      {item.description && (
                        <p data-testid="home-challenge-description" className="mt-1 line-clamp-2 text-xs italic text-muted-foreground">
                          {item.description}
                        </p>
                      )}
                    </div>

                    <div
                      data-testid="home-challenge-progress-row"
                      className="mt-auto grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2 text-[11px] text-muted-foreground"
                    >
                      <span data-testid="home-challenge-progress-current" className="shrink-0 whitespace-nowrap">
                        {target != null ? `${current} / ${target}` : current}
                      </span>
                      <div data-testid="home-challenge-progress-track" className="min-w-0 overflow-hidden rounded-full bg-muted-foreground/25">
                        <div className="h-2 w-full">
                          <div
                            data-testid="home-challenge-progress-fill"
                            className="h-full rounded-full bg-sky-500 transition-all duration-300"
                            style={{ width: `${progressPercent}%` }}
                          />
                        </div>
                      </div>
                      <span data-testid="home-challenge-progress-percent" className="shrink-0 whitespace-nowrap text-right">
                        {Math.round(progressPercent)}%
                      </span>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </section>
  )
}
