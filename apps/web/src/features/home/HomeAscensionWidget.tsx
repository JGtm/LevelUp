/**
 * HomeAscensionWidget — aperçu compact des streaks actives sur la home page.
 *
 * Affiche toutes les streaks non-cassées (active + paused) avec leur longueur
 * courante et le multiplicateur PP associé. Lien vers la page Ascension complète.
 */
import { Link } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { useStreaks } from '@/features/ascension/queries'
import { getAscensionText } from '@/features/ascension/i18n'
import { formatMultiplier } from '@/features/ascension/format'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { Streak } from '@/features/ascension/types'

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

function StreakRow({ streak, typeName }: { streak: Streak; typeName: string }) {
  const dimmed = streak.status !== 'active'
  return (
    <li className={`flex items-center gap-2 text-sm ${dimmed ? 'opacity-60' : ''}`}>
      <span className="min-w-0 flex-1 truncate">{typeName}</span>
      <span className="shrink-0 font-semibold tabular-nums">{streak.current_length}</span>
      <span className="w-11 shrink-0 text-right text-xs text-muted-foreground tabular-nums">
        {formatMultiplier(streak.pp_multiplier)}
      </span>
    </li>
  )
}

export function HomeAscensionWidget({ playerSlug, locale }: HomeAscensionWidgetProps) {
  const t = getAscensionText(locale)
  const { data, isLoading, isError } = useStreaks(playerSlug, !!playerSlug)

  if (isError) return null

  const streaks = isLoading ? [] : (data?.items ?? []).filter((s) => s.status !== 'broken')

  return (
    <Card className="relative flex flex-col overflow-hidden isolate">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-cover bg-center bg-no-repeat opacity-30"
        style={{ backgroundImage: "url('/auntie-dot.webp')" }}
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 -z-10 bg-card/75"
      />
      <CardHeader className="pb-2">
        <CardTitle className="flex items-center gap-1.5 text-base">
          <FlameIcon />
          Ascension
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-3">
        {isLoading ? (
          <div className="space-y-2">
            {[0, 1, 2].map((i) => (
              <div key={i} className="h-5 animate-pulse rounded bg-muted" />
            ))}
          </div>
        ) : streaks.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.streaksEmpty}</p>
        ) : (
          <ul className="space-y-2">
            {streaks.map((s) => (
              <StreakRow key={s.id} streak={s} typeName={t.streakTypeName[s.type]} />
            ))}
          </ul>
        )}
        <Link
          to="/players/$playerSlug/ascension"
          params={{ playerSlug }}
          className="mt-auto text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          {locale === 'en' ? 'View all →' : 'Voir tout →'}
        </Link>
      </CardContent>
    </Card>
  )
}
