/**
 * ChartLegend — légende HTML réutilisable rendue HORS canvas (pied de card via la
 * prop `legend` de ChartCard, ou en ligne). Chaque entrée = pastille couleur + libellé.
 *
 * Les couleurs sont des valeurs DÉJÀ RÉSOLUES via token (`resolveToken` / `fragClassColor`…)
 * — la MÊME source que les séries ECharts (cf. `_utils.getLegendBase` et les builders).
 * Ce composant est purement présentationnel : la réactivité palette/thème est portée par
 * le composant appelant (qui re-résout ses couleurs sur `useColorPaletteVersion` /
 * `useThemeVersion`, à l'image de FragSunburst). AUCUN littéral hex ici (garde-rail
 * color-tokens) : `backgroundColor` reçoit une valeur runtime, jamais une couleur codée.
 *
 * Style aligné sur la légende de `SquadToggleLegendChart` (pastille `rounded-sm` bordée,
 * texte discret) pour une cohérence visuelle avec l'existant.
 */
import type { CSSProperties } from 'react'

/** Opacité d'une entrée estompée (survol lié : classes/series non survolées). */
const DIMMED_OPACITY = 0.28

export interface ChartLegendItem {
  /** Libellé i18n-résolu en amont. */
  label: string
  /** Couleur résolue via token (hex/rgba) — jamais un littéral hex codé en dur ici. */
  color: string
  /** Clé stable (React key + argument des callbacks de survol). Défaut = label. */
  key?: string
  /** Estompe l'entrée (survol lié : entrée hors de la classe survolée). */
  dimmed?: boolean
}

export interface ChartLegendProps {
  items: ChartLegendItem[]
  /** Alignement horizontal (défaut center — pied de card). */
  align?: 'center' | 'start'
  className?: string
  /** Survol lié optionnel : remonte la clé survolée (ou null en sortie). */
  onItemHover?: (key: string | null) => void
}

export function ChartLegend({ items, align = 'center', className = '', onItemHover }: ChartLegendProps) {
  if (items.length === 0) return null
  const justify = align === 'center' ? 'justify-center' : 'justify-start'
  return (
    <ul
      className={`flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground ${justify} ${className}`.trim()}
      data-testid="chart-legend"
    >
      {items.map((it) => {
        const key = it.key ?? it.label
        const style: CSSProperties = { opacity: it.dimmed ? DIMMED_OPACITY : 1 }
        const hoverProps = onItemHover
          ? { onMouseEnter: () => onItemHover(key), onMouseLeave: () => onItemHover(null) }
          : {}
        return (
          <li key={key} className="inline-flex cursor-default items-center gap-1.5" style={style} {...hoverProps}>
            <span
              className="inline-block h-2.5 w-2.5 shrink-0 rounded-sm border border-border"
              style={{ backgroundColor: it.color }}
              aria-hidden
            />
            <span>{it.label}</span>
          </li>
        )
      })}
    </ul>
  )
}
