/**
 * useReplayTimeline — TOUT CE QUE LA FRISE DEMANDE, assemblé en un objet.
 *
 * TREIZIÈME EXTRACTION IMPOSÉE PAR LE CLIQUET DE TAILLE (`placementFamily.guard.test.ts`) : la
 * barre de lecture de la planche 2a (2026-08-28) fait entrer quatre pistes, leurs échelles, la
 * réduction du fil, les trois horloges de l'axe et le clavier. Posés dans le canvas, c'était
 * une trentaine de lignes — sur un fichier qui n'en a plus une seule de marge. Ils partagent
 * une nature : ils décrivent LA FRISE, pas le dessin de la carte.
 *
 * LE FIL VIENT DE LA PAGE, IL N'EST PAS RECONSTRUIT ICI. `buildFeedEntries` est appelé une fois
 * dans `replay.tsx` et sert le fil de droite ET ces pistes. Un second assemblage aurait son
 * propre recalage d'horloge (origine publiée ou appariement statistique selon l'artefact) : une
 * marque de la frise ne serait alors plus garantie être la ligne qu'on lit à côté. Ce hook ne
 * fait que RÉDUIRE ces entrées à ce que les pistes demandent — un acteur, un instant, une clé.
 *
 * QUI EST L'ACTEUR D'UNE LIGNE : pour une élimination, le TUEUR (`kill.xuid`) — la marque est à
 * lui ; pour une mort, le DÉFUNT (`death.xuid`). Une élimination dont on est la VICTIME est une
 * mort, et c'est la forme que prend la majorité des morts d'un match : la réduction la range
 * donc avec les morts, faute de quoi la piste « Toi » n'en montrerait presque aucune.
 */
import { useCallback, useMemo, type ChangeEvent, type ComponentProps, type RefObject } from 'react'

import { useCapability } from '@/lib/capabilities'
import { formatClockMMSS } from '@/lib/formatters'
import type { LeadChange } from '@/lib/replay/scoreTimeline'

import type { ReplayFeedEntry } from './killFeedLogic'
import type { ReplayLocale } from './i18n'
import type { PlayerMarkKind } from './playerMarks'
import { EMPTY_MEDIA, SKIP_SECONDS } from './replayCanvasConfig'
import type { ReplayTimelineTracks } from './ReplayTimelineTracks'
import {
  buildDominance,
  buildEventTracks,
  placeMedia,
  trackScale,
  type ReplayMediaItem,
  type TrackDeath,
  type TrackKill,
} from './replayTimelineTracksLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { displayClockMs, type ReplayWindowBounds } from './replayWindow'
import { useReplayShortcuts } from './useReplayShortcuts'

/** Ce que le canvas prête à la frise : le document, le cadrage, le fil et la lecture. */
export interface ReplayTimelineOptions {
  doc: ReplayDocumentReady
  playWindow: ReplayWindowBounds | null
  /** Le fil aligné, assemblé UNE fois par la page (cf. l'en-tête). */
  feedEntries: readonly ReplayFeedEntry[]
  /** Marques d'identité par xuid : elles décident de la piste, jamais de la couleur. */
  marks: ReadonlyMap<string, PlayerMarkKind>
  /** Les retournements et les deux cascades d'équipe (cf. `useLeadMarks`). */
  lead: {
    changes: readonly LeadChange[]
    allyOf: (teamId: number) => boolean | null
    labelOf: (teamId: number) => string
  }
  /** La lecture, telle que `useReplayPlayback` la rend. */
  playback: ReplayPlaybackForTimeline
  /**
   * LES MÉDIAS DU MATCH, mappés une fois par la page (`buildReplayMedia`) : même doctrine que
   * le fil. Défaut vide — le canvas peut être monté avant que la vue du match soit là.
   */
  media?: readonly ReplayMediaItem[]
  /** Bascule du son, pour le raccourci « M ». */
  toggleSound: () => void
  /** Largeur de dessin : 0 = pas de rejeu à l'écran, le clavier n'écoute rien. */
  renderWidth: number
  locale: ReplayLocale
}

/** La part de `ReplayPlayback` dont la frise et le clavier ont besoin. */
export interface ReplayPlaybackForTimeline {
  sliderRef: RefObject<HTMLInputElement | null>
  startFrame: number
  endFrame: number
  onScrub: (e: ChangeEvent<HTMLInputElement>) => void
  playing: boolean
  togglePlay: () => void
  restart: () => void
  seekBy: (seconds: number) => void
  stepFrames: (frames: number) => void
}

/** L'objet unique que le canvas repasse tel quel à la barre (patron de `ReplaySound`). */
export type ReplayTimeline = ComponentProps<typeof ReplayTimelineTracks>

export function useReplayTimeline(o: ReplayTimelineOptions): ReplayTimeline {
  const { doc, playWindow, feedEntries, marks, lead, playback, toggleSound, renderWidth, locale } = o
  const { media: mediaItems = EMPTY_MEDIA } = o
  const { frameIntervalMs, frameCount } = doc
  // LA RANGÉE MÉDIAS EST UNE AFFAIRE DE TITRE, pas de match : le rejeu n'est gardé que par
  // `matchmaking`, et un titre sans médias afficherait sinon une piste éternellement vide —
  // qui se lirait « aucun média sur ce match » au lieu de « ce jeu n'en a pas ».
  const showMediaTrack = useCapability('media')

  const scale = useMemo(() => trackScale(playWindow, frameCount), [playWindow, frameCount])
  // L'HORLOGE AFFICHÉE, celle du GAMEPLAY (D-A2) : la même règle que le fil, le bandeau et les
  // infobulles — le coup d'envoi se lit 0:00, le countdown ne se compte pas.
  const clockOf = useCallback(
    (replayMs: number) => formatClockMMSS(displayClockMs(replayMs, playWindow)),
    [playWindow],
  )
  const { kills, deaths } = useMemo(() => reduceFeed(feedEntries, marks), [feedEntries, marks])
  const tracks = useMemo(
    () => buildEventTracks(kills, deaths, marks, frameIntervalMs ?? 0, scale, clockOf),
    [kills, deaths, marks, frameIntervalMs, scale, clockOf],
  )
  const dominance = useMemo(() => buildDominance(lead.changes, scale), [lead.changes, scale])
  // LES MÉDIAS ARRIVENT DE LA PAGE, déjà sur l'axe du rejeu (phase 2, 2026-08-28) : ce hook ne
  // fait que les POSER sur l'échelle de la frise, comme il pose les marques du fil. Le recalage,
  // lui, a eu lieu une seule fois dans `buildReplayMedia`.
  const media = useMemo(
    () => placeMedia(mediaItems, frameIntervalMs ?? 0, scale),
    [mediaItems, frameIntervalMs, scale],
  )

  const { startClock, midClock, endClock } = useMemo(
    () => axisClocks(scale.from, scale.span, frameIntervalMs, clockOf),
    [scale.from, scale.span, frameIntervalMs, clockOf],
  )

  useReplayShortcuts({
    togglePlay: playback.togglePlay,
    seekBy: playback.seekBy,
    stepFrames: playback.stepFrames,
    restart: playback.restart,
    toggleSound,
    skipSeconds: SKIP_SECONDS,
    enabled: renderWidth > 0,
  })

  return {
    sliderRef: playback.sliderRef,
    minFrame: playback.startFrame,
    maxFrame: playback.endFrame,
    onScrub: playback.onScrub,
    own: tracks.own,
    allies: tracks.allies,
    dominance,
    allyOf: lead.allyOf,
    labelOf: lead.labelOf,
    media,
    showMediaTrack,
    playing: playback.playing,
    // OUVRIR UN MÉDIA MET LE REJEU EN PAUSE : la frise n'appelle ceci que lorsque la lecture
    // tourne, donc la bascule vaut « pause » — jamais un redémarrage inattendu.
    onRequestPause: playback.togglePlay,
    startClock,
    midClock,
    endClock,
    locale,
  }
}

/**
 * reduceFeed ramène le fil à ce que les pistes demandent. Les MÉDAILLES SEULES n'y entrent pas :
 * elles n'ont ni tueur ni défunt, et une piste d'événements dit qui a marqué ou qui est tombé.
 *
 * EXPORTÉ POUR ÊTRE TESTÉ (revue R1), pas pour être appelé d'ailleurs : c'est ici que se décide
 * À QUI appartient une ligne, et une inversion tueur/victime y serait invisible à la relecture
 * comme à l'écran — les deux pistes resteraient peuplées, avec les mauvais événements.
 */
export function reduceFeed(
  entries: readonly ReplayFeedEntry[],
  marks: ReadonlyMap<string, PlayerMarkKind>,
): { kills: TrackKill[]; deaths: TrackDeath[] } {
  const kills: TrackKill[] = []
  const deaths: TrackDeath[] = []
  for (const entry of entries) {
    if (entry.death) {
      deaths.push({ key: entry.key, replayMs: entry.replayMs, xuid: entry.death.xuid })
      continue
    }
    const kill = entry.kill
    if (!kill) continue
    kills.push({ key: entry.key, replayMs: entry.replayMs, xuid: kill.xuid })
    // LA MÊME LIGNE PEUT ÊTRE LES DEUX : le frag de l'un est la mort de l'autre. On ne la range
    // du côté des morts que si la victime porte une marque — sinon `buildEventTracks` l'écarte
    // de toute façon, et la clé dérivée ne servirait à rien.
    if (kill.victimXuid && marks.get(kill.victimXuid) === 'me') {
      deaths.push({ key: `${entry.key}-v`, replayMs: entry.replayMs, xuid: kill.victimXuid })
    }
  }
  return { kills, deaths }
}

/**
 * axisClocks date les trois repères sous la frise : début, milieu, fin de la FENÊTRE. Sans
 * échelle temporelle (artefact sans axe T réel), les trois restent vides plutôt que d'afficher
 * une durée fabriquée à partir d'un index d'images.
 */
function axisClocks(
  fromFrame: number,
  span: number,
  frameIntervalMs: number | undefined,
  clockOf: (replayMs: number) => string,
): { startClock: string; midClock: string; endClock: string } {
  if (!frameIntervalMs || span <= 0) return { startClock: '', midClock: '', endClock: '' }
  const msOf = (frame: number) => clockOf(frame * frameIntervalMs)
  return {
    startClock: msOf(fromFrame),
    midClock: msOf(fromFrame + span / 2),
    endClock: msOf(fromFrame + span),
  }
}
