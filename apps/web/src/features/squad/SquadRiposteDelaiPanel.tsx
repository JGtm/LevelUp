/**
 * SquadRiposteDelaiPanel — repli « Combien de temps on met » de la carte « Riposte ».
 *
 * COMBIEN DE TEMPS met notre camp à riposter. Les cinq premières barres couvrent la
 * fenêtre de riposte (0-1 … 4-5 s, la borne de 5 s comprise) ; les deux dernières sont HORS
 * FENÊTRE : elles sont MONTRÉES, HACHURÉES, et n'entrent dans AUCUN taux.
 *
 * POURQUOI LES MONTRER. Une distribution qui s'arrêterait net à 5 s ne dirait pas si la
 * fenêtre coupe une population dense ou du vide — « 19 % de morts ripostées » se lit très
 * différemment selon que les ripostes manquées arrivent à 5,2 s ou à 40 s.
 *
 * TROIS INDICES POUR LA MÊME CHOSE, et c'est voulu : la HACHURE sur la barre, le REPÈRE
 * vertical tireté « fenêtre N s » posé sur la borne, et le MOT (« hors fenêtre » en
 * étiquette d'axe, la note de pied). Un seuil qui décide d'un taux ne peut pas se deviner.
 *
 * CE N'EST PLUS UNE CARTE (D19, 2026-09-21) : c'était une SectionCard autonome, la
 * troisième de huit à énoncer la même mesure. Elle est devenue le DÉTAIL du chiffre
 * « 2,4 s de délai médian », replié sous la carte « Riposte » — une SectionCard dans une
 * SectionCard n'existe pas, ce composant ne pose donc plus aucun chrome.
 *
 * Les intervalles sont PRÉ-BINNÉS par le serveur (ADR 0010) : ce composant ne choisit
 * aucune borne.
 */
import { useMemo } from 'react'

import { HistogramChart, type ChartPointHistogram } from '@/components/charts/HistogramChart'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { delaisSeries, resumeDelais } from './squadRiposte.logic'
import { getSquadRiposteText } from './squadRiposteStrings'

export interface SquadRiposteDelaiPanelProps {
  echange: SquadEchange
}

export function SquadRiposteDelaiPanel({ echange }: SquadRiposteDelaiPanelProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadRiposteText(locale)

  const secondes = echange.fenetre_ms / 1000
  const series = useMemo(() => delaisSeries(echange), [echange])
  const resume = useMemo(() => resumeDelais(echange), [echange])
  const buckets = useMemo(() => echange.delais ?? [], [echange.delais])

  // Étiquette d'axe : les bornes sont en SECONDES, et une barre hors fenêtre le dit en
  // toutes lettres — une barre hachurée sans mot laisserait deviner.
  const formatBin = useMemo(
    () => (point: ChartPointHistogram) => {
      const b = buckets.find((x) => x.debut_ms / 1000 === point.binStart)
      const base = b?.ouvert
        ? t.delayBinOpen(point.binStart)
        : t.delayBin(point.binStart, point.binEnd)
      return b?.hors_fenetre ? `${base} (${t.delayOutOfWindowSuffix})` : base
    },
    [buckets, t],
  )

  const binHatched = useMemo(
    () => (point: ChartPointHistogram) =>
      buckets.find((x) => x.debut_ms / 1000 === point.binStart)?.hors_fenetre === true,
    [buckets],
  )

  // Le repère se pose sur la borne : le PREMIER intervalle hors fenêtre en marque le début.
  const windowMark = useMemo(() => {
    const index = buckets.findIndex((b) => b.hors_fenetre)
    return index > 0 ? { binIndex: index, label: t.delayWindowMark(secondes) } : undefined
  }, [buckets, t, secondes])

  // L'intervalle le plus peuplé DANS la fenêtre : c'est le « pic » que la phrase nomme.
  const pic = useMemo(() => {
    const dedans = buckets.filter((b) => !b.hors_fenetre)
    if (dedans.length === 0) return ''
    const top = dedans.reduce((a, b) => (b.nombre > a.nombre ? b : a))
    return top.ouvert
      ? t.delayBinOpen(top.debut_ms / 1000)
      : t.delayBin(top.debut_ms / 1000, top.fin_ms / 1000)
  }, [buckets, t])

  if (resume.total === 0) {
    return <EmptyStateNotice title={t.emptyTitle} description={t.delayNarrativeEmpty} />
  }

  return (
    <div className="space-y-2" data-testid="squad-riposte-delai">
      {/* La ligne narrative vit AU-DESSUS du graphe, jamais en dessous. */}
      <p className="border-l-2 border-info pl-3 text-sm text-foreground">
        {t.delaySay({
          morts: echange.couverture.n,
          dedans: resume.dansLaFenetre,
          dehors: resume.horsFenetre,
          pic,
        })}
      </p>
      <HistogramChart
        series={series}
        xAxisLabel={t.delayXAxis}
        yAxisLabel={t.delayYAxis}
        formatBin={formatBin}
        binHatched={binHatched}
        showValues
        windowMark={windowMark}
        frameless
      />
      <p className="text-2xs text-muted-foreground">
        {t.delayFigure(resume.total, echange.couverture.n)} · {t.delayWindow(secondes)}{' '}
        {t.delayFoot(secondes)}
      </p>
    </div>
  )
}
