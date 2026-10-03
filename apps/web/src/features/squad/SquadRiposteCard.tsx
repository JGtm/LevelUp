/**
 * SquadRiposteCard — la RIPOSTE de la section Coordination, en TROIS BLOCS.
 *
 * ELLE ABSORBE SIX BLOCS (D19, variante A de la maquette du 2026-09-21). L'onglet Synergies
 * montait huit SectionCard de même poids visuel dont SEPT portaient la même notion : le
 * taux y était énoncé trois fois (en prose, en tuile, en ligne d'habituel), le délai deux
 * fois, le détail par joueur deux fois. Le vocabulaire changeait à chaque carte — échange,
 * vengeance, assistance croisée — sans que la mesure change.
 *
 * PLUS UNE CARTE UNIQUE À QUATRE ÉTAGES (décision utilisateur du 2026-09-22). Les deux
 * graphes de tête vivaient DANS la carte « Riposte », sans cadre ni titre propre : deux
 * mesures distinctes — une PART et une DISTRIBUTION — partageaient un seul bandeau, une
 * seule infobulle, et leurs titres n'étaient que des libellés de graphe. Ce composant rend
 * désormais un FRAGMENT de trois SectionCard de plein droit :
 *
 *   1. « Morts ripostées » — le donut des deux parts exclusives, le taux au centre ;
 *   2. « Temps de riposte » — la distribution du délai (intervalles pré-binnés, barres hors
 *      fenêtre hachurées, repère de fenêtre, MÉDIANE à sa vraie place entre deux barres) ;
 *      ces deux-là côte à côte sur UNE RANGÉE de deux colonnes ;
 *   3. « Riposte » — pleine largeur sous elles : la frise soirée par soirée et le repli
 *      fermé du détail par couple.
 *
 * PLUS DE CHIFFRES NUS (2026-09-22). Le chiffre d'appel rendait cinq nombres en colonnes
 * (« 15,2 % », « 2,1 s », « 45 ripostes, 251 morts sans réponse ») là où deux graphes les
 * portent tous, avec leur forme : un taux est une PART, un délai est une DISTRIBUTION. Le
 * repli « combien de temps on met » a disparu avec eux — son histogramme est désormais un
 * bloc à part entière, et un repli qui répète le graphe au-dessus n'est pas un détail.
 *
 * PLUS DE PHRASE DE LECTEUR (D22-verbosité, LOI du 2026-09-21) : graphes et légendes
 * seulement, l'explication tient dans l'infobulle (i) du titre, en trois phrases au plus.
 * CHAQUE BLOC PORTE LA SIENNE — un bloc sans (i) n'aurait nulle part où dire sa méthode.
 *
 * LES CHIFFRES DE LA MAQUETTE NE SONT JAMAIS REPRIS : tout vient de `appelRiposte`.
 */
import { useMemo } from 'react'

import { DonutChart, type ChartPointDonut } from '@/components/charts/DonutChart'
import { HistogramChart, type ChartPointHistogram } from '@/components/charts/HistogramChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SemanticToken } from '@/lib/accessibility'
import { withLowSampleNote } from '@/lib/formatters/lowSampleNote'
import { intlLocale } from '@/lib/formatters'
import type { SquadEchange } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { SquadRiposteMatricePanel } from './SquadRiposteMatricePanel'
import { SquadRiposteSessionsChart } from './SquadRiposteSessionsChart'
import {
  appelRiposte,
  delaisSeries,
  friseRiposte,
  positionMediane,
  resumeDelais,
  type ResumeDelais,
  FENETRE_TENDANCE,
  PLANCHER_MORTS,
} from './squadRiposte.logic'
import { getSquadRiposteText } from './squadRiposteStrings'

export interface SquadRiposteCardProps {
  echange: SquadEchange
}

export function SquadRiposteCard({ echange }: SquadRiposteCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadRiposteText(locale)
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
  const secFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )
  const intFmt = useMemo(() => new Intl.NumberFormat(numLoc), [numLoc])

  const secondes = echange.fenetre_ms / 1000
  const appel = useMemo(() => appelRiposte(echange), [echange])
  const frise = useMemo(() => friseRiposte(echange), [echange])
  // Le résumé des délais est lu ICI et pas seulement dans le graphe : son total est le
  // premier mot de l'infobulle du bloc, et une infobulle vit dans le bandeau du bloc.
  const resume = useMemo(() => resumeDelais(echange), [echange])

  const aide = aideRiposte(t, secondes, appel.echantillonFaible)

  return (
    <>
      {/* RANGÉE DE TÊTE — deux blocs de même poids, chacun une mesure, chacun son (i). */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <SectionCard
          title={t.donutTitle}
          titleAdornment={titleWithInfo(
            <TooltipParagraphs items={[t.donutHelp(appel.mortsEquipe, secondes)]} />,
            { testId: 'squad-riposte-donut-aide' },
          )}
        >
          <div className="px-3 py-2">
            <DonutMortsRipostees appel={appel} pctFmt={pctFmt} intFmt={intFmt} t={t} />
          </div>
        </SectionCard>
        <SectionCard
          title={t.delaiTitle}
          titleAdornment={titleWithInfo(aideDelai(t, secondes, resume.total, echange.couverture.n), {
            testId: 'squad-riposte-delai-aide',
          })}
        >
          <div className="px-3 py-2">
            <DistributionDelai
              echange={echange}
              appel={appel}
              resume={resume}
              secFmt={secFmt}
              t={t}
            />
          </div>
        </SectionCard>
      </div>
      {/* LE BLOC « RIPOSTE » — la frise et le repli du détail par couple. */}
      <SectionCard
        title={t.title}
        label={t.label}
        titleAdornment={titleWithInfo(aide, { testId: 'squad-riposte-low-sample' })}
      >
        <div className="px-3 py-2" data-testid="squad-riposte">
          {frise.soirees.length > 0 ? (
            <SquadRiposteSessionsChart
              frise={frise}
              emptyMessage={t.friseEmpty}
              aboveLabel={t.friseAbove}
              belowLabel={t.friseBelow}
              trendLabel={t.friseTrend(FENETRE_TENDANCE)}
              usualLabel={t.friseUsual(pctFmt.format(appel.habituel))}
              yAxisLabel={t.friseYAxis}
              volumeAxisLabel={t.friseVolumeAxis}
              volumeTooltip={t.friseVolume}
            />
          ) : (
            <EmptyStateNotice title={t.emptyTitle} description={t.friseEmpty} />
          )}
        </div>
        <details className="border-t border-border pb-2" data-testid="squad-riposte-fold-matrice">
          <summary className="cursor-pointer px-3 py-1 text-xs text-muted-foreground">
            {t.foldMatrix}
          </summary>
          <div className="px-3 pb-1">
            <SquadRiposteMatricePanel echange={echange} />
          </div>
        </details>
      </SectionCard>
    </>
  )
}

type Appel = ReturnType<typeof appelRiposte>
type Texte = ReturnType<typeof getSquadRiposteText>

/**
 * L'INFOBULLE DU BLOC « RIPOSTE » : la définition (la fenêtre), la règle de l'habituel, et
 * la réserve d'échantillon quand elle s'applique. Tout ce qui dit la MÉTHODE y tient —
 * rien de cela ne doit être du gris entre les graphes.
 */
function aideRiposte(t: Texte, secondes: number, echantillonFaible: boolean) {
  return (
    <TooltipParagraphs
      items={[
        t.definition(secondes),
        t.helpUsual,
        echantillonFaible
          ? withLowSampleNote(t.lowSampleHint(PLANCHER_MORTS), true, t.lowSample)
          : null,
      ]}
    />
  )
}

/**
 * L'INFOBULLE DU BLOC « TEMPS DE RIPOSTE », en TROIS PHRASES — celles qui étaient en gris
 * SOUS le graphe jusqu'au 2026-09-22 : ce que la distribution compte (les ripostes
 * SURVENUES, sur le total des morts d'équipe), la fenêtre qui décide du taux, et le sort
 * des barres hors fenêtre — montrées, hachurées, jamais comptées.
 */
function aideDelai(t: Texte, secondes: number, ripostes: number, morts: number) {
  return (
    <TooltipParagraphs
      items={[t.delayFigure(ripostes, morts), t.delayWindow(secondes), t.delayFoot(secondes)]}
    />
  )
}

/**
 * Jetons des deux parts du donut.
 *
 * « ripostées » porte la COULEUR DE SÉRIE de la carte — celle des barres de l'histogramme
 * voisin (`seriesColor(0)` = `chart-series-1`) : la même population, la même encre, à un
 * demi-écran d'écart.
 *
 * « sans réponse » prend `zone-neutral`, ACHROMATIQUE DANS TOUTES LES PALETTES par
 * définition (voir `semantic-tokens.ts`) : ce n'est pas un verdict, c'est le reste. Lui
 * donner `stat-deaths` en ferait une seconde mesure de morts alors que les DEUX parts sont
 * des morts ; lui donner `outcome-loss` en ferait une défaite alors qu'une mort sans
 * réponse n'en est pas une.
 */
const JETON_RIPOSTEES: SemanticToken = 'chart-series-1'
const JETON_SANS_REPONSE: SemanticToken = 'zone-neutral'

/**
 * DonutMortsRipostees — les morts de notre camp coupées en DEUX PARTS EXCLUSIVES.
 *
 * Le dénominateur (« Sur N morts des nôtres ») n'est plus le titre : il vit dans
 * l'infobulle (i) du bandeau du bloc, dont le titre dit la MESURE (« Morts ripostées »).
 *
 * ÉTIQUETTES EXTÉRIEURES, AVEC CONNECTEURS (décision utilisateur du 2026-09-22 : plus de
 * `compact`). Le donut a désormais une colonne entière ; posées DANS l'arc, les deux
 * étiquettes de valeur se marchaient dessus sur la part mince.
 *
 * AUCUNE PHRASE SOUS LE DONUT. L'écart à l'habituel y était rendu en toutes lettres
 * (« -1 pts sous l'habituel ») — c'était la dernière phrase de lecteur de la carte, et la
 * frise du bloc « Riposte » porte déjà ce repère, tracé, soirée par soirée.
 *
 * LE TEXTE `sr-only` N'EST PAS UN DOUBLON : le taux et les deux parts vivent dans un
 * CANVAS, que ni un lecteur d'écran ni un test ne sait lire. Il porte le même contenu que
 * le centre et les arcs, rien de plus.
 */
function DonutMortsRipostees({
  appel,
  pctFmt,
  intFmt,
  t,
}: {
  appel: Appel
  pctFmt: Intl.NumberFormat
  intFmt: Intl.NumberFormat
  t: Texte
}) {
  const series = useMemo<ChartSeries<ChartPointDonut>[]>(
    () => [
      {
        key: 'riposte-parts',
        datapoints: [
          {
            name: t.appelUnit,
            value: appel.ripostes,
            valueLabel: intFmt.format(appel.ripostes),
          },
          {
            name: t.donutUnanswered,
            value: appel.sansReponse,
            valueLabel: intFmt.format(appel.sansReponse),
          },
        ],
      },
    ],
    [appel.ripostes, appel.sansReponse, intFmt, t],
  )
  const sliceColors = useMemo<Record<string, SemanticToken>>(
    () => ({ [t.appelUnit]: JETON_RIPOSTEES, [t.donutUnanswered]: JETON_SANS_REPONSE }),
    [t],
  )

  return (
    <div>
      <DonutChart
        series={series}
        sliceColors={sliceColors}
        arcLabelKind="value"
        centerValue={pctFmt.format(appel.taux)}
        centerLabel={t.appelUnit}
        height={240}
        frameless
      />
      <p className="sr-only" data-testid="squad-riposte-taux">
        {t.donutAlt({
          rate: pctFmt.format(appel.taux),
          morts: appel.mortsEquipe,
          ripostes: appel.ripostes,
          sans: appel.sansReponse,
        })}
      </p>
    </div>
  )
}

/**
 * DistributionDelai — COMBIEN DE TEMPS met notre camp à riposter, et où tombe la médiane.
 *
 * Les cinq premières barres couvrent la fenêtre de riposte (0-1 … 4-5 s, la borne de 5 s
 * comprise) ; les deux dernières sont HORS FENÊTRE : elles sont MONTRÉES, HACHURÉES, et
 * n'entrent dans AUCUN taux.
 *
 * POURQUOI LES MONTRER. Une distribution qui s'arrêterait net à 5 s ne dirait pas si la
 * fenêtre coupe une population dense ou du vide — « 19 % de morts ripostées » se lit très
 * différemment selon que les ripostes manquées arrivent à 5,2 s ou à 40 s.
 *
 * TROIS INDICES POUR LA FENÊTRE, et c'est voulu : la HACHURE sur la barre, le REPÈRE
 * vertical tireté « fenêtre N s » posé sur la borne, et le MOT (« hors fenêtre » en
 * étiquette d'axe, la note de pied). Un seuil qui décide d'un taux ne peut pas se deviner.
 *
 * LA MÉDIANE EST LE QUATRIÈME REPÈRE, et le seul qui ne dise pas une règle mais une
 * mesure : elle se pose à sa position FRACTIONNAIRE entre deux barres (`positionMediane`).
 *
 * RIEN N'EST PLUS ÉCRIT SOUS LE GRAPHE (décision utilisateur du 2026-09-22) : ni la
 * légende texte de la médiane — elle est DANS le graphe, sur son repère —, ni la note de
 * pied, passée dans l'infobulle (i) du bandeau (`aideDelai`). Du gris sous un graphe est
 * une phrase de lecteur, et la LOI du 2026-09-21 les interdit.
 *
 * Les intervalles sont PRÉ-BINNÉS par le serveur (ADR 0010) : ce composant ne choisit
 * aucune borne.
 */
function DistributionDelai({
  echange,
  appel,
  resume,
  secFmt,
  t,
}: {
  echange: SquadEchange
  appel: Appel
  resume: ResumeDelais
  secFmt: Intl.NumberFormat
  t: Texte
}) {
  const secondes = echange.fenetre_ms / 1000
  const series = useMemo(() => delaisSeries(echange), [echange])
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

  // Le libellé de la médiane ne sert qu'à SON REPÈRE, dans le graphe : la légende texte
  // qui le répétait dessous a disparu le 2026-09-22.
  const libelleMediane =
    appel.delaiMedianS != null ? t.delaiMedianMark(secFmt.format(appel.delaiMedianS)) : null
  const seuils = useMemo(() => {
    const at = positionMediane(echange)
    return at != null && libelleMediane ? [{ at, label: libelleMediane }] : undefined
  }, [echange, libelleMediane])

  if (resume.total === 0) {
    return <EmptyStateNotice title={t.emptyTitle} description={t.delayNarrativeEmpty} />
  }

  return (
    <div data-testid="squad-riposte-delai">
      <HistogramChart
        series={series}
        xAxisLabel={t.delayXAxis}
        yAxisLabel={t.delayYAxis}
        formatBin={formatBin}
        binHatched={binHatched}
        showValues
        windowMark={windowMark}
        thresholds={seuils}
        height={240}
        frameless
      />
    </div>
  )
}
