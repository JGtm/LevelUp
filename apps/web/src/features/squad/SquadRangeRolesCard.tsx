/**
 * SquadRangeRolesCard — « Rôles de portée », le nuage des rôles de la section « Appui et
 * portée » de Synergies.
 *
 * QUI TIENT LA LIGNE DE FRONT, QUI JOUE LOIN, ET QUI A CHANGÉ. Un point par (match,
 * joueur) : en abscisse le match, du plus ancien au plus récent ; en ordonnée l'écart de sa
 * médiane de frag à celle de TOUS les joueurs du lobby — la seule échelle qui neutralise la
 * carte et le mode. Trois bandes de fond portent les rôles, et leurs seuils sont les TIERS
 * de la période, pas des mètres en dur. Une tendance fine par joueur (moyenne glissante de
 * `FENETRE_ROLE` matchs) dit le rôle ; un point atypique se voit sans faire basculer
 * l'étiquette.
 *
 * ELLE SE PLACE À DROITE DE L'APPUI, MÊME HAUTEUR : l'appui dit ce que l'escouade se donne, la
 * portée à quelle distance chacun se tient.
 *
 * D22-VERBOSITÉ (LOI) : graphe et légende du graphe seulement. La lecture (point creux sous le
 * plancher de frags mesurés, tendance glissante) tient dans l'infobulle du titre ; sous le graphe,
 * ni légende ajoutée ni compte de frags mesurés.
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
  ordonnerProfils,
  PLANCHER_MESURE,
  rolesFenetre,
  seriesPortee,
  seuilsRoles,
  type RoleDePortee,
} from './squadRangeRoles.logic'
import { getSquadRangeRolesText } from './squadRangeRolesStrings'

export interface SquadRangeRolesCardProps {
  bloc: MatchRangeBlock
  /** Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. */
  roster: string[]
}

/** Préfixe des `data-testid` de la carte et de sa bande des rôles. */
const TID = 'squad-portee'

export function SquadRangeRolesCard({ bloc, roster }: SquadRangeRolesCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadRangeRolesText(locale)
  const numLoc = intlLocale(locale)

  const numFmt = useMemo(
    () => new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1 }),
    [numLoc],
  )

  const profils = useMemo(() => ordonnerProfils(bloc.profiles ?? []), [bloc.profiles])
  const categories = useMemo(() => categoriesMatchs(profils), [profils])
  const series = useMemo(
    () => seriesPortee(profils, roster),
    [profils, roster],
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
    () => (series.length > 0 ? [{ key: TID, datapoints: series }] : []),
    [series],
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
      }),
    [series, categories, seuils, couleurs, mesuresMin, mesuresMax, t, numFmt],
  )

  const vide = series.length === 0

  return (
    <SectionCard
      title={t.cardTitle}
      label={t.sectionLabel}
      titleAdornment={titleWithInfo(<TooltipParagraphs items={[t.help(PLANCHER_MESURE)]} />)}
      className="h-full"
    >
      <div className="flex flex-1 flex-col justify-center px-3 py-2" data-testid={`${TID}-roles`}>
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <ChartCard series={chartSeries} buildOption={buildOption} height={380} frameless />
        )}
      </div>
      {!vide && (
        <details className="border-t border-border pb-2" data-testid={`${TID}-fold-bande`}>
          <summary className="cursor-pointer px-3 py-1 text-xs text-muted-foreground">
            {t.foldTape}
          </summary>
          <div className="px-3 pb-1">
            <SquadRangeRolesTape
              series={series}
              roles={roles}
              categories={categories}
              t={t}
            />
          </div>
        </details>
      )}
    </SectionCard>
  )
}
