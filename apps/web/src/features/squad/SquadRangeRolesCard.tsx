/**
 * SquadRangeRolesCard — le nuage des RÔLES de la section Coordination : « Rôles de portée »
 * (D22-5 du 2026-09-21) et, avec `grandeur="hauteur"`, « Rôles de hauteur » (E1 / D24 du
 * 2026-09-22).
 *
 * UNE SEULE CARTE POUR LES DEUX GRANDEURS, pas une copie : la grammaire est la même à la
 * lettre — un point par (match, joueur), l'écart à la médiane du LOBBY du match en
 * ordonnée, trois bandes aux tiers de la période, une tendance glissante par joueur. Seuls
 * changent la grandeur lue sur le profil (`seriesPortee`) et le préfixe de libellés du
 * manifest. Deux lectures qui s'empilent au lieu de se concurrencer.
 *
 * QUI TIENT LA LIGNE DE FRONT, QUI JOUE LOIN, ET QUI A CHANGÉ. Un point par (match,
 * joueur) : en abscisse le match, du plus ancien au plus récent ; en ordonnée l'écart de sa
 * médiane de frag à celle de TOUS les joueurs du lobby — la seule échelle qui neutralise la
 * carte et le mode. Trois bandes de fond portent les rôles, et leurs seuils sont les TIERS
 * de la période, pas des mètres en dur. Une tendance fine par joueur (moyenne glissante de
 * `FENETRE_ROLE` matchs) dit le rôle ; un point atypique se voit sans faire basculer
 * l'étiquette.
 *
 * ELLE SE PLACE DANS LA SECTION COORDINATION, APRÈS LE NUAGE D'ISOLEMENT : les trois cartes
 * de la section répondent à la même question — comment l'escouade s'occupe de l'espace
 * entre ses joueurs. La riposte dit qui vient, l'isolement dit pourquoi personne ne vient,
 * la portée dit à quelle distance chacun se tient. Elle vient en dernier parce qu'elle est
 * la seule à ne pas parler de morts.
 *
 * D22-VERBOSITÉ (LOI) : graphe et légendes seulement. La lecture tient dans l'infobulle du
 * titre, en trois phrases ; sous le graphe, rien d'autre que la légende et la couverture.
 */
import { useMemo } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { TooltipParagraphs } from '@/components/ui/info-tooltip'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { intlLocale } from '@/lib/formatters'
import type { MatchRangeBlock } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { buildSquadRangeRolesOption } from './charts/squadRangeRolesChart'
import { getSquadPlayerColors } from './colors'
import { SquadRangeRolesTape } from './SquadRangeRolesTape'
import {
  categoriesMatchs,
  FENETRE_ROLE,
  ordonnerProfils,
  PLANCHER_MESURE,
  rolesFenetre,
  seriesPortee,
  seuilsRoles,
  TAILLE_POINT_MIN,
  type GrandeurProfil,
  type RoleDePortee,
} from './squadRangeRoles.logic'
import { getSquadRangeRolesText } from './squadRangeRolesStrings'

// ENCRE PLEINE SUR LES DEUX GRANDEURS (2026-09-22, retour utilisateur « rendu terne ») : les
// points de la grandeur hauteur portaient une opacité de 0,5 pour laisser les courbes de
// tendance passer devant. Cette atténuation ne codait AUCUNE information — deux points de même
// nature s'affichaient plus pâles ici que sur la carte voisine, à un mètre d'écart. La
// lecture qu'elle cherchait à privilégier — la tendance — est tenue autrement : la courbe passe
// elle aussi en encre pleine et porte le gamertag à son bout. Supprimée ici et dans l'option (la
// prop `opacitePoints` n'avait pas d'autre appelant).

export interface SquadRangeRolesCardProps {
  bloc: MatchRangeBlock
  /** Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. */
  roster: string[]
  /**
   * La grandeur portée en ordonnée. `portee` (défaut, lot R) : la distance des frags.
   * `hauteur` (E1, D24 du 2026-09-22) : leur dénivelé signé. MÊME CARTE, MÊME NUAGE, MÊME
   * BANDE — seuls la grandeur lue sur le profil et les libellés du manifest changent.
   */
  grandeur?: GrandeurProfil
}

export function SquadRangeRolesCard({
  bloc,
  roster,
  grandeur = 'portee',
}: SquadRangeRolesCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadRangeRolesText(locale, grandeur)
  // Préfixe des `data-testid` : deux cartes cohabitent sur la page, leurs repères aussi.
  const tid = `squad-${grandeur}`
  // La lecture de E1 tient dans la tendance (4 joueurs x 20 matchs = 80 points) : chaque
  // courbe porte son gamertag au bout. Les points, eux, restent en encre pleine (2026-09-22).
  const hauteur = grandeur === 'hauteur'
  const numLoc = intlLocale(locale)

  const numFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )

  const profils = useMemo(() => ordonnerProfils(bloc.profiles ?? []), [bloc.profiles])
  const categories = useMemo(() => categoriesMatchs(profils), [profils])
  const series = useMemo(
    () => seriesPortee(profils, roster, grandeur),
    [profils, roster, grandeur],
  )
  const seuils = useMemo(() => seuilsRoles(series), [series])
  const roles = useMemo(() => {
    const out = new Map<string, (RoleDePortee | null)[]>()
    for (const s of series) out.set(s.xuid, rolesFenetre(s.points, seuils))
    return out
  }, [series, seuils])

  const couleurs = useMemo(() => {
    const [main, ...coequipiers] = roster
    return getSquadPlayerColors(main ?? '', coequipiers)
  }, [roster])

  // Les extrêmes RÉELS de `measured` sur la période : la taille des points s'y projette.
  const mesures = series.flatMap((s) => s.points).map((p) => p.mesures)
  const mesuresMin = mesures.length > 0 ? Math.min(...mesures) : 0
  const mesuresMax = mesures.length > 0 ? Math.max(...mesures) : 0

  const chartSeries = useMemo(
    () => (series.length > 0 ? [{ key: tid, datapoints: series }] : []),
    [series, tid],
  )

  const buildOption = useMemo(
    () => () =>
      buildSquadRangeRolesOption(series, {
        categories,
        seuils,
        couleurs,
        mesuresMin,
        mesuresMax,
        libelles: {
          xAxis: t.xAxis,
          yAxis: t.yAxis,
          lobbyLine: t.lobbyLine,
          bandes: t.bandes,
          tooltipMedian: t.tooltipMedian,
          tooltipDelta: t.tooltipDelta,
          tooltipMeasured: t.tooltipMeasured,
        },
        fmtM: (v: number) => numFmt.format(v),
        etiquetteBout: hauteur,
      }),
    [series, categories, seuils, couleurs, mesuresMin, mesuresMax, t, numFmt, hauteur],
  )

  const vide = series.length === 0

  return (
    <SectionCard
      title={t.cardTitle}
      label={t.sectionLabel}
      titleAdornment={titleWithInfo(<TooltipParagraphs items={[t.help(PLANCHER_MESURE)]} />)}
    >
      <div className="space-y-2 px-3 py-2" data-testid={`${tid}-roles`}>
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <>
            <ChartCard series={chartSeries} buildOption={buildOption} height={380} frameless />
            {/* Légende des deux encodages qu'ECharts ne sait pas nommer : le point creux et
                la tendance. Les joueurs, eux, sont dans la légende du graphe. */}
            <div
              className="flex flex-wrap items-center gap-4 text-2xs text-muted-foreground"
              data-testid={`${tid}-legende`}
            >
              <span className="flex items-center gap-1.5">
                <span
                  className="inline-block rounded-full border border-dashed border-muted-foreground"
                  style={{ width: TAILLE_POINT_MIN, height: TAILLE_POINT_MIN }}
                />
                {t.legendLowSample(PLANCHER_MESURE)}
              </span>
              <span className="flex items-center gap-1.5">
                <span className="inline-block h-px w-5 bg-muted-foreground" />
                {t.legendTrend(FENETRE_ROLE)}
              </span>
            </div>
            <p className="text-2xs text-muted-foreground" data-testid={`${tid}-couverture`}>
              {t.coverage(bloc.kills_measured, bloc.kills_total)}
            </p>
          </>
        )}
      </div>
      {!vide && (
        <details className="border-t border-border pb-2" data-testid={`${tid}-fold-bande`}>
          <summary className="cursor-pointer px-3 py-1 text-xs text-muted-foreground">
            {t.foldTape}
          </summary>
          <div className="px-3 pb-1">
            <SquadRangeRolesTape
              series={series}
              roles={roles}
              categories={categories}
              t={t}
              prefixeTest={tid}
            />
          </div>
        </details>
      )}
    </SectionCard>
  )
}
