/**
 * UsageBandLegend — LA LÉGENDE DES BANDES DE RÉGULARITÉ, écrite UNE fois sous les bandes.
 *
 * « X/X au-dessus de la parité » était répété à droite de CHAQUE ligne (2026-09-21, retour
 * utilisateur) : sur cinq grandeurs, la même phrase cinq fois pour cinq fractions. Le compte
 * reste à droite de sa ligne, nu (« 3/8 ») ; ce que la couleur des cases veut dire vit ici,
 * une seule fois, avec les vraies encres (`BAND_TONE_INKS`).
 */
import { BAND_TONE_INKS } from './usageInks'
import type { UsageText } from './usageI18n'

function Cell({ ink, label }: { ink: string | undefined; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span
        aria-hidden="true"
        className="inline-block h-3 w-3 flex-none bg-muted"
        style={ink != null ? { backgroundColor: ink } : undefined}
      />
      {label}
    </span>
  )
}

export function UsageBandLegend({ t }: { t: UsageText }) {
  return (
    <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-3xs text-muted-foreground">
      <Cell ink={BAND_TONE_INKS.above} label={t.bandLegendAbove} />
      <Cell ink={BAND_TONE_INKS.near} label={t.bandLegendNear} />
      <Cell ink={BAND_TONE_INKS.below} label={t.bandLegendBelow} />
      <Cell ink={BAND_TONE_INKS.unmeasured} label={t.bandLegendUnmeasured} />
    </div>
  )
}
