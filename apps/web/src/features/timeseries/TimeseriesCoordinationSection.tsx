/**
 * TimeseriesCoordinationSection — « Riposte » et « Appui reçu » dans le temps, sur
 * l'onglet Progression des Séries temporelles (lot Q ; D22-3 et D22-6/7 du 2026-09-21).
 *
 * UN SEUL GRAPHE PAR SUJET (amendement D22-3 à la maquette, qui en dessinait deux), et
 * depuis D23-3 (2026-09-22) EN ÉCART À SON REPÈRE : chaque grandeur est tracée comme sa
 * distance à sa propre référence — l'habituel de la période de référence (`habituel_pct`,
 * lot S) pour la première, la part équitable 1/n (`parity_pct`) pour la seconde.
 *
 * POURQUOI. En valeur, « je suis couvert » vit vers 48 % et « je riposte » vers 25 % :
 * deux bandes de bâtons qui ne se croisent jamais, sur un axe commun qui ne porte donc
 * rien, et une soirée qu'il faut juger en mesurant À L'ŒIL la distance de chaque bâton à
 * SON tireté. En écart, les deux grandeurs partagent enfin une unité — des points — les
 * deux tiretés se confondent dans UNE ligne zéro qui les nomme, et la soirée se lit par
 * la seule DIRECTION de ses bâtons. La valeur absolue ne quitte pas la carte : elle reste
 * dans les chiffres d'appel et dans l'infobulle, qui porte les deux lectures.
 *
 * D22-VERBOSITÉ : aucune phrase de lecteur. Les chiffres d'appel restent (ils disent le
 * fait), l'explication tient dans l'infobulle ⓘ du titre, en trois phrases au plus.
 *
 * AUCUNE REQUÊTE NEUVE : le bloc Coordination arrive avec la réponse de page (lot N1).
 * Le grain est la SOIRÉE et non le match : le dénominateur de la première grandeur, ce
 * sont mes morts — 8 à 14 par match en arène, où une seule mort déplace le point de
 * dix points. La rangée est CONSERVÉE quand le bloc est indisponible (D8) : deux cartes
 * qui nomment leur absence, jamais une section qui disparaît sans rien dire.
 */
import { useMemo, type ReactNode } from 'react'

import { SessionBarsTrendChart } from '@/components/charts/SessionBarsTrendCard'
import type { SessionBarsSeriesSpec } from '@/components/charts/sessionBarsTrendChart'
import { seriesColor } from '@/components/charts/_utils'
import { SectionCard } from '@/components/ui/section-card'
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { resolveToken } from '@/lib/accessibility'
import { intlLocale } from '@/lib/formatters'
import type { CoordinationBlock } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import {
  coordinationDessinable,
  delaiMedianS,
  habituelOuTaux,
  labelsDeSoirees,
  moyenneGlissante,
  pariteOuRien,
  serieDeSoirees,
  soireesDe,
  type SerieDeSoirees,
} from './timeseriesCoordination.logic'
import {
  getTimeseriesCoordinationText,
  type TimeseriesCoordinationText,
} from './timeseriesCoordinationStrings'

const HAUTEUR = 300

export interface TimeseriesCoordinationSectionProps {
  block: CoordinationBlock | undefined
  locale: Locale
}

export function TimeseriesCoordinationSection({
  block,
  locale,
}: TimeseriesCoordinationSectionProps) {
  const t = useMemo(() => getTimeseriesCoordinationText(locale), [locale])
  const numLoc = intlLocale(locale)
  const pctFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { style: 'percent', maximumFractionDigits: 1 }),
    [numLoc],
  )
  const secFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )

  // Titre sans bloc de coordination (capability absente, réponse ancienne) : rien n'est
  // rendu — une carte qui ne peut RIEN dire n'est pas une carte vide, elle n'existe pas.
  if (!block) return null

  const dessinable = coordinationDessinable(block)
  const sessions = soireesDe(block)
  const labels = labelsDeSoirees(sessions)
  const couverture = t.coverage(block.matches_measured, block.matches_total)
  const indisponible = block.available === false ? block.unavailable_reason || t.unavailable : null

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <CarteRiposte
        block={block}
        labels={labels}
        sessions={sessions}
        dessinable={dessinable}
        indisponible={indisponible}
        couverture={couverture}
        t={t}
        pctFmt={pctFmt}
        secFmt={secFmt}
      />
      <CarteAppui
        block={block}
        labels={labels}
        sessions={sessions}
        dessinable={dessinable}
        indisponible={indisponible}
        couverture={couverture}
        t={t}
        pctFmt={pctFmt}
      />
    </div>
  )
}

interface CarteProps {
  block: CoordinationBlock
  labels: string[]
  sessions: ReturnType<typeof soireesDe>
  dessinable: boolean
  indisponible: string | null
  couverture: string
  t: TimeseriesCoordinationText
  pctFmt: Intl.NumberFormat
}

interface GrandeurDeCarte {
  name: string
  color: string
  serie: SerieDeSoirees
}

/**
 * specEcart — une grandeur prête pour le MODE ÉCART de la frise.
 *
 * Le repère n'est plus un tireté sur l'axe : la frise le SOUSTRAIT, et son libellé part
 * rejoindre l'étiquette de la ligne zéro (`labelZero`). D'où `color` absent du repère —
 * il ne se dessine plus. Un repère `null` (parité non mesurée) laisse la grandeur en
 * valeur brute : on ne lui invente pas d'origine.
 *
 * La tendance est calculée sur les valeurs ABSOLUES et hors soirées à échantillon faible ;
 * la frise lui applique le même décalage qu'aux bâtons.
 */
function specEcart(
  g: GrandeurDeCarte,
  reperePct: number | null,
  repereLabel: string | null,
  t: TimeseriesCoordinationText,
): SessionBarsSeriesSpec {
  return {
    name: g.name,
    color: g.color,
    valuesPct: g.serie.valuesPct,
    hollow: g.serie.hollow,
    ...(reperePct != null && repereLabel != null
      ? { usual: { valuePct: reperePct, label: repereLabel } }
      : {}),
    trend: {
      valuesPct: moyenneGlissante(g.serie),
      label: t.trend(g.name),
      color: g.color,
    },
  }
}

/** L'étiquette de la ligne zéro : elle NOMME les repères qu'elle a confondus en elle. */
function labelZero(
  habituel: string,
  parite: string | null,
  t: TimeseriesCoordinationText,
): string {
  return parite == null ? habituel : t.zeroLabel(habituel, parite)
}

/** « Riposte » : je suis couvert (mes morts ripostées) contre je riposte (part du camp). */
function CarteRiposte({
  block,
  labels,
  sessions,
  dessinable,
  indisponible,
  couverture,
  t,
  pctFmt,
  secFmt,
}: CarteProps & { secFmt: Intl.NumberFormat }) {
  const r = block.riposte
  const couvert = serieDeSoirees(sessions, (p) => p.riposte.je_suis_couvert)
  const mien = serieDeSoirees(sessions, (p) => p.riposte.je_riposte)
  const parite = pariteOuRien(r.parity_pct)
  const delai = delaiMedianS(r.delai_median_ms)

  const habituel = habituelOuTaux(r.habituel_pct, r.je_suis_couvert)
  const labelHabituel = t.usual(pctFmt.format(habituel / 100))
  const labelParite = parite == null ? null : t.parity(pctFmt.format(parite / 100))
  const specs: SessionBarsSeriesSpec[] = [
    specEcart({ name: t.covered, color: seriesColor(0), serie: couvert }, habituel, labelHabituel, t),
    specEcart({ name: t.iRiposte, color: seriesColor(1), serie: mien }, parite, labelParite, t),
  ]

  return (
    <CarteDeCoordination
      title={t.riposteTitle}
      aide={
        <TooltipParagraphs
          items={[t.riposteTooltip(block.fenetre_ms / 1000), t.tooltipZero, t.tooltipTrend]}
        />
      }
      appels={[
        { label: t.covered, value: pctFmt.format(r.je_suis_couvert.taux) },
        { label: t.iRiposte, value: pctFmt.format(r.je_riposte.taux) },
        { label: t.delay, value: delai == null ? '—' : `${secFmt.format(delai)} s` },
      ]}
      indisponible={indisponible}
      empty={t.empty}
      couverture={couverture}
      testId="timeseries-coord-riposte"
    >
      {dessinable && (
        <SessionBarsTrendChart
          labels={labels}
          series={specs}
          yAxisLabel={t.yAxisDelta}
          baseline={{ label: labelZero(labelHabituel, labelParite, t), deltaUnit: t.points }}
          hollowLegend={{ label: t.hollow, color: seriesColor(0) }}
          tooltipLines={(i: number) => [
            t.volMyDeaths(couvert.volumes[i] ?? 0),
            t.volTeamDeaths(mien.volumes[i] ?? 0),
          ]}
          height={HAUTEUR}
          emptyMessage={t.empty}
        />
      )}
    </CarteDeCoordination>
  )
}

/** « Appui reçu » : on me prépare (mes frags appuyés) contre ma part des appuis du camp. */
function CarteAppui({
  block,
  labels,
  sessions,
  dessinable,
  indisponible,
  couverture,
  t,
  pctFmt,
}: CarteProps) {
  const a = block.appui
  const prepare = serieDeSoirees(sessions, (p) => p.appui.on_me_prepare)
  const part = serieDeSoirees(sessions, (p) => p.appui.ma_part_des_appuis)
  const parite = pariteOuRien(a.parity_pct)

  // Les deux SENS de l'assistance portent leurs jetons dédiés (famille des stats de
  // combat, 2026-09-17) : on me prépare = je REÇOIS, ma part des appuis = je DONNE.
  const habituel = habituelOuTaux(a.habituel_pct, a.on_me_prepare)
  const labelHabituel = t.usual(pctFmt.format(habituel / 100))
  const labelParite = parite == null ? null : t.parity(pctFmt.format(parite / 100))
  const specs: SessionBarsSeriesSpec[] = [
    specEcart(
      { name: t.prepared, color: resolveToken('assist-received'), serie: prepare },
      habituel,
      labelHabituel,
      t,
    ),
    specEcart(
      { name: t.myShare, color: resolveToken('assist-given'), serie: part },
      parite,
      labelParite,
      t,
    ),
  ]

  return (
    <CarteDeCoordination
      title={t.appuiTitle}
      aide={<TooltipParagraphs items={[t.appuiTooltip, t.tooltipZero, t.tooltipTrend]} />}
      appels={[
        { label: t.prepared, value: pctFmt.format(a.on_me_prepare.taux) },
        { label: t.myShare, value: pctFmt.format(a.ma_part_des_appuis.taux) },
      ]}
      indisponible={indisponible}
      empty={t.empty}
      couverture={couverture}
      testId="timeseries-coord-appui"
    >
      {dessinable && (
        <SessionBarsTrendChart
          labels={labels}
          series={specs}
          yAxisLabel={t.yAxisDelta}
          baseline={{ label: labelZero(labelHabituel, labelParite, t), deltaUnit: t.points }}
          hollowLegend={{ label: t.hollow, color: resolveToken('assist-received') }}
          tooltipLines={(i: number) => [
            t.volMyKills(prepare.volumes[i] ?? 0),
            t.volTeamAssists(part.volumes[i] ?? 0),
          ]}
          height={HAUTEUR}
          emptyMessage={t.empty}
        />
      )}
    </CarteDeCoordination>
  )
}

interface AppelChiffre {
  label: string
  value: string
}

interface CarteDeCoordinationProps {
  title: string
  aide: ReactNode
  appels: AppelChiffre[]
  /** Raison d'indisponibilité servie par le serveur — l'état vide NOMMÉ de la carte. */
  indisponible: string | null
  empty: string
  couverture: string
  testId: string
  children: ReactNode
}

/** Le gabarit commun : chiffres d'appel, graphe, couverture en pied. */
function CarteDeCoordination({
  title,
  aide,
  appels,
  indisponible,
  empty,
  couverture,
  testId,
  children,
}: CarteDeCoordinationProps) {
  return (
    <SectionCard
      title={title}
      label={title}
      titleAdornment={titleWithInfo(aide)}
      footer={
        <p className="border-t border-border px-3 py-2 text-xs text-muted-foreground">
          {couverture}
        </p>
      }
    >
      <div className="space-y-2 px-3 py-2" data-testid={testId}>
        {indisponible ? (
          <p className="py-6 text-center text-sm text-muted-foreground">{indisponible}</p>
        ) : (
          <>
            <div className="flex flex-wrap items-baseline gap-x-6 gap-y-1">
              {appels.map((a) => (
                <span key={a.label} className="flex items-baseline gap-1.5">
                  <span className="text-xs uppercase tracking-wide text-muted-foreground">
                    {a.label}
                  </span>
                  <span className="tabular-nums text-sm font-semibold text-foreground">
                    {a.value}
                  </span>
                </span>
              ))}
            </div>
            {children || (
              <p className="py-6 text-center text-sm text-muted-foreground">{empty}</p>
            )}
          </>
        )}
      </div>
    </SectionCard>
  )
}
