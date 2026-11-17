/**
 * ReplayStage — L'UNIQUE MONTAGE DE LA TOILE et de sa barre de lecture, posé dans la zone `scene` du
 * gabarit des cadres. Jamais démonté par une bascule de cadre : c'est l'instance de `ReplayCanvas` qui
 * porte la position de lecture, le son, le zoom, l'export en cours et les calques cuits.
 *
 * Ce que la scène contient, dans l'ordre : l'avis « instant non calé » (le lien tactique demandait un
 * instant que ce rejeu ne peut pas caler : il s'ouvre au début, jamais sur un saut approximatif), le
 * bandeau de score quand le cadre le pose ici (`scorePlacement` ; le Standard le porte dans son
 * en-tête et passe `null`), puis la toile et ses surcouches.
 *
 * LES TROIS SURCOUCHES SE POSENT SUR LA CARTE, PAS SUR LE LECTEUR : l'écran de fin, le message
 * inter-manche et le compte à rebours de la bombe descendent dans le conteneur de la toile (`mapOverlays`),
 * seul endroit dont les bornes sont celles du terrain. Montées en sœurs de la toile, leur `inset-0`
 * couvrirait le bandeau de score, la carte ET la barre de lecture.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'

import { REPLAY_TEXT } from '../../i18n/i18n'
import type { EmplacementScore } from '../../model/cotesBandeauScore'
import { ReplayBombCountdownOverlay } from '../ReplayBombCountdownOverlay'
import { ReplayCanvas } from '../ReplayCanvas'
import { ReplayRoundBreakOverlay } from '../ReplayRoundBreakOverlay'
import { ReplayScoreBanner } from '../ReplayScoreBanner'
import { ReplayVictoryOverlay } from '../ReplayVictoryOverlay'
import type { ReplayFrameView } from './replayFrameTypes'

/** Les trois surcouches de la carte : l'écran de fin, le message inter-manche, le compte à rebours de la bombe. */
function StageMapOverlays({ view }: { view: ReplayFrameView }) {
  const { doc, model, header, playFrame, locale } = view
  return (
    <>
      <ReplayVictoryOverlay
        doc={doc}
        scoreboard={model.scoreboard}
        allegiance={model.allegiance}
        outcomeCode={header?.outcome_code}
        viewpoint={view.viewpoint.xuid}
        finalScore={model.score}
        playWindow={model.window}
        frame={playFrame}
        titleSlug={view.titleSlug}
        locale={locale}
      />
      <ReplayRoundBreakOverlay doc={doc} frame={playFrame} locale={locale} />
      <ReplayBombCountdownOverlay doc={doc} frame={playFrame} locale={locale} />
    </>
  )
}

export function ReplayStage({ view, scorePlacement }: { view: ReplayFrameView; scorePlacement: EmplacementScore | null }) {
  const { doc, model, header, locale } = view
  const t = REPLAY_TEXT[locale]
  return (
    <>
      {view.showMissingOriginNotice && (
        <EmptyStateNotice className="mb-2" title={t.openAtNoOriginTitle} description={t.openAtNoOriginDescription} />
      )}
      {/* `null` : le cadre pose le score ailleurs (cellule centrale de l'en-tête du Standard). La place
          de l'élément dans le fragment ne change pas, donc la toile qui suit n'est jamais remontée. */}
      {scorePlacement && (
        <ReplayScoreBanner
          doc={doc}
          allegiance={model.allegiance}
          frame={view.playFrame}
          nowMs={view.nowMs}
          playWindow={model.window}
          locale={locale}
          placement={scorePlacement}
        />
      )}
      <ReplayCanvas
        doc={doc}
        locale={locale}
        playWindow={model.window}
        playbackStore={view.playbackStore}
        openAtFrame={view.openAtFrame}
        background={view.mapBackground}
        mapId={header?.map_id}
        callouts={view.callouts}
        playerSlug={view.playerSlug}
        matchId={view.matchId}
        scoreboard={model.scoreboard}
        allegiance={model.allegiance}
        marks={model.marks}
        endMatch={view.endMatchSound}
        viewpoint={view.viewpoint.xuid}
        playerFocus={view.playerPanel.focus}
        onFollowStop={view.playerPanel.onFollowStop}
        fullscreen={view.fullscreen}
        frameControl={view.frameControl}
        players={model.players}
        onSelectViewpoint={view.viewpoint.select}
        // LE VERDICT, POUR L'EXPORT SEUL : l'écran de fin est du DOM, qu'aucun encodeur vidéo ne voit.
        // L'export le repeint dans la toile à partir du CODE seul ; le mot se résout des deux côtés
        // depuis les mappings du titre, sur l'issue lue.
        outcome={{ code: header?.outcome_code, finalScore: model.score }}
        feedEntries={model.feed}
        media={model.media}
        scoreTimelineKind={header ? (header.score_timeline_kind ?? '') : undefined}
        mapOverlays={<StageMapOverlays view={view} />}
      />
    </>
  )
}
