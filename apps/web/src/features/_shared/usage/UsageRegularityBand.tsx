/**
 * UsageRegularityBand — une case par match mesuré, dans l'ordre de la session, teintée par
 * l'écart à la parité.
 *
 * Extrait de `UsageForms.tsx` le 2026-09-21 (seuil de taille, CLAUDE.md n°5).
 *
 * LA LÉGENDE N'EST PLUS RÉPÉTÉE PAR LIGNE (même jour, retour utilisateur). « 3/8 au-dessus
 * de la parité » s'écrivait à droite de CHAQUE bande : sur cinq grandeurs, la même phrase
 * cinq fois pour cinq fractions. Ce qui reste à droite de la ligne est le COMPTE NU
 * (« 3/8 ») ; ce que les couleurs veulent dire vit dans `UsageBandLegend`, posée UNE fois
 * sous les bandes par l'appelant — c'est lui qui sait combien de bandes il empile.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { LABEL_WIDTH } from './UsageForms'
import { BAND_TONE_INKS } from './usageInks'
import type { UsageBandCell } from './usageRegularityBandModel'

/** La même largeur de libellé qu'en colonne divisée dans `UsageForms` (alignement). */
const DENSE_LABEL_WIDTH = 104

/** L'encre d'une case : au-dessus / à / sous la parité, ou non mesurée. */
function bandCellStyle(cell: UsageBandCell) {
  const ink = BAND_TONE_INKS[cell.tone]
  // `unmeasured` n'a pas d'encre : la case reste sur le fond `bg-muted` — non mesuré n'est
  // pas une donnée.
  return ink != null ? { backgroundColor: ink } : undefined
}

export function UsageRegularityBand({
  label,
  cells,
  caption,
  dense = false,
}: {
  label: string
  cells: UsageBandCell[]
  /** Le COMPTE NU des matchs au-dessus de la parité d'équipe (« 3/8 »), jamais une phrase. */
  caption: string | null
  /** Colonne divisée : MÊMES cases, plus petites — jamais une bande retirée. */
  dense?: boolean
}) {
  if (cells.length === 0) return null
  return (
    <div
      className={`grid items-center gap-y-0.5 ${dense ? 'gap-x-2' : 'gap-x-3.5'}`}
      style={{ gridTemplateColumns: `${dense ? DENSE_LABEL_WIDTH : LABEL_WIDTH}px 1fr` }}
    >
      <div
        className={`overflow-hidden whitespace-nowrap ${dense ? 'text-3xs' : 'text-xs'}`}
        title={label}
      >
        <span className="truncate">{label}</span>
      </div>
      <div className={`flex flex-wrap items-center ${dense ? 'gap-[2px]' : 'gap-[3px]'}`}>
        {cells.map((cell) => (
          <Tooltip key={cell.matchId} content={cell.tooltip}>
            <span
              className={`flex-none bg-muted ${dense ? 'h-2.5 w-2.5' : 'h-3.5 w-3.5'}`}
              style={bandCellStyle(cell)}
              tabIndex={0}
              role="img"
              aria-label={cell.tooltip}
            />
          </Tooltip>
        ))}
        {caption != null && (
          <span
            className={`pl-2 tabular-nums text-muted-foreground ${dense ? 'text-3xs' : 'text-[11px]'}`}
          >
            {caption}
          </span>
        )}
      </div>
    </div>
  )
}
