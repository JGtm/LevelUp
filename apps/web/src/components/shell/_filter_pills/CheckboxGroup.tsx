/**
 * CheckboxGroup — groupe de cases à cocher pour FiltresPill.
 *
 * Comportement (plan smart-filter-counts) :
 *   - Chaque option affiche son count à droite.
 *   - Options à count > 0 : visibles directement, cochables.
 *   - Options à count = 0 : repliées sous "+ N options indisponibles" (progressive
 *     disclosure). Une fois dépliées : grisées + tooltip explicatif. Si peu d'options
 *     au total (≤ COLLAPSE_THRESHOLD), on les laisse visibles grisées sans pliage.
 *   - Zombies (valeurs sélectionnées mais absentes du dataset) : toujours visibles
 *     en haut avec barre + warning, comme aujourd'hui.
 */
import { useState } from 'react'
import type { LabelValue } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

const COLLAPSE_THRESHOLD = 4

interface CheckboxGroupProps {
  title: string
  options: LabelValue[]
  selected: string[]
  onToggle: (value: string) => void
  zombies?: string[]
  /** Si true, désactive le pliage progressif (ex : Experience types — 3 valeurs max). */
  disableCollapse?: boolean
}

export function CheckboxGroup({
  title,
  options,
  selected,
  onToggle,
  zombies = [],
  disableCollapse = false,
}: CheckboxGroupProps) {
  const [showUnavailable, setShowUnavailable] = useState(false)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  if (options.length === 0 && zombies.length === 0) return null

  // Sépare options actives (count > 0 OU déjà cochées) des indisponibles (count = 0).
  // Une option cochée à count=0 reste visible (cas zombie partiel : sélection encore présente).
  const active: LabelValue[] = []
  const unavailable: LabelValue[] = []
  for (const opt of options) {
    if (opt.count > 0 || selected.includes(opt.value)) {
      active.push(opt)
    } else {
      unavailable.push(opt)
    }
  }

  const collapseEnabled = !disableCollapse && options.length > COLLAPSE_THRESHOLD
  const showInline = !collapseEnabled || showUnavailable

  return (
    <div className="flex flex-col">
      <h4 className="mb-1 text-2xs font-semibold uppercase tracking-wider text-muted-foreground">
        {title}
        {selected.length > 0 && (
          <span className="ml-1 text-primary">({selected.length})</span>
        )}
      </h4>
      <div className="max-h-44 overflow-y-auto rounded border border-border">
        {active.map((opt) => (
          <OptionRow
            key={opt.value}
            opt={opt}
            checked={selected.includes(opt.value)}
            onToggle={onToggle}
            disabled={false}
          />
        ))}

        {unavailable.length > 0 && collapseEnabled && !showUnavailable && (
          <button
            type="button"
            onClick={() => setShowUnavailable(true)}
            className="flex w-full items-center justify-center gap-1 px-2 py-1.5 text-2xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            + {unavailable.length} option{unavailable.length > 1 ? 's' : ''} indisponible
            {unavailable.length > 1 ? 's' : ''}
          </button>
        )}

        {unavailable.length > 0 && showInline && (
          <>
            {unavailable.map((opt) => (
              <OptionRow
                key={opt.value}
                opt={opt}
                checked={false}
                onToggle={onToggle}
                disabled
              />
            ))}
            {collapseEnabled && (
              <button
                type="button"
                onClick={() => setShowUnavailable(false)}
                className="flex w-full items-center justify-center gap-1 px-2 py-1 text-2xs font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
              >
                ↑ Masquer
              </button>
            )}
          </>
        )}

        {zombies.map((value) => (
          <label
            key={`zombie-${value}`}
            title={t('common.filters.incompatible_title')}
            className="flex cursor-pointer items-center gap-2 px-2 py-1 text-xs text-destructive/70 line-through transition-colors hover:bg-destructive/10"
          >
            <input
              type="checkbox"
              checked
              onChange={() => onToggle(value)}
              className="h-3 w-3 cursor-pointer rounded border-destructive/50 opacity-60 focus:ring-1 focus:ring-ring"
            />
            <span className="flex-1 truncate">{value}</span>
            <span className="shrink-0 text-2xs font-medium text-destructive">✕</span>
          </label>
        ))}
      </div>
    </div>
  )
}

interface OptionRowProps {
  opt: LabelValue
  checked: boolean
  onToggle: (value: string) => void
  disabled: boolean
}

function OptionRow({ opt, checked, onToggle, disabled }: OptionRowProps) {
  return (
    <label
      title={disabled ? '0 match si cette option est cochée' : undefined}
      aria-disabled={disabled}
      className={[
        'flex items-center gap-2 px-2 py-1 text-xs transition-colors',
        disabled
          ? 'cursor-not-allowed text-muted-foreground opacity-50'
          : 'cursor-pointer text-foreground hover:bg-muted',
      ].join(' ')}
    >
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={() => !disabled && onToggle(opt.value)}
        className="h-3 w-3 cursor-pointer rounded border-input text-primary focus:ring-1 focus:ring-ring disabled:cursor-not-allowed"
      />
      <span className="flex-1 truncate">{opt.label}</span>
      <span className="shrink-0 text-2xs tabular-nums text-muted-foreground">
        {opt.count}
      </span>
    </label>
  )
}
