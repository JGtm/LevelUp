/**
 * FdaGapTooltipText — texte explicatif UNIQUE « Écart cumulé au FDA attendu ».
 *
 * Partagé par les deux instances du graphe (Séries temporelles et Sessions) :
 * un seul libellé (clé i18n `common.charts.fda_gap_tooltip`) pour le même
 * concept, sur le patron de `EfficiencyTooltipText`. Le graphe n'avait aucune
 * aide jusqu'ici : le texte explique en langage courant ce que l'écart mesure
 * et comment le lire (la pente d'abord, la ligne 0 ensuite).
 */
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'

export function FdaGapTooltipText({ locale }: { locale: ManifestLocale }) {
  return <>{formatMessage(commonManifest, 'common.charts.fda_gap_tooltip', locale)}</>
}
