/**
 * StarButton — bouton favori SVG pour les matchs.
 *
 * Variante compact : overlay absolu sur une MatchCard (top-left).
 * Variante standalone : bouton inline dans un header (MatchViewPage).
 */
interface StarButtonProps {
  isFavorite: boolean
  onToggle: () => void
  compact?: boolean
  disabled?: boolean
  className?: string
}

export function StarButton({
  isFavorite,
  onToggle,
  compact = false,
  disabled = false,
  className = '',
}: StarButtonProps) {
  const starPath =
    'M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z'

  return (
    <button
      type="button"
      disabled={disabled}
      onClick={(e) => {
        e.stopPropagation()
        onToggle()
      }}
      aria-label={isFavorite ? 'Retirer des favoris' : 'Ajouter aux favoris'}
      className={
        compact
          ? `absolute left-1.5 top-1.5 flex h-7 w-7 items-center justify-center rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isFavorite ? 'bg-warning text-warning-foreground' : 'bg-card/70 text-muted-foreground hover:bg-warning/70 hover:text-warning-foreground'} ${className}`
          : `inline-flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-60 ${isFavorite ? 'bg-warning/15 text-warning hover:bg-warning/25' : 'text-muted-foreground hover:bg-muted hover:text-foreground'} ${className}`
      }
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        className={compact ? 'h-3.5 w-3.5' : 'h-3.5 w-3.5'}
        fill={isFavorite ? 'currentColor' : 'none'}
        stroke="currentColor"
        strokeWidth={isFavorite ? 0 : 2}
        strokeLinecap="round"
        strokeLinejoin="round"
        aria-hidden="true"
      >
        <path d={starPath} />
      </svg>
      {!compact && (
        <span>{isFavorite ? 'Favori' : 'Favori'}</span>
      )}
    </button>
  )
}
