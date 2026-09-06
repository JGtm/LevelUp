/**
 * SquadEchangeCapCard — le « Cap du moment » de l'Escouade, en tête de Synergies.
 *
 * UNE CARTE, DEUX CADRAGES, JAMAIS UNE ALERTE. Accent `info` dans les deux sens :
 * « vous consolidez » quand le camp échange plus que d'habitude, « X mérite votre
 * attention » quand il échange moins. Ni rouge ni `destructive` — un écart de
 * quelques points sur une sélection de matchs est une observation, pas une faute, et
 * la peindre en rouge transformerait un tableau de bord en reproche.
 *
 * DEUX SEUILS, TOUS LES DEUX NÉCESSAIRES (plan §1, arrêtés par l'utilisateur le
 * 2026-09-06) : au moins 30 morts d'équipe dans le périmètre ET au moins 5 points
 * d'écart au taux habituel. En dessous, la carte n'est PAS rendue — aucun état vide,
 * aucun « rien à signaler » : sous 30 morts l'écart mesuré est du tirage, et sous
 * 5 points il n'y a rien à dire.
 *
 * La règle vit dans `squadEchange.logic.capDuMoment`, pure et testée : ce composant
 * n'a aucune décision à prendre.
 */
import { useMemo } from 'react'

import { KpiCard } from '@/components/cards/KpiCard'
import { formatSignedPoints } from '@/lib/baseline'
import { intlLocale } from '@/lib/formatters'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { capDuMoment } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export interface SquadEchangeCapCardProps {
  echange: SquadEchange | null | undefined
}

export function SquadEchangeCapCard({ echange }: SquadEchangeCapCardProps) {
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

  const cap = capDuMoment(echange)
  if (!cap) return null

  const delta = formatSignedPoints(Math.abs(cap.ecart))
  const phrase =
    cap.ton === 'consolide'
      ? t.capConsolidate(delta, pctFmt.format(cap.taux), pctFmt.format(cap.habituel))
      : t.capAttention(delta, pctFmt.format(cap.taux), pctFmt.format(cap.habituel))

  return (
    <KpiCard accent="info" testId="squad-echange-cap">
      <div className="px-3 py-2">
        <p className="text-2xs uppercase tracking-wide text-muted-foreground">{t.capTitle}</p>
        <p className="mt-0.5 text-sm font-semibold text-foreground" data-testid="squad-echange-cap-phrase">
          {phrase}
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">{t.capBasis(cap.morts, cap.matchs)}</p>
      </div>
    </KpiCard>
  )
}
