/**
 * PalmaresRelationsPage — hub Communauté > Relations (Phase 2).
 *
 * Consomme l'endpoint backend réel POST /pages/palmares/relations (forme
 * {overview, relations[]}). Barre de segmentation serveur (useLocalFilterBar :
 * expérience / saison / période / playlist / mode / vue solo-escouade), hero
 * enrichi (binôme / bête noire / noyau dur), segmented control + toggle « amis »,
 * tableau paginé (langage MatchEncountersTable) et section « Noyau dur » détaillée.
 */
import { useMemo, type ReactNode } from 'react'
import { useNavigate, useParams } from '@tanstack/react-router'

import { KpiCard } from '@/components/cards/KpiCard'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useLocalFilterBar } from '@/features/_shared/useLocalFilterBar'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { formatPercent } from '@/lib/formatters'
import type { FilterContextInput, RelationDuelEntry, RelationInsight } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useRelationsPrefsStore } from '@/stores/relationsPrefsStore'

import { getPalmaresText, normalizePalmaresLocale, type PalmaresLocale, type PalmaresText } from './i18n'
import { useRelationsMoments, useRelationsPage } from './queries'
import { RelationBadges } from './RelationBadges'
import { RelationSplitBar } from './RelationSplitBar'
import { RelationsMomentsSection } from './RelationsMomentsSection'
import { RelationsTable } from './RelationsTable'
import { coreRelations, filterRelations, type RelationFilter } from './relationsFilter'

type RelationsText = PalmaresText['relations']

const FILTER_CHIPS: RelationFilter[] = ['all', 'core', 'allies', 'rivals', 'recent']

/** findRelation — retrouve la ligne complète (RelationInsight) d'une référence hero. */
function findRelation(relations: RelationInsight[], gamertag: string | undefined | null): RelationInsight | null {
  if (!gamertag) return null
  return relations.find((r) => r.gamertag === gamertag) ?? null
}

function winLossColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  return v >= 0.5 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss')
}

function ratioColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  if (v > 1) return tokenCssVar('outcome-win')
  if (v < 1) return tokenCssVar('outcome-loss')
  return tokenCssVar('outcome-draw')
}

function formatRatio(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return v.toFixed(2)
}

// kdaColor : le KDA est un NET signé (peut être négatif) — vert si positif,
// rouge si négatif, neutre à 0 (cohérent avec kdaDivergentScale).
function kdaColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  if (v > 0) return tokenCssVar('outcome-win')
  if (v < 0) return tokenCssVar('outcome-loss')
  return tokenCssVar('outcome-draw')
}

// duelOutcomeToken : couleur d'un carré de la mini-frise (win/loss/neutre).
function duelOutcomeToken(outcome: string): SemanticToken {
  if (outcome === 'win') return 'outcome-win'
  if (outcome === 'loss') return 'outcome-loss'
  return 'outcome-draw'
}

// DuelMiniTape : frise compacte des derniers duels (ancien→récent), un carré
// coloré par duel. Décorative (le détail au survol vit dans la section Revanche).
function DuelMiniTape({ duels }: { duels: RelationDuelEntry[] }) {
  const recent = duels.slice(-14)
  return (
    <span className="inline-flex shrink-0 gap-0.5" aria-hidden="true">
      {recent.map((d, i) => (
        <span
          key={`${d.match_id}-${i}`}
          className="h-3.5 w-1.5 rounded-sm"
          style={{ backgroundColor: tokenCssVar(duelOutcomeToken(d.outcome)) }}
        />
      ))}
    </span>
  )
}

/**
 * HeroRelationCard — carte hero enrichie : binôme (mode ally) ou bête noire
 * (mode enemy). Affiche gamertag cliquable + badges + WR + ratio + frags/morts +
 * volume, depuis la ligne RelationInsight complète (pas seulement la référence).
 */
function HeroRelationCard({
  title,
  emptyLabel,
  accent,
  relation,
  mode,
  labels,
  locale,
  onPlayerClick,
  duels,
}: {
  title: string
  emptyLabel: string
  accent: Parameters<typeof KpiCard>[0]['accent']
  relation: RelationInsight | null
  mode: 'ally' | 'enemy'
  labels: RelationsText
  locale: 'fr' | 'en'
  onPlayerClick: (gamertag: string) => void
  duels?: RelationDuelEntry[]
}) {
  if (!relation) {
    return (
      <KpiCard accent={accent} className="flex h-full flex-col">
        <div className="flex flex-1 flex-col p-4">
          <p className="text-xs uppercase tracking-label-md text-muted-foreground">{title}</p>
          <p className="mt-2 text-sm text-muted-foreground">{emptyLabel}</p>
        </div>
      </KpiCard>
    )
  }
  const wr = mode === 'ally' ? relation.teammate_win_rate : relation.enemy_win_rate
  const matches = mode === 'ally' ? relation.teammate_matches : relation.enemy_matches
  const wrLabel = mode === 'ally' ? labels.table.winRateAlly : labels.table.winRateEnemy
  const avgKda = mode === 'ally' ? relation.avg_kda_with : relation.avg_kda_against
  return (
    <KpiCard accent={accent} className="flex h-full flex-col">
      <div className="flex flex-1 flex-col p-4">
        <p className="text-xs uppercase tracking-label-md text-muted-foreground">{title}</p>
        <span className="mt-1 whitespace-nowrap">
          <button
            type="button"
            className="text-left text-2xl font-semibold text-info hover:underline"
            onClick={() => onPlayerClick(relation.gamertag)}
          >
            {relation.gamertag}
          </button>
          <RelationBadges badges={relation.badges} locale={locale} />
        </span>
        <div className="mt-3 flex items-baseline gap-5">
          <div>
            <span className="font-mono text-2xl font-bold" style={{ color: winLossColor(wr) }}>
              {formatPercent(wr, 0)}
            </span>
            <span className="ml-1 text-xs text-muted-foreground">{wrLabel}</span>
          </div>
          <div>
            {mode === 'ally' && avgKda != null && Number.isFinite(avgKda) ? (
              <>
                <span className="font-mono text-lg font-semibold" style={{ color: kdaColor(avgKda) }}>
                  {formatRatio(avgKda)}
                </span>
                <span className="ml-1 text-xs text-muted-foreground">{labels.table.kdaTogether}</span>
              </>
            ) : (
              <>
                <span className="font-mono text-lg font-semibold" style={{ color: ratioColor(relation.duel_ratio) }}>
                  {formatRatio(relation.duel_ratio)}
                </span>
                <span className="ml-1 text-xs text-muted-foreground">{labels.table.ratio}</span>
              </>
            )}
          </div>
        </div>
        <div className="mt-3">
          {mode === 'ally' ? (
            <RelationSplitBar
              label={labels.table.encounters}
              leftValue={relation.teammate_matches}
              rightValue={relation.enemy_matches}
              leftToken="team-ally"
              rightToken="team-enemy"
              locale={locale}
            />
          ) : (
            <RelationSplitBar
              label={labels.table.fragsDeaths}
              leftValue={relation.kills_dealt}
              rightValue={relation.deaths_suffered}
              leftToken="outcome-win"
              rightToken="outcome-loss"
              locale={locale}
            />
          )}
        </div>
        {mode === 'enemy' && duels && duels.length > 0 ? (
          <div className="mt-3 flex items-center gap-2">
            <DuelMiniTape duels={duels} />
            <span className="text-xs text-muted-foreground">
              {labels.hero.matchesPlayed(matches.toLocaleString(locale))}
            </span>
          </div>
        ) : (
          <p className="mt-2 text-xs text-muted-foreground">
            {labels.hero.matchesPlayed(matches.toLocaleString(locale))}
          </p>
        )}
      </div>
    </KpiCard>
  )
}

/**
 * CoreSummaryCard — résumé qualitatif du noyau dur (#6) : compte + WR moyen avec
 * eux + volume + 2 noms cliquables. La LISTE détaillée vit dans la section dédiée
 * ci-dessous (plus de doublon « compte seul » vs « liste »).
 */
function CoreSummaryCard({
  title,
  unit,
  coreRows,
  labels,
  locale,
  onPlayerClick,
}: {
  title: string
  unit: string
  coreRows: RelationInsight[]
  labels: RelationsText
  locale: 'fr' | 'en'
  onPlayerClick: (gamertag: string) => void
}) {
  const count = coreRows.length
  const wrs = coreRows
    .map((r) => r.teammate_win_rate)
    .filter((v): v is number => v != null && Number.isFinite(v))
  const avgWr = wrs.length > 0 ? wrs.reduce((a, b) => a + b, 0) / wrs.length : null
  const totalGames = coreRows.reduce((a, r) => a + r.total_matches, 0)
  const topNames = coreRows.slice(0, 2)
  return (
    <KpiCard accent="info" className="flex h-full flex-col">
      <div className="flex flex-1 flex-col p-4">
        <p className="text-xs uppercase tracking-label-md text-muted-foreground">{title}</p>
        <p className="mt-2 text-2xl font-semibold text-foreground">
          {count.toLocaleString(locale)} <span className="text-base font-normal text-muted-foreground">{unit}</span>
        </p>
        <div className="mt-2 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-xs">
          {avgWr != null && (
            <span>
              <span className="font-mono font-bold" style={{ color: winLossColor(avgWr) }}>
                {formatPercent(avgWr, 0)}
              </span>{' '}
              <span className="text-muted-foreground">{labels.table.winRateAlly}</span>
            </span>
          )}
          <span className="text-muted-foreground">{labels.hero.matchesPlayed(totalGames.toLocaleString(locale))}</span>
        </div>
        {topNames.length > 0 && (
          <div className="mt-2 flex flex-wrap items-center gap-2 text-sm">
            {topNames.map((r) => (
              <button
                key={r.xuid}
                type="button"
                className="truncate font-semibold text-info hover:underline"
                onClick={() => onPlayerClick(r.gamertag)}
              >
                {r.gamertag}
              </button>
            ))}
            {count > topNames.length && (
              <span className="text-xs text-muted-foreground">+{count - topNames.length}</span>
            )}
          </div>
        )}
      </div>
    </KpiCard>
  )
}

/** SegmentedFilter — segmented control (charte) : une piste, segment actif plein. */
function SegmentedFilter({
  active,
  onChange,
  labels,
}: {
  active: RelationFilter
  onChange: (f: RelationFilter) => void
  labels: RelationsText['chips']
}) {
  const text: Record<RelationFilter, string> = {
    all: labels.all,
    core: labels.core,
    allies: labels.allies,
    rivals: labels.rivals,
    recent: labels.recent,
  }
  return (
    <div
      className="inline-flex flex-wrap rounded-lg border border-border bg-card p-0.5"
      data-testid="palmares-relations-chips"
    >
      {FILTER_CHIPS.map((chip) => {
        const isActive = chip === active
        return (
          <button
            key={chip}
            type="button"
            aria-pressed={isActive}
            onClick={() => onChange(chip)}
            className={`rounded-md px-3 py-1 text-sm font-medium transition-colors ${
              isActive ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {text[chip]}
          </button>
        )
      })}
    </div>
  )
}

/** CoreCards — section détaillée du noyau dur (mini-cards gris foncé). */
function CoreCards({
  rows,
  labels,
  locale,
  onPlayerClick,
}: {
  rows: RelationInsight[]
  labels: RelationsText
  locale: 'fr' | 'en'
  onPlayerClick: (gamertag: string) => void
}) {
  if (rows.length === 0) {
    return <p className="text-sm text-muted-foreground">{labels.core.empty}</p>
  }
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {rows.map((r) => (
        <div key={r.xuid} className="rounded-lg bg-card p-4">
          <span className="whitespace-nowrap">
            <button
              type="button"
              className="truncate text-sm font-semibold text-info hover:underline"
              onClick={() => onPlayerClick(r.gamertag)}
            >
              {r.gamertag}
            </button>
            <RelationBadges badges={r.badges} locale={locale} />
          </span>
          <p className="mt-1 text-xs text-muted-foreground">
            {labels.core.together(r.total_matches.toLocaleString(locale))}
          </p>
          <div className="mt-2 flex flex-col gap-1.5">
            <RelationSplitBar
              label={labels.table.encounters}
              leftValue={r.teammate_matches}
              rightValue={r.enemy_matches}
              leftToken="team-ally"
              rightToken="team-enemy"
              locale={locale}
            />
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 font-mono text-xs">
              {r.teammate_win_rate != null && Number.isFinite(r.teammate_win_rate) && (
                <span>
                  <span className="font-bold" style={{ color: winLossColor(r.teammate_win_rate) }}>
                    {formatPercent(r.teammate_win_rate, 0)}
                  </span>{' '}
                  <span className="text-muted-foreground">{labels.table.winRateAlly}</span>
                </span>
              )}
              {r.avg_kda_with != null && Number.isFinite(r.avg_kda_with) && (
                <span>
                  <span className="font-bold" style={{ color: kdaColor(r.avg_kda_with) }}>
                    {formatRatio(r.avg_kda_with)}
                  </span>{' '}
                  <span className="text-muted-foreground">{labels.table.kdaTogether}</span>
                </span>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

export function PalmaresRelationsPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = normalizePalmaresLocale(useAppShellStore((state) => state.locale))
  const text = getPalmaresText(locale)
  const rel = text.relations
  const navigate = useNavigate()
  const filter = useRelationsPrefsStore((s) => s.filter)
  const setFilter = useRelationsPrefsStore((s) => s.setFilter)
  const includeFriends = useRelationsPrefsStore((s) => s.includeFriends)
  const setIncludeFriends = useRelationsPrefsStore((s) => s.setIncludeFriends)

  const { committedFilterContext, committedHash, bar } = useLocalFilterBar({
    playerSlug,
    labels: {
      experience: rel.filters.experience,
      experienceAll: rel.filters.experienceAll,
      experienceRanked: rel.filters.experienceRanked,
      experienceUnranked: rel.filters.experienceUnranked,
      playlists: rel.filters.playlists,
      modes: rel.filters.modes,
      reset: rel.filters.reset,
      analyser: rel.filters.analyser,
    },
    viewLabels: {
      view: rel.filters.view,
      viewAll: rel.filters.viewAll,
      viewSolo: rel.filters.viewSolo,
      viewSquad: rel.filters.viewSquad,
    },
  })

  const { data, isLoading, isError, error, refetch } = useRelationsPage(
    playerSlug,
    committedFilterContext,
    committedHash,
  )

  function goToExplorer(gamertag: string) {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  // Filtre segment (client) + toggle « amis » : sans les amis, on masque les
  // relations purement coéquipières (jamais affrontées).
  const visibleRows = useMemo(() => {
    const base = data ? filterRelations(data.relations ?? [], filter) : []
    return includeFriends ? base : base.filter((r) => r.enemy_matches > 0)
  }, [data, filter, includeFriends])
  const coreRows = useMemo(() => (data ? coreRelations(data.relations ?? []) : []), [data])

  let body: ReactNode
  if (isLoading) {
    body = (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  } else if (isError || !data) {
    body = (
      <EmptyStateCard
        title={rel.unavailableTitle}
        description={error?.message ?? rel.unavailableDescription}
        actionLabel={rel.retry}
        onAction={() => refetch()}
      />
    )
  } else if ((data.relations?.length ?? 0) === 0) {
    body = <EmptyStateCard title={rel.emptyTitle} description={rel.emptyDescription} />
  } else {
    body = (
      <RelationsContent
        data={data}
        rel={rel}
        locale={locale}
        filter={filter}
        setFilter={setFilter}
        includeFriends={includeFriends}
        setIncludeFriends={setIncludeFriends}
        visibleRows={visibleRows}
        coreRows={coreRows}
        onPlayerClick={goToExplorer}
        playerSlug={playerSlug}
        filterContext={committedFilterContext}
        filterHash={committedHash}
      />
    )
  }

  return (
    <div className="flex flex-col">
      {bar}
      <div className="flex flex-col gap-6 p-6">{body}</div>
    </div>
  )
}

function RelationsContent({
  data,
  rel,
  locale,
  filter,
  setFilter,
  includeFriends,
  setIncludeFriends,
  visibleRows,
  coreRows,
  onPlayerClick,
  playerSlug,
  filterContext,
  filterHash,
}: {
  data: NonNullable<ReturnType<typeof useRelationsPage>['data']>
  rel: RelationsText
  locale: PalmaresLocale
  filter: RelationFilter
  setFilter: (f: RelationFilter) => void
  includeFriends: boolean
  setIncludeFriends: (v: boolean) => void
  visibleRows: RelationInsight[]
  coreRows: RelationInsight[]
  onPlayerClick: (gamertag: string) => void
  playerSlug: string
  filterContext: FilterContextInput
  filterHash: string
}) {
  const ov = data.overview
  const relations = data.relations ?? []
  const allyRelation = findRelation(relations, ov.top_ally?.gamertag)
  const nemesisRelation = findRelation(relations, ov.top_nemesis?.gamertag)
  // Frise du hero « Bête noire » : réutilise la donnée Moments (même queryKey →
  // dédupliquée par TanStack Query, pas d'appel réseau supplémentaire).
  const { data: momentsData } = useRelationsMoments(playerSlug, filterContext, filterHash, true)
  const nemesisDuels = useMemo<RelationDuelEntry[] | undefined>(() => {
    if (!momentsData || !nemesisRelation) return undefined
    const riv = (momentsData.rivalries ?? []).find((r) => r.xuid === nemesisRelation.xuid)
    return riv?.duels ?? undefined
  }, [momentsData, nemesisRelation])
  return (
    <>
      <div className="grid gap-4 lg:grid-cols-3" data-testid="palmares-relations-overview">
        <HeroRelationCard
          title={rel.hero.topAllyTitle}
          emptyLabel={rel.hero.topAllyEmpty}
          accent="outcome-win"
          relation={allyRelation}
          mode="ally"
          labels={rel}
          locale={locale}
          onPlayerClick={onPlayerClick}
        />
        <HeroRelationCard
          title={rel.hero.topNemesisTitle}
          emptyLabel={rel.hero.topNemesisEmpty}
          accent="outcome-loss"
          relation={nemesisRelation}
          mode="enemy"
          labels={rel}
          locale={locale}
          onPlayerClick={onPlayerClick}
          duels={nemesisDuels}
        />
        <CoreSummaryCard
          title={rel.hero.coreTitle}
          unit={rel.hero.coreUnit}
          coreRows={coreRows}
          labels={rel}
          locale={locale}
          onPlayerClick={onPlayerClick}
        />
      </div>

      <div className="flex flex-wrap items-center gap-3">
        <SegmentedFilter active={filter} onChange={setFilter} labels={rel.chips} />
        <button
          type="button"
          aria-pressed={includeFriends}
          onClick={() => setIncludeFriends(!includeFriends)}
          className={`rounded-lg border px-3 py-1 text-sm font-medium transition-colors ${
            includeFriends
              ? 'border-info text-foreground'
              : 'border-border text-muted-foreground hover:text-foreground'
          }`}
        >
          {includeFriends ? rel.filters.friendsIncluded : rel.filters.includeFriends}
        </button>
      </div>

      <RelationsTable
        rows={visibleRows}
        labels={rel}
        locale={locale}
        onPlayerClick={onPlayerClick}
        emptyMessage={rel.filterEmptyDescription}
      />

      <section className="flex flex-col gap-3">
        <div>
          <h2 className="text-base font-semibold text-foreground">{rel.core.sectionTitle}</h2>
          <p className="text-sm text-muted-foreground">{rel.core.sectionDescription}</p>
        </div>
        <CoreCards rows={coreRows} labels={rel} locale={locale} onPlayerClick={onPlayerClick} />
      </section>

      <RelationsMomentsSection
        playerSlug={playerSlug}
        filterContext={filterContext}
        filterHash={filterHash}
        text={rel.moments}
      />
    </>
  )
}
