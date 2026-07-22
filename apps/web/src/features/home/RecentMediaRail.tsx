import { useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useRecentMediaRail, useToggleMediaLike } from '@/features/media/queries'
import { MediaLightbox, MediaThumbnailCard } from '@/features/media/MediaViewer'
import { buildOwnerColorMap } from '@/features/media/mediaOwnerColors'
import { MediaMatchPicker } from '@/features/media/MediaMatchPicker'
import { useMediaPicker } from '@/features/media/useMediaPicker'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import { useAppShellStore } from '@/stores/appShellStore'

const HOME_MEDIA_LIMIT = 20

type MediaTab = 'recent' | 'liked'

interface RecentMediaRailProps {
  playerSlug: string
}

export function RecentMediaRail({ playerSlug }: RecentMediaRailProps) {
  const [mediaTab, setMediaTab] = useState<MediaTab>('recent')
  const recentQuery = useRecentMediaRail(playerSlug, HOME_MEDIA_LIMIT, false)
  const likedQuery = useRecentMediaRail(playerSlug, HOME_MEDIA_LIMIT, true)
  const { data, isLoading, isError } = mediaTab === 'liked' ? likedQuery : recentQuery
  const toggleMediaLike = useToggleMediaLike(playerSlug)
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)
  const [autoChain, setAutoChain] = useState(false)
  const picker = useMediaPicker()
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: HomeManifestKey) => formatMessage(homeManifest, key, locale)
  const items = useMemo(() => data?.items.items ?? [], [data?.items.items])
  const playerColorMap = useMemo(() => buildOwnerColorMap(items, playerSlug), [items, playerSlug])
  const recentTotal = recentQuery.data?.items.pagination.total
  const likedTotal = likedQuery.data?.items.pagination.total

  return (
    <section className="flex flex-col gap-3">
      {lightboxIndex !== null && (
        <MediaLightbox
          items={items}
          onToggleLike={(item) => toggleMediaLike.mutate({ file_path: item.file_path, liked: !item.liked })}
          startIndex={lightboxIndex}
          onClose={() => setLightboxIndex(null)}
          likeDisabled={toggleMediaLike.isPending}
          autoChain={autoChain}
          onToggleAutoChain={() => setAutoChain((c) => !c)}
          playerSlug={playerSlug}
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

      <header className="flex items-center justify-between gap-3">
        <h3 className="text-base font-semibold text-foreground">
          {mediaTab === 'recent' ? t('home.media.title_recent') : t('home.media.title_liked')}
        </h3>
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
              {t('home.media.tab_recent')}{recentTotal !== undefined && <span className="ml-1.5 opacity-70">({recentTotal})</span>}
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
              {t('home.media.tab_liked')}{likedTotal !== undefined && <span className="ml-1.5 opacity-70">({likedTotal})</span>}
            </Button>
          </div>
      </header>

        {isLoading ? (
          <div className="flex min-h-32 items-center justify-center">
            <Spinner size="md" label={t('home.media.loading')} />
          </div>
        ) : isError ? (
          <EmptyStateNotice
            title={t('home.media.unavailable_title')}
            description={t('home.media.unavailable_description')}
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
                  playerSlug={playerSlug}
                  playerColorMap={playerColorMap}
                  onAssociate={picker.openFor}
                />
              </CarouselItem>
            ))}
          </Carousel>
        ) : mediaTab === 'liked' ? (
          <EmptyStateNotice
            title={t('home.media.empty_liked_title')}
            description={t('home.media.empty_liked_description')}
          />
        ) : (
          <EmptyStateNotice
            title={t('home.media.empty_recent_title')}
            description={t('home.media.empty_recent_description')}
          />
        )}
    </section>
  )
}
