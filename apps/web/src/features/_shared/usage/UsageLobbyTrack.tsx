/**
 * UsageLobbyTrack — LA PISTE 100 % DU LOBBY : coloré = nous, découpé par joueur ; hachuré =
 * eux, anonyme. Le compte brut et la part sont écrits dans les segments assez larges, et
 * toujours dans l'infobulle.
 *
 * Extrait de `UsageForms.tsx` le 2026-09-21 (seuil de taille, CLAUDE.md n°5).
 *
 * DEUX CORRECTIONS DU MÊME JOUR, sur la même cause. La hachure de « eux » portait
 * `opacity: 0.45` SUR LE SEGMENT : elle délavait donc aussi l'étiquette blanche posée
 * dessus, et, faute d'assise, le segment se lisait plus MINCE que ses voisins pleins (des
 * rayures sur le fond de la carte contre un aplat franc). La texture est désormais un
 * CALQUE `aria-hidden` sous l'étiquette, sur une assise `--muted` (`ENEMY_HATCH`) : même
 * rectangle, même épaisseur, étiquette à pleine encre sur TOUS les segments.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { UsageHatchLegend } from './UsageHatchLegend'
import { usagePlayerInk } from './usageGrids'
import type { UsageText } from './usageI18n'
import type { UsageTrackSegment } from './usageLobbyTrackModel'
import { ALLY_INK, ENEMY_HATCH } from './usageInks'

/** La part du segment qui doit peser assez pour porter son étiquette écrite. */
const LABEL_MIN_FRACTION = 0.12

/**
 * L'ASSISE d'un segment, selon sa nature (voir l'en-tête du fichier). « eux » n'en a pas de
 * couleur : sa texture est un calque, rendu à part.
 */
function segmentFill(seg: UsageTrackSegment): string | undefined {
  switch (seg.kind) {
    case 'me':
      return usagePlayerInk('me')
    case 'squad':
      return usagePlayerInk('squad', seg.squadIndex ?? 0)
    case 'team-rest':
      return ALLY_INK
    case 'enemy':
      return undefined
  }
}

export function UsageLobbyTrack({
  segments,
  label,
  t,
  dense = false,
}: {
  segments: UsageTrackSegment[]
  label: string
  t: UsageText
  /** Colonne divisée : même piste, hauteur réduite (rien n'est retiré). */
  dense?: boolean
}) {
  const total = segments.reduce((a, s) => a + s.count, 0)
  return (
    <div className="min-w-0">
      <div className={dense ? 'flex h-[16px]' : 'flex h-[22px]'} role="img" aria-label={label}>
        {/* La largeur est portée par l'ITEM du flex, en `calc(%)` — jamais un flexGrow sur le
            contenu d'un Tooltip : le wrapper du Tooltip garde flex-grow 0 et les segments se
            dimensionneraient à leur texte, pas à leurs comptes (revue adversariale
            2026-09-05 ; pattern correct : MatchPadControlSection). */}
        {segments.map((seg) => {
          const fill = segmentFill(seg)
          return (
            <div
              key={seg.key}
              className="mr-[2px] h-full last:mr-0"
              style={{ width: total > 0 ? `calc(${(seg.count / total) * 100}% - 2px)` : '0%' }}
            >
              <Tooltip content={seg.tooltip} className="h-full w-full">
                {/* `text-white` : le libellé est posé SUR l'aplat du segment, quelle que soit
                    la palette réglée — le contraste d'un texte dans un aplat, pas une couleur
                    sémantique (même usage que `components/charts/StackedTrack`). L'étiquette est
                    au-dessus du calque de texture (`relative`), jamais sous lui. */}
                <div
                  className="relative flex h-full w-full items-center justify-center overflow-hidden whitespace-nowrap bg-muted px-1 text-3xs font-semibold text-white"
                  style={fill != null ? { backgroundColor: fill } : ENEMY_HATCH}
                  tabIndex={0}
                  role="img"
                  aria-label={seg.tooltip}
                >
                  {/* Le libellé n'est écrit que si le segment pèse assez pour le porter. */}
                  <span className="relative">
                    {total > 0 && seg.count / total >= LABEL_MIN_FRACTION
                      ? `${seg.count} · ${seg.pctText}`
                      : ''}
                  </span>
                </div>
              </Tooltip>
            </div>
          )
        })}
      </div>
      <div className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums">
        <span className="absolute left-0 top-0.5">0</span>
        <span className="absolute left-1/2 top-0.5 -translate-x-1/2">50</span>
        <span className="absolute right-0 top-0.5">100 %</span>
      </div>
      <UsageHatchLegend t={t} variant="track" />
    </div>
  )
}
