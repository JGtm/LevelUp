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

import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { formatClock } from '../../../lib/replay/replayLogic'
import { EMPTY_MEDIA, SKIP_SECONDS } from '../layers/replayCanvasConfig'
import type { ReplayTimelineTracks } from '../ui/ReplayTimelineTracks'
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
  type TrackMedal,
  type TrackScale,
} from '../model/replayTimelineTracksLogic'
import { presenceShades, teammatesAbsence } from '../model/presenceTrackLogic'
import { roundTransitions } from '../model/roundsLogic'
import { buildViewpointOptions } from '../model/viewpointOptions'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { displayClockMs, type ReplayWindowBounds } from '../model/replayWindow'
import { usePersistedFlag, TIMELINE_EXPANDED_KEY } from '../settings/useReplaySettings'
import { useReplayShortcuts, type ReplayShortcutHandlers } from './useReplayShortcuts'

/** Ce que le canvas prête à la frise : le document, le cadrage, le fil et la lecture. */
export interface ReplayTimelineOptions {
  doc: ReplayDocumentReady
  playWindow: ReplayWindowBounds | null
  /** Le fil aligné, assemblé UNE fois par la page (cf. l'en-tête). */
  feedEntries: readonly ReplayFeedEntry[]
  /**
   * Marques d'identité par xuid. ELLES NE DÉCIDENT PLUS DE LA PISTE (2026-09-07, décision 5) :
   * la seconde piste est celle des COÉQUIPIERS du point de vue, et non plus celle des amis. Ce
   * qui reste des marques sur la frise, c'est la FORME — un ami prend le losange (décision 4).
   */
  marks: ReadonlyMap<string, PlayerMarkKind>
  /**
   * PAR LES YEUX DE QUI (2026-09-07) : le joueur dont la piste du haut porte les kills ET les
   * morts, et dont le camp définit « coéquipier ». Résolu une fois par la page
   * (`model.viewpoint`), relayé par le canvas — jamais redécouvert ici.
   */
  viewpoint: string | null
  /** Camp de chaque xuid RELATIF au point de vue (`model.identity`) : qui est coéquipier. */
  identity: ReadonlyMap<string, { ally: boolean }>
  /** Le roster joint du match (`model.players`) : ce que le menu de point de vue propose. */
  players: readonly ReplayPlayer[]
  /** Poser le point de vue depuis le menu. `null` revient au joueur de la page. */
  onSelectViewpoint: (xuid: string | null) => void
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
  /** Aller à une image précise, sans mettre en pause : le clic sur un repère de présence. */
  seekToFrame: (frame: number) => void
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
  const { media: mediaItems = EMPTY_MEDIA, viewpoint, identity, players, onSelectViewpoint } = o
  const t = REPLAY_TEXT[locale]
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
  const reduit = useMemo(() => reduceFeed(feedEntries, viewpoint), [feedEntries, viewpoint])
  // LES TROIS LECTURES D'IDENTITÉ VOYAGENT ENSEMBLE (cf. `TrackAudience`) : qui on regarde, qui
  // est de son camp, qui est un ami. Groupées, elles n'allongent pas la liste d'arguments et se
  // mémoïsent d'un bloc — les trois changent en même temps, à chaque bascule de point de vue.
  const audience = useMemo(
    () => ({ viewpoint, identity, marks }),
    [viewpoint, identity, marks],
  )
  const tracks = useMemo(
    () => buildEventTracks(reduit, audience, frameIntervalMs ?? 0, scale, clockOf),
    [reduit, audience, frameIntervalMs, scale, clockOf],
  )
  // L'OMBRAGE DE PRÉSENCE (2026-09-07, lot L4) LIT LE MÊME FIL QUE LES MARQUES, et c'est tout
  // ce qui garantit qu'une porte tombe à l'endroit où le fil dit « a rejoint ». Les lignes de
  // présence y sont déjà fusionnées (`mergeFeedWithPresence`), déjà recalées sur l'axe du rejeu.
  //
  // QUAND LE FIL N'EN PORTE AUCUNE, IL N'Y A PAS D'OMBRE, ET C'EST VOULU : `presenceEntries`
  // rend `[]` sans fenêtre de gameplay ni horloge établie (même porte que les lignes du fil).
  // L'ombrage est alors ABSENT plutôt que FAUX — posé sur un axe non recalé, il se tromperait de
  // 3,6 à 50,8 s. Rien à réparer ici : la dégradation a lieu en amont, une fois.
  const shades = useMemo(
    () => presenceShades(feedEntries, viewpoint, frameIntervalMs ?? 0, scale, clockOf),
    [feedEntries, viewpoint, frameIntervalMs, scale, clockOf],
  )
  // L'EFFECTIF DE RÉFÉRENCE DE LA PISTE COÉQUIPIERS : les alliés du point de vue, lui-même
  // exclu. `identity` est déjà RELATIVE au point de vue (cf. `resolveXuidMeta` à trois
  // arguments) — « allié » y veut dire « du côté de celui qu'on regarde », jamais du joueur de
  // la page.
  const teammateXuids = useMemo(() => {
    const out: string[] = []
    for (const [xuid, meta] of identity) {
      if (meta.ally && xuid !== viewpoint) out.push(xuid)
    }
    return out
  }, [identity, viewpoint])
  const absence = useMemo(
    () => teammatesAbsence(feedEntries, teammateXuids, frameIntervalMs ?? 0, scale),
    [feedEntries, teammateXuids, frameIntervalMs, scale],
  )
  // LE MENU DE POINT DE VUE : les sections viennent du roster joint, les trois libellés de
  // l'i18n de la feature. La règle de valeur (le piège des bots) et la règle d'inertie (un
  // joueur sans ligne de tableau de score) vivent dans `viewpointOptions`, pures et testées là.
  const viewpointGroups = useMemo(
    () => buildViewpointOptions(players, { teamLabelOf: lead.labelOf, noTeam: t.viewpointNoTeam, noData: t.viewpointNoData }),
    [players, lead.labelOf, t.viewpointNoTeam, t.viewpointNoData],
  )
  // LA DOMINANCE SE LIT SUR LES FRAGS (2026-08-28), plus sur le compteur du mode : elle vient
  // donc du MÊME fil que les deux pistes du dessus, jamais d'un second calque.
  const dominance = useMemo(
    () => buildFragDominance(reduit.frags, frameIntervalMs ?? 0, scale),
    [reduit, frameIntervalMs, scale],
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
    teammates: tracks.teammates,
    shades,
    absence,
    identity,
    onSeekFrame: playback.seekToFrame,
    viewpoint,
    viewpointGroups,
    onSelectViewpoint,
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
 * reduceFeed ramène le fil à ce que les pistes demandent.
 *
 * LES MÉDAILLES Y ENTRENT DEPUIS LE 2026-09-07 (lot L4), et de deux façons qui ne se confondent
 * pas. Celles qui sont RATTACHÉES à un kill (±500 ms, `killFeedLogic`) voyagent avec lui en
 * IDENTITÉ COMPLÈTE (`MedalEvent`, pas seulement leur libellé depuis le 2026-09-09, décision D7) :
 * la marque de kill existe déjà, elle recevra le badge du jeu en surimpression et son infobulle
 * portera titre ET description (décision 9 — pas de second repère au même endroit). Celles qui
 * restent ORPHELINES, elles, n'ont ni tueur ni défunt — ce sont des médailles d'OBJECTIF, et sans
 * marque à décorer elles en prennent une à elles.
 *
 * UN LIBELLÉ VIDE NE PASSE PAS. Sur les matchs antérieurs au backfill des médailles du fil, le
 * nom peut manquer : une décoration sans libellé dirait « il s'est passé quelque chose » sans
 * pouvoir dire quoi, et une marque orpheline muette serait un point de plus sur la frise sans
 * infobulle. Dans les deux cas, rien.
 *
 * CE QUI EST FILTRÉ ICI ET CE QUI NE L'EST PAS : reduceFeed dit QUI, `buildEventTracks` dit SUR
 * QUELLE PISTE. Les médailles orphelines de TOUTE la salle sortent donc d'ici, comme les kills,
 * et c'est le point de vue qui ne retient que les siennes en aval — une seule règle de piste, un
 * seul endroit où la lire.
 *
 * EXPORTÉ POUR ÊTRE TESTÉ (revue R1), pas pour être appelé d'ailleurs : c'est ici que se décide
 * À QUI appartient une ligne, et une inversion tueur/victime y serait invisible à la relecture
 * comme à l'écran — les deux pistes resteraient peuplées, avec les mauvais événements.
 */
export function reduceFeed(
  entries: readonly ReplayFeedEntry[],
  viewpoint: string | null,
): { kills: TrackKill[]; deaths: TrackDeath[]; medals: TrackMedal[]; frags: TrackFrag[] } {
  const kills: TrackKill[] = []
  const deaths: TrackDeath[] = []
  const medals: TrackMedal[] = []
  const frags: TrackFrag[] = []
  for (const entry of entries) {
    if (entry.medal) {
      if (entry.medal.label) {
        medals.push({
          key: entry.key,
          replayMs: entry.replayMs,
          xuid: entry.medal.xuid,
          medal: entry.medal,
        })
      }
      continue
    }
    if (entry.death) {
      deaths.push({ key: entry.key, replayMs: entry.replayMs, xuid: entry.death.xuid })
      continue
    }
    const kill = entry.kill
    if (!kill) continue
    kills.push({
      key: entry.key,
      replayMs: entry.replayMs,
      xuid: kill.xuid,
      medals: kill.medals.filter((m) => m.label !== ''),
    })
    // LES FRAGS COMPTENT TOUTE LA SALLE, pas seulement les joueurs marqués : la dominance
    // oppose deux CAMPS. Un tueur dont le camp n'est pas résolu (acteur hors scoreboard) ne
    // compte pour personne — l'attribuer par défaut fausserait le meneur.
    if (kill.teamID != null) frags.push({ replayMs: entry.replayMs, teamId: kill.teamID })
    // LA MÊME LIGNE PEUT ÊTRE LES DEUX : le frag de l'un est la mort de l'autre. On ne la range
    // du côté des morts que si la victime EST le joueur regardé — sinon `buildEventTracks`
    // l'écarte de toute façon (les morts ne vont que sur sa piste), et la clé dérivée ne
    // servirait à rien. Ce test lisait la marque `me` de `playerMarks` jusqu'au 2026-09-07 :
    // c'était le même joueur, par un détour de plus. Le point de vue le dit directement.
    if (kill.victimXuid && viewpoint != null && kill.victimXuid === viewpoint) {
      deaths.push({ key: `${entry.key}-v`, replayMs: entry.replayMs, xuid: kill.victimXuid })
    }
  }
  return { kills, deaths, medals, frags }
}

