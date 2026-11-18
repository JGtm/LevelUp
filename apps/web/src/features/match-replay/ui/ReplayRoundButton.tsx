/**
 * ReplayRoundButton — UN BOUTON ROND DES BARRES DE LECTURE : fantôme (recommencer, zoom) ou bordé (les
 * sauts). Partagé par la barre complète et la barre d'une ligne ; le nom accessible est l'`aria-label` et
 * le `title`, le contenu est une icône ou un glyphe.
 */
import type { ReactNode } from 'react'

export function ReplayRoundButton({
  onClick, label, children, ghost, bordered, disabled,
}: {
  onClick: () => void
  label: string
  children: ReactNode
  ghost?: boolean
  bordered?: boolean
  /** Neutralise pendant un export (la lecture et l'export se disputeraient l'image courante) ou hors de portée. */
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
      className={`inline-flex h-[30px] w-[30px] items-center justify-center rounded-full transition-colors ${
        disabled ? 'cursor-not-allowed opacity-40' : 'cursor-pointer'
      } ${bordered ? 'border border-input hover:bg-accent' : ''} ${
        ghost ? 'text-muted-foreground hover:bg-accent hover:text-accent-foreground' : ''
      }`}
    >
      {children}
    </button>
  )
}
