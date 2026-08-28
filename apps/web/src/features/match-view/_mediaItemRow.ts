/**
 * _mediaItemRow.ts — projection d'un media associe a un match (`MatchAssociatedMedia`,
 * ce que `media_tab.media_items` sert reellement) vers la ligne de galerie partagee
 * (`MediaItemRow`), consommee par `MediaThumbnailCard` / `MediaLightbox`.
 *
 * Le seul point non trivial est le `kind` : il est REQUIS au contrat, donc le chemin
 * nominal passe toujours par `normalizeMediaKind` ('video' -> 'clip', 'image' ->
 * 'screenshot'). Le repli sur la duree ne sert QUE les reponses en cache navigateur
 * d'avant le fix kind ; depuis le lot medias la duree arrive peuplee, donc « duree
 * presente » y vaut bien « clip ».
 *
 * `!= null` et pas `!== null` : `duration_seconds` est `omitempty` au contrat, donc
 * ABSENT (undefined) et jamais null sur une image — le test strict classait toute image
 * sans kind en 'clip'. Couvert par `_mediaItemRow.test.ts` (cas mute).
 */
import { normalizeMediaKind } from '@/features/media/queries'
import type { MatchAssociatedMedia, MediaItemRow } from '@/lib/api/types'

export function toMediaItemRow(item: MatchAssociatedMedia, matchId: string): MediaItemRow {
  return {
    basename: item.file_name,
    file_path: item.file_path,
    kind: item.kind
      ? normalizeMediaKind(item.kind)
      : item.duration_seconds != null
        ? 'clip'
        : 'screenshot',
    thumbnail_path: item.thumbnail_url ?? null,
    match_id: matchId,
    capture_end_utc: item.capture_time ?? null,
    match_start_time: null,
    section: 'mine',
    owner_gamertag: null,
    map_name: null,
    mode_name: null,
    liked: item.liked,
    like_count: 0,
  }
}
