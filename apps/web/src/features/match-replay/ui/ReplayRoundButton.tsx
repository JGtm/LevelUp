/**
 * ReplayRoundButton — UN BOUTON DES BARRES DE LECTURE : rond, fantôme (recommencer, zoom) ou bordé (les sauts),
 * dans la barre complète ; carré fantôme au format de la barre d'une ligne (`compact`, cf. `compactButton`). Le nom
 * accessible est l'`aria-label` et le `title`, le contenu est une icône ou un glyphe.
 */
import type { ReactNode } from 'react'

import { COMPACT_BUTTON_CLASS, COMPACT_BUTTON_SQUARE, compactButtonCursor } from './compactButton'

export function ReplayRoundButton({
  onClick, label, children, ghost, bordered, disabled, compact,
}: {
  onClick: () => void
  label: string
  children: ReactNode
  ghost?: boolean
  bordered?: boolean
  /** Neutralise pendant un export (la lecture et l'export se disputeraient l'image courante) ou hors de portée. */
  disabled?: boolean
  /** Le format de la barre d'une ligne : carré, sans bordure ni fond au repos, quelle que soit la variante demandée. */
  compact?: boolean
}) {
  if (compact) {
    return (
      <button
        type="button"
        onClick={onClick}
        disabled={disabled}
        aria-label={label}
        title={label}
        style={COMPACT_BUTTON_SQUARE}
        className={`${COMPACT_BUTTON_CLASS} text-muted-foreground ${compactButtonCursor(!!disabled)}`}
      >
        {children}
      </button>
    )
  }
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
