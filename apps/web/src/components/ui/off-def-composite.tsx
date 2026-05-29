import type { CSSProperties } from 'react'

import { tokenCssVar } from '@/lib/accessibility'

export interface OffDefCompositeProps {
  /** Valeur BRUTE avg_offensive_conversion (ex 0.42). null/undefined → fallback "—". */
  offensiveConversion?: number | null
  /** Valeur BRUTE avg_defensive_resistance, baseline 1.0 (ex 1.18). null/undefined → fallback "—". */
  defensiveResistance?: number | null
  /** Alignement du contenu sous le label parent : 'center' (home) ou 'start' (grille session). */
  align?: 'center' | 'start'
}

/**
 * OffDefComposite — barre composite Rendement (OC) / Résistance (DR) + 2 valeurs colorées.
 *
 * Format repris de HomeHeroKPIGrid (tuile Off/Def). Réutilisé tel quel par la home
 * (align="center") et par la grille SessionBriefing (align="start").
 *
 * Dissociation volontaire (iso-visuelle avec la home) :
 *   - proportions des segments = valeurs BRUTES → off/(off+def) et def/(off+def)
 *   - valeurs AFFICHÉES = off*100 % et (def-1)*100 % (baseline DR = 1.0)
 * Le segment DR utilise donc `def` brut, pas `def-1`.
 *
 * Couleurs (tokens uniquement) : OC = divergent-pos, DR = divergent-neutral.
 */
export function OffDefComposite({
  offensiveConversion,
  defensiveResistance,
  align = 'center',
}: OffDefCompositeProps) {
  const hasOffDef = offensiveConversion != null || defensiveResistance != null
  const off = offensiveConversion ?? 0
  const def = defensiveResistance ?? 0
  const total = off + def

  if (!hasOffDef) {
    return <p className="text-xl font-bold text-muted-foreground">—</p>
  }

  const valuesJustify = align === 'center' ? 'justify-center' : 'justify-start'
  const segWidth = (raw: number): CSSProperties => ({
    width: total > 0 ? `${(raw / total) * 100}%` : '50%',
  })

  return (
    <div className="w-full">
      <div className="h-2 w-full rounded-full overflow-hidden flex">
        {off > 0 && (
          <div className="h-full" style={{ ...segWidth(off), backgroundColor: tokenCssVar('divergent-pos') }} />
        )}
        {def > 0 && (
          <div className="h-full" style={{ ...segWidth(def), backgroundColor: tokenCssVar('divergent-neutral') }} />
        )}
      </div>
      <div className={`flex ${valuesJustify} gap-3 mt-2`}>
        <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-pos') }}>
          {(off * 100).toFixed(0)}%
        </span>
        <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-neutral') }}>
          {((def - 1) * 100).toFixed(0)}%
        </span>
      </div>
    </div>
  )
}
