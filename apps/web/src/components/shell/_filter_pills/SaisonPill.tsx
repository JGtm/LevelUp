/**
 * SaisonPill — pill de filtre saison Halo Infinite.
 *
 * Sélectionner une saison applique sa fenêtre temporelle [startDate, endDate)
 * via setPeriod() (logique dans le caller, qui passe `onSelectSeason`).
 *
 * Folding "+N saisons sans matchs ▾" : les saisons avec count=0 sous la
 * cascade courante sont repliées sous un disclosure (pattern aligné sur
 * commit 8fd8574c côté FilterOmnibar).
 */
import { useMemo } from 'react'
import type { SeasonCount } from '@/lib/api/types'
import type { SeasonEntry } from '@/lib/i18n/fieldMappings'
import { useDismissable } from './_hooks'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

export interface SaisonPillProps {
  open: boolean
  onToggle: () => void
  onClose: () => void
  /** Catalog complet des saisons du titre courant (depuis useSeasons). */
  seasons: SeasonEntry[]
  /** Saison active = celle dont la fenêtre matche pile la `period` du store
   *  global. null si la fenêtre courante ne matche aucune saison. */
  activeSeason: SeasonEntry | null
  /** Counts cascade-aware fournis par le backend (FilterContextResolved.season_counts).
   *  Si absent : on n'affiche aucun count et le folding est désactivé (toutes
   *  les saisons sont visibles d'office). */
  seasonCounts?: SeasonCount[]
  onSelectSeason: (s: SeasonEntry) => void
  /** Optionnel : remettre la période à zéro (vide setPeriod). Utilisé par le
   *  bouton "Toutes saisons" en haut du popover. */
  onClear?: () => void
}

export function SaisonPill({
  open,
  onToggle,
  onClose,
  seasons,
  activeSeason,
  seasonCounts,
  onSelectSeason,
  onClear,
}: SaisonPillProps) {
  const ref = useDismissable(open, onClose)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  const countsByID = useMemo(() => {
    const m = new Map<string, number>()
    for (const c of seasonCounts ?? []) m.set(c.season_id, c.count)
    return m
  }, [seasonCounts])

  // Partition : saisons avec count > 0 (visibles d'office) vs count === 0 (foldées).
  // Si seasonCounts est absent, tout est "available" (pas de folding).
  const { available, unavailable } = useMemo(() => {
    if (!seasonCounts) {
      return { available: seasons, unavailable: [] as SeasonEntry[] }
    }
    const av: SeasonEntry[] = []
    const un: SeasonEntry[] = []
    for (const s of seasons) {
      const c = countsByID.get(s.id) ?? 0
      ;(c > 0 ? av : un).push(s)
    }
    return { available: av, unavailable: un }
  }, [seasons, seasonCounts, countsByID])

  const triggerLabel = activeSeason
    ? `${activeSeason.shortLabel} — ${activeSeason.label}`
    : t('common.filters.season_pill_label')

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        type="button"
        onClick={onToggle}
        aria-haspopup="dialog"
        aria-expanded={open}
        className={[
          'flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs font-medium transition-colors',
          activeSeason
            ? 'border-primary bg-primary/10 text-primary hover:bg-primary/20'
            : 'border-input bg-background text-muted-foreground hover:bg-muted hover:text-foreground',
        ].join(' ')}
      >
        <span>{triggerLabel}</span>
        <span className="text-2xs opacity-60">▾</span>
      </button>

      {open && (
        <div
          role="dialog"
          aria-label={t('common.filters.season_aria')}
          className="absolute left-0 top-full z-40 mt-1 flex w-72 flex-col gap-1 rounded-md border border-border bg-background p-2 shadow-lg"
        >
          {onClear && (
            <button
              type="button"
              onClick={() => {
                onClear()
                onClose()
              }}
              className={[
                'rounded px-2 py-1 text-left text-xs font-medium transition-colors',
                activeSeason
                  ? 'text-muted-foreground hover:bg-muted hover:text-foreground'
                  : 'cursor-default bg-primary/10 text-primary',
              ].join(' ')}
              disabled={!activeSeason}
            >
              {t('common.filters.season_all')}
            </button>
          )}

          <ul className="flex flex-col gap-0.5">
            {available.map((s) => (
              <SeasonRow
                key={s.id}
                season={s}
                count={countsByID.get(s.id)}
                isActive={activeSeason?.id === s.id}
                onSelect={() => {
                  onSelectSeason(s)
                  onClose()
                }}
              />
            ))}
          </ul>

          {unavailable.length > 0 && (
            <details className="mt-1">
              <summary className="cursor-pointer rounded px-2 py-1 text-3xs text-muted-foreground hover:bg-muted">
                {formatMessage(commonManifest, 'common.filters.season_empty_fold', locale, { n: unavailable.length })}
              </summary>
              <ul className="mt-0.5 flex flex-col gap-0.5">
                {unavailable.map((s) => (
                  <SeasonRow
                    key={s.id}
                    season={s}
                    count={0}
                    isActive={activeSeason?.id === s.id}
                    disabled
                  />
                ))}
              </ul>
            </details>
          )}
        </div>
      )}
    </div>
  )
}

interface SeasonRowProps {
  season: SeasonEntry
  count: number | undefined
  isActive: boolean
  disabled?: boolean
  onSelect?: () => void
}

function SeasonRow({ season, count, isActive, disabled, onSelect }: SeasonRowProps) {
  return (
    <li>
      <button
        type="button"
        onClick={disabled ? undefined : onSelect}
        disabled={disabled}
        className={[
          'flex w-full items-center justify-between gap-2 rounded px-2 py-1 text-left text-xs transition-colors',
          disabled
            ? 'cursor-not-allowed text-muted-foreground opacity-60'
            : isActive
              ? 'bg-primary/10 text-primary'
              : 'hover:bg-muted',
        ].join(' ')}
      >
        <span className="flex items-center gap-1.5">
          <span className="rounded bg-muted/40 px-1 py-0.5 text-2xs font-mono tabular-nums">
            {season.shortLabel}
          </span>
          <span className="font-medium">{season.label}</span>
        </span>
        {count !== undefined && (
          <span className="text-2xs tabular-nums opacity-70">({count})</span>
        )}
      </button>
    </li>
  )
}
