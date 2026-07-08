import { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import type { HomePlaylistRank } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { unrankedBadgeURL } from '@/lib/staticAssets'

// GH2-A3 : une playlist dont le nom n'a pas été résolu côté backend (asset_translations
// sans entrée pour la locale de requête) retombe sur le playlist_id brut = un UUID. On
// ne l'affiche JAMAIS tel quel : on substitue le libellé neutre localisé
// `common.home.unknown_playlist`. Cause data documentée (backfill EN) — cf. thought_log.
const RAW_ASSET_ID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
function isRawAssetId(name: string | null | undefined): boolean {
  return !!name && RAW_ASSET_ID_RE.test(name.trim())
}

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
    <section className="flex flex-col gap-3">
      {/* Titre de section (type 1 du catalogue), SORTI de la carte (cf. demande user). */}
      <header className="flex items-center gap-1.5">
        <h3 className="text-base font-semibold text-foreground">{t('common.home.recent_playlists')}</h3>
      </header>
      <Card data-testid="home-recent-playlists-card" className="flex flex-1 flex-col">
      <CardContent className="pt-6">
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
              // Bande ORDINALE (matured uniquement) + valeur formatée — calqué skill peak.
              const tierProgress =
                !isPlacement && item.tier_progress_pct != null ? item.tier_progress_pct : null
              const ratingValueStr =
                item.rating_value != null
                  ? item.rating_type === 'LUSR'
                    ? `${Math.round(item.rating_value)} pts`
                    : `${Math.round(item.rating_value)} CSR`
                  : null

              return (
                <li
                  key={item.playlist_name}
                  data-testid="home-recent-playlist-item"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-12 w-12 shrink-0 items-center justify-center">
                      {badgeImageURL ? (
                        <RankBadge
                          imageUrl={badgeImageURL}
                          label={isPlacement ? (placementCompleted === 0 ? 'Non classé' : 'En placement') : (item.tier_label ?? 'Rang')}
                          testId={isPlacement ? 'home-rank-unranked-image' : undefined}
                          opacity={isPlacement ? 'dim' : 'full'}
                        />
                      ) : (
                        <NeutralRankPlaceholder />
                      )}
                    </div>

                    {/* Colonne : nom de playlist AU-DESSUS du palier (swap), valeur en
                        secondaire dessous — même colonne (cf. demande user). */}
                    <div className="min-w-0">
                      <p
                        data-testid="home-recent-playlist-name"
                        className="truncate text-2xs font-medium uppercase tracking-label-md text-muted-foreground"
                      >
                        {isRawAssetId(item.playlist_name)
                          ? t('common.home.unknown_playlist')
                          : item.playlist_name || t('common.home.unknown_playlist')}
                      </p>
                      {showTierLabel ? (
                        <p
                          data-testid="home-rank-tier-label"
                          className="truncate text-sm font-semibold text-foreground"
                        >
                          {item.tier_label}
                        </p>
                      ) : isPlacement ? (
                        <p
                          data-testid="home-rank-unranked-label"
                          className="text-sm font-semibold text-foreground"
                        >
                          {placementCompleted === 0
                            ? 'Non classé'
                            : placementCompleted != null
                              ? `En placement (${placementCompleted}/${placementTotal})`
                              : 'En placement'}
                        </p>
                      ) : item.rating_value == null ? (
                        <p
                          data-testid="home-rank-neutral-label"
                          className="text-sm text-muted-foreground"
                        >
                          {t('common.home.unranked')}
                        </p>
                      ) : null}

                      {showRatingValue && ratingValueStr && (
                        <p
                          data-testid="home-recent-playlist-value"
                          className="text-2xs font-medium tabular-nums text-muted-foreground"
                        >
                          {ratingValueStr}
                        </p>
                      )}
                    </div>

                    {/* Barre ORDINALE + sous-palier suivant à l'extrémité droite
                        (matured uniquement) — même rendu que le skill peak. */}
                    {tierProgress != null && (
                      <div className="flex min-w-0 flex-1 items-center gap-2">
                        <div className="min-w-0 flex-1">
                          <CompositeProgressBar
                            value={tierProgress}
                            fillTestId="home-recent-playlist-tier-progress-fill"
                          />
                        </div>
                        {item.next_tier_label && (
                          <span
                            data-testid="home-recent-playlist-next-tier"
                            className="shrink-0 truncate text-2xs font-medium text-muted-foreground"
                          >
                            {item.next_tier_label}
                          </span>
                        )}
                      </div>
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
    </section>
  )
}
