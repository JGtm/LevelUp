/**
 * replayTimelineTracksLogic — CE QUE LA FRISE MONTRE, en dehors du temps : les marques des
 * pistes (les tiennes, celles de tes alliés), les segments de DOMINANCE, et les médias posés
 * sur le match. Logique pure, testable, sans JSX — les composants n'en font que le rendu.
 *
 * TROIS PISTES PLUTÔT QU'UNE FRISE : une frise seule donne la position dans le temps et rien
 * d'autre — c'est le constat qui avait donné les marques de retournement (`ReplayLeadMarks`,
 * supprimé le 2026-08-28 : la piste DOMINANCE dit la même chose en DURÉES). Empilées, les
 * pistes disent la FORME du match avant qu'on l'ait lu — où tu as marqué, où tu es tombé,
 * quand ton équipe a pris l'ascendant.
 *
 * L'ÉCHELLE EST CELLE DE LA FRISE, donc celle de la FENÊTRE DE GAMEPLAY (`replayWindow.ts`) :
 * une marque calculée sur le film entier se poserait à côté de l'instant qu'elle désigne. Et
 * comme les pistes s'alignent sur un `input[type=range]`, elles héritent de sa géométrie : la
 * piste utile court de THUMB_PX / 2 à largeur − THUMB_PX / 2 — le navigateur ne publie pas la
 * largeur du curseur.
 *
 * LE SUFFIXE `Logic` LÈVE UNE COLLISION DE NOMS, il n'est pas décoratif : Windows ne distingue
 * pas ce fichier de `ReplayTimelineTracks.tsx`, et TypeScript refuse alors les deux dans le
 * même programme (TS1149). C'est aussi le patron du dépôt (killFeedLogic, victoryLogic).
 */
import type { PlayerMarkKind } from './playerMarks'
import type { ReplayWindowBounds } from './replayWindow'

/**
 * Largeur supposée du curseur natif, en px. Mesure héritée des marques de retournement : le
 * navigateur ne publie pas cette valeur, 16 px est celle des thèmes par défaut, et l'écart
 * résiduel se compte en pixels sur une frise qui en fait plusieurs centaines.
 */
export const THUMB_PX = 16

/** Position CSS d'un ratio [0..1] sur la piste, curseur compris. */
export function trackLeft(ratio: number): string {
  const r = Math.min(1, Math.max(0, ratio))
  return `calc(${THUMB_PX / 2}px + (100% - ${THUMB_PX}px) * ${r})`
}

/** Largeur CSS d'un intervalle [a..b] de ratios sur la même piste. */
export function trackWidth(from: number, to: number): string {
  const span = Math.min(1, Math.max(0, to)) - Math.min(1, Math.max(0, from))
  return `calc((100% - ${THUMB_PX}px) * ${Math.max(0, span)})`
}

/** Bornes de la frise, en IMAGES : les mêmes que celles du curseur. */
export interface TrackScale {
  from: number
  span: number
}

/**
 * L'échelle des pistes, lue de la fenêtre de gameplay comme la frise. `span <= 0` = rien à
 * placer (film d'une image, fenêtre dégénérée) : les appelants ne rendent alors aucune piste.
 */
export function trackScale(playWindow: ReplayWindowBounds | null, frameCount: number): TrackScale {
  const from = playWindow?.startFrame ?? 0
  const span = (playWindow?.endFrame ?? frameCount - 1) - from
  return { from, span }
}

/** Ratio [0..1] d'un instant du rejeu (ms) sur l'échelle des pistes. */
export function ratioOfMs(replayMs: number, frameIntervalMs: number, scale: TrackScale): number {
  if (!frameIntervalMs || scale.span <= 0) return 0
  return (replayMs / frameIntervalMs - scale.from) / scale.span
}

/** Une marque d'événement sur une piste. `kind` décide de sa couleur, jamais sa piste. */
export interface TrackMark {
  key: string
  ratio: number
  kind: 'kill' | 'death'
  /** Instant, pour l'infobulle (mm:ss déjà mis en forme par l'appelant). */
  clock: string
}

/** Les deux pistes d'événements : la tienne, celle de tes alliés. */
export interface EventTracks {
  own: TrackMark[]
  allies: TrackMark[]
}

/** Un kill, réduit à ce dont les pistes ont besoin. */
export interface TrackKill {
  key: string
  replayMs: number
  /** xuid du tueur (celui à qui la marque appartient). */
  xuid: string
}

/** Une mort, réduite de même. `xuid` est celui du défunt. */
export interface TrackDeath {
  key: string
  replayMs: number
  xuid: string
}

/**
 * buildEventTracks range les kills et les morts sur DEUX pistes, selon la marque d'identité du
 * joueur (`playerMarks.ts` : 'me' | 'friend'). Un acteur SANS marque n'est sur aucune piste —
 * la frise ne parle que du joueur de la page et de ses amis, pas de la salle entière ; deviner
 * un camp par défaut peuplerait les pistes de gens que personne ne suit.
 *
 * Les morts ne vont QUE sur la piste 'me' : « où je suis tombé » est une lecture de soi. Une
 * piste alliée mêlant leurs kills et leurs morts deviendrait illisible à huit joueurs.
 */
export function buildEventTracks(
  kills: readonly TrackKill[],
  deaths: readonly TrackDeath[],
  marks: ReadonlyMap<string, PlayerMarkKind>,
  frameIntervalMs: number,
  scale: TrackScale,
  clockOf: (replayMs: number) => string,
): EventTracks {
  const own: TrackMark[] = []
  const allies: TrackMark[] = []
  if (scale.span <= 0) return { own, allies }
  for (const k of kills) {
    const mark = marks.get(k.xuid)
    if (!mark) continue
    const at: TrackMark = {
      key: k.key,
      ratio: ratioOfMs(k.replayMs, frameIntervalMs, scale),
      kind: 'kill',
      clock: clockOf(k.replayMs),
    }
    if (at.ratio < 0 || at.ratio > 1) continue
    ;(mark === 'me' ? own : allies).push(at)
  }
  for (const d of deaths) {
    if (marks.get(d.xuid) !== 'me') continue
    const at: TrackMark = {
      key: d.key,
      ratio: ratioOfMs(d.replayMs, frameIntervalMs, scale),
      kind: 'death',
      clock: clockOf(d.replayMs),
    }
    if (at.ratio < 0 || at.ratio > 1) continue
    own.push(at)
  }
  return { own, allies }
}

/** Un segment de DOMINANCE : qui menait, de quand à quand (en ratios de frise). */
export interface DominanceSegment {
  key: string
  from: number
  to: number
  teamId: number
}

/** Un changement de meneur, tel que `scoreTimeline` le publie. */
export interface LeadChangeLike {
  frame: number
  teamId: number
}

/**
 * buildDominance transforme les CHANGEMENTS de meneur en SEGMENTS pleins : un changement est
 * un instant, la dominance est une durée. Le premier segment part du début de la fenêtre (le
 * meneur d'avant le premier retournement n'est pas publié : c'est l'autre camp du premier
 * changement — on ne l'invente pas, on laisse le segment initial vide plutôt que de fabriquer
 * un camp), le dernier court jusqu'à la fin.
 *
 * Aucun changement = AUCUN segment, et non « une équipe a mené tout du long » : la donnée dit
 * seulement qu'il n'y a pas eu de retournement, pas qui menait.
 */
export function buildDominance(
  changes: readonly LeadChangeLike[],
  scale: TrackScale,
): DominanceSegment[] {
  if (changes.length === 0 || scale.span <= 0) return []
  const out: DominanceSegment[] = []
  const ratio = (frame: number) => Math.min(1, Math.max(0, (frame - scale.from) / scale.span))
  for (let i = 0; i < changes.length; i += 1) {
    const c = changes[i]
    const next = changes[i + 1]
    out.push({
      key: `${c.frame}-${c.teamId}`,
      from: ratio(c.frame),
      to: next ? ratio(next.frame) : 1,
      teamId: c.teamId,
    })
  }
  return out
}

/**
 * UN MÉDIA DU JOUEUR posé sur le match : une capture (instant) ou un clip (durée). La source
 * est le `media_tab` de la vue match, réduit à ce format par `buildReplayMedia`
 * (replayMediaLogic.ts) — c'est LÀ que vivent la règle « début = fin − durée » et le recalage
 * sur l'axe du rejeu ; ici, `replayMs` est déjà sur cet axe.
 */
export interface ReplayMediaItem {
  id: string
  kind: 'image' | 'clip'
  /** Instant du média sur l'axe du rejeu, en ms. */
  replayMs: number
  /** Durée du clip, en ms. Absente ou 0 pour une capture. */
  durationMs?: number
  /** Vignette (petite) et média plein, tels que l'API les publiera. */
  thumbUrl: string
  url: string
  label?: string
}

/** Un média placé sur la piste : ratio de début, et largeur pour un clip. */
export interface PlacedMedia {
  item: ReplayMediaItem
  from: number
  to: number
}

/**
 * placeMedia pose les médias sur l'échelle de la frise. UN CLIP OCCUPE SA DURÉE (c'est ce qui
 * le distingue d'une capture à l'œil, sans légende) ; une capture reçoit une largeur nulle et
 * la piste lui donne sa taille de vignette. Un média hors fenêtre de gameplay est écarté :
 * la frise ne montre pas le countdown, elle ne montrera pas ce qui s'y est passé.
 */
export function placeMedia(
  media: readonly ReplayMediaItem[],
  frameIntervalMs: number,
  scale: TrackScale,
): PlacedMedia[] {
  if (scale.span <= 0) return []
  const out: PlacedMedia[] = []
  for (const item of media) {
    const from = ratioOfMs(item.replayMs, frameIntervalMs, scale)
    if (from < 0 || from > 1) continue
    const to =
      item.kind === 'clip' && item.durationMs
        ? Math.min(1, ratioOfMs(item.replayMs + item.durationMs, frameIntervalMs, scale))
        : from
    out.push({ item, from, to })
  }
  return out.sort((a, b) => a.from - b.from)
}

/**
 * COMBIEN D'IMAGES MONTRER D'UN CLIP dans la lightbox : une image toutes les trois secondes,
 * bornée à [4..12]. La borne basse évite une bande d'une seule vignette (elle ne dirait pas
 * qu'il s'agit d'une durée), la borne haute évite une bande illisible sur un long clip.
 */
export function clipFrameCount(durationMs: number): number {
  return Math.min(12, Math.max(4, Math.round(durationMs / 3_000)))
}
