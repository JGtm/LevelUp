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
 *  - 'intensity'  : intensité NEUTRE À TROIS POINTS, dont le dernier est un
 *    EXTRÊME RARE (carte de chaleur du rejeu 2D). La rampe de fréquence est
 *    mono-teinte : elle dit « plus ou moins », jamais « et là, beaucoup plus ».
 *    Celle-ci change de teinte deux fois, ce qui rend visible le haut de
 *    l'échelle — c'est la demande du 2026-08-18 (« bleu -> rouge -> violet aux
 *    extrêmes rares »). Elle n'est PAS mono-teinte, donc pas monotone en
 *    luminance : elle s'emploie là où le nombre de paliers est grand (64) et où
 *    l'OPACITÉ monte avec l'intensité, ce qui rétablit l'ordre perceptuel.
 */
export type HeatmapRampMode = 'sequential' | 'divergent' | 'frequency' | 'intensity'

/** Rampe de fréquence : mono-teinte, monotone en luminance → CVD-safe partout. */
const FREQ_RAMP: readonly SemanticToken[] = ['heatmap-freq-low', 'heatmap-freq-high']

/** Rampe séquentielle « à connotation » (vision standard uniquement). */
const CONNOTATION_RAMP: readonly SemanticToken[] = ['heatmap-cold', 'heatmap-hot']

/**
 * Rampe d'INTENSITÉ à trois points : bleu (peu) → rouge (beaucoup) → violet (extrême rare).
 *
 * TROIS TOKENS DÉJÀ SÉMANTIQUES, et c'est délibéré : chaque palette d'accessibilité les
 * remappe déjà pour rester distinguables entre eux (sous Okabe-Ito, la rampe devient Sky
 * Blue → Vermillion → Reddish Purple, trois couleurs de la même référence CUD). Inventer
 * trois nouveaux tokens aurait dupliqué ce travail sans rien ajouter.
 */
const INTENSITY_RAMP: readonly SemanticToken[] = ['info', 'destructive', 'extreme']

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
    case 'intensity':
      return [...INTENSITY_RAMP]
    case 'sequential':
    default:
      // Palette CVD → rampe luminance-monotone ; sinon rampe cold→hot familière.
      return colorPalette === 'default' ? [...CONNOTATION_RAMP] : [...FREQ_RAMP]
  }
}
