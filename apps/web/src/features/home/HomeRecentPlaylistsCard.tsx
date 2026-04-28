import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import type { HomePlaylistRank } from '@/lib/api/types'
import { unrankedBadgeURL } from '@/lib/staticAssets'

function RankBadge({ imageUrl, label }: { imageUrl: string; label: string }) {
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
      className="h-12 w-12 object-contain"
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

  return (
    <Card data-testid="home-recent-playlists-card" className="flex self-start flex-col">
      <CardHeader className="space-y-0 pb-3">
        <CardTitle className="text-base">Playlists récentes</CardTitle>
      </CardHeader>

      <CardContent>
        {items.length > 0 ? (
          <ul className="space-y-4" data-testid="home-recent-playlists-list">
            {items.map((item) => {
              const showPlacement = item.is_ranked && !item.badge_image_url && item.rating_value == null && !item.tier_label

              return (
              <li
                key={item.playlist_name}
                className="flex items-center gap-3"
                data-testid="home-recent-playlist-item"
              >
                {/* Badge rang ou indicateur visuel */}
                <div className="flex h-12 w-12 shrink-0 items-center justify-center">
                  {item.badge_image_url ? (
                    <RankBadge
                      imageUrl={item.badge_image_url}
                      label={item.tier_label ?? 'Rang'}
                    />
                  ) : showPlacement ? (
                    <img
                      src={unrankedBadgeURL()}
                      alt="En placement"
                      data-testid="home-rank-unranked-image"
                      className="h-12 w-12 object-contain opacity-70"
                      loading="lazy"
                      decoding="async"
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
                  ) : showPlacement ? (
                    <p
                      data-testid="home-rank-unranked-label"
                      className="text-xs text-muted-foreground"
                    >
                      En placement
                    </p>
                  ) : item.rating_value == null && (
                    <p
                      data-testid="home-rank-neutral-label"
                      className="text-xs text-muted-foreground"
                    >
                      Sans classement
                    </p>
                  )}
                </div>
              </li>
              )
            })}
          </ul>
        ) : (
          <p className="text-sm text-muted-foreground">
            Aucune playlist récente disponible.
          </p>
        )}
      </CardContent>
    </Card>
  )
}
