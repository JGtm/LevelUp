/**
 * PalmaresRelationsPage — hub Communauté > Relations (Phase 1).
 *
 * Consomme l'endpoint backend réel /pages/palmares/relations (forme {overview,
 * relations[]}). Hero (binôme / bête noire / noyau dur), chips de filtre CLIENT,
 * tableau unique (langage MatchEncountersTable) et section « Noyau dur » en
 * mini-cards flottantes. Phase 2 ajoutera la barre de segmentation serveur.
 */
import { useMemo, useState } from 'react'
import { useNavigate, useParams } from '@tanstack/react-router'

import { KpiCard } from '@/components/cards/KpiCard'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { tokenCssVar } from '@/lib/accessibility'
import { formatPercent } from '@/lib/formatters'
import type { RelationRef } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { getPalmaresText, normalizePalmaresLocale, type PalmaresText } from './i18n'
import { useRelationsPage } from './queries'
import { RelationsTable } from './RelationsTable'
import { coreRelations, filterRelations, type RelationFilter } from './relationsFilter'

type RelationsText = PalmaresText['relations']

const FILTER_CHIPS: RelationFilter[] = ['all', 'core', 'allies', 'rivals', 'recent']

function HeroRefCard({
  title,
  emptyLabel,
  accent,
  relation,
  matchesPlayed,
}: {
  title: string
  emptyLabel: string
  accent: Parameters<typeof KpiCard>[0]['accent']
  relation: RelationRef | null
  matchesPlayed: (count: string) => string
}) {
  return (
    <KpiCard accent={accent} className="flex h-full flex-col">
      <div className="flex flex-1 flex-col p-4">
        <p className="text-xs uppercase tracking-label-md text-muted-foreground">{title}</p>
        {relation ? (
          <>
            <p className="mt-2 truncate text-2xl font-semibold text-foreground">{relation.gamertag}</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {formatPercent(relation.win_rate, 0)} · {matchesPlayed(relation.matches.toLocaleString())}
            </p>
          </>
        ) : (
          <p className="mt-2 text-sm text-muted-foreground">{emptyLabel}</p>
        )}
      </div>
    </KpiCard>
  )
}

function CoreHeroCard({
  title,
  unit,
  count,
  locale,
}: {
  title: string
  unit: string
  count: number
  locale: string
}) {
  return (
    <KpiCard accent="info" className="flex h-full flex-col">
      <div className="flex flex-1 flex-col p-4">
        <p className="text-xs uppercase tracking-label-md text-muted-foreground">{title}</p>
        <p className="mt-2 text-2xl font-semibold text-foreground">
          {count.toLocaleString(locale)} <span className="text-base font-normal text-muted-foreground">{unit}</span>
        </p>
      </div>
    </KpiCard>
  )
}

function FilterChips({
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
    <div className="flex flex-wrap gap-2" data-testid="palmares-relations-chips">
      {FILTER_CHIPS.map((chip) => {
        const isActive = chip === active
        return (
          <button
            key={chip}
            type="button"
            onClick={() => onChange(chip)}
            className={`rounded-full border px-3 py-1 text-sm transition-colors ${
              isActive
                ? 'border-primary bg-primary text-primary-foreground'
                : 'border-border text-muted-foreground hover:text-foreground'
            }`}
          >
            {text[chip]}
          </button>
        )
      })}
    </div>
  )
}

function CoreCards({
  rows,
  labels,
  locale,
  onPlayerClick,
}: {
  rows: ReturnType<typeof coreRelations>
  labels: RelationsText
  locale: string
  onPlayerClick: (gamertag: string) => void
}) {
  if (rows.length === 0) {
    return <p className="text-sm text-muted-foreground">{labels.core.empty}</p>
  }
  return (
    <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {rows.map((r) => (
        <div key={r.xuid} className="rounded-lg bg-card p-4">
          <button
            type="button"
            className="truncate text-sm font-semibold text-info hover:underline"
            onClick={() => onPlayerClick(r.gamertag)}
          >
            {r.gamertag}
          </button>
          <p className="mt-1 text-xs text-muted-foreground">
            {labels.core.together(r.total_matches.toLocaleString(locale))}
          </p>
          <div className="mt-2 flex items-center gap-3 font-mono text-xs">
            <span style={{ color: tokenCssVar('team-ally') }}>{r.teammate_matches}</span>
            <span className="text-muted-foreground">·</span>
            <span style={{ color: tokenCssVar('team-enemy') }}>{r.enemy_matches}</span>
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
  const [filter, setFilter] = useState<RelationFilter>('all')
  const { data, isLoading, isError, error, refetch } = useRelationsPage(playerSlug)

  function goToExplorer(gamertag: string) {
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  const visibleRows = useMemo(
    () => (data ? filterRelations(data.relations, filter) : []),
    [data, filter],
  )
  const coreRows = useMemo(() => (data ? coreRelations(data.relations) : []), [data])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="flex flex-col gap-6 p-6">
        <EmptyStateCard
          title={rel.unavailableTitle}
          description={error?.message ?? rel.unavailableDescription}
          actionLabel={rel.retry}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  if (data.relations.length === 0) {
    return (
      <div className="flex flex-col gap-6 p-6">
        <EmptyStateCard title={rel.emptyTitle} description={rel.emptyDescription} />
      </div>
    )
  }

  const ov = data.overview
  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="grid gap-4 lg:grid-cols-3" data-testid="palmares-relations-overview">
        <HeroRefCard
          title={rel.hero.topAllyTitle}
          emptyLabel={rel.hero.topAllyEmpty}
          accent="outcome-win"
          relation={ov.top_ally}
          matchesPlayed={rel.hero.matchesPlayed}
        />
        <HeroRefCard
          title={rel.hero.topNemesisTitle}
          emptyLabel={rel.hero.topNemesisEmpty}
          accent="outcome-loss"
          relation={ov.top_nemesis}
          matchesPlayed={rel.hero.matchesPlayed}
        />
        <CoreHeroCard
          title={rel.hero.coreTitle}
          unit={rel.hero.coreUnit}
          count={ov.core_count}
          locale={text.intlLocale}
        />
      </div>

      <FilterChips active={filter} onChange={setFilter} labels={rel.chips} />

      <RelationsTable
        rows={visibleRows}
        labels={rel}
        locale={locale}
        onPlayerClick={goToExplorer}
        emptyMessage={rel.filterEmptyDescription}
      />

      <section className="flex flex-col gap-3">
        <div>
          <h2 className="text-base font-semibold text-foreground">{rel.core.sectionTitle}</h2>
          <p className="text-sm text-muted-foreground">{rel.core.sectionDescription}</p>
        </div>
        <CoreCards rows={coreRows} labels={rel} locale={text.intlLocale} onPlayerClick={goToExplorer} />
      </section>
    </div>
  )
}
