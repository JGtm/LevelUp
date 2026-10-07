/**
 * TimeseriesCoordinationSection — « Appui reçu » dans le temps, sur l'onglet Progression des
 * Séries temporelles (lot Q ; D22-6/7 du 2026-09-21). La carte « Riposte » a quitté la page
 * (décision V5 du plan PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05). Le composant rend la CARTE
 * seule, sans grille : l'appelant la pose à droite de « Rendement & Résistance », sur la même
 * rangée (retour utilisateur du 2026-10-07). Aucune mention de couverture en pied : la
 * réserve de mesure tient dans l'infobulle et dans les bâtons creux.
 *
 * EN ÉCART À SON REPÈRE (D23-3, 2026-09-22) : chaque grandeur est tracée comme sa distance à
 * sa propre référence — l'habituel de la période de référence (`habituel_pct`) pour « on me
 * prépare », la part équitable 1/n (`parity_pct`) pour « ma part des appuis ». Les deux
 * grandeurs partagent une unité — des points —, les deux tiretés se confondent dans UNE ligne
 * zéro qui les nomme, et la soirée se lit par la seule DIRECTION de ses bâtons. La valeur
 * absolue reste dans les chiffres d'appel et dans l'infobulle.
 *
 * D22-VERBOSITÉ : aucune phrase de lecteur. Les chiffres d'appel restent (ils disent le
 * fait), l'explication tient dans l'infobulle ⓘ du titre, en trois phrases au plus.
 *
 * AUCUNE REQUÊTE NEUVE : le bloc Coordination arrive avec la réponse de page (lot N1).
 * Le grain est la SOIRÉE et non le match : les dénominateurs, mes frags et les appuis du camp,
 * sont trop peu nombreux par match pour un point lisible. La
 * carte est CONSERVÉE quand le bloc est indisponible (D8) : elle nomme son absence, jamais une
 * section qui disparaît sans rien dire.
 */
import { useMemo, type ReactNode } from 'react'

import { SessionBarsTrendChart } from '@/components/charts/SessionBarsTrendCard'
import type { SessionBarsSeriesSpec } from '@/components/charts/sessionBarsTrendChart'
import { SectionCard } from '@/components/ui/section-card'
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { resolveToken } from '@/lib/accessibility'
import { intlLocale } from '@/lib/formatters'
import type { CoordinationBlock } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import {
  coordinationDessinable,
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


  // Titre sans bloc de coordination (capability absente, réponse ancienne) : rien n'est
  // rendu — une carte qui ne peut RIEN dire n'est pas une carte vide, elle n'existe pas.
  if (!block) return null

  const dessinable = coordinationDessinable(block)
  const sessions = soireesDe(block)
  const labels = labelsDeSoirees(sessions)
  const indisponible = block.available === false ? block.unavailable_reason || t.unavailable : null

  return (
    <CarteAppui
      block={block}
      labels={labels}
      sessions={sessions}
      dessinable={dessinable}
      indisponible={indisponible}
      t={t}
      pctFmt={pctFmt}
    />
  )
}

interface CarteProps {
  block: CoordinationBlock
  labels: string[]
  sessions: ReturnType<typeof soireesDe>
  dessinable: boolean
  indisponible: string | null
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

/** « Appui reçu » : on me prépare (mes frags appuyés) contre ma part des appuis du camp. */
function CarteAppui({
  block,
  labels,
  sessions,
  dessinable,
  indisponible,
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
  testId: string
  children: ReactNode
}

/** Le gabarit commun : chiffres d'appel puis graphe, centré dans la hauteur de la rangée. */
function CarteDeCoordination({
  title,
  aide,
  appels,
  indisponible,
  empty,
  testId,
  children,
}: CarteDeCoordinationProps) {
  return (
    <SectionCard
      title={title}
      label={title}
      titleAdornment={titleWithInfo(aide)}
    >
      <div className="flex flex-1 flex-col justify-center gap-2 px-3 py-2" data-testid={testId}>
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
