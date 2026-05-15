'use client'

import { useEffect, useMemo, useRef, useState } from 'react'

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

type SlotPos = { x: string; scale: number; opacity: number }
const POSITIONS: Record<number, SlotPos> = {
  [-2]: { x: '-115%', scale: 0.55, opacity: 0 },
  [-1]: { x: '-55%', scale: 0.7, opacity: 0.45 },
  [0]: { x: '0%', scale: 1, opacity: 1 },
  [1]: { x: '55%', scale: 0.7, opacity: 0.45 },
  [2]: { x: '115%', scale: 0.55, opacity: 0 },
}
const ANIM_MS = 500
const ANIM_EASE = 'cubic-bezier(0.32, 0.72, 0, 1)'
const WINDOW_RADIUS = 2

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

function clampIndex(idx: number, len: number) {
  if (len <= 0) return 0
  return Math.min(Math.max(idx, 0), len - 1)
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

  useEffect(() => {
    if (animatingRef.current) return
    setCurrentIndex(clampIndex(startIndex, rewards.length))
    // eslint-disable-next-line react-hooks/exhaustive-deps -- on resync uniquement quand le parent change explicitement startIndex
  }, [startIndex])

  const clampedIdx = clampIndex(currentIndex, rewards.length)
  const canPrev = clampedIdx > 0
  const canNext = clampedIdx < rewards.length - 1
  const current = rewards[clampedIdx] ?? null

  function navigate(dir: 'next' | 'prev') {
    if (animatingRef.current) return
    if (dir === 'next' && !canNext) return
    if (dir === 'prev' && !canPrev) return
    animatingRef.current = true
    setCurrentIndex(clampedIdx + (dir === 'next' ? 1 : -1))
    window.setTimeout(() => {
      animatingRef.current = false
    }, ANIM_MS)
  }

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowLeft') navigate('prev')
      if (event.key === 'ArrowRight') navigate('next')
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- navigate() lit clampedIdx+rewards.length via closure, déjà couverts dans les deps
  }, [clampedIdx, rewards.length, onClose])

  const slots = useMemo(() => {
    const result: { reward: RewardLightboxData; relPos: number; absIdx: number }[] = []
    for (let off = -WINDOW_RADIUS; off <= WINDOW_RADIUS; off++) {
      const idx = clampedIdx + off
      if (idx >= 0 && idx < rewards.length) {
        result.push({ reward: rewards[idx], relPos: off, absIdx: idx })
      }
    }
    return result
  }, [rewards, clampedIdx])

  if (!current) {
    return null
  }

  const rankLabel = current.rank == null ? null : `#${current.rank}`
  const rarityTier = normalizeRarity(current.quality)
  const rarityStyles = rarityStyle(rarityTier)
  const typeLabel = itemTypeLabel(current.itemType)
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
              <p className="text-[11px] uppercase tracking-[0.24em] text-muted-foreground">
                {subtitle}
              </p>
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

        <div className="relative flex-1 overflow-visible">
          <div className="relative mx-auto aspect-[4/3] w-full">
            {slots.map(({ reward, relPos, absIdx }) => {
              const pos = POSITIONS[relPos]
              const isCenter = relPos === 0
              const slotRarity = normalizeRarity(reward.quality)
              const slotStyles = rarityStyle(slotRarity)
              return (
                <div
                  key={absIdx}
                  className={[
                    'absolute inset-0 flex items-center justify-center overflow-hidden rounded-xl',
                    slotStyles?.bg ?? 'bg-muted/60',
                  ].join(' ')}
                  style={{
                    transform: `translateX(${pos.x}) scale(${pos.scale})`,
                    opacity: pos.opacity,
                    transition: `transform ${ANIM_MS}ms ${ANIM_EASE}, opacity ${ANIM_MS}ms ${ANIM_EASE}`,
                    pointerEvents: pos.opacity === 0 ? 'none' : 'auto',
                    zIndex: 10 - Math.abs(relPos),
                    willChange: 'transform, opacity',
                    cursor: isCenter ? 'default' : 'pointer',
                  }}
                  onClick={
                    isCenter
                      ? undefined
                      : (event) => {
                          event.stopPropagation()
                          if (relPos < 0) navigate('prev')
                          else navigate('next')
                        }
                  }
                  aria-hidden={isCenter ? undefined : true}
                >
                  {reward.imageUrl ? (
                    <img
                      src={reward.imageUrl}
                      alt={isCenter ? reward.title : ''}
                      className="max-h-full max-w-full object-contain p-4"
                      data-testid={isCenter ? 'battle-pass-reward-lightbox-image' : undefined}
                    />
                  ) : (
                    <div className="flex h-48 w-48 items-center justify-center rounded-xl bg-muted/40 text-center">
                      <p className="text-5xl font-semibold text-foreground">{reward.rank ?? '?'}</p>
                    </div>
                  )}
                </div>
              )
            })}
          </div>

          {canPrev && (
            <button
              type="button"
              onClick={(event) => {
                event.stopPropagation()
                navigate('prev')
              }}
              aria-label="Récompense précédente"
              className="absolute left-2 top-1/2 z-20 flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-background/80 text-foreground/80 transition-colors hover:bg-background hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m15 18-6-6 6-6" /></svg>
            </button>
          )}
          {canNext && (
            <button
              type="button"
              onClick={(event) => {
                event.stopPropagation()
                navigate('next')
              }}
              aria-label="Récompense suivante"
              className="absolute right-2 top-1/2 z-20 flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-background/80 text-foreground/80 transition-colors hover:bg-background hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6" /></svg>
            </button>
          )}
        </div>

        {(current.badges?.length || current.description || rarityTier) && (
          <div className="space-y-3 border-t border-border bg-muted/40 px-5 py-4">
            {(current.badges?.length || rarityTier) && (
              <div className="flex flex-wrap gap-2">
                {rarityTier && rarityStyles && (
                  <span
                    className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${rarityStyles.badge}`}
                    data-testid="battle-pass-reward-lightbox-rarity"
                  >
                    {rarityLabel(rarityTier)}
                  </span>
                )}
                {current.badges?.map((badge, i) => (
                  <Badge key={`${badge.label}-${i}`} variant={badgeVariantFor(badge.tone)}>
                    {badge.label}
                  </Badge>
                ))}
              </div>
            )}
            {current.description && (
              <p className="text-sm leading-6 text-foreground">{current.description}</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
