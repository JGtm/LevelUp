/**
 * ChallengesCarousel — carousel de défis pour la home page.
 *
 * Référence : Axe 3 + Axe 8 du plan conceptuel.
 * Placé au-dessus de la section "Faits Marquants".
 * Switch Actifs / Terminés + petit bouton "+ Nouveau".
 *
 * Phase 5 minimale : pas de scroll auto, pas de keyboard navigation.
 * Évoluera avec les vrais besoins UX au moment des maquettes finales.
 */
import { useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { prestigeApi, type Challenge } from '@/lib/prestige'
import { ChallengeCard } from './ChallengeCard'

interface ChallengesCarouselProps {
  userId: string
  titleSlug: string
  playerSlug: string
}

export function ChallengesCarousel({ userId, titleSlug, playerSlug }: ChallengesCarouselProps) {
  const [filter, setFilter] = useState<'active' | 'completed'>('active')

  const { data, isLoading, isError } = useQuery({
    queryKey: ['prestige', 'challenges', userId, titleSlug],
    queryFn: () => prestigeApi.listActiveChallenges(userId, titleSlug),
    // Si Prestige est désactivé côté backend (404), pas de retry agressif.
    retry: false,
    staleTime: 30_000,
  })

  // La feature peut être désactivée (PRESTIGE_ENABLED=false) → on rend rien.
  if (isError) {
    return null
  }

  const challenges: Challenge[] = data?.challenges ?? []
  const filtered = challenges.filter((c) =>
    filter === 'active' ? c.status === 'active' : c.status === 'completed',
  )

  return (
    <section className="space-y-3">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Mes défis</h2>
        <div className="flex items-center gap-2">
          <div className="flex items-center rounded-md border border-border bg-card p-0.5 text-xs">
            <button
              type="button"
              onClick={() => setFilter('active')}
              className={[
                'rounded px-2 py-1 transition-colors',
                filter === 'active' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground',
              ].join(' ')}
            >
              Actifs
            </button>
            <button
              type="button"
              onClick={() => setFilter('completed')}
              className={[
                'rounded px-2 py-1 transition-colors',
                filter === 'completed' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground',
              ].join(' ')}
            >
              Terminés
            </button>
          </div>
          <Link
            to={`/players/${playerSlug}/ascension` as never}
            className="rounded-md border border-border px-2 py-1 text-xs hover:bg-accent"
          >
            + Nouveau
          </Link>
        </div>
      </header>

      {isLoading ? (
        <div className="flex gap-2 overflow-x-auto pb-2">
          {[0, 1, 2].map((i) => (
            <div key={i} className="h-32 w-64 shrink-0 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          {filter === 'active'
            ? 'Aucun défi actif. Crée ton premier défi sur la page Objectifs.'
            : 'Aucun défi terminé récemment.'}
        </div>
      ) : (
        <div className="flex gap-2 overflow-x-auto pb-2">
          {filtered.map((c) => (
            <ChallengeCard key={c.id} challenge={c} />
          ))}
        </div>
      )}
    </section>
  )
}
