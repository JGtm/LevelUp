interface SpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  label?: string
  className?: string
}

export function Spinner({ size = 'md', label, className = '' }: SpinnerProps) {
  const sizes: Record<string, string> = {
    sm: 'h-4 w-4',
    md: 'h-8 w-8',
    lg: 'h-12 w-12',
  }
  return (
    <div className={`flex flex-col items-center gap-2 ${className}`}>
      <svg
        className={`animate-spin text-primary ${sizes[size]}`}
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        viewBox="0 0 24 24"
        aria-hidden="true"
      >
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
        <path
          className="opacity-75"
          fill="currentColor"
          d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
        />
      </svg>
      {label && <span className="text-sm text-muted-foreground">{label}</span>}
    </div>
  )
}

/** Overlay de chargement centré sur toute la page */
export function PageSpinner({ label = 'Chargement…' }: { label?: string }) {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <Spinner size="lg" label={label} />
    </div>
  )
}
