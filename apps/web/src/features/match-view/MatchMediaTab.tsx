import { useState, useMemo } from 'react'
import type { MatchAssociatedMedia, MediaItemRow } from '@/lib/api/types'
import { MediaThumbnailCard, MediaLightbox } from '@/features/media/MediaViewer'
import { MediaMatchPicker } from '@/features/media/MediaMatchPicker'
import { useMediaPicker } from '@/features/media/useMediaPicker'
import { usePlayerGamertag } from '@/features/media/usePlayerGamertag'
import { useToggleMediaLike } from '@/features/media/queries'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { toMediaItemRow } from './_mediaItemRow'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { MatchViewText } from './i18n'

interface MatchMediaTabProps {
  items: MatchAssociatedMedia[]
  playerSlug: string
  matchId: string
  t: MatchViewText
}

export function MatchMediaTab({ items, playerSlug, matchId, t }: MatchMediaTabProps) {
  const initialRows = useMemo(
    () => items.map((item) => toMediaItemRow(item, matchId)),
    [items, matchId],
  )
  const [mediaRows, setMediaRows] = useState<MediaItemRow[]>(initialRows)
  const [lightboxIdx, setLightboxIdx] = useState<number | null>(null)
  const [autoChain, setAutoChain] = useState(false)
  const picker = useMediaPicker()
  const toggleLike = useToggleMediaLike(playerSlug)
  // Profil d'ami (lecture seule) : pas de réassociation. Le cœur reste actif : le like est
  // l'action du visiteur, attribuée à son propre profil (ADR 0029).
  const canWrite = useCanWritePlayer(playerSlug)
  // L'onglet montre aussi les clips des coéquipiers : réassocier n'est offert que sur les
  // siens (propriétaire ou admin, comme le serveur).
  const currentPlayerGamertag = usePlayerGamertag(playerSlug)
  const likeDisabled = toggleLike.isPending

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
    return <EmptyStateNotice title={t.mediaNoCaptures} description={t.mediaNoCapturesDesc} />
  }

  return (
    <>
      {lightboxIdx !== null && (
        <MediaLightbox
          items={mediaRows}
          onToggleLike={handleToggleLike}
          startIndex={lightboxIdx}
          onClose={() => setLightboxIdx(null)}
          likeDisabled={likeDisabled}
          autoChain={autoChain}
          onToggleAutoChain={() => setAutoChain((c) => !c)}
          playerSlug={playerSlug}
          currentMatchId={matchId}
          onReassociate={canWrite ? picker.openFor : undefined}
          currentPlayerGamertag={currentPlayerGamertag}
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
            likeDisabled={likeDisabled}
            playerSlug={playerSlug}
            currentMatchId={matchId}
            onAssociate={canWrite ? picker.openFor : undefined}
            currentPlayerGamertag={currentPlayerGamertag}
          />
        ))}
      </div>
    </>
  )
}
