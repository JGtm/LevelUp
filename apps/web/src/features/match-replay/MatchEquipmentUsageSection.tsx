/**
 * MatchEquipmentUsageSection — « Usage d'équipements, par joueur » : LE BILAN D'ÉQUIPEMENT D'UN
 * MATCH, joueur par joueur (Vue match, carte F du plan PLAN_MATCHVIEW_EMPRISE_2026-10-06, D15).
 *
 * Le rejeu 2D montre chaque geste d'équipement à l'IMAGE où il a lieu ; cette carte les COMPTE —
 * aucune donnée nouvelle, aucun appel de plus. La grille partagée `components/charts/ValueGrid` :
 * lignes = joueurs dans l'ordre du roster, équipe par équipe, filet entre les deux ; colonnes =
 * familles décidées par la donnée (`usageColumnGroups`), CHACUNE AVEC SON ÉCHELLE ; une pile par
 * colonne sur les trois issues d'un objet pris (servi, gardé sans servir, lâché), les tractions pour
 * le grappin. La part de chaque équipe, qui avait sa carte voisine, se lit désormais dans « Contrôle
 * des ressources, par match ».
 *
 * ELLE VIT DANS `match-replay/` ET NON DANS `match-view/` : chaque libellé qu'elle écrit appartient
 * au dictionnaire du rejeu (`REPLAY_TEXT`) et aux tables du document.
 *
 * DOUBLE PORTE : pas d'artefact OU aucune grandeur mesurée -> RIEN. La MÊME clé de cache que la
 * courbe de score (`useMatchReplay`, gaté par `header.replay_available`).
 *
 * CE QUE L'ÉCRAN DIT DE SA PROPRE MESURE : l'aide du titre dit ce que la grille compte, puis les
 * gestes que le film mesure sans en nommer l'auteur ou l'origine (réserve, jamais cachée —
 * décision utilisateur du 2026-09-09) ; aucun texte de pied.
 *
 * COULEURS : les issues prennent `USAGE_OUTCOME_TOKENS`, le grappin son encre de famille ; les
 * équipes, au nom des lignes, `teamTokenCssVar` (allégeance lue dans le film).
 */
import { useCallback, useMemo } from 'react'

import { ChartLegend } from '@/components/charts/ChartLegend'
import { ValueGrid } from '@/components/charts/ValueGrid'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { SectionCard } from '@/components/ui/section-card'

import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import { meXUIDOf } from '@/features/match-view/xuidMeta'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'
import { campLabel, type ReplayCamp } from '@/lib/replay/replayCamps'

import { buildUsageGrid, usageGroupColor, usageOutcomeColor } from './model/equipmentUsageChart'
import { usageColumnGroups } from './model/equipmentUsageColumns'
import {
  buildEquipmentUsage,
  hasEquipmentUsage,
  tallyTotal,
  type EquipmentUsage,
} from './model/equipmentUsageLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n/i18n'
import type { ReplayText } from './i18n/i18nContract'
import { useMatchReplay } from '../../lib/replay/queries'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — le même gate que le lien rejeu et que la courbe de score. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  locale: ReplayLocale
}

export function MatchEquipmentUsageSection({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  locale,
}: Props) {
  const t = REPLAY_TEXT[locale]
  const u = t.equipmentUsage
  const { data } = useMatchReplay(playerSlug, matchId, replayAvailable)
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const usage = useMemo(() => (data ? buildEquipmentUsage(data, board) : null), [data, board])
  // PLUS DE REPLI DEPUIS LE 2026-09-19 (plan d'ajustements pré-v7.5, lot 2) : toutes les
  // colonnes que la donnée justifie s'affichent au chargement. Le bouton « Voir plus (N) /
  // Replier » et la partition « game changers » qui l'alimentait ont été retirés — la même
  // décision que le bloc voisin (« Contrôle des armes spéciales ») avait déjà prise le
  // 2026-09-13 : une carte qui cache sa mesure par défaut ne se lit pas.
  const groups = useUsageGroups(usage, t)
  const reserve = useMemo(() => usageReserve(usage), [usage])
  const meXUID = useMemo(() => meXUIDOf(board), [board])
  // L'ALLÉGEANCE DU FILM, VUE DU JOUEUR DE LA PAGE (2026-10-06) : son équipe du film dit quel
  // camp est le sien — les camps eux-mêmes sont ceux du film (`equipmentUsageLogic`).
  const allegiance = useMemo(() => filmAllegianceOf(data, board, meXUID), [data, board, meXUID])

  // LE NOM D'UN CAMP DU FILM : la cascade des colonnes de fiches (`campLabel`) sur la feuille,
  // « Équipe N » de son désignateur quand elle se tait — jamais « sans équipe ».
  const teamLabel = useCallback((camp: ReplayCamp) => campLabel(camp, board, t), [board, t])
  // « Allié » = du camp du FILM du joueur de la page. Quand le film ne le situe pas (absent,
  // équipe tue), l'allégeance est INCONNUE (null) : encre neutre, jamais l'une des deux couleurs
  // d'équipe (même règle que `ReplayTeamHeader`).
  const teamAccent = useCallback((camp: ReplayCamp) => teamTokenCssVar(allegiance.ofTeam(camp.team)), [allegiance])

  const grid = useMemo(
    () =>
      buildUsageGrid({
        teams: usage?.byTeam ?? [],
        groups,
        meXUID,
        teamLabel,
        teamAccent,
        tipFmt: t.equipmentUsage.gridTipFmt,
      }),
    [usage, groups, meXUID, teamLabel, teamAccent, t],
  )
  // Double porte : pas d'artefact, ou rien de mesuré -> rien du tout. MÊME prédicat que
  // celui lu par le parent pour poser (ou non) son titre de section.
  if (!hasEquipmentUsage(usage) || groups.length === 0) return null

  // L'AIDE DE LA CARTE (plan PLAN_MATCHVIEW_EMPRISE_2026-10-06, D15) : ce que la grille mesure, puis la
  // RÉSERVE quand elle n'est pas nulle — elle ne se cache pas (décision utilisateur 2026-09-09).
  const info = (
    <>
      <p>{u.infoByPlayer}</p>
      {reserve > 0 && <p>{u.coverageReserveFmt(reserve)}</p>}
    </>
  )
  // LA LÉGENDE DIT LES TROIS ISSUES d'une pile (servi, gardé sans servir, lâché), plus les tractions
  // quand la colonne du grappin existe : la couleur d'une pile est celle de l'issue.
  const legend = [
    { key: 'used', label: u.legendUsed, color: usageOutcomeColor('used') },
    { key: 'kept', label: u.legendKept, color: usageOutcomeColor('kept') },
    { key: 'dropped', label: u.legendDropped, color: usageOutcomeColor('dropped') },
    ...(groups.some((g) => g.key === 'grapple') ? [{ key: 'grapple', label: u.legendGrapple, color: usageGroupColor('grapple') }] : []),
  ]

  return (
    <SectionCard title={u.viewByPlayer} label={u.viewByPlayer} titleAdornment={titleWithInfo(info)}>
      <div className="px-3 pb-3 pt-3">
        <ValueGrid model={grid} />
        <ChartLegend className="pt-2" items={legend} />
      </div>
    </SectionCard>
  )
}

/**
 * useUsageGroups — les colonnes réellement rendues : TOUTES celles que la donnée justifie,
 * dans l'ordre écrit de `usageColumnGroups` (plus de repli depuis le 2026-09-19).
 */
function useUsageGroups(usage: EquipmentUsage | null, t: ReplayText) {
  return useMemo(() => (usage ? usageColumnGroups(usage, t) : []), [usage, t])
}
/**
 * unknownOriginPlacements — LES POSES D'ORIGINE INCONNUE : la somme des poses `famille/unknown`
 * de `coverage.placements.byFamilyOrigin` — ~5 % du parc mesurés par E0 (681/11 438).
 * `origin: 'unknown'` n'est ni un déploiement ni un lâcher (schéma antérieur au 10, ou pose sans
 * poseur mesuré) : elle n'est comptée dans AUCUNE des deux vues, et pourtant elle a eu lieu.
 */
function unknownOriginPlacements(cov: EquipmentUsage['coverage']): number {
  let total = 0
  for (const [key, n] of Object.entries(cov.placementsByFamilyOrigin)) {
    if (key.endsWith('/unknown')) total += n
  }
  return total
}

/**
 * usageReserve — CE QUE LES DEUX VUES NE COMPTENT PAS, en un seul nombre.
 *
 * Deux mesures, une seule réserve : les gestes sans propriétaire (slot n'appartenant à aucun
 * joueur, ou poseur non mesuré) et les poses d'origine inconnue. Elles ont eu lieu, le film ne
 * les rattache à personne, elles ne peuvent donc entrer ni dans la grille par joueur ni dans la
 * part par équipe. La réserve NE SE CACHE PAS (décision utilisateur 2026-09-09) : depuis le
 * 2026-09-14 elle se dit dans l'INFOBULLE DU TITRE, en une phrase, et non plus en pied de carte
 * (l'utilisateur ne veut aucun texte de pied sous ce bloc).
 *
 * `unnamedTaken` (objets pris dont le rang n'a pas de famille connue) n'y entre PAS : décision
 * utilisateur du 2026-09-09, ces objets ne s'affichent pas dans l'interface — le compteur reste
 * publié par la logique pour l'outillage d'investigation.
 */
function usageReserve(usage: EquipmentUsage | null): number {
  if (!usage) return 0
  return tallyTotal(usage.unattributed) + unknownOriginPlacements(usage.coverage)
}
