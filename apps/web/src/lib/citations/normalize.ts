/**
 * Normaliseurs des données brutes citations/commendations vers le view-model partagé
 * (CitationDisplayItem / CitationsViewModel). SEULS appelants de citationMastery() pour
 * les données de page — l'UI consomme ensuite `.isMastered` pré-calculé. Tout titre
 * ajoute son normaliseur ici sans toucher à l'UI.
 */
import type {
  CitationItem,
  CitationsPageResponse,
  NativeCommendationsTotalsResponse,
  NativeCommendationTotal,
} from '@/lib/api/types'
import { citationMastery } from './mastery'
import type {
  CitationCategoryView,
  CitationDisplayItem,
  CitationsViewModel,
} from './types'

// ─── Items (un par citation/commendation) ───────────────────────────────────

function fromInfiniteItem(c: CitationItem): CitationDisplayItem {
  return {
    key: c.name_norm,
    name: c.name_display,
    description: c.description ?? undefined,
    imageUrl: c.image_url ?? undefined,
    pct: c.mastery_pct,
    tierIndex: c.earned_tiers,
    tierCount: c.tier_count,
    total: c.total,
    nextTierTarget: c.next_tier_target,
    isMastered: citationMastery(c),
    isNewlyMastered: false,
    source: 'infinite',
  }
}

function fromNativeTotal(c: NativeCommendationTotal): CitationDisplayItem {
  const name = c.name && c.name.trim() !== '' ? c.name : `#${c.id.slice(0, 8)}`
  return {
    key: c.id,
    name,
    imageUrl: c.icon_url ?? undefined,
    pct: c.progress_pct,
    tierIndex: c.tier_index ?? 0,
    tierCount: c.tier_count ?? 0,
    total: c.total,
    nextTierTarget: c.next_tier_target ?? 0,
    isMastered: citationMastery(c),
    isNewlyMastered: false,
    source: 'native',
  }
}

// ─── Pages (groupées par catégorie) ─────────────────────────────────────────

export function normalizeInfinitePage(resp: CitationsPageResponse): CitationsViewModel {
  const groups = resp.citations_by_category ?? []
  const categories: CitationCategoryView[] = groups.map((g) => ({
    category: g.category,
    items: (g.items ?? []).map(fromInfiniteItem),
    completed: g.completed,
  }))
  return {
    categories,
    masteredTotal: categories.reduce((acc, c) => acc + c.completed, 0),
    itemsTotal: (resp.citations ?? []).length,
    source: 'infinite',
    hasFilters: true,
  }
}

export function normalizeNativeTotals(
  resp: NativeCommendationsTotalsResponse,
): CitationsViewModel {
  const groups = resp.categories ?? []
  const categories: CitationCategoryView[] = groups.map((g) => {
    const items = (g.items ?? []).map(fromNativeTotal)
    return {
      category: g.category,
      items,
      // Natif : pas de `completed` côté API → dérivé de la maîtrise des items.
      completed: items.filter((i) => i.isMastered).length,
    }
  })
  return {
    categories,
    masteredTotal: categories.reduce((acc, c) => acc + c.completed, 0),
    itemsTotal: resp.total_count,
    source: 'native',
    hasFilters: false,
  }
}
