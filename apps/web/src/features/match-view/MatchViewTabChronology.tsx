/**
 * MatchViewTabChronology — contenu de l'onglet « Chronologie » de la page match.
 *
 * Extrait de MatchViewPage.tsx au passage à 3 onglets (Général / Chronologie /
 * Joueurs, 2026-08-24) : les blocs sont DÉPLACÉS tels quels depuis la sous-section
 * « Déroulé du match » de l'ancien onglet Détails, dont le sous-titre est conservé.
 */
import { EngagementMatchSection } from '@/features/engagement/EngagementMatchSection'
import { MatchEquipmentUsageSection } from '@/features/match-replay/MatchEquipmentUsageSection'
import { MatchPadControlSection } from '@/features/match-replay/MatchPadControlSection'
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
import { MatchScoreEventsChart } from './MatchScoreEventsChart'
import { MatchTugOfWarChart } from './MatchTugOfWarChart'
import type { MatchViewText } from './i18n'
import { SCORE_TIMELINE_EVENTS } from './scoreTimelineKind'

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
  /**
   * `header.score_timeline_kind` — la lecture du bloc « Score dans le temps » décidée par
   * la DONNÉE du titre : rien (le mode marque au frag), des barres aux instants de marque,
   * ou la courbe. Absent = la courbe, le repli sûr.
   */
  scoreTimelineKind?: string
  /**
   * `header.t0_ms` — le countdown d'avant-match, en ms. IL FAIT L'AXE COMMUN DE CET ONGLET :
   * « Frags cumulés » date ses points en `event_time_ms`, dont le zéro est le coup d'envoi
   * PARCE QUE le serveur a retranché ce countdown ; le bloc « Score dans le temps », lui,
   * vient du film, qui compte depuis son premier paquet de position. Sans cette valeur, les
   * deux blocs empilés l'un sous l'autre nommeraient « 0m00s » deux instants distants de
   * −24 à +4,5 s (registre 2026-09-05, P0-7). La conversion vit dans `lib/replay/matchClock`.
   */
  t0Ms?: number
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
  scoreTimelineKind,
  t0Ms,
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
          Sans artefact il ne rend rien, et la mise en page ne bouge pas.

          DEUX LECTURES, ET C'EST LA DONNÉE QUI TRANCHE (`header.score_timeline_kind`,
          regulation.toml [score_timeline]) : les modes qui marquent en trois à cinq fois
          (drapeau, colline, bombe) prennent les BARRES d'instants — une courbe y serait un
          escalier vide ; les autres gardent la courbe. En Slayer, la courbe s'efface d'
          elle-même : « Frags cumulés » vient de le dire, juste au-dessus.

          LES DEUX LECTURES REÇOIVENT `t0Ms`, ET C'EST CE QUI LES MET SUR L'AXE DU BLOC
          CI-DESSUS : le film compte depuis son premier paquet de position, les frags depuis
          le coup d'envoi (cf. la prop). */}
      {scoreTimelineKind === SCORE_TIMELINE_EVENTS ? (
        <MatchScoreEventsChart
          playerSlug={playerSlug}
          matchId={matchId}
          replayAvailable={replayAvailable}
          scoreboard={scoreboard}
          meXUID={meXUID}
          t0Ms={t0Ms}
          t={t}
        />
      ) : (
        <MatchScoreCurveChart
          playerSlug={playerSlug}
          matchId={matchId}
          replayAvailable={replayAvailable}
          scoreboard={scoreboard}
          meXUID={meXUID}
          t0Ms={t0Ms}
          scoreTimelineKind={scoreTimelineKind}
          t={t}
        />
      )}

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

      {/* Le CONTRÔLE DES ARMES SPÉCIALES (film), juste après le bilan d'équipement dont il est
          le complément : celui-ci compte les socles vidés SANS ramasseur, celui-là les nomme
          (padPickups[].xuid, schéma 30). Même artefact, même clé de cache, aucun appel de
          plus. Sans artefact, sans socle ou sans prise attribuée, il ne rend rien. */}
      <MatchPadControlSection
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
