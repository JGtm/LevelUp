/**
 * CombatYieldDisplay — bande Rendement / Résistance partagée.
 *
 * Reprend le format de la Synthesis : `dégâts/frag · rendement% · barre · résistance% · dégâts/mort`.
 * S'adapte à la largeur du bloc parent (ResizeObserver) sur 3 paliers :
 *   - large  (≥ HORIZONTAL_MIN)         → ruban horizontal complet
 *   - moyen  (≥ assez pour MIN_BAR)     → empilé : barre au-dessus, valeurs en dessous
 *   - étroit (barre < MIN_BAR_PX)       → barre masquée, valeurs seules
 *
 * Les valeurs chiffrées (2 %, dégâts/frag, dégâts/mort) restent TOUJOURS affichées :
 * seul le visuel de la barre disparaît quand l'espace est insuffisant.
 *
 * Source unique réutilisée par Home, KpiGrid (solo/squad), SessionSummaryCard,
 * tuile match et Synthesis. Couleurs via tokens (divergent-pos / divergent-neutral).
 */
import { useEffect, useRef, useState } from 'react'
import { tokenCssVar } from '@/lib/accessibility'
import { formatOffensiveConversion, formatDefensiveResistance } from '@/lib/formatters'
import { CombatYieldBar } from './combat-yield-bar'

/** Au-dessus de cette largeur totale, on rend le ruban horizontal. */
const HORIZONTAL_MIN_PX = 420
/** Largeur réservée aux textes (2 % + 2 dégâts + gaps) en mode horizontal. */
const TEXT_RESERVE_PX = 250
/** Largeur native max de la barre (= défaut CombatYieldBar). */
const MAX_BAR_PX = 304
/** Sous cette largeur de barre, on masque la barre (valeurs seules). */
const MIN_BAR_PX = 100
/** Largeur réservée aux 2 % qui encadrent la barre en mode empilé. */
const STACKED_PCT_RESERVE_PX = 96

export interface CombatYieldDisplayProps {
  /** offensive_conversion brut (ex 0.42). */
  offensiveConversion?: number | null
  /** defensive_resistance brut, baseline 1.0 (ex 1.18). */
  defensiveResistance?: number | null
  /** Dégâts moyens par frag (Σ damage_dealt / Σ kills). */
  dmgPerKill?: number | null
  /** Dégâts moyens par mort (Σ damage_taken / Σ deaths). */
  dmgPerDeath?: number | null
  /** Label optionnel affiché au-dessus (ex "Rendement / Résistance"). */
  label?: string
  /** Alignement horizontal du contenu (défaut center). */
  align?: 'center' | 'start'
  className?: string
}

function ocColor() {
  return tokenCssVar('divergent-pos')
}
function drColor() {
  return tokenCssVar('divergent-neutral')
}

export function CombatYieldDisplay({
  offensiveConversion,
  defensiveResistance,
  dmgPerKill,
  dmgPerDeath,
  label,
  align = 'center',
  className,
}: CombatYieldDisplayProps) {
  const ref = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState<number | null>(null)

  useEffect(() => {
    const el = ref.current
    if (!el || typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver((entries) => {
      for (const entry of entries) setWidth(entry.contentRect.width)
    })
    ro.observe(el)
    // Mesure initiale synchrone (évite un premier rendu sans barre).
    setWidth(el.clientWidth)
    return () => ro.disconnect()
  }, [])

  const hasData =
    (offensiveConversion != null && offensiveConversion > 0) ||
    (defensiveResistance != null && defensiveResistance > 0)

  const horizontal = width != null && width >= HORIZONTAL_MIN_PX
  const barBudget = horizontal
    ? Math.min(MAX_BAR_PX, width - TEXT_RESERVE_PX)
    : Math.min(MAX_BAR_PX, (width ?? MAX_BAR_PX) - STACKED_PCT_RESERVE_PX)
  const showBar = width != null && hasData && barBudget >= MIN_BAR_PX

  const alignClass = align === 'center' ? 'items-center' : 'items-start'

  if (!hasData) {
    return (
      <div ref={ref} className={`flex flex-col ${alignClass} ${className ?? ''}`}>
        {label && <span className="text-xs text-muted-foreground">{label}</span>}
        <span className="text-lg font-bold text-muted-foreground">—</span>
      </div>
    )
  }

  const dmgFrag = dmgPerKill != null && (
    <span className="text-2xs text-muted-foreground tabular-nums">{Math.round(dmgPerKill)} dégâts/frag</span>
  )
  const dmgMort = dmgPerDeath != null && (
    <span className="text-2xs text-muted-foreground tabular-nums">{Math.round(dmgPerDeath)} dégâts/mort</span>
  )
  const ocPct = (
    <span className="text-sm font-semibold tabular-nums" style={{ color: ocColor() }}>
      {formatOffensiveConversion(offensiveConversion)}
    </span>
  )
  const drPct = (
    <span className="text-sm font-semibold tabular-nums" style={{ color: drColor() }}>
      {formatDefensiveResistance(defensiveResistance)}
    </span>
  )
  const bar = showBar && (
    <CombatYieldBar
      offensiveConversion={offensiveConversion}
      defensiveResistance={defensiveResistance}
      damagePerKill={dmgPerKill}
      damagePerDeath={dmgPerDeath}
      widthPx={barBudget}
    />
  )

  // Ruban horizontal (parent large) : dégâts/frag · % · barre · % · dégâts/mort.
  if (horizontal) {
    return (
      <div ref={ref} className={`flex flex-col ${alignClass} gap-0.5 ${className ?? ''}`}>
        {label && <span className="text-xs text-muted-foreground">{label}</span>}
        <div className="flex items-center gap-2">
          {dmgFrag}
          {ocPct}
          {bar}
          {drPct}
          {dmgMort}
        </div>
      </div>
    )
  }

  // Empilé (parent moyen/étroit) : ligne barre avec les 2 % aux extrémités
  // (barre au centre), puis les dégâts/frag·mort en dessous aux extrémités.
  return (
    <div ref={ref} className={`flex flex-col ${alignClass} gap-1 ${className ?? ''}`}>
      {label && <span className="text-xs text-muted-foreground">{label}</span>}
      <div className="flex w-full items-center justify-between gap-2">
        {ocPct}
        {bar}
        {drPct}
      </div>
      {(dmgFrag || dmgMort) && (
        <div className="flex w-full items-center justify-between gap-3">
          {dmgFrag ?? <span />}
          {dmgMort ?? <span />}
        </div>
      )}
    </div>
  )
}
