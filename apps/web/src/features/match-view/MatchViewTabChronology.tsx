/**
 * MatchViewTabChronology — contenu de l'onglet « Chronologie » de la page match.
 *
 * Extrait de MatchViewPage.tsx au passage à 3 onglets (Général / Chronologie /
 * Joueurs, 2026-08-24) : les blocs sont DÉPLACÉS tels quels depuis la sous-section
 * « Déroulé du match » de l'ancien onglet Détails, dont le sous-titre est conservé.
 */
import { EngagementMatchSection } from '@/features/engagement/EngagementMatchSection'
import { MatchEquipmentUsageSection } from '@/features/match-replay/MatchEquipmentUsageSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import type {
  MatchHighlightEvent,
  MatchImpactBadge,
  MatchObjectiveEvent,
  MatchPlayerPosition,
  MatchScoreboardRow,
  MatchTugOfWarBin,
  MatchViewCadence,
} from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { DetailSection } from './DetailSection'
import { MatchCadenceChart } from './MatchCadenceChart'
import { MatchImpactBadgesBar } from './MatchImpactBadgesBar'
import { MatchKDCumulChart } from './MatchKDCumulChart'
import { MatchPositionsHeatmap } from './MatchPositionsHeatmap'
import { MatchScoreCurveChart } from './MatchScoreCurveChart'
import { MatchTugOfWarChart } from './MatchTugOfWarChart'
import type { MatchViewText } from './i18n'

interface Props {
  playerSlug: string
  matchId: string
  replayAvailable: boolean
  impactBadges: MatchImpactBadge[]
  highlightEvents: MatchHighlightEvent[]
  scoreboard: MatchScoreboardRow[]
  meXUID: string | null
  objectiveEvents: MatchObjectiveEvent[] | undefined
  matchPositions: MatchPlayerPosition[] | undefined
  tugOfWar: MatchTugOfWarBin[]
  cadence: MatchViewCadence | null | undefined
  locale: Locale
  t: MatchViewText
}

export function MatchViewTabChronology({
  playerSlug,
  matchId,
  replayAvailable,
  impactBadges,
  highlightEvents,
  scoreboard,
  meXUID,
  objectiveEvents,
  matchPositions,
  tugOfWar,
  cadence,
  locale,
  t,
}: Props) {
  return (
    /* §1 — Déroulé du match (lecture chronologique) */
    <DetailSection title={t.sectionFlow}>
      {/* Faits marquants | Frags cumulés */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[180px_1fr]">
        <MatchImpactBadgesBar
          badges={impactBadges}
          scoreboard={scoreboard}
          t={t}
          playerSlug={playerSlug}
          matchId={matchId}
          replayAvailable={replayAvailable}
        />
        <MatchKDCumulChart
          events={highlightEvents}
          badges={impactBadges}
          scoreboard={scoreboard}
          meXUID={meXUID}
          objectiveEvents={objectiveEvents}
          t={t}
        />
      </div>

      {/* Le SCORE DANS LE TEMPS (film) ouvre le déroulé : c'est le fil du match.
          Sans artefact il ne rend rien, et la mise en page ne bouge pas. */}
      <MatchScoreCurveChart
        playerSlug={playerSlug}
        matchId={matchId}
        replayAvailable={replayAvailable}
        scoreboard={scoreboard}
        meXUID={meXUID}
        t={t}
      />

      {/* Le BILAN d'équipement du match (film), juste après la courbe : même artefact, même
          clé de cache, aucun appel de plus. Le rejeu montre ces gestes image par image ; ce
          tableau les compte. Sans artefact ou sans grandeur mesurée, il ne rend rien. */}
      <MatchEquipmentUsageSection
        playerSlug={playerSlug}
        matchId={matchId}
        replayAvailable={replayAvailable}
        scoreboard={scoreboard}
        locale={locale}
      />

      {/* Dominance | Cadence des frags */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <MatchTugOfWarChart
          bins={tugOfWar}
          events={highlightEvents}
          scoreboard={scoreboard}
          meXUID={meXUID}
          objectiveEvents={objectiveEvents}
          t={t}
        />
        <MatchCadenceChart
          cadence={cadence}
          scoreboard={scoreboard}
          meXUID={meXUID}
          t={t}
        />
      </div>

      {/* Heatmap positions (film keyframe, match-level §N). Le composant
          se masque lui-même si aucune position n'a été décodée — titre
          sans film, ou match non backfillé (503). */}
      <MatchPositionsHeatmap
        positions={matchPositions}
        locale={locale}
      />

      {/* Engagement — remonté ici (avant Frags différentiel cumulé).
          Gaté sur `engagement` : évite le fetch + la carte placeholder
          pour un titre sans score d'engagement intra-match. */}
      <FeatureGate capability="engagement">
        <EngagementMatchSection
          playerSlug={playerSlug}
          matchId={matchId}
          granularity="intra"
          emptyBehavior="placeholder"
        />
      </FeatureGate>
    </DetailSection>
  )
}
