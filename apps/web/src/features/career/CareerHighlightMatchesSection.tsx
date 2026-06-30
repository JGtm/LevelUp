/**
 * CareerHighlightMatchesSection — section "Matchs marquants" de la page Carrière.
 *
 * Filtres locaux à la section (state non synchronisé avec le store global) :
 *  - Expérience : single-select Tous / Classé / Non classé
 *  - Saisons    : multi-select (réutilise MultiSelectFilter de l'Explorer)
 *
 * Cascade côté backend : les counts par option respectent l'autre filtre actif
 * (ex. "Saisons" affiche les counts par saison conditionnés au filtre Expérience).
 *
 * Toggle "Meilleures performances" / "Pires performances" → bascule entre
 * 15 best (outcome=2) et 15 worst (outcome=3) matchs au format ExplorerMatchRow
 * (réutilisation directe de ExplorerMatchesTable, mêmes 21 colonnes que la
 * page Explorer).
 *
 * Pas de wrapper Card / border autour du tableau — le user a explicitement
 * demandé "pas dans un bloc comme la section Citations".
 */
import { useMemo, useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import { MultiSelectFilter, type MultiSelectOption } from '@/features/explorer/MultiSelectFilter'
import { ExperienceDropdown, type Experience, type ExperienceCount } from '@/features/_shared/ExperienceDropdown'
import { Spinner } from '@/components/ui/spinner'
import { useCareerHighlightMatches } from './queries'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSeasons } from '@/lib/i18n/fieldMappings'
import { tokenCssVar } from '@/lib/accessibility'
import type { CareerHighlightFilters } from '@/lib/api/types'

type Variant = 'best' | 'worst'

export function CareerHighlightMatchesSection() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const seasons = useSeasons()

  const [variant, setVariant] = useState<Variant>('best')
  const [experience, setExperience] = useState<Experience>('all')
  const [selectedSeasons, setSelectedSeasons] = useState<Set<string>>(() => new Set())
  const [selectedModes, setSelectedModes] = useState<Set<string>>(() => new Set())
  const [selectedPlaylists, setSelectedPlaylists] = useState<Set<string>>(() => new Set())

  const filters: CareerHighlightFilters = useMemo(
    () => ({
      experience,
      season_ids: Array.from(selectedSeasons),
      mode_uis: Array.from(selectedModes),
      playlist_names: Array.from(selectedPlaylists),
    }),
    [experience, selectedSeasons, selectedModes, selectedPlaylists],
  )

  const { data, isLoading, isError } = useCareerHighlightMatches(playerSlug, filters)

  // Le contrat autorise best_matches / worst_matches = null (le backend Go peut
  // renvoyer null pour un filtre sans résultat).
  const rows = !data ? [] : (variant === 'best' ? data.best_matches : data.worst_matches) ?? []

  // Le contrat type `value: string` ; ExperienceDropdown attend l'union Experience.
  // Le backend ne renvoie que all/ranked/unranked pour ce filtre → on narrow.
  const VALID_EXPERIENCES: readonly Experience[] = ['all', 'ranked', 'unranked']
  const experienceCounts: ExperienceCount[] = useMemo(
    () =>
      (data?.available_experience ?? [])
        .filter((c): c is { value: Experience; count: number } =>
          (VALID_EXPERIENCES as readonly string[]).includes(c.value),
        )
        .map((c) => ({ value: c.value, count: c.count })),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [data?.available_experience],
  )

  // Construit les options de la dropdown Saisons à partir du catalog des saisons
  // côté frontend (label localisé), enrichies des cascade counts du backend.
  const seasonCountsMap = useMemo(() => {
    const map = new Map<string, number>()
    for (const c of data?.available_seasons ?? []) map.set(c.value, c.count)
    return map
  }, [data?.available_seasons])

  const seasonOptions: MultiSelectOption[] = useMemo(() => {
    return seasons
      .map((s) => ({
        value: s.id,
        label: `${s.shortLabel} — ${s.label}`,
        count: seasonCountsMap.get(s.id) ?? 0,
      }))
      .filter((o) => o.count > 0 || selectedSeasons.has(o.value))
  }, [seasons, seasonCountsMap, selectedSeasons])

  const modeOptions: MultiSelectOption[] = useMemo(() => {
    return (data?.available_modes ?? [])
      .map((m) => ({ value: m.value, label: m.value, count: m.count }))
      .filter((o) => o.count > 0 || selectedModes.has(o.value))
  }, [data?.available_modes, selectedModes])

  const playlistOptions: MultiSelectOption[] = useMemo(() => {
    return (data?.available_playlists ?? [])
      .map((p) => ({ value: p.value, label: p.value, count: p.count }))
      .filter((o) => o.count > 0 || selectedPlaylists.has(o.value))
  }, [data?.available_playlists, selectedPlaylists])

  const hasActiveFilters =
    experience !== 'all' ||
    selectedSeasons.size > 0 ||
    selectedModes.size > 0 ||
    selectedPlaylists.size > 0

  const resetFilters = () => {
    setExperience('all')
    setSelectedSeasons(new Set())
    setSelectedModes(new Set())
    setSelectedPlaylists(new Set())
  }

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <h2 className="text-sm font-semibold">{t('career.highlight_matches.section_title')}</h2>
        <div className="flex items-center flex-wrap gap-2">
          <ExperienceDropdown
            value={experience}
            onChange={setExperience}
            counts={experienceCounts}
            dense
            labels={{
              placeholder: t('career.highlight_matches.filter_experience'),
              all: t('career.highlight_matches.experience_all'),
              ranked: t('career.highlight_matches.experience_ranked'),
              unranked: t('career.highlight_matches.experience_unranked'),
            }}
          />
          <MultiSelectFilter
            options={seasonOptions}
            selected={selectedSeasons}
            toggle={(v) => {
              setSelectedSeasons((prev) => {
                const next = new Set(prev)
                if (next.has(v)) next.delete(v)
                else next.add(v)
                return next
              })
            }}
            placeholder={t('career.highlight_matches.filter_seasons')}
            alwaysShow
            dense
          />
          <MultiSelectFilter
            options={playlistOptions}
            selected={selectedPlaylists}
            toggle={(v) => {
              setSelectedPlaylists((prev) => {
                const next = new Set(prev)
                if (next.has(v)) next.delete(v)
                else next.add(v)
                return next
              })
            }}
            placeholder={t('career.highlight_matches.filter_playlists')}
            alwaysShow
            dense
            disabled={playlistOptions.length === 0 && selectedPlaylists.size === 0}
          />
          <MultiSelectFilter
            options={modeOptions}
            selected={selectedModes}
            toggle={(v) => {
              setSelectedModes((prev) => {
                const next = new Set(prev)
                if (next.has(v)) next.delete(v)
                else next.add(v)
                return next
              })
            }}
            placeholder={t('career.highlight_matches.filter_modes')}
            alwaysShow
            dense
            disabled={modeOptions.length === 0 && selectedModes.size === 0}
          />
          {hasActiveFilters && (
            <button
              type="button"
              onClick={resetFilters}
              aria-label={t('career.highlight_matches.reset_filters')}
              title={t('career.highlight_matches.reset_filters')}
              className="rounded border border-input px-2 py-1 text-xs text-muted-foreground hover:border-foreground hover:text-foreground transition-colors inline-flex items-center gap-1"
            >
              <svg width="12" height="12" viewBox="0 0 16 16" aria-hidden="true" fill="none" stroke="currentColor" strokeWidth="1.5">
                <path d="M2 8a6 6 0 1 1 1.76 4.24" strokeLinecap="round" />
                <path d="M2 13.5V9.5h4" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
              {t('career.highlight_matches.reset_filters')}
            </button>
          )}
          <div role="tablist" aria-label={t('career.highlight_matches.section_title')} className="inline-flex border border-border rounded-md overflow-hidden text-xs">
            <button
              type="button"
              role="tab"
              aria-selected={variant === 'best'}
              onClick={() => setVariant('best')}
              className={`px-3 py-1.5 transition-colors ${
                variant === 'best' ? 'font-semibold' : 'hover:bg-accent/40 text-muted-foreground'
              }`}
              style={
                variant === 'best'
                  ? { backgroundColor: tokenCssVar('outcome-win'), color: 'var(--background)' }
                  : undefined
              }
            >
              {t('career.highlight_matches.tab_best')}
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={variant === 'worst'}
              onClick={() => setVariant('worst')}
              className={`px-3 py-1.5 transition-colors border-l border-border ${
                variant === 'worst' ? 'font-semibold' : 'hover:bg-accent/40 text-muted-foreground'
              }`}
              style={
                variant === 'worst'
                  ? { backgroundColor: tokenCssVar('outcome-loss'), color: 'var(--background)' }
                  : undefined
              }
            >
              {t('career.highlight_matches.tab_worst')}
            </button>
          </div>
        </div>
      </div>

      {isLoading && (
        <div className="flex h-32 items-center justify-center">
          <Spinner size="md" />
        </div>
      )}
      {isError && <p className="text-sm text-destructive">{t('career.errors.load_progression_failed')}</p>}
      {!isLoading && !isError && rows.length === 0 && (
        <p className="text-xs text-muted-foreground">{t('career.highlight_matches.empty')}</p>
      )}
      {!isLoading && !isError && rows.length > 0 && (
        <ExplorerMatchesTable
          rows={rows}
          playerSlug={playerSlug}
          contextDescriptor={{ kind: 'top_matches' }}
          alwaysShowPagination={false}
        />
      )}
    </section>
  )
}


