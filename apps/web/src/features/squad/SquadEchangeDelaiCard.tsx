/**
 * SquadEchangeDelaiCard — « Combien, et à quelle vitesse » (onglet Synergies).
 *
 * COMBIEN DE TEMPS met votre camp à venger une mort. Les cinq premières barres couvrent la
 * fenêtre d'échange (0-1 … 4-5 s, la borne de 5 s comprise) ; les deux dernières sont HORS
 * FENÊTRE : elles sont MONTRÉES, HACHURÉES, et n'entrent dans AUCUN taux.
 *
 * POURQUOI LES MONTRER. Une distribution qui s'arrêterait net à 5 s ne dirait pas si la
 * fenêtre coupe une population dense ou du vide — « 19 % de morts vengées » se lit très
 * différemment selon que les ripostes manquées arrivent à 5,2 s ou à 40 s.
 *
 * TROIS INDICES POUR LA MÊME CHOSE, et c'est voulu : la HACHURE sur la barre, le REPÈRE
 * vertical tireté « fenêtre N s » posé sur la borne, et le MOT (« hors fenêtre » en
 * étiquette d'axe, la note de pied). Un seuil qui décide d'un taux ne peut pas se deviner.
 *
 * DÉPLACÉE DE « DYNAMIQUE » VERS « SYNERGIES » le 2026-09-13 (maquette 4c520da6) : elle est
 * la deuxième carte du récit de l'échange — d'abord le compte, puis la vitesse, puis qui.
 *
 * Les intervalles sont PRÉ-BINNÉS par le serveur (ADR 0010) : ce composant ne choisit
 * aucune borne.
 */
import { useMemo } from 'react'

import { HistogramChart, type ChartPointHistogram } from '@/components/charts/HistogramChart'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { delaisSeries, resumeDelais } from './squadEchange.logic'
import { getSquadEchangeText } from './squadEchangeStrings'

export interface SquadEchangeDelaiCardProps {
  echange: SquadEchange
}

export function SquadEchangeDelaiCard({ echange }: SquadEchangeDelaiCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadEchangeText(locale)

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

  const help = (
    <span className="space-y-1.5">
      <span className="block">{t.definition(secondes)}</span>
      <span className="block">{t.delayWindow(secondes)}</span>
    </span>
  )

  const footer = (
    <div className="border-t border-border px-3 py-2">
      <p className="text-xs text-muted-foreground">{t.delayFoot(secondes)}</p>
    </div>
  )

  return (
    <SectionCard
      title={t.delayTitle}
      label={t.delayLabel}
      titleAdornment={(label) => (
        <span className="flex items-center gap-1.5">
          {label}
          <InfoTooltip content={help} />
        </span>
      )}
      footer={footer}
    >
      <div className="space-y-2 px-3 py-2" data-testid="squad-echange-delai">
        {resume.total === 0 ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.delayNarrativeEmpty} />
        ) : (
          <>
            {/* La ligne narrative vit AU-DESSUS du graphe, jamais en dessous. */}
            <p className="border-l-2 border-info pl-3 text-sm text-foreground">
              {t.delaySay({
                morts: echange.couverture.n,
                dedans: resume.dansLaFenetre,
                dehors: resume.horsFenetre,
                pic,
              })}
            </p>
            <p className="text-xs text-muted-foreground">
              {t.delayFigure(resume.total, echange.couverture.n)}
            </p>
            <HistogramChart
              series={series}
              xAxisLabel={t.delayXAxis}
              yAxisLabel={t.delayYAxis}
              formatBin={formatBin}
              binHatched={binHatched}
              showValues
              windowMark={windowMark}
            />
          </>
        )}
      </div>
    </SectionCard>
  )
}
