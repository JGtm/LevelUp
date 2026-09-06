/**
 * useReplayTimeline — TOUT CE QUE LA FRISE DEMANDE, assemblé en un objet.
 *
 * TREIZIÈME EXTRACTION IMPOSÉE PAR LE SEUIL DE TAILLE (`max-lines` eslint, R5) : la
 * barre de lecture de la planche 2a (2026-08-28) fait entrer quatre pistes, leurs échelles, la
 * réduction du fil, les trois horloges de l'axe et le clavier. Posés dans le canvas, c'était
 * une trentaine de lignes — sur un fichier qui n'en a plus une seule de marge. Ils partagent
 * une nature : ils décrivent LA FRISE, pas le dessin de la carte.
 *
 * LE FIL VIENT DE LA PAGE, IL N'EST PAS RECONSTRUIT ICI. `buildFeedEntries` est appelé une fois
 * dans `replay.tsx` et sert le fil de droite ET ces pistes. Un second assemblage aurait son
 * propre recalage d'horloge (origine publiée ou appariement statistique selon l'artefact) : une
 * marque de la frise ne serait alors plus garantie être la ligne qu'on lit à côté. Ce hook ne
 * fait que RÉDUIRE ces entrées à ce que les pistes demandent — un acteur, un instant, une clé,
 * et le CAMP du tueur depuis que la dominance se compte en frags (2026-08-28).
 *
 * QUI EST L'ACTEUR D'UNE LIGNE : pour une élimination, le TUEUR (`kill.xuid`) — la marque est à
 * lui ; pour une mort, le DÉFUNT (`death.xuid`). Une élimination dont on est la VICTIME est une
 * mort, et c'est la forme que prend la majorité des morts d'un match : la réduction la range
 * donc avec les morts, faute de quoi la piste « Toi » n'en montrerait presque aucune.
 */
import { useCallback, useMemo, type ChangeEvent, type ComponentProps, type RefObject } from 'react'

import { useCapability } from '@/lib/capabilities'
import { leaderStates, scoreTimelineOf } from '@/lib/replay/scoreTimeline'

import type { ReplayFeedEntry } from './killFeedLogic'
import type { ReplayLocale } from './i18n/i18n'
import type { PlayerMarkKind } from './playerMarks'
import { formatClock } from './replayLogic'
import { EMPTY_MEDIA, SKIP_SECONDS } from './replayCanvasConfig'
import type { ReplayTimelineTracks } from './ReplayTimelineTracks'
import {
  buildEventTracks,
  buildFragDominance,
  buildScoreDominance,
  placeMedia,
  roundSeparators,
  sameLeadSegments,
  trackScale,
  type DominanceSegment,
  type ReplayMediaItem,
  type ReplayScoreTrack,
  type TrackDeath,
  type TrackFrag,
  type TrackKill,
  type TrackScale,
} from './replayTimelineTracksLogic'
import { roundTransitions } from './roundsLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { displayClockMs, type ReplayWindowBounds } from './replayWindow'
import { usePersistedFlag, TIMELINE_EXPANDED_KEY } from './settings/useReplaySettings'
import { useReplayShortcuts, type ReplayShortcutHandlers } from './useReplayShortcuts'

/** Ce que le canvas prête à la frise : le document, le cadrage, le fil et la lecture. */
export interface ReplayTimelineOptions {
  doc: ReplayDocumentReady
  playWindow: ReplayWindowBounds | null
  /** Le fil aligné, assemblé UNE fois par la page (cf. l'en-tête). */
  feedEntries: readonly ReplayFeedEntry[]
  /** Marques d'identité par xuid : elles décident de la piste, jamais de la couleur. */
  marks: ReadonlyMap<string, PlayerMarkKind>
  /** Les deux cascades d'équipe de la piste Dominance (cf. `useTeamCascades`). */
  lead: {
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
  /** Le cadrage, relaye tel quel aux raccourcis clavier (cf. useReplayShortcuts). */
  zoom?: ReplayShortcutHandlers['zoom']
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
/**
 * TOUT CE QUE LA FRISE DEMANDE, SAUF L'HORLOGE. Depuis que le temps s'affiche sous le curseur
 * (2026-09-02), la frise reçoit aussi `clockRef` — mais cette référence appartient au CANVAS
 * (`useReplayClock`), pas à ce hook : elle traverse la barre de lecture et se greffe au dernier
 * moment. L'exclure ici est ce qui empêche ce hook de prétendre la fournir.
 */
export type ReplayTimeline = Omit<ComponentProps<typeof ReplayTimelineTracks>, 'clockRef'>

export function useReplayTimeline(o: ReplayTimelineOptions): ReplayTimeline {
  const { doc, playWindow, feedEntries, marks, lead, playback, toggleSound, renderWidth, locale, zoom } = o
  const { media: mediaItems = EMPTY_MEDIA } = o
  const { frameIntervalMs, frameCount } = doc
  // LE REPLI EST UNE PRÉFÉRENCE DU LECTEUR, pas un calque : il ne passe pas par le tiroir mais
  // par un chevron sur la frise. Persisté (patron des autres réglages), DÉPLIÉ par défaut — la
  // frise à pistes est ce que le lot précédent a livré, on ne la cache pas d'office.
  const [tracksExpanded, toggleTracks] = usePersistedFlag(TIMELINE_EXPANDED_KEY, true)
  // LA RANGÉE MÉDIAS EST UNE AFFAIRE DE TITRE, pas de match : le rejeu n'est gardé que par
  // `matchmaking`, et un titre sans médias afficherait sinon une piste éternellement vide —
  // qui se lirait « aucun média sur ce match » au lieu de « ce jeu n'en a pas ».
  const showMediaTrack = useCapability('media')

  const scale = useMemo(() => trackScale(playWindow, frameCount), [playWindow, frameCount])
  // L'HORLOGE AFFICHÉE, celle du GAMEPLAY (D-A2) : la même règle que le fil, le bandeau et les
  // infobulles — le coup d'envoi se lit 0:00, le countdown ne se compte pas.
  //
  // ET C'EST LE MÊME FORMATEUR QUE TOUT LE REJEU (résidu P0-6, 2026-09-05). Cette ligne
  // appelait le formateur d'INSTANT de `lib/formatters`, qui ARRONDIT à la seconde là où
  // l'horloge du lecteur, le fil et les infobulles TRONQUENT (`replayLogic.formatClock`) :
  // deux écritures du même instant sur le même écran, à une seconde d'écart, sur les marques
  // de la frise. Une seule subsiste dans cette feature — garde-rail
  // `replayClockFormat.guard.test.ts`.
  const clockOf = useCallback(
    (replayMs: number) => formatClock(displayClockMs(replayMs, playWindow)),
    [playWindow],
  )
  const { kills, deaths, frags } = useMemo(() => reduceFeed(feedEntries, marks), [feedEntries, marks])
  const tracks = useMemo(
    () => buildEventTracks(kills, deaths, marks, frameIntervalMs ?? 0, scale, clockOf),
    [kills, deaths, marks, frameIntervalMs, scale, clockOf],
  )
  // LA DOMINANCE SE LIT SUR LES FRAGS (2026-08-28), plus sur le compteur du mode : elle vient
  // donc du MÊME fil que les deux pistes du dessus, jamais d'un second calque.
  const dominance = useMemo(
    () => buildFragDominance(frags, frameIntervalMs ?? 0, scale),
    [frags, frameIntervalMs, scale],
  )
  const score = useMemo(() => scoreTrack(doc, dominance, scale), [doc, dominance, scale])
  // LES MÉDIAS ARRIVENT DE LA PAGE, déjà sur l'axe du rejeu (phase 2, 2026-08-28) : ce hook ne
  // fait que les POSER sur l'échelle de la frise, comme il pose les marques du fil. Le recalage,
  // lui, a eu lieu une seule fois dans `buildReplayMedia`.
  const media = useMemo(
    () => placeMedia(mediaItems, frameIntervalMs ?? 0, scale),
    [mediaItems, frameIntervalMs, scale],
  )

  useReplayShortcuts({
    togglePlay: playback.togglePlay,
    seekBy: playback.seekBy,
    stepFrames: playback.stepFrames,
    restart: playback.restart,
    toggleSound,
    skipSeconds: SKIP_SECONDS,
    enabled: renderWidth > 0,
    zoom,
  })

  return {
    sliderRef: playback.sliderRef,
    minFrame: playback.startFrame,
    maxFrame: playback.endFrame,
    onScrub: playback.onScrub,
    own: tracks.own,
    allies: tracks.allies,
    dominance,
    score,
    allyOf: lead.allyOf,
    labelOf: lead.labelOf,
    media,
    showMediaTrack,
    tracksExpanded,
    onToggleTracks: toggleTracks,
    playing: playback.playing,
    // OUVRIR UN MÉDIA MET LE REJEU EN PAUSE : la frise n'appelle ceci que lorsque la lecture
    // tourne, donc la bascule vaut « pause » — jamais un redémarrage inattendu.
    onRequestPause: playback.togglePlay,
    locale,
  }
}

/**
 * scoreTrack assemble la piste SCORE, ou rend `null` quand elle n'a rien à dire.
 *
 * TROIS RAISONS DE NE PAS L'AFFICHER, et elles disent toutes la même chose — « cette piste
 * répéterait ce qui est déjà à l'écran, ou n'est pas mesurée » :
 *  1. AUCUN CALQUE DE SCORE exploitable : artefact antérieur au schéma 12, ou horloge du film
 *     non recalée (`scoreTimelineOf` porte cette garde et rend `undefined`).
 *  2. LA PISTE SERAIT LE SOSIE DE LA DOMINANCE (Slayer : le score EST le compte des frags) —
 *     mêmes meneurs, mêmes frontières au pixel près (`sameLeadSegments`, décision user D1
 *     2026-09-02). L'ancienne garde comparait les totaux à l'égalité stricte et un seul kill
 *     non attribué réaffichait le doublon.
 *  3. MOINS DE DEUX CAMPS IDENTIFIÉS : `buildScoreDominance` rend alors une liste vide, et une
 *     rangée vide se lirait « personne n'a marqué » au lieu de « on ne sait pas ».
 *
 * LES SÉPARATEURS DE MANCHE viennent de `roundTransitions` — le foyer des pastilles du bandeau
 * et de l'écran inter-manche. Aucun sur un mode à manche unique, par construction.
 */
function scoreTrack(
  doc: ReplayDocumentReady,
  dominance: readonly DominanceSegment[],
  scale: TrackScale,
): ReplayScoreTrack | null {
  const timeline = scoreTimelineOf(doc)
  if (!timeline) return null
  const segments = buildScoreDominance(leaderStates(timeline), scale)
  if (segments.length === 0) return null
  if (sameLeadSegments(segments, dominance)) return null
  return { segments, rounds: roundSeparators(roundTransitions(timeline), scale) }
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
): { kills: TrackKill[]; deaths: TrackDeath[]; frags: TrackFrag[] } {
  const kills: TrackKill[] = []
  const deaths: TrackDeath[] = []
  const frags: TrackFrag[] = []
  for (const entry of entries) {
    if (entry.death) {
      deaths.push({ key: entry.key, replayMs: entry.replayMs, xuid: entry.death.xuid })
      continue
    }
    const kill = entry.kill
    if (!kill) continue
    kills.push({ key: entry.key, replayMs: entry.replayMs, xuid: kill.xuid })
    // LES FRAGS COMPTENT TOUTE LA SALLE, pas seulement les joueurs marqués : la dominance
    // oppose deux CAMPS. Un tueur dont le camp n'est pas résolu (acteur hors scoreboard) ne
    // compte pour personne — l'attribuer par défaut fausserait le meneur.
    if (kill.teamID != null) frags.push({ replayMs: entry.replayMs, teamId: kill.teamID })
    // LA MÊME LIGNE PEUT ÊTRE LES DEUX : le frag de l'un est la mort de l'autre. On ne la range
    // du côté des morts que si la victime porte une marque — sinon `buildEventTracks` l'écarte
    // de toute façon, et la clé dérivée ne servirait à rien.
    if (kill.victimXuid && marks.get(kill.victimXuid) === 'me') {
      deaths.push({ key: `${entry.key}-v`, replayMs: entry.replayMs, xuid: kill.victimXuid })
    }
  }
  return { kills, deaths, frags }
}

