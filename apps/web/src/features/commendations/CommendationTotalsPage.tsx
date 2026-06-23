/**
 * CommendationTotalsPage — totaux à vie des commendations NATIVES (Halo 5, AXE B).
 * Le total d'une commendation = le dernier progress absolu du joueur. Groupées par
 * catégorie : icône + nom + total. Réponse vide pour les titres sans commendations
 * natives → état vide.
 */
import { useEffect } from 'react'
import { useParams } from '@tanstack/react-router'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { MedalIcon } from '@/components/ui/MedalIcon'
import { useAppShellStore } from '@/stores/appShellStore'
import { useCommendationTotals } from './queries'
import { COMMENDATIONS_TEXT } from './i18n'
import type { NativeCommendationTotal } from '@/lib/api/types'

export function CommendationTotalsPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale)
  const t = COMMENDATIONS_TEXT[locale]

  useEffect(() => {
    document.title = `LevelUp - ${t.pageTitle}`
    return () => {
      document.title = 'LevelUp'
    }
  }, [locale, t.pageTitle])

  const { data, isLoading, isError } = useCommendationTotals(playerSlug)

  if (isLoading) {
    return <div className="px-6 py-8 text-sm text-muted-foreground">…</div>
  }

  const categories = data?.categories ?? []
  const isEmpty = isError || categories.length === 0

  return (
    <div className="flex flex-col gap-6 px-6 py-6">
      <header className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold text-foreground">{t.pageTitle}</h1>
        <p className="text-sm text-muted-foreground">{t.subtitle}</p>
        {!isEmpty && (
          <p className="text-xs text-muted-foreground">
            {t.commendationsCount(data?.total_count ?? 0)}
          </p>
        )}
      </header>

      {isEmpty ? (
        <EmptyStateCard title={t.empty} description={t.emptyHint} />
      ) : (
        <div className="flex flex-col gap-6">
          {categories.map((cat) => (
            <section
              key={cat.category}
              className="rounded-lg border border-border bg-card"
            >
              <div className="border-b border-border px-4 py-2 text-sm font-medium text-foreground">
                {cat.category}
              </div>
              <div className="flex flex-wrap gap-x-5 gap-y-4 p-4">
                {(cat.items ?? []).map((c) => (
                  <CommendationTile
                    key={c.id}
                    commendation={c}
                    totalLabel={t.total(c.total)}
                  />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  )
}

function CommendationTile({
  commendation,
  totalLabel,
}: {
  commendation: NativeCommendationTotal
  totalLabel: string
}) {
  const label =
    commendation.name && commendation.name.trim() !== ''
      ? commendation.name
      : `#${commendation.id.slice(0, 8)}`
  return (
    <div
      title={label}
      className="flex w-[84px] cursor-default flex-col items-center gap-1"
    >
      {commendation.icon_url ? (
        <MedalIcon
          imageUrl={commendation.icon_url}
          label={label}
          size={52}
          className="object-contain"
        />
      ) : (
        <div className="flex h-[52px] w-[52px] items-center justify-center rounded bg-muted px-1">
          <span className="text-3xs leading-none break-all text-center text-muted-foreground">
            {label}
          </span>
        </div>
      )}
      <span className="w-full truncate text-center text-3xs leading-tight text-muted-foreground">
        {label}
      </span>
      <span className="text-sm font-semibold leading-none text-foreground">
        {totalLabel}
      </span>
    </div>
  )
}
