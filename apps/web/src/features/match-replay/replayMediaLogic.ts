/**
 * replayMediaLogic.ts — LES MÉDIAS DU MATCH POSÉS SUR L'AXE DU REJEU.
 *
 * TROIS HORLOGES, UNE SEULE SOUSTRACTION. Un média porte des instants ABSOLUS (l'heure à
 * laquelle la capture a eu lieu) ; le match commence à `header.start_time` ; le film, lui,
 * cale son image zéro `doc.originMs` plus tard (cf. killFeedLogic). D'où :
 *
 *     replayMs = (capture − header.start_time) − originMs
 *
 * C'est la MÊME doctrine que le fil (`replayMs = event_time_ms + t0Ms − originMs`), écrite
 * pour une source qui donne des dates au lieu d'un offset : `capture − start_time` est
 * exactement ce que `event_time_ms + t0Ms` reconstruit. Le recalage se fait ICI et une seule
 * fois — aucun composant ne le refait, deux recalages menés séparément divergeraient.
 *
 * `capture_time` EST UNE FIN DE CAPTURE, pas un début. Poser un clip dessus le décalerait de
 * sa propre durée : un clip de trente secondes tourné autour d'un frag apparaîtrait trente
 * secondes après lui. On prend donc le début quand la base le connaît, et on le RECONSTITUE
 * sinon en retranchant la durée de la fin (décision utilisateur du 2026-08-28).
 *
 * CE QU'ON N'INVENTE JAMAIS : un média sans le moindre horodatage est ÉCARTÉ, et un match
 * dont l'en-tête n'a pas d'heure de début n'a aucun média sur sa frise. Une pose au hasard
 * serait pire que l'absence — elle se lirait comme un fait.
 *
 * Le cadrage, lui, n'est pas notre affaire : `placeMedia` écarte ce qui tombe hors de la
 * fenêtre de gameplay, comme pour les autres pistes.
 *
 * Pas de React : logique pure, testée (replayMediaLogic.test.ts).
 */
import type { MatchMediaTab, MatchViewHeader } from '@/lib/api/types'

import type { ReplayMediaItem } from './replayTimelineTracksLogic'

/** Ce que les médias lisent de l'en-tête : l'origine de l'axe du match. */
export type ReplayMediaHeader = Pick<MatchViewHeader, 'start_time'>

/** Un élément de l'onglet médias du match, tel que l'API le sert (URLs déjà servables). */
type MediaTabItem = NonNullable<MatchMediaTab['media_items']>[number]

/**
 * buildReplayMedia mappe l'onglet médias du match vers la piste Médias de la frise.
 *
 * `originMs` absent (artefact antérieur au schéma v4, ou origine que le producteur a refusé
 * d'établir) vaut zéro : le placement se dégrade du décalage de l'image zéro plutôt que de
 * faire disparaître la piste. C'est le même choix que le reste du lecteur — la frise reste
 * lisible, elle n'est simplement plus au millième.
 */
export function buildReplayMedia(
  mediaTab: MatchMediaTab | null | undefined,
  header: ReplayMediaHeader | null | undefined,
  originMs: number | null | undefined,
): ReplayMediaItem[] {
  const items = mediaTab?.media_items
  const matchStartMs = parseInstant(header?.start_time)
  if (!items || items.length === 0 || matchStartMs === null) return []

  const origin = Number.isFinite(originMs) ? (originMs as number) : 0
  const out: ReplayMediaItem[] = []
  for (const item of items) {
    const built = toReplayMedia(item, matchStartMs, origin)
    if (built) out.push(built)
  }
  return out.sort((a, b) => a.replayMs - b.replayMs)
}

/** Un média mappé, ou `null` quand il n'a aucun horodatage exploitable. */
function toReplayMedia(
  item: MediaTabItem,
  matchStartMs: number,
  originMs: number,
): ReplayMediaItem | null {
  // LE VOCABULAIRE DE LA FRISE N'EST PAS CELUI DE LA GALERIE : elle dit 'image'/'clip' là où
  // `normalizeMediaKind` dit 'screenshot'/'clip'. Tout ce qui n'est pas une vidéo est une
  // image — un kind inconnu se montre plutôt que de disparaître.
  const kind: ReplayMediaItem['kind'] = item.kind === 'video' ? 'clip' : 'image'
  const durationMs = clipDurationMs(item, kind)
  const startMs = captureStartMs(item, durationMs)
  if (startMs === null) return null

  const url = item.file_path
  return {
    // L'IDENTITÉ EST `file_id`, JAMAIS `file_path` : le chemin est MUTÉ par la conversion et
    // le transcodage HLS (cf. le schéma de media_files), un identifiant qui bouge casserait
    // le rapprochement d'un rendu à l'autre.
    id: String(item.file_id),
    kind,
    replayMs: startMs - matchStartMs - originMs,
    ...(durationMs !== undefined ? { durationMs } : {}),
    // Les vignettes sont les WebP ANIMÉS du pipeline (legacy .gif servis pareil) : un <img>
    // les anime sans rien de plus. À défaut, une image se sert d'elle-même en vignette ; un
    // clip n'a rien à montrer d'immobile, la piste retombe sur son rendu sans vignette.
    thumbUrl: item.thumbnail_url ?? (kind === 'image' ? url : ''),
    url,
    label: item.file_name,
  }
}

/**
 * Durée d'un clip en millisecondes, `undefined` pour une image ou pour un clip dont la base
 * ne connaît pas la durée (ffprobe absent à l'ingestion) — il sera posé ponctuellement.
 */
function clipDurationMs(item: MediaTabItem, kind: ReplayMediaItem['kind']): number | undefined {
  if (kind !== 'clip') return undefined
  const seconds = item.duration_seconds
  if (typeof seconds !== 'number' || !Number.isFinite(seconds) || seconds <= 0) return undefined
  return seconds * 1000
}

/**
 * Instant de DÉBUT de la capture, en ms epoch : celui de la base s'il existe, sinon la fin
 * moins la durée. Un clip sans début ni durée se pose sur sa fin — au pire il apparaît sa
 * propre durée trop tard, ce qui reste plus juste que de le retirer de la frise.
 */
function captureStartMs(item: MediaTabItem, durationMs: number | undefined): number | null {
  const start = parseInstant(item.capture_start_time)
  if (start !== null) return start
  const end = parseInstant(item.capture_time)
  if (end === null) return null
  return durationMs !== undefined ? end - durationMs : end
}

/** Parse un horodatage RFC3339 en ms epoch. `null` si absent ou illisible. */
function parseInstant(value: string | null | undefined): number | null {
  if (!value) return null
  const ms = Date.parse(value)
  return Number.isNaN(ms) ? null : ms
}
