/**
 * HomeAscensionWidget — aperçu compact des streaks actives sur la home page.
 *
 * Affiche les streaks non-cassées (active + paused) sous forme de mini-cartes
 * (`StreakCard` mode compact, partagé avec la page Ascension) : type, statut,
 * longueur, multiplicateur PP, prochain palier, record et boucliers. Lien vers
 * la page Ascension complète.
 */
import { Link } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { useStreaks } from '@/features/ascension/queries'
import { getAscensionText } from '@/features/ascension/i18n'
import { StreakCard } from '@/features/ascension/StreakCard'
import type { ManifestLocale } from '@/lib/i18n/format'

interface HomeAscensionWidgetProps {
  playerSlug: string
  locale: ManifestLocale
}

function FlameIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z" />
    </svg>
  )
}

export function HomeAscensionWidget({ playerSlug, locale }: HomeAscensionWidgetProps) {
  const t = getAscensionText(locale)
  const { data, isLoading, isError } = useStreaks(playerSlug, !!playerSlug)

  if (isError) return null

  const streaks = isLoading ? [] : (data?.items ?? []).filter((s) => s.status !== 'broken')

  return (
    <section className="flex h-full flex-col gap-3">
      {/* Titre de section (type 1) sorti de la carte (cf. demande user). */}
      <header className="flex items-center gap-1.5">
        <h3 className="flex items-center gap-1.5 text-base font-semibold text-foreground">
          <FlameIcon />
          Ascension
        </h3>
      </header>
      <Card className="relative flex flex-1 flex-col overflow-hidden isolate">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-cover bg-center bg-no-repeat opacity-30"
        style={{ backgroundImage: "url('/auntie-dot.webp')" }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-card/75"
      />
      <CardContent className="flex flex-1 flex-col gap-3 pt-6">
        {isLoading ? (
          <div className="space-y-2">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-20 animate-pulse rounded-md bg-muted" />
            ))}
          </div>
        ) : streaks.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.streaksEmpty}</p>
        ) : (
          <ul className="space-y-2">
            {streaks.map((s) => (
              <li key={s.id}>
                <StreakCard streak={s} locale={locale} t={t} compact />
              </li>
            ))}
          </ul>
        )}
        {/* Le widget montre des séries (surface du passé) → pointe l'onglet
            « Réalisations », pas l'index Profil (B2). */}
        <Link
          to="/players/$playerSlug/ascension/realisations"
          params={{ playerSlug }}
          className="mt-auto text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          {locale === 'en' ? 'View all →' : 'Voir tout →'}
        </Link>
      </CardContent>
      </Card>
    </section>
  )
}
