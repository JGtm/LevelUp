/**
 * ExplorerPage — bloc "Mode Matchs".
 *
 * Découpé depuis ExplorerPage.tsx (audit #6 god-file split).
 * Contenu : barre de filtres + tableau résultats + tri/export.
 * Tous les états (filtres, sort, dates) sont contrôlés par le parent.
 */
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { MultiSelectFilter, type MultiSelectOption } from './MultiSelectFilter'
import { ExplorerMatchesTable } from './ExplorerMatchesTable'
import { ExplorerBriefingStrip } from './ExplorerBriefingStrip'
import { SaisonPill } from '@/components/shell/FilterOmnibar'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { ContextDescriptor, MatchFilterSpec } from '@/lib/match-nav/navContext'
import type { ExplorerMatchRow, ExplorerMatchesQueryResponse } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { SeasonEntry } from '@/lib/i18n/fieldMappings'

/**
 * Normalise les lignes du tableau Explorer renvoyées par le contrat
 * (`map_ui` / `mode_ui` / `playlist_label` peuvent être `null` côté Go) vers
 * `ExplorerMatchRow[]` attendu par <ExplorerMatchesTable> (libellés string).
 * Coalesce `null` → '' au moment du passage de prop ; aucun changement de rendu
 * (une cellule null s'affichait déjà vide).
 */
export function normalizeExplorerTableRows(
  items: ExplorerMatchesQueryResponse['table']['items'],
): ExplorerMatchRow[] {
  return (items ?? []).map((r) => ({
    ...r,
    map_ui: r.map_ui ?? '',
    mode_ui: r.mode_ui ?? '',
    playlist_label: r.playlist_label ?? '',
  }))
}

export interface ExplorerMatchesModeProps {
  playerSlug: string
  t: (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

  // Date / ID / scope filters
  startDate: string
  endDate: string
  matchIDSearch: string
  squadScope: '' | 'solo' | 'squad'
  squadCountByValue: Map<string, number>
  onStartDateChange: (v: string) => void
  onEndDateChange: (v: string) => void
  onMatchIDSearchChange: (v: string) => void
  onSquadScopeChange: (v: '' | 'solo' | 'squad') => void

  // Seasons
  seasons: SeasonEntry[]
  activeSeason: SeasonEntry | null
  saisonOpen: boolean
  onSaisonToggle: () => void
  onSaisonClose: () => void
  onSelectSeason: (s: SeasonEntry) => void
  onClearPeriod: () => void

  // Multi-select filters
  expTypes: Set<string>
  playlists: Set<string>
  modeNames: Set<string>
  mapNames: Set<string>
  outcomeFilter: Set<string>
  perfTiers: Set<string>
  skillTiers: Set<string>
  expTypeOptions: MultiSelectOption[]
  playlistOptions: MultiSelectOption[]
  modeOptions: MultiSelectOption[]
  mapOptions: MultiSelectOption[]
  outcomeOptions: MultiSelectOption[]
  perfTierOptions: MultiSelectOption[]
  skillTierOptions: MultiSelectOption[]
  rankedContext: 'ranked' | 'unranked' | ''
  onToggleExpType: (v: string) => void
  onTogglePlaylist: (v: string) => void
  onToggleModeName: (v: string) => void
  onToggleMapName: (v: string) => void
  onToggleOutcome: (v: string) => void
  onTogglePerfTier: (v: string) => void
  onToggleSkillTier: (v: string) => void

  // Reset
  hasActiveFilter: boolean
  onResetFilters: () => void

  // Results
  matchesQuery: {
    data: ExplorerMatchesQueryResponse | undefined
    isLoading: boolean
    isError: boolean
    refetch: () => void
  }
  matchesContextDescriptor: ContextDescriptor | undefined
  /** filterSpec dérivé des filtres Explorer locaux (Phase 4) — propagé à la
   *  nav contextuelle prev/next via ExplorerMatchesTable.filterSpecOverride. */
  matchesFilterSpec?: MatchFilterSpec
}

export function ExplorerMatchesMode(props: ExplorerMatchesModeProps) {
  return (
    <div className="space-y-4">
      <ExplorerFiltersBar {...props} />
      <ExplorerMatchesResultsBlock
        playerSlug={props.playerSlug}
        t={props.t}
        matchesQuery={props.matchesQuery}
        matchesContextDescriptor={props.matchesContextDescriptor}
        matchesFilterSpec={props.matchesFilterSpec}
      />
    </div>
  )
}

// ── Barre de filtres ───────────────────────────────────────────────────────

function ExplorerFiltersBar({
  t,
  startDate,
  endDate,
  matchIDSearch,
  squadScope,
  squadCountByValue,
  onStartDateChange,
  onEndDateChange,
  onMatchIDSearchChange,
  onSquadScopeChange,
  seasons,
  activeSeason,
  saisonOpen,
  onSaisonToggle,
  onSaisonClose,
  onSelectSeason,
  onClearPeriod,
  expTypes,
  playlists,
  modeNames,
  mapNames,
  outcomeFilter,
  perfTiers,
  skillTiers,
  expTypeOptions,
  playlistOptions,
  modeOptions,
  mapOptions,
  outcomeOptions,
  perfTierOptions,
  skillTierOptions,
  rankedContext,
  onToggleExpType,
  onTogglePlaylist,
  onToggleModeName,
  onToggleMapName,
  onToggleOutcome,
  onTogglePerfTier,
  onToggleSkillTier,
  hasActiveFilter,
  onResetFilters,
}: ExplorerMatchesModeProps) {
  // Le filtre paliers de skill (CSR Halo Bronze→Onyx) est une surface 'ranked' :
  // masqué pour un titre sans rang (NO-OP halo_infinite). Distinct du `disabled`
  // existant, qui dépend du contexte playlist classée sélectionné.
  const hasRanked = useCapability('ranked')
  return (
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
              onChange={(e) => onStartDateChange(e.target.value)}
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
              onChange={(e) => onEndDateChange(e.target.value)}
              className="rounded border border-input px-2 py-1 text-sm bg-background w-36"
            />
          </div>
          {seasons.length > 0 && (
            <SaisonPill
              open={saisonOpen}
              onToggle={onSaisonToggle}
              onClose={onSaisonClose}
              seasons={seasons}
              activeSeason={activeSeason}
              onSelectSeason={onSelectSeason}
              onClear={onClearPeriod}
              // Aligne la police sur les autres filtres de la barre Explorer (text-sm).
              dense={false}
            />
          )}
          <input
            type="text"
            value={matchIDSearch}
            onChange={(e) => onMatchIDSearchChange(e.target.value)}
            placeholder={t('explorer.filters.match_id')}
            className="rounded border border-input px-2 py-1 text-sm bg-background w-52"
          />
          <select
            value={squadScope}
            onChange={(e) => onSquadScopeChange(e.target.value as '' | 'solo' | 'squad')}
            className="rounded border border-input bg-background px-2 py-1 text-sm text-foreground"
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
            toggle={onToggleExpType}
            placeholder={t('explorer.filters.experience_type')}
          />
          <MultiSelectFilter
            options={playlistOptions}
            selected={playlists}
            toggle={onTogglePlaylist}
            placeholder={t('explorer.filters.playlist')}
          />
          <MultiSelectFilter
            options={modeOptions}
            selected={modeNames}
            toggle={onToggleModeName}
            placeholder={t('explorer.filters.mode')}
          />
          <MultiSelectFilter
            options={mapOptions}
            selected={mapNames}
            toggle={onToggleMapName}
            placeholder={t('explorer.filters.map')}
          />
          <MultiSelectFilter
            options={outcomeOptions}
            selected={outcomeFilter}
            toggle={onToggleOutcome}
            placeholder={t('explorer.filters.outcome_label')}
            alwaysShow
          />
          <MultiSelectFilter
            options={perfTierOptions}
            selected={perfTiers}
            toggle={onTogglePerfTier}
            placeholder={t('explorer.filters.perf_tier_label')}
            alwaysShow
          />
          {hasRanked && (
            <MultiSelectFilter
              options={skillTierOptions}
              selected={skillTiers}
              toggle={onToggleSkillTier}
              placeholder={t('explorer.filters.skill_tier_label')}
              alwaysShow
              disabled={rankedContext === ''}
              title={rankedContext === '' ? t('explorer.filters.skill_tier_disabled') : undefined}
            />
          )}
          {hasActiveFilter && (
            <button
              className="ml-auto text-xs text-primary hover:underline"
              onClick={onResetFilters}
            >
              {t('explorer.filters.reset')}
            </button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

// ── Bloc résultats ─────────────────────────────────────────────────────────

interface ResultsBlockProps {
  playerSlug: string
  t: (key: ExplorerManifestKey, values?: Record<string, string | number>) => string
  matchesQuery: ExplorerMatchesModeProps['matchesQuery']
  matchesContextDescriptor: ContextDescriptor | undefined
  matchesFilterSpec?: MatchFilterSpec
}

// Exporté pour tester la mise en page (compteur « N matchs trouvés » retiré, export CSV
// ancré à GAUCHE du pied du tableau) sans monter toute la barre de filtres.
export function ExplorerMatchesResultsBlock({
  playerSlug,
  t,
  matchesQuery,
  matchesContextDescriptor,
  matchesFilterSpec,
}: ResultsBlockProps) {
  if (matchesQuery.isLoading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t('explorer.matches.loading')} />
      </div>
    )
  }
  if (matchesQuery.isError) {
    return (
      <div className="rounded-lg border border-destructive/30 bg-destructive/10 p-6 text-center">
        <p className="text-destructive">{t('explorer.matches.error')}</p>
        <button
          onClick={() => matchesQuery.refetch()}
          className="mt-2 text-sm text-primary underline"
        >
          {t('explorer.matches.retry')}
        </button>
      </div>
    )
  }
  if (!matchesQuery.data) {
    return (
      <Card>
        <CardContent className="py-4 pt-4">
          <EmptyStateNotice
            title={t('explorer.matches.empty_title')}
            description={t('explorer.matches.empty_description')}
          />
        </CardContent>
      </Card>
    )
  }
  return (
    <div className="space-y-2">
      {/* Bandeau de briefing (mode Matchs) — au-dessus du tableau. Le compteur redondant
          « N matchs trouvés » a été retiré : la tuile Matchs du briefing le porte, et le
          pied du tableau accueille désormais le bouton Export CSV à sa place. */}
      <ExplorerBriefingStrip briefing={matchesQuery.data.briefing} t={t} />

      {/* Tableau résultats — composant repris depuis Squad. `sortable` active le
          tri CLIENT par clic sur les en-têtes, sur toutes les colonnes (toutes les
          lignes du scope sont chargées ; cf. ExplorerMatchesTable). L'export CSV est
          ancré à GAUCHE du pied du tableau via `footerLeadingSlot` (à la place du
          compteur, redondant avec la tuile Matchs du briefing) ; il reste visible même
          sans pagination. Sans token d'export → slot `undefined` → le pied retombe sur
          son compteur (cas rare). */}
      <ExplorerMatchesTable
        rows={normalizeExplorerTableRows(matchesQuery.data.table.items)}
        playerSlug={playerSlug}
        contextDescriptor={matchesContextDescriptor}
        filterSpecOverride={matchesFilterSpec}
        sortable
        footerLeadingSlot={
          matchesQuery.data.export_hint?.token ? (
            <a
              href={`${import.meta.env.VITE_API_BASE_URL ?? '/api/v1'}/players/${playerSlug}/pages/match-history/export?token=${encodeURIComponent(matchesQuery.data.export_hint.token)}`}
              download
              title={t('explorer.matches.export_csv')}
              className="inline-flex h-8 shrink-0 items-center rounded-md border border-input bg-background px-3 text-xs font-medium text-foreground hover:bg-muted transition-colors"
            >
              {t('explorer.matches.export_csv')}
            </a>
          ) : undefined
        }
      />
    </div>
  )
}
