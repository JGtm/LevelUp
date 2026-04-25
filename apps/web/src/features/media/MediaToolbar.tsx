import type { LabelValue } from '@/lib/api/types'
import { Select } from '@/components/ui/select'
import type { MediaText } from './i18n'

interface MediaToolbarProps {
  text: MediaText
  kindFilter: string
  sectionFilter: string
  mapFilter: string
  modeFilter: string
  groupBy: string
  sortKey: string
  likedOnly: boolean
  mapOptions: LabelValue[]
  modeOptions: LabelValue[]
  onKindChange: (value: string) => void
  onSectionChange: (value: string) => void
  onMapChange: (value: string) => void
  onModeChange: (value: string) => void
  onSortChange: (value: string) => void
  onGroupByChange: (value: string) => void
  onLikedOnlyChange: (value: boolean) => void
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
  sectionFilter,
  mapFilter,
  modeFilter,
  groupBy,
  sortKey,
  likedOnly,
  mapOptions,
  modeOptions,
  onKindChange,
  onSectionChange,
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
  const sectionOptions = [
    { value: '', label: text.toolbar.allAuthors },
    { value: 'mine', label: text.toolbar.mine },
    { value: 'teammate', label: text.toolbar.teammates },
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
      <Select
        aria-label={text.toolbar.sectionAriaLabel}
        className={compactSelectClass}
        value={sectionFilter}
        onChange={(event) => onSectionChange(event.target.value)}
      >
        {sectionOptions.map((option) => (
          <option key={option.value} value={option.value}>{option.label}</option>
        ))}
      </Select>
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
