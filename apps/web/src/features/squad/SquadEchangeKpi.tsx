/**
 * SquadEchangeKpi — la tuile « Taux d'échange » du bandeau de KPI de l'Escouade.
 *
 * TROIS GRANDEURS ENSEMBLE, jamais le taux seul (doctrine `SquadAssistPairsTable`) :
 * la valeur, le compte brut (« N vengées sur M »), et l'écart au taux HABITUEL en
 * points signés avec sa flèche.
 *
 * L'ÉCART SE TAIT QUAND LE PÉRIMÈTRE COUVRE TOUT L'HISTORIQUE. Le périmètre filtré
 * est toujours un sous-ensemble de la référence : à cardinalités égales les deux
 * ensembles sont identiques, l'écart est nul par construction, et « ±0 pts vs
 * habituel » ferait croire à une mesure là où il n'y a qu'une tautologie
 * (`isFullHistoryScope`, @/lib/baseline).
 *
 * Rien n'est rendu quand la section est absente du contrat : un titre qui ne nomme
 * pas le tueur de chaque mort n'a pas un taux nul, il n'a pas de taux.
 */
import { useMemo } from 'react'

import { KPIStrip, type KPICardData } from '@/components/layout/KPIStrip'
import { formatSignedPoints, isFullHistoryScope } from '@/lib/baseline'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

import { useSquadContext } from './SquadContext'
import { trendEcart } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export function SquadEchangeKpi() {
  const { pageData } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadEchangeText(locale)
  const numLoc = intlLocale(locale)

  const pctFmt = useMemo(
    () =>
      new Intl.NumberFormat(numLoc, {
        style: 'percent',
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [numLoc],
  )

  const echange = pageData?.echange
  if (!echange) return null

  const pleinHistorique = isFullHistoryScope(echange.matchs_total, echange.matchs_habituel)
  const ecart = echange.couverture.taux - echange.habituel.taux
  // La réserve d'échantillon faible s'affiche AVEC la valeur : elle ne la cache pas,
  // elle interdit de la comparer.
  const secondaire = echange.couverture.echantillon_faible
    ? `${t.kpiSecondary(echange.couverture.brut, echange.couverture.n)} — ${t.lowSample}`
    : t.kpiSecondary(echange.couverture.brut, echange.couverture.n)

  const card: KPICardData = {
    id: 'squad-echange',
    label: t.kpiLabel,
    primary: pctFmt.format(echange.couverture.taux),
    secondary: secondaire,
    trend: pleinHistorique ? 'none' : trendEcart(Math.round(ecart * 100)),
    custom: pleinHistorique ? undefined : (
      <span className="text-2xs text-muted-foreground" data-testid="squad-echange-kpi-delta">
        {t.kpiVsUsual(formatSignedPoints(ecart))}
      </span>
    ),
  }
  return <KPIStrip cards={[card]} />
}
