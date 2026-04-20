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
    { value: 'liked', label: text.toolbar.likedFirst },
  ]
  const safeMapOptions = withSelectedOption(mapOptions, mapFilter)
  const safeModeOptions = withSelectedOption(modeOptions, modeFilter)

  return (
    <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
      <div className="flex flex-wrap items-center gap-2.5">
        <span className="text-sm font-semibold text-muted-foreground">{text.toolbar.filterLabel}</span>
        <Select
          aria-label={text.toolbar.kindAriaLabel}
          className="w-[9.5rem]"
          value={kindFilter}
          onChange={(event) => onKindChange(event.target.value)}
        >
          {kindOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </Select>
        <Select
          aria-label={text.toolbar.sectionAriaLabel}
          className="w-[9.5rem]"
          value={sectionFilter}
          onChange={(event) => onSectionChange(event.target.value)}
        >
          {sectionOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </Select>
        <Select
          aria-label={text.toolbar.mapAriaLabel}
          className="w-[10.5rem]"
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
          className="w-[10.5rem]"
          value={modeFilter}
          onChange={(event) => onModeChange(event.target.value)}
        >
          <option value="">{text.toolbar.allModes}</option>
          {safeModeOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </Select>
        <label className="flex cursor-pointer items-center gap-1.5 whitespace-nowrap text-sm text-foreground">
          <input
            type="checkbox"
            checked={likedOnly}
            onChange={(event) => onLikedOnlyChange(event.target.checked)}
            className="rounded"
          />
          {text.toolbar.likedOnly}
        </label>
      </div>

      <div className="hidden h-6 w-px bg-border lg:block" aria-hidden="true" />

      <div className="flex flex-wrap items-center gap-2.5">
        <span className="text-sm font-semibold text-muted-foreground">{text.toolbar.sortLabel}</span>
        <Select
          aria-label={text.toolbar.sortAriaLabel}
          className="w-[9.5rem]"
          value={sortKey}
          onChange={(event) => onSortChange(event.target.value)}
        >
          {sortOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </Select>
        <Select
          aria-label={text.toolbar.groupAriaLabel}
          className="w-[10rem]"
          value={groupBy}
          onChange={(event) => onGroupByChange(event.target.value)}
        >
          {groupOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </Select>
      </div>
    </div>
  )
}