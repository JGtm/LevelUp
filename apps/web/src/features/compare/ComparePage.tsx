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
import type { CompareMetricRow, CompareResponse } from '@/lib/api/types'

import { CompareBar } from './CompareBar'
import { CompareMirrorRow } from './CompareMirrorRow'
import { getCompareText, normalizeCompareLocale, type CompareText } from './i18n'
import { useCompare } from './queries'

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

// ─── Mode 2 joueurs ──────────────────────────────────────────────────────────

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

// ─── Mode miroir (3 joueurs) ─────────────────────────────────────────────────

interface CategoryMirrorSectionProps {
  title: string
  keys: readonly string[]
  metricsLeft: CompareMetricRow[]   // comparaison A vs B
  metricsRight: CompareMetricRow[]  // comparaison A vs C
  text: CompareText
}

function CategoryMirrorSection({ title, keys, metricsLeft, metricsRight, text }: CategoryMirrorSectionProps) {
  const byKeyLeft = new Map(metricsLeft.map(r => [r.metric, r]))
  const byKeyRight = new Map(metricsRight.map(r => [r.metric, r]))

  const rows = keys.flatMap(k => {
    const left = byKeyLeft.get(k)
    const right = byKeyRight.get(k)
    if (!left || !right) return []
    return [{ left, right }]
  })

  if (rows.length === 0) return null
  return (
    <div className="space-y-2">
      <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground border-b border-border/40 pb-1.5 mb-3">
        {title}
      </h2>
      <div className="space-y-3">
        {rows.map(({ left, right }) => {
          const label = text.metrics[left.metric] ?? left.label_fr
          const valA = formatMetricValue(left.metric, left.value_a, text)
          const valB = formatMetricValue(left.metric, left.value_b, text)
          const valC = formatMetricValue(right.metric, right.value_b, text)
          const sampleNoteB =
            typeof left.sample_size_b === 'number' && left.sample_size_b > 0 && left.sample_size_b < 10
              ? text.sampleSize(left.sample_size_b)
              : undefined
          const sampleNoteC =
            typeof right.sample_size_b === 'number' && right.sample_size_b > 0 && right.sample_size_b < 10
              ? text.sampleSize(right.sample_size_b)
              : undefined
          return (
            <CompareMirrorRow
              key={left.metric}
              label={label}
              valueA={valA}
              valueB={valB}
              valueC={valC}
              rawA={left.value_a}
              rawB={left.value_b}
              rawC={right.value_b}
              winnerAB={left.winner}
              winnerAC={right.winner}
              sampleNoteB={sampleNoteB}
              sampleNoteC={sampleNoteC}
            />
          )
        })}
      </div>
    </div>
  )
}

// ─── Header joueurs ──────────────────────────────────────────────────────────

function PlayerHeader({ data, text }: { data: CompareResponse; text: CompareText }) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  return (
    <div className="flex items-center justify-between pb-3 border-b border-border/50">
      <span className="text-base font-bold" style={{ color: colorA }}>{data.player_a.gamertag}</span>
      <span className="text-xs text-muted-foreground font-medium px-4">{text.vs}</span>
      <span className="text-base font-bold" style={{ color: colorB }}>{data.player_b.gamertag}</span>
    </div>
  )
}

function MirrorHeader({
  gamertagA, gamertagB, gamertagC,
}: { gamertagA: string; gamertagB: string; gamertagC: string }) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  return (
    <div className="grid pb-3 border-b border-border/50" style={{ gridTemplateColumns: '1fr auto 1fr' }}>
      <span className="text-base font-bold" style={{ color: colorB }}>{gamertagB}</span>
      <span className="text-base font-bold px-6 text-center" style={{ color: colorA }}>{gamertagA}</span>
      <span className="text-base font-bold text-right" style={{ color: colorB }}>{gamertagC}</span>
    </div>
  )
}

// ─── Page principale ─────────────────────────────────────────────────────────

export function ComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const search = useSearch({ from: '/players/$playerSlug/compare' }) as {
    target?: string
    target2?: string
    from?: 'explorer'
  }
  const target = search.target
  const target2 = search.target2
  const fromExplorer = search.from === 'explorer'
  const isMirror = !!target && !!target2

  const locale = normalizeCompareLocale(useAppShellStore((s) => s.locale))
  const { data: fieldMappings } = useFieldMappings()
  const text = getCompareText(locale, fieldMappings)

  const leftCompare = useCompare(playerSlug)   // A vs B
  const rightCompare = useCompare(playerSlug)  // A vs C (mirror uniquement)

  useEffect(() => {
    document.title = `LevelUp - ${text.pageTitle}`
    return () => { document.title = 'LevelUp' }
  }, [text.pageTitle])

  useEffect(() => {
    leftCompare.reset()
    if (target?.trim()) leftCompare.mutate({ target_gamertag: target.trim() })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target])

  useEffect(() => {
    rightCompare.reset()
    if (target2?.trim()) rightCompare.mutate({ target_gamertag: target2.trim() })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target2])

  function handleComboChange(values: string[]) {
    void navigate({
      to: '/players/$playerSlug/compare',
      params: { playerSlug },
      search: {
        target: values[0] ?? undefined,
        target2: values[1] ?? undefined,
        from: search.from,
      },
    })
  }

  const isPending = leftCompare.isPending || (isMirror && rightCompare.isPending)
  const leftData = leftCompare.data
  const rightData = rightCompare.data

  return (
    <div className="p-6 space-y-6 w-full max-w-5xl mx-auto">
      {/* Navigation + titre + combobox */}
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
              selected={[target, target2].filter((v): v is string => !!v)}
              onChange={handleComboChange}
              max={2}
              placeholder={text.searchPlaceholder}
              allowFreeInput
            />
          </div>
        </div>
      </div>

      {/* États de chargement / erreur */}
      {isPending && (
        <div className="flex justify-center py-12">
          <Spinner size="lg" label={text.loading} />
        </div>
      )}
      {!isPending && leftCompare.isError && (
        <EmptyStateCard
          title={leftCompare.error?.message?.includes('404') ? text.notFoundTitle : text.errorTitle}
          description={
            leftCompare.error?.message?.includes('404')
              ? text.notFoundDescription
              : (leftCompare.error?.message ?? text.errorDescription)
          }
        />
      )}
      {!isPending && !leftData && !target && (
        <p className="py-12 text-center text-sm text-muted-foreground">{text.emptyPrompt}</p>
      )}

      {/* ── Mode miroir (2 challengers) ── */}
      {!isPending && isMirror && leftData && rightData && (
        <div className="space-y-6">
          <MirrorHeader
            gamertagA={leftData.player_a.gamertag}
            gamertagB={leftData.player_b.gamertag}
            gamertagC={rightData.player_b.gamertag}
          />

          <div className="space-y-8">
            <CategoryMirrorSection
              title={text.catCombat}
              keys={CATEGORY_KEYS.combat}
              metricsLeft={leftData.metrics}
              metricsRight={rightData.metrics}
              text={text}
            />
            <CategoryMirrorSection
              title={text.catPrecision}
              keys={CATEGORY_KEYS.precision}
              metricsLeft={leftData.metrics}
              metricsRight={rightData.metrics}
              text={text}
            />
            <CategoryMirrorSection
              title={text.catBilan}
              keys={CATEGORY_KEYS.bilan}
              metricsLeft={leftData.metrics}
              metricsRight={rightData.metrics}
              text={text}
            />
          </div>
        </div>
      )}

      {/* ── Mode normal (1 challenger) ── */}
      {!isPending && !isMirror && leftData && (
        <div className="space-y-6">
          <PrivacyBanner warning={leftData.privacy_warning} />
          {leftData.player_b_partial && !leftData.privacy_warning && (
            <p className="rounded bg-warning/10 px-3 py-2 text-xs text-warning">
              {text.partialWarning(leftData.player_b.gamertag)}
            </p>
          )}

          <PlayerHeader data={leftData} text={text} />

          {leftData.metrics.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-10 gap-y-8">
              <CategoryColumn
                title={text.catCombat}
                rows={getCategoryRows(leftData.metrics, CATEGORY_KEYS.combat)}
                text={text}
                gamertagA={leftData.player_a.gamertag}
                gamertagB={leftData.player_b.gamertag}
              />
              <CategoryColumn
                title={text.catPrecision}
                rows={getCategoryRows(leftData.metrics, CATEGORY_KEYS.precision)}
                text={text}
                gamertagA={leftData.player_a.gamertag}
                gamertagB={leftData.player_b.gamertag}
              />
              <CategoryColumn
                title={text.catBilan}
                rows={getCategoryRows(leftData.metrics, CATEGORY_KEYS.bilan)}
                text={text}
                gamertagA={leftData.player_a.gamertag}
                gamertagB={leftData.player_b.gamertag}
              />
            </div>
          )}

          {/* Arme favorite */}
          {(leftData.player_a.favorite_weapon ?? leftData.player_b.favorite_weapon) && (
            <div className="rounded-md border border-border/50 bg-muted/30 px-4 py-3 space-y-2">
              <p className="text-[11px] text-center text-muted-foreground">{text.favoriteWeapon}</p>
              <div className="flex items-start justify-between gap-4 text-sm">
                <div className="flex-1 text-left">
                  {leftData.player_a.favorite_weapon ? (
                    <>
                      <span className="font-medium" style={{ color: tokenCssVar('compare-a' as SemanticToken) }}>
                        {locale === 'fr' ? leftData.player_a.favorite_weapon.label_fr : leftData.player_a.favorite_weapon.label_en}
                      </span>
                      <span className="text-muted-foreground text-xs ml-1">
                        · {text.killsWith(leftData.player_a.favorite_weapon.kills)}
                      </span>
                    </>
                  ) : (
                    <span className="text-muted-foreground">{text.noWeaponData}</span>
                  )}
                </div>
                <div className="flex-1 text-right">
                  {leftData.player_b.favorite_weapon ? (
                    <>
                      <span className="text-muted-foreground text-xs mr-1">
                        {text.killsWith(leftData.player_b.favorite_weapon.kills)} ·
                      </span>
                      <span className="font-medium" style={{ color: tokenCssVar('compare-b' as SemanticToken) }}>
                        {locale === 'fr' ? leftData.player_b.favorite_weapon.label_fr : leftData.player_b.favorite_weapon.label_en}
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
