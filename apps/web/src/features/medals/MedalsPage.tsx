/**
 * MedalsPage — conteneur de la sous-page Carrière « Médailles » (Lot A2).
 *
 * Modèle : page Citations. Affiche TOUTES les médailles du titre (dont celles
 * jamais obtenues), regroupées à la SpartanRecord (super-sections → catégories).
 * Le filtre (toutes / obtenues / non-obtenues) et le tri (total par catégorie /
 * nombre par médaille / nom) sont 100% CLIENT sur les groupes déjà renvoyés triés
 * par le backend (données bornées). En-tête « Médailles — {obtenues}/{catalogue} ».
 */
import { useMemo, useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { intlLocale } from '@/lib/formatters'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { medalsManifest, type MedalsManifestKey } from '@/lib/i18n/generated/medals'
import { useAppShellStore } from '@/stores/appShellStore'
import { useMedalsPage } from './queries'
import { medalCategoryLabel, medalSuperSectionLabel } from './labels'
import {
  MedalsView,
  type MedalCategoryView,
  type MedalSuperSectionView,
  type MedalsViewModel,
} from './MedalsView'
import type { MedalCategoryGroup, MedalSummaryItem } from '@/lib/api/types'

type MedalFilter = 'all' | 'obtained' | 'not_obtained'
type MedalSort = 'category_total' | 'medal_count' | 'medal_name'

const FILTERS: readonly MedalFilter[] = ['all', 'obtained', 'not_obtained']
const SORTS: readonly MedalSort[] = ['category_total', 'medal_count', 'medal_name']

export function MedalsPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: MedalsManifestKey, vars?: Record<string, unknown>) =>
    formatMessage(medalsManifest, key, locale, vars)

  const [filter, setFilter] = useState<MedalFilter>('all')
  const [sort, setSort] = useState<MedalSort>('category_total')

  const { data, isLoading, isError, refetch } = useMedalsPage(playerSlug)

  const vm = useMemo(
    () => (data ? buildViewModel(data.categories ?? [], filter, sort, locale) : null),
    [data, filter, sort, locale],
  )

  if (isLoading) {
    return <div className="px-6 py-8 text-sm text-muted-foreground">…</div>
  }
  if (isError) {
    return (
      <div className="px-6 py-8">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">{t('medals.errors.load_failed')}</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              {t('medals.errors.retry')}
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }
  if (!data || !vm) {
    return (
      <div className="px-6 py-8">
        <EmptyStateCard
          title={t('medals.empty.no_data')}
          description={t('medals.empty.no_data_description')}
          actionLabel={t('medals.errors.retry')}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 py-4">
      <div className="flex flex-col gap-3 px-6">
        <h2 className="text-sm font-semibold text-foreground">
          {t('medals.header.title', { earned: data.earned_total, total: data.catalog_total })}
        </h2>
        <MedalsToolbar filter={filter} onFilter={setFilter} sort={sort} onSort={setSort} locale={locale} />
      </div>
      <MedalsView
        vm={vm}
        locale={locale}
        emptyTitle={t('medals.empty.no_items')}
        emptyDescription={t('medals.empty.no_items_description')}
      />
    </div>
  )
}

// ─── Barre filtre + tri (client) ─────────────────────────────────────────────

interface MedalsToolbarProps {
  filter: MedalFilter
  onFilter: (f: MedalFilter) => void
  sort: MedalSort
  onSort: (s: MedalSort) => void
  locale: ManifestLocale
}

function MedalsToolbar({ filter, onFilter, sort, onSort, locale }: MedalsToolbarProps) {
  const t = (key: MedalsManifestKey) => formatMessage(medalsManifest, key, locale)
  return (
    <div className="flex flex-wrap items-center gap-3">
      <div
        role="group"
        aria-label={t('medals.filter.label')}
        className="inline-flex rounded-md border border-border bg-card p-0.5"
      >
        {FILTERS.map((f) => (
          <button
            key={f}
            type="button"
            aria-pressed={filter === f}
            onClick={() => onFilter(f)}
            className={`rounded px-2.5 py-1 text-xs font-medium transition-colors ${
              filter === f ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted'
            }`}
          >
            {t(`medals.filter.${f}` as MedalsManifestKey)}
          </button>
        ))}
      </div>
      <label className="flex items-center gap-1.5 text-xs text-muted-foreground">
        {t('medals.sort.label')}
        <select
          value={sort}
          onChange={(e) => onSort(e.target.value as MedalSort)}
          className="rounded-md border border-input bg-background px-2 py-1 text-xs text-foreground"
        >
          {SORTS.map((s) => (
            <option key={s} value={s}>
              {t(`medals.sort.${s}` as MedalsManifestKey)}
            </option>
          ))}
        </select>
      </label>
    </div>
  )
}

// ─── Transformation client (filtre + tri + regroupement + localisation) ───────

function matchesFilter(item: MedalSummaryItem, filter: MedalFilter): boolean {
  if (filter === 'obtained') return item.count > 0
  if (filter === 'not_obtained') return item.count === 0
  return true
}

/** Tri des médailles DANS une catégorie. category_total → ordre backend conservé. */
function sortItems(items: MedalSummaryItem[], sort: MedalSort, locale: ManifestLocale): MedalSummaryItem[] {
  if (sort === 'medal_count') return [...items].sort((a, b) => b.count - a.count)
  if (sort === 'medal_name') {
    const collator = new Intl.Collator(intlLocale(locale), { sensitivity: 'base' })
    return [...items].sort((a, b) => collator.compare(a.name, b.name))
  }
  return items
}

interface FilteredGroup {
  group: MedalCategoryGroup
  items: MedalSummaryItem[]
}

/** Tri des catégories DANS une super-section. category_total → par total décroissant. */
function sortGroups(groups: FilteredGroup[], sort: MedalSort): FilteredGroup[] {
  if (sort !== 'category_total') return groups
  return [...groups].sort((a, b) => b.group.total_count - a.group.total_count)
}

function toCategoryView(fg: FilteredGroup, sort: MedalSort, locale: ManifestLocale): MedalCategoryView {
  const { group } = fg
  const pct = group.total > 0 ? (group.earned / group.total) * 100 : 0
  return {
    key: group.category,
    label: medalCategoryLabel(group.category, locale),
    pct,
    isMastered: group.total > 0 && group.earned === group.total,
    masteryLabel: formatMessage(medalsManifest, 'medals.category.mastery', locale, {
      earned: group.earned,
      total: group.total,
    }),
    totalAwardedLabel: formatMessage(medalsManifest, 'medals.category.total_awarded', locale, {
      count: group.total_count.toLocaleString(intlLocale(locale)),
    }),
    items: sortItems(fg.items, sort, locale),
  }
}

/**
 * Construit le view-model groupé/trié/localisé. Regroupe les catégories par
 * super-section en préservant l'ordre de PREMIÈRE APPARITION du backend (déjà
 * ordonné super-section→catégorie, « other » en dernier — title-agnostic, aucune
 * liste de clés en dur). Les catégories/super-sections vides après filtre sont
 * masquées.
 */
function buildViewModel(
  categories: MedalCategoryGroup[],
  filter: MedalFilter,
  sort: MedalSort,
  locale: ManifestLocale,
): MedalsViewModel {
  const bySection = new Map<string, FilteredGroup[]>()
  const order: string[] = []
  for (const group of categories) {
    const items = (group.items ?? []).filter((it) => matchesFilter(it, filter))
    if (items.length === 0) continue
    if (!bySection.has(group.super_section)) {
      bySection.set(group.super_section, [])
      order.push(group.super_section)
    }
    bySection.get(group.super_section)!.push({ group, items })
  }

  const superSections: MedalSuperSectionView[] = order.map((key) => ({
    key,
    label: medalSuperSectionLabel(key, locale),
    categories: sortGroups(bySection.get(key) ?? [], sort).map((fg) => toCategoryView(fg, sort, locale)),
  }))
  return { superSections }
}
