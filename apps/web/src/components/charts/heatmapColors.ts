/**
 * heatmapColors — sélection CENTRALISÉE des rampes de couleur des visualMap de
 * heatmaps ECharts. Un seul endroit décide quels tokens sémantiques composent la
 * rampe, selon la nature de la donnée ET la palette d'accessibilité active.
 *
 * Motivation accessibilité (daltonisme) : la rampe « à connotation » cold→hot
 * (heatmap-cold / heatmap-hot) oppose bleu et vermillon, quasi ISO-LUMINANTS
 * dans les palettes CVD (okabe-ito, cividis, tol-bright). Privé de l'axe chroma,
 * un daltonien n'y perçoit presque aucun dégradé — la heatmap s'effondre en
 * aplat. La rampe NEUTRE de fréquence (heatmap-freq-low / heatmap-freq-high) est
 * mono-teinte et monotone en luminance (sombre → clair) dans TOUTES les palettes :
 * lisible pour tous les types de daltonisme, et même en niveaux de gris.
 *
 * Règle : dès qu'une palette d'accessibilité est active, les heatmaps continues
 * séquentielles basculent sur la rampe de fréquence (luminance monotone). En
 * palette 'default' (vision standard), on conserve la rampe cold→hot familière
 * (vert = bon / rouge = mauvais). Les heatmaps intrinsèquement neutres
 * (mode 'frequency') utilisent la rampe de fréquence dans toutes les palettes.
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { ColorPalette } from '@/stores/settingsDraftStore'

/**
 * HeatmapRampMode — nature de la donnée encodée par la heatmap :
 *  - 'sequential' : intensité À CONNOTATION (taux de victoires, perf) — rampe
 *    cold→hot en vision standard, rampe fréquence (luminance monotone) en CVD.
 *  - 'divergent'  : indicateur signé (K/D vs 0) — rampe bas → neutre → haut.
 *  - 'frequency'  : intensité NEUTRE (nombre de rencontres, activité) — rampe
 *    fréquence mono-teinte dans toutes les palettes.
 */
export type HeatmapRampMode = 'sequential' | 'divergent' | 'frequency'

/** Rampe de fréquence : mono-teinte, monotone en luminance → CVD-safe partout. */
const FREQ_RAMP: readonly SemanticToken[] = ['heatmap-freq-low', 'heatmap-freq-high']

/** Rampe séquentielle « à connotation » (vision standard uniquement). */
const CONNOTATION_RAMP: readonly SemanticToken[] = ['heatmap-cold', 'heatmap-hot']

/** Rampe divergente signée (bas → neutre → haut). */
const DIVERGENT_RAMP: readonly SemanticToken[] = [
  'heatmap-divergent-low',
  'divergent-neutral',
  'heatmap-divergent-high',
]

/**
 * heatmapRampTokens — tokens sémantiques (ordonnés bas → haut) de la rampe
 * `visualMap.inRange.color` d'une heatmap. Pur et testable : ne résout aucun
 * hex (l'appelant fait `.map(resolveToken)`).
 */
export function heatmapRampTokens(
  mode: HeatmapRampMode,
  colorPalette: ColorPalette = 'default',
): SemanticToken[] {
  switch (mode) {
    case 'divergent':
      return [...DIVERGENT_RAMP]
    case 'frequency':
      return [...FREQ_RAMP]
    case 'sequential':
    default:
      // Palette CVD → rampe luminance-monotone ; sinon rampe cold→hot familière.
      return colorPalette === 'default' ? [...CONNOTATION_RAMP] : [...FREQ_RAMP]
  }
}
