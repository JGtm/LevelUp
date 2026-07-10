/**
 * CombatYieldBar — barre composite rendement combat (Sprint 56).
 *
 * Deux segments poussant depuis le centre :
 *  - Gauche (vert)  : offensive_conversion  — normalisé par le repère 0.90
 *  - Droite (bleu)  : defensive_resistance  — normalisé DEPUIS 1.0 (baseline)
 *                     sur la plage (repère - 1.0) = 0.65. DR=1.0 → 0px, DR=1.65 → pleine barre.
 *
 * Largeur adaptative : `widthPx` (total) pilote la largeur ; par défaut 304px.
 * La géométrie (segments + badges) se recalcule linéairement sur la demi-largeur.
 * Clip à 1.5× la plage de référence. Badge débordement : valeur réelle affichée
 * hors-barre quand clippée ou ∞. Tooltip : dégâts bruts (jamais le ratio interne).
 */
import { useState } from 'react'
import { tokenCssVar } from '@/lib/accessibility'
import { useOffensiveConversionP80, useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import type { ManifestLocale } from '@/lib/i18n/format'

/** Libellé universel (FR=EN) quand la Résistance n'est pas calculable faute de
 *  damage_taken (ex. Halo 5). Aligné sur la convention `notAvailable: 'N/A'` du
 *  module compare ; glyphe non traduit, comme `—`/`∞` dans formatDefensiveResistance. */
const DR_NA_LABEL = 'N/A'

/** Repères barre (frontière élite). OC = title-aware (useOffensiveConversionP80 :
 *  0.90 Infinite / 1.264 Halo 5) ; DR = miroir const Go (h5 sans damage_taken → DR N/A). */
const DR_P80 = 1.65
const DR_BASELINE = 1.0
const DR_RANGE = DR_P80 - DR_BASELINE // 0.65 — plage utile au-dessus du baseline
const CLIP_FACTOR = 1.5
const DEFAULT_PER_SIDE_PX = 150
/** Largeur structurelle entre les deux demi-barres (séparateur + 2 gaps). */
const CENTER_GAP_PX = 5

export interface CombatYieldBarProps {
  /** offensive_conversion = 225 × (kills + assists/3) / damage_dealt */
  offensiveConversion?: number | null
  /** defensive_resistance = damage_taken / (225 × deaths) */
  defensiveResistance?: number | null
  /** Dégâts moyens par frag-équivalent (kills + assists/3) — pour tooltip */
  damagePerKill?: number | null
  /** Dégâts moyens par mort (pour tooltip) */
  damagePerDeath?: number | null
  /** Largeur totale en px (défaut 304). La demi-barre = (widthPx − gap) / 2. */
  widthPx?: number
}

function ocBarWidth(value: number | null | undefined, perSide: number, ocP80: number): number {
  if (value == null || value <= 0) return 0
  const clipped = Math.min(value, ocP80 * CLIP_FACTOR)
  return Math.round((clipped / ocP80 / CLIP_FACTOR) * perSide)
}

function drBarWidth(value: number | null | undefined, perSide: number): number {
  if (value == null) return 0
  if (value < 0) return perSide // sentinel ∞ (deaths == 0)
  if (value <= DR_BASELINE) return 0
  const excess = Math.min(value - DR_BASELINE, DR_RANGE * CLIP_FACTOR)
  return Math.round((excess / DR_RANGE / CLIP_FACTOR) * perSide)
}

interface TooltipProps {
  offensiveConversion: number | null | undefined
  defensiveResistance: number | null | undefined
  damagePerKill: number | null | undefined
  damagePerDeath: number | null | undefined
  /** false (titre sans damage_taken) → Résistance affichée N/A, sans dégâts/mort. */
  providesDamageTaken: boolean
  /** GH3-3 : locale de l'UI (labels Rendement/Résistance + dégâts/frag·mort). */
  locale: ManifestLocale
}

function Tooltip({ offensiveConversion, defensiveResistance, damagePerKill, damagePerDeath, providesDamageTaken, locale }: TooltipProps) {
  return (
    <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50 w-48 rounded-md bg-popover border border-border px-3 py-2 text-xs shadow-lg pointer-events-none">
      <div className="flex justify-between gap-2 mb-1">
        <span className="font-semibold" style={{ color: tokenCssVar('divergent-pos') }}>{formatMessage(commonManifest, 'common.match_card.offensive_yield', locale)}</span>
        <span className="text-muted-foreground">{offensiveConversion != null ? `${Math.round(offensiveConversion * 100)}%` : '—'}</span>
      </div>
      {damagePerKill != null && (
        <div className="text-muted-foreground mb-1">{formatMessage(commonManifest, 'common.match_card.dmg_per_kill', locale, { n: Math.round(damagePerKill) })}</div>
      )}
      <div className="flex justify-between gap-2 mb-1">
        <span className="font-semibold" style={{ color: tokenCssVar('divergent-neutral') }}>{formatMessage(commonManifest, 'common.match_card.defensive_resistance', locale)}</span>
        <span className="text-muted-foreground">
          {!providesDamageTaken ? DR_NA_LABEL : defensiveResistance == null ? '—' : defensiveResistance < 0 ? '∞' : `${Math.round((defensiveResistance - 1) * 100)}%`}
        </span>
      </div>
      {providesDamageTaken && damagePerDeath != null && (
        <div className="text-muted-foreground">{formatMessage(commonManifest, 'common.match_card.dmg_per_death', locale, { n: Math.round(damagePerDeath) })}</div>
      )}
      {/* triangle pointer */}
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
  const locale = useAppShellStore((s) => s.locale)
  const ocP80 = useOffensiveConversionP80() // 0.90 Infinite / 1.264 h5 (titre courant)
  // false (Halo 5 : API sans damage_taken) → la Résistance n'est pas calculable.
  // On neutralise tout le visuel DR (barre/badge à 0, tooltip N/A) au lieu
  // d'afficher une barre vide trompeuse à 0. Défaut true → Infinite inchangé.
  const providesDamageTaken = useProvidesDamageTaken()

  const perSide = widthPx != null
    ? Math.max(1, Math.floor((widthPx - CENTER_GAP_PX) / 2))
    : DEFAULT_PER_SIDE_PX

  const ocWidth = ocBarWidth(offensiveConversion, perSide, ocP80)
  const drWidth = providesDamageTaken ? drBarWidth(defensiveResistance, perSide) : 0

  const hasData =
    (offensiveConversion != null && offensiveConversion > 0) ||
    (providesDamageTaken &&
      defensiveResistance != null &&
      (defensiveResistance > DR_BASELINE || defensiveResistance < 0))

  const ocClipped = offensiveConversion != null && offensiveConversion > ocP80 * CLIP_FACTOR
  const drIsInfinite = providesDamageTaken && defensiveResistance != null && defensiveResistance < 0
  const drClipped = !drIsInfinite && providesDamageTaken && drWidth === perSide
  const showDrBadge = drIsInfinite || drClipped

  return (
    <div
      className="relative flex items-center justify-center gap-0.5"
      style={{ width: perSide * 2 + CENTER_GAP_PX, height: 10 }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Badge débordement OC — affiché à gauche hors-barre */}
      {hasData && ocClipped && (
        <div
          className="absolute text-[9px] font-semibold leading-none"
          style={{
            right: perSide + 6,
            color: tokenCssVar('divergent-pos'),
            whiteSpace: 'nowrap',
          }}
        >
          {Math.round(offensiveConversion! * 100)}%
        </div>
      )}

      {/* Barre offensive (gauche, pousse vers la droite) */}
      <div className="flex justify-end" style={{ width: perSide, position: 'relative' }}>
        {hasData && (
          <div
            className="h-2 rounded-l-full transition-all duration-300"
            style={{
              width: ocWidth,
              backgroundColor: ocWidth > 0 ? tokenCssVar('divergent-pos') : 'transparent',
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
              backgroundColor: drWidth > 0 ? tokenCssVar('divergent-neutral') : 'transparent',
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
          {drIsInfinite ? '∞' : `${Math.round((defensiveResistance! - 1) * 100)}%`}
        </div>
      )}

      {hovered && hasData && (
        <Tooltip
          offensiveConversion={offensiveConversion}
          defensiveResistance={defensiveResistance}
          damagePerKill={damagePerKill}
          damagePerDeath={damagePerDeath}
          providesDamageTaken={providesDamageTaken}
          locale={locale}
        />
      )}
    </div>
  )
}
