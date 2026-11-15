/**
 * EfficiencyTooltipText — texte explicatif UNIQUE « Rendement / Résistance ».
 *
 * Partagé par les blocs d'efficacité de Timeseries, Sessions et Escouade : un
 * seul libellé (clé i18n `common.charts.efficiency_tooltip`) pour le même
 * concept, au lieu de trois variantes divergentes. Chaque graphe affiche en plus
 * sa propre ligne de référence (P80, 1 vie…) qui en précise l'échelle.
 */
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { useEffectiveHpToKill, substituteHpToken } from '@/lib/damage/effectiveHp'

export function EfficiencyTooltipText({ locale }: { locale: ManifestLocale }) {
  // Barème PV-pour-tuer du titre (« ≈ {{HP}} points »), lu du bootstrap : rien tant qu'il n'est pas chargé.
  const hp = useEffectiveHpToKill()
  if (hp === undefined) return null
  return <>{substituteHpToken(formatMessage(commonManifest, 'common.charts.efficiency_tooltip', locale), hp)}</>
}
