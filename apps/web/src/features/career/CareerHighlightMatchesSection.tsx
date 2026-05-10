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
import { useEffect, useMemo, useRef, useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { ExplorerMatchesTable } from '@/features/explorer/ExplorerMatchesTable'
import { MultiSelectFilter, type MultiSelectOption } from '@/features/explorer/MultiSelectFilter'
import { Spinner } from '@/components/ui/spinner'
import { useCareerHighlightMatches } from './queries'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSeasons } from '@/lib/i18n/fieldMappings'
import { tokenCssVar } from '@/lib/accessibility'
import type { CareerHighlightFilters } from '@/lib/api/types'

type Variant = 'best' | 'worst'
type Experience = 'all' | 'ranked' | 'unranked'

export function CareerHighlightMatchesSection() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const seasons = useSeasons()

  const [variant, setVariant] = useState<Variant>('best')
  const [experience, setExperience] = useState<Experience>('all')
  const [selectedSeasons, setSelectedSeasons] = useState<Set<string>>(() => new Set())

  const filters: CareerHighlightFilters = useMemo(
    () => ({
      experience,
      season_ids: Array.from(selectedSeasons),
    }),
    [experience, selectedSeasons],
  )

  const { data, isLoading, isError } = useCareerHighlightMatches(playerSlug, filters)

  const rows = !data ? [] : variant === 'best' ? data.best_matches : data.worst_matches

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

  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <h2 className="text-sm font-semibold">{t('career.highlight_matches.section_title')}</h2>
        <div className="flex items-center flex-wrap gap-2">
          <ExperienceDropdown
            value={experience}
            onChange={setExperience}
            counts={data?.available_experience ?? []}
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
          />
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

// ---------------------------------------------------------------------------
// ExperienceDropdown — single-select dropdown (Tous / Classé / Non classé).
// Visuellement aligné sur MultiSelectFilter (pill button + popover) mais avec
// boutons radio. Inline ici car spécifique à cette section et trivial (3 valeurs).
// ---------------------------------------------------------------------------

interface ExperienceCount {
  value: 'all' | 'ranked' | 'unranked'
  count: number
}

interface ExperienceDropdownProps {
  value: Experience
  onChange: (next: Experience) => void
  counts: ExperienceCount[]
  labels: { placeholder: string; all: string; ranked: string; unranked: string }
}

function ExperienceDropdown({ value, onChange, counts, labels }: ExperienceDropdownProps) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const countMap = new Map<string, number>()
  for (const c of counts) countMap.set(c.value, c.count)

  const currentLabel = value === 'ranked' ? labels.ranked : value === 'unranked' ? labels.unranked : labels.all
  const isActive = value !== 'all'

  const options: { value: Experience; label: string }[] = [
    { value: 'all', label: labels.all },
    { value: 'ranked', label: labels.ranked },
    { value: 'unranked', label: labels.unranked },
  ]

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className={`rounded border px-2 py-1 text-sm bg-background flex items-center gap-1 whitespace-nowrap transition-colors ${
          isActive
            ? 'border-primary text-primary'
            : 'border-input text-muted-foreground hover:border-foreground'
        }`}
      >
        {labels.placeholder} : {currentLabel}
        <svg width="10" height="10" viewBox="0 0 12 12" aria-hidden="true">
          <path d="M2 4l4 4 4-4" stroke="currentColor" strokeWidth="1.5" fill="none" />
        </svg>
      </button>
      {open && (
        <div className="absolute right-0 z-20 mt-1 min-w-[16rem] rounded border border-border bg-popover p-1 shadow-md">
          {options.map((opt) => {
            const c = countMap.get(opt.value) ?? 0
            const selected = opt.value === value
            return (
              <button
                key={opt.value}
                type="button"
                onClick={() => {
                  onChange(opt.value)
                  setOpen(false)
                }}
                className={`w-full px-2 py-1 text-left text-sm rounded flex items-center justify-between transition-colors ${
                  selected ? 'bg-accent/60 font-semibold' : 'hover:bg-accent/40'
                }`}
              >
                <span>{opt.label}</span>
                <span className="font-mono text-xs text-muted-foreground">{c}</span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

