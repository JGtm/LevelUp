import { useState, useMemo } from 'react'
import type { MatchAssociatedMedia, MediaItemRow } from '@/lib/api/types'
import { MediaThumbnailCard, MediaLightbox } from '@/features/media/MediaViewer'
import { MediaMatchPicker } from '@/features/media/MediaMatchPicker'
import { useMediaPicker } from '@/features/media/useMediaPicker'
import { useToggleMediaLike } from '@/features/media/queries'
import { toMediaItemRow } from './_mediaItemRow'
import type { MatchViewLocale } from './i18n'
import { MATCH_VIEW_TEXT } from './i18n'

function CameraOffIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.5}
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-8 w-8 text-muted-foreground"
      aria-hidden="true"
    >
      <path d="M2 2l20 20" />
      <path d="M7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16" />
      <path d="M9.5 4h5l2 3h3" />
      <path d="M14.121 15.121A3 3 0 1 1 9.88 10.88" />
    </svg>
  )
}

interface MatchMediaTabProps {
  items: MatchAssociatedMedia[]
  playerSlug: string
  matchId: string
  locale: MatchViewLocale
}

export function MatchMediaTab({ items, playerSlug, matchId, locale }: MatchMediaTabProps) {
  const t = MATCH_VIEW_TEXT[locale]
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

  if (mediaRows.length === 0) {
    return (
      <div className="flex min-h-48 flex-col items-center justify-center gap-5 px-8 py-12 text-center">
        <div className="flex h-16 w-16 items-center justify-center rounded-full bg-muted">
          <CameraOffIcon />
        </div>
        <div className="space-y-1.5">
          <p className="font-semibold text-foreground">{t.mediaNoCaptures}</p>
          <p className="mx-auto max-w-xs text-sm text-muted-foreground">
            {t.mediaNoCapturesDesc}
          </p>
        </div>
      </div>
    )
  }

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
