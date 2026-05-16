/**
 * ExplorerPage — page Explorer (recherche + filtres).
 *
 * Mode Matchs : filtres (dropdowns + date range) + tableau paginé
 *               (ExplorerMatchesTable, repris du SquadMatchHistoryTable).
 * Mode Joueur : historique commun paginé (20/page) + badges encounter.
 *
 * URL params : ?mode=player&target=<gamertag> — auto-switch au chargement.
 */
import { useState, useEffect } from 'react'
import { useParams, useNavigate, useSearch } from '@tanstack/react-router'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { GamertagSearchInput } from './GamertagSearchInput'
import { MultiSelectFilter, type MultiSelectOption } from './MultiSelectFilter'
import { ExplorerMatchesTable } from './ExplorerMatchesTable'
import { ExplorerEncounterBriefing } from './ExplorerEncounterBriefing'
import { ExplorerActivityHeatmapChart } from './ExplorerActivityHeatmapChart'
import { useExplorerMatches, useExplorerPlayer } from './queries'
import { DEFAULT_FILTER_CONTEXT } from '@/stores/createFilterStore'
import { SaisonPill } from '@/components/shell/FilterOmnibar'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import type { LabelValue, MatchEncounterBadge } from '@/lib/api/types'
import type { ContextDescriptor } from '@/lib/match-nav/navContext'
import { formatMessage } from '@/lib/i18n/format'
import { SKILL_TIER_VALUES } from '@/lib/skillTiers'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { squadManifest, type SquadManifestKey } from '@/lib/i18n/generated/squad'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenCssVar, tokenVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

type SearchMode = 'matches' | 'player'

// ─── Helpers badges rencontre ────────────────────────────────────────────────

function isEncounterSemanticToken(s: string): s is SemanticToken {
  return s.startsWith('narrative-') || s.startsWith('outcome-') || s.startsWith('perf-')
}

function renderEncounterBadges(badges: MatchEncounterBadge[], locale: string) {
  const manifestLocale: 'fr' | 'en' = locale === 'en' ? 'en' : 'fr'
  const sqT = (key: SquadManifestKey, values?: Record<string, string | number>) =>
    formatMessage(squadManifest, key, manifestLocale, values)
  return badges.map((badge, i) => {
    const labelKey = badge.label_key as SquadManifestKey
    const ordinal =
      badge.detail && typeof badge.detail['ordinal'] === 'number'
        ? (badge.detail['ordinal'] as number)
        : undefined
    const label = ordinal !== undefined ? sqT(labelKey, { ordinal }) : sqT(labelKey)
    const colorVar = isEncounterSemanticToken(badge.color_token)
      ? tokenVar(badge.color_token as SemanticToken)
      : undefined
    return <NarrativeBadge key={i} label={label} colorVar={colorVar} size="lg" />
  })
}

// ─── ExplorerPage ─────────────────────────────────────────────────────────────

export function ExplorerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const search = useSearch({ from: '/players/$playerSlug/explorer/' }) as {
    mode?: SearchMode
    target?: string
  }

  const locale = useAppShellStore((s) => s.locale)

  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, locale, values)

  const [mode, setMode] = useState<SearchMode>(search.mode ?? 'matches')
  const [targetGamertag, setTargetGamertag] = useState(search.target ?? '')

  // ─── Filtres ───────────────────────────────────────────────────────────────
  const [perfTiers, setPerfTiers] = useState<Set<string>>(new Set())
  const [skillTiers, setSkillTiers] = useState<Set<string>>(new Set())
  const [outcomeFilter, setOutcomeFilter] = useState<Set<string>>(new Set())
  const [sortKey, setSortKey] = useState('start_time:desc')
  const [sortField, sortDir] = sortKey.split(':') as [string, string]

  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [saisonOpen, setSaisonOpen] = useState(false)
  const [squadScope, setSquadScope] = useState<'' | 'solo' | 'squad'>('')
  const [expTypes, setExpTypes] = useState<Set<string>>(new Set())
  const [playlists, setPlaylists] = useState<Set<string>>(new Set())
  const [mapNames, setMapNames] = useState<Set<string>>(new Set())
  const [modeNames, setModeNames] = useState<Set<string>>(new Set())
  const [matchIDSearch, setMatchIDSearch] = useState('')

  function toggleSet(setter: React.Dispatch<React.SetStateAction<Set<string>>>, value: string) {
    setter((prev) => {
      const next = new Set(prev)
      if (next.has(value)) next.delete(value)
      else next.add(value)
      return next
    })
  }

  function handleStartDate(v: string) {
    setStartDate(v)
    if (endDate && v && endDate < v) setEndDate('')
  }

  // Saisons : dérivées du catalog du titre courant. activeSeason est calculée
  // depuis les inputs date locaux (Du/Au), pas du filterContext shell — Explorer
  // override ce dernier (vue tout-historique) donc la saison agit comme un
  // raccourci sur les dates locales.
  const { seasons, activeSeason } = useActiveSeason({
    start_date: startDate || null,
    end_date: endDate || null,
  })

  // ranked_context auto-déduit du multi-select Type d'expérience.
  // Sélection mono-valeur "PVP classé" → "ranked" (gate skill_tier sur CSR).
  // Sélection mono-valeur "PVP non classé" → "unranked" (gate skill_tier sur LUSR).
  // Toute autre combinaison (multi-valeurs, PVE seul, vide) → "" : skill_tier
  // resté désactivé pour éviter le mélange CSR/LUSR ambigu.
  // Cf. thought_log 2026-05-09 P3 — fusion du single-select "Expérience" dans
  // le multi-select "Type d'expérience" (Option A).
  const rankedContext: 'ranked' | 'unranked' | '' = (() => {
    if (expTypes.size !== 1) return ''
    if (expTypes.has('PVP classé')) return 'ranked'
    if (expTypes.has('PVP non classé')) return 'unranked'
    return ''
  })()

  // Quand la dérivation change, le skill_tier doit être réinitialisé pour
  // éviter de garder un tier CSR sélectionné après bascule en non-classé.
  useEffect(() => {
    if (rankedContext === '') setSkillTiers(new Set())
  }, [rankedContext])

  // ─── URL sync ──────────────────────────────────────────────────────────────
  // Init unique depuis l'URL au mount — légitime pour hydrater l'état initial.
  useEffect(() => {
    if (search.mode) setMode(search.mode)
    if (search.target) setTargetGamertag(search.target)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  function setModeAndUrl(m: SearchMode) {
    setMode(m)
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: (prev) => ({ ...prev, mode: m }),
    })
  }

  function selectTarget(gamertag: string) {
    setTargetGamertag(gamertag)
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: (prev) => ({ ...prev, mode: 'player', target: gamertag }),
    })
  }

  // ─── Queries ───────────────────────────────────────────────────────────────
  // Explorer = vue historique complète, 100% locale. On part de DEFAULT_FILTER_CONTEXT
  // (aucun héritage du store global solo/squad) et on pilote le scope via les
  // filtres locaux date/exp/playlist/etc. ci-dessus. pageSize=200 = max accepté
  // par maxPageSize backend ; pagination client gère le découpage 20/page.
  const explorerFilterContext = DEFAULT_FILTER_CONTEXT
  // Hash constant : le scope global n'influence plus la query. Les variations
  // locales (perfTiers/skillTiers/dates/etc.) sont déjà dans la queryKey.
  const filterContextHash = 'explorer-local'
  const matchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: explorerFilterContext,
      pagination: { page: 1, page_size: 10000 },
      include_export_hint: true,
      perf_tiers: perfTiers.size > 0 ? [...perfTiers].map(Number) : undefined,
      skill_tiers: skillTiers.size > 0 ? [...skillTiers] : undefined,
      ranked_context: rankedContext || undefined,
      outcome_filter: outcomeFilter.size > 0 ? [...outcomeFilter].map(Number) : undefined,
      sort_field: sortField,
      sort_dir: sortDir,
      match_start_date: startDate || null,
      match_end_date: endDate || null,
      experience_types: expTypes.size > 0 ? [...expTypes] : undefined,
      playlists: playlists.size > 0 ? [...playlists] : undefined,
      map_names: mapNames.size > 0 ? [...mapNames] : undefined,
      mode_names: modeNames.size > 0 ? [...modeNames] : undefined,
      squad_scope: squadScope || undefined,
      match_id_search: matchIDSearch || undefined,
    },
    filterContextHash,
  )

  const playerQuery = useExplorerPlayer(playerSlug, {
    target_gamertag: targetGamertag,
  })

  // Mode Joueur : extraction des match_ids communs séparés par rôle
  // (ally vs enemy) depuis la réponse player-query, pour piloter les 2
  // tableaux scopés ci-dessous.
  const allyMatchIds = (playerQuery.data?.common_matches ?? [])
    .filter((m) => m.were_teammates)
    .map((m) => m.match_id)
  const enemyMatchIds = (playerQuery.data?.common_matches ?? [])
    .filter((m) => !m.were_teammates)
    .map((m) => m.match_id)

  // Requête tableau "matchs en allié" — réutilise le pipeline matches-query
  // avec un filtre match_ids (whitelist). Activée uniquement quand on a des
  // match_ids ET qu'on est en mode Joueur.
  const allyMatchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: explorerFilterContext,
      pagination: { page: 1, page_size: 10000 },
      sort_field: 'start_time',
      sort_dir: 'desc',
      match_ids: allyMatchIds,
    },
    filterContextHash,
    mode === 'player' && allyMatchIds.length > 0,
  )

  const enemyMatchesQuery = useExplorerMatches(
    playerSlug,
    {
      filters: explorerFilterContext,
      pagination: { page: 1, page_size: 10000 },
      sort_field: 'start_time',
      sort_dir: 'desc',
      match_ids: enemyMatchIds,
    },
    filterContextHash,
    mode === 'player' && enemyMatchIds.length > 0,
  )

  const summary = matchesQuery.data?.summary

  // Descriptor du contexte de navigation pour les matchs ouverts depuis le
  // tableau mode Matchs. Priorité au filtre le plus spécifique : 1 playlist >
  // 1 mode > période active > undefined (Q25 fallback générique côté match-view).
  const matchesContextDescriptor: ContextDescriptor | undefined = (() => {
    if (playlists.size === 1) {
      const [name] = [...playlists]
      return name ? { kind: 'playlist', name } : undefined
    }
    if (modeNames.size === 1) {
      const [category] = [...modeNames]
      return category ? { kind: 'mode', category } : undefined
    }
    if (startDate || endDate) {
      const toIso = (d: string) => (d ? new Date(d).toISOString() : undefined)
      return { kind: 'period', from: toIso(startDate), to: toIso(endDate) }
    }
    return undefined
  })()

  // ─── Options pour les MultiSelectFilter ───────────────────────────────────
  const expTypeOptions: MultiSelectOption[] = (summary?.available_experience_types ?? []).map(
    (v) => ({ value: v, label: v }),
  )
  const playlistOptions: MultiSelectOption[] = (summary?.available_playlists ?? []).map((v) => ({
    value: v,
    label: v,
  }))
  const modeOptions: MultiSelectOption[] = (summary?.available_modes ?? []).map((v) => ({
    value: v,
    label: v,
  }))
  const mapOptions: MultiSelectOption[] = (summary?.available_maps ?? []).map((v) => ({
    value: v,
    label: v,
  }))

  // Mappe les counts backend (LabelValue[]) sur les options frontend (qui portent
  // labels i18n + swatch). Si pas de backend data, count reste undefined (pas de
  // grayout). Match par `value` exact.
  function withCounts(opts: MultiSelectOption[], backend?: LabelValue[]): MultiSelectOption[] {
    if (!backend) return opts
    const map = new Map(backend.map((b) => [b.value, b.count]))
    return opts.map((o) => ({ ...o, count: map.get(o.value) ?? 0 }))
  }

  const perfTierOptions: MultiSelectOption[] = withCounts(
    [
      { value: '1', label: t('explorer.filters.perf_tier_excellent'), swatch: tokenCssVar('perf-tier-1' as SemanticToken) },
      { value: '2', label: t('explorer.filters.perf_tier_bon'), swatch: tokenCssVar('perf-tier-2' as SemanticToken) },
      { value: '3', label: t('explorer.filters.perf_tier_correct'), swatch: tokenCssVar('perf-tier-3' as SemanticToken) },
      { value: '4', label: t('explorer.filters.perf_tier_faible'), swatch: tokenCssVar('perf-tier-4' as SemanticToken) },
      { value: '5', label: t('explorer.filters.perf_tier_mauvais'), swatch: tokenCssVar('perf-tier-5' as SemanticToken) },
    ],
    summary?.available_perf_tiers,
  )

  const outcomeOptions: MultiSelectOption[] = withCounts(
    [
      { value: '2', label: t('explorer.filters.outcome_win'), swatch: tokenCssVar('outcome-win' as SemanticToken) },
      { value: '3', label: t('explorer.filters.outcome_loss'), swatch: tokenCssVar('outcome-loss' as SemanticToken) },
      { value: '1', label: t('explorer.filters.outcome_tie'), swatch: tokenCssVar('outcome-draw' as SemanticToken) },
    ],
    summary?.available_outcomes,
  )

  const skillTierOptions: MultiSelectOption[] = withCounts(
    [
      { value: SKILL_TIER_VALUES[0], label: t('explorer.filters.skill_tier_bronze') },
      { value: SKILL_TIER_VALUES[1], label: t('explorer.filters.skill_tier_silver') },
      { value: SKILL_TIER_VALUES[2], label: t('explorer.filters.skill_tier_gold') },
      { value: SKILL_TIER_VALUES[3], label: t('explorer.filters.skill_tier_platinum') },
      { value: SKILL_TIER_VALUES[4], label: t('explorer.filters.skill_tier_diamond') },
      { value: SKILL_TIER_VALUES[5], label: t('explorer.filters.skill_tier_onyx') },
    ],
    summary?.available_skill_tiers,
  )

  // Count pour le single-select squad scope — interpolé dans les <option>
  // labels et désactive celles à count=0.
  const squadCountByValue = new Map(
    (summary?.available_squad_scopes ?? []).map((b) => [b.value, b.count]),
  )

  const hasActiveFilter =
    !!startDate ||
    !!endDate ||
    !!squadScope ||
    !!matchIDSearch ||
    expTypes.size > 0 ||
    playlists.size > 0 ||
    mapNames.size > 0 ||
    modeNames.size > 0 ||
    perfTiers.size > 0 ||
    skillTiers.size > 0 ||
    outcomeFilter.size > 0 ||
    sortKey !== 'start_time:desc'

  function resetFilters() {
    setStartDate('')
    setEndDate('')
    setSquadScope('')
    setMatchIDSearch('')
    setExpTypes(new Set())
    setPlaylists(new Set())
    setMapNames(new Set())
    setModeNames(new Set())
    setPerfTiers(new Set())
    setSkillTiers(new Set())
    setOutcomeFilter(new Set())
    setSortKey('start_time:desc')
  }

  return (
    <div className="flex flex-col">
      <div className="p-6 space-y-6">
        {/* Onglets mode */}
        <div className="flex gap-2">
          <Button
            variant={mode === 'matches' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setModeAndUrl('matches')}
          >
            {t('explorer.mode.matches')}
          </Button>
          <Button
            variant={mode === 'player' ? 'default' : 'outline'}
            size="sm"
            onClick={() => setModeAndUrl('player')}
          >
            {t('explorer.mode.player')}
          </Button>
        </div>

        {/* ─── Mode Joueur ─────────────────────────────────────────────────── */}
        {mode === 'player' && (
          <div className="space-y-4">
            <div className="flex items-center gap-2 flex-wrap">
              <GamertagSearchInput onSelect={selectTarget} initialValue={targetGamertag} />
              {targetGamertag && !!playerQuery.data?.badges?.length && (
                <div className="flex flex-wrap gap-1.5">
                  {renderEncounterBadges(playerQuery.data!.badges!, locale)}
                </div>
              )}
              {targetGamertag && playerQuery.data?.encounter_stats && (
                <button
                  type="button"
                  onClick={() => void navigate({
                    to: '/players/$playerSlug/compare',
                    params: { playerSlug },
                    search: { target: playerQuery.data?.target_gamertag ?? targetGamertag, from: 'explorer' },
                  })}
                  className="inline-flex h-9 items-center rounded border border-input bg-background px-3 text-xs font-medium hover:bg-muted transition-colors"
                >
                  {t('explorer.player.head_to_head')}
                </button>
              )}
            </div>

            {!targetGamertag && (
              <Card>
                <CardContent className="py-4 pt-4">
                  <EmptyStateNotice
                    title={t('explorer.player.no_selection_title')}
                    description={t('explorer.player.no_selection_description')}
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag && playerQuery.isLoading && !playerQuery.data && (
              <div className="flex justify-center py-8">
                <Spinner label={t('explorer.player.searching')} />
              </div>
            )}

            {targetGamertag && playerQuery.isError && (
              <Card>
                <CardContent className="py-4 pt-4">
                  <EmptyStateNotice
                    title={t('explorer.player.error_title')}
                    description={t('explorer.player.error_description')}
                  />
                </CardContent>
              </Card>
            )}

            {targetGamertag &&
              !playerQuery.isLoading &&
              !playerQuery.isError &&
              !playerQuery.data && (
                <Card>
                  <CardContent className="py-4 pt-4">
                    <EmptyStateNotice
                      title={t('explorer.player.empty_title')}
                      description={t('explorer.player.empty_description')}
                    />
                  </CardContent>
                </Card>
              )}

            {targetGamertag && playerQuery.data && (
              <>
                {playerQuery.data.encounter_stats ? (
                  <ExplorerEncounterBriefing
                    stats={playerQuery.data.encounter_stats}
                    locale={locale}
                  />
                ) : (
                  <Card>
                    <CardContent className="py-4 pt-4">
                      <EmptyStateNotice
                        title={t('explorer.player.no_common_matches_title')}
                        description={t('explorer.player.no_common_matches_description')}
                      />
                    </CardContent>
                  </Card>
                )}

                {/* Tableau "matchs en allié" — affiché uniquement si on a au moins
                    un match commun en tant qu'alliés. Pattern team-banner aligné
                    sur MatchScoreboard (token team-ally). */}
                {allyMatchIds.length > 0 && allyMatchesQuery.data && (
                  <ExplorerMatchesTable
                    rows={allyMatchesQuery.data.table.items}
                    playerSlug={playerSlug}
                    teamBanner={{
                      variant: 'ally',
                      label: t('explorer.player.table_as_ally'),
                    }}
                    contextDescriptor={{
                      kind: 'with_player',
                      gamertag: playerQuery.data.target_gamertag || targetGamertag,
                    }}
                    alwaysShowPagination
                  />
                )}

                {/* Tableau "matchs en ennemi" — token team-enemy. */}
                {enemyMatchIds.length > 0 && enemyMatchesQuery.data && (
                  <ExplorerMatchesTable
                    rows={enemyMatchesQuery.data.table.items}
                    playerSlug={playerSlug}
                    teamBanner={{
                      variant: 'enemy',
                      label: t('explorer.player.table_as_enemy'),
                    }}
                    contextDescriptor={{
                      kind: 'with_player',
                      gamertag: playerQuery.data.target_gamertag || targetGamertag,
                    }}
                    alwaysShowPagination
                  />
                )}

                {/* Heatmap d'activité commune — agrégat jour × heure de tous
                    les matchs communs (alliés + ennemis). Coloration par
                    intensité d'activité (count), pas par win-rate. */}
                {(playerQuery.data.activity_heatmap?.length ?? 0) > 0 && (
                  <ExplorerActivityHeatmapChart
                    title={t('explorer.player.activity_heatmap_title')}
                    cells={playerQuery.data.activity_heatmap ?? []}
                  />
                )}
              </>
            )}
          </div>
        )}

        {/* ─── Mode Matchs ─────────────────────────────────────────────────── */}
        {mode === 'matches' && (
          <div className="space-y-4">
            {/* Filtres */}
            <Card>
              <CardContent className="py-3 pt-3 space-y-3">
                {/* Ligne 1 : période + ID + escouade */}
                <div className="flex flex-wrap items-center gap-x-3 gap-y-2">
                  <div className="flex items-center gap-1.5">
                    <label className="text-xs text-muted-foreground whitespace-nowrap">
                      {t('explorer.filters.date_from_label')}
                    </label>
                    <input
                      type="date"
                      value={startDate}
                      onChange={(e) => handleStartDate(e.target.value)}
                      className="rounded border border-input px-2 py-1 text-sm bg-background w-36"
                    />
                  </div>
                  <div className="flex items-center gap-1.5">
                    <label className="text-xs text-muted-foreground whitespace-nowrap">
                      {t('explorer.filters.date_to_label')}
                    </label>
                    <input
                      type="date"
                      value={endDate}
                      min={startDate || undefined}
                      onChange={(e) => setEndDate(e.target.value)}
                      className="rounded border border-input px-2 py-1 text-sm bg-background w-36"
                    />
                  </div>
                  {seasons.length > 0 && (
                    <SaisonPill
                      open={saisonOpen}
                      onToggle={() => setSaisonOpen((o) => !o)}
                      onClose={() => setSaisonOpen(false)}
                      seasons={seasons}
                      activeSeason={activeSeason}
                      onSelectSeason={(s) => {
                        const p = seasonToPeriod(s)
                        setStartDate(p.start_date ?? '')
                        setEndDate(p.end_date ?? '')
                        setSaisonOpen(false)
                      }}
                      onClear={() => {
                        setStartDate('')
                        setEndDate('')
                      }}
                    />
                  )}
                  <input
                    type="text"
                    value={matchIDSearch}
                    onChange={(e) => setMatchIDSearch(e.target.value)}
                    placeholder={t('explorer.filters.match_id')}
                    className="rounded border border-input px-2 py-1 text-sm bg-background w-52"
                  />
                  <select
                    value={squadScope}
                    onChange={(e) => setSquadScope(e.target.value as '' | 'solo' | 'squad')}
                    className="rounded border border-input px-2 py-1 text-sm bg-background"
                  >
                    {(['', 'solo', 'squad'] as const).map((v) => {
                      const labelKey =
                        v === '' ? 'explorer.filters.context_all'
                        : v === 'solo' ? 'explorer.filters.context_solo'
                        : 'explorer.filters.context_squad'
                      const c = squadCountByValue.get(v)
                      const isCurrent = v === squadScope
                      return (
                        <option key={v} value={v} disabled={c === 0 && !isCurrent}>
                          {t(labelKey)}{c !== undefined ? ` (${c})` : ''}
                        </option>
                      )
                    })}
                  </select>
                </div>

                {/* Ligne 2 : tous les multi-selects */}
                <div className="flex flex-wrap items-center gap-2">
                  <MultiSelectFilter
                    options={expTypeOptions}
                    selected={expTypes}
                    toggle={(v) => toggleSet(setExpTypes, v)}
                    placeholder={t('explorer.filters.experience_type')}
                  />
                  <MultiSelectFilter
                    options={playlistOptions}
                    selected={playlists}
                    toggle={(v) => toggleSet(setPlaylists, v)}
                    placeholder={t('explorer.filters.playlist')}
                  />
                  <MultiSelectFilter
                    options={modeOptions}
                    selected={modeNames}
                    toggle={(v) => toggleSet(setModeNames, v)}
                    placeholder={t('explorer.filters.mode')}
                  />
                  <MultiSelectFilter
                    options={mapOptions}
                    selected={mapNames}
                    toggle={(v) => toggleSet(setMapNames, v)}
                    placeholder={t('explorer.filters.map')}
                  />
                  <MultiSelectFilter
                    options={outcomeOptions}
                    selected={outcomeFilter}
                    toggle={(v) => toggleSet(setOutcomeFilter, v)}
                    placeholder={t('explorer.filters.outcome_label')}
                    alwaysShow
                  />
                  <MultiSelectFilter
                    options={perfTierOptions}
                    selected={perfTiers}
                    toggle={(v) => toggleSet(setPerfTiers, v)}
                    placeholder={t('explorer.filters.perf_tier_label')}
                    alwaysShow
                  />
                  <MultiSelectFilter
                    options={skillTierOptions}
                    selected={skillTiers}
                    toggle={(v) => toggleSet(setSkillTiers, v)}
                    placeholder={t('explorer.filters.skill_tier_label')}
                    alwaysShow
                    disabled={rankedContext === ''}
                    title={rankedContext === '' ? t('explorer.filters.skill_tier_disabled') : undefined}
                  />
                  {hasActiveFilter && (
                    <button
                      className="ml-auto text-xs text-primary hover:underline"
                      onClick={resetFilters}
                    >
                      {t('explorer.filters.reset')}
                    </button>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* Résultats */}
            <div className="space-y-2">
              {matchesQuery.isLoading ? (
                <div className="flex justify-center py-16">
                  <Spinner label={t('explorer.matches.loading')} />
                </div>
              ) : matchesQuery.isError ? (
                <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-6 text-center">
                  <p className="text-destructive">{t('explorer.matches.error')}</p>
                  <button
                    onClick={() => matchesQuery.refetch()}
                    className="mt-2 text-sm text-primary underline"
                  >
                    {t('explorer.matches.retry')}
                  </button>
                </div>
              ) : matchesQuery.data ? (
                <>
                  {/* Barre résultats + tri + export */}
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-sm text-muted-foreground">
                      {t('explorer.matches.count_label', {
                        n: matchesQuery.data.summary?.total_matches ?? 0,
                      })}
                    </p>
                    <div className="flex items-center gap-2 shrink-0">
                      <div className="flex items-center gap-1.5">
                        <label className="text-xs text-muted-foreground whitespace-nowrap">
                          {t('explorer.sort.label')} :
                        </label>
                        <select
                          value={sortKey}
                          onChange={(e) => setSortKey(e.target.value)}
                          className="rounded border border-input px-2 py-1 text-xs bg-background"
                        >
                          <option value="start_time:desc">{t('explorer.sort.start_time_desc')}</option>
                          <option value="start_time:asc">{t('explorer.sort.start_time_asc')}</option>
                          <option value="performance_score_relative:desc">{t('explorer.sort.perf_desc')}</option>
                          <option value="performance_score_relative:asc">{t('explorer.sort.perf_asc')}</option>
                          <option value="kda:desc">{t('explorer.sort.kda_desc')}</option>
                          <option value="kills:desc">{t('explorer.sort.kills_desc')}</option>
                          <option value="delta_mmr:desc">{t('explorer.sort.delta_mmr_desc')}</option>
                          <option value="outcome:desc">{t('explorer.sort.outcome')}</option>
                        </select>
                      </div>
                      {matchesQuery.data.export_hint?.token && (
                        <a
                          href={`${import.meta.env.VITE_API_BASE_URL ?? '/api/v1'}/players/${playerSlug}/pages/match-history/export?token=${encodeURIComponent(matchesQuery.data.export_hint.token)}`}
                          download
                          title={t('explorer.matches.export_csv')}
                          className="inline-flex h-8 items-center rounded-md border border-input bg-background px-3 text-xs font-medium text-foreground hover:bg-muted transition-colors"
                        >
                          {t('explorer.matches.export_csv')}
                        </a>
                      )}
                    </div>
                  </div>

                  {/* Tableau résultats — composant repris depuis Squad */}
                  <ExplorerMatchesTable
                    rows={matchesQuery.data.table.items}
                    playerSlug={playerSlug}
                    contextDescriptor={matchesContextDescriptor}
                  />
                </>
              ) : (
                <Card>
                  <CardContent className="py-4 pt-4">
                    <EmptyStateNotice
                      title={t('explorer.matches.empty_title')}
                      description={t('explorer.matches.empty_description')}
                    />
                  </CardContent>
                </Card>
              )}
            </div>
          </div>
        )}
      </div>

    </div>
  )
}
