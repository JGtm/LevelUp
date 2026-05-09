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

import { CompareBar } from './CompareBar'
import { getCompareText, normalizeCompareLocale, type CompareText } from './i18n'
import { useCompare } from './queries'

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
    <div className="p-6 space-y-6 max-w-2xl mx-auto">
      {/* Navigation + titre */}
      <div className="space-y-3">
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
        <h1 className="text-lg font-semibold text-foreground">{text.pageTitle}</h1>
        <GamertagCombobox
          selected={target ? [target] : []}
          onChange={handleComboChange}
          max={1}
          placeholder={text.searchPlaceholder}
          allowFreeInput
        />
      </div>

      {/* États */}
      {isPending && (
        <div className="flex justify-center py-8">
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
        <p className="py-8 text-center text-sm text-muted-foreground">{text.emptyPrompt}</p>
      )}

      {/* Résultats */}
      {data && (
        <div className="space-y-4">
          <PrivacyBanner warning={data.privacy_warning} />
          {data.player_b_partial && !data.privacy_warning && (
            <p className="rounded bg-warning/10 px-3 py-2 text-xs text-warning">
              {text.partialWarning(data.player_b.gamertag)}
            </p>
          )}

          {/* Header joueurs */}
          <div className="flex items-center justify-between px-1">
            <span
              className="text-sm font-semibold"
              style={{ color: tokenCssVar('compare-a' as SemanticToken) }}
            >
              {data.player_a.gamertag}
            </span>
            <span className="text-xs text-muted-foreground">{text.vs}</span>
            <span
              className="text-sm font-semibold"
              style={{ color: tokenCssVar('compare-b' as SemanticToken) }}
            >
              {data.player_b.gamertag}
            </span>
          </div>

          {/* Barres comparatives */}
          {data.metrics.length > 0 && (
            <div className="space-y-3">
              {data.metrics.map((row) => {
                const label = text.metrics[row.metric] ?? row.label_fr
                const valA = formatMetricValue(row.metric, row.value_a, text)
                const valB = formatMetricValue(row.metric, row.value_b, text)
                const ariaLabel =
                  row.winner === 'tie' || row.winner == null
                    ? text.ariaEqual
                    : row.winner === 'a'
                      ? text.ariaWinner(data.player_a.gamertag)
                      : text.ariaWinner(data.player_b.gamertag)
                const sampleNote =
                  typeof row.sample_size_b === 'number' &&
                  row.sample_size_b > 0 &&
                  row.sample_size_b < 10
                    ? text.sampleSize(row.sample_size_b)
                    : undefined
                return (
                  <CompareBar
                    key={row.metric}
                    label={label}
                    valueA={valA}
                    valueB={valB}
                    winner={row.winner}
                    ariaLabel={ariaLabel}
                    sampleNote={sampleNote}
                  />
                )
              })}
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
                        {locale === 'fr'
                          ? data.player_a.favorite_weapon.label_fr
                          : data.player_a.favorite_weapon.label_en}
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
                        {locale === 'fr'
                          ? data.player_b.favorite_weapon.label_fr
                          : data.player_b.favorite_weapon.label_en}
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
