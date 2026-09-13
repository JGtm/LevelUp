/**
 * SquadAssistPairsChart — « Assistances dans l'escouade » (page Synergies).
 *
 * BARRES VERTICALES EMPILÉES, une par ASSISTANT, segments = les BÉNÉFICIAIRES qu'il a
 * servis. C'est la forme déjà retenue pour le même fait sur la page Match
 * (`MatchAssistChart`) : à question identique, forme identique — un tableau ici et un
 * graphe là obligeaient à réapprendre la lecture d'un écran à l'autre.
 *
 * Remplace `SquadAssistPairsTable` (retiré le 2026-09-13, décision utilisateur). Les deux
 * grandeurs du tableau survivent, dans l'infobulle : le COMPTE (hauteur du segment) ET LA
 * PART dans les assistances mesurées de l'escouade — « 3 assistances sur 6 n'est pas 3 sur
 * 300 » —, plus les éliminations volées quand il y en a.
 *
 * COULEURS PAR JOUEUR (`getSquadPlayerColors`) : un joueur garde la même teinte partout sur
 * la page. Un segment désigne donc le bénéficiaire par sa couleur, sans lecture de légende.
 *
 * LE BANDEAU DE COUVERTURE VIT AU-DESSUS DU GRAPHE, comme avant et pour la même raison :
 * l'assistance se lit dans le film du match, et les films Theater EXPIRENT côté serveur —
 * le manque est DÉFINITIF, pas un retard.
 */
import { useMemo } from 'react'

import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { intlLocale } from '@/lib/formatters'
import type { SquadAssistPairs } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getSquadPlayerColors } from './colors'
import { getSquadText } from './i18n'
import {
  assistBeneficiaires,
  assistCle,
  assistPairsSeries,
  assistPartParCouple,
  assistVoleesParCouple,
} from './squadAssistPairs.logic'

export interface SquadAssistPairsChartProps {
  block: SquadAssistPairs
  /** Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. */
  roster: string[]
}

export function SquadAssistPairsChart({ block, roster }: SquadAssistPairsChartProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const labels = t.assists
  const numLoc = intlLocale(locale)

  // `pairs` est un tableau NULLABLE au contrat (toute tranche Go sort ainsi) : comblé ici,
  // à la frontière, une seule fois.
  const pairs = useMemo(() => block.pairs ?? [], [block.pairs])

  const pctFmt = useMemo(
    () =>
      new Intl.NumberFormat(numLoc, {
        style: 'percent',
        minimumFractionDigits: 1,
        maximumFractionDigits: 1,
      }),
    [numLoc],
  )

  const series = useMemo(() => assistPairsSeries(pairs, roster), [pairs, roster])
  const beneficiaires = useMemo(() => assistBeneficiaires(pairs, roster), [pairs, roster])
  const volees = useMemo(() => assistVoleesParCouple(pairs), [pairs])
  const parts = useMemo(() => assistPartParCouple(block), [block])

  // Couleurs par joueur : le premier du roster est le joueur principal (pastille
  // `squad-player-1`), les suivants prennent les tokens coéquipiers dans l'ordre.
  const componentHexColors = useMemo(() => {
    const [main, ...teammates] = roster
    const parJoueur = getSquadPlayerColors(main ?? '', teammates)
    const out: Record<string, string> = {}
    for (const gt of beneficiaires) {
      const couleur = parJoueur[gt]
      if (couleur) out[gt] = couleur
    }
    return out
  }, [roster, beneficiaires])

  const tooltipComponentNote = useMemo(
    () => (assistant: string, beneficiaire: string) => {
      const cle = assistCle(assistant, beneficiaire)
      const notes: string[] = []
      const part = parts.get(cle)
      if (part != null) notes.push(labels.tooltipShare(pctFmt.format(part)))
      const volee = volees.get(cle)
      if (volee) notes.push(labels.tooltipStolen(volee))
      return notes.length > 0 ? notes.join(' · ') : undefined
    },
    [parts, volees, labels, pctFmt],
  )

  const coverage = (
    <p
      className="text-xs text-muted-foreground"
      data-testid="squad-assist-pairs-coverage"
      title={labels.coverageHint}
    >
      {labels.coverage(block.matches_measured, block.matches_total)}
    </p>
  )

  return (
    <div className="space-y-2" data-testid="squad-assist-pairs-chart">
      {/* Bandeau de couverture AU-DESSUS du graphe (doctrine de la page). */}
      {coverage}
      <BarStackedChart
        series={series}
        height={320}
        emptyMessage={labels.noPairs}
        componentOrder={beneficiaires}
        componentHexColors={componentHexColors}
        tooltipHideZero
        tooltipComponentNote={tooltipComponentNote}
      />
    </div>
  )
}
