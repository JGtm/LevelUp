'use client'

/**
 * AlertDialog — modal de confirmation à intention destructive.
 *
 * Pattern aligné sur les autres modals du projet (BattlePassRewardLightbox,
 * FiltresPill…) : pas de dépendance Radix, ARIA `role="alertdialog"`,
 * fermeture sur Escape, focus initial sur le bouton de confirmation.
 *
 * Usage :
 *   <AlertDialog
 *     open={isOpen}
 *     onOpenChange={setIsOpen}
 *     title="Exclure ce match ?"
 *     description="Ce match sera marqué non pertinent…"
 *     confirmLabel="Exclure"
 *     cancelLabel="Annuler"
 *     destructive
 *     onConfirm={handleExclude}
 *   />
 */
import { useEffect, useRef, type ReactNode } from 'react'

import { Button } from '@/components/ui/button'

export interface AlertDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  confirmLabel: string
  cancelLabel: string
  /** Si true, le bouton de confirmation prend la variante `destructive`. */
  destructive?: boolean
  /** Désactive les deux boutons (ex: mutation en cours). */
  busy?: boolean
  /**
   * Contenu additionnel rendu sous la description (ex: champ de saisie). Quand
   * il contient un élément focusable auto-focus, passer `autoFocusConfirm={false}`
   * pour lui laisser le focus initial.
   */
  children?: ReactNode
  /** Focus initial sur le bouton de confirmation (défaut true). */
  autoFocusConfirm?: boolean
  onConfirm: () => void
}

export function AlertDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  cancelLabel,
  destructive = false,
  busy = false,
  children,
  autoFocusConfirm = true,
  onConfirm,
}: AlertDialogProps) {
  const confirmRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    if (!open) return
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape' && !busy) {
        event.preventDefault()
        onOpenChange(false)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [open, busy, onOpenChange])

  useEffect(() => {
    if (open && autoFocusConfirm) {
      confirmRef.current?.focus()
    }
  }, [open, autoFocusConfirm])

  if (!open) return null

  return (
    <div
      role="alertdialog"
      aria-modal="true"
      aria-labelledby="alert-dialog-title"
      aria-describedby="alert-dialog-description"
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
    >
      {/*
        Backdrop : div cliquable (pas un <button>) pour éviter de polluer le
        rôle ARIA — le bouton "Annuler" du footer reste l'alternative
        accessible. aria-hidden="true" pour ne pas confondre les lecteurs
        d'écran ; clic enregistré mais ignoré quand busy=true.
      */}
      <div
        aria-hidden="true"
        onClick={() => !busy && onOpenChange(false)}
        className={
          'absolute inset-0 bg-black/60 backdrop-blur-sm' +
          (busy ? ' cursor-wait' : ' cursor-pointer')
        }
      />

      {/* Panel */}
      <div
        className="relative z-10 w-full max-w-md rounded-lg border border-border bg-card text-card-foreground shadow-xl"
      >
        <div className="px-6 py-5">
          <h2
            id="alert-dialog-title"
            className="text-lg font-semibold tracking-tight text-foreground"
          >
            {title}
          </h2>
          <p
            id="alert-dialog-description"
            className="mt-2 text-sm text-muted-foreground"
          >
            {description}
          </p>
          {children && <div className="mt-3">{children}</div>}
        </div>
        <div className="flex items-center justify-end gap-2 border-t border-border px-6 py-4">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => onOpenChange(false)}
            disabled={busy}
          >
            {cancelLabel}
          </Button>
          <Button
            ref={confirmRef}
            type="button"
            variant={destructive ? 'destructive' : 'default'}
            size="sm"
            onClick={onConfirm}
            loading={busy}
            disabled={busy}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  )
}
