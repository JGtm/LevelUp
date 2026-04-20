import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useRecentMediaRail, useToggleMediaLike } from '@/features/media/queries'
import { MediaLightbox, MediaThumbnailCard } from '@/features/media/MediaViewer'

const HOME_MEDIA_LIMIT = 12

type MediaTab = 'recent' | 'liked'

interface RecentMediaRailProps {
  playerSlug: string
}

export function RecentMediaRail({ playerSlug }: RecentMediaRailProps) {
  const [mediaTab, setMediaTab] = useState<MediaTab>('recent')
  const { data, isLoading, isError } = useRecentMediaRail(playerSlug, HOME_MEDIA_LIMIT, mediaTab === 'liked')
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

      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-base">
            {mediaTab === 'recent' ? 'Médias récents' : 'Médias aimés'}
          </CardTitle>
          <div className="flex items-center gap-1 border-b border-transparent">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setMediaTab('recent')}
              className={`rounded-none border-b-2 px-3 py-1.5 text-xs ${
                mediaTab === 'recent'
                  ? 'border-primary text-primary font-semibold'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              Récents
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setMediaTab('liked')}
              className={`rounded-none border-b-2 px-3 py-1.5 text-xs ${
                mediaTab === 'liked'
                  ? 'border-primary text-primary font-semibold'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              Aimés
            </Button>
          </div>
        </div>
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
          <Carousel>
            {items.map((item, index) => (
              <CarouselItem key={`${item.file_path}-${index}`} className="w-[240px]">
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
              </CarouselItem>
            ))}
          </Carousel>
        ) : mediaTab === 'liked' ? (
          <EmptyStateNotice
            title="Aucun média aimé"
            description="Marquez vos captures ou clips avec le ♥ pour les retrouver ici."
          />
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
