/**
 * UsageCountsGrid.tsx — LE RENDU DE LA VARIANTE COMPTES (décision P9,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.8/E6.2) : une ligne par grandeur (famille
 * d'équipement OU coéquipier suivi), l'axe est en objets pris, AUCUN trait de parité.
 *
 * RÉUTILISE `UsageGauge` de `UsageForms.tsx` À L'IDENTIQUE (rail, pile des trois issues,
 * repères de taux, texte) — seul le sens de `valuePct` change (`usageCountsModel.ts`
 * calcule une longueur relative au maximum de l'axe, jamais une part d'équipe) ; aucune
 * seconde définition de la cellule (CLAUDE.md n°6). Même chose pour le DÉPLIABLE de lignes
 * (D2, les armes de base) : `UsageCollapseToggle` est le bouton des deux grilles.
 *
 * DOM ET CSS, PAS ECHARTS — même choix que `ValueGrid`/`UsageForms` : un problème de
 * MISE EN PAGE (alignement de rails), sans zoom ni animation.
 *
 * PAS DE LÉGENDE DE TEXTURE ICI, et ce n'est pas un oubli : cette grille n'a QU'UN
 * dénominateur, donc aucune hachure à expliquer (`UsageGauge` y rend toujours l'aplat).
 */
import { Fragment, useId, useState } from 'react'

import { COLUMN_GAP, LABEL_WIDTH, UsageCollapseToggle, UsageGauge } from './UsageForms'
import type { UsageCountsGridModel, UsageCountsRowModel } from './usageCountsModel'

/** L'axe gradué 0 · milieu · max+unité d'une grille en comptes. */
function CountsAxis({ axisMaxText }: { axisMaxText: string }) {
  return (
    <div className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums">
      <span className="absolute left-0 top-0.5">0</span>
      <span className="absolute right-0 top-0.5">{axisMaxText}</span>
    </div>
  )
}

function CountsRows({ rows }: { rows: UsageCountsRowModel[] }) {
  return (
    <>
      {rows.map((row) => (
        <Fragment key={row.key}>
          <div className="overflow-hidden whitespace-nowrap text-xs" title={row.hint ?? row.label}>
            <span className="truncate">{row.label}</span>
          </div>
          <UsageGauge gauge={row.gauge} />
        </Fragment>
      ))}
    </>
  )
}

export function UsageCountsGrid({
  grid,
  collapsedRows,
  collapsedLabel,
}: {
  grid: UsageCountsGridModel
  /** Lignes repliées derrière un bouton, FERMÉ par défaut (D2 : les armes de base). */
  collapsedRows?: UsageCountsRowModel[]
  collapsedLabel?: string
}) {
  const [open, setOpen] = useState(false)
  const groupId = useId()
  const collapsed = collapsedRows ?? []
  if (grid.rows.length === 0 && collapsed.length === 0) return null

  // PLUS DE SCROLL HORIZONTAL (2026-09-19) : la colonne de jauge part de ZÉRO
  // (`minmax(0, 1fr)`) et la grille n'a plus de largeur plancher — elle se répartit dans
  // la carte au lieu de la déborder. Une grille de comptes qui défile latéralement cache
  // la moitié de ses lignes sans le dire.
  const gridStyle = {
    gridTemplateColumns: `${LABEL_WIDTH}px minmax(0, 1fr) max-content`,
    columnGap: COLUMN_GAP,
  }

  return (
    <div className="min-w-0">
      <div className="grid items-center gap-y-[6px]" style={gridStyle} id={groupId}>
        <CountsRows rows={grid.rows} />
        {collapsed.length > 0 && collapsedLabel != null && (
          <>
            <UsageCollapseToggle
              label={collapsedLabel}
              open={open}
              onToggle={() => setOpen((v) => !v)}
              controls={groupId}
            />
            {open && <CountsRows rows={collapsed} />}
          </>
        )}
        <div aria-hidden="true" />
        <CountsAxis axisMaxText={grid.axisMaxText} />
        <div aria-hidden="true" />
      </div>
    </div>
  )
}
