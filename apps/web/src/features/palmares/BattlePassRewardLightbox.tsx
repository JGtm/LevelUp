'use client'

import { useEffect, useRef, useState } from 'react'

import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

import { type SeasonPassBadgeRole } from './battlePassBadgeStyle'
import { itemTypeLabel, normalizeRarity, rarityLabel, rarityStyle } from './rarity'
import { SeasonPassBadge } from './SeasonPassBadge'

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

const ANIM_MS = 280

function toneRole(tone: RewardLightboxBadgeTone | undefined): SeasonPassBadgeRole {
  switch (tone) {
    case 'current':  return 'active'
    case 'obtained': return 'completed'
    case 'premium':  return 'premium'
    case 'free':     return 'free'
    case 'upcoming': return 'neutral'
    default:         return 'neutral'
  }
}

function clampIndex(idx: number, len: number) {
  if (len <= 0) return 0
  return Math.min(Math.max(idx, 0), len - 1)
}

/** Vignette latérale (prev ou next) affichée en dehors du bloc principal. */
function SideThumbnail({
  reward,
  dir,
  label,
  onClick,
}: {
  reward: RewardLightboxData
  dir: 'prev' | 'next'
  label: string
  onClick: () => void
}) {
  const rarityTier = normalizeRarity(reward.quality)
  const slotStyles = rarityStyle(rarityTier)
  return (
    <button
      type="button"
      onClick={(e) => { e.stopPropagation(); onClick() }}
      aria-label={label}
      className={[
        'flex h-52 w-36 shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-xl border border-border/50 opacity-45 transition-all duration-200 hover:opacity-75 hover:scale-105 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        slotStyles?.bg ?? 'bg-muted/60',
        dir === 'prev' ? 'origin-right' : 'origin-left',
      ].join(' ')}
    >
      {reward.imageUrl ? (
        <img
          src={reward.imageUrl}
          alt=""
          aria-hidden="true"
          className="max-h-full max-w-full object-contain p-2"
        />
      ) : (
        <span className="text-lg font-semibold text-foreground/60">{reward.rank ?? '?'}</span>
      )}
    </button>
  )
}

export function BattlePassRewardLightbox({
  rewards,
  startIndex,
  onClose,
}: {
  rewards: RewardLightboxData[]
  startIndex: number
  onClose: () => void
}) {
  const [currentIndex, setCurrentIndex] = useState(() => clampIndex(startIndex, rewards.length))
  const animatingRef = useRef(false)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  useEffect(() => {
    if (animatingRef.current) return
    setCurrentIndex(clampIndex(startIndex, rewards.length))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [startIndex])

  const clampedIdx = clampIndex(currentIndex, rewards.length)
  const canPrev = clampedIdx > 0
  const canNext = clampedIdx < rewards.length - 1
  const current = rewards[clampedIdx] ?? null
  const prevReward = canPrev ? rewards[clampedIdx - 1] : null
  const nextReward = canNext ? rewards[clampedIdx + 1] : null

  function navigate(dir: 'next' | 'prev') {
    if (animatingRef.current) return
    if (dir === 'next' && !canNext) return
    if (dir === 'prev' && !canPrev) return
    animatingRef.current = true
    setCurrentIndex(clampedIdx + (dir === 'next' ? 1 : -1))
    window.setTimeout(() => { animatingRef.current = false }, ANIM_MS)
  }

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowLeft') navigate('prev')
      if (event.key === 'ArrowRight') navigate('next')
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clampedIdx, rewards.length, onClose])

  if (!current) return null

  const rankLabel = current.rank == null ? null : `#${current.rank}`
  const rarityTier = normalizeRarity(current.quality)
  const rarityStyles = rarityStyle(rarityTier)
  const typeLabel = itemTypeLabel(current.itemType, locale)
  const subtitle = [rankLabel, typeLabel].filter(Boolean).join(' · ')

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-background/85 p-4 backdrop-blur-sm"
      role="dialog"
      aria-modal="true"
      aria-label={current.title}
      onClick={onClose}
      data-testid="battle-pass-reward-lightbox"
    >
      {/* Layout : [prev] [bloc central] [next] */}
      <div className="relative flex w-full max-w-2xl items-center gap-3">

        {/* Vignette précédente — EN DEHORS du bloc */}
        <div className="flex w-36 shrink-0 justify-center">
          {prevReward && (
            <SideThumbnail
              reward={prevReward}
              dir="prev"
              label={t('common.battlepass.prev_aria')}
              onClick={() => navigate('prev')}
            />
          )}
        </div>

        {/* Bloc central */}
        <div
          className={[
            'relative flex min-w-0 flex-1 flex-col overflow-hidden rounded-2xl border border-border bg-card text-foreground shadow-2xl',
            rarityStyles?.glow ?? '',
          ].filter(Boolean).join(' ')}
          onClick={(e) => e.stopPropagation()}
        >
          {/* En-tête */}
          <div className="flex items-start justify-between gap-4 border-b border-border bg-muted/60 px-5 py-3">
            <div className="min-w-0 space-y-1">
              {subtitle && (
                <p className="text-3xs uppercase tracking-label-xl text-muted-foreground">{subtitle}</p>
              )}
              <h2 className="truncate text-lg font-semibold sm:text-xl">{current.title}</h2>
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

          {/* Image */}
          <div
            className={[
              'flex aspect-square w-full items-center justify-center p-6',
              rarityStyles?.bg ?? 'bg-muted/30',
            ].join(' ')}
          >
            {current.imageUrl ? (
              <img
                src={current.imageUrl}
                alt={current.title}
                className="max-h-full max-w-full object-contain"
                data-testid="battle-pass-reward-lightbox-image"
              />
            ) : (
              <div className="flex h-32 w-32 items-center justify-center rounded-xl bg-muted/40">
                <p className="text-5xl font-semibold text-foreground">{current.rank ?? '?'}</p>
              </div>
            )}
          </div>

          {/* Pied : badges + rareté */}
          {(current.badges?.length || rarityTier) && (
            <div className="space-y-3 border-t border-border bg-muted/40 px-5 py-4">
              <div className="flex flex-wrap gap-2">
                {rarityTier && rarityStyles && (
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${rarityStyles.badge}`}
                    data-testid="battle-pass-reward-lightbox-rarity"
                  >
                    {rarityLabel(rarityTier, locale)}
                  </span>
                )}
                {current.badges?.map((badge, i) => (
                  <SeasonPassBadge key={`${badge.label}-${i}`} role={toneRole(badge.tone)} label={badge.label} />
                ))}
              </div>
            </div>
          )}

          {/* Flèches sur le bord du bloc (complément au clic sur les vignettes) */}
          {canPrev && (
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); navigate('prev') }}
              aria-label={t('common.battlepass.prev_aria')}
              className="absolute left-2 top-1/2 z-20 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-background/80 text-foreground/80 opacity-0 transition-opacity hover:bg-background hover:text-foreground group-hover:opacity-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg>
            </button>
          )}
          {canNext && (
            <button
              type="button"
              onClick={(e) => { e.stopPropagation(); navigate('next') }}
              aria-label={t('common.battlepass.next_aria')}
              className="absolute right-2 top-1/2 z-20 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full bg-background/80 text-foreground/80 opacity-0 transition-opacity hover:bg-background hover:text-foreground group-hover:opacity-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg>
            </button>
          )}
        </div>

        {/* Vignette suivante — EN DEHORS du bloc */}
        <div className="flex w-36 shrink-0 justify-center">
          {nextReward && (
            <SideThumbnail
              reward={nextReward}
              dir="next"
              label={t('common.battlepass.next_aria')}
              onClick={() => navigate('next')}
            />
          )}
        </div>

      </div>
    </div>
  )
}
