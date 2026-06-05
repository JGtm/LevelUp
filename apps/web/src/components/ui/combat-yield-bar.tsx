/**
 * CombatYieldBar — barre composite rendement combat (Sprint 56).
 *
 * Deux segments poussant depuis le centre :
 *  - Gauche (vert)  : offensive_conversion  — normalisé par p80 = 0.83
 *  - Droite (bleu)  : defensive_resistance  — normalisé DEPUIS 1.0 (baseline)
 *                     sur la plage (p80 - 1.0) = 0.59. DR=1.0 → 0px, DR=1.59 → pleine barre.
 *
 * Largeur adaptative : `widthPx` (total) pilote la largeur ; par défaut 304px.
 * La géométrie (segments + badges) se recalcule linéairement sur la demi-largeur.
 * Clip à 1.5× la plage de référence. Badge débordement : valeur réelle affichée
 * hors-barre quand clippée ou ∞. Tooltip : dégâts bruts (jamais le ratio interne).
 */
import { useState } from 'react'
import { tokenCssVar } from '@/lib/accessibility'

/** p80, baseline et constantes — miroir des constantes Go combat_yield.go */
const OC_P80 = 0.83
const DR_P80 = 1.59
const DR_BASELINE = 1.0
const DR_RANGE = DR_P80 - DR_BASELINE // 0.59 — plage utile au-dessus du baseline
const CLIP_FACTOR = 1.5
const DEFAULT_PER_SIDE_PX = 150
/** Largeur structurelle entre les deux demi-barres (séparateur + 2 gaps). */
const CENTER_GAP_PX = 5

export interface CombatYieldBarProps {
  /** offensive_conversion = 225 × (kills + assists/3) / damage_dealt */
  offensiveConversion?: number | null
  /** defensive_resistance = damage_taken / (225 × deaths) */
  defensiveResistance?: number | null
  /** Dégâts moyens par kill (pour tooltip) */
  damagePerKill?: number | null
  /** Dégâts moyens par mort (pour tooltip) */
  damagePerDeath?: number | null
  /** Largeur totale en px (défaut 304). La demi-barre = (widthPx − gap) / 2. */
  widthPx?: number
}

function ocBarWidth(value: number | null | undefined, perSide: number): number {
  if (value == null || value <= 0) return 0
  const clipped = Math.min(value, OC_P80 * CLIP_FACTOR)
  return Math.round((clipped / OC_P80 / CLIP_FACTOR) * perSide)
}

function drBarWidth(value: number | null | undefined, perSide: number): number {
  if (value == null) return 0
  if (value < 0) return perSide // sentinel ∞ (deaths == 0)
  if (value <= DR_BASELINE) return 0
  const excess = Math.min(value - DR_BASELINE, DR_RANGE * CLIP_FACTOR)
  return Math.round((excess / DR_RANGE / CLIP_FACTOR) * perSide)
}

interface TooltipProps {
  damagePerKill: number | null | undefined
  damagePerDeath: number | null | undefined
}

function Tooltip({ damagePerKill, damagePerDeath }: TooltipProps) {
  return (
    <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50 w-52 rounded-md bg-popover border border-border px-3 py-2 text-xs shadow-lg pointer-events-none">
      <div className="flex justify-between gap-2 mb-1">
        <span className="font-semibold" style={{ color: tokenCssVar('divergent-pos') }}>Rendement</span>
        <span className="text-muted-foreground">
          {damagePerKill != null ? `${Math.round(damagePerKill)} dégâts/frag` : '—'}
        </span>
      </div>
      <div className="flex justify-between gap-2 mb-1">
        <span className="font-semibold" style={{ color: tokenCssVar('divergent-neutral') }}>Résistance</span>
        <span className="text-muted-foreground">
          {damagePerDeath != null ? `${Math.round(damagePerDeath)} dégâts/mort` : '—'}
        </span>
      </div>
      <div className="text-[10px] text-muted-foreground/60 mt-1">réf. 225 = 1 vie</div>
      <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-popover" />
    </div>
  )
}

export function CombatYieldBar({
  offensiveConversion,
  defensiveResistance,
  damagePerKill,
  damagePerDeath,
  widthPx,
}: CombatYieldBarProps) {
  const [hovered, setHovered] = useState(false)

  const perSide = widthPx != null
    ? Math.max(1, Math.floor((widthPx - CENTER_GAP_PX) / 2))
    : DEFAULT_PER_SIDE_PX

  const ocWidth = ocBarWidth(offensiveConversion, perSide)
  const drWidth = drBarWidth(defensiveResistance, perSide)

  const hasData =
    (offensiveConversion != null && offensiveConversion > 0) ||
    (defensiveResistance != null && (defensiveResistance > DR_BASELINE || defensiveResistance < 0))

  const ocClipped = offensiveConversion != null && offensiveConversion > OC_P80 * CLIP_FACTOR
  const drIsInfinite = defensiveResistance != null && defensiveResistance < 0
  const drClipped = !drIsInfinite && drWidth === perSide
  const showDrBadge = drIsInfinite || drClipped

  return (
    <div
      className="relative flex items-center justify-center gap-0.5"
      style={{ width: perSide * 2 + CENTER_GAP_PX, height: 10 }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Badge débordement OC — affiché à gauche hors-barre */}
      {hasData && ocClipped && damagePerKill != null && (
        <div
          className="absolute text-[9px] font-semibold leading-none"
          style={{
            right: perSide + 6,
            color: tokenCssVar('divergent-pos'),
            whiteSpace: 'nowrap',
          }}
        >
          {Math.round(damagePerKill)}
        </div>
      )}

      {/* Barre offensive (gauche, pousse vers la droite) */}
      <div className="flex justify-end" style={{ width: perSide, position: 'relative' }}>
        {hasData && (
          <div
            className="h-2 rounded-l-full transition-all duration-300"
            style={{
              width: ocWidth,
              // Gradient : extrémité gauche (loin du centre = bonne efficacité) → solide ;
              // extrémité droite (près du centre = baseline) → transparent.
              background: ocWidth > 0
                ? `linear-gradient(to right, ${tokenCssVar('divergent-pos')}, color-mix(in srgb, ${tokenCssVar('divergent-pos')} 10%, transparent))`
                : 'transparent',
              opacity: ocWidth > 0 ? 1 : 0,
            }}
          />
        )}
      </div>

      {/* Séparateur central (point zéro entre les deux barres divergentes) */}
      <div className="w-px h-3 bg-border flex-shrink-0" />

      {/* Barre défensive (droite) */}
      <div className="flex justify-start" style={{ width: perSide, position: 'relative' }}>
        {hasData && (
          <div
            className="h-2 rounded-r-full transition-all duration-300"
            style={{
              width: drWidth,
              // Gradient : extrémité droite (loin du centre = bonne résistance) → solide ;
              // extrémité gauche (près du centre = baseline) → transparent.
              background: drWidth > 0
                ? `linear-gradient(to left, ${tokenCssVar('divergent-neutral')}, color-mix(in srgb, ${tokenCssVar('divergent-neutral')} 10%, transparent))`
                : 'transparent',
              opacity: drWidth > 0 ? 1 : 0,
            }}
          />
        )}
      </div>

      {/* Badge débordement DR — affiché à droite hors-barre */}
      {hasData && showDrBadge && (
        <div
          className="absolute text-[9px] font-semibold leading-none"
          style={{
            left: perSide + 6,
            color: tokenCssVar('divergent-neutral'),
            whiteSpace: 'nowrap',
          }}
        >
          {drIsInfinite ? '∞' : damagePerDeath != null ? String(Math.round(damagePerDeath)) : ''}
        </div>
      )}

      {hovered && hasData && (
        <Tooltip
          damagePerKill={damagePerKill}
          damagePerDeath={damagePerDeath}
        />
      )}
    </div>
  )
}
