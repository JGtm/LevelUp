/**
 * MatchViewTabControl — contenu de l'onglet « Contrôle » de la page match.
 *
 * Créé le 2026-09-19 (plan `.ai/V7.5/PLAN_AJUSTEMENTS_PRE_V75_2026-09-19.md`, lot 2) : les
 * trois blocs qui disent QUI A TENU QUOI sur le terrain sortent de « Chronologie », où ils
 * s'intercalaient entre deux courbes temporelles sans en être une :
 *   1. « Usages d'équipement » (`MatchEquipmentUsageSection`) — le bilan des gestes, par
 *      joueur et par camp ;
 *   2. « Contrôle des armes spéciales » (`MatchPadControlSection`) — le ramasseur de chaque
 *      socle, même modèle de carte que le bilan d'équipement ;
 *   3. « Occupation du terrain » (`MatchPositionsHeatmap`) — les positions du film sur le
 *      plan du match.
 * Les deux premiers lisent le MÊME artefact de rejeu (une seule clé de cache) ; le troisième
 * lit les positions keyframe, tirées par la page UNIQUEMENT quand cet onglet est actif.
 * Chaque bloc se masque lui-même sans donnée : l'onglet ne pose aucun cadre vide.
 */
import { MatchEquipmentUsageSection } from '@/features/match-replay/MatchEquipmentUsageSection'
import { MatchPadControlSection } from '@/features/match-replay/MatchPadControlSection'
import type { MatchPlayerPosition, MatchScoreboardRow } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { MatchPositionsHeatmap } from './MatchPositionsHeatmap'

interface Props {
  playerSlug: string
  matchId: string
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[]
  matchPositions: MatchPlayerPosition[] | undefined
  locale: Locale
}

export function MatchViewTabControl({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  matchPositions,
  locale,
}: Props) {
  return (
    <div className="space-y-4">
      <MatchEquipmentUsageSection
        playerSlug={playerSlug}
        matchId={matchId}
        replayAvailable={replayAvailable}
        scoreboard={scoreboard}
        locale={locale}
      />
      <MatchPadControlSection
        playerSlug={playerSlug}
        matchId={matchId}
        replayAvailable={replayAvailable}
        scoreboard={scoreboard}
        locale={locale}
      />
      <MatchPositionsHeatmap
        playerSlug={playerSlug}
        matchId={matchId}
        positions={matchPositions}
        locale={locale}
      />
    </div>
  )
}
