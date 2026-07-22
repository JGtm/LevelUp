import { useEffect } from 'react'
import { useParams, useNavigate, useSearch, Link } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'

import { EmptyStateCard } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { Spinner } from '@/components/ui/spinner'
import { Tooltip } from '@/components/ui/tooltip'
import { GamertagCombobox } from '@/components/ui/GamertagCombobox'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { tokenCssVar, tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { formatMessage } from '@/lib/i18n/format'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { localizeTierLabel } from '@/lib/skillTiers'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CompareMetricRow, CompareResponse, MatchEncounterBadge } from '@/lib/api/types'

import { CompareBar } from './CompareBar'
import { CompareMirrorRow } from './CompareMirrorRow'
import { getCompareText, normalizeCompareLocale, type CompareText } from './i18n'
import { useCompare } from './queries'

const CATEGORY_KEYS = {
  combat: ['win_rate', 'kda', 'kdr', 'kills_per_game', 'deaths_per_game', 'assists_per_game', 'damage_per_game', 'rendement'],
  precision: ['accuracy', 'headshot_kills_per_game', 'perfect_kills_per_game', 'max_killing_spree', 'avg_life_secs', 'resistance', 'damage_taken_per_game'],
  bilan: ['matches', 'career_rank', 'csr', 'csr_alltime', 'perf_ath', 'lusr_ath'],
} as const

function formatMetricValue(
  metric: string,
  value: number | string,
  text: CompareText,
  available: boolean = true,
  display?: string | null,
) {
  if (!available) return text.notAvailable
  // Libellé prêt-à-afficher fourni par le back (rang carrière → titre, CSR → tier).
  // Le palier CSR/LUSR est composé en FR au sync (non locale-aware) → on localise le
  // NOM de palier à l'affichage. Un titre de rang carrière (déjà localisé côté back)
  // ou tout libellé sans palier connu est renvoyé inchangé par localizeTierLabel.
  if (display) {
    const loc = text.intlLocale.toLowerCase().startsWith('en') ? 'en' : 'fr'
    return localizeTierLabel(display, loc) ?? display
  }
  if (typeof value !== 'number') return String(value)
  if (metric === 'win_rate' || metric === 'accuracy') {
    return `${(value * 100).toLocaleString(text.intlLocale, { maximumFractionDigits: 1 })} %`
  }
  // Rendement (OC) / Résistance (DR) : même convention que CombatYieldDisplay.
  // OC = valeur×100 ; DR = (valeur−1)×100 (baseline 1.0). Entier.
  if (metric === 'rendement') {
    return `${(value * 100).toLocaleString(text.intlLocale, { maximumFractionDigits: 0 })} %`
  }
  if (metric === 'resistance') {
    return `${((value - 1) * 100).toLocaleString(text.intlLocale, { maximumFractionDigits: 0 })} %`
  }
  if (metric === 'matches' || metric === 'career_rank' || metric === 'csr' || metric === 'max_killing_spree' || metric === 'perf_ath' || metric === 'lusr_ath') {
    return value.toLocaleString(text.intlLocale, { maximumFractionDigits: 0 })
  }
  if (metric === 'avg_life_secs') {
    const m = Math.floor(value / 60)
    const s = Math.round(value % 60)
    return m > 0 ? `${m}m ${s}s` : `${s}s`
  }
  return value.toLocaleString(text.intlLocale, { minimumFractionDigits: 1, maximumFractionDigits: 2 })
}

// Le contrat type winner en `string` (struct Go) ; les barres de comparaison
// n'admettent que le domaine fermé 'a' | 'b' | 'tie' | null. Toute valeur hors
// domaine retombe sur null (traité comme 'tie' par les composants).
function asWinner(winner: string): 'a' | 'b' | 'tie' | null {
  return winner === 'a' || winner === 'b' || winner === 'tie' ? winner : null
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
    <div className="rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</h2>
      </div>
      <div className="p-3 space-y-3">
        {rows.map((row) => {
          const label = text.metrics[row.metric] ?? row.label_fr
          const availableA = row.value_a_available !== false
          const availableB = row.value_b_available !== false
          const bothAvailable = availableA && availableB
          const valA = formatMetricValue(row.metric, row.value_a, text, availableA, row.display_a)
          const valB = formatMetricValue(row.metric, row.value_b, text, availableB, row.display_b)
          const ariaLabel = !bothAvailable
            ? text.ariaNotAvailable
            : row.winner === 'tie' || row.winner == null
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
              winner={asWinner(row.winner)}
              ariaLabel={ariaLabel}
              sampleNote={sampleNote}
              availableA={availableA}
              availableB={availableB}
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
    <div className="rounded-lg border border-border bg-card">
      <div className="border-b border-border px-3 py-2">
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</h2>
      </div>
      <div className="p-3 space-y-3">
        {rows.map(({ left, right }) => {
          const label = text.metrics[left.metric] ?? left.label_fr
          // Player A est partagé entre left/right ; on prend le OR pour considérer
          // la donnée comme disponible dès que l'une des deux comparaisons l'expose.
          const availableA = left.value_a_available !== false || right.value_a_available !== false
          const availableB = left.value_b_available !== false
          const availableC = right.value_b_available !== false
          const valA = formatMetricValue(left.metric, left.value_a, text, availableA, left.display_a)
          const valB = formatMetricValue(left.metric, left.value_b, text, availableB, left.display_b)
          const valC = formatMetricValue(right.metric, right.value_b, text, availableC, right.display_b)
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
              winnerAB={asWinner(left.winner)}
              winnerAC={asWinner(right.winner)}
              sampleNoteB={sampleNoteB}
              sampleNoteC={sampleNoteC}
              availableA={availableA}
              availableB={availableB}
              availableC={availableC}
              lessIsBetter={left.less_is_better ?? right.less_is_better ?? false}
            />
          )
        })}
      </div>
    </div>
  )
}

// ─── Badges d'encounter ──────────────────────────────────────────────────────

function isSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-') || s.startsWith('outcome-') || s.startsWith('perf-')
}

const ENCOUNTER_BADGE_TOOLTIPS: Record<string, { fr: string; en: string }> = {
  'narrative.encounter.ally_plus': { fr: 'Allié récurrent : bon taux de victoire ensemble', en: 'Recurring ally: good win rate together' },
  'narrative.encounter.tough_enemy': { fr: 'Dur à cuire : ratio morts/frags > 2 en ennemi', en: 'Tough nut: death/kill ratio > 2 as enemy' },
  'narrative.encounter.coriace': { fr: 'Coriace : taux de victoire en ennemi ≤ 35 %', en: 'Tough opponent: win rate vs enemy ≤ 35 %' },
  'narrative.encounter.ordinal': { fr: 'Total rencontres croisées (allié + ennemi)', en: 'Total cross encounters (ally + enemy)' },
}

function EncounterBadgesInline({ badges, locale }: { badges?: MatchEncounterBadge[]; locale: 'fr' | 'en' }) {
  if (!badges?.length) return null
  return (
    <span className="ml-2 inline-flex flex-wrap gap-1 align-middle">
      {badges.map((badge, i) => {
        const labelKey = badge.label_key as SquadManifestKey
        const ordinal = badge.detail && typeof badge.detail['ordinal'] === 'number' ? (badge.detail['ordinal'] as number) : undefined
        const label = ordinal !== undefined ? formatMessage(squadManifest, labelKey, locale, { ordinal }) : formatMessage(squadManifest, labelKey, locale)
        const colorVar = isSemanticToken(badge.color_token) ? tokenVar(badge.color_token as SemanticToken) : undefined
        const tooltip = ENCOUNTER_BADGE_TOOLTIPS[badge.label_key]?.[locale]
        const badgeEl = <NarrativeBadge label={label} colorVar={colorVar} solid size="md" />
        return tooltip ? (
          <Tooltip key={i} content={tooltip}>{badgeEl}</Tooltip>
        ) : (
          <span key={i}>{badgeEl}</span>
        )
      })}
    </span>
  )
}

// ─── Header joueurs ──────────────────────────────────────────────────────────

function PlayerHeader({ data, text, locale }: { data: CompareResponse; text: CompareText; locale: 'fr' | 'en' }) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  return (
    <div className="flex items-center pb-3 border-b border-border/50">
      <span className="flex-1 text-right text-base font-bold" style={{ color: colorA }}>{data.player_a.gamertag}</span>
      <span className="shrink-0 text-xs text-muted-foreground font-medium px-4">{text.vs}</span>
      <span className="flex-1 inline-flex items-center text-base font-bold" style={{ color: colorB }}>
        {data.player_b.gamertag}
        <EncounterBadgesInline badges={data.encounter_badges} locale={locale} />
      </span>
    </div>
  )
}

function MirrorHeader({
  gamertagA, gamertagB, gamertagC, badgesB, badgesC, locale,
}: {
  gamertagA: string
  gamertagB: string
  gamertagC: string
  badgesB?: MatchEncounterBadge[]
  badgesC?: MatchEncounterBadge[]
  locale: 'fr' | 'en'
}) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  const colorC = tokenCssVar('compare-c' as SemanticToken)
  return (
    <div className="grid pb-3 border-b border-border/50" style={{ gridTemplateColumns: '1fr auto 1fr' }}>
      <span className="inline-flex items-center text-base font-bold" style={{ color: colorB }}>
        {gamertagB}
        <EncounterBadgesInline badges={badgesB} locale={locale} />
      </span>
      <span className="text-base font-bold px-6 text-center" style={{ color: colorA }}>{gamertagA}</span>
      <span className="inline-flex items-center justify-end text-base font-bold" style={{ color: colorC }}>
        {gamertagC}
        <EncounterBadgesInline badges={badgesC} locale={locale} />
      </span>
    </div>
  )
}

// ─── Page principale ─────────────────────────────────────────────────────────

export function ComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()
  const search = useSearch({ from: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/compare' }) as {
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
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/community/compare',
      params: { titleSlug, playerSlug },
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
    <div className="p-6 space-y-6 w-full">
      {/* Navigation + titre + combobox */}
      <div className="space-y-2">
        {fromExplorer && (
          <Link
            to="/{-$lang}/t/$titleSlug/players/$playerSlug/explorer"
            params={{ titleSlug, playerSlug }}
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
              colors={[
                tokenCssVar('compare-b' as SemanticToken),
                tokenCssVar('compare-c' as SemanticToken),
              ]}
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
            badgesB={leftData.encounter_badges}
            badgesC={rightData.encounter_badges}
            locale={locale}
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

          <PlayerHeader data={leftData} text={text} locale={locale} />

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
        </div>
      )}
    </div>
  )
}
