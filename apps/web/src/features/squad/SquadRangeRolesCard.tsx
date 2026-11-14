/**
 * SquadRangeRolesCard — « Rôles de portée », le ruban de la section « Appui et portée » de
 * Synergies.
 *
 * QUI TIENT LA LIGNE DE FRONT, QUI JOUE LOIN, ET QUI A CHANGÉ. Une ligne par joueur, une colonne
 * par match (du plus ancien au plus récent) : la case porte la couleur du RÔLE lu sur la fenêtre
 * glissante de `FENETRE_ROLE` matchs (Ligne de front / Polyvalent / Tireur d'élite). Le rôle se
 * lit sur l'écart de la médiane de frag d'un joueur à celle de TOUS les joueurs du lobby — la
 * seule échelle qui neutralise la carte et le mode —, et ses seuils sont les TIERS de la période,
 * pas des mètres en dur. Une case vide n'a pas encore de rôle (fenêtre incomplète, ou match sous
 * `PLANCHER_MESURE` frags mesurés) : le petit échantillon se lit dans l'infobulle de la case,
 * jamais sur le ruban.
 *
 * ELLE SE PLACE À DROITE DE L'APPUI, MÊME HAUTEUR : l'appui dit ce que l'escouade se donne, la
 * portée à quelle distance chacun se tient. Le nuage de rôles du Résumé solo est un autre
 * graphe (`charts/squadRangeRolesChart.ts`).
 *
 * D22-VERBOSITÉ (LOI) : graphe et légende du graphe seulement. La lecture tient dans
 * l'infobulle du titre ; sous le graphe, ni légende ajoutée ni compte de frags mesurés.
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

import { buildRangeRolesRibbonOption, hauteurRuban } from './charts/squadRangeRolesRibbon'
import {
  categoriesMatchs,
  FENETRE_ROLE,
  lignesRuban,
  ordonnerProfils,
  PLANCHER_MESURE,
  seriesPortee,
  seuilsRoles,
} from './squadRangeRoles.logic'
import { getSquadRangeRolesText } from './squadRangeRolesStrings'

export interface SquadRangeRolesCardProps {
  bloc: MatchRangeBlock
  /** Roster dans l'ordre de la page : joueur principal d'abord, puis les coéquipiers. */
  roster: string[]
  /** gamertag → couleur hex, palette de la page (`useSquadPlayerPalette`). */
  colorByPlayer: Record<string, string>
}

/** Préfixe des `data-testid` de la carte. */
const TID = 'squad-portee'

export function SquadRangeRolesCard({ bloc, roster, colorByPlayer }: SquadRangeRolesCardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadRangeRolesText(locale)
  const numLoc = intlLocale(locale)

  // Écart au lobby SIGNÉ (« +2,0 », « −1,5 ») : le sens se lit sans la couleur.
  const deltaFmt = useMemo(
    () =>
      new Intl.NumberFormat(numLoc, { minimumFractionDigits: 1, maximumFractionDigits: 1, signDisplay: 'exceptZero' }),
    [numLoc],
  )

  const profils = useMemo(() => ordonnerProfils(bloc.profiles ?? []), [bloc.profiles])
  const categories = useMemo(() => categoriesMatchs(profils), [profils])
  const series = useMemo(() => seriesPortee(profils, roster), [profils, roster])
  const seuils = useMemo(() => seuilsRoles(series), [series])
  const lignes = useMemo(() => lignesRuban(series, seuils), [series, seuils])

  const chartSeries = useMemo(() => (lignes.length > 0 ? [{ key: TID, datapoints: lignes }] : []), [lignes])

  const buildOption = useMemo(
    () => () =>
      buildRangeRolesRibbonOption(lignes, {
        categories,
        couleurs: colorByPlayer,
        libelles: {
          xAxis: t.xAxis,
          roles: t.bandes,
          tooltipRole: t.tooltipRole,
          tooltipNoRole: t.tooltipNoRole,
          tooltipDelta: t.tooltipDelta,
          tooltipMeasured: t.tooltipMeasured,
          tooltipUnsure: t.tooltipUnsure,
        },
        fmtDeltaM: (v: number) => deltaFmt.format(v),
      }),
    [lignes, categories, colorByPlayer, t, deltaFmt],
  )

  const vide = lignes.length === 0

  return (
    <SectionCard
      title={t.cardTitle}
      label={t.sectionLabel}
      titleAdornment={titleWithInfo(<TooltipParagraphs items={[t.help(PLANCHER_MESURE, FENETRE_ROLE)]} />)}
      className="h-full"
    >
      <div className="flex flex-1 flex-col justify-center px-3 py-2" data-testid={`${TID}-roles`}>
        {vide ? (
          <EmptyStateNotice title={t.emptyTitle} description={t.emptyDescription} />
        ) : (
          <ChartCard series={chartSeries} buildOption={buildOption} height={hauteurRuban(lignes.length)} frameless />
        )}
      </div>
    </SectionCard>
  )
}
