/**
 * SquadEchangeMatrixCard — « Qui échange pour qui » (onglet Synergies).
 *
 * Un ÉCHANGE est une mort vengée : un coéquipier abat le tueur dans les 5 s.
 * LIGNE = celui qui venge, COLONNE = celui qui est vengé — la même orientation que
 * `SquadAssistPairsTable` (Assistant / Bénéficiaire), son voisin immédiat.
 *
 * LE BANDEAU DE COUVERTURE VIT AU-DESSUS DU GRAPHE, pas en note de bas de page, et
 * pour la raison exacte du tableau d'assistance : le journal des morts vient du film
 * du match, et les films Theater EXPIRENT côté serveur. Le manque est DÉFINITIF, pas
 * un retard, et un chiffre calculé sur une fraction de la sélection sans dire
 * laquelle ne serait pas reproductible.
 *
 * PALETTE : rampe de FRÉQUENCE, mono-teinte (`heatmapRampTokens('frequency')` via
 * `paletteMode="frequency"`). Un nombre de vengeances n'est ni chaud ni froid — la
 * rampe cold→hot lui collerait un jugement, et les couleurs PAR JOUEUR
 * (`squad-player-*`) feraient croire que la teinte désigne quelqu'un.
 *
 * La carte n'est pas montée quand la section est absente du contrat (cf.
 * SquadSynergiesPage) : une section omise n'est pas une section à zéro.
 */
import { useMemo } from 'react'

import { Heatmap2DChart } from '@/components/charts/Heatmap2DChart'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenVar } from '@/lib/accessibility'
import { intlLocale } from '@/lib/formatters'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { extremesCouverture, matriceSeries, matriceVide, PLANCHER_MORTS } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export interface SquadEchangeMatrixCardProps {
  echange: SquadEchange
}

export function SquadEchangeMatrixCard({ echange }: SquadEchangeMatrixCardProps) {
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
  const perMatchFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    [numLoc],
  )

  const secondes = echange.fenetre_ms / 1000
  const series = useMemo(() => matriceSeries(echange), [echange])
  const extremes = useMemo(() => extremesCouverture(echange), [echange])
  const vide = matriceVide(echange)

  // Tooltip PROPRE à cette lecture : le libellé par défaut du wrapper parle de taux
  // de victoire et de matchs, ce qu'une case de cette matrice n'est pas.
  const formatTooltip = useMemo(
    () => (p: { x: string; y: string; value: number; detail?: Record<string, unknown> }) =>
      t.matrixTooltip(p.y, p.x, p.value, perMatchFmt.format(Number(p.detail?.perMatch ?? 0))),
    [t, perMatchFmt],
  )

  // Le CONSTAT EN MOTS, au-dessus du graphe : « sur N matchs, X des Y morts de votre
  // camp ont été vengées dans les 5 s ». Un lecteur doit pouvoir repartir avec la
  // phrase sans lire la grille.
  const narrative = t.narrative({
    matches: echange.matchs_total,
    brut: echange.couverture.brut,
    n: echange.couverture.n,
    seconds: secondes,
    rate: pctFmt.format(echange.couverture.taux),
  })

  const footer = (
    <div className="space-y-1 border-t border-border px-3 py-2">
      <p className="text-xs text-muted-foreground">{t.definition(secondes)}</p>
      {echange.couverture.echantillon_faible && (
        <p className="text-xs text-muted-foreground" data-testid="squad-echange-low-sample">
          {t.lowSample} — {t.lowSampleHint(PLANCHER_MORTS)}
        </p>
      )}
    </div>
  )

  return (
    <SectionCard title={t.sectionTitle} label={t.sectionLabel} footer={footer}>
      <div className="space-y-2 px-3 py-2" data-testid="squad-echange-matrix">
        {/* Bandeau de couverture AU-DESSUS du graphe (doctrine SquadAssistPairsTable). */}
        <p
          className="text-xs text-muted-foreground"
          data-testid="squad-echange-coverage"
          title={t.coverageHint}
        >
          {t.coverage(echange.matchs_mesures, echange.matchs_total)}
        </p>
        <p className="text-sm text-foreground" data-testid="squad-echange-narrative">
          {narrative}
        </p>
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.noPairs} />
        ) : (
          <>
            <Heatmap2DChart
              series={series}
              paletteMode="frequency"
              formatTooltip={formatTooltip}
            />
            {/* Axes nommés : sans eux, « ligne » et « colonne » sont deux gamertags
                et rien ne dit lequel venge l'autre. */}
            <p className="text-2xs uppercase tracking-wide text-muted-foreground">
              {t.axisAvenger} × {t.axisAvenged}
            </p>
            {extremes && (
              <div className="flex flex-wrap gap-2" data-testid="squad-echange-badges">
                <NarrativeBadge
                  size="sm"
                  colorVar={tokenVar('outcome-win')}
                  label={t.badgeMostCovered(extremes.plusCouvert.gamertag)}
                  detailSuffix={t.badgeCoveredDetail(extremes.plusCouvert.vengeances)}
                />
                <NarrativeBadge
                  size="sm"
                  colorVar={tokenVar('info')}
                  label={t.badgeLeastCovered(extremes.moinsCouvert.gamertag)}
                  detailSuffix={t.badgeCoveredDetail(extremes.moinsCouvert.vengeances)}
                />
              </div>
            )}
          </>
        )}
      </div>
    </SectionCard>
  )
}
