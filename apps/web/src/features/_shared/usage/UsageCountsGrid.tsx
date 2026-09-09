/**
 * UsageCountsGrid.tsx — LE RENDU DE LA VARIANTE COMPTES (décision P9,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.8/E6.2) : une ligne par grandeur (famille
 * d'équipement OU coéquipier suivi), l'axe est en objets pris, AUCUN trait de parité.
 *
 * RÉUTILISE `UsageGauge` de `UsageForms.tsx` À L'IDENTIQUE (rail, pile des trois issues,
 * repères de taux, texte) — seul le sens de `valuePct` change (`usageCountsModel.ts`
 * calcule une longueur relative au maximum de l'axe, jamais une part d'équipe) ; aucune
 * seconde définition de la cellule (CLAUDE.md n°6).
 *
 * DOM ET CSS, PAS ECHARTS — même choix que `ValueGrid`/`UsageForms` : un problème de
 * MISE EN PAGE (alignement de rails), sans zoom ni animation.
 */
import { Fragment } from 'react'

import { COLUMN_GAP, GAUGE_MIN, LABEL_WIDTH, UsageGauge } from './UsageForms'
import type { UsageCountsGridModel } from './usageCountsModel'

/** L'axe gradué 0 · milieu · max+unité d'une grille en comptes. */
function CountsAxis({ axisMaxText }: { axisMaxText: string }) {
  return (
    <div className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums">
      <span className="absolute left-0 top-0.5">0</span>
      <span className="absolute right-0 top-0.5">{axisMaxText}</span>
    </div>
  )
}

export function UsageCountsGrid({ grid }: { grid: UsageCountsGridModel }) {
  if (grid.rows.length === 0) return null

  const gridStyle = {
    gridTemplateColumns: `${LABEL_WIDTH}px minmax(${GAUGE_MIN}px, 1fr) max-content`,
    minWidth: LABEL_WIDTH + GAUGE_MIN + COLUMN_GAP,
    columnGap: COLUMN_GAP,
  }

  return (
    <div className="overflow-x-auto">
      <div className="grid items-center gap-y-[6px]" style={gridStyle}>
        {grid.rows.map((row) => (
          <Fragment key={row.key}>
            <div className="overflow-hidden whitespace-nowrap text-xs" title={row.label}>
              <span className="truncate">{row.label}</span>
            </div>
            <UsageGauge gauge={row.gauge} />
          </Fragment>
        ))}
        <div aria-hidden="true" />
        <CountsAxis axisMaxText={grid.axisMaxText} />
        <div aria-hidden="true" />
      </div>
    </div>
  )
}
