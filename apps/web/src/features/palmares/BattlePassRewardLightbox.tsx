import { useEffect } from 'react'

import { Badge } from '@/components/ui/badge'

import { itemTypeLabel, normalizeRarity, rarityLabel, rarityStyle } from './rarity'

export type RewardLightboxBadgeTone = 'free' | 'premium' | 'obtained' | 'current' | 'upcoming' | 'neutral'

export interface RewardLightboxBadge {
  label: string
  tone?: RewardLightboxBadgeTone
}

export interface RewardLightboxData {
  title: string
  rank?: number | null
  imageUrl?: string | null
  description?: string | null
  /** Rareté brute (Common / Rare / Epic / Legendary / Mythic). */
  quality?: string | null
  /** Catégorie brute (ArmorCoating, WeaponCharm…). */
  itemType?: string | null
  badges?: RewardLightboxBadge[]
}

function badgeVariantFor(tone: RewardLightboxBadgeTone | undefined) {
  switch (tone) {
    case 'current':
      return 'default' as const
    case 'obtained':
      return 'success' as const
    case 'premium':
      return 'outline' as const
    case 'free':
      return 'secondary' as const
    case 'upcoming':
      return 'outline' as const
    default:
      return 'secondary' as const
  }
}

export function BattlePassRewardLightbox({
  reward,
  onClose,
  onPrev,
  onNext,
}: {
  reward: RewardLightboxData | null
  onClose: () => void
  onPrev?: () => void
  onNext?: () => void
}) {
  useEffect(() => {
    if (!reward) return
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowLeft') onPrev?.()
      if (event.key === 'ArrowRight') onNext?.()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [reward, onClose, onPrev, onNext])

  if (!reward) {
    return null
  }

  const rankLabel = reward.rank == null ? null : `#${reward.rank}`
  const rarityTier = normalizeRarity(reward.quality)
  const rarityStyles = rarityStyle(rarityTier)
  const typeLabel = itemTypeLabel(reward.itemType)
  const subtitle = [rankLabel, typeLabel].filter(Boolean).join(' · ')

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/85 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label={reward.title}
      onClick={onClose}
      data-testid="battle-pass-reward-lightbox"
    >
      <div
        className={[
          'relative flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-border bg-card text-foreground shadow-2xl',
          rarityStyles?.glow ?? '',
        ].filter(Boolean).join(' ')}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4 border-b border-border bg-muted/60 px-5 py-3">
          <div className="min-w-0 space-y-1">
            {subtitle && (
              <p
                className="text-[11px] uppercase tracking-[0.24em] text-muted-foreground"
              >
                {subtitle}
              </p>
            )}
            <h2 className="truncate text-lg font-semibold sm:text-xl">{reward.title}</h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xl leading-none text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label="Fermer"
          >
            ×
          </button>
        </div>

        <div className="relative flex-1 overflow-hidden">
          <div
            className={[
              'flex h-full items-center justify-center overflow-auto p-6',
              rarityStyles?.bg ?? 'bg-muted/60',
            ].join(' ')}
          >
            {reward.imageUrl ? (
              <img
                src={reward.imageUrl}
                alt={reward.title}
                className="max-h-[60vh] max-w-full object-contain"
                data-testid="battle-pass-reward-lightbox-image"
              />
            ) : (
              <div className="flex h-64 w-64 items-center justify-center rounded-xl bg-muted/40 text-center">
                <p className="text-5xl font-semibold text-foreground">{reward.rank ?? '?'}</p>
              </div>
            )}
          </div>
          {onPrev && (
            <button
              type="button"
              onClick={onPrev}
              aria-label="Récompense précédente"
              className="absolute left-2 top-1/2 z-10 flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-background/80 text-foreground/80 transition-colors hover:bg-background hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg>
            </button>
          )}
          {onNext && (
            <button
              type="button"
              onClick={onNext}
              aria-label="Récompense suivante"
              className="absolute right-2 top-1/2 z-10 flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-background/80 text-foreground/80 transition-colors hover:bg-background hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg>
            </button>
          )}
        </div>

        {(reward.badges?.length || reward.description || rarityTier) && (
          <div className="space-y-3 border-t border-border bg-muted/40 px-5 py-4">
            {(reward.badges?.length || rarityTier) && (
              <div className="flex flex-wrap gap-2">
                {rarityTier && rarityStyles && (
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${rarityStyles.badge}`}
                    data-testid="battle-pass-reward-lightbox-rarity"
                  >
                    {rarityLabel(rarityTier)}
                  </span>
                )}
                {reward.badges?.map((badge, i) => (
                  <Badge key={`${badge.label}-${i}`} variant={badgeVariantFor(badge.tone)}>
                    {badge.label}
                  </Badge>
                ))}
              </div>
            )}
            {reward.description && (
              <p
                className="text-sm leading-6 text-foreground"
              >
                {reward.description}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
