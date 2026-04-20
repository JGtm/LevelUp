import { useMemo, useState } from 'react'
import type { ChallengeItem } from '@/lib/api/types'

type ChallengeCadence = 'daily' | 'weekly' | 'capstone' | null

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

function challengeCadenceLabel(cadence: ChallengeCadence): string | null {
  if (cadence === 'daily') {
    return 'Quotidien'
  }
  if (cadence === 'weekly') {
    return 'Hebdo'
  }
  if (cadence === 'capstone') {
    return 'Capstone'
  }
  return null
}

function challengeCadenceClasses(cadence: ChallengeCadence): string {
  if (cadence === 'daily') {
    return 'border-amber-500/25 bg-amber-500/8'
  }
  if (cadence === 'weekly') {
    return 'border-sky-500/25 bg-sky-500/8'
  }
  if (cadence === 'capstone') {
    return 'border-fuchsia-500/25 bg-fuchsia-500/8'
  }
  return 'border-border/70 bg-muted/35'
}

function challengeCadenceBadgeClasses(cadence: ChallengeCadence): string {
  if (cadence === 'daily') {
    return 'bg-amber-500/14 text-amber-700 dark:text-amber-300'
  }
  if (cadence === 'weekly') {
    return 'bg-sky-500/14 text-sky-700 dark:text-sky-300'
  }
  if (cadence === 'capstone') {
    return 'bg-fuchsia-500/14 text-fuchsia-700 dark:text-fuchsia-300'
  }
  return 'bg-muted text-muted-foreground'
}

function ChallengeThumb({ imageUrl, title }: { imageUrl?: string | null; title: string }) {
  const [imageFailed, setImageFailed] = useState(false)

  if (!imageUrl || imageFailed) {
    return (
      <div className="flex h-16 w-16 shrink-0 items-center justify-center rounded-md border border-border bg-muted text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
        Défi
      </div>
    )
  }

  return (
    <img
      src={imageUrl}
      alt={title}
      className="h-16 w-16 shrink-0 rounded-md border border-border bg-muted object-cover"
      onError={() => setImageFailed(true)}
    />
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

  return (
    <div className="space-y-3">
      {sortedItems.map((item) => {
        const progressPercent = Math.max(0, Math.min(100, item.progress_percent ?? 0))
        const current = item.progress_current ?? 0
        const target = item.progress_target
        const cadence = deriveChallengeCadence(item.challenge_path)
        const cadenceLabel = challengeCadenceLabel(cadence)

        return (
          <div
            key={item.tracking_id ?? item.challenge_path}
            data-testid="home-challenge-item"
            className={`flex items-stretch gap-3 rounded-lg border p-3 ${challengeCadenceClasses(cadence)}`}
          >
            <ChallengeThumb imageUrl={item.image_url} title={item.title} />

            <div className="flex min-h-16 min-w-0 flex-1 flex-col justify-between gap-2">
              <div className="min-w-0">
                {cadenceLabel && (
                  <span
                    data-testid="home-challenge-kind"
                    className={`mb-1 inline-flex rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-[0.14em] ${challengeCadenceBadgeClasses(cadence)}`}
                  >
                    {cadenceLabel}
                  </span>
                )}
                <p data-testid="home-challenge-title" className="truncate text-[15px] font-semibold leading-tight text-foreground">
                  {item.title}
                </p>
                {item.description && (
                  <p data-testid="home-challenge-description" className="mt-1 line-clamp-2 text-xs italic text-muted-foreground">
                    {item.description}
                  </p>
                )}
              </div>

              <div className="mt-auto space-y-1">
                <div className="flex items-center justify-between text-[11px] text-muted-foreground">
                  <span>
                    {target != null ? `${current} / ${target}` : current}
                  </span>
                  <span>{Math.round(progressPercent)}%</span>
                </div>
                <div data-testid="home-challenge-progress-track" className="h-2 w-full overflow-hidden rounded-full bg-muted-foreground/25">
                  <div
                    data-testid="home-challenge-progress-fill"
                    className="h-full rounded-full bg-sky-500 transition-all duration-300"
                    style={{ width: `${progressPercent}%` }}
                  />
                </div>
              </div>
            </div>
          </div>
        )
      })}
    </div>
  )
}