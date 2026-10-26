/**
 * Segmented — bascule à boutons partagée (horizon et vue de Tendances, échelle du graphe
 * d'évolution LUSR / CSR de la Carrière). Gabarit EXACT de la bascule de contexte de
 * `components/shell/FilterOmnibar` (groupe `role="group"`, boutons `aria-pressed`, mêmes
 * classes) : hors de l'omnibar, c'est le SEUL endroit qui écrit ce motif.
 */
export interface SegmentedOption<V extends string | number> {
  value: V
  label: string
}

export interface SegmentedProps<V extends string | number> {
  options: readonly SegmentedOption<V>[]
  value: V
  onChange: (value: V) => void
  ariaLabel: string
}

export function Segmented<V extends string | number>({
  options,
  value,
  onChange,
  ariaLabel,
}: SegmentedProps<V>) {
  return (
    <div
      className="flex shrink-0 items-center gap-0.5 rounded-md border border-input bg-background p-0.5 text-xs"
      role="group"
      aria-label={ariaLabel}
    >
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          onClick={() => onChange(option.value)}
          aria-pressed={value === option.value}
          className={[
            'rounded px-2 py-0.5 transition-colors',
            value === option.value
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:text-foreground',
          ].join(' ')}
        >
          {option.label}
        </button>
      ))}
    </div>
  )
}
