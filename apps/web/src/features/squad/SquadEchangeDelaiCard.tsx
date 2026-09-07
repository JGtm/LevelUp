/**
 * SquadEchangeDelaiCard — « Délai d'échange » (onglet Dynamique).
 *
 * COMBIEN DE TEMPS met votre camp à venger une mort. Les cinq premières barres
 * couvrent la fenêtre d'échange (0-1 … 4-5 s, la borne de 5 s comprise) ; les deux
 * dernières sont HORS FENÊTRE et rendues en teinte atténuée : elles sont MONTRÉES et
 * n'entrent dans AUCUN taux.
 *
 * POURQUOI LES MONTRER. Une distribution qui s'arrêterait net à 5 s ne dirait pas si
 * la fenêtre coupe une population dense ou du vide — « 40 % de morts vengées » se lit
 * très différemment selon que les ripostes manquées arrivent à 5,2 s ou à 40 s.
 *
 * LA FENÊTRE EST MARQUÉE DANS LE PIED DE CARTE ET DANS LES ÉTIQUETTES DE BARRE
 * (suffixe « hors fenêtre »), pas par une markLine : le wrapper `HistogramChart`
 * n'expose pas de markLine, et en fabriquer une exigerait un second wrapper.
 *
 * Les intervalles sont PRÉ-BINNÉS par le serveur (ADR 0010) : ce composant ne
 * choisit aucune borne.
 */
import { useMemo } from 'react'

import { HistogramChart, type ChartPointHistogram } from '@/components/charts/HistogramChart'
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
  const buckets = echange.delais ?? []

  // Étiquette d'axe : les bornes sont en SECONDES, et une barre hors fenêtre le dit
  // en toutes lettres — une barre atténuée sans mot laisserait deviner.
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

  // ATTÉNUATION, pas seconde teinte : les barres hors fenêtre gardent la couleur de
  // série et perdent en OPACITÉ — un seul indice graphique.
  //
  // Le plan disait « barres hachurées » ; le liseré tireté qui traduisait ce mot n'était
  // pas visible (même couleur que le remplissage, sous la même opacité globale), et il a
  // été retiré le 2026-09-06 plutôt que maquillé. LE SECOND INDICE EST LE MOT : chaque
  // étiquette d'axe concernée porte le suffixe « hors fenêtre », et le pied de carte le
  // redit. Un indice qu'on lit vaut mieux qu'un indice qu'on prétend peindre.
  //
  // MESURÉ SUR PIÈCES le 2026-09-06 (revue W2) : AUCUN token sémantique du dépôt
  // n'est achromatique dans les quatre palettes. `divergent-neutral` — le candidat
  // évident — vaut #60A5FA (blue-400) dans la palette PAR DÉFAUT, soit un bleu PLUS
  // SOUTENU que la série (`chart-series-1` = blue-300) ; il n'est gris que sous
  // okabe-ito, cividis et tol-bright. Un token « neutre » aurait donc peint ces
  // barres en bleu appuyé pour la majorité des utilisateurs.
  const binAttenuated = useMemo(
    () => (point: ChartPointHistogram) =>
      buckets.find((x) => x.debut_ms / 1000 === point.binStart)?.hors_fenetre === true,
    [buckets],
  )

  const footer = (
    <div className="space-y-1 border-t border-border px-3 py-2">
      <p className="text-xs text-muted-foreground">{t.definition(secondes)}</p>
      <p className="text-xs text-muted-foreground">{t.delayWindow(secondes)}</p>
      <p className="text-xs text-muted-foreground" data-testid="squad-echange-delai-coverage">
        {t.coverage(echange.matchs_mesures, echange.matchs_total)}
      </p>
    </div>
  )

  return (
    <SectionCard title={t.delayTitle} label={t.delayLabel} footer={footer}>
      <div className="space-y-2 px-3 py-2" data-testid="squad-echange-delai">
        {/* La ligne narrative et l'état vide ne coexistent pas : dire deux fois
            « aucune riposte mesurée » l'un sous l'autre n'informe pas deux fois. */}
        {resume.total === 0 ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.delayNarrativeEmpty} />
        ) : (
          <>
            <p className="text-sm text-foreground" data-testid="squad-echange-delai-narrative">
              {t.delayNarrative(resume.dansLaFenetre, resume.horsFenetre, resume.total, secondes)}
            </p>
            <HistogramChart
              series={series}
              xAxisLabel={t.delayXAxis}
              yAxisLabel={t.delayYAxis}
              formatBin={formatBin}
              binAttenuated={binAttenuated}
            />
          </>
        )}
      </div>
    </SectionCard>
  )
}
