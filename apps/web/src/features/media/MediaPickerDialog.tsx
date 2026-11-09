/**
 * MediaPickerDialog — coquille commune des fenêtres de choix d'un média
 * (réassociation à un match, fichiers introuvables).
 *
 * Quatre briques sans logique métier, toutes les chaînes viennent de l'appelant :
 *   - MediaPickerDialog        superposition, carte, en-tête, barre d'outils, zone de défilement
 *   - MediaPickerListStatus    état de chargement, d'erreur ou de liste vide
 *   - MediaPickerRow           ligne sélectionnable (état normal, en attente, courant), action optionnelle
 *   - MediaPickerConfirmBar    pied de confirmation en deux temps
 */
import type { ReactNode } from 'react'

interface MediaPickerDialogProps {
  title: string
  subtitle?: ReactNode
  closeLabel: string
  /** Contenu de la barre d'outils sous l'en-tête ; la barre n'existe que s'il est fourni. */
  toolbar?: ReactNode
  onClose: () => void
  /** Pied de fenêtre, rendu sous la zone de défilement. */
  footer?: ReactNode
  children: ReactNode
}

export function MediaPickerDialog({
  title,
  subtitle,
  closeLabel,
  toolbar,
  onClose,
  footer,
  children,
}: MediaPickerDialogProps) {
  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-background/70"
      onClick={onClose}
    >
      <div
        className="mx-4 flex max-h-[85vh] w-full max-w-3xl flex-col rounded-lg bg-background shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <header className="flex items-center justify-between border-b border-border px-5 py-3">
          <div>
            <h2 className="text-base font-semibold">{title}</h2>
            {subtitle && <p className="text-xs text-muted-foreground">{subtitle}</p>}
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded p-1 text-sm text-muted-foreground hover:bg-accent"
            aria-label={closeLabel}
          >
            ✕
          </button>
        </header>

        {toolbar && (
          <div className="flex items-center gap-2 border-b border-border px-5 py-2 text-xs">
            {toolbar}
          </div>
        )}

        <div className="flex-1 overflow-y-auto px-3 py-2">{children}</div>

        {footer}
      </div>
    </div>
  )
}

interface MediaPickerListStatusProps {
  isLoading: boolean
  isError: boolean
  isEmpty: boolean
  loadingLabel: string
  errorLabel: string
  emptyLabel: string
}

export function MediaPickerListStatus({
  isLoading,
  isError,
  isEmpty,
  loadingLabel,
  errorLabel,
  emptyLabel,
}: MediaPickerListStatusProps) {
  return (
    <>
      {isLoading && <p className="p-4 text-center text-sm text-muted-foreground">{loadingLabel}</p>}
      {isError && <p className="p-4 text-center text-sm text-destructive">{errorLabel}</p>}
      {!isLoading && !isError && isEmpty && (
        <p className="p-4 text-center text-sm text-muted-foreground">
          {emptyLabel}
        </p>
      )}
    </>
  )
}

interface MediaPickerRowProps {
  selected: boolean
  current: boolean
  onSelect: () => void
  /** Vignette à gauche du contenu. */
  leading: ReactNode
  /**
   * Action propre à la ligne, posée à droite hors du bouton de sélection (un bouton ne peut pas
   * en contenir un autre). Sans elle, le rendu de la ligne est inchangé.
   */
  action?: ReactNode
  children: ReactNode
}

export function MediaPickerRow({ selected, current, onSelect, leading, action, children }: MediaPickerRowProps) {
  return (
    <li className={action ? 'relative' : undefined}>
      <button
        type="button"
        disabled={current}
        onClick={onSelect}
        className={`flex w-full gap-3 rounded-md border px-3 py-2 text-left transition-colors ${
          current
            ? 'border-primary/60 bg-primary/5 cursor-default'
            : selected
            ? 'border-primary/80 bg-primary/10 ring-2 ring-primary/40'
            : 'border-border hover:border-primary/50 hover:bg-accent'
        }`}
      >
        {leading}
        {children}
      </button>
      {action && <div className="absolute bottom-2 right-3">{action}</div>}
    </li>
  )
}

interface MediaPickerConfirmBarProps {
  title: string
  detail: ReactNode
  cancelLabel: string
  confirmLabel: string
  busyLabel: string
  busy: boolean
  onCancel: () => void
  onConfirm: () => void
}

export function MediaPickerConfirmBar({
  title,
  detail,
  cancelLabel,
  confirmLabel,
  busyLabel,
  busy,
  onCancel,
  onConfirm,
}: MediaPickerConfirmBarProps) {
  return (
    <footer className="flex items-center justify-between gap-3 border-t border-border bg-muted/40 px-5 py-3 text-sm">
      <div>
        <p className="font-medium">{title}</p>
        <p className="text-xs text-muted-foreground">{detail}</p>
      </div>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="rounded border border-border px-3 py-1.5 text-xs hover:bg-accent"
        >
          {cancelLabel}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={onConfirm}
          className="rounded bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:opacity-90 disabled:opacity-50"
        >
          {busy ? busyLabel : confirmLabel}
        </button>
      </div>
    </footer>
  )
}
