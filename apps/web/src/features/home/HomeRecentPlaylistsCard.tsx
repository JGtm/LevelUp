import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { HomePlaylistRank } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { unrankedBadgeURL } from '@/lib/staticAssets'

function RankBadge({
  imageUrl,
  label,
  testId,
  opacity,
}: {
  imageUrl: string
  label: string
  testId?: string
  opacity?: 'full' | 'dim'
}) {
  const [failed, setFailed] = useState(false)

  if (failed) {
    return (
      <div
        className="flex h-12 w-12 shrink-0 items-center justify-center rounded-md border border-border bg-muted text-xs font-semibold text-muted-foreground"
        aria-label={label}
      >
        ?
      </div>
    )
  }

  return (
    <img
      src={imageUrl}
      alt={label}
      data-testid={testId}
      className={`h-12 w-12 object-contain ${opacity === 'dim' ? 'opacity-70' : ''}`}
      onError={() => setFailed(true)}
      loading="lazy"
      decoding="async"
    />
  )
}

function NeutralRankPlaceholder() {
  return (
    <div
      data-testid="home-rank-neutral-placeholder"
      className="flex h-12 w-12 shrink-0 items-center justify-center rounded-md border border-dashed border-border bg-muted text-lg font-semibold text-muted-foreground"
      aria-hidden="true"
    >
      —
    </div>
  )
}

export function HomeRecentPlaylistsCard({
  recentPlaylistRanks,
}: {
  recentPlaylistRanks?: HomePlaylistRank[]
}) {
  const items = recentPlaylistRanks ?? []
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  return (
    <Card data-testid="home-recent-playlists-card" className="flex self-start flex-col">
      <CardHeader className="space-y-0 pb-3">
        <CardTitle className="text-base">{t('common.home.recent_playlists')}</CardTitle>
      </CardHeader>

      <CardContent>
        {items.length > 0 ? (
          <ul className="space-y-4" data-testid="home-recent-playlists-list">
            {items.map((item) => {
              // Détection placement alignée sur HomeSkillPeakCard.resolveSkillPeakState :
              // measurement_matches_remaining > 0 est la source de vérité ; on garde
              // l'ancien fallback (rating null + pas de tier) pour les payloads legacy.
              const remaining = item.measurement_matches_remaining ?? 0
              const isPlacement =
                remaining > 0 ||
                (item.is_ranked && item.rating_value == null && !item.tier_label)
              // Phase 6 : placement_total injecté par le backend (5 ou 10 selon saison).
              // Fallback 10 pour payloads legacy.
              const placementTotal = item.placement_total ?? 10
              const placementCompleted =
                remaining > 0
                  ? Math.min(placementTotal - 1, Math.max(0, placementTotal - remaining))
                  : null
              const badgeImageURL =
                item.badge_image_url ?? (isPlacement ? unrankedBadgeURL() : null)
              const showRatingValue = !isPlacement && item.rating_value != null
              const showTierLabel = !isPlacement && item.tier_label

              return (
                <li
                  key={item.playlist_name}
                  className="flex items-center gap-3"
                  data-testid="home-recent-playlist-item"
                >
                  <div className="flex h-12 w-12 shrink-0 items-center justify-center">
                    {badgeImageURL ? (
                      <RankBadge
                        imageUrl={badgeImageURL}
                        label={isPlacement ? 'En placement' : (item.tier_label ?? 'Rang')}
                        testId={isPlacement ? 'home-rank-unranked-image' : undefined}
                        opacity={isPlacement ? 'dim' : 'full'}
                      />
                    ) : (
                      <NeutralRankPlaceholder />
                    )}
                  </div>

                  <div className="min-w-0 flex-1">
                    <p
                      data-testid="home-recent-playlist-name"
                      className="truncate text-sm font-medium text-foreground"
                    >
                      {item.playlist_name || 'Playlist inconnue'}
                    </p>

                    {showRatingValue && (
                      <p className="text-xs font-semibold tabular-nums text-foreground">
                        {item.rating_type === 'LUSR'
                          ? `${Math.round(item.rating_value!)} pts`
                          : `${Math.round(item.rating_value!)} CSR`}
                      </p>
                    )}

                    {showTierLabel ? (
                      <p
                        data-testid="home-rank-tier-label"
                        className="truncate text-xs text-muted-foreground"
                      >
                        {item.tier_label}
                      </p>
                    ) : isPlacement ? (
                      <p
                        data-testid="home-rank-unranked-label"
                        className="text-xs text-muted-foreground"
                      >
                        {placementCompleted != null
                          ? `En placement (${placementCompleted}/${placementTotal})`
                          : 'En placement'}
                      </p>
                    ) : item.rating_value == null && (
                      <p
                        data-testid="home-rank-neutral-label"
                        className="text-xs text-muted-foreground"
                      >
                        {t('common.home.unranked')}
                      </p>
                    )}
                  </div>
                </li>
              )
            })}
          </ul>
        ) : (
          <p className="text-sm text-muted-foreground">
            {t('common.home.no_recent_playlist')}
          </p>
        )}
      </CardContent>
    </Card>
  )
}
