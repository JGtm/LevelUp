/**
 * HomeMatchesSection — « Matchs récents / Favoris » de l'Accueil (extrait de HomePage.tsx).
 *
 * Récents : la tête arrive avec la page ; les tuiles suivantes se chargent par pages à 5 tuiles
 * de la fin du carrousel (le carrousel ne monte que les tuiles chargées). Favoris : leur propre
 * lecture, servie en entier. Ouvrir un match des récents ne transporte PAS de liste d'identifiants :
 * les voisins viennent de l'API (tout l'historique) et le libellé du contexte, du descripteur.
 */
import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SectionTitle } from '@/components/ui/detail-section'
import { MatchCard } from '@/components/ui/match-card'
import { useSetMatchFavorite } from '@/features/match-history/queries'
import { useCanWritePlayer } from '@/lib/players/playerWriteAccess'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import type { MatchNavContext } from '@/lib/match-nav/navContext'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import type { RecentMatchItem } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeTilesHead } from '@/lib/home-tiles/homeTiles.logic'
import { useHomeFavorites, useHomeTiles } from './queries'

type MatchTab = 'recent' | 'favorites'

interface HomeMatchesSectionProps {
  playerSlug: string
  head: HomeTilesHead
}

function tabClass(active: boolean): string {
  return `rounded-none border-b-2 px-3 py-1.5 text-xs ${
    active
      ? 'border-primary text-primary font-semibold'
      : 'border-transparent text-muted-foreground hover:text-foreground'
  }`
}

export function HomeMatchesSection({ playerSlug, head }: HomeMatchesSectionProps) {
  const locale = useAppShellStore((s) => s.locale)
  const userTimezone = useAppShellStore((s) => s.userTimezone)
  const t = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)
  const [tab, setTab] = useState<MatchTab>('recent')
  const tiles = useHomeTiles(playerSlug, head)
  const favorites = useHomeFavorites(playerSlug)
  const favoriteMatches = favorites.data?.items ?? []
  const favoriteMutation = useSetMatchFavorite(playerSlug)
  const canWriteProfile = useCanWritePlayer(playerSlug) // profil d'ami : pas d'étoile favori
  const navigateToMatch = useNavigateToMatch(playerSlug)

  function goToMatch(matchId: string, source: 'home_recent' | 'home_favorites') {
    const ctx: MatchNavContext =
      source === 'home_recent'
        ? { source, matchIds: [], contextDescriptor: { kind: 'recent' } }
        : { source, matchIds: favoriteMatches.map((m) => m.match_id), contextDescriptor: { kind: 'favorites' } }
    navigateToMatch(matchId, ctx)
  }

  function card(m: RecentMatchItem, source: 'home_recent' | 'home_favorites') {
    return (
      <CarouselItem key={m.match_id} className="w-72">
        <MatchCard
          match={m}
          locale={locale}
          timezone={userTimezone}
          playerSlug={playerSlug}
          onClick={() => goToMatch(m.match_id, source)}
          onToggleFavorite={
            !canWriteProfile
              ? undefined
              : () => favoriteMutation.mutate({ matchId: m.match_id, favorite: !m.is_favorite })
          }
          favoriteDisabled={favoriteMutation.isPending && favoriteMutation.variables?.matchId === m.match_id}
        />
      </CarouselItem>
    )
  }

  return (
    <section className="flex flex-col gap-3">
      <header className="flex items-center justify-between gap-3">
        <SectionTitle>
          {tab === 'recent' ? t('home.matches.recent_title') : t('home.matches.favorites_title')}
        </SectionTitle>
        <div className="flex items-center gap-1 border-b border-transparent">
          <Button variant="ghost" size="sm" onClick={() => setTab('recent')} className={tabClass(tab === 'recent')}>
            {t('home.matches.tab_recent')}
          </Button>
          <Button variant="ghost" size="sm" onClick={() => setTab('favorites')} className={tabClass(tab === 'favorites')}>
            {t('home.matches.tab_favorites')}
            {favoriteMatches.length > 0 && (
              <span className="ml-1.5 rounded-full bg-warning/20 px-1.5 py-0.5 text-2xs text-warning">
                {favoriteMatches.length}
              </span>
            )}
          </Button>
        </div>
      </header>
      {tab === 'recent' ? (
        tiles.tiles.length > 0 ? (
          <>
            <Carousel
              onNearEnd={tiles.hasMore ? tiles.fetchMore : undefined}
              loadingMore={tiles.loadingMore}
            >
              {tiles.tiles.map((m) => card(m, 'home_recent'))}
            </Carousel>
            {tiles.loadError && (
              <div role="alert" className="flex items-center gap-3 text-xs text-muted-foreground">
                <span>{t('home.matches.load_error')}</span>
                <Button variant="ghost" size="sm" onClick={tiles.retry}>
                  {t('home.matches.load_retry')}
                </Button>
              </div>
            )}
          </>
        ) : (
          <EmptyStateNotice
            title={t('home.matches.recent_empty_title')}
            description={t('home.matches.recent_empty_description')}
          />
        )
      ) : favoriteMatches.length > 0 ? (
        <Carousel>{favoriteMatches.map((m) => card(m, 'home_favorites'))}</Carousel>
      ) : (
        <EmptyStateNotice
          title={t('home.matches.favorites_empty_title')}
          description={t('home.matches.favorites_empty_description')}
        />
      )}
    </section>
  )
}
