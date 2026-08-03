import { useEffect, useRef, useState } from 'react'
import type { LabelValue, MediaAuthor } from '@/lib/api/types'
import { Select } from '@/components/ui/select'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { MediaText } from './i18n'
import { MediaAudioConfigButton } from './MediaAudioConfigButton'

interface MediaToolbarProps {
  text: MediaText
  playerSlug: string
  kindFilter: string
  authorSlugs: string[]
  authors: MediaAuthor[]
  playlistFilter: string
  mapFilter: string
  modeFilter: string
  groupBy: string
  sortKey: string
  likedOnly: boolean
  unassignedOnly: boolean
  totalUnassigned: number
  playlistOptions: LabelValue[]
  mapOptions: LabelValue[]
  modeOptions: LabelValue[]
  onKindChange: (value: string) => void
  onAuthorSlugsChange: (slugs: string[]) => void
  onPlaylistChange: (value: string) => void
  onMapChange: (value: string) => void
  onModeChange: (value: string) => void
  onSortChange: (value: string) => void
  onGroupByChange: (value: string) => void
  onLikedOnlyChange: (value: boolean) => void
  onUnassignedOnlyChange: (value: boolean) => void
}

interface AuthorsMultiSelectProps {
  text: MediaText
  authors: MediaAuthor[]
  selected: string[]
  onChange: (slugs: string[]) => void
}

// Sentinelle pour exprimer "0 sélectionné" (sinon [] = tous, sémantique backend).
const NONE_SENTINEL = '__no_authors_selected__'

function AuthorsMultiSelect({ text, authors, selected, onChange }: AuthorsMultiSelectProps) {
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (!open) return
    function handlePointer(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    function handleEscape(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', handlePointer)
    document.addEventListener('keydown', handleEscape)
    return () => {
      document.removeEventListener('mousedown', handlePointer)
      document.removeEventListener('keydown', handleEscape)
    }
  }, [open])

  const allChecked = selected.length === 0
  const noneChecked = selected.length === 1 && selected[0] === NONE_SENTINEL
  const indeterminate = !allChecked && !noneChecked

  function isAuthorChecked(slug: string): boolean {
    if (allChecked) return true
    if (noneChecked) return false
    return selected.includes(slug)
  }

  let summary: string
  if (allChecked) {
    summary = text.toolbar.allAuthors
  } else if (noneChecked) {
    summary = '0'
  } else if (selected.length === 1) {
    const match = authors.find((a) => a.player_slug === selected[0])
    summary = match?.gamertag ?? selected[0]
  } else {
    summary = `${selected.length}/${authors.length}`
  }

  function normalize(next: string[]): string[] {
    if (next.length === 0) return [NONE_SENTINEL]
    if (next.length === authors.length) return []
    return next
  }

  function toggleAll() {
    if (allChecked) {
      onChange([NONE_SENTINEL])
    } else {
      onChange([])
    }
  }

  function toggleOne(slug: string) {
    let base: string[]
    if (allChecked) base = authors.map((a) => a.player_slug)
    else if (noneChecked) base = []
    else base = selected

    const next = base.includes(slug) ? base.filter((s) => s !== slug) : [...base, slug]
    onChange(normalize(next))
  }

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-label={text.toolbar.authorsAriaLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((v) => !v)}
        className="flex h-8 items-center gap-1 rounded-md border border-input bg-background px-2 text-xs text-foreground shadow-sm hover:border-border"
      >
        <span className="max-w-[10rem] truncate">{summary}</span>
        <span className="opacity-60" aria-hidden="true">▾</span>
      </button>
      {open && (
        <div
          role="listbox"
          aria-multiselectable="true"
          className="absolute left-0 top-9 z-20 max-h-72 w-56 overflow-y-auto rounded-md border border-input bg-popover p-1 text-xs shadow-md"
        >
          {authors.length === 0 ? (
            <div className="px-2 py-3 text-center text-muted-foreground">{text.toolbar.noAuthors}</div>
          ) : (
            <>
              <label className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 font-semibold hover:bg-accent">
                <input
                  type="checkbox"
                  checked={allChecked}
                  ref={(el) => {
                    if (el) el.indeterminate = indeterminate
                  }}
                  onChange={toggleAll}
                  className="rounded"
                />
                <span>{text.toolbar.allAuthorsToggle}</span>
              </label>
              <div className="my-1 h-px bg-border" aria-hidden="true" />
              {authors.map((author) => (
                <label
                  key={author.player_slug}
                  className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 hover:bg-accent"
                >
                  <input
                    type="checkbox"
                    checked={isAuthorChecked(author.player_slug)}
                    onChange={() => toggleOne(author.player_slug)}
                    className="rounded"
                  />
                  <span className="flex-1 truncate">{author.gamertag}</span>
                  {author.is_self && (
                    <span className="text-2xs uppercase tracking-wide opacity-60">{text.toolbar.mine}</span>
                  )}
                  <span className="text-2xs tabular-nums opacity-50">{author.media_count}</span>
                </label>
              ))}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function withSelectedOption(options: LabelValue[], selectedValue: string) {
  if (!selectedValue || options.some((option) => option.value === selectedValue)) {
    return options
  }
  return [{ label: selectedValue, value: selectedValue, count: 0 }, ...options]
}

/**
 * Rend les options du filtre Mode en groupes hiérarchiques HTML <optgroup> :
 *   - Catégories racines (sans parent) : devient le label de l'<optgroup> (traduit FR via i18n)
 *     + 1ère option dans le groupe = "Toute la catégorie X" (value = catégorie EN)
 *   - Sous-modes (parent != "") : option dans l'<optgroup> de leur catégorie
 *     (label déjà traduit FR par mode_name_tr backend, fallback EN brut)
 *
 * Les options orphelines (sans parent et sans groupe correspondant) sont rendues à plat.
 */
function renderModeOptions(
  options: LabelValue[],
  text: MediaText,
  resolveCategoryLabel: (value: string) => string,
): React.ReactNode {
  const categories: { value: string; children: LabelValue[] }[] = []
  const orphans: LabelValue[] = []
  for (const opt of options) {
    if (!opt.parent) {
      categories.push({ value: opt.value, children: [] })
    }
  }
  const byCategory = new Map(categories.map((c) => [c.value, c]))
  for (const opt of options) {
    if (opt.parent) {
      const cat = byCategory.get(opt.parent)
      if (cat) cat.children.push(opt)
      else orphans.push(opt)
    }
  }
  return (
    <>
      {categories.map((cat) => {
        const localizedCat = resolveCategoryLabel(cat.value)
        return (
          <optgroup key={cat.value} label={localizedCat}>
            <option value={cat.value}>{text.toolbar.allInCategory(localizedCat)}</option>
            {cat.children.map((c) => (
              <option key={c.value} value={c.value}>{c.label}</option>
            ))}
          </optgroup>
        )
      })}
      {orphans.map((o) => (
        <option key={o.value} value={o.value}>{o.label}</option>
      ))}
    </>
  )
}

export function MediaToolbar({
  text,
  playerSlug,
  kindFilter,
  authorSlugs,
  authors,
  playlistFilter,
  mapFilter,
  modeFilter,
  groupBy,
  sortKey,
  likedOnly,
  unassignedOnly,
  totalUnassigned,
  playlistOptions,
  mapOptions,
  modeOptions,
  onKindChange,
  onAuthorSlugsChange,
  onPlaylistChange,
  onMapChange,
  onModeChange,
  onSortChange,
  onGroupByChange,
  onLikedOnlyChange,
  onUnassignedOnlyChange,
}: MediaToolbarProps) {
  const kindOptions = [
    { value: '', label: text.toolbar.allTypes },
    { value: 'screenshot', label: text.toolbar.screenshots },
    { value: 'clip', label: text.toolbar.clips },
  ]
  const sortOptions = [
    { value: 'date_desc', label: text.toolbar.dateDesc },
    { value: 'date_asc', label: text.toolbar.dateAsc },
    { value: 'map_asc', label: text.toolbar.mapAsc },
    { value: 'mode_asc', label: text.toolbar.modeAsc },
  ]
  const groupOptions = [
    { value: '', label: text.toolbar.noGrouping },
    { value: 'owner', label: text.toolbar.byOwner },
    { value: 'map', label: text.toolbar.byMap },
    { value: 'mode', label: text.toolbar.byMode },
    { value: 'session', label: text.toolbar.bySession },
  ]
  const safePlaylistOptions = withSelectedOption(playlistOptions, playlistFilter)
  const safeMapOptions = withSelectedOption(mapOptions, mapFilter)
  const safeModeOptions = withSelectedOption(modeOptions, modeFilter)

  // Phase 3.3 plan finition multi-titres : résolution des labels de catégories
  // de mode via useAssetLabel('mode', value). Repli sur le dict React legacy
  // pour préserver l'UX quand le titre ne déclare pas la catégorie (cas réel :
  // halo_5 n'a aucune section [assets.mode], cf. note G9 de son assets.toml).
  const { data: fieldMappings } = useFieldMappings()
  const resolveCategoryLabel = (value: string): string => {
    const fromTOML = fieldMappings?.assets?.mode?.[value]?.label
    if (fromTOML) return fromTOML
    return (
      text.toolbar.modeCategories[value as keyof typeof text.toolbar.modeCategories] ?? value
    )
  }

  const compactSelectClass = 'h-8 w-auto px-2 pr-6 text-xs'

  return (
    <div className="flex items-center gap-2">
      <span className="shrink-0 whitespace-nowrap text-xs font-semibold text-muted-foreground">
        {text.toolbar.filterLabel}
      </span>
      <Select
        aria-label={text.toolbar.kindAriaLabel}
        className={compactSelectClass}
        value={kindFilter}
        onChange={(event) => onKindChange(event.target.value)}
      >
        {kindOptions.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>
      <AuthorsMultiSelect
        text={text}
        authors={authors}
        selected={authorSlugs}
        onChange={onAuthorSlugsChange}
      />
      <Select
        aria-label={text.toolbar.mapAriaLabel}
        className={compactSelectClass}
        value={mapFilter}
        onChange={(event) => onMapChange(event.target.value)}
      >
        <option value="">{text.toolbar.allMaps}</option>
        {safeMapOptions.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>
      <Select
        aria-label={text.toolbar.playlistAriaLabel}
        className={compactSelectClass}
        value={playlistFilter}
        onChange={(event) => onPlaylistChange(event.target.value)}
      >
        {/* "Toutes playlists" toujours présent + chaque playlist détectée
            (même si une seule — pour montrer explicitement la playlist active
            au lieu de masquer le filtre). */}
        <option value="">{text.toolbar.allPlaylists}</option>
        {safePlaylistOptions.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>
      <Select
        aria-label={text.toolbar.modeAriaLabel}
        className={compactSelectClass}
        value={modeFilter}
        onChange={(event) => onModeChange(event.target.value)}
      >
        <option value="">{text.toolbar.allModes}</option>
        {renderModeOptions(safeModeOptions, text, resolveCategoryLabel)}
      </Select>
      <button
        type="button"
        aria-label={text.toolbar.likedOnlyAriaLabel}
        aria-pressed={likedOnly}
        title={text.toolbar.likedOnlyAriaLabel}
        onClick={() => onLikedOnlyChange(!likedOnly)}
        className={`flex h-8 items-center justify-center rounded-md border px-2 text-base leading-none transition-colors ${
          likedOnly
            ? 'border-rose-500/60 bg-rose-500/10 text-rose-500' // color-allow: rose pour le filtre "favoris" (heart icon) — CLAUDE.md §20 tolère rose pour liked
            : 'border-input text-muted-foreground hover:text-foreground'
        }`}
      >
        <span aria-hidden="true">{likedOnly ? '♥' : '♡'}</span>
      </button>
      {(unassignedOnly || totalUnassigned > 0) && (
        <button
          type="button"
          aria-label={text.toolbar.unassignedOnlyAriaLabel}
          aria-pressed={unassignedOnly}
          title={text.toolbar.unassignedOnlyAriaLabel}
          onClick={() => onUnassignedOnlyChange(!unassignedOnly)}
          className={`flex h-8 items-center justify-center gap-1 rounded-md border px-2 text-xs leading-none transition-colors ${
            unassignedOnly
              ? 'border-amber-500/60 bg-amber-500/10 text-amber-500' // color-allow: amber pour le filtre "sans match" — CLAUDE.md §20 tolère warning/amber dans badges d'état système
              : 'border-input text-muted-foreground hover:text-foreground'
          }`}
        >
          <span aria-hidden="true">⊘</span>
          {!unassignedOnly && totalUnassigned > 0 && (
            <span className="tabular-nums">{totalUnassigned}</span>
          )}
        </button>
      )}

      <div className="mx-1 h-5 w-px shrink-0 bg-border" aria-hidden="true" />

      <span className="shrink-0 whitespace-nowrap text-xs font-semibold text-muted-foreground">
        {text.toolbar.sortLabel}
      </span>
      <Select
        aria-label={text.toolbar.sortAriaLabel}
        className={compactSelectClass}
        value={sortKey}
        onChange={(event) => onSortChange(event.target.value)}
      >
        {sortOptions.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>
      <Select
        aria-label={text.toolbar.groupAriaLabel}
        className={compactSelectClass}
        value={groupBy}
        onChange={(event) => onGroupByChange(event.target.value)}
      >
        {groupOptions.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>

      <div className="mx-1 h-5 w-px shrink-0 bg-border" aria-hidden="true" />

      <MediaAudioConfigButton playerSlug={playerSlug} />
    </div>
  )
}
