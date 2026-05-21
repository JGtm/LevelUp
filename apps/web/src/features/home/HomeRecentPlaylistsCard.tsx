import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { HomePlaylistRank } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

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
              const isPlacement =
                item.is_ranked && item.rating_value == null && !item.tier_label
              const placementCompleted =
                item.measurement_matches_remaining != null
                  ? Math.min(9, Math.max(0, 10 - item.measurement_matches_remaining))
                  : null

              return (
                <li
                  key={item.playlist_name}
                  className="flex items-center gap-3"
                  data-testid="home-recent-playlist-item"
                >
                  {/* Badge rang ou indicateur visuel — badge_image_url est toujours fourni
                      par le backend pour les playlists classées (rang CSR ou unranked_N.png
                      en placement). Fallback NeutralRankPlaceholder uniquement pour les
                      playlists non classées sans rating calculé. */}
                  <div className="flex h-12 w-12 shrink-0 items-center justify-center">
                    {item.badge_image_url ? (
                      <RankBadge
                        imageUrl={item.badge_image_url}
                        label={item.tier_label ?? (isPlacement ? 'En placement' : 'Rang')}
                        testId={isPlacement ? 'home-rank-unranked-image' : undefined}
                        opacity={isPlacement ? 'dim' : 'full'}
                      />
                    ) : (
                      <NeutralRankPlaceholder />
                    )}
                  </div>

                  {/* Infos playlist + rang */}
                  <div className="min-w-0 flex-1">
                    <p
                      data-testid="home-recent-playlist-name"
                      className="truncate text-sm font-medium text-foreground"
                    >
                      {item.playlist_name || 'Playlist inconnue'}
                    </p>

                    {item.rating_value != null && (
                      <p className="text-xs font-semibold tabular-nums text-foreground">
                        {item.rating_type === 'LUSR'
                          ? `${Math.round(item.rating_value)} pts`
                          : `${Math.round(item.rating_value)} CSR`}
                      </p>
                    )}

                    {item.tier_label ? (
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
                          ? `En placement ${placementCompleted}/10`
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
