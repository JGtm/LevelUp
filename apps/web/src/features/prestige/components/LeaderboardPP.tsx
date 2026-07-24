/**
 * LeaderboardPP — composant Leaderboard PP entre amis dérivés.
 *
 * Référence : Axe 5 + Axe 8 du plan PLAN_challenges_xp_system.md.
 * Affichage décomposé par ligne : Score brut / + Bonus défis / Score total.
 *
 * Phase 5 minimale : version sans données réelles (le backend leaderboard
 * cross-amis dépend du wiring sources amis dérivées de squad + relations).
 * Affiche un état vide et la structure visuelle attendue.
 *
 * EXCEPTION tri client par en-têtes (I16) : pas de tri par en-tête pour
 * l'instant — scaffold sans données réelles (cf. ci-dessus), classement déjà
 * imposé (total_pp DESC). À rouvrir quand le backend sera branché.
 */
import type { Tier } from '@/lib/prestige'
import { TIER_COLORS, TIER_LABELS_FR } from '@/lib/prestige'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { intlLocale } from '@/lib/formatters'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export interface LeaderboardEntry {
  user_id: string
  gamertag: string
  raw_score: number
  bonus_score: number
  total_pp: number
  level_index: number
  level_name: string
  active_arc?: string
  /** Palier du dernier défi validé pour décoration. */
  last_tier?: Tier
}

interface LeaderboardPPProps {
  entries: LeaderboardEntry[]
  period?: 'week' | 'month' | 'all'
  onPeriodChange?: (p: 'week' | 'month' | 'all') => void
  /** Filtre arc actif (sinon tous). */
  arcFilter?: string
  arcChoices?: Array<{ id: string; title: string }>
  onArcFilterChange?: (id: string | undefined) => void
}

export function LeaderboardPP({
  entries,
  period = 'all',
  onPeriodChange,
  arcFilter,
  arcChoices = [],
  onArcFilterChange,
}: LeaderboardPPProps) {
  const sorted = [...entries].sort((a, b) => b.total_pp - a.total_pp)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <PeriodToggle value={period} onChange={onPeriodChange} />
        {arcChoices.length > 0 && onArcFilterChange && (
          <select
            value={arcFilter ?? ''}
            onChange={(e) => onArcFilterChange(e.target.value || undefined)}
            className="rounded-md border border-border bg-card px-2 py-1 text-xs text-card-foreground"
          >
            <option value="">{t('common.prestige.all_arcs')}</option>
            {arcChoices.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title}
              </option>
            ))}
          </select>
        )}
      </div>

      {sorted.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          {t('common.prestige.no_friends_for_period')}
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead className="border-b border-border bg-card text-xs uppercase text-muted-foreground">
              <tr>
                <th className="px-3 py-2 text-left">#</th>
                <th className="px-3 py-2 text-left">Joueur</th>
                <th className="px-3 py-2 text-right">Score brut</th>
                <th className="px-3 py-2 text-right">+ Bonus</th>
                <th className="px-3 py-2 text-right">Total</th>
                <th className="px-3 py-2 text-left">Niveau</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map((e, idx) => (
                <Row key={e.user_id} entry={e} rank={idx + 1} locale={locale} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function PeriodToggle({
  value,
  onChange,
}: {
  value: 'week' | 'month' | 'all'
  onChange?: (p: 'week' | 'month' | 'all') => void
}) {
  if (!onChange) return null
  return (
    <div className="flex items-center rounded-md border border-border bg-card p-0.5 text-xs">
      {(['week', 'month', 'all'] as const).map((opt) => (
        <button
          key={opt}
          type="button"
          onClick={() => onChange(opt)}
          className={[
            'rounded px-2 py-1 transition-colors',
            value === opt
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:text-foreground',
          ].join(' ')}
        >
          {opt === 'week' ? 'Semaine' : opt === 'month' ? 'Mois' : 'Tout'}
        </button>
      ))}
    </div>
  )
}

function Row({ entry, rank, locale }: { entry: LeaderboardEntry; rank: number; locale: ManifestLocale }) {
  const tierColor = entry.last_tier ? TIER_COLORS[entry.last_tier] : undefined
  // Phase 4 plan finition multi-titres : libellé du tier via TOML, fallback dict.
  const tierLabelFromTOML = useAssetLabel('challenge_tier', entry.last_tier ?? '')
  const tierLabel = entry.last_tier
    ? (tierLabelFromTOML !== entry.last_tier ? tierLabelFromTOML : TIER_LABELS_FR[entry.last_tier])
    : ''
  return (
    <tr className="border-b border-border last:border-0 hover:bg-accent/40">
      <td className="px-3 py-2 text-muted-foreground">{rank}</td>
      <td className="px-3 py-2">
        <div className="flex items-center gap-2">
          <span className="font-medium">{entry.gamertag}</span>
          {entry.last_tier && (
            <span
              className="rounded-full px-1.5 py-0.5 text-2xs uppercase"
              style={{ backgroundColor: `${tierColor}20`, color: tierColor }}
            >
              {tierLabel}
            </span>
          )}
        </div>
      </td>
      <td className="px-3 py-2 text-right tabular-nums">{entry.raw_score.toFixed(2)}</td>
      <td className="px-3 py-2 text-right tabular-nums text-muted-foreground">
        +{entry.bonus_score.toFixed(2)}
      </td>
      <td className="px-3 py-2 text-right font-bold tabular-nums">
        {(entry.raw_score + entry.bonus_score).toFixed(2)}
      </td>
      <td className="px-3 py-2 text-xs">
        <span className="font-medium">{entry.level_name}</span>
        <span className="ml-1 text-muted-foreground">
          ({entry.total_pp.toLocaleString(intlLocale(locale))} PP)
        </span>
      </td>
    </tr>
  )
}
