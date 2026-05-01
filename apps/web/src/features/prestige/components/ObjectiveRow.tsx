/**
 * ObjectiveRow — ligne d'objectif Prestige, format aligné sur HomeChallengesList
 * (défis officiels) : badge à gauche, titre + description + progression à droite.
 *
 * Le badge provient de `/static/prestige-assets/Objectives-badges/` selon la
 * cadence (fréquence) et le tier (difficulté). Le fond est teinté par la couleur
 * du tier pour souligner la difficulté.
 */
import { useState } from 'react'
import type { Cadence, Challenge, Tier } from '@/lib/prestige'
import { TIER_COLORS, TIER_LABELS_FR } from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'

interface ObjectiveRowProps {
  challenge: Challenge
  /** Valeur courante mesurée — override le `challenge.current_value`. */
  currentValue?: number
  onClick?: () => void
}

/** Mappe (cadence, tier) → URL du badge dans /static/prestige-assets/Objectives-badges. */
function objectiveBadgeUrl(cadence: Cadence, tier: Tier): string {
  const base = '/static/prestige-assets/Objectives-badges'
  // weekly + mythic n'existe pas → capstone-mythic est l'équivalent.
  if (cadence === 'weekly' && tier === 'mythic') {
    return `${base}/objective-capstone-mythic.png`
  }
  if (cadence === 'daily' || cadence === 'weekly') {
    return `${base}/objective-${cadence}-${tier}.png`
  }
  // monthly / free : fallback sur weekly (mythic → capstone, sinon legendary).
  if (tier === 'mythic') {
    return `${base}/objective-capstone-mythic.png`
  }
  return `${base}/objective-weekly-${tier}.png`
}

function ObjectiveBadge({ cadence, tier, alt }: { cadence: Cadence; tier: Tier; alt: string }) {
  const [failed, setFailed] = useState(false)
  const url = objectiveBadgeUrl(cadence, tier)

  if (failed) {
    return (
      <div
        className="flex h-12 w-12 shrink-0 items-center justify-center rounded-md text-[9px] font-semibold uppercase tracking-[0.14em]"
        style={{ color: TIER_COLORS[tier] }}
      >
        {tier.slice(0, 4)}
      </div>
    )
  }

  return (
    <div className="h-12 w-12 shrink-0 overflow-hidden rounded-md">
      <img
        src={url}
        alt={alt}
        className="h-full w-full object-contain"
        onError={() => setFailed(true)}
      />
    </div>
  )
}

export function ObjectiveRow({ challenge, currentValue, onClick }: ObjectiveRowProps) {
  const cv = currentValue ?? challenge.current_value ?? 0
  const tier: Tier = challenge.tier ?? 'normal'
  const cadence: Cadence = challenge.cadence
  const tierColor = TIER_COLORS[tier]
  const tierLabelFromTOML = useAssetLabel('challenge_tier', tier)
  const tierLabel = tierLabelFromTOML !== tier ? tierLabelFromTOML : TIER_LABELS_FR[tier]

  const target = challenge.target
  const progressPercent = target > 0
    ? Math.max(0, Math.min(100, Math.round((cv / target) * 100)))
    : 0
  const isComplete = challenge.status === 'completed'
  const title = challenge.label || challenge.metric

  // Fond teinté par la couleur du tier (faible opacité — ~8 %).
  // Bord coloré par tier pour l'accent vertical (cohérent avec ChallengeCard).
  const rowStyle: React.CSSProperties = {
    backgroundColor: `${tierColor}14`, // ~8% opacity en hex (0x14)
    borderColor: `${tierColor}40`, // ~25% opacity
  }

  return (
    <button
      type="button"
      onClick={onClick}
      data-testid="objective-row"
      className="flex w-full items-center gap-2.5 rounded-md border p-2 text-left transition-colors hover:brightness-110"
      style={rowStyle}
    >
      <ObjectiveBadge cadence={cadence} tier={tier} alt={title} />

      <div className="flex min-w-0 flex-1 flex-col justify-center gap-1">
        <div className="flex items-center justify-between gap-2">
          <p
            data-testid="objective-row-title"
            className="min-w-0 truncate text-[13px] font-semibold leading-tight text-foreground"
          >
            {title}
          </p>
          <div className="flex shrink-0 items-center gap-1.5">
            {challenge.is_squad && (
              <span
                data-testid="objective-row-squad-badge"
                className="rounded border border-muted-foreground/30 px-1 py-0.5 text-[9px] font-medium uppercase tracking-wide text-muted-foreground"
              >
                Escouade
              </span>
            )}
            <span
              className="shrink-0 rounded-full px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wider"
              style={{ backgroundColor: `${tierColor}30`, color: tierColor }}
            >
              {tierLabel}
            </span>
          </div>
        </div>

        <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-1.5 text-[10px] text-muted-foreground">
          <span className="shrink-0 whitespace-nowrap tabular-nums">
            {cv.toFixed(2)}/{target.toFixed(2)}
          </span>
          <div className="min-w-0 overflow-hidden rounded-full bg-muted-foreground/25">
            <div className="h-1.5 w-full">
              <div
                className="h-full rounded-full transition-all duration-300"
                style={{
                  width: `${progressPercent}%`,
                  backgroundColor: tierColor,
                  opacity: isComplete ? 1 : 0.85,
                }}
              />
            </div>
          </div>
          <span className="shrink-0 whitespace-nowrap text-right tabular-nums">
            {progressPercent}%
          </span>
        </div>
      </div>
    </button>
  )
}
