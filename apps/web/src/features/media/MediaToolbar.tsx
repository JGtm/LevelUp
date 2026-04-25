import { useEffect, useRef, useState } from 'react'
import type { LabelValue, MediaAuthor } from '@/lib/api/types'
import { Select } from '@/components/ui/select'
import type { MediaText } from './i18n'

interface MediaToolbarProps {
  text: MediaText
  kindFilter: string
  authorSlugs: string[]
  authors: MediaAuthor[]
  mapFilter: string
  modeFilter: string
  groupBy: string
  sortKey: string
  likedOnly: boolean
  mapOptions: LabelValue[]
  modeOptions: LabelValue[]
  onKindChange: (value: string) => void
  onAuthorSlugsChange: (slugs: string[]) => void
  onMapChange: (value: string) => void
  onModeChange: (value: string) => void
  onSortChange: (value: string) => void
  onGroupByChange: (value: string) => void
  onLikedOnlyChange: (value: boolean) => void
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
                    <span className="text-[10px] uppercase tracking-wide opacity-60">{text.toolbar.mine}</span>
                  )}
                  <span className="text-[10px] tabular-nums opacity-50">{author.media_count}</span>
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
  return [{ label: selectedValue, value: selectedValue }, ...options]
}

export function MediaToolbar({
  text,
  kindFilter,
  authorSlugs,
  authors,
  mapFilter,
  modeFilter,
  groupBy,
  sortKey,
  likedOnly,
  mapOptions,
  modeOptions,
  onKindChange,
  onAuthorSlugsChange,
  onMapChange,
  onModeChange,
  onSortChange,
  onGroupByChange,
  onLikedOnlyChange,
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
    { value: 'week', label: text.toolbar.byWeek },
  ]
  const safeMapOptions = withSelectedOption(mapOptions, mapFilter)
  const safeModeOptions = withSelectedOption(modeOptions, modeFilter)

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
        aria-label={text.toolbar.modeAriaLabel}
        className={compactSelectClass}
        value={modeFilter}
        onChange={(event) => onModeChange(event.target.value)}
      >
        <option value="">{text.toolbar.allModes}</option>
        {safeModeOptions.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>
      <button
        type="button"
        aria-label={text.toolbar.likedOnlyAriaLabel}
        aria-pressed={likedOnly}
        title={text.toolbar.likedOnlyAriaLabel}
        onClick={() => onLikedOnlyChange(!likedOnly)}
        className={`flex h-8 items-center justify-center rounded-md border px-2 text-base leading-none transition-colors ${
          likedOnly
            ? 'border-rose-500/60 bg-rose-500/10 text-rose-500'
            : 'border-input text-muted-foreground hover:text-foreground'
        }`}
      >
        <span aria-hidden="true">{likedOnly ? '♥' : '♡'}</span>
      </button>

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
    </div>
  )
}
