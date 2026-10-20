/**
 * TendancesSegmented — la bascule à boutons de l'onglet Tendances (horizon, et plus tard
 * vue et pas). Gabarit EXACT de la bascule de contexte de `components/shell/FilterOmnibar`
 * (groupe `role="group"`, boutons `aria-pressed`, mêmes classes) : aucun composant partagé
 * n'existe pour ce motif, et c'est le SEUL endroit de la feature qui l'écrit.
 */
export interface TendancesSegmentedOption<V extends string | number> {
  value: V
  label: string
}

export interface TendancesSegmentedProps<V extends string | number> {
  options: readonly TendancesSegmentedOption<V>[]
  value: V
  onChange: (value: V) => void
  ariaLabel: string
}

export function TendancesSegmented<V extends string | number>({
  options,
  value,
  onChange,
  ariaLabel,
}: TendancesSegmentedProps<V>) {
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
