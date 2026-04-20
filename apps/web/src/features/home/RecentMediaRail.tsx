import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useRecentMediaRail, useToggleMediaLike } from '@/features/media/queries'
import { MediaLightbox, MediaThumbnailCard } from '@/features/media/MediaViewer'

const HOME_MEDIA_LIMIT = 4

interface RecentMediaRailProps {
  playerSlug: string
}

export function RecentMediaRail({ playerSlug }: RecentMediaRailProps) {
  const { data, isLoading, isError } = useRecentMediaRail(playerSlug, HOME_MEDIA_LIMIT)
  const toggleMediaLike = useToggleMediaLike(playerSlug)
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)
  const items = data?.items.items ?? []

  return (
    <Card>
      {lightboxIndex !== null && (
        <MediaLightbox
          items={items}
          onToggleLike={(item) => toggleMediaLike.mutate({ file_path: item.file_path, liked: !item.liked })}
          startIndex={lightboxIndex}
          onClose={() => setLightboxIndex(null)}
          likeDisabled={toggleMediaLike.isPending}
        />
      )}

      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle className="text-base">Médias récents</CardTitle>
        <Link
          to="/players/$playerSlug/media"
          params={{ playerSlug }}
          className="text-xs text-primary hover:underline"
        >
          Voir tout →
        </Link>
      </CardHeader>

      <CardContent>
        {isLoading ? (
          <div className="flex min-h-32 items-center justify-center">
            <Spinner size="md" label="Chargement des médias récents…" />
          </div>
        ) : isError ? (
          <EmptyStateNotice
            title="Médias récents indisponibles"
            description="La preview média n'a pas pu être chargée pour le moment."
          />
        ) : items.length > 0 ? (
          <div className="flex gap-3 overflow-x-auto pb-2">
            {items.map((item, index) => (
              <div key={`${item.file_path}-${index}`} className="w-[240px] shrink-0">
                <MediaThumbnailCard
                  item={item}
                  onToggleLike={(currentItem) => {
                    toggleMediaLike.mutate({
                      file_path: currentItem.file_path,
                      liked: !currentItem.liked,
                    })
                  }}
                  onOpen={() => setLightboxIndex(index)}
                  likeDisabled={toggleMediaLike.isPending}
                />
              </div>
            ))}
          </div>
        ) : (
          <EmptyStateNotice
            title="Aucun média récent disponible"
            description="Aucune capture ou clip n'est associé au joueur pour le scope actuel."
          />
        )}
      </CardContent>
    </Card>
  )
}
