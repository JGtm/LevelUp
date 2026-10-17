/**
 * MatchViewTabArsenal — contenu de l'onglet « Armes et terrain » de la page match.
 *
 * DEUX SECTIONS TITRÉES (plan PLAN_MATCHVIEW_EMPRISE_2026-10-06, §3 et S11) :
 *   1. « Frags et armes » — la répartition des frags (anneau) et les outils de destruction côte à
 *      côte (`MatchFragCard`), puis la distance des frags par arme (`MatchKillDistanceSection`) ;
 *   2. « Équipement et terrain », suivi de sa couverture (« film décodé · n joueurs présents à la
 *      fin ») — contrôle des ressources par match, prises par joueur, usage d'équipements par joueur
 *      (`MatchEquipmentUsageSection`, l'artefact de rejeu), frags et rendement par ressource côte à
 *      côte, isolement par joueur, puis les positions du film sur le plan (`MatchPositionsHeatmap`).
 * Les cartes de l'Emprise lisent les blocs du Go (`emprise`, `lives_near_teammate`,
 * `combat_tab.weapon_tools`) ; l'usage d'équipements et les positions lisent l'artefact de rejeu et
 * les positions keyframe, tirées par la page UNIQUEMENT quand cet onglet est actif.
 *
 * UN TITRE DE SECTION NE S'AFFICHE JAMAIS AU-DESSUS DE RIEN (règle du chantier, 2026-09-22). Le
 * prédicat de rendu de chaque bloc vit hors de son fichier de composant — `hasMatchFragData`,
 * `hasPositions` et `useHasKillDistanceSection` dans `./blockPredicates`, la présence des cartes de
 * l'Emprise dans `./matchEmprise.logic`, `hasEquipmentUsage` avec sa mesure dans `match-replay/model/`
 * — et c'est le MÊME prédicat qui commande le rendu du bloc et la pose du titre ici. Les deux sections
 * muettes -> un état vide nommé, jamais un onglet blanc.
 */
import { useMemo } from 'react'

import { MatchEquipmentUsageSection } from '@/features/match-replay/MatchEquipmentUsageSection'
import { buildEquipmentUsage, hasEquipmentUsage } from '@/features/match-replay/model/equipmentUsageLogic'
import { DetailSection } from '@/components/ui/detail-section'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { AnalysisPendingSlot } from '@/components/ui/analysis-pending-slot'
import { ANALYSIS_PART, hasPendingPart, type AnalysisPendingCarrier } from '@/lib/analysis-pending/analysisPending.logic'
import type {
  FragDistribution,
  MatchEmpriseBlock,
  MatchKillDistancePlayer,
  MatchLivesNearTeammate,
  MatchPlayerPosition,
  MatchRosterRow,
  MatchScoreboardRow,
  SquadWeaponTools,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useMatchReplay } from '@/lib/replay/queries'
import { MatchFragCard } from './MatchFragCard'
import { MatchKillDistanceSection } from './MatchKillDistanceSection'
import { MatchPositionsHeatmap } from './MatchPositionsHeatmap'
import { arsenalPendingSlots, hasMatchFragData, hasPositions, useHasKillDistanceSection } from './blockPredicates'
import type { MatchViewText } from './i18n'
import { hasEquipmentEmpriseCard } from './matchEmprise.logic'
import { useMatchEmprise } from './useMatchEmprise'

interface Props {
  playerSlug: string
  matchId: string
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[]
  roster: MatchRosterRow[]
  /** Répartition des frags du viewer (classe -> rôle) — absente sur un match sans frags. */
  fragDistribution: FragDistribution | null | undefined
  /** « Outils de destruction » du viewer — absents sans frag. */
  weaponTools: SquadWeaponTools | null | undefined
  /** Distance des frags par arme et par joueur — absente sans positions de film. */
  killDistance: MatchKillDistancePlayer[] | null | undefined
  /** L'Emprise du match (fiches = l'équipe du viewer) — absente sans viewer au tableau. */
  emprise: MatchEmpriseBlock | null | undefined
  /** « Isolement » de chaque joueur de l'équipe — absent sans positions de film. */
  livesNearTeammate: MatchLivesNearTeammate | null | undefined
  /** L'en-tête de la vue (ses causes `analysis_pending_parts`) : chaque section qui dépend du
   *  film montre un emplacement d'attente, au lieu de son état vide, quand SA cause est servie. */
  analysisCauses?: AnalysisPendingCarrier | null
  meXUID: string | null
  friendGamertags: readonly string[]
  matchPositions: MatchPlayerPosition[] | undefined
  locale: Locale
  t: MatchViewText
}

/**
 * useHasEquipmentUsage — le prédicat de l'usage d'équipements, lu à la source : la mesure se rebâtit
 * avec le MÊME constructeur que la carte (`buildEquipmentUsage`) depuis l'artefact déjà en cache.
 */
function useHasEquipmentUsage(playerSlug: string, matchId: string, replayAvailable: boolean, scoreboard: MatchScoreboardRow[]): boolean {
  const { data } = useMatchReplay(playerSlug, matchId, replayAvailable)
  return useMemo(() => (data ? hasEquipmentUsage(buildEquipmentUsage(data, scoreboard)) : false), [data, scoreboard])
}

export function MatchViewTabArsenal({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  roster,
  fragDistribution,
  weaponTools,
  killDistance,
  emprise,
  livesNearTeammate,
  analysisCauses,
  meXUID,
  friendGamertags,
  matchPositions,
  locale,
  t,
}: Props) {
  const equipment = useHasEquipmentUsage(playerSlug, matchId, replayAvailable, scoreboard)
  const e = useMatchEmprise({ emprise, lives: livesNearTeammate, scoreboard, roster, meXUID, friendGamertags, locale })
  // La distance des frags a une porte de TITRE (le titre mesure-t-il les positions ?) : quand
  // elle est ouverte, la section s'affiche même sans donnée pour CE match — elle écrit alors
  // pourquoi elle est vide, et cette phrase est justement ce qu'il faut montrer.
  const showKillDistance = useHasKillDistanceSection()
  const showKillsWeapons = hasMatchFragData(fragDistribution, weaponTools) || showKillDistance
  const showEquipmentTerrain = hasEquipmentEmpriseCard(e.present) || equipment || hasPositions(matchPositions)

  // Analyse du film en cours : l'arme de chaque frag (cause kill_source) et le rejeu 2D
  // (cause replay : Emprise, équipements, positions) s'annoncent par un emplacement d'attente
  // au lieu de l'état vide, chacun pour SA cause.
  const pending = arsenalPendingSlots({
    killSourcePending: hasPendingPart(analysisCauses, ANALYSIS_PART.killSource),
    replayPending: hasPendingPart(analysisCauses, ANALYSIS_PART.replay),
    hasKillDistance: (killDistance?.length ?? 0) > 0,
    hasTerrain: showEquipmentTerrain,
  })
  const showKillsWeaponsSection = showKillsWeapons || pending.killsWeapons
  const showTerrainSection = showEquipmentTerrain || pending.terrain

  if (!showKillsWeaponsSection && !showTerrainSection) {
    // Convention de l'application : l'état vide pointillé vit DANS un bloc parent, jamais nu.
    return (
      <Card data-testid="match-arsenal-empty">
        <CardContent className="pt-4">
          <EmptyStateNotice title={t.arsenalEmptyTitle} description={t.arsenalEmptyDescription} />
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {showKillsWeaponsSection && (
        <DetailSection title={t.sectionKillsWeapons}>
          <MatchFragCard distribution={fragDistribution} tools={weaponTools} locale={locale} />
          {pending.killsWeapons ? (
            <AnalysisPendingSlot label={t.analysisPendingSlotInProgress} />
          ) : (
            <MatchKillDistanceSection players={killDistance} scoreboard={scoreboard} roster={roster} meXUID={meXUID} friendGamertags={friendGamertags} t={t} />
          )}
        </DetailSection>
      )}

      {showTerrainSection && (
        <DetailSection title={t.sectionEquipmentTerrain}>
          {pending.terrain && <AnalysisPendingSlot label={t.analysisPendingSlotPreparing} />}
          {e.control}
          {e.sheets}
          <MatchEquipmentUsageSection playerSlug={playerSlug} matchId={matchId} replayAvailable={replayAvailable} scoreboard={scoreboard} locale={locale} />
          {(e.present.production || e.present.yield) && (
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              {e.production}
              {e.yieldCard}
            </div>
          )}
          {e.lives}
          <MatchPositionsHeatmap playerSlug={playerSlug} matchId={matchId} positions={matchPositions} locale={locale} />
        </DetailSection>
      )}
    </div>
  )
}
