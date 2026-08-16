/**
 * ReplayHeatmapLegend — la légende de la carte de chaleur, posée dans un coin du canvas.
 *
 * EN DOM, PAS DANS LE CANVAS. Un texte peint au canvas n'est ni sélectionnable, ni lu par
 * un lecteur d'écran, ni traduit par le navigateur : la légende est ce qui EXPLIQUE le
 * calque, c'est le dernier endroit où l'écrire en pixels. Le canvas garde le dessin, le DOM
 * garde les mots (même partage que la barre de lecture sous la carte).
 *
 * LA RAMPE VIENT DU MÊME ENDROIT QUE LE CALQUE — `heatmapRampTokens`, la source unique du
 * dépôt (garde-rail heatmapColors.guard.test.ts). Ici en variables CSS (`tokenCssVar`), là
 * en hex résolus (`resolveToken`) : deux lectures du MÊME token, donc la légende ne peut
 * pas montrer une autre rampe que celle peinte sur la carte.
 *
 * LES EXTRÉMITÉS NOMMENT LA QUANTITÉ (« rare » -> « fréquent ») et le titre dit DE QUOI il
 * s'agit (présence, éliminations) : une légende dont les bouts ne se lisent pas n'informe
 * de rien. Le détail de l'étalonnage (médiane -> 95e centile, saturation au-delà) est dans
 * l'infobulle, pas dans le cadre — il ne doit pas manger la carte.
 */
import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { tokenCssVar } from '@/lib/accessibility'

import type { HeatmapMode } from './heatmapLayer'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'

interface ReplayHeatmapLegendProps {
  locale: ReplayLocale
  mode: HeatmapMode
}

export function ReplayHeatmapLegend({ locale, mode }: ReplayHeatmapLegendProps) {
  const t = REPLAY_TEXT[locale]
  // Rampe de FRÉQUENCE : la grandeur mesurée est une intensité NEUTRE (du temps, un nombre
  // de morts), pas une performance. La rampe « à connotation » dirait vert = bon / rouge =
  // mauvais sur des lieux qui ne sont ni l'un ni l'autre — et elle est iso-luminante en
  // vision daltonienne, là où celle-ci reste monotone en luminance dans TOUTES les palettes.
  const [low, high] = heatmapRampTokens('frequency').map(tokenCssVar)
  return (
    <div
      className="absolute bottom-2 left-2 rounded border border-border bg-card/85 px-2 py-1.5 text-[10px] leading-tight text-muted-foreground"
      title={t.heatLegendHint}
    >
      <div className="font-medium text-foreground">{t.heatmapMode[mode]}</div>
      <div className="mt-1 flex items-center gap-1.5">
        <span>{t.heatLegendLow}</span>
        <span
          aria-hidden
          className="h-2 w-16 rounded-sm"
          style={{ backgroundImage: `linear-gradient(to right, ${low}, ${high})` }}
        />
        <span>{t.heatLegendHigh}</span>
      </div>
    </div>
  )
}
