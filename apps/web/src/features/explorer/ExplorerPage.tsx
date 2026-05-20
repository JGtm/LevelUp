/**
 * ExplorerPage — page Explorer (recherche + filtres).
 *
 * Orchestrateur slim. Le détail de chaque mode est dans son propre fichier
 * (voir audit #6 god-file split) :
 *   - ExplorerPage.matchesMode.tsx — filtres + tableau paginé
 *   - ExplorerPage.playerMode.tsx  — recherche gamertag + briefing + tables
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
import { useExplorerMatches, useExplorerPlayer } from './queries'
import { DEFAULT_FILTER_CONTEXT } from '@/stores/createFilterStore'
import { useActiveSeason, seasonToPeriod } from '@/features/squad/useActiveSeason'
import type { ContextDescriptor } from '@/lib/match-nav/navContext'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { useAppShellStore } from '@/stores/appShellStore'

import { ExplorerMatchesMode } from './ExplorerPage.matchesMode'
import { ExplorerPlayerMode } from './ExplorerPage.playerMode'
import { buildExplorerFilterOptions } from './ExplorerPage.filterOptions'

type SearchMode = 'matches' | 'player'

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

  function openHeadToHead(gamertag: string) {
    void navigate({
      to: '/players/$playerSlug/compare',
      params: { playerSlug },
      search: { target: gamertag, from: 'explorer' },
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

  // ─── Options pour les MultiSelectFilter (extrait dans helper) ────────────
  const {
    expTypeOptions,
    playlistOptions,
    modeOptions,
    mapOptions,
    perfTierOptions,
    outcomeOptions,
    skillTierOptions,
    squadCountByValue,
  } = buildExplorerFilterOptions(summary, t)

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

        {mode === 'player' && (
          <ExplorerPlayerMode
            playerSlug={playerSlug}
            targetGamertag={targetGamertag}
            locale={locale}
            t={t}
            playerQuery={playerQuery}
            allyMatchIds={allyMatchIds}
            enemyMatchIds={enemyMatchIds}
            allyMatchesData={allyMatchesQuery.data}
            enemyMatchesData={enemyMatchesQuery.data}
            onSelectTarget={selectTarget}
            onOpenHeadToHead={openHeadToHead}
          />
        )}

        {mode === 'matches' && (
          <ExplorerMatchesMode
            playerSlug={playerSlug}
            t={t}
            startDate={startDate}
            endDate={endDate}
            matchIDSearch={matchIDSearch}
            squadScope={squadScope}
            squadCountByValue={squadCountByValue}
            onStartDateChange={handleStartDate}
            onEndDateChange={setEndDate}
            onMatchIDSearchChange={setMatchIDSearch}
            onSquadScopeChange={setSquadScope}
            seasons={seasons}
            activeSeason={activeSeason}
            saisonOpen={saisonOpen}
            onSaisonToggle={() => setSaisonOpen((o) => !o)}
            onSaisonClose={() => setSaisonOpen(false)}
            onSelectSeason={(s) => {
              const p = seasonToPeriod(s)
              setStartDate(p.start_date ?? '')
              setEndDate(p.end_date ?? '')
              setSaisonOpen(false)
            }}
            onClearPeriod={() => {
              setStartDate('')
              setEndDate('')
            }}
            expTypes={expTypes}
            playlists={playlists}
            modeNames={modeNames}
            mapNames={mapNames}
            outcomeFilter={outcomeFilter}
            perfTiers={perfTiers}
            skillTiers={skillTiers}
            expTypeOptions={expTypeOptions}
            playlistOptions={playlistOptions}
            modeOptions={modeOptions}
            mapOptions={mapOptions}
            outcomeOptions={outcomeOptions}
            perfTierOptions={perfTierOptions}
            skillTierOptions={skillTierOptions}
            rankedContext={rankedContext}
            onToggleExpType={(v) => toggleSet(setExpTypes, v)}
            onTogglePlaylist={(v) => toggleSet(setPlaylists, v)}
            onToggleModeName={(v) => toggleSet(setModeNames, v)}
            onToggleMapName={(v) => toggleSet(setMapNames, v)}
            onToggleOutcome={(v) => toggleSet(setOutcomeFilter, v)}
            onTogglePerfTier={(v) => toggleSet(setPerfTiers, v)}
            onToggleSkillTier={(v) => toggleSet(setSkillTiers, v)}
            sortKey={sortKey}
            onSortKeyChange={setSortKey}
            hasActiveFilter={hasActiveFilter}
            onResetFilters={resetFilters}
            matchesQuery={matchesQuery}
            matchesContextDescriptor={matchesContextDescriptor}
          />
        )}
      </div>
    </div>
  )
}
