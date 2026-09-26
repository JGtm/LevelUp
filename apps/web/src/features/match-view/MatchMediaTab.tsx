import { useState, useMemo } from 'react'
import type { MatchAssociatedMedia, MediaItemRow } from '@/lib/api/types'
import { MediaThumbnailCard, MediaLightbox } from '@/features/media/MediaViewer'
import { MediaMatchPicker } from '@/features/media/MediaMatchPicker'
import { useMediaPicker } from '@/features/media/useMediaPicker'
import { useToggleMediaLike } from '@/features/media/queries'
import { toMediaItemRow } from './_mediaItemRow'
import { hasMediaItems } from './blockPredicates'

interface MatchMediaTabProps {
  items: MatchAssociatedMedia[]
  playerSlug: string
  matchId: string
}

export function MatchMediaTab({ items, playerSlug, matchId }: MatchMediaTabProps) {
  const initialRows = useMemo(
    () => items.map((item) => toMediaItemRow(item, matchId)),
    [items, matchId],
  )
  const [mediaRows, setMediaRows] = useState<MediaItemRow[]>(initialRows)
  const [lightboxIdx, setLightboxIdx] = useState<number | null>(null)
  const [autoChain, setAutoChain] = useState(false)
  const picker = useMediaPicker()
  const toggleLike = useToggleMediaLike(playerSlug)

  function handleToggleLike(item: MediaItemRow) {
    const nextLiked = !item.liked
    const delta = nextLiked ? 1 : -1
    setMediaRows((prev) =>
      prev.map((r) =>
        r.file_path === item.file_path
          ? { ...r, liked: nextLiked, like_count: Math.max(0, r.like_count + delta) }
          : r,
      ),
    )
    toggleLike.mutate({ file_path: item.file_path, liked: nextLiked })
  }

  // MÊME prédicat que celui lu par la page pour poser (ou non) son titre de section.
  if (!hasMediaItems(items)) return null

  return (
    <>
      {lightboxIdx !== null && (
        <MediaLightbox
          items={mediaRows}
          onToggleLike={handleToggleLike}
          startIndex={lightboxIdx}
          onClose={() => setLightboxIdx(null)}
          likeDisabled={toggleLike.isPending}
          autoChain={autoChain}
          onToggleAutoChain={() => setAutoChain((c) => !c)}
          playerSlug={playerSlug}
          currentMatchId={matchId}
          onReassociate={picker.openFor}
        />
      )}
      {picker.state && (
        <MediaMatchPicker
          playerSlug={playerSlug}
          filePath={picker.state.filePath}
          hasCurrentMatch={picker.state.hasCurrentMatch}
          onClose={picker.close}
        />
      )}
      <div className="grid gap-4 [grid-template-columns:repeat(auto-fill,minmax(200px,1fr))]">
        {mediaRows.map((item, idx) => (
          <MediaThumbnailCard
            key={item.file_path}
            item={item}
            onToggleLike={handleToggleLike}
            onOpen={() => setLightboxIdx(idx)}
            likeDisabled={toggleLike.isPending}
            playerSlug={playerSlug}
            currentMatchId={matchId}
            onAssociate={picker.openFor}
          />
        ))}
      </div>
    </>
  )
}
