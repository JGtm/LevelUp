/**
 * SquadEchangeDonneRecuCard — « Donné et reçu, par coéquipier » (onglet Synergies).
 *
 * LA MÊME DONNÉE QUE LA MATRICE, EN PLUS GROSSIER : la somme de la LIGNE (donné) et celle
 * de la COLONNE (reçu) de chaque joueur. Aucune mesure nouvelle, aucun appel serveur —
 * `donneRecuParJoueur` dérive les deux sommes des `cellules` déjà chargées.
 *
 * POURQUOI GARDER LES DEUX CARTES. La matrice dit QUI couvre QUI, couple par couple ; celle
 * -ci dit seulement combien chacun donne et reçoit, et c'est justement ce qui se lit d'un
 * coup d'œil quand le roster grandit et que la grille devient dense.
 *
 * Deux séries GROUPÉES (pas empilées) : « donné » et « reçu » sont deux grandeurs du même
 * joueur, pas deux parts d'un tout — les empiler ferait lire une somme qui n'existe pas.
 */
import { useMemo } from 'react'

import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { donneRecuParJoueur, donneRecuSeries, matriceVide } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export interface SquadEchangeDonneRecuCardProps {
  echange: SquadEchange
}

export function SquadEchangeDonneRecuCard({ echange }: SquadEchangeDonneRecuCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadEchangeText(locale)

  const rows = useMemo(() => donneRecuParJoueur(echange), [echange])
  const series = useMemo(
    () => donneRecuSeries(echange, t.donneRecuGiven, t.donneRecuReceived),
    [echange, t],
  )

  // La phrase nomme le DÉSÉQUILIBRE le plus marqué du roster, quand il y en a un : celui
  // qui reçoit le plus par rapport à ce qu'il donne. À l'équilibre, elle le dit aussi —
  // une carte muette laisserait croire à une donnée manquante.
  const dominant = useMemo(() => {
    if (rows.length === 0) return null
    const top = rows.reduce((a, b) => (b.recu - b.donne > a.recu - a.donne ? b : a))
    return top.recu > top.donne ? top : null
  }, [rows])

  return (
    <SectionCard title={t.donneRecuTitle} label={t.donneRecuLabel}>
      <div className="space-y-2 px-3 py-2" data-testid="squad-echange-donne-recu">
        {matriceVide(echange) ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.noPairs} />
        ) : (
          <>
            <p className="border-l-2 border-info pl-3 text-sm text-foreground">
              {dominant
                ? t.donneRecuSay({
                    joueur: dominant.gamertag,
                    recu: dominant.recu,
                    donne: dominant.donne,
                  })
                : t.donneRecuSayEquilibre}
            </p>
            <BarGroupedChart series={series} height={280} showValues />
          </>
        )}
      </div>
    </SectionCard>
  )
}
