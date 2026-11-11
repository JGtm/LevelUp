/**
 * cumulativeGapChartsVisibility — « Écart d'engagement cumulé » et « Balance des dégâts cumulée »
 * sont retirés de l'écran sans être supprimés du code.
 *
 * Interrupteur basculé à `false` le 2026-11-11 (le format de ces deux graphes ne convient pas et
 * doit être repensé) ; les composants, leurs tests et leurs textes restent en place pour la
 * refonte. Retrait de l'interrupteur : à la refonte, soit les graphes reviennent sous leur
 * nouveau format et la constante disparaît, soit ils sont supprimés avec leurs composants.
 * Un seul point de décision pour les trois pages (Escouade, Sessions, Séries temporelles).
 */
export const CUMULATIVE_GAP_CHARTS_SHOWN = false
