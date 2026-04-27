/**
 * Table de mapping section UI -> mode CapabilityGap.
 *
 * Conformément au PLAN_META_FOUNDATIONS_GO § 3.4.5 : centraliser ici les
 * décisions UX pour qu'elles soient revues par l'équipe produit avant d'être
 * réparties dans les composants.
 *
 * Une section UI peut être :
 *   - hide        : disparait quand la capability est absente
 *   - placeholder : affiche un message statique sans action
 *   - cta         : affiche un bouton qui mène à la résolution (sync, doc)
 *
 * Les clés sont des identifiants de section stables, format
 * `<page>.<section>` en snake_case. Les pages qui n'ont pas d'entrée ici
 * tombent par défaut sur `placeholder` (sécuritaire).
 */
import type { CapabilityGapMode } from '@/components/feedback/CapabilityGap'

/**
 * Mapping section -> mode. À étendre au fil des chunks Phase 1+ qui adoptent
 * le composant CapabilityGap.
 *
 * Le format choisi (constante explicite) plutôt qu'un Record dynamique
 * permet l'autocomplétion stricte côté composants consommateurs.
 */
export const SECTION_GAP_MODES = {
  // Squad (Phase 1 pilote)
  'squad.impact_heatmap': 'placeholder', // sans events filmés -> juste un message
  'squad.cadence': 'placeholder',
  'squad.intensity': 'placeholder',
  'squad.medals_gallery': 'hide', // sans sync médailles -> on cache
  'squad.weapons_table': 'placeholder',

  // Match View (Phase 1 pilote)
  'match_view.kill_feed': 'placeholder',
  'match_view.cadence': 'placeholder',
  'match_view.dominance_badge': 'hide', // si flag absent : on n'affiche rien
  'match_view.radar_participation': 'cta', // demande sync personal_score_awards
  'match_view.encounters': 'placeholder',

  // Career
  'career.lusr_chart': 'placeholder',
  'career.top_matches': 'placeholder',
  'career.encounters': 'placeholder',

  // Timeseries
  'timeseries.first_events_rolling': 'placeholder',
  'timeseries.intensity_heatmap': 'placeholder',
  'timeseries.cadence': 'placeholder',

  // Citations
  'citations.medals_distribution': 'placeholder',

  // Synthesis
  'synthesis.heatmap_winrate': 'placeholder',

  // Home (page live, gap rare)
  'home.battlepass': 'cta', // si titre n'a pas matchmaking : explication
} as const satisfies Record<string, CapabilityGapMode>

/**
 * SectionGapKey est l'union typée des clés ci-dessus. Les composants
 * consommateurs doivent accepter ce type au lieu d'un `string` libre pour
 * bénéficier de la garde de cohérence.
 */
export type SectionGapKey = keyof typeof SECTION_GAP_MODES

/**
 * resolveGapMode retourne le mode défini pour une section, ou `placeholder`
 * par défaut si la clé n'est pas répertoriée. Le défaut sécuritaire évite
 * qu'une section disparaisse silencieusement (mode `hide`) avant d'avoir
 * été revue UX.
 */
export function resolveGapMode(key: string): CapabilityGapMode {
  if (key in SECTION_GAP_MODES) {
    return SECTION_GAP_MODES[key as SectionGapKey]
  }
  return 'placeholder'
}
