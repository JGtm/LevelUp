/**
 * AchievementCard — carte compacte d'un achievement individuel.
 *
 * Format : icône + nom (1 ligne) + gamerscore + (description optionnelle ou
 * progression). Largeur fixe pour s'aligner dans une rangée horizontale.
 */
import type { AchievementEntry } from '@/lib/api/types'
import { Tooltip } from '@/components/ui/tooltip'
import {
  ACHIEVEMENTS_TEXT,
  formatUnlockedDate,
  pickLocalized,
  type AchievementsLocale,
} from './i18n'

const PLACEHOLDER_SVG =
  "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='80' height='80' viewBox='0 0 80 80'%3E%3Crect width='80' height='80' fill='%23374151'/%3E%3Ccircle cx='40' cy='40' r='14' fill='%236b7280' opacity='.5'/%3E%3C/svg%3E"

interface Props {
  achievement: AchievementEntry
  locale: AchievementsLocale
  /** Si true (par défaut), la carte a une largeur fixe pour s'intégrer dans
   *  un scroll horizontal. Si false, prend la largeur du parent. */
  fixedWidth?: boolean
}

export function AchievementCard({ achievement, locale, fixedWidth = true }: Props) {
  const t = ACHIEVEMENTS_TEXT[locale]
  const name = pickLocalized(achievement.name_en, achievement.name_fr, locale)

  // Description : utilise locked_desc si verrouillé et disponible, sinon description normale.
  let description: string
  if (achievement.unlocked) {
    description = pickLocalized(achievement.description_en, achievement.description_fr, locale)
  } else {
    const locked = pickLocalized(achievement.locked_desc_en, achievement.locked_desc_fr, locale)
    description = locked || pickLocalized(achievement.description_en, achievement.description_fr, locale)
  }

  const unlockedDate = formatUnlockedDate(achievement.unlocked_at, locale)
  const progress =
    achievement.target_progress !== undefined &&
    achievement.target_progress > 0 &&
    achievement.current_progress !== undefined
      ? { current: achievement.current_progress, target: achievement.target_progress }
      : null

  return (
    <div
      className={[
        'flex flex-col rounded border bg-card p-3 transition-opacity',
        fixedWidth ? 'w-56 flex-shrink-0 snap-start' : '',
        achievement.unlocked ? 'border-border' : 'border-muted opacity-60',
      ].join(' ')}
      title={name}
    >
      <div className="flex items-start gap-3">
        <img
          src={achievement.image_url || PLACEHOLDER_SVG}
          alt={name}
          loading="lazy"
          decoding="async"
          className="h-14 w-14 flex-shrink-0 rounded object-cover"
          onError={(e) => {
            ;(e.currentTarget as HTMLImageElement).src = PLACEHOLDER_SVG
          }}
        />
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex items-baseline justify-between gap-2">
            <p className="truncate text-sm font-semibold text-foreground">{name}</p>
            <span className="flex-shrink-0 text-xs font-medium text-muted-foreground">
              {achievement.gamerscore} G
            </span>
          </div>
          {description && (
            <Tooltip content={description} className="w-full">
              <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
                {description}
              </p>
            </Tooltip>
          )}
        </div>
      </div>

      {progress && (
        <div className="mt-2 flex items-center gap-2">
          <div className="h-1 flex-1 overflow-hidden rounded-full bg-muted">
            <div
              className="h-full bg-primary"
              style={{ width: `${Math.min(100, (progress.current / progress.target) * 100)}%` }}
            />
          </div>
          <span className="text-[10px] tabular-nums text-muted-foreground">
            {t.progress(progress.current, progress.target)}
          </span>
        </div>
      )}

      {unlockedDate && (
        <p className="mt-2 text-[10px] text-primary">{t.unlockedAt(unlockedDate)}</p>
      )}
    </div>
  )
}
