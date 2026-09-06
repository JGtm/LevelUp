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

import type { HeatmapMode } from '../layers/heatmapLayer'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

interface ReplayHeatmapLegendProps {
  locale: ReplayLocale
  mode: HeatmapMode
}

export function ReplayHeatmapLegend({ locale, mode }: ReplayHeatmapLegendProps) {
  const t = REPLAY_TEXT[locale]
  // Rampe d'INTENSITÉ (2026-08-18) : bleu -> rouge -> violet, le violet ne peignant que le
  // haut de l'échelle. La grandeur mesurée reste une intensité NEUTRE (du temps, un nombre de
  // morts), donc la rampe « à connotation » — qui dirait vert = bon / rouge = mauvais sur des
  // lieux qui ne sont ni l'un ni l'autre — reste écartée. Le dégradé ci-dessous porte TOUS les
  // arrêts : une légende à deux bouts mentirait sur une rampe qui en a trois.
  const arrets = heatmapRampTokens('intensity').map(tokenCssVar)
  return (
    <div
      className="absolute bottom-2 left-3 rounded border border-border bg-card/85 px-2 py-1.5 text-[10px] leading-tight text-muted-foreground"
      title={t.heatLegendHint}
    >
      <div className="font-medium text-foreground">{t.heatmapMode[mode]}</div>
      <div className="mt-1 flex items-center gap-1.5">
        <span>{t.heatLegendLow}</span>
        <span
          aria-hidden
          className="h-2 w-16 rounded-sm"
          style={{ backgroundImage: `linear-gradient(to right, ${arrets.join(', ')})` }}
        />
        <span>{t.heatLegendHigh}</span>
      </div>
    </div>
  )
}
