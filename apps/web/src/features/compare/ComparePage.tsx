import { useEffect } from 'react'
import { useParams, useNavigate, useSearch, Link } from '@tanstack/react-router'

import { EmptyStateCard } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { Spinner } from '@/components/ui/spinner'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CompareMetricRow } from '@/lib/api/types'

import { CompareBar } from './CompareBar'
import { getCompareText, normalizeCompareLocale, type CompareText } from './i18n'
import { useCompare } from './queries'

// Groupes de métriques — ordre d'affichage dans chaque colonne.
const CATEGORY_KEYS = {
  combat: ['win_rate', 'kda', 'kdr', 'kills_per_game', 'deaths_per_game', 'assists_per_game', 'damage_per_game', 'rendement'],
  precision: ['accuracy', 'headshot_kills_per_game', 'perfect_kills_per_game', 'max_killing_spree', 'avg_life_secs', 'resistance', 'damage_taken_per_game'],
  bilan: ['matches', 'csr_current', 'csr_best', 'career_rank', 'perf_ath', 'lusr_ath'],
} as const

function formatMetricValue(metric: string, value: number | string, text: CompareText) {
  if (typeof value !== 'number') return String(value)
  if (metric === 'win_rate' || metric === 'accuracy' || metric === 'rendement' || metric === 'resistance') {
    return `${(value * 100).toLocaleString(text.intlLocale, { maximumFractionDigits: 1 })} %`
  }
  if (metric === 'matches' || metric === 'csr_current' || metric === 'csr_best' || metric === 'career_rank' || metric === 'max_killing_spree' || metric === 'perf_ath' || metric === 'lusr_ath') {
    return value.toLocaleString(text.intlLocale, { maximumFractionDigits: 0 })
  }
  if (metric === 'avg_life_secs') {
    const m = Math.floor(value / 60)
    const s = Math.round(value % 60)
    return m > 0 ? `${m}m ${s}s` : `${s}s`
  }
  return value.toLocaleString(text.intlLocale, { minimumFractionDigits: 1, maximumFractionDigits: 2 })
}

function getCategoryRows(rows: CompareMetricRow[], keys: readonly string[]) {
  const byKey = new Map(rows.map(r => [r.metric, r]))
  return keys.flatMap(k => {
    const row = byKey.get(k)
    return row ? [row] : []
  })
}

interface CategoryColumnProps {
  title: string
  rows: CompareMetricRow[]
  text: CompareText
  gamertagA: string
  gamertagB: string
}

function CategoryColumn({ title, rows, text, gamertagA, gamertagB }: CategoryColumnProps) {
  if (rows.length === 0) return null
  return (
    <div className="space-y-2">
      <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/40 pb-1.5 mb-3">
        {title}
      </h2>
      <div className="space-y-3">
        {rows.map((row) => {
          const label = text.metrics[row.metric] ?? row.label_fr
          const valA = formatMetricValue(row.metric, row.value_a, text)
          const valB = formatMetricValue(row.metric, row.value_b, text)
          const ariaLabel =
            row.winner === 'tie' || row.winner == null
              ? text.ariaEqual
              : row.winner === 'a'
                ? text.ariaWinner(gamertagA)
                : text.ariaWinner(gamertagB)
          const sampleNote =
            typeof row.sample_size_b === 'number' && row.sample_size_b > 0 && row.sample_size_b < 10
              ? text.sampleSize(row.sample_size_b)
              : undefined
          return (
            <CompareBar
              key={row.metric}
              label={label}
              valueA={valA}
              valueB={valB}
              rawA={row.value_a}
              rawB={row.value_b}
              winner={row.winner}
              ariaLabel={ariaLabel}
              sampleNote={sampleNote}
            />
          )
        })}
      </div>
    </div>
  )
}

export function ComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const search = useSearch({ from: '/players/$playerSlug/compare' }) as {
    target?: string
    from?: 'explorer'
  }
  const target = search.target
  const fromExplorer = search.from === 'explorer'

  const locale = normalizeCompareLocale(useAppShellStore((s) => s.locale))
  const { data: fieldMappings } = useFieldMappings()
  const text = getCompareText(locale, fieldMappings)

  const { mutate, data, isPending, isError, error, reset } = useCompare(playerSlug)

  useEffect(() => {
    document.title = `LevelUp - ${text.pageTitle}`
    return () => { document.title = 'LevelUp' }
  }, [text.pageTitle])

  useEffect(() => {
    reset()
    if (target?.trim()) {
      mutate({ target_gamertag: target.trim() })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  function handleComboChange(v: string[]) {
    void navigate({
      to: '/players/$playerSlug/compare',
      params: { playerSlug },
      search: { target: v[0] ?? undefined, from: search.from },
    })
  }

  return (
    <div className="p-6 space-y-6 w-full max-w-5xl mx-auto">
      {/* Navigation + titre + combobox sur la même ligne */}
      <div className="space-y-2">
        {fromExplorer && (
          <Link
            to="/players/$playerSlug/explorer"
            params={{ playerSlug }}
            search={{ mode: 'player', target: target || undefined }}
            className="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          >
            ← {text.backToExplorer}
          </Link>
        )}
        <div className="flex items-center gap-4">
          <h1 className="text-lg font-semibold text-foreground shrink-0">{text.pageTitle}</h1>
          <div className="flex-1 max-w-sm">
            <GamertagCombobox
              selected={target ? [target] : []}
              onChange={handleComboChange}
              max={1}
              placeholder={text.searchPlaceholder}
              allowFreeInput
            />
          </div>
        </div>
      </div>

      {/* États */}
      {isPending && (
        <div className="flex justify-center py-12">
          <Spinner size="lg" label={text.loading} />
        </div>
      )}
      {isError && (
        <EmptyStateCard
          title={error?.message?.includes('404') ? text.notFoundTitle : text.errorTitle}
          description={
            error?.message?.includes('404')
              ? text.notFoundDescription
              : (error?.message ?? text.errorDescription)
          }
        />
      )}
      {!isPending && !isError && !data && !target && (
        <p className="py-12 text-center text-sm text-muted-foreground">{text.emptyPrompt}</p>
      )}

      {/* Résultats */}
      {data && (
        <div className="space-y-6">
          <PrivacyBanner warning={data.privacy_warning} />
          {data.player_b_partial && !data.privacy_warning && (
            <p className="rounded bg-warning/10 px-3 py-2 text-xs text-warning">
              {text.partialWarning(data.player_b.gamertag)}
            </p>
          )}

          {/* Header joueurs */}
          <div className="flex items-center justify-between pb-3 border-b border-border/50">
            <span
              className="text-base font-bold"
              style={{ color: tokenCssVar('compare-a' as SemanticToken) }}
            >
              {data.player_a.gamertag}
            </span>
            <span className="text-xs text-muted-foreground font-medium px-4">{text.vs}</span>
            <span
              className="text-base font-bold"
              style={{ color: tokenCssVar('compare-b' as SemanticToken) }}
            >
              {data.player_b.gamertag}
            </span>
          </div>

          {/* Grille 3 colonnes par catégorie */}
          {data.metrics.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-10 gap-y-8">
              <CategoryColumn
                title={text.catCombat}
                rows={getCategoryRows(data.metrics, CATEGORY_KEYS.combat)}
                text={text}
                gamertagA={data.player_a.gamertag}
                gamertagB={data.player_b.gamertag}
              />
              <CategoryColumn
                title={text.catPrecision}
                rows={getCategoryRows(data.metrics, CATEGORY_KEYS.precision)}
                text={text}
                gamertagA={data.player_a.gamertag}
                gamertagB={data.player_b.gamertag}
              />
              <CategoryColumn
                title={text.catBilan}
                rows={getCategoryRows(data.metrics, CATEGORY_KEYS.bilan)}
                text={text}
                gamertagA={data.player_a.gamertag}
                gamertagB={data.player_b.gamertag}
              />
            </div>
          )}

          {/* Arme favorite */}
          {(data.player_a.favorite_weapon ?? data.player_b.favorite_weapon) && (
            <div className="rounded-md border border-border/50 bg-muted/30 px-4 py-3 space-y-2">
              <p className="text-[11px] text-center text-muted-foreground">{text.favoriteWeapon}</p>
              <div className="flex items-start justify-between gap-4 text-sm">
                <div className="flex-1 text-left">
                  {data.player_a.favorite_weapon ? (
                    <>
                      <span className="font-medium" style={{ color: tokenCssVar('compare-a' as SemanticToken) }}>
                        {locale === 'fr' ? data.player_a.favorite_weapon.label_fr : data.player_a.favorite_weapon.label_en}
                      </span>
                      <span className="text-muted-foreground text-xs ml-1">
                        · {text.killsWith(data.player_a.favorite_weapon.kills)}
                      </span>
                    </>
                  ) : (
                    <span className="text-muted-foreground">{text.noWeaponData}</span>
                  )}
                </div>
                <div className="flex-1 text-right">
                  {data.player_b.favorite_weapon ? (
                    <>
                      <span className="text-muted-foreground text-xs mr-1">
                        {text.killsWith(data.player_b.favorite_weapon.kills)} ·
                      </span>
                      <span className="font-medium" style={{ color: tokenCssVar('compare-b' as SemanticToken) }}>
                        {locale === 'fr' ? data.player_b.favorite_weapon.label_fr : data.player_b.favorite_weapon.label_en}
                      </span>
                    </>
                  ) : (
                    <span className="text-muted-foreground">{text.noWeaponData}</span>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
