/**
 * SquadEchangeCompteCard — « Le compte » (onglet Synergies, en tête).
 *
 * QUATRE GRANDEURS ENSEMBLE, JAMAIS LE TAUX SEUL. Le taux d'échange se calcule sur les
 * morts de votre camp : il monte quand vous mourez moins, sans qu'aucune vengeance de plus
 * n'ait eu lieu. Il ne sort donc jamais sans le DÉLAI MÉDIAN (quand l'échange a lieu), le
 * COMPTE BRUT des morts sans réponse, et ces mêmes morts RAMENÉES AU MATCH — les deux
 * dernières ne dépendent pas de combien vous mourez.
 *
 * L'ÉCART SE TAIT QUAND LE PÉRIMÈTRE COUVRE TOUT L'HISTORIQUE. Le périmètre filtré est
 * toujours un sous-ensemble de la référence : à cardinalités égales les deux ensembles sont
 * identiques, l'écart est nul par construction, et « ±0 pts vs habituel » ferait croire à
 * une mesure là où il n'y a qu'une tautologie (`isFullHistoryScope`, @/lib/baseline).
 *
 * Remplace l'ancienne tuile unique du bandeau de l'Escouade (`SquadEchangeKpi`, retirée le
 * 2026-09-13) : une tuile de taux seule, posée au-dessus des onglets, disait exactement ce
 * que la doctrine de la page interdit de dire seul.
 */
import { useMemo } from 'react'

import { KPIStrip, type KPICardData } from '@/components/layout/KPIStrip'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
import { formatSignedPoints } from '@/lib/baseline'
import { intlLocale } from '@/lib/formatters'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { compteEchange, trendEcart } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export interface SquadEchangeCompteCardProps {
  echange: SquadEchange
}

export function SquadEchangeCompteCard({ echange }: SquadEchangeCompteCardProps) {
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
  const unFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )
  const entierFmt = useMemo(() => new Intl.NumberFormat(numLoc), [numLoc])
  const deuxFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    [numLoc],
  )

  const compte = useMemo(() => compteEchange(echange), [echange])
  const secondes = echange.fenetre_ms / 1000

  // La réserve d'échantillon faible s'affiche AVEC la valeur : elle ne la cache pas, elle
  // interdit de la comparer (forme unique du dépôt : `withLowSampleNote`).
  const sousTitreTaux = withLowSampleNote(
    t.kpiRateSub(secondes),
    compte.echantillonFaible,
    t.lowSample,
  )

  const cards: KPICardData[] = [
    {
      id: 'squad-echange-taux',
      label: t.kpiLabel,
      primary: pctFmt.format(compte.taux),
      secondary: sousTitreTaux,
      trend: compte.pleinHistorique ? 'none' : trendEcart(compte.ecartPoints),
      custom: compte.pleinHistorique ? undefined : (
        <span className="text-2xs text-muted-foreground" data-testid="squad-echange-kpi-delta">
          {t.kpiVsUsual(formatSignedPoints(compte.ecart))}
        </span>
      ),
    },
    {
      id: 'squad-echange-delai-median',
      label: t.kpiDelayLabel,
      primary:
        compte.delaiMedianS === null
          ? t.kpiNoValue
          : t.kpiDelayValue(unFmt.format(compte.delaiMedianS)),
      secondary: t.kpiDelaySub,
    },
    {
      id: 'squad-echange-sans-reponse',
      label: t.kpiUnansweredLabel,
      primary: entierFmt.format(compte.sansReponse),
      secondary: t.kpiUnansweredSub(compte.mortsEquipe),
    },
    {
      id: 'squad-echange-sans-reponse-par-match',
      label: t.kpiUnansweredRateLabel,
      primary:
        compte.sansReponseParMatch === null
          ? t.kpiNoValue
          : deuxFmt.format(compte.sansReponseParMatch),
      secondary: t.kpiUnansweredRateSub,
    },
  ]

  return (
    <SectionCard
      title={t.compteTitle}
      label={t.compteLabel}
      // Le pied de carte passe en infobulle ⓘ du titre (2026-09-21) : c'est de la méthode,
      // elle n'a pas à occuper une bande grise sous les chiffres à chaque visite.
      titleAdornment={(label) => (
        <span className="flex items-center gap-1.5">
          {label}
          <InfoTooltip content={t.compteFoot} />
        </span>
      )}
    >
      <div className="space-y-3 px-3 py-3" data-testid="squad-echange-compte">
        {/* La ligne narrative vit AU-DESSUS des chiffres, jamais en dessous. */}
        <p className="border-l-2 border-info pl-3 text-sm text-foreground">{t.compteSay}</p>
        <KPIStrip cards={cards} />
      </div>
    </SectionCard>
  )
}
