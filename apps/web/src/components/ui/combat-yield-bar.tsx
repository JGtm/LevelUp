/**
 * CombatYieldBar — barre composite rendement combat (Sprint 56).
 *
 * Deux segments poussant depuis le centre :
 *  - Gauche (vert)  : offensive_conversion  — normalisé par p80 = 0.83
 *  - Droite (bleu)  : defensive_resistance  — normalisé par p80 = 1.59
 *
 * Largeur max par côté : 120px. Clip à 1.5× p80.
 * Tooltip : dégâts bruts (dmg/kill et dmg/death), jamais le ratio interne.
 */
import { useState } from 'react'

/** p80 et clip factor — miroir des constantes Go combat_yield.go */
const OC_P80 = 0.83
const DR_P80 = 1.59
const CLIP_FACTOR = 1.5
const BAR_MAX_PX = 120

export interface CombatYieldBarProps {
  /** offensive_conversion = 225 × (kills + assists/3) / damage_dealt */
  offensiveConversion?: number | null
  /** defensive_resistance = damage_taken / (225 × deaths) */
  defensiveResistance?: number | null
  /** Dégâts moyens par kill (pour tooltip) */
  damagePerKill?: number | null
  /** Dégâts moyens par mort (pour tooltip) */
  damagePerDeath?: number | null
}

function normalizeBar(value: number, p80: number): number {
  const clipped = Math.min(value, p80 * CLIP_FACTOR)
  return Math.min(clipped / p80, CLIP_FACTOR) / CLIP_FACTOR
}

function barWidth(value: number | null | undefined, p80: number): number {
  if (value == null || value <= 0) return 0
  return Math.round(normalizeBar(value, p80) * BAR_MAX_PX)
}

interface TooltipProps {
  offensiveConversion: number | null | undefined
  defensiveResistance: number | null | undefined
  damagePerKill: number | null | undefined
  damagePerDeath: number | null | undefined
}

function Tooltip({ offensiveConversion, defensiveResistance, damagePerKill, damagePerDeath }: TooltipProps) {
  return (
    <div className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 z-50 w-48 rounded-md bg-popover border border-border px-3 py-2 text-xs shadow-lg pointer-events-none">
      <div className="flex justify-between gap-2 mb-1">
        <span className="text-[#4ade80] font-semibold">Offensif</span>
        <span className="text-muted-foreground">{offensiveConversion != null ? offensiveConversion.toFixed(2) : '—'}</span>
      </div>
      {damagePerKill != null && (
        <div className="text-muted-foreground mb-1">{Math.round(damagePerKill)} dmg/kill</div>
      )}
      <div className="flex justify-between gap-2 mb-1">
        <span className="text-[#60a5fa] font-semibold">Défensif</span>
        <span className="text-muted-foreground">{defensiveResistance != null ? defensiveResistance.toFixed(2) : '—'}</span>
      </div>
      {damagePerDeath != null && (
        <div className="text-muted-foreground">{Math.round(damagePerDeath)} dmg/mort</div>
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
}: CombatYieldBarProps) {
  const [hovered, setHovered] = useState(false)

  const ocWidth = barWidth(offensiveConversion, OC_P80)
  const drWidth = barWidth(defensiveResistance, DR_P80)

  const hasData = (offensiveConversion != null && offensiveConversion > 0) ||
    (defensiveResistance != null && defensiveResistance > 0)

  return (
    <div
      className="relative flex items-center justify-center gap-0.5"
      style={{ width: BAR_MAX_PX * 2 + 4, height: 10 }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      {/* Barre offensive (gauche, pousse vers la droite) */}
      <div className="flex justify-end" style={{ width: BAR_MAX_PX }}>
        {hasData && (
          <div
            className="h-2 rounded-l-full transition-all duration-300"
            style={{
              width: ocWidth,
              backgroundColor: ocWidth > 0 ? '#4ade80' : 'transparent',
              opacity: ocWidth > 0 ? 1 : 0,
            }}
          />
        )}
      </div>

      {/* Séparateur central */}
      <div className="w-px h-3 bg-border flex-shrink-0" />

      {/* Barre défensive (droite) */}
      <div className="flex justify-start" style={{ width: BAR_MAX_PX }}>
        {hasData && (
          <div
            className="h-2 rounded-r-full transition-all duration-300"
            style={{
              width: drWidth,
              backgroundColor: drWidth > 0 ? '#60a5fa' : 'transparent',
              opacity: drWidth > 0 ? 1 : 0,
            }}
          />
        )}
      </div>

      {hovered && hasData && (
        <Tooltip
          offensiveConversion={offensiveConversion}
          defensiveResistance={defensiveResistance}
          damagePerKill={damagePerKill}
          damagePerDeath={damagePerDeath}
        />
      )}
    </div>
  )
}
