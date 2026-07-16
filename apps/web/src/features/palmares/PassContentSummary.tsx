/**
 * PassContentSummary — résumé visuel du contenu d'un pass saisonnier.
 *
 * Trois rangées de chips compactes :
 *   1. Devises          → paliers, cR, Pts Spartan, XP Boosts, Relances
 *   2. Items            → pièces d'armure, cosmétiques (split type-aware)
 *   3. Raretés          → chips avec point coloré par tier (Légendaire, Épique…)
 *
 * Mode `compact` (cartes secondaires) : seules les rangées 1 et 2 sont rendues.
 * En mode complet (showcase actif), les tags de catégories d'items sont ajoutés
 * en bas pour la granularité fine.
 *
 * Mode « acquis » : si `remaining` est fourni (contenu des paliers PAS encore
 * atteints), chaque valeur s'affiche en « acquis/total » (XX/YY), où acquis =
 * total − restant (reflète la complétion, cf. demande user). Sinon, totaux seuls.
 * `remaining` null ⇒ restant = 0 ⇒ acquis = total (ex. rang max, tout débloqué).
 */
import type { SeasonPassContentSummary } from '@/lib/api/types'

import { isArmorItemType, itemTypeLabel, rarityLabel, rarityStyle, type RarityTier } from './rarity'

const RARITY_ORDER: RarityTier[] = ['mythic', 'legendary', 'epic', 'rare', 'common']

export interface ContentLabels {
  creditsLabel: string
  spartanPointsLabel: string
  xpBoostsLabel: string
  challengeSwapsLabel: string
  cosmeticsLabel: string
  armorLabel: string
  cosmeticsSplitLabel: string
  tiersLabel: string
  rarityTitle: string
  typeTitle: string
}

/** Formateur de valeur : « restant/total » en mode restant, sinon « total ». */
type ValueFormatter = (total: number, remaining: number) => string

interface Chip {
  key: string
  value: string
  label: string
}

/**
 * Sépare le total des items en deux : pièces d'armure (Armor* + SpartanBody)
 * et cosmétiques purs (le reste : armes, véhicules, profil, IA).
 * Si aucune méta de type n'est disponible, retourne null pour afficher le total brut.
 */
function splitArmorVsCosmetic(
  total: number | null | undefined,
  breakdown: Record<string, number> | null | undefined,
): { armor: number; cosmetic: number } | null {
  if (!total || total <= 0) return null
  if (!breakdown || Object.keys(breakdown).length === 0) return null
  let armor = 0
  for (const [type, count] of Object.entries(breakdown)) {
    if (isArmorItemType(type)) armor += count
  }
  const cosmetic = Math.max(0, total - armor)
  return { armor, cosmetic }
}

function buildCurrencyChips(
  content: SeasonPassContentSummary,
  remaining: SeasonPassContentSummary | null,
  fmt: ValueFormatter,
  labels: ContentLabels,
): Chip[] {
  const chips: Chip[] = []
  if (content.total_tiers > 0) {
    chips.push({ key: 'tiers', value: fmt(content.total_tiers, remaining?.total_tiers ?? 0), label: labels.tiersLabel })
  }
  if (content.credits) {
    chips.push({ key: 'cr', value: fmt(content.credits, remaining?.credits ?? 0), label: labels.creditsLabel })
  }
  if (content.spartan_points) {
    chips.push({ key: 'sp', value: fmt(content.spartan_points, remaining?.spartan_points ?? 0), label: labels.spartanPointsLabel })
  }
  if (content.xp_boosts) {
    chips.push({ key: 'xp', value: fmt(content.xp_boosts, remaining?.xp_boosts ?? 0), label: labels.xpBoostsLabel })
  }
  if (content.challenge_swaps) {
    chips.push({ key: 'swap', value: fmt(content.challenge_swaps, remaining?.challenge_swaps ?? 0), label: labels.challengeSwapsLabel })
  }
  return chips
}

function buildItemChips(
  content: SeasonPassContentSummary,
  remaining: SeasonPassContentSummary | null,
  fmt: ValueFormatter,
  labels: ContentLabels,
): Chip[] {
  const chips: Chip[] = []
  const splitT = splitArmorVsCosmetic(content.cosmetics_total, content.type_breakdown)
  if (splitT) {
    const splitR = splitArmorVsCosmetic(remaining?.cosmetics_total, remaining?.type_breakdown) ?? { armor: 0, cosmetic: 0 }
    if (splitT.armor > 0) chips.push({ key: 'armor', value: fmt(splitT.armor, splitR.armor), label: labels.armorLabel })
    if (splitT.cosmetic > 0) chips.push({ key: 'cosmetic', value: fmt(splitT.cosmetic, splitR.cosmetic), label: labels.cosmeticsSplitLabel })
  } else if (content.cosmetics_total) {
    chips.push({ key: 'cosmetics', value: fmt(content.cosmetics_total, remaining?.cosmetics_total ?? 0), label: labels.cosmeticsLabel })
  }
  return chips
}

function StatChip({ value, label }: { value: string; label: string }) {
  return (
    <span className="flex items-baseline gap-1">
      <span className="text-sm font-semibold tabular-nums text-foreground">{value}</span>
      <span className="text-xs text-muted-foreground">{label}</span>
    </span>
  )
}

function ChipRow({ chips, compact = false }: { chips: Chip[]; compact?: boolean }) {
  if (chips.length === 0) return null
  const gap = compact ? 'gap-x-3' : 'gap-x-5'
  return (
    <div className={`flex flex-wrap items-center ${gap} gap-y-1.5`}>
      {chips.map((chip, i) => (
        <span key={chip.key} className={`flex items-center ${gap}`}>
          <StatChip value={chip.value} label={chip.label} />
          {i < chips.length - 1 && <span className="h-3 w-px bg-border/60" aria-hidden="true" />}
        </span>
      ))}
    </div>
  )
}

function RarityChips({
  breakdown,
  remaining,
  fmt,
  locale,
}: {
  breakdown: Record<string, number>
  remaining: Record<string, number> | null | undefined
  fmt: ValueFormatter
  locale: string
}) {
  const ordered = RARITY_ORDER
    .map((tier) => ({ tier, count: breakdown[tier] ?? 0, rem: remaining?.[tier] ?? 0 }))
    .filter((e) => e.count > 0)

  if (ordered.length === 0) return null

  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
      {ordered.map(({ tier, count, rem }, i) => {
        const styles = rarityStyle(tier)
        return (
          <span key={tier} className="flex items-center gap-x-4">
            <span className="flex items-center gap-1.5">
              <span className={`inline-block h-2 w-2 shrink-0 rounded-full ${styles?.segment ?? 'bg-muted'}`} />
              <span className="text-xs text-muted-foreground">
                {rarityLabel(tier, locale)}{' '}
                <span className="font-semibold tabular-nums text-foreground">{fmt(count, rem)}</span>
              </span>
            </span>
            {i < ordered.length - 1 && <span className="h-3 w-px bg-border/60" aria-hidden="true" />}
          </span>
        )
      })}
    </div>
  )
}

function TypeTags({ breakdown, locale, title }: { breakdown: Record<string, number>; locale: string; title: string }) {
  const sorted = Object.entries(breakdown).sort((a, b) => b[1] - a[1]).slice(0, 6)
  if (sorted.length === 0) return null

  return (
    <div className="space-y-2">
      <p className="text-3xs uppercase tracking-label-md text-muted-foreground">{title}</p>
      <div className="flex flex-wrap gap-1.5">
        {sorted.map(([type, count]) => (
          <span
            key={type}
            className="inline-flex items-center gap-1.5 rounded border border-border/50 bg-muted/40 px-2 py-0.5 text-3xs"
          >
            <span className="text-muted-foreground">{itemTypeLabel(type, locale) ?? type}</span>
            <span className="font-medium tabular-nums text-foreground">{count.toLocaleString(locale)}</span>
          </span>
        ))}
      </div>
    </div>
  )
}

export function PassContentSummary({
  content,
  remaining,
  labels,
  locale,
  compact = false,
}: {
  content: SeasonPassContentSummary
  /** Contenu restant (paliers non atteints). Fourni ⇒ affichage « restant/total » (XX/YY). */
  remaining?: SeasonPassContentSummary | null
  labels: ContentLabels
  locale: string
  compact?: boolean
}) {
  const showRemaining = remaining !== undefined
  const rem = remaining ?? null
  // Mode XX/YY : affiche l'ACQUIS (= total − restant) sur le total (cf. demande user :
  // « X obtenus sur Y », colle à la complétion). r = valeur des paliers NON atteints.
  const fmt: ValueFormatter = (total, r) =>
    showRemaining ? `${Math.max(0, total - r).toLocaleString(locale)}/${total.toLocaleString(locale)}` : total.toLocaleString(locale)

  const currencyChips = buildCurrencyChips(content, rem, fmt, labels)
  const itemChips = buildItemChips(content, rem, fmt, labels)
  const hasRarity = content.rarity_breakdown && Object.keys(content.rarity_breakdown).length > 0
  const hasTypes = !compact && content.type_breakdown && Object.keys(content.type_breakdown).length > 0

  if (currencyChips.length === 0 && itemChips.length === 0 && !hasRarity) return null

  if (compact) {
    // Items (cosmétiques) avant les devises (paliers/cR/XP), cf. demande user.
    const allChips = [...itemChips, ...currencyChips]
    if (allChips.length === 0 && !hasRarity) return null
    return (
      <div className="space-y-2">
        {/* Raretés EN PREMIER, puis la ligne combinée cosmétiques/devises (cf. demande user). */}
        {hasRarity && <RarityChips breakdown={content.rarity_breakdown!} remaining={rem?.rarity_breakdown} fmt={fmt} locale={locale} />}
        {allChips.length > 0 && <ChipRow chips={allChips} compact />}
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {/* Raretés EN PREMIER, puis items (cosmétiques), puis devises (cf. demande user). */}
      {hasRarity && <RarityChips breakdown={content.rarity_breakdown!} remaining={rem?.rarity_breakdown} fmt={fmt} locale={locale} />}
      <ChipRow chips={itemChips} />
      <ChipRow chips={currencyChips} />
      {hasTypes && <TypeTags breakdown={content.type_breakdown!} locale={locale} title={labels.typeTitle} />}
    </div>
  )
}
