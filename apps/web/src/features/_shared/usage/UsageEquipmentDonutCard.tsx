/**
 * UsageEquipmentDonutCard.tsx — LE RENDU d'un des deux donuts (décisions P10/P11,
 * PLAN_EQUIPEMENT_GACHIS_2026-09-09, étapes E5.9/E6.3) : l'anneau (via la primitive
 * `DonutChart`, JAMAIS un donut écrit à la main) + une légende COULEUR SEULE (P11 : la
 * légende ne dit que la couleur) + les sous-totaux emboîtés, séparés par un filet.
 *
 * LE CENTRE NE PORTE QUE LE NOMBRE. Le libellé d'unité sous l'anneau a été retiré le
 * 2026-09-19 : le titre de la carte dit déjà de quoi il s'agit, et ECharts
 * posait ce texte sur une ligne sans largeur, derrière la couronne.
 *
 * Aucun calcul ici : `usageEquipmentPartiesModel.ts` construit le modèle complet.
 */
import { DonutChart } from '@/components/charts/DonutChart'
import { tokenCssVar } from '@/lib/accessibility'

import type { UsageDonutModel } from './usageEquipmentPartiesModel'

export function UsageEquipmentDonutCard({ model }: { model: UsageDonutModel | null }) {
  if (model == null) return null
  return (
    <div className="flex flex-wrap items-center gap-5">
      {/* `min-w` + `flex-1` : l'anneau a besoin de largeur pour que sa légende d'unité tienne
          sur une ou deux lignes lisibles ; sans plancher, la colonne se réduisait à la taille
          du canvas et le libellé se serrait en colonne de mots. */}
      <div className="flex min-w-[200px] flex-1 flex-col items-center gap-1">
        <DonutChart
          series={model.series}
          sliceColors={model.sliceColors}
          height={188}
          centerValue={model.centerValue}
          arcLabelKind="value"
          // La SectionCard porte déjà le cadre : sans ceci, l'anneau s'entourait d'une
          // seconde bordure et d'un second fond (prop ajoutée à ChartCard en b8166c9bf).
          frameless
        />
      </div>
      <div className="flex min-w-[190px] flex-col gap-1.5 text-xs">
        {model.legendRows.map((row) => (
          <div key={row.label} className="grid grid-cols-[12px_1fr] items-baseline gap-2">
            <span
              aria-hidden="true"
              className="h-3 w-3 rounded-sm"
              style={{ backgroundColor: tokenCssVar(row.token) }}
            />
            <span className="truncate">{row.label}</span>
          </div>
        ))}
        {model.subtotals.map((sub) => (
          <div
            key={sub.label}
            className="mt-0.5 grid grid-cols-[12px_1fr_max-content] items-baseline gap-2 border-t border-border pt-1.5 text-muted-foreground"
          >
            <span aria-hidden="true" />
            <span>{sub.label}</span>
            <span className="tabular-nums font-semibold text-foreground">{sub.value}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
