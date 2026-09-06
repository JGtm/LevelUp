/**
 * SquadEchangeConstatCard — le « Constat du moment » de l'Escouade, en tête de Synergies.
 *
 * « CONSTAT », ET SURTOUT PAS « CAP » (décision utilisateur du 2026-09-06). La page
 * porte déjà un « Cap d'escouade » (`SquadFocusStrip`), à quelques centimètres au-dessus
 * des onglets, et l'onglet Entraînement un « Cap du moment » (`CoachFocusCard`) : ces
 * deux-là regardent DEVANT — objectifs de la composition, axe à renforcer. Celle-ci
 * regarde DERRIÈRE : elle constate ce que disent les matchs affichés, et ne propose
 * aucune direction. Trois « Cap » côte à côte, dont un rétrospectif, se marchaient dessus.
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
 * La règle vit dans `squadEchange.logic.constatDuMoment`, pure et testée : ce composant
 * n'a aucune décision à prendre.
 */
import { useMemo } from 'react'

import { KpiCard } from '@/components/cards/KpiCard'
import { formatPoints } from '@/lib/baseline'
import { intlLocale } from '@/lib/formatters'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { constatDuMoment } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export interface SquadEchangeConstatCardProps {
  echange: SquadEchange | null | undefined
}

export function SquadEchangeConstatCard({ echange }: SquadEchangeConstatCardProps) {
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

  const cap = constatDuMoment(echange)
  if (!cap) return null

  // MAGNITUDE, pas grandeur signée : la phrase porte déjà la direction (« de moins »
  // / « de plus »), et un « +10 pts de moins que d'habitude » se contredit tout seul.
  const delta = formatPoints(cap.ecart)
  const phrase =
    cap.ton === 'consolide'
      ? t.constatConsolidate(delta, pctFmt.format(cap.taux), pctFmt.format(cap.habituel))
      : t.constatAttention(delta, pctFmt.format(cap.taux), pctFmt.format(cap.habituel))

  return (
    <KpiCard accent="info" testId="squad-echange-constat">
      <div className="px-3 py-2">
        <p className="text-2xs uppercase tracking-wide text-muted-foreground">{t.constatTitle}</p>
        <p className="mt-0.5 text-sm font-semibold text-foreground" data-testid="squad-echange-constat-phrase">
          {phrase}
        </p>
        <p className="mt-0.5 text-xs text-muted-foreground">{t.constatBasis(cap.morts, cap.matchs)}</p>
      </div>
    </KpiCard>
  )
}
