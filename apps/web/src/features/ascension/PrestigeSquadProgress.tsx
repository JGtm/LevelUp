// cross-feature-allow: bloc Progression Prestige de la tab Réalisations —
// consomme queryKeys.prestige (lib/query) + PRESTIGE_LEVEL_NAMES_FALLBACK depuis
// features/prestige et useSettings (friend_gamertags) depuis features/settings.
/**
 * PrestigeSquadProgress — bloc "Progression Prestige" de la tab Réalisations.
 *
 * Affiche la barre de progression Prestige du joueur courant + celles des amis
 * de l'escouade (Settings → friend_gamertags), triées par PP décroissant.
 *
 * Rendu calqué sur la barre de rang de carrière / Prestige de la home :
 * CompositeProgressBar + grille [valeur courante | barre | cible]. Légende
 * couleurs identique (tokenCssVar info/success, gérée par CompositeProgressBar).
 *
 * Les PP n'existent que pour les joueurs trackés : un ami absent de
 * availablePlayers (donc sans DB locale) n'a pas de UserPrestige et est
 * silencieusement omis. Si aucune barre n'a de données (PRESTIGE_ENABLED=false
 * ou squad vide), le bloc ne s'affiche pas.
 */
import { useMemo } from 'react'
import { useQueries } from '@tanstack/react-query'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { useSettings } from '@/features/settings/queries'
import { queryKeys } from '@/lib/query/keys'
import { PRESTIGE_LEVEL_NAMES_FALLBACK } from '@/features/prestige/fallback.i18n'
import { prestigeApi, type UserPrestige } from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import type { PlayerSummary } from '@/lib/api/types'

interface RowData {
  slug: string
  gamertag: string
  isMe: boolean
  prestige: UserPrestige
}

/**
 * Construit la liste ordonnée des slugs à interroger : le joueur courant en
 * tête, puis les amis de l'escouade résolus en player_slug via la roster.
 * Les gamertags non trackés (absents de la roster) sont ignorés.
 */
function resolveSquadSlugs(
  meSlug: string,
  friendGts: string[],
  players: PlayerSummary[],
): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  if (meSlug) {
    out.push(meSlug)
    seen.add(meSlug)
  }
  for (const gt of friendGts) {
    const match = players.find((p) => p.gamertag.toLowerCase() === gt.toLowerCase())
    if (match && !seen.has(match.player_slug)) {
      seen.add(match.player_slug)
      out.push(match.player_slug)
    }
  }
  return out
}

export function PrestigeSquadProgress() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const availablePlayers = useAppShellStore((s) => s.availablePlayers)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const locale = useAppShellStore((s) => s.locale)
  const { data: settings } = useSettings()

  const meSlug = currentPlayer?.player_slug ?? ''
  const friendGts = settings?.friend_gamertags
  const slugs = useMemo(
    () => resolveSquadSlugs(meSlug, friendGts ?? [], availablePlayers),
    [meSlug, friendGts, availablePlayers],
  )

  const results = useQueries({
    queries: slugs.map((slug) => ({
      queryKey: queryKeys.prestige.me(slug, titleSlug),
      queryFn: () => prestigeApi.getMyPrestige(slug, titleSlug),
      retry: false,
      enabled: !!slug,
    })),
  })

  if (results.some((r) => r.isLoading)) {
    return <BarsSkeleton count={slugs.length} />
  }

  const rows: RowData[] = slugs
    .map((slug, i) => {
      const data = results[i]?.data
      if (!data) return null
      const gamertag =
        slug === meSlug
          ? currentPlayer?.gamertag ?? slug
          : availablePlayers.find((p) => p.player_slug === slug)?.gamertag ?? slug
      return { slug, gamertag, isMe: slug === meSlug, prestige: data }
    })
    .filter((r): r is RowData => r != null)
    .sort((a, b) => b.prestige.total_pp - a.prestige.total_pp)

  if (rows.length === 0) return null

  const numberLocale = intlLocale(locale)

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {locale === 'en' ? 'Prestige progression' : 'Progression Prestige'}
      </h2>
      <div className="space-y-3">
        {rows.map((r) => (
          <SquadPrestigeRow key={r.slug} row={r} numberLocale={numberLocale} locale={locale} />
        ))}
      </div>
    </section>
  )
}

function SquadPrestigeRow({
  row,
  numberLocale,
  locale,
}: {
  row: RowData
  numberLocale: string
  locale: 'fr' | 'en'
}) {
  const { prestige, gamertag, isMe } = row
  const level = prestige.current_level
  const levelKey = String(level)
  const levelLabel = useAssetLabel('prestige_level', levelKey)
  const levelName =
    levelLabel !== levelKey
      ? levelLabel
      : PRESTIGE_LEVEL_NAMES_FALLBACK[level] ?? PRESTIGE_LEVEL_NAMES_FALLBACK[0]

  const lvl = prestige.level
  const isMax = lvl ? lvl.next_threshold_pp <= 0 : false
  const progressPct = lvl ? Math.round(lvl.progress_ratio * 100) : 0
  const nextPP = lvl?.next_threshold_pp ?? 0
  const maxLabel = locale === 'en' ? 'Max tier' : 'Niveau max'

  return (
    <div
      className={[
        'space-y-2 rounded-lg border p-3',
        isMe
          ? 'border-primary/60 bg-background/60'
          : 'border-border bg-background/30 opacity-80',
      ].join(' ')}
    >
      <div className="flex items-center justify-between gap-3 text-xs">
        <span className="min-w-0 truncate font-semibold text-foreground">
          {gamertag}
          {isMe && (
            <span className="ml-1.5 text-2xs uppercase tracking-wide text-primary">
              {locale === 'en' ? 'you' : 'moi'}
            </span>
          )}
          <span className="ml-2 font-normal text-muted-foreground">{levelName}</span>
        </span>
        <span className="shrink-0 text-muted-foreground">
          {isMax ? maxLabel : `${progressPct}%`}
        </span>
      </div>
      <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
        <span className="shrink-0 whitespace-nowrap text-3xs font-medium tabular-nums text-foreground/85">
          {prestige.total_pp.toLocaleString(numberLocale)} PP
        </span>
        <CompositeProgressBar value={progressPct} />
        <span className="shrink-0 whitespace-nowrap text-3xs font-medium tabular-nums text-foreground/85">
          {isMax ? '—' : `${nextPP.toLocaleString(numberLocale)} PP`}
        </span>
      </div>
    </div>
  )
}

function BarsSkeleton({ count }: { count: number }) {
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <div className="h-4 w-40 animate-pulse rounded bg-muted" />
      <div className="space-y-3">
        {Array.from({ length: Math.max(1, count) }).map((_, i) => (
          <div key={i} className="h-14 animate-pulse rounded-lg bg-muted" />
        ))}
      </div>
    </section>
  )
}
