/**
 * HomeCitationsNearCompletion — section accueil « Citations bientôt terminées ».
 *
 * Placée juste au-dessus des tuiles de matchs. Sélectionne les citations les plus
 * proches de leur prochain palier via `selectNearCompletion` (cf. nearCompletion.ts),
 * sur le view-model partagé `CitationDisplayItem` — donc title-agnostic.
 *
 * DEUX sources, comme `UnifiedCitationsPage` :
 *   - Infinite : moteur de citations dérivé (`/pages/citations`).
 *   - Halo 5   : commendations NATIVES, totaux à vie (`/commendations/totals`).
 * Le SEUL signal front pour distinguer h5 est le slug courant (aucune capability
 * coarse ne le fait ; `citations.engine`/`commendations.native` sont fines et non
 * exposées au front) — même règle que NavL1/NavL2. Chaque source appelle EXACTEMENT
 * un data-hook → règles des hooks respectées.
 *
 * Frontière front-only : aucune plomberie backend (les deux endpoints existaient
 * déjà). La période du filtre Infinite est laissée à vie (DEFAULT_FILTER_CONTEXT,
 * bornes nulles + cascade vide) → totaux cumulés, cohérent avec « bientôt terminée ».
 *
 * Dégradation gracieuse : tant que la requête charge, ou si aucune citation ne
 * franchit le seuil de proximité, la section ne rend RIEN (pas de carte vide).
 */
import { useNavigate } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useCitationsPage } from '@/features/citations/queries'
import { useCommendationTotals } from '@/features/commendations/queries'
import { normalizeInfinitePage, normalizeNativeTotals } from '@/lib/citations/normalize'
import {
  allCitationsMastered,
  selectNearCompletion,
  type NearCompletionItem,
} from '@/lib/citations/nearCompletion'
import { DEFAULT_FILTER_CONTEXT } from '@/stores/createFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'

// Hash de cache STABLE et dédié à la home (contexte à vie). Distinct du hash de la
// page Citations → entrée react-query propre, sans collision avec ses filtres.
const HOME_NEAR_COMPLETION_HASH = 'home-near-completion'

export function HomeCitationsNearCompletion({ playerSlug }: { playerSlug: string }) {
  const isHalo5 = useAppShellStore((s) => s.currentTitleSlug === 'halo_5')
  return isHalo5 ? (
    <NativeNearCompletion playerSlug={playerSlug} />
  ) : (
    <InfiniteNearCompletion playerSlug={playerSlug} />
  )
}

// ─── Source Infinite (citations dérivées) ───────────────────────────────────

function InfiniteNearCompletion({ playerSlug }: { playerSlug: string }) {
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()
  const { data, isLoading } = useCitationsPage(
    playerSlug,
    { filters: DEFAULT_FILTER_CONTEXT },
    HOME_NEAR_COMPLETION_HASH,
  )
  if (isLoading || !data) return null
  const items = normalizeInfinitePage(data).categories.flatMap((c) => c.items)
  const near = selectNearCompletion(items)
  const allDone = allCitationsMastered(items)
  // On rend la section si une citation est en approche OU si tout est maîtrisé
  // (message de félicitations) ; sinon rien à montrer → self-hide.
  if (near.length === 0 && !allDone) return null
  return (
    <NearCompletionSection
      near={near}
      allDone={allDone}
      onSeeAll={() => void navigate({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/citations', params: { titleSlug, playerSlug } })}
    />
  )
}

// ─── Source native (commendations Halo 5, totaux à vie) ──────────────────────

function NativeNearCompletion({ playerSlug }: { playerSlug: string }) {
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()
  const { data, isLoading } = useCommendationTotals(playerSlug)
  if (isLoading || !data) return null
  const items = normalizeNativeTotals(data).categories.flatMap((c) => c.items)
  const near = selectNearCompletion(items)
  const allDone = allCitationsMastered(items)
  if (near.length === 0 && !allDone) return null
  return (
    <NearCompletionSection
      near={near}
      allDone={allDone}
      onSeeAll={() => void navigate({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/commendations', params: { titleSlug, playerSlug } })}
    />
  )
}

// ─── Rendu partagé ───────────────────────────────────────────────────────────

function NearCompletionSection({
  near,
  allDone,
  onSeeAll,
}: {
  near: NearCompletionItem[]
  allDone: boolean
  onSeeAll: () => void
}) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)

  return (
    <section className="flex flex-col gap-3">
      <header className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-1.5">
          <h3 className="text-base font-semibold text-foreground">
            {t('home.near_completion.section_title')}
          </h3>
          <InfoTooltip content={<p>{t('home.near_completion.section_tooltip')}</p>} />
        </div>
        <button
          type="button"
          onClick={onSeeAll}
          className="shrink-0 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
        >
          {t('home.near_completion.view_all')}
        </button>
      </header>
      {near.length > 0 ? (
        // Une seule ligne sur grand écran (jusqu'à 5 tuiles, cf. NEAR_COMPLETION_DEFAULT_LIMIT).
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-5">
          {near.map((n) => (
            <NearCompletionTile key={n.item.key} entry={n} locale={locale} t={t} onClick={onSeeAll} />
          ))}
        </div>
      ) : allDone ? (
        <EmptyStateNotice
          title={t('home.near_completion.all_complete_title')}
          description={t('home.near_completion.all_complete_description')}
        />
      ) : null}
    </section>
  )
}

function NearCompletionTile({
  entry,
  locale,
  t,
  onClick,
}: {
  entry: NearCompletionItem
  locale: ManifestLocale
  t: (key: HomeManifestKey, values?: Record<string, string | number>) => string
  onClick: () => void
}) {
  const { item, remaining, isFinalTier } = entry
  const numberLocale = intlLocale(locale)
  return (
    <button
      type="button"
      onClick={onClick}
      title={item.description ? `${item.name} : ${item.description}` : item.name}
      className="flex items-center gap-2.5 rounded-lg border bg-card/40 p-2 text-left transition-colors hover:bg-muted/40"
    >
      <CitationProgressRing pct={item.pct} imageUrl={item.imageUrl} isMastered={false} size={44} />
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium text-foreground">{item.name}</p>
        <p className="text-xs text-muted-foreground">
          {t('home.near_completion.remaining', { count: remaining.toLocaleString(numberLocale) })}
        </p>
        {isFinalTier ? (
          <p className="text-2xs font-semibold uppercase tracking-wide text-primary">
            {t('home.near_completion.final_tier')}
          </p>
        ) : (
          <p className="text-2xs text-muted-foreground/80">
            {t('home.near_completion.tier_progress', {
              index: item.tierIndex,
              count: item.tierCount,
            })}
          </p>
        )}
      </div>
    </button>
  )
}
