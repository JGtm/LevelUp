/**
 * skillTierBands — helpers partagés pour cadrer un axe Y de classement (LUSR/CSR)
 * sur les bandes de palier et dessiner ces bandes (markArea ECharts).
 *
 * Consommé par le graphe « Classement » par-match (features/timeseries) et le
 * graphe « Évolution LUSR / CSR » de la carrière (features/career), pour éviter
 * les axes partant de 0 (CSR < 1200) et l'expansion d'échelle par la bande Onyx
 * (yAxis 9999).
 */
import { resolveToken } from '@/lib/accessibility'
import { LUSR_TIERS } from '@/lib/skillTiers'
import type { ManifestLocale } from '@/lib/i18n/format'

/**
 * Cadre [min, max] de l'axe Y sur la/les bande(s) de palier LUSR contenant les
 * données. Pose un min ET un max explicites : (1) révèle les petites variances
 * (vs un axe partant de 0), (2) empêche la bande Onyx (yAxis 9999) de tirer
 * l'échelle vers le haut. Hors plage des paliers (CSR < 1200, ou palier Onyx
 * ouvert), arrondit à un pas propre (multiples de 100) autour des données.
 */
export function frameToTier(dataMin: number, dataMax: number): { min: number; max: number } {
  const STEP = 100
  const lo = LUSR_TIERS.find(t => dataMin >= t.min && dataMin < t.max)
  const hi = LUSR_TIERS.find(t => dataMax >= t.min && dataMax < t.max)
  let min = lo ? lo.min : Math.floor(dataMin / STEP) * STEP
  // Le palier Onyx (max 9999) est ouvert vers le haut → ne jamais l'utiliser comme plafond.
  let max = hi && hi.max < 9000 ? hi.max : Math.ceil(dataMax / STEP) * STEP
  // Garde-fou anti-dégénérescence (données plates ou pile sur un multiple de STEP).
  if (max - min < STEP) { min -= STEP; max += STEP }
  return { min: Math.max(0, min), max }
}

/**
 * Bandes de palier LUSR en markArea ECharts, clippées à [yMin, yMax] pour que le
 * label de palier reste visible même quand l'axe est zoomé à l'intérieur d'une bande.
 */
export function buildLusrTierMarkArea(locale: ManifestLocale, yMin: number, yMax: number) {
  return {
    silent: true,
    label: { show: true, position: 'insideTopLeft' as const, fontSize: 10, opacity: 0.6 },
    data: LUSR_TIERS
      .filter(tier => tier.max > yMin && tier.min < yMax)
      .map(tier => [
        {
          yAxis: Math.max(tier.min, yMin),
          name: locale === 'fr' ? tier.fr : tier.en,
          itemStyle: { color: resolveToken(tier.token) + '30' },
          label: { color: resolveToken(tier.token) },
        },
        { yAxis: Math.min(tier.max, yMax) },
      ]),
  }
}
